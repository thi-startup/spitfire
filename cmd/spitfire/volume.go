package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thi-startup/spitfire/pkg/output"
)

// newVolumeCommands creates the volume subcommand with all its sub-commands
func newVolumeCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "volume",
		Short: "Volume management",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// Volume ls command with output format support
	lsCmd := &cobra.Command{
		Use:   "ls",
		Short: "List volumes",
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFormat, _ := cmd.Flags().GetString("output")
			quiet, _ := cmd.Flags().GetBool("quiet")
			return runVolumeList(outputFormat, quiet)
		},
	}
	lsCmd.Flags().StringP("output", "o", "table", "Output format. One of: table, json")
	lsCmd.Flags().BoolP("quiet", "q", false, "Only display volume names")

	// Volume create command (stub for now)
	createCmd := &cobra.Command{
		Use:   "create [NAME]",
		Short: "Create a volume",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented yet")
		},
	}

	cmd.AddCommand(lsCmd, createCmd)
	return cmd
}

// runVolumeList implements the volume ls command
// This demonstrates how easy it is to add new listing commands with our modular interface
func runVolumeList(outputFormat string, quiet bool) error {
	// Parse output format
	format, err := output.ParseOutputFormat(outputFormat)
	if err != nil {
		return err
	}

	// Mock data for demonstration - in real implementation this would come from storage
	mockVolumes := []struct {
		name      string
		driver    string
		size      string
		created   string
		mountPath string
	}{
		{"db-data", "local", "10GB", "2025-08-03 01:30:00", "/var/lib/postgresql/data"},
		{"web-assets", "nfs", "5GB", "2025-08-03 01:31:00", "/var/www/html"},
		{"logs", "local", "2GB", "2025-08-03 01:32:00", "/var/log/app"},
	}

	// Prepare data for output
	var volumeNames []string
	var tableRows [][]string

	for _, vol := range mockVolumes {
		volumeNames = append(volumeNames, vol.name)
		tableRows = append(tableRows, []string{
			vol.name,
			vol.driver,
			vol.size,
			vol.created,
			vol.mountPath,
		})
	}

	// Handle output - same clean pattern as VM listing
	printer := output.NewPrinter(format, nil)
	
	if quiet {
		output.PrintQuiet(volumeNames, nil)
		return nil
	}

	// Print using the same generic interface - no volume-specific code needed!
	headers := []string{"NAME", "DRIVER", "SIZE", "CREATED", "MOUNT POINT"}
	return printer.PrintTable(headers, tableRows)
}