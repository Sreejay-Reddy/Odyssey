package main

import (
	"context"
	"fmt"
	"os"

	odyssey "github.com/sreejay-reddy/odyssey/odyssey-go"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the Odyssey database",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbURL := os.Getenv("DATABASE_URL")

		if dbURL == "" {
			return fmt.Errorf("DATABASE_URL is required")
		}

		client, err := odyssey.NewClient(dbURL)
		if err != nil {
			return err
		}

		if err := client.InitDB(context.Background()); err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}

		fmt.Println("Odyssey database initialized successfully")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}