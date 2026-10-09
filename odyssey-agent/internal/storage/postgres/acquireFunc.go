package postgres

import (
	"context"
	"log/slog"
	"time"

	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/registry"
	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/storage"
)

func (w *Writer) AcquireFunc(
    ctx context.Context,
    registry *registry.Registry,
    workerID string,
    limit int,
) (_ []storage.Execution, retErr error) {
    totalStart := time.Now()

    var (
        beginDuration      time.Duration
        selectDuration     time.Duration
        scanDuration       time.Duration
        registryDuration   time.Duration
        functionDuration   time.Duration
        resultScanDuration time.Duration
        commitDuration     time.Duration
        candidateCount     int
        executionCount     int
    )

    defer func() {
        slog.DebugContext(ctx, "acquire timing",
            "worker_id", workerID,
            "limit", limit,
            "candidates", candidateCount,
            "executions", executionCount,
            "begin", beginDuration,
            "select_query", selectDuration,
            "scan", scanDuration,
            "registry_lookup", registryDuration,
            "claim_function", functionDuration,
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

    type candidate struct {
        key    string
        target string
        ttlMS  int64
    }

    // 2. Select and lock queued jobs.
    stageStart = time.Now()
    rows, err := tx.Query(ctx, `
        SELECT key, target
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

    // 3. Scan candidate rows.
    candidates := make([]candidate, 0, limit)

    stageStart = time.Now()
    for rows.Next() {
        var c candidate

        if err := rows.Scan(&c.key, &c.target); err != nil {
            rows.Close()
            scanDuration = time.Since(stageStart)
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
        err = tx.Commit(ctx)
        commitDuration = time.Since(stageStart)
        if err != nil {
            return nil, err
        }
        return nil, nil
    }

    // 4. Resolve target TTLs using the existing Go registry.
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

    // 5. Call the PostgreSQL function to claim the batch.
    stageStart = time.Now()
    resultRows, err := tx.Query(ctx, `
        SELECT key, target, input, status, attempts
        FROM odyssey_claim_batch($1, $2, $3, $4)
    `, keys, targets, ttls, workerID)
    functionDuration = time.Since(stageStart)
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

    // 7. Commit after consuming all returned rows.
    stageStart = time.Now()
    err = tx.Commit(ctx)
    commitDuration = time.Since(stageStart)
    if err != nil {
        return nil, err
    }

    return executions, nil
}