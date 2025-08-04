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

// newSetupCommand creates the setup subcommand with all its sub-commands
func newSetupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Driver-specific system setup and verification",
		Long: `Manage driver-specific system setup requirements.

This command helps prepare your system for running VMs with different hypervisor
drivers. Each driver has its own setup requirements (KVM permissions, networking,
etc.) and this command provides automated setup and verification.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// Setup run command
	runCmd := &cobra.Command{
		Use:   "run [driver]",
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

			return runSetup(driverName, interactive, dryRun, force, components)
		},
	}
	runCmd.Flags().BoolP("interactive", "i", true, "Allow interactive prompts for user input")
	runCmd.Flags().Bool("dry-run", false, "Show what would be done without making changes")
	runCmd.Flags().Bool("force", false, "Skip confirmation prompts and proceed automatically")
	runCmd.Flags().StringSlice("components", nil, "Specific components to setup (e.g., kvm,networking)")

	// Setup verify command
	verifyCmd := &cobra.Command{
		Use:   "verify [driver]",
		Short: "Verify setup status for a specific driver",
		Long: `Check whether the specified driver is properly set up and ready to use.

This will verify all prerequisites and report any issues that need to be resolved.
Use this command to diagnose setup problems or confirm that setup completed successfully.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			driverName := args[0]
			outputFormat, _ := cmd.Flags().GetString("output")

			return runSetupVerify(driverName, outputFormat)
		},
	}
	verifyCmd.Flags().StringP("output", "o", "table", "Output format: table, json")

	// Setup instructions command  
	instructionsCmd := &cobra.Command{
		Use:   "instructions [driver]",
		Short: "Show setup instructions for a specific driver",
		Long: `Display comprehensive setup instructions for the specified driver.

This shows both automated and manual setup steps, along with links to
relevant documentation and troubleshooting guides.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			driverName := args[0]
			return runSetupInstructions(driverName)
		},
	}

	cmd.AddCommand(runCmd, verifyCmd, instructionsCmd)
	return cmd
}

// runSetup executes the setup process for a specific driver
func runSetup(driverName string, interactive, dryRun, force bool, components []string) error {
	// Get the driver definition
	driverDef := driver.GetDriver(driverName)
	if driverDef.Name == "" {
		return fmt.Errorf("driver %q not found", driverName)
	}

	log.Progress("Setting up driver: %s", driverName)

	// Create a temporary driver instance for setup
	driverConfig := &driver.Config{
		Name:   "setup-temp",
		CPUs:   1,
		Memory: 512, // 512MB
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
func runSetupVerify(driverName, outputFormat string) error {
	// Parse output format
	format, err := output.ParseOutputFormat(outputFormat)
	if err != nil {
		return err
	}

	// Get the driver definition
	driverDef := driver.GetDriver(driverName)
	if driverDef.Name == "" {
		return fmt.Errorf("driver %q not found", driverName)
	}

	log.Progress("Verifying setup for driver: %s", driverName)

	// Create a temporary driver instance for verification
	driverConfig := &driver.Config{
		Name:   "verify-temp",
		CPUs:   1,
		Memory: 512, // 512MB
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
func runSetupInstructions(driverName string) error {
	// Get the driver definition
	driverDef := driver.GetDriver(driverName)
	if driverDef.Name == "" {
		return fmt.Errorf("driver %q not found", driverName)
	}

	log.Progress("Getting setup instructions for driver: %s", driverName)

	// Create a temporary driver instance
	driverConfig := &driver.Config{
		Name:   "instructions-temp",
		CPUs:   1,
		Memory: 512, // 512MB
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
		log.UserLn("(These can be run with: spitfire setup run %s)", driverName)
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