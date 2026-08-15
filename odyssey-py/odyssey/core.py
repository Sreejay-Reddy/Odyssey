import json
from .utils import get_owner_id, row_to_dict
from .results import AcquireResult, OperationResult, InspectResult
from psycopg.types.json import Jsonb

async def acquire(conn, key, *, target, owner_id=None, ttl_ms=10000):

    owner_id = owner_id or get_owner_id()
    ttl_ms = ttl_ms if ttl_ms and ttl_ms > 0 else 10000

    row = None 

    async with conn.cursor() as cur:
        await cur.execute("""
        UPDATE odyssey_ledger
        SET
            started_at = NOW()
        WHERE key = %s
            AND target = %s
            AND status = 'claimed'
        RETURNING TRUE;
        """, (key, target))

        ledger_result = await cur.fetchone()
    
        await cur.execute("""
        INSERT INTO odyssey_journeys (
            key,
            target,
            owner_id,
            expires_at,
            updated_at,
            fencing_token
        )
        VALUES (
            %s,
            %s,
            %s,
            NOW() + (%s * INTERVAL '1 millisecond'),
            NOW(),
            nextval('odyssey_token_seq')
        )
        ON CONFLICT (key, target)
        DO UPDATE
        SET
            owner_id = EXCLUDED.owner_id,
            expires_at = EXCLUDED.expires_at,
            updated_at = NOW(),
            attempts = odyssey_journeys.attempts + 1,
            fencing_token = nextval('odyssey_token_seq')
        WHERE odyssey_journeys.expires_at < NOW() AND odyssey_journeys.status = 'claimed'
        RETURNING owner_id, expires_at, fencing_token, status, target, expires_at > NOW() AS journey_alive;
        """, (key, target, owner_id, ttl_ms))

        journey_result = await cur.fetchone()

        success = journey_result is not None and ledger_result is not None
        if success:
            row = row_to_dict(cur, journey_result)

    if row is None:
        await conn.rollback()

    if row is not None and row["fencing_token"] is None:
        raise Exception("Invariant violation: fencing_token is None")

    if row is not None:
        await conn.commit()
        return AcquireResult(
            acquired=True,
            owner_id=row["owner_id"],
            target=row["target"],
            expires_at=row["expires_at"],
            journey_alive=row["journey_alive"],
            fencing_token=row["fencing_token"],
            status=row["status"]
        )
    
    async with conn.cursor() as cur:
        await cur.execute("""
        SELECT owner_id, target, expires_at, fencing_token, status, expires_at > NOW() AS journey_alive
        FROM odyssey_journeys
        WHERE key = %s
        AND target = %s
        """, (key, target))

        result = await cur.fetchone()

        if result is not None:
            row = row_to_dict(cur, result)

    if row is not None:
        return AcquireResult(
            acquired=False,
            owner_id=row["owner_id"],
            target=row["target"],
            expires_at=row["expires_at"],
            journey_alive=row["journey_alive"],
            fencing_token=row["fencing_token"],
            status=row["status"])

async def start_execution(conn, key, *, target, fencing_token):

    row = None
    
    async with conn.cursor() as cur:
        await cur.execute("""
        UPDATE odyssey_ledger
        SET 
            status = 'executing'
        WHERE key = %s
            AND target = %s
            AND status = 'claimed'
        RETURNING TRUE;
        """, (key, target))

        ledger_result = await cur.fetchone()

        await cur.execute("""
        UPDATE odyssey_journeys
        SET status = 'executing',
            updated_at = NOW()
        WHERE key = %s
          AND target = %s
          AND fencing_token = %s
          AND status = 'claimed'
        RETURNING TRUE;
        """, (key, target, fencing_token))

        journey_result = await cur.fetchone()

        success = journey_result is not None and ledger_result is not None
        if success:
            row = row_to_dict(cur, journey_result)
    
    if row is None:
        await conn.rollback()
        return OperationResult(success)

    await conn.commit()
    return OperationResult(success, status=row["status"])

async def complete(conn, key, *, target, fencing_token, execution_result=None):

    serialized_result = (
        Jsonb(execution_result)
        if execution_result is not None
        else None
    )

    async with conn.cursor() as cur:

        await cur.execute("""
        UPDATE odyssey_ledger
        SET
            status = 'completed',
            completed_at = NOW()
        WHERE key = %s
            AND target = %s
            AND status = 'executing'
        RETURNING TRUE;
        """, (key, target))

        ledger_success = await cur.fetchone() is not None

        await cur.execute("""
        UPDATE odyssey_journeys
        SET
            status = 'completed',
            execution_result = %s,
            updated_at = NOW()
        WHERE key = %s
          AND target = %s
          AND fencing_token = %s
          AND status = 'executing'
        RETURNING TRUE;
        """, (serialized_result, key, target, fencing_token))

        journey_success = await cur.fetchone() is not None

    if not ledger_success or not journey_success:
        await conn.rollback()
        return OperationResult(success=False)

    await conn.commit()
    return OperationResult(success=True)

async def abandon(conn, key, *, target, fencing_token):
    async with conn.cursor() as cur:
        await cur.execute("""
        UPDATE odyssey_journeys
        SET expires_at = NOW(),
            updated_at = NOW()
        WHERE key = %s
            AND target = %s
            AND fencing_token = %s
            AND status = 'executing'
        RETURNING TRUE;
        """, (key, target, fencing_token))
        success = await cur.fetchone() is not None

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
