CREATE OR REPLACE FUNCTION odyssey_claim_batch(
    p_keys      TEXT[],
    p_targets   TEXT[],
    p_ttls_ms   BIGINT[],
    p_worker_id TEXT
)
RETURNS SETOF odyssey_journeys
LANGUAGE plpgsql
AS $$
BEGIN
    IF cardinality(p_keys) IS DISTINCT FROM cardinality(p_targets)
       OR cardinality(p_keys) IS DISTINCT FROM cardinality(p_ttls_ms)
    THEN
        RAISE EXCEPTION 'Acquire arrays must have equal lengths';
    END IF;

    RETURN QUERY
    WITH claims AS (
        SELECT c.key, c.target, c.ttl_ms
        FROM unnest(
            p_keys,
            p_targets,
            p_ttls_ms
        ) AS c(key, target, ttl_ms)
    )
    UPDATE odyssey_journeys AS j
    SET
        status = 'claimed',
        worker_id = p_worker_id,
        expires_at = NOW()
            + (claims.ttl_ms * INTERVAL '1 millisecond'),
        attempts = j.attempts + 1
    FROM claims
    WHERE j.key = claims.key
      AND j.target = claims.target
      AND j.status = 'queued'
    RETURNING j.*;
END;
$$;