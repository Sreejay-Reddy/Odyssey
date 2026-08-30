from .utils import row_to_dict
from .results import AcquireResult, OperationResult, InspectResult
from psycopg.types.json import Jsonb

async def acquire(conn, key, *, target, ttl_ms=10000):

    ttl_ms = ttl_ms if ttl_ms and ttl_ms > 0 else 10000

    row = None 

    async with conn.cursor() as cur:
        await cur.execute(
            """
            UPDATE odyssey_ledger
            SET 
                started_at = NOW(),
                expires_at = NOW() + (%s * INTERVAL '1 millisecond'),
                attempts = attempts + 1
            WHERE key = %s
                AND target = %s
                AND status = 'claimed'
                AND expires_at < NOW()
            RETURNING input, status, attempts
            """,
            (
                ttl_ms,
                key,
                target
            ),
        )

        result = await cur.fetchone()
        if result:
            row = row_to_dict(cur, result)

    if row is None:
        await conn.rollback()
        async with conn.cursor() as cur:
            await cur.execute("""
            SELECT status, attempts
            FROM odyssey_ledger
            WHERE key = %s
            AND target = %s
            """, (key, target))

            result = await cur.fetchone()

            if result is None:
                raise RuntimeError(
                    f"Durable record missing for key={key!r}, target={target!r}"
                )

            row = row_to_dict(cur, result)

            return AcquireResult(
                acquired=False,
                target=target,
                status=row["status"],
                attempts=row["attempts"])

    await conn.commit()
    return AcquireResult(
        acquired=True,
        target=target,
        status=row["status"],
        input=row["input"],
        attempts=row["attempts"]
    )


async def complete(conn, key, *, target, attempt, execution_result=None):
    serialized_result = (
        Jsonb(execution_result)
        if execution_result is not None
        else None
    )

    async with conn.cursor() as cur:
        await cur.execute(
            """
            UPDATE odyssey_ledger
            SET
                status = 'completed',
                completed_at = NOW(),
                execution_result = %s
            WHERE key = %s
                AND target = %s
                AND status = 'claimed'
                AND attempts = %s
            RETURNING TRUE
            """,
            (
                serialized_result,
                key,
                target,
                attempt,
            ),
        )

        result = await cur.fetchone()

    if not result:
        await conn.rollback()
        return OperationResult(success=False)

    await conn.commit()

    return OperationResult(success=True)

async def abandon(conn, key, *, target, attempt):
    async with conn.cursor() as cur:
        await cur.execute("""
        UPDATE odyssey_ledger
        SET expires_at = NOW()
        WHERE key = %s
            AND target = %s
            AND attempts = %s
            AND status = 'claimed'
        RETURNING TRUE;
        """, (key, target, attempt))
        success = await cur.fetchone() is not None

    if not success:
        await conn.rollback()
        return OperationResult(success=False)

    await conn.commit()
    return OperationResult(success)

# needs to be completely overhauled because each key and target represents a different notation
def inspect(conn, key):
    with conn.cursor() as cur:
        cur.execute("""
        SELECT owner_id, expires_at, updated_at, fencing_token, status, 
        expires_at > NOW() AS journey_alive, execution_result
        FROM odyssey_journeys
        WHERE key = %s 
        """, (key,))

        result = cur.fetchone()

        if result is None:
            return None

        row = row_to_dict(cur, result)

    return InspectResult(
        key = key,
        owner_id = row["owner_id"],
        fencing_token = row["fencing_token"],
        status = row["status"],
        journey_alive = row["journey_alive"],
        expires_at = row["expires_at"],
        updated_at = row["updated_at"],
        execution_result = row["execution_result"])
