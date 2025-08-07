package integration

import (
	"context"
	"testing"

	"github.com/thi-startup/spitfire/pkg/driver"
	_ "github.com/thi-startup/spitfire/pkg/driver/mock" // Import for side effects (registration)
)

func TestDriverIntegration(t *testing.T) {
	// Mock driver should be automatically registered via init()
	// Just verify it's available

	// Test that the driver is now available
	if !driver.Supported("mock") {
		t.Error("Expected mock driver to be supported after registration")
	}

	// Test driver selection
	options := driver.DefaultSelectorOptions()
	pick, alternatives, rejects := driver.Suggest(options)

	if pick.Empty() {
		t.Error("Expected a driver to be selected")
	}

	t.Logf("Selected driver: %s", pick.Name)
	t.Logf("Alternatives: %d", len(alternatives))
	t.Logf("Rejects: %d", len(rejects))

	// Test creating a driver instance
	config := &driver.Config{
		Name:   "test-vm",
		Memory: 512,
		CPUs:   1,
	}

	driverDef := driver.GetDriver("mock")
	if driverDef.Empty() {
		t.Fatal("Expected to find mock driver")
	}

	d, err := driverDef.Create(config)
	if err != nil {
		t.Fatalf("Failed to create driver instance: %v", err)
	}

	if d.DriverName() != "mock" {
		t.Errorf("Expected driver name 'mock', got '%s'", d.DriverName())
	}

	// Test VM lifecycle
	ctx := context.Background()

	// Initially should be in None state
	state, err := d.GetState(ctx)
	if err != nil {
		t.Fatalf("Failed to get initial state: %v", err)
	}
	if state != driver.None {
		t.Errorf("Expected initial state None, got %s", state)
	}

	// Create VM
	err = d.Create(ctx)
	if err != nil {
		t.Fatalf("Failed to create VM: %v", err)
	}

	// Should now be stopped
	state, err = d.GetState(ctx)
	if err != nil {
		t.Fatalf("Failed to get state after create: %v", err)
	}
	if state != driver.Stopped {
		t.Errorf("Expected state Stopped after create, got %s", state)
	}

	// Start VM
	err = d.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start VM: %v", err)
	}

	// Should now be running
	state, err = d.GetState(ctx)
	if err != nil {
		t.Fatalf("Failed to get state after start: %v", err)
	}
	if state != driver.Running {
		t.Errorf("Expected state Running after start, got %s", state)
	}

	// Test getting IP when running
	ip, err := d.GetIP(ctx)
	if err != nil {
		t.Fatalf("Failed to get IP: %v", err)
	}
	if ip == nil {
		t.Error("Expected non-nil IP when VM is running")
	}

	// Test SSH command
	output, err := d.RunSSH(ctx, "echo hello")
	if err != nil {
		t.Fatalf("Failed to run SSH command: %v", err)
	}
	if output == "" {
		t.Error("Expected non-empty output from SSH command")
	}

	// Test VM info
	info, err := d.GetInfo(ctx)
	if err != nil {
		t.Fatalf("Failed to get VM info: %v", err)
	}
	if info.Name != "test-vm" {
		t.Errorf("Expected VM name 'test-vm', got '%s'", info.Name)
	}
	if info.Driver != "mock" {
		t.Errorf("Expected driver 'mock', got '%s'", info.Driver)
	}

	// Stop VM
	err = d.Stop(ctx)
	if err != nil {
		t.Fatalf("Failed to stop VM: %v", err)
	}

	// Should now be stopped
	state, err = d.GetState(ctx)
	if err != nil {
		t.Fatalf("Failed to get state after stop: %v", err)
	}
	if state != driver.Stopped {
		t.Errorf("Expected state Stopped after stop, got %s", state)
	}

	// Delete VM
	err = d.Delete(ctx)
	if err != nil {
		t.Fatalf("Failed to delete VM: %v", err)
	}

	// Should now be none
	state, err = d.GetState(ctx)
	if err != nil {
		t.Fatalf("Failed to get state after delete: %v", err)
	}
	if state != driver.None {
		t.Errorf("Expected state None after delete, got %s", state)
	}

	// Test supported features
	features := d.SupportedFeatures()
	if len(features) == 0 {
		t.Error("Expected mock driver to support some features")
	}

	// Test requires root
	if d.RequiresRoot() {
		t.Error("Expected mock driver to not require root")
	}
}
