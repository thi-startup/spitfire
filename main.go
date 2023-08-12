package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/thi-startup/spitfire/cmd"
)

func init() {
	log.SetOutput(os.Stderr)
	log.SetLevel(log.DebugLevel)
}

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

	rootCmd.AddCommand(cmd.MakeCreateCmd())
	rootCmd.AddCommand(cmd.MakeInitCmd())
	rootCmd.AddCommand(cmd.MkRunCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
