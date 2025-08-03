package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/thi-startup/spitfire/pkg/config"
	"github.com/thi-startup/spitfire/pkg/driver"
	"github.com/thi-startup/spitfire/pkg/state"
)

func main() {
	// Create a temporary demo directory
	tempDir, err := os.MkdirTemp("", "spitfire-state-demo")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir)

	// Set spitfire home to our temp directory
	os.Setenv("SPITFIRE_HOME", tempDir)

	fmt.Println("🔥 Spitfire State Management Demo")
	fmt.Println("================================")
	fmt.Printf("Demo directory: %s\n\n", tempDir)

	// Create a sample configuration
	demoConfig := &config.Config{
		Version: "1",
		Driver:  "firecracker",
		Globals: &config.GlobalConfig{
			Resources: &config.Resources{
				VCPU:   1,
				Memory: "512MB",
			},
		},
		VMs: map[string]*config.VM{
			"web": {
				Image: "nginx:alpine",
				Resources: &config.Resources{
					Memory: "256MB",
				},
			},
			"api": {
				Image: "myapp:latest",
				Resources: &config.Resources{
					VCPU:   2,
					Memory: "1GB",
				},
			},
		},
	}

	// Create a fake config file
	configPath := filepath.Join(tempDir, "demo-config.yml")
	configData := []byte(`version: "1"
driver: firecracker
vms:
  web:
    image: nginx:alpine
  api:
    image: myapp:latest`)
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		panic(err)
	}

	// Initialize project state
	projectName := "demo-project"
	manager := state.NewManager(projectName)

	fmt.Println("1. Initializing project...")
	projectState, err := manager.InitializeProject(configPath, demoConfig)
	if err != nil {
		panic(err)
	}

	fmt.Printf("   ✓ Project '%s' created with %d VMs\n", projectState.Name, len(projectState.VMs))
	fmt.Printf("   ✓ Status: %s\n", projectState.Status)

	// Show directory structure
	fmt.Println("\n2. Directory structure created:")
	showDirectoryTree(state.ProjectPath(projectName), "   ")

	// Simulate starting a VM
	fmt.Println("\n3. Starting 'web' VM...")
	err = manager.UpdateVMState("web", func(vm *state.VMRuntime) error {
		vm.State = driver.Running
		vm.PID = 12345
		return nil
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("   ✓ VM 'web' started (PID: 12345)")

	// Record an event
	fmt.Println("\n4. Recording event...")
	event := &state.Event{
		ID:       "demo-event-1",
		Type:     state.EventTypeVM,
		Resource: "web",
		Action:   "start",
		Status:   state.EventStatusSuccess,
		Message:  "VM started successfully in demo",
	}
	if err := manager.RecordEvent(event); err != nil {
		panic(err)
	}
	fmt.Println("   ✓ Event recorded")

	// Show current state
	fmt.Println("\n5. Current project state:")
	currentState, err := manager.LoadProjectState()
	if err != nil {
		panic(err)
	}

	for vmName, vm := range currentState.VMs {
		fmt.Printf("   VM '%s': %s", vmName, vm.State)
		if vm.PID > 0 {
			fmt.Printf(" (PID: %d)", vm.PID)
		}
		fmt.Println()
	}

	// Show events
	fmt.Println("\n6. Recent events:")
	events, err := manager.GetProjectEvents(5)
	if err != nil {
		panic(err)
	}

	for _, event := range events {
		fmt.Printf("   %s: %s %s -> %s\n",
			event.Type, event.Resource, event.Action, event.Status)
	}

	// List all projects
	fmt.Println("\n7. All projects:")
	projects, err := state.ListProjects()
	if err != nil {
		panic(err)
	}

	for _, project := range projects {
		fmt.Printf("   - %s\n", project)
	}

	fmt.Println("\n✨ Demo completed successfully!")
	fmt.Printf("   State persisted in: %s\n", state.SpitfirePath())
}

func showDirectoryTree(root string, prefix string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}

	for i, entry := range entries {
		isLast := i == len(entries)-1
		connector := "├── "
		if isLast {
			connector = "└── "
		}

		fmt.Printf("%s%s%s", prefix, connector, entry.Name())

		if entry.IsDir() {
			fmt.Println("/")
			nextPrefix := prefix + "│   "
			if isLast {
				nextPrefix = prefix + "    "
			}
			showDirectoryTree(filepath.Join(root, entry.Name()), nextPrefix)
		} else {
			fmt.Println()
		}
	}
}
