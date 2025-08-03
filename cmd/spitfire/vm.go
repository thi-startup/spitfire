package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/thi-startup/spitfire/pkg/driver"
	"github.com/thi-startup/spitfire/pkg/driver/firecracker"
	"github.com/thi-startup/spitfire/pkg/log"
	"github.com/thi-startup/spitfire/pkg/output"
	"github.com/thi-startup/spitfire/pkg/state"
)

// newVMCommands creates the VM subcommand with all its sub-commands
func newVMCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vm",
		Short: "VM lifecycle management",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// VM up command
	upCmd := &cobra.Command{
		Use:   "up",
		Short: "Start services defined in config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile, _ := cmd.Flags().GetString("file")
			detach, _ := cmd.Flags().GetBool("detach")

			return runVMUp(configFile, detach)
		},
	}
	upCmd.Flags().StringP("file", "f", "", "Specify an alternate spitfire config file (default: spitfire.yaml)")
	upCmd.Flags().BoolP("detach", "d", false, "Detached mode: Run containers in the background")

	// VM down command
	downCmd := &cobra.Command{
		Use:   "down",
		Short: "Stop services",
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile, _ := cmd.Flags().GetString("file")
			removeVolumes, _ := cmd.Flags().GetBool("volumes")

			return runVMDown(configFile, removeVolumes)
		},
	}
	downCmd.Flags().StringP("file", "f", "", "Specify an alternate spitfire config file (default: spitfire.yaml)")
	downCmd.Flags().BoolP("volumes", "v", false, "Remove named volumes declared in the volumes section")

	// VM ps command
	psCmd := &cobra.Command{
		Use:   "ps",
		Short: "List running VMs",
		RunE: func(cmd *cobra.Command, args []string) error {
			all, _ := cmd.Flags().GetBool("all")
			quiet, _ := cmd.Flags().GetBool("quiet")
			outputFormat, _ := cmd.Flags().GetString("output")

			return runVMPs(all, quiet, outputFormat)
		},
	}
	psCmd.Flags().BoolP("all", "a", false, "Show all VMs (default shows just running)")
	psCmd.Flags().BoolP("quiet", "q", false, "Only display VM IDs")
	psCmd.Flags().StringP("output", "o", "table", "Output format. One of: table, json")

	cmd.AddCommand(upCmd, downCmd, psCmd)
	return cmd
}

// runVMPs implements the vm ps command
func runVMPs(showAll bool, quiet bool, outputFormat string) error {
	// Parse output format
	format, err := output.ParseOutputFormat(outputFormat)
	if err != nil {
		return err
	}

	// Get current working directory for state management
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Initialize state manager
	projectName := filepath.Base(cwd)
	stateManager := state.NewManager(projectName)

	// Load project state
	if !stateManager.ProjectExists() {
		if !quiet && format == output.TableFormat {
			log.UserLn("No VMs found")
		}
		return nil
	}

	projectState, err := stateManager.LoadProjectState()
	if err != nil {
		return fmt.Errorf("failed to load project state: %w", err)
	}

	if len(projectState.VMs) == 0 {
		if !quiet && format == output.TableFormat {
			log.UserLn("No VMs found")
		}
		return nil
	}

	ctx := context.Background()
	var vmNames []string
	var tableRows [][]string

	for vmName, vmRuntime := range projectState.VMs {
		// Get current state
		d, err := firecracker.NewFirecrackerDriver(vmRuntime.DriverConfig)
		currentState := vmRuntime.State // fallback to stored state
		var currentIP string

		if err == nil {
			if state, stateErr := d.GetState(ctx); stateErr == nil {
				currentState = state
			}
			if ip, ipErr := d.GetIP(ctx); ipErr == nil && ip != nil {
				currentIP = ip.String()
			}
		}

		// Skip stopped VMs unless --all is specified
		if !showAll && currentState != driver.Running {
			continue
		}

		// Collect VM name for quiet mode
		vmNames = append(vmNames, vmName)

		// Prepare table row data
		if currentIP == "" {
			currentIP = "-"
		}
		
		createdStr := vmRuntime.CreatedAt.Format("2006-01-02 15:04:05")
		memory := fmt.Sprintf("%dMB", vmRuntime.DriverConfig.Memory)
		cpus := fmt.Sprintf("%d", vmRuntime.DriverConfig.CPUs)

		tableRows = append(tableRows, []string{
			vmName,
			currentState.String(),
			currentIP,
			createdStr,
			vmRuntime.Driver,
			memory,
			cpus,
		})
	}

	// Handle output based on mode
	printer := output.NewPrinter(format, nil)
	
	if quiet {
		output.PrintQuiet(vmNames, nil)
		return nil
	}

	// Print using the clean generic interface
	headers := []string{"NAME", "STATE", "IP", "CREATED", "DRIVER", "MEMORY", "CPUS"}
	return printer.PrintTable(headers, tableRows)
}

// runVMUp implements the vm up command
func runVMUp(configFile string, detach bool) error {
	// Find and load config file
	configFile, err := findConfigFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to find config file: %w", err)
	}

	log.Progress("Loading configuration from %s", configFile)
	cfg, err := loadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get current working directory for state management
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Initialize state manager
	projectName := filepath.Base(cwd)
	stateManager := state.NewManager(projectName)

	ctx := context.Background()

	// Check if project already exists, if not initialize it
	if stateManager.ProjectExists() {
		_, err = stateManager.LoadProjectState()
		if err != nil {
			return fmt.Errorf("failed to load project state: %w", err)
		}
	} else {
		_, err = stateManager.InitializeProject(configFile, cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize project: %w", err)
		}
	}

	// Process each VM
	for vmName, vm := range cfg.VMs {
		log.Progress("Creating VM '%s'", vmName)

		// Convert VM config to driver config
		driverConfig, err := convertVMToDriverConfig(vmName, vm, cfg)
		if err != nil {
			return fmt.Errorf("failed to convert VM config for %s: %w", vmName, err)
		}

		// Create the driver
		d, err := firecracker.NewFirecrackerDriver(driverConfig)
		if err != nil {
			return fmt.Errorf("failed to create driver for %s: %w", vmName, err)
		}

		// Create the VM
		if err := d.Create(ctx); err != nil {
			return fmt.Errorf("failed to create VM %s: %w", vmName, err)
		}

		// Start the VM
		log.Progress("Starting VM '%s'", vmName)
		if err := d.Start(ctx); err != nil {
			return fmt.Errorf("failed to start VM %s: %w", vmName, err)
		}

		// Update VM state
		err = stateManager.UpdateVMState(vmName, func(vmRuntime *state.VMRuntime) error {
			// Get current VM info
			vmInfo, err := d.GetInfo(ctx)
			if err != nil {
				return fmt.Errorf("failed to get VM info: %w", err)
			}

			// Update runtime state
			vmRuntime.VMInfo = *vmInfo
			vmRuntime.DriverConfig = driverConfig
			vmRuntime.UpdatedAt = time.Now()
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to update VM state for %s: %w", vmName, err)
		}

		log.Success("VM '%s' started successfully", vmName)
	}

	log.Success("All services started successfully!")
	if !detach {
		log.UserLn("Use 'spitfire vm ps' to view running VMs")
		log.UserLn("Use 'spitfire vm down' to stop all services")
	}

	return nil
}

// runVMDown implements the vm down command
func runVMDown(configFile string, removeVolumes bool) error {
	// Get current working directory for state management
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Initialize state manager
	projectName := filepath.Base(cwd)
	stateManager := state.NewManager(projectName)

	// Load project state
	if !stateManager.ProjectExists() {
		log.UserLn("No VMs found to stop")
		return nil
	}

	projectState, err := stateManager.LoadProjectState()
	if err != nil {
		return fmt.Errorf("failed to load project state: %w", err)
	}

	if len(projectState.VMs) == 0 {
		log.UserLn("No VMs found to stop")
		return nil
	}

	ctx := context.Background()

	// Stop each VM
	for vmName, vmRuntime := range projectState.VMs {
		log.Progress("Stopping VM '%s'", vmName)

		// Create driver instance
		d, err := firecracker.NewFirecrackerDriver(vmRuntime.DriverConfig)
		if err != nil {
			log.Warnf("Failed to create driver for VM %s: %v", vmName, err)
			continue
		}

		// Stop the VM
		if err := d.Stop(ctx); err != nil {
			log.Warnf("Failed to stop VM %s: %v", vmName, err)
			continue
		}

		// Delete the VM
		if err := d.Delete(ctx); err != nil {
			log.Warnf("Failed to delete VM %s: %v", vmName, err)
			continue
		}

		// Update state to mark as stopped
		err = stateManager.UpdateVMState(vmName, func(vmRuntime *state.VMRuntime) error {
			vmRuntime.State = driver.Stopped
			vmRuntime.UpdatedAt = time.Now()
			return nil
		})
		if err != nil {
			log.Warnf("Failed to update VM %s state: %v", vmName, err)
		}

		log.Success("VM '%s' stopped and removed", vmName)
	}

	// TODO: Handle volume removal if removeVolumes is true
	if removeVolumes {
		log.UserLn("Volume removal not yet implemented")
	}

	log.Success("All VMs stopped successfully!")
	return nil
}