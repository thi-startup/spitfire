package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thi-startup/spitfire/pkg/driver"
	"github.com/thi-startup/spitfire/pkg/output"
)

// driverCmd encapsulates state and shared methods for driver subcommands
type driverCmd struct {
	// Add fields here for any state shared across driver subcommands (eg. output format, paging, etc.)
	format string
}

// newDriverCmd creates the `spitfire driver` parent command and subcommands
func newDriverCmd() *cobra.Command {
	dc := &driverCmd{
		format: "table",
	}

	cmd := &cobra.Command{
		Use:   "driver",
		Short: "Manage virtualization drivers and host requirements",
		Long: `Show information about supported drivers, run setup, inspect driver status,
and perform diagnostics. Drivers provide the virtualization backend for Spitfire VMs.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(dc.newLsCmd())
	// Placeholders for future: setup, verify, instructions, etc.

	return cmd
}

// newLsCmd creates the "spitfire driver ls" command (list drivers)
func (dc *driverCmd) newLsCmd() *cobra.Command {
	lsCmd := &cobra.Command{
		Use:     "ls",
		Aliases: []string{"list"},
		Short:   "List available drivers and their status",
		Long:    "Show all virtualization drivers known to this Spitfire installation, including status and health.",
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFormat, _ := cmd.Flags().GetString("output")
			dc.format = outputFormat
			return dc.runListDrivers()
		},
	}

	lsCmd.Flags().StringP("output", "o", "table", "Output format. One of: table, json")

	return lsCmd
}

// runListDrivers implements the logic for `spitfire driver ls`
func (dc *driverCmd) runListDrivers() error {
	format, err := output.ParseOutputFormat(dc.format)
	if err != nil {
		return err
	}

	drivers := driver.Available() // []driver.DriverState

	headers := []string{"NAME", "PRIORITY", "STATUS", "VERSION", "DEFAULT", "DESCRIPTION"}
	rows := [][]string{}

	for _, d := range drivers {
		priority := fmt.Sprintf("%d", d.Priority)
		status := "unhealthy"
		if d.State.Healthy && d.State.Installed {
			status = "healthy"
		} else if !d.State.Installed {
			status = "not installed"
		}

		def := ""
		if d.Default {
			def = "yes"
		}

		version := d.State.Version
		desc := d.Description

		rows = append(rows, []string{
			d.Name,
			priority,
			status,
			version,
			def,
			desc,
		})
	}

	printer := output.NewPrinter(format, nil)
	return printer.PrintTable(headers, rows)
}
