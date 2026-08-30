SCHEMA_SQL = """

DO $$
BEGIN
    CREATE TYPE odyssey_status AS ENUM (
        'claimed',
        'completed',
        'reconciling'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE odyssey_execution_mode AS ENUM (
        'local',
        'delegated'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE delivery_status AS ENUM (
        'delivered',
        'failed',
        'pending'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS odyssey_ledger (
    key TEXT NOT NULL,
    target TEXT NOT NULL,
    sequence BIGINT NOT NULL,
    status odyssey_status NOT NULL DEFAULT 'claimed',
    mode odyssey_execution_mode NOT NULL,
    input JSONB,
    execution_result JSONB DEFAULT NULL,
    attempts BIGINT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    PRIMARY KEY (key, target)
);

CREATE TABLE IF NOT EXISTS odyssey_deliveries (
    key TEXT NOT NULL,
    target TEXT NOT NULL,
    emit_to TEXT NOT NULL,
    payload JSONB,
    status delivery_status NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    last_attempt_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (key, target, emit_to),
    FOREIGN KEY (key, target)
        REFERENCES odyssey_ledger(key, target)
);
"""