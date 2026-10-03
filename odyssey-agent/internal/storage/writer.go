package storage

import (
	"context"
	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/registry"
)

type Writer interface {
	InitDB(ctx context.Context) error
	Acquire(ctx context.Context, registry *registry.Registry, workerID []string, limit int) (map[string][]Execution, error)
	Complete(ctx context.Context, executions []Execution) error
}