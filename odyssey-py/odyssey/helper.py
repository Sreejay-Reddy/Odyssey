from .results import OperationResult
from .utils import row_to_dict

async def validate_and_extend(
    conn,
    key,
    *,
    target,
    fencing_token,
    ttl_ms
):
    row = None

    async with conn.cursor() as cur:
        await cur.execute("""
        UPDATE odyssey_journeys
        SET
            expires_at = NOW() + (%s * INTERVAL '1 millisecond'),
            updated_at = NOW()
        WHERE key = %s
          AND target = %s
          AND fencing_token = %s
          AND expires_at > NOW()
          AND status = 'claimed'
        RETURNING status;
        """, (
            ttl_ms,
            key,
            target,
            fencing_token
        ))

        result = await cur.fetchone()
        success = result is not None

        if result is not None:
            row = row_to_dict(cur, result)

    await conn.commit()

    if row is None:
        return OperationResult(success)

    return OperationResult(success, status=row["status"])

async def fetch_cached_response(conn, key, *, target):
    row = None

    async with conn.cursor() as cur:
        await cur.execute("""
        SELECT
            execution_result,
            status
        FROM odyssey_journeys
        WHERE key = %s
            AND target = %s
            AND status = 'completed'
        """, (key, target))

        result = await cur.fetchone()

        if result is not None:
            row = row_to_dict(cur, result)

    if row is None:
        return None

    return {
        "response": row["execution_result"],
        "status": row["status"]
    }

async def fetch_input(conn, key, *, target):
    row = None

    async with conn.cursor() as cur:
        await cur.execute("""
        SELECT input FROM odyssey_ledger 
        WHERE key = %s
        AND target = %s
        """, (key, target))

        result = await cur.fetchone()

        if result is not None:
            row = row_to_dict(cur, result)

    if row is None:
        return None

    return row["input"]
