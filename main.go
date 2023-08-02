package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	cmd "github.com/thi-startup/spitfire/cmd/mkroot"
)

var (
	version   string
	hash      string
	buildTime string
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "spitfire",
		Version: fmt.Sprintf("Version:\t%s+%s\nBuildTime:\t%s\n", version, hash, buildTime),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	rootCmd.SetVersionTemplate(`{{printf "%s" .Version}}`)

	rootCmd.AddCommand(cmd.Mkroot())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
