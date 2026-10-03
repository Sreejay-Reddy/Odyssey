package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/registry"
	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/storage"
)

func (w *Writer) Acquire(
	ctx context.Context,
	registry *registry.Registry,
	workerIDs []string,
	limit int,
) (map[string][]storage.Execution, error) {

	if len(workerIDs) == 0 {
		return nil, fmt.Errorf("no workers provided")
	}

	ttl := make(map[string]int64)

	for _, target := range registry.All() {
		ttl[target.Target] = int64(target.TTLMS)
	}

	ttlJSON, err := json.Marshal(ttl)
	if err != nil {
		return nil, err
	}

	rows, err := w.pool.Query(
		ctx,
		`SELECT *
		 FROM odyssey_acquire_work(
			 $1::text[],
			 $2::integer,
			 $3::jsonb
		 )`,
		workerIDs,
		limit,
		ttlJSON,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	executions := make(map[string][]storage.Execution, len(workerIDs))

	for _, workerID := range workerIDs {
		executions[workerID] = make([]storage.Execution, 0, limit)
	}

	totalRows := 0

	for rows.Next() {
		var e storage.Execution

		if err := rows.Scan(
			&e.Key,
			&e.Target,
			&e.WorkerID,
			&e.Input,
			&e.Status,
			&e.Attempts,
			&e.TTLMS,
		); err != nil {
			return nil, err
		}

		executions[e.WorkerID] = append(
			executions[e.WorkerID],
			e,
		)

		totalRows++
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return executions, nil
}