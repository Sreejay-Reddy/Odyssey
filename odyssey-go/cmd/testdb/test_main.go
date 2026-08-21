package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5"

	"github.com/sreejay-reddy/odyssey/odyssey-go"
	"github.com/sreejay-reddy/odyssey/odyssey-go/configutil"
)

func main() {
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, `
		DROP TABLE IF EXISTS
			odyssey_deliveries,
			odyssey_journeys,
			odyssey_ledger
		CASCADE;

		DROP TYPE IF EXISTS
			delivery_status,
			odyssey_execution_mode,
			odyssey_status
		CASCADE;

		DROP SEQUENCE IF EXISTS odyssey_token_seq;
	`)
	if err != nil {
		log.Fatal(err)
	}

	client := odyssey.NewClient(dbURL, configutil.Config{})

	if err := client.InitDB(ctx); err != nil {
		log.Fatal(err)
	}

	log.Println("Odyssey test database reset and initialized")
}