package postgres

import (
	"github.com/jackc/pgx/v5"
	
	"github.com/sreejay-reddy/odyssey/odyssey-agent/internal/config"
)

type Writer struct {
	conn *pgx.Conn
	cfg config.Config
}

func New(conn *pgx.Conn, cfg config.Config) *Writer {
	return &Writer{
		conn: conn,
		cfg: cfg,
	}
}