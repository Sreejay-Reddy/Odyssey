package postgres

import (
	"context"

	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/storage"
	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/registry"
	"github.com/jackc/pgx/v5"
)

func (w *Writer) Acquire(
	ctx context.Context,
	registry *registry.Registry,
	workerID string,
	limit int,
) ([]storage.Execution, error) {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT key, target, input
		FROM odyssey_journeys
		WHERE status = 'queued'
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, limit)
	if err != nil {
		return nil, err
	}

	var executions []storage.Execution

	for rows.Next() {
		var e storage.Execution

		if err := rows.Scan(
			&e.Key,
			&e.Target,
			&e.Input,
		); err != nil {
			rows.Close()
			return nil, err
		}

		e.WorkerID = workerID

		registred, err := registry.GetByName(e.Target)
		if err != nil {
			return nil, err
		}
		e.TTLMS = registred.TTLMS

		executions = append(executions, e)
	}

	rows.Close()

	if err := rows.Err(); err != nil {
		return nil, err
	}

	batch := &pgx.Batch{}

	for _, e := range executions {
		batch.Queue(
			`UPDATE odyssey_journeys
			 SET
				started_at = NOW(),
				attempts = attempts + 1,
				status = 'claimed',
				worker_id = $1,
				expires_at = now() + ($2 * interval '1 millisecond') + (interval '30 second')
			 WHERE key = $3
			   AND target = $4
			 RETURNING input, status, attempts`,
			e.WorkerID,
			e.TTLMS,
			e.Key,
			e.Target,
		)
	}

	results := tx.SendBatch(ctx, batch)
	defer results.Close()

	for i := range executions {
		e := &executions[i]

		rows, err := results.Query()
		if err != nil {
			return nil, err
		}

		if rows.Next() {
			if err := rows.Scan(
				&e.Input,
				&e.Status,
				&e.Attempts,
			); err != nil {
				rows.Close()
				return nil, err
			}
		} else {
			e.Status = "acquire_failed"
		}

		rows.Close()
	}

	if len(executions) == 0 {
        return executions, nil
    }

	if err := results.Close(); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return executions, nil
}