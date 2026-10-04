package storage

import (
	"context"
	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/registry"
)

type Writer interface {
	Acquire(ctx context.Context, registry *registry.Registry, workerID string, limit int) ([]Execution, error)
	Complete(ctx context.Context, executions []Execution) error
}