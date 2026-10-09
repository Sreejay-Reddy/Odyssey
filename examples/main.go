package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/sreejay-reddy/odyssey/odyssey-go"
)

const totalJobs = 100000

type HelloInput struct {
	Name string `json:"name"`
}

type HelloOutput struct {
	Message string `json:"message"`
}

var executed atomic.Int64

func Hello(ctx context.Context, input HelloInput) (HelloOutput, error) {
	executed.Add(1)

	return HelloOutput{
		Message: "Hello, " + input.Name,
	}, nil
}

// func seed(ctx context.Context, dbURL string) error {
// 	conn, err := pgx.Connect(ctx, dbURL)
// 	if err != nil {
// 		return err
// 	}
// 	defer conn.Close(ctx)

// 	// Clean previous benchmark.
// 	_, err = conn.Exec(ctx, `
// 		DELETE FROM odyssey_journeys
// 		WHERE key LIKE 'bench-%';

// 	DELETE FROM odyssey_ledger
// 		WHERE key LIKE 'bench-%';
// 	`)
// 	if err != nil {
// 		return err
// 	}

// 	tx, err := conn.Begin(ctx)
// 	if err != nil {
// 		return err
// 	}
// 	defer tx.Rollback(ctx)

// 	for i := 0; i < totalJobs; i++ {
// 		key := fmt.Sprintf("bench-%06d", i)

// 		_, err = tx.Exec(ctx, `
// 			INSERT INTO odyssey_ledger (
// 				key,
// 				status
// 			)
// 			VALUES ($1, 'queued')
// 		`, key)
// 		if err != nil {
// 			return err
// 		}

// 		_, err = tx.Exec(ctx, `
// 			INSERT INTO odyssey_journeys (
// 				key,
// 				target,
// 				sequence,
// 				mode,
// 				input,
// 				status,
// 				attempts
// 			)
// 			VALUES (
// 				$1,
// 				'hello',
// 				0,
// 				'local',
// 				'{"name":"Benchmark"}'::jsonb,
// 				'queued',
// 				0
// 			)
// 		`, key)
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	return tx.Commit(ctx)
// }

func waitForCompletion(ctx context.Context, dbURL string) error {
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		var completed int

		err := conn.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM odyssey_journeys
			WHERE key LIKE 'bench-%'
			  AND status = 'completed'
		`).Scan(&completed)

		if err != nil {
			return err
		}

		if completed == totalJobs {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	ctx := context.Background()

	client, err := odyssey.NewClient(dbURL)
	if err != nil {
		log.Fatal(err)
	}

	if err := client.InitDB(ctx); err != nil {
		log.Fatal(err)
	}

	if err := client.Register("hello", Hello, 30000); err != nil {
		log.Fatal(err)
	}

	log.Println("Starting benchmark...")

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start SDK.
	serveErr := make(chan error, 1)

	go func() {
		serveErr <- client.Serve(runCtx)
	}()

	// Give SDK a moment to establish its sockets.
	time.Sleep(500 * time.Millisecond)

	start := time.Now()

	// Wait for every journey to reach completed.
	if err := waitForCompletion(runCtx, dbURL); err != nil {
		log.Fatal(err)
	}

	elapsed := time.Since(start)

	cancel()

	select {
	case err := <-serveErr:
		if err != nil && err != context.Canceled {
			log.Printf("SDK stopped: %v", err)
		}
	case <-time.After(time.Second):
	}

	jobsPerSecond := float64(totalJobs) / elapsed.Seconds()

	fmt.Println()
	fmt.Println("========== Odyssey Benchmark ==========")
	fmt.Printf("Jobs:          %d\n", totalJobs)
	fmt.Printf("Elapsed:       %s\n", elapsed)
	fmt.Printf("Jobs/sec:      %.2f\n", jobsPerSecond)
	fmt.Printf("SDK executions: %d\n", executed.Load())
	fmt.Println("=======================================")
}