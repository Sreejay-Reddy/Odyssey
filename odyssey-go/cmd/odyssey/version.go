package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

const version = "0.0.6"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show Odyssey version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Odyssey v%s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}