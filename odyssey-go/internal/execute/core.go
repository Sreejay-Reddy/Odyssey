package execute

import (
	"context"
	"time"
	"errors"
	"github.com/jackc/pgx/v5"
)

type journeyMetadata struct {
    ownerID      string
    expiresAt    time.Time
    fencingToken int64
    status       string
    target       string
    journeyAlive bool
	response any 
}

type execution struct{
	key string
	target string
	ownerID string
	ttlMS int64
	input any

	conn *pgx.Conn
	metadata journeyMetadata
}

func (e *execution) acquire(ctx context.Context) (bool, error) {
	e.ownerID = getOwnerID()

	tx, err := e.conn.Begin(ctx)

	if err != nil {
		return false, err 
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(
		ctx,
		`UPDATE odyssey_ledger
        SET
            started_at = NOW()
        WHERE key = $1
            AND target = $2
            AND status = 'claimed'
        RETURNING TRUE;`,
		e.key,
		e.target,
	).Scan(new(bool))

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil 
		}

		return false, err
	}

	err = tx.QueryRow(
		ctx,
		`INSERT INTO odyssey_journeys (
            key,
            target,
            owner_id,
            expires_at,
            updated_at,
            fencing_token
        )
        VALUES (
            $1,
            $2,
            $3,
            NOW() + ($4 * INTERVAL '1 millisecond'),
            NOW(),
            nextval('odyssey_token_seq')
        )
        ON CONFLICT (key, target)
        DO UPDATE
        SET
            owner_id = EXCLUDED.owner_id,
            expires_at = EXCLUDED.expires_at,
            updated_at = NOW(),
            attempts = odyssey_journeys.attempts + 1,
            fencing_token = nextval('odyssey_token_seq')
        WHERE odyssey_journeys.expires_at < NOW() AND odyssey_journeys.status = 'claimed'
        RETURNING owner_id, expires_at, fencing_token, status, target, expires_at > NOW() AS journey_alive;`,
		e.key,
		e.target,
		e.ownerID,
		e.ttlMS,
	).Scan(
	&e.metadata.ownerID,
    &e.metadata.expiresAt,
    &e.metadata.fencingToken,
    &e.metadata.status,
    &e.metadata.target,
    &e.metadata.journeyAlive)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		return false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}

	return true, nil
}

func (e *execution) startExecution(ctx context.Context) (bool, error){
	tx, err := e.conn.Begin(ctx)

	if err != nil {
		return false, err
	}

	defer tx.Rollback(ctx)

	err = tx.QueryRow(
		ctx,
		`UPDATE odyssey_ledger
        SET 
            status = 'executing'
        WHERE key = $1
            AND target = $2
            AND status = 'claimed'
        RETURNING TRUE`,
		e.key,
		e.target,
	).Scan(new(bool))

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil 
		}

		return false, err
	}

	err = tx.QueryRow(
		ctx,
		`UPDATE odyssey_journeys
        SET status = 'executing',
            updated_at = NOW()
        WHERE key = $1
          AND target = $2
          AND fencing_token = $3
          AND status = 'claimed'
        RETURNING status;`,
		e.key,
		e.target,
		e.metadata.fencingToken,
	).Scan(new(bool))

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil 
		}

		return false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}

	return true, nil
}

func (e *execution) complete(ctx context.Context) (bool, error){
	tx, err := e.conn.Begin(ctx)

	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(
		ctx,
		`UPDATE odyssey_ledger
        SET
            status = 'completed',
            completed_at = NOW()
        WHERE key = $1
            AND target = $2
            AND status = 'executing'
        RETURNING TRUE;`,
		e.key,
		e.target,
	).Scan(new(bool))

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows){
			return false, nil
		}

		return false, err
	}

	err = tx.QueryRow(
		ctx,
		`UPDATE odyssey_journeys
        SET
            status = 'completed',
            execution_result = $1,
            updated_at = NOW()
        WHERE key = $2
          AND target = $3
          AND fencing_token = $4
          AND status = 'executing'
        RETURNING TRUE;`,
		e.metadata.response,
		e.key,
		e.target,
		e.metadata.fencingToken,
	).Scan(new(bool))

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows){
			return false, nil
		}

		return false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}

	return true, nil
}

func (e *execution) abandon(ctx context.Context) (bool, error){
	tx, err := e.conn.Begin(ctx)

	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(
		ctx,
		`UPDATE odyssey_journeys
        SET expires_at = NOW(),
            updated_at = NOW()
        WHERE key = $1
            AND target = $2
            AND fencing_token = $3
            AND status = 'executing'
        RETURNING TRUE;`,
		e.key,
		e.target,
		e.metadata.fencingToken,
	).Scan(new(bool))

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows){
			return false, nil
		}

		return false, err
	}

	if err = tx.Commit(ctx); err != nil {
		return false, err
	}

	return true, nil
}