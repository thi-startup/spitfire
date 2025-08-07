package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thi-startup/spitfire/pkg/driver"
	"github.com/thi-startup/spitfire/pkg/log"
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
	cmd.AddCommand(dc.newInfoCmd())
	cmd.AddCommand(dc.newSetupCmd())
	cmd.AddCommand(dc.newVerifyCmd())
	cmd.AddCommand(dc.newInstructionsCmd())

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

// newInfoCmd creates the "spitfire driver info" command
func (dc *driverCmd) newInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info [driver]",
		Short: "Show detailed information about a specific driver",
		Long: `Display comprehensive information about a driver including status,
supported features, platform requirements, and configuration details.

This command shows all available metadata about the driver to help you
understand its capabilities and requirements.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			driverName := args[0]
			outputFormat, _ := cmd.Flags().GetString("output")
			dc.format = outputFormat

			return dc.runDriverInfo(driverName)
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format: table, json")

	return cmd
}

// runDriverInfo shows detailed information about a specific driver
func (dc *driverCmd) runDriverInfo(driverName string) error {
	// Get the driver definition
	driverDef := driver.GetDriver(driverName)
	if driverDef.Name == "" {
		return fmt.Errorf("driver %q not found", driverName)
	}

	log.Progress("Getting information for driver: %s", driverName)

	// Get driver state
	state := driverDef.Status()

	// For JSON output, create a structured response
	format, err := output.ParseOutputFormat(dc.format)
	if err != nil {
		return err
	}

	if format == output.JSONFormat {
		info := map[string]interface{}{
			"name":        driverDef.Name,
			"aliases":     driverDef.Aliases,
			"description": driverDef.Description,
			"priority":    driverDef.Priority.String(),
			"default":     driverDef.Default,
			"status": map[string]interface{}{
				"installed": state.Installed,
				"healthy":   state.Healthy,
				"running":   state.Running,
				"version":   state.Version,
			},
		}

		if state.Error != nil {
			info["status"].(map[string]interface{})["error"] = state.Error.Error()
		}

		printer := output.NewPrinter(format, nil)
		return printer.PrintJSON(info)
	}

	// Table format - human-readable output
	log.UserLn("Driver: %s", driverDef.Name)
	log.UserLn(strings.Repeat("=", len("Driver: ")+len(driverDef.Name)))
	log.UserLn("")

	log.UserLn("Basic Information:")
	log.UserLn("  Name:         %s", driverDef.Name)
	if len(driverDef.Aliases) > 0 {
		log.UserLn("  Aliases:      %s", strings.Join(driverDef.Aliases, ", "))
	}
	log.UserLn("  Description:  %s", driverDef.Description)
	log.UserLn("  Priority:     %s (%d/8)", driverDef.Priority.String(), int(driverDef.Priority))
	log.UserLn("  Default:      %s", dc.boolToYesNo(driverDef.Default))
	if state.Version != "" {
		log.UserLn("  Version:      %s", state.Version)
	}

	log.UserLn("")

	// Status section
	log.UserLn("Status:")
	log.UserLn("  Installed:    %s", dc.statusIcon(state.Installed))
	log.UserLn("  Healthy:      %s", dc.statusIcon(state.Healthy))
	log.UserLn("  Running:      %s", dc.statusIcon(state.Running))

	// Show errors if present
	if state.Error != nil {
		log.UserLn("")
		log.UserLn("Issues:")
		log.UserLn("  Error: %s", state.Error.Error())
		if state.Reason != "" {
			log.UserLn("  Reason: %s", state.Reason)
		}
		if state.Fix != "" {
			log.UserLn("  Fix: %s", state.Fix)
		}
	}

	log.UserLn("")

	// Quick Start section
	log.UserLn("Quick Start:")
	log.UserLn("  spitfire driver ls                   # List all drivers")
	log.UserLn("  spitfire driver setup %s        # Run automated setup", driverDef.Name)
	log.UserLn("  spitfire driver verify %s       # Verify setup status", driverDef.Name)

	return nil
}

// statusIcon returns a visual indicator for boolean status
func (dc *driverCmd) statusIcon(status bool) string {
	if status {
		return "✓ Yes"
	}
	return "✗ No"
}

// boolToYesNo converts boolean to Yes/No string
func (dc *driverCmd) boolToYesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

// newSetupCmd creates the "spitfire driver setup" command
func (dc *driverCmd) newSetupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup [driver]",
		Short: "Run automated setup for a specific driver",
		Long: `Run automated setup for the specified driver.

This will attempt to automatically configure system prerequisites like:
- KVM permissions and device access
- Network bridge and TAP device setup  
- Required system groups and permissions
- Firewall rules and routing configuration

Use --dry-run to see what would be done without making changes.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			driverName := args[0]
			interactive, _ := cmd.Flags().GetBool("interactive")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			force, _ := cmd.Flags().GetBool("force")
			components, _ := cmd.Flags().GetStringSlice("components")

			return dc.runSetup(driverName, interactive, dryRun, force, components)
		},
	}
	cmd.Flags().BoolP("interactive", "i", true, "Allow interactive prompts for user input")
	cmd.Flags().Bool("dry-run", false, "Show what would be done without making changes")
	cmd.Flags().Bool("force", false, "Skip confirmation prompts and proceed automatically")
	cmd.Flags().StringSlice("components", nil, "Specific components to setup (e.g., kvm,networking)")

	return cmd
}

// newVerifyCmd creates the "spitfire driver verify" command
func (dc *driverCmd) newVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify [driver]",
		Short: "Verify setup status for a specific driver",
		Long: `Check whether the specified driver is properly set up and ready to use.

This will verify all prerequisites and report any issues that need to be resolved.
Use this command to diagnose setup problems or confirm that setup completed successfully.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			driverName := args[0]
			outputFormat, _ := cmd.Flags().GetString("output")
			dc.format = outputFormat

			return dc.runSetupVerify(driverName)
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format: table, json")

	return cmd
}

// newInstructionsCmd creates the "spitfire driver instructions" command
func (dc *driverCmd) newInstructionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instructions [driver]",
		Short: "Show setup instructions for a specific driver",
		Long: `Display comprehensive setup instructions for the specified driver.

This shows both automated and manual setup steps, along with links to
relevant documentation and troubleshooting guides.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			driverName := args[0]
			return dc.runSetupInstructions(driverName)
		},
	}

	return cmd
}

// runSetup executes the setup process for a specific driver
func (dc *driverCmd) runSetup(driverName string, interactive, dryRun, force bool, components []string) error {
	// Get the driver definition
	driverDef := driver.GetDriver(driverName)
	if driverDef.Name == "" {
		return fmt.Errorf("driver %q not found", driverName)
	}

	log.Progress("Setting up driver: %s", driverName)

	// Create a minimal driver instance for setup
	// Note: This is just for setup - we provide minimal config to avoid validation errors
	driverConfig := &driver.Config{
		Name:   "setup-temp",
		CPUs:   1,
		Memory: 512, // 512MB
		Rootfs: "/dev/null", // Dummy rootfs just to pass validation
	}

	d, err := driverDef.Create(driverConfig)
	if err != nil {
		return fmt.Errorf("failed to create driver instance: %w", err)
	}

	// Prepare setup options
	opts := &driver.SetupOptions{
		Interactive: interactive,
		DryRun:      dryRun,
		Force:       force,
		Components:  components,
	}

	if dryRun {
		log.UserLn("Running in dry-run mode - no changes will be made")
	}

	// Run the setup
	ctx := context.Background()
	result, err := d.Setup(ctx, opts)
	if err != nil {
		return fmt.Errorf("setup failed: %w", err)
	}

	// Display results
	if result.Success {
		log.Success("Setup completed successfully!")
	} else {
		log.Warnf("Setup completed with issues")
	}

	// Show actions taken
	if len(result.Actions) > 0 {
		log.UserLn("\nActions performed:")
		for _, action := range result.Actions {
			status := "✓"
			if !action.Success {
				status = "✗"
			}
			log.UserLn("  %s %s: %s", status, action.Component, action.Description)
			if action.Command != "" {
				log.UserLn("    Command: %s", action.Command)
			}
			if !action.Success && action.Error != "" {
				log.UserLn("    Error: %s", action.Error)
			}
		}
	}

	// Show warnings
	if len(result.Warnings) > 0 {
		log.UserLn("\nWarnings:")
		for _, warning := range result.Warnings {
			log.UserLn("  ⚠ %s", warning)
		}
	}

	// Show next steps
	if len(result.NextSteps) > 0 {
		log.UserLn("\nNext steps:")
		for i, step := range result.NextSteps {
			log.UserLn("  %d. %s", i+1, step)
		}
	}

	return nil
}

// runSetupVerify checks the setup status for a specific driver
func (dc *driverCmd) runSetupVerify(driverName string) error {
	// Parse output format
	format, err := output.ParseOutputFormat(dc.format)
	if err != nil {
		return err
	}

	// Get the driver definition
	driverDef := driver.GetDriver(driverName)
	if driverDef.Name == "" {
		return fmt.Errorf("driver %q not found", driverName)
	}

	log.Progress("Verifying setup for driver: %s", driverName)

	// Create a minimal driver instance for verification
	driverConfig := &driver.Config{
		Name:   "verify-temp",
		CPUs:   1,
		Memory: 512, // 512MB
		Rootfs: "/dev/null", // Dummy rootfs just to pass validation
	}

	d, err := driverDef.Create(driverConfig)
	if err != nil {
		return fmt.Errorf("failed to create driver instance: %w", err)
	}

	// Verify setup
	ctx := context.Background()
	status, err := d.VerifySetup(ctx)
	if err != nil {
		return fmt.Errorf("setup verification failed: %w", err)
	}

	// Display results based on format
	if format == output.JSONFormat {
		printer := output.NewPrinter(format, nil)
		return printer.PrintJSON(status)
	}

	// Table format - show detailed status
	if status.Ready {
		log.Success("Driver %s is ready to use!", driverName)
	} else {
		log.Warnf("Driver %s has setup issues", driverName)
	}

	if status.Summary != "" {
		log.UserLn("\nSummary: %s", status.Summary)
	}

	// Show component status
	if len(status.Components) > 0 {
		log.UserLn("\nComponent Status:")
		
		var tableRows [][]string
		for _, component := range status.Components {
			statusStr := "✓ Ready"
			if !component.Ready {
				statusStr = "✗ Not Ready"
			}
			required := "Optional"
			if component.Required {
				required = "Required"
			}
			
			tableRows = append(tableRows, []string{
				component.Name,
				statusStr,
				required,
				component.Description,
			})
		}
		
		printer := output.NewPrinter(output.TableFormat, nil)
		headers := []string{"COMPONENT", "STATUS", "REQUIRED", "DESCRIPTION"}
		if err := printer.PrintTable(headers, tableRows); err != nil {
			return err
		}
	}

	// Show issues
	if len(status.Issues) > 0 {
		log.UserLn("\nIssues found:")
		for _, issue := range status.Issues {
			severity := strings.ToUpper(string(issue.Severity))
			log.UserLn("  [%s] %s: %s", severity, issue.Component, issue.Description)
			if issue.Resolution != "" {
				log.UserLn("    Resolution: %s", issue.Resolution)
			}
			if len(issue.Commands) > 0 {
				log.UserLn("    Commands to run:")
				for _, cmd := range issue.Commands {
					log.UserLn("      %s", cmd)
				}
			}
		}
	}

	return nil
}

// runSetupInstructions displays setup instructions for a specific driver
func (dc *driverCmd) runSetupInstructions(driverName string) error {
	// Get the driver definition
	driverDef := driver.GetDriver(driverName)
	if driverDef.Name == "" {
		return fmt.Errorf("driver %q not found", driverName)
	}

	log.Progress("Getting setup instructions for driver: %s", driverName)

	// Create a minimal driver instance for instructions
	driverConfig := &driver.Config{
		Name:   "instructions-temp",
		CPUs:   1,
		Memory: 512, // 512MB
		Rootfs: "/dev/null", // Dummy rootfs just to pass validation
	}

	d, err := driverDef.Create(driverConfig)
	if err != nil {
		return fmt.Errorf("failed to create driver instance: %w", err)
	}

	// Get setup instructions
	ctx := context.Background()
	instructions, err := d.GetSetupInstructions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get setup instructions: %w", err)
	}

	// Display instructions
	log.UserLn("Setup Instructions for %s Driver", driverName)
	log.UserLn("=====================================")

	if instructions.Overview != "" {
		log.UserLn("\nOverview:")
		log.UserLn("%s", instructions.Overview)
	}

	if len(instructions.Prerequisites) > 0 {
		log.UserLn("\nPrerequisites:")
		for i, prereq := range instructions.Prerequisites {
			log.UserLn("  %d. %s", i+1, prereq)
		}
	}

	if len(instructions.AutomatedSteps) > 0 {
		log.UserLn("\nAutomated Setup Steps:")
		log.UserLn("(These can be run with: spitfire driver setup %s)", driverName)
		for i, step := range instructions.AutomatedSteps {
			required := ""
			if step.Required {
				required = " (required)"
			}
			sudo := ""
			if step.Sudo {
				sudo = " (requires sudo)"
			}
			log.UserLn("  %d. %s%s%s", i+1, step.Name, required, sudo)
			log.UserLn("     %s", step.Description)
			if len(step.Commands) > 0 {
				log.UserLn("     Commands:")
				for _, cmd := range step.Commands {
					log.UserLn("       %s", cmd)
				}
			}
		}
	}

	if len(instructions.ManualSteps) > 0 {
		log.UserLn("\nManual Setup Steps:")
		log.UserLn("(These require manual intervention)")
		for i, step := range instructions.ManualSteps {
			required := ""
			if step.Required {
				required = " (required)"
			}
			log.UserLn("  %d. %s%s", i+1, step.Name, required)
			log.UserLn("     %s", step.Description)
			if len(step.Commands) > 0 {
				log.UserLn("     Commands:")
				for _, cmd := range step.Commands {
					log.UserLn("       %s", cmd)
				}
			}
			if step.Verification != "" {
				log.UserLn("     Verification: %s", step.Verification)
			}
		}
	}

	if len(instructions.PostSetup) > 0 {
		log.UserLn("\nPost-Setup:")
		for i, step := range instructions.PostSetup {
			log.UserLn("  %d. %s", i+1, step)
		}
	}

	if len(instructions.Documentation) > 0 {
		log.UserLn("\nDocumentation:")
		for _, doc := range instructions.Documentation {
			log.UserLn("  • %s: %s", doc.Title, doc.URL)
			if doc.Description != "" {
				log.UserLn("    %s", doc.Description)
			}
		}
	}

	if len(instructions.Troubleshooting) > 0 {
		log.UserLn("\nTroubleshooting:")
		for _, doc := range instructions.Troubleshooting {
			log.UserLn("  • %s: %s", doc.Title, doc.URL)
			if doc.Description != "" {
				log.UserLn("    %s", doc.Description)
			}
		}
	}

	return nil
}
