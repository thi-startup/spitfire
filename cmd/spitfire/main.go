package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thi-startup/spitfire/cmd/spitfire/vmctl"
	"github.com/thi-startup/spitfire/cmd/spitfire/volume"
)

var (
	version   string
	hash      string
	buildTime string
)

func main() {
	root := &cobra.Command{
		Use:     "spitfire",
		Version: fmt.Sprintf("Version:\t%s+%s\nBuildTime:\t%s\n", version, hash, buildTime),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	root.SetVersionTemplate(`{{printf "%s" .Version}}`)

	root.AddCommand(volume.Root(), vmctl.Root())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
