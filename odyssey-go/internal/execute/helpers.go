package execute

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

func (e *execution) fetchResponse(ctx context.Context) (bool, error) {
	err := e.conn.QueryRow(
		ctx,
		`SELECT
            execution_result,
            status
        FROM odyssey_journeys
        WHERE key = $1
            AND target = $2
            AND status = 'completed'`,
		e.key,
		e.target,
	).Scan(&e.metadata.response, &e.metadata.status)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows){
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (e *execution) fetchInput(ctx context.Context) (bool, error) {
	err := e.conn.QueryRow(
		ctx,
		`SELECT input FROM odyssey_ledger 
        WHERE key = $1
        AND target = $2`,
		e.key,
		e.target,
	).Scan(&e.input)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows){
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func getOwnerID() string {
    hostname, _ := os.Hostname()

    return fmt.Sprintf("%s-%d", hostname, os.Getpid())
}