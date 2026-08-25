package main

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "odyssey",
	Short: "Durable execution engine",
	Long:  "Odyssey is a durable execution engine built around PostgreSQL.",
}