package batcher

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/registry"
	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/storage"
    "github.com/sreejay-reddy/odyssey/odyssey-agent/internal/workpool"
)

type Batcher struct {
    writer   storage.Writer
    registry *registry.Registry
    limit    int
    interval time.Duration
    nextID   atomic.Uint64
}

type Batch struct {
    ID         uint64
    Executions []storage.Execution
}

func New(
    writer storage.Writer,
    registry *registry.Registry,
    limit int,
    interval time.Duration,
) *Batcher {
    return &Batcher{
        writer:   writer,
        registry: registry,
        limit:    limit,
        interval: interval,
    }
}

func (b *Batcher) Next(
    ctx context.Context,
    wp *workpool.WorkPool,
    workerID string,
) (Batch, error) {

    executions, err := wp.Pull(ctx, workerID)
    if err != nil {
        return Batch{}, err
    }

    batchID := b.nextID.Add(1)

    return Batch{
        ID:         batchID,
        Executions: executions,
    }, nil
}


func (b *Batcher) BatchComplete(ctx context.Context, executions []storage.Execution) (error) {
    err := b.writer.Complete(ctx, executions)
    if err != nil {
        return err
    }
    
    return nil
}