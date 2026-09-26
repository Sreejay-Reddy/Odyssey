package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sreejay-reddy/odyssey/odyssey-go"
)

type HelloInput struct {
	Name string `json:"name"`
}

type HelloOutput struct {
	Message string `json:"message"`
}

func Hello(ctx context.Context, input HelloInput) (HelloOutput, error) {
	return HelloOutput{
		Message: "Hello, " + input.Name,
	}, nil
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	client, err := odyssey.NewClient(dbURL)
	if err != nil {
		log.Fatal(err)
	}

	if err := client.InitDB(context.Background()); err != nil {
		log.Fatal(err)
	}

	if err := client.Register("hello", Hello, 30000); err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	log.Println("Starting Odyssey SDK")

	if err := client.Serve(ctx); err != nil {
		log.Println("Odyssey stopped:", err)
	}
}