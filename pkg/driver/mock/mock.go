package mock

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/thi-startup/spitfire/pkg/driver"
)

// Driver implements driver.Driver for testing
type Driver struct {
	config *driver.Config
	state  driver.VMState
	ip     net.IP
}

// NewDriver creates a new mock driver
func NewDriver(config *driver.Config) (driver.Driver, error) {
	return &Driver{
		config: config,
		state:  driver.None,
		ip:     net.ParseIP("192.168.1.100"),
	}, nil
}

// Create simulates VM creation
func (d *Driver) Create(ctx context.Context) error {
	if d.state != driver.None {
		return fmt.Errorf("VM already exists")
	}
	d.state = driver.Stopped
	return nil
}

// Start simulates VM startup
func (d *Driver) Start(ctx context.Context) error {
	if d.state == driver.None {
		return fmt.Errorf("VM does not exist")
	}
	if d.state == driver.Running {
		return nil
	}
	d.state = driver.Starting
	// Simulate startup time
	time.Sleep(100 * time.Millisecond)
	d.state = driver.Running
	return nil
}

// Stop simulates VM shutdown
func (d *Driver) Stop(ctx context.Context) error {
	if d.state == driver.None {
		return fmt.Errorf("VM does not exist")
	}
	if d.state == driver.Stopped {
		return nil
	}
	d.state = driver.Stopping
	// Simulate shutdown time
	time.Sleep(50 * time.Millisecond)
	d.state = driver.Stopped
	return nil
}

// Delete simulates VM deletion
func (d *Driver) Delete(ctx context.Context) error {
	d.state = driver.None
	return nil
}

// GetState returns the current VM state
func (d *Driver) GetState(ctx context.Context) (driver.VMState, error) {
	return d.state, nil
}

// GetIP returns the VM's IP address
func (d *Driver) GetIP(ctx context.Context) (net.IP, error) {
	if d.state != driver.Running {
		return nil, fmt.Errorf("VM is not running")
	}
	return d.ip, nil
}

// GetInfo returns VM information
func (d *Driver) GetInfo(ctx context.Context) (*driver.VMInfo, error) {
	return &driver.VMInfo{
		ID:      d.config.Name,
		Name:    d.config.Name,
		State:   d.state,
		IP:      d.ip,
		Created: time.Now(),
		Driver:  "mock",
		Metadata: map[string]string{
			"memory": fmt.Sprintf("%dMB", d.config.Memory),
			"cpus":   fmt.Sprintf("%d", d.config.CPUs),
		},
	}, nil
}

// RunSSH simulates SSH command execution
func (d *Driver) RunSSH(ctx context.Context, command string) (string, error) {
	if d.state != driver.Running {
		return "", fmt.Errorf("VM is not running")
	}
	return fmt.Sprintf("mock output for: %s", command), nil
}

// CopyFile simulates file copying
func (d *Driver) CopyFile(ctx context.Context, src, dest string) error {
	if d.state != driver.Running {
		return fmt.Errorf("VM is not running")
	}
	return nil
}

// CreateNetwork simulates network creation
func (d *Driver) CreateNetwork(ctx context.Context, network *driver.NetworkConfig) error {
	return nil
}

// DeleteNetwork simulates network deletion
func (d *Driver) DeleteNetwork(ctx context.Context, networkName string) error {
	return nil
}

// CreateVolume simulates volume creation
func (d *Driver) CreateVolume(ctx context.Context, volume *driver.VolumeConfig) error {
	return nil
}

// DeleteVolume simulates volume deletion
func (d *Driver) DeleteVolume(ctx context.Context, volumeName string) error {
	return nil
}

// AttachVolume simulates volume attachment
func (d *Driver) AttachVolume(ctx context.Context, volumeName, mountPath string) error {
	return nil
}

// DriverName returns the driver name
func (d *Driver) DriverName() string {
	return "mock"
}

// RequiresRoot returns whether the driver requires root privileges
func (d *Driver) RequiresRoot() bool {
	return false
}

// SupportedFeatures returns the features supported by this driver
func (d *Driver) SupportedFeatures() []driver.Feature {
	return []driver.Feature{
		driver.FeatureNetworking,
		driver.FeatureVolumes,
		driver.FeaturePortForward,
		driver.FeatureFileSync,
	}
}

// Setup simulates driver setup for testing
func (d *Driver) Setup(ctx context.Context, opts *driver.SetupOptions) (*driver.SetupResult, error) {
	actions := []driver.SetupAction{
		{
			Component:   "mock-component",
			Description: "Mock setup action for testing",
			Command:     "echo 'mock setup'",
			Success:     true,
		},
	}

	if opts.DryRun {
		return &driver.SetupResult{
			Success:   true,
			Actions:   actions,
			NextSteps: []string{"This is a mock driver - no real setup needed"},
		}, nil
	}

	return &driver.SetupResult{
		Success: true,
		Actions: actions,
	}, nil
}

// VerifySetup simulates setup verification for testing
func (d *Driver) VerifySetup(ctx context.Context) (*driver.SetupStatus, error) {
	return &driver.SetupStatus{
		Ready: true,
		Components: []driver.ComponentStatus{
			{
				Name:        "mock-component",
				Ready:       true,
				Description: "Mock component for testing",
				Status:      "Ready",
				Required:    true,
			},
		},
		Summary: "Mock driver is ready for testing",
	}, nil
}

// GetSetupInstructions returns mock setup instructions for testing
func (d *Driver) GetSetupInstructions(ctx context.Context) (*driver.SetupInstructions, error) {
	return &driver.SetupInstructions{
		Overview: "This is a mock driver for testing purposes",
		Prerequisites: []string{
			"No prerequisites needed for mock driver",
		},
		AutomatedSteps: []driver.SetupStep{
			{
				Name:        "mock-setup",
				Description: "Simulated setup step",
				Commands:    []string{"echo 'mock setup complete'"},
				Required:    true,
			},
		},
		Documentation: []driver.DocLink{
			{
				Title:       "Mock Driver Documentation",
				URL:         "https://example.com/mock-driver",
				Description: "Documentation for the mock driver",
			},
		},
	}, nil
}

// Register registers the mock driver
func Register() error {
	return driver.Register(driver.DriverDef{
		Name:        "mock",
		Aliases:     []string{"test"},
		Create:      NewDriver,
		Status:      status,
		Priority:    driver.Default,
		Default:     false,
		Description: "Mock driver for testing",
	})
}

// init automatically registers the mock driver
func init() {
	if err := Register(); err != nil {
		// Don't panic for mock driver - just print error
		fmt.Printf("Warning: failed to register mock driver: %v\n", err)
	}
}

// status returns the mock driver status
func status() driver.State {
	return driver.State{
		Installed: true,
		Healthy:   true,
		Running:   true,
		Version:   "1.0.0-mock",
	}
}
