package postgres

import (
	"context"
	"log/slog"
	"time"

	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/registry"
	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/storage"
)

func (w *Writer) Acquirecte(
    ctx context.Context,
    registry *registry.Registry,
    workerID string,
    limit int,
) (_ []storage.Execution, retErr error) {
    totalStart := time.Now()

    var (
        beginDuration       time.Duration
        selectDuration      time.Duration
        scanDuration        time.Duration
        registryDuration    time.Duration
        updateDuration      time.Duration
        resultScanDuration  time.Duration
        commitDuration      time.Duration
        candidateCount      int
        executionCount      int
    )

    defer func() {
        slog.InfoContext(ctx, "acquire timing",
            "worker_id", workerID,
            "limit", limit,
            "candidates", candidateCount,
            "executions", executionCount,
            "begin", beginDuration,
            "select_query", selectDuration,
            "scan", scanDuration,
            "registry_lookup", registryDuration,
            "batch_update", updateDuration,
            "result_scan", resultScanDuration,
            "commit", commitDuration,
            "total", time.Since(totalStart),
            "error", retErr,
        )
    }()

    if limit <= 0 {
        return nil, nil
    }

    // 1. Begin transaction.
    stageStart := time.Now()
    tx, err := w.pool.Begin(ctx)
    beginDuration = time.Since(stageStart)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback(ctx)

    // 2. Fetch and lock queued candidates.
    stageStart = time.Now()
    rows, err := tx.Query(ctx, `
        SELECT key, target, input
        FROM odyssey_journeys
        WHERE status = 'queued'
        ORDER BY key, target
        LIMIT $1
        FOR UPDATE SKIP LOCKED
    `, limit)
    selectDuration = time.Since(stageStart)
    if err != nil {
        return nil, err
    }

    type candidate struct {
        key    string
        target string
        input  []byte
        ttlMS  int64
    }

    candidates := make([]candidate, 0, limit)

    // 3. Scan candidate rows.
    stageStart = time.Now()
    for rows.Next() {
        var c candidate

        if err := rows.Scan(&c.key, &c.target, &c.input); err != nil {
            rows.Close()
            return nil, err
        }

        candidates = append(candidates, c)
    }
    scanDuration = time.Since(stageStart)

    rowsErr := rows.Err()
    rows.Close()
    if rowsErr != nil {
        return nil, rowsErr
    }

    candidateCount = len(candidates)

    if candidateCount == 0 {
        stageStart = time.Now()
        err := tx.Commit(ctx)
        commitDuration = time.Since(stageStart)
        if err != nil {
            return nil, err
        }
        return nil, nil
    }

    // 4. Resolve TTLs from the registry.
    stageStart = time.Now()
    for i := range candidates {
        targetDef, err := registry.GetByName(candidates[i].target)
        if err != nil {
            registryDuration = time.Since(stageStart)
            return nil, err
        }

        candidates[i].ttlMS = int64(targetDef.TTLMS)
    }
    registryDuration = time.Since(stageStart)

    keys := make([]string, candidateCount)
    targets := make([]string, candidateCount)
    ttls := make([]int64, candidateCount)

    for i, c := range candidates {
        keys[i] = c.key
        targets[i] = c.target
        ttls[i] = c.ttlMS
    }

    // 5. Claim the batch with one set-based UPDATE.
    stageStart = time.Now()
    resultRows, err := tx.Query(ctx, `
        WITH claims AS (
            SELECT *
            FROM unnest(
                $1::text[],
                $2::text[],
                $3::bigint[]
            ) AS c(key, target, ttl_ms)
        )
        UPDATE odyssey_journeys AS j
        SET
            status = 'claimed',
            worker_id = $4,
            expires_at = NOW()
                + (claims.ttl_ms * INTERVAL '1 millisecond'),
            attempts = j.attempts + 1
        FROM claims
        WHERE j.key = claims.key
          AND j.target = claims.target
          AND j.status = 'queued'
        RETURNING
            j.key,
            j.target,
            j.input,
            j.status,
            j.attempts
    `, keys, targets, ttls, workerID)
    updateDuration = time.Since(stageStart)
    if err != nil {
        return nil, err
    }

    // 6. Scan claimed executions.
    executions := make([]storage.Execution, 0, candidateCount)

    stageStart = time.Now()
    for resultRows.Next() {
        var execution storage.Execution

        if err := resultRows.Scan(
            &execution.Key,
            &execution.Target,
            &execution.Input,
            &execution.Status,
            &execution.Attempts,
        ); err != nil {
            resultRows.Close()
            resultScanDuration = time.Since(stageStart)
            return nil, err
        }

        execution.WorkerID = workerID
        executions = append(executions, execution)
    }
    resultScanDuration = time.Since(stageStart)

    resultErr := resultRows.Err()
    resultRows.Close()
    if resultErr != nil {
        return nil, resultErr
    }

    executionCount = len(executions)

    // 7. Commit.
    stageStart = time.Now()
    err = tx.Commit(ctx)
    commitDuration = time.Since(stageStart)
    if err != nil {
        return nil, err
    }

    return executions, nil
}