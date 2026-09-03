package postgres

import(
	"context"

	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/storage"
	"github.com/jackc/pgx/v5"
)

func (w *Writer) complete(ctx context.Context, executions []storage.Execution) error {
	batch := &pgx.Batch{}

	for _, e := range executions {
		batch.Queue(
			`UPDATE odyssey_journeys
			 SET
				completed_at = NOW(),
				status = 'completed',
				execution_result = $1
			 WHERE key = $2
				AND target = $3
				AND status = 'claimed'
			 RETURNING status`,
			e.ExecutionResult,
			e.Key,
			e.Target,
		)
	}

	results := w.conn.SendBatch(ctx, batch)
	defer results.Close()

	for i := range executions {
		e := &executions[i]

		rows, err := results.Query()
		if err != nil {
			return err
		}

		if rows.Next() {
			err := rows.Scan(
				&e.Status,
			)
			rows.Close()

			if err != nil {
				return err
			}
		}else{
			rows.Close()
			e.Status = "complete_failed"
		}
	}

	return nil
}