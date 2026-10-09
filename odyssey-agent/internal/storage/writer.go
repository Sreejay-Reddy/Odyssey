package storage

import (
	"context"
	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/registry"
)

type Writer interface {
	Acquire(ctx context.Context, registry *registry.Registry, workerID string, limit int) ([]Execution, error)
	Acquirecte(ctx context.Context, registry *registry.Registry, workerID string, limit int) (_ []Execution, retErr error)
	AcquireFunc(ctx context.Context, registry *registry.Registry, workerID string, limit int) (_ []Execution, retErr error)
	Complete(ctx context.Context, executions []Execution) error
	InitDB(ctx context.Context) error 
}