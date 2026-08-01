import json
from .utils import get_owner_id, row_to_dict
from .results import AcquireResult, OperationResult, InspectResult

def acquire(conn, key, *, target, owner_id=None, ttl_ms=10000):

    owner_id = owner_id or get_owner_id()
    ttl_ms = ttl_ms if ttl_ms and ttl_ms > 0 else 10000

    row = None 

    with conn.cursor() as cur:
        cur.execute("""
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
            attempts = attempts = odyssey_journeys.attempts + 1,
            fencing_token = nextval('odyssey_token_seq')
        WHERE odyssey_journeys.expires_at < NOW() AND odyssey_journeys.status = 'claimed'
        RETURNING owner_id, expires_at, fencing_token, status, target, expires_at > NOW() AS journey_alive;
        """, (key, target, owner_id, ttl_ms))

        result = cur.fetchone()

        if result is not None:
            row = row_to_dict(cur, result)

    conn.commit()

    if row is not None and row["fencing_token"] is None:
        raise Exception("Invariant violation: fencing_token is None")

    if row is not None:
        return AcquireResult(
            acquired=True,
            owner_id=row["owner_id"],
            target=row["target"],
            expires_at=row["expires_at"],
            journey_alive=row["journey_alive"],
            fencing_token=row["fencing_token"],
            status=row["status"]
        )
    
    with conn.cursor() as cur:
        cur.execute("""
        SELECT owner_id, expires_at, fencing_token, status, expires_at > NOW() AS journey_alive
        FROM odyssey_journeys
        WHERE key = %s
        AND target = %s
        """, (key, target))

        result = cur.fetchone()

        if result is not None:
            row = row_to_dict(cur, result)

    if row is not None:
        return AcquireResult(acquired=False,
            owner_id=row["owner_id"],
            target=row["target"],
            expires_at=row["expires_at"],
            journey_alive=row["journey_alive"],
            fencing_token=row["fencing_token"],
            status=row["status"])

def start_execution(conn, key, *, target, fencing_token):

    row = None
    
    with conn.cursor() as cur:
        cur.execute("""
        UPDATE odyssey_journeys
        SET status = 'executing',
            updated_at = NOW()
        WHERE key = %s
          AND target = %s
          AND fencing_token = %s
          AND status = 'claimed'
        RETURNING status;
        """, (key, target, fencing_token))

        result = cur.fetchone()
        success = result is not None
        if result is not None:
            row = row_to_dict(cur, result)
    
    conn.commit()
    if row is None:
        return OperationResult(success)

    return OperationResult(success, status=row["status"])

def complete(conn, key, *, target, fencing_token, execution_result=None):

    serialized_result = (
        json.dumps(execution_result)
        if execution_result is not None
        else None
    )

    with conn.cursor() as cur:
        cur.execute("""
        UPDATE odyssey_journeys
        SET
            status = 'completed',
            execution_result = %s,
            updated_at = NOW()
        WHERE key = %s
          AND target = %s
          AND fencing_token = %s
          AND status = 'executing'
        RETURNING 1;
        """, (serialized_result, key, target, fencing_token))

        success = cur.fetchone() is not None

    conn.commit()
    return OperationResult(success)

def abandon(conn, key, *, target, fencing_token):
    with conn.cursor() as cur:
        cur.execute("""
        UPDATE odyssey_journeys
        SET expires_at = NOW(),
            updated_at = NOW()
        WHERE key = %s
            AND target = %s
            AND fencing_token = %s
            AND status = 'executing'
        RETURNING 1;
        """, (key, target, fencing_token))
        success = cur.fetchone() is not None

    conn.commit()
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
