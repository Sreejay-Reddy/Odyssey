package postgres

import (
	"context"
	"embed"
)

//go:embed schema.sql bulkacquire.sql
var sqlFS embed.FS

func (w *Writer) InitDB(ctx context.Context) error {
	schema, err := sqlFS.ReadFile("schema.sql")
	if err != nil {
		return err
	}

	if _, err := w.pool.Exec(ctx, string(schema)); err != nil {
		return err
	}

	bulkAcquire, err := sqlFS.ReadFile("bulkacquire.sql")
	if err != nil {
		return err
	}

	_, err = w.pool.Exec(ctx, string(bulkAcquire))
	return err
}