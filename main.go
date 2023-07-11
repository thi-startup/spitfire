package main

import (
	"os"

	"github.com/spf13/cobra"
	cmd "github.com/thi-startup/spitfire/cmd/mkroot"
)

func main() {
	rootCmd := &cobra.Command{
		Use: "spitfire",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	rootCmd.AddCommand(cmd.Mkroot())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
