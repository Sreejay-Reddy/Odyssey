package postgres

const schemaSQL = `

DO $$
BEGIN
    CREATE TYPE odyssey_status AS ENUM (
		'queued',
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
    status odyssey_status NOT NULL DEFAULT 'claimed',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    PRIMARY KEY (key)
);

CREATE TABLE IF NOT EXISTS odyssey_journeys (
    key TEXT NOT NULL,
    target TEXT NOT NULL,
	sequence BIGINT NOT NULL,
    mode odyssey_execution_mode NOT NULL,
    worker_id TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    input JSONB,
    execution_result JSONB,
    status odyssey_status NOT NULL DEFAULT 'queued',
    attempts INTEGER NOT NULL DEFAULT 0,

    PRIMARY KEY (key, target),
    FOREIGN KEY (key)
        REFERENCES odyssey_ledger(key)
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
        REFERENCES odyssey_journeys(key, target)
);
`