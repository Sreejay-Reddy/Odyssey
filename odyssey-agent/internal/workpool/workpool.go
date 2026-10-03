package workpool

import (
	"context"
	"fmt"
	"time"

	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/registry"
	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/storage"
)

type WorkPool struct {
	Pools           map[string]chan []storage.Execution
    workerIDs       []string

	r 				*registry.Registry
	w 				storage.Writer

	limit 			int	
	MaxSize			int
	interval		time.Duration

	start 			chan struct{}
}

func New(
    registry *registry.Registry,
    writer storage.Writer,
    workerIDs []string,
    limit int,
    maxSize int,
    interval time.Duration,
) *WorkPool {

    pools := make(map[string]chan []storage.Execution, len(workerIDs))

    for _, workerID := range workerIDs {
        pools[workerID] = make(chan []storage.Execution, maxSize)
    }

    if len(workerIDs) == 0 {
        panic("workpool requires at least one worker")
    }

    return &WorkPool{
        Pools:    pools,
        workerIDs: workerIDs,
        r:        registry,
        w:        writer,
        limit:    limit,
        MaxSize:  maxSize,
        interval: interval,
        start:    make(chan struct{}, 1),
    }
}

func (w *WorkPool) Publish(
    ctx context.Context,
    workerID string,
    executions []storage.Execution,
) error {
    pool, ok := w.Pools[workerID]
    if !ok {
        return fmt.Errorf("unknown worker: %s", workerID)
    }

    if len(executions) == 0 {
        return nil
    }

    select {
    case pool <- executions:
        return nil

    case <-ctx.Done():
        return ctx.Err()
    }
}

func (w *WorkPool) StartWorkPool(ctx context.Context) error {
	backoff := w.interval
	maxBackoff := w.interval * 5

	for {
		executions, err := w.w.Acquire(
			ctx,
			w.r,
			w.workerIDs,
			w.limit,
		)
		if err != nil {
			return err
		}

		total := 0

		for workerID, batch := range executions {
			if len(batch) == 0 {
				continue
			}

			if err := w.Publish(ctx, workerID, batch); err != nil {
				return err
			}

			total += len(batch)
		}

		if total > 0 {
			backoff = w.interval
			continue
		}

		timer := time.NewTimer(backoff)

		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()

		case <-timer.C:
		}

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func (w *WorkPool) Pull(
    ctx context.Context,
    workerID string,
) ([]storage.Execution, error) {

    pool, exists := w.Pools[workerID]
    if !exists {
        return nil, fmt.Errorf("unknown worker: %s", workerID)
    }

    select {
    case <-ctx.Done():
        return nil, ctx.Err()

    case executions := <-pool:
        if len(pool) <= w.MaxSize/2 {
            select {
            case w.start <- struct{}{}:
            default:
            }
        }

        return executions, nil
    }
}