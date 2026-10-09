package postgres

import (
	"context"
	"embed"
)

//go:embed schema.sql
var sqlFS embed.FS

func (w *Writer) InitDB(ctx context.Context) error {
	schema, err := sqlFS.ReadFile("schema.sql")
	if err != nil {
		return err
	}

	if _, err := w.pool.Exec(ctx, string(schema)); err != nil {
		return err
	}
	
	return nil
}