package odyssey

import (
    "context"
    "github.com/jackc/pgx/v5"
)

type Client struct{
	dbURL string
}

func NewClient(dbURL string) *Client {
    return &Client{
        dbURL: dbURL,
    }
}

func (c *Client) connect(ctx context.Context) (*pgx.Conn, error) {
    return pgx.Connect(ctx, c.dbURL)
}

func (c *Client) initDB(ctx context.Context) error{
    conn, err := c.connect(ctx)
    if err != nil{
        return err
    }
    defer conn.Close(ctx)

    tx, err := conn.Begin(ctx)
    if err != nil{
        return err
    }
    defer tx.Rollback(ctx)

    _, err = tx.Exec(ctx, schemaSQL)
    if err != nil{
        return err
    }

    return tx.Commit(ctx)
}