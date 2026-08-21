package main

import (
	"context"
	"log"
	"os"

	"github.com/sreejay-reddy/odyssey/odyssey-go"
	"github.com/sreejay-reddy/odyssey/odyssey-go/configutil"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	cfg := configutil.Config{}

	client := odyssey.NewClient(dbURL, cfg)

	if err := client.InitDB(context.Background()); err != nil {
		log.Fatal(err)
	}

	log.Println("Odyssey database initialized")
}