BEGIN;

DELETE FROM odyssey_journeys;
DELETE FROM odyssey_ledger;

INSERT INTO odyssey_ledger (key, status)
SELECT
    'bench-' || LPAD(i::text, LENGTH(:jobs::text), '0'),
    'queued'
FROM generate_series(1, :jobs) AS i;

INSERT INTO odyssey_journeys (
    key,
    target,
    sequence,
    mode,
    input,
    status,
    attempts
)
SELECT
    'bench-' || LPAD(i::text, LENGTH(:jobs::text), '0'),
    'hello',
    0,
    'local',
    '{"name":"Benchmark"}'::jsonb,
    'queued',
    0
FROM generate_series(1, :jobs) AS i;

COMMIT;