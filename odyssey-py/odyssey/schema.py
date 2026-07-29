SCHEMA_SQL = """
CREATE SEQUENCE IF NOT EXISTS odyssey_token_seq;

DO $$
BEGIN
    CREATE TYPE odyssey_status AS ENUM (
        'claimed',
        'executing',
        'completed',
        'reconciling'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS odyssey_journeys (
    key TEXT PRIMARY KEY,
    owner_id TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    execution_result JSONB,
    status odyssey_status NOT NULL DEFAULT 'claimed',
    fencing_token BIGINT NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_odyssey_expiry
    ON odyssey_journeys (expires_at);
"""