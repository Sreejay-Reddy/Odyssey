CREATE OR REPLACE FUNCTION odyssey_acquire_work(
    p_worker_ids TEXT[],
    p_batch_size INTEGER,
    p_ttl_ms JSONB
)
RETURNS TABLE (
    key TEXT,
    target TEXT,
    worker_id TEXT,
    input JSONB,
    status TEXT,
    attempts INTEGER,
    ttl_ms INTEGER
)
LANGUAGE plpgsql
AS $$
DECLARE
    v_worker_id TEXT;
BEGIN
    IF p_worker_ids IS NULL
       OR array_length(p_worker_ids, 1) IS NULL THEN
        RETURN;
    END IF;

    IF p_batch_size <= 0 THEN
        RAISE EXCEPTION 'batch size must be greater than zero';
    END IF;

    FOREACH v_worker_id IN ARRAY p_worker_ids
    LOOP
        RETURN QUERY
        WITH candidates AS (
            SELECT
                j.key,
                j.target
            FROM odyssey_journeys j
            WHERE j.status = 'queued'
            ORDER BY j.key
            LIMIT p_batch_size
            FOR UPDATE SKIP LOCKED
        )
        UPDATE odyssey_journeys j
        SET
            started_at = NOW(),
            attempts = j.attempts + 1,
            status = 'claimed',
            worker_id = v_worker_id,
            expires_at =
                NOW()
                + (
                    COALESCE(
                        (p_ttl_ms ->> j.target)::INTEGER,
                        30000
                    ) * INTERVAL '1 millisecond'
                )
                + INTERVAL '30 seconds'
        FROM candidates c
        WHERE j.key = c.key
          AND j.target = c.target
        RETURNING
            j.key,
            j.target,
            j.worker_id,
            j.input,
            j.status::TEXT,
            j.attempts,
            COALESCE(
                (p_ttl_ms ->> j.target)::INTEGER,
                30000
            );
    END LOOP;
END;
$$;