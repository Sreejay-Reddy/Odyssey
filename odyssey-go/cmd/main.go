package main

import (
	"context"
	"log"
	"os"

	"odyssey-go"
	"odyssey-go/internal/config"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	cfg := config.Config{}

	client := odyssey.NewClient(dbURL, cfg)

	if err := client.InitDB(context.Background()); err != nil {
		log.Fatal(err)
	}

	log.Println("Odyssey database initialized")
}