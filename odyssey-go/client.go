package odyssey

import (
    "context"

    "github.com/jackc/pgx/v5"
    "github.com/sreejay-reddy/odyssey/odyssey-go/internal/buildledger"
    "github.com/sreejay-reddy/odyssey/odyssey-go/internal/registry"
    "github.com/sreejay-reddy/odyssey/odyssey-go/configutil"
)

type Client struct{
	dbURL string
    config configutil.Config
}

func NewClient(dbURL string, cfg configutil.Config) *Client {
    return &Client{
        dbURL: dbURL,
        config: cfg,
    }
}

func (c *Client) connect(ctx context.Context) (*pgx.Conn, error) {
    return pgx.Connect(ctx, c.dbURL)
}

func (c *Client) InitDB(ctx context.Context) error {
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

func (c *Client) Register(target string, fn any, ttlMS int64) error {
    return registry.Register(c.config, target, fn, ttlMS)
}

func (c *Client) BuildLedger(
    ctx context.Context, 
    key string, 
    steps []buildledger.Step) (error) {
        conn, err := c.connect(ctx)
        if err != nil {
            return err
        }
        defer conn.Close(ctx)

        _ , err = buildledger.BuildLedger(
            ctx,
            conn,
            c.config,
            key, 
            steps,
        )

        return err
}

func (c *Client) Serve(addr string) error {
    server := Server{
        client: c,
    }

    return server.Serve(addr)
}