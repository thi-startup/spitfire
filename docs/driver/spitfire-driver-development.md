# Spitfire Driver Development Guide

## Overview

This guide walks you through creating a new driver for Spitfire. By following this guide, you'll learn how to implement the driver interface, integrate with the registry system, and follow best practices for driver development.

## Getting Started

### Prerequisites

- Go 1.19 or later
- Understanding of the target virtualization technology
- Familiarity with the [Driver API Reference](./spitfire-driver-api.md)

### Project Structure

```
pkg/driver/
├── types.go           # Core interfaces and types
├── registry.go        # Driver registry
├── selector.go        # Driver selection logic
├── mydriver/          # Your driver implementation
│   ├── mydriver.go    # Main driver implementation
│   ├── mydriver_test.go # Unit tests
│   └── README.md      # Driver-specific documentation
└── integration/       # Integration tests
```

## Step 1: Create Driver Package

Create a new package for your driver:

```bash
mkdir pkg/driver/mydriver
cd pkg/driver/mydriver
```

## Step 2: Implement Driver Interface

Create the main driver file:

```go
// pkg/driver/mydriver/mydriver.go
package mydriver

import (
    "context"
    "fmt"
    "net"
    "time"

    "github.com/thi-startup/spitfire/pkg/driver"
)

// Driver implements the spitfire driver interface for MyTech
type Driver struct {
    config *driver.Config
    // Add fields specific to your virtualization technology
    // e.g., client connections, process handles, etc.
}

// NewDriver creates a new MyTech driver instance
func NewDriver(config *driver.Config) (driver.Driver, error) {
    // Validate configuration
    if config.Name == "" {
        return nil, fmt.Errorf("VM name is required")
    }

    d := &Driver{
        config: config,
    }

    // Initialize driver-specific components
    if err := d.initialize(); err != nil {
        return nil, fmt.Errorf("failed to initialize driver: %w", err)
    }

    return d, nil
}

func (d *Driver) initialize() error {
    // Initialize connections, validate dependencies, etc.
    return nil
}
```

### Implement Core Methods

```go
// Create implements driver.Driver
func (d *Driver) Create(ctx context.Context) error {
    // Check if VM already exists
    exists, err := d.vmExists()
    if err != nil {
        return fmt.Errorf("failed to check VM existence: %w", err)
    }
    if exists {
        return nil // Idempotent operation
    }

    // Create VM using your virtualization technology
    if err := d.createVM(ctx); err != nil {
        return fmt.Errorf("failed to create VM: %w", err)
    }

    return nil
}

// Start implements driver.Driver  
func (d *Driver) Start(ctx context.Context) error {
    state, err := d.GetState(ctx)
    if err != nil {
        return fmt.Errorf("failed to get VM state: %w", err)
    }

    switch state {
    case driver.Running:
        return nil // Already running
    case driver.None:
        return fmt.Errorf("VM does not exist, create it first")
    case driver.Stopped:
        // Start the VM
        return d.startVM(ctx)
    default:
        return fmt.Errorf("cannot start VM in state %s", state)
    }
}

// Stop implements driver.Driver
func (d *Driver) Stop(ctx context.Context) error {
    state, err := d.GetState(ctx)
    if err != nil {
        return err
    }

    if state == driver.Stopped || state == driver.None {
        return nil // Already stopped or doesn't exist
    }

    // Attempt graceful shutdown first
    if err := d.gracefulShutdown(ctx); err != nil {
        // Fall back to force shutdown
        return d.forceShutdown(ctx)
    }

    return nil
}

// Delete implements driver.Driver
func (d *Driver) Delete(ctx context.Context) error {
    // Stop VM if running
    if err := d.Stop(ctx); err != nil {
        return fmt.Errorf("failed to stop VM before deletion: %w", err)
    }

    // Remove VM and all associated resources
    return d.deleteVM(ctx)
}
```

### Implement State Management

```go
// GetState implements driver.Driver
func (d *Driver) GetState(ctx context.Context) (driver.VMState, error) {
    // Query your virtualization technology for VM state
    // Map the technology-specific state to driver.VMState
    
    exists, err := d.vmExists()
    if err != nil {
        return driver.Error, err
    }
    if !exists {
        return driver.None, nil
    }

    running, err := d.isRunning()
    if err != nil {
        return driver.Error, err
    }
    if running {
        return driver.Running, nil
    }

    return driver.Stopped, nil
}

// GetIP implements driver.Driver
func (d *Driver) GetIP(ctx context.Context) (net.IP, error) {
    state, err := d.GetState(ctx)
    if err != nil {
        return nil, err
    }
    if state != driver.Running {
        return nil, fmt.Errorf("VM is not running")
    }

    // Get IP address from your virtualization technology
    return d.getVMIP()
}

// GetInfo implements driver.Driver
func (d *Driver) GetInfo(ctx context.Context) (*driver.VMInfo, error) {
    state, err := d.GetState(ctx)
    if err != nil {
        return nil, err
    }

    ip, _ := d.GetIP(ctx) // Ignore error if VM not running

    return &driver.VMInfo{
        ID:      d.config.Name,
        Name:    d.config.Name,
        State:   state,
        IP:      ip,
        Created: time.Now(), // Get actual creation time
        Driver:  d.DriverName(),
        Metadata: map[string]string{
            "memory": fmt.Sprintf("%dMB", d.config.Memory),
            "cpus":   fmt.Sprintf("%d", d.config.CPUs),
            // Add driver-specific metadata
        },
    }, nil
}
```

### Implement Operations

```go
// RunSSH implements driver.Driver
func (d *Driver) RunSSH(ctx context.Context, command string) (string, error) {
    ip, err := d.GetIP(ctx)
    if err != nil {
        return "", fmt.Errorf("failed to get VM IP: %w", err)
    }

    // Implement SSH execution
    // You can use existing SSH libraries or implement your own
    return d.executeSSH(ip, command)
}

// CopyFile implements driver.Driver
func (d *Driver) CopyFile(ctx context.Context, src, dest string) error {
    ip, err := d.GetIP(ctx)
    if err != nil {
        return fmt.Errorf("failed to get VM IP: %w", err)
    }

    // Implement file copying (SCP, SFTP, etc.)
    return d.copyViaSCP(ip, src, dest)
}
```

### Implement Network Operations

```go
// CreateNetwork implements driver.Driver
func (d *Driver) CreateNetwork(ctx context.Context, network *driver.NetworkConfig) error {
    // Create network using your virtualization technology
    // This might involve creating bridges, virtual switches, etc.
    
    if !d.supportsNetworking() {
        return driver.ErrNotSupported
    }

    return d.createNetwork(network)
}

// DeleteNetwork implements driver.Driver
func (d *Driver) DeleteNetwork(ctx context.Context, networkName string) error {
    if !d.supportsNetworking() {
        return driver.ErrNotSupported
    }

    return d.deleteNetwork(networkName)
}
```

### Implement Volume Operations

```go
// CreateVolume implements driver.Driver
func (d *Driver) CreateVolume(ctx context.Context, volume *driver.VolumeConfig) error {
    if !d.supportsVolumes() {
        return driver.ErrNotSupported
    }

    return d.createVolume(volume)
}

// DeleteVolume implements driver.Driver
func (d *Driver) DeleteVolume(ctx context.Context, volumeName string) error {
    if !d.supportsVolumes() {
        return driver.ErrNotSupported
    }

    return d.deleteVolume(volumeName)
}

// AttachVolume implements driver.Driver
func (d *Driver) AttachVolume(ctx context.Context, volumeName, mountPath string) error {
    if !d.supportsVolumes() {
        return driver.ErrNotSupported
    }

    return d.attachVolume(volumeName, mountPath)
}
```

### Implement Metadata Methods

```go
// DriverName implements driver.Driver
func (d *Driver) DriverName() string {
    return "mydriver"
}

// RequiresRoot implements driver.Driver
func (d *Driver) RequiresRoot() bool {
    // Return true if your driver needs root privileges
    return false
}

// SupportedFeatures implements driver.Driver
func (d *Driver) SupportedFeatures() []driver.Feature {
    features := []driver.Feature{}
    
    if d.supportsNetworking() {
        features = append(features, driver.FeatureNetworking)
    }
    
    if d.supportsVolumes() {
        features = append(features, driver.FeatureVolumes)
    }
    
    // Add other supported features
    features = append(features, driver.FeatureFileSync)
    
    return features
}
```

## Step 3: Driver Helper Methods

Implement driver-specific helper methods:

```go
// Driver-specific helper methods
func (d *Driver) vmExists() (bool, error) {
    // Check if VM exists in your virtualization technology
    return false, nil
}

func (d *Driver) isRunning() (bool, error) {
    // Check if VM is currently running
    return false, nil
}

func (d *Driver) createVM(ctx context.Context) error {
    // Create VM using your technology's API/CLI
    return nil
}

func (d *Driver) startVM(ctx context.Context) error {
    // Start the VM
    return nil
}

func (d *Driver) gracefulShutdown(ctx context.Context) error {
    // Attempt graceful shutdown
    return nil
}

func (d *Driver) forceShutdown(ctx context.Context) error {
    // Force shutdown if graceful fails
    return nil
}

func (d *Driver) deleteVM(ctx context.Context) error {
    // Delete VM and clean up resources
    return nil
}

func (d *Driver) getVMIP() (net.IP, error) {
    // Get VM's IP address
    return net.ParseIP("192.168.1.100"), nil
}

func (d *Driver) executeSSH(ip net.IP, command string) (string, error) {
    // Execute SSH command
    return fmt.Sprintf("output of: %s", command), nil
}

func (d *Driver) copyViaSCP(ip net.IP, src, dest string) error {
    // Copy file via SCP
    return nil
}

func (d *Driver) supportsNetworking() bool {
    // Return true if driver supports networking
    return true
}

func (d *Driver) supportsVolumes() bool {
    // Return true if driver supports volumes
    return true
}

func (d *Driver) createNetwork(network *driver.NetworkConfig) error {
    // Create network
    return nil
}

func (d *Driver) deleteNetwork(networkName string) error {
    // Delete network
    return nil
}

func (d *Driver) createVolume(volume *driver.VolumeConfig) error {
    // Create volume
    return nil
}

func (d *Driver) deleteVolume(volumeName string) error {
    // Delete volume
    return nil
}

func (d *Driver) attachVolume(volumeName, mountPath string) error {
    // Attach volume to VM
    return nil
}
```

## Step 4: Status Checker

Implement a status checker for the registry:

```go
// status checks if the driver is available and healthy
func status() driver.State {
    // Check if required binaries are installed
    if !isBinaryAvailable("mytool") {
        return driver.State{
            Installed: false,
            Healthy:   false,
            Error:     fmt.Errorf("mytool binary not found"),
            Fix:       "Install mytool from https://example.com/install",
            Doc:       "https://docs.example.com/mytool",
        }
    }

    // Check if service is running (if applicable)
    if !isServiceRunning("mytool-daemon") {
        return driver.State{
            Installed: true,
            Healthy:   false,
            Running:   false,
            Error:     fmt.Errorf("mytool daemon is not running"),
            Fix:       "Start the daemon with: sudo systemctl start mytool-daemon",
            Doc:       "https://docs.example.com/daemon",
        }
    }

    // Check permissions
    if !hasRequiredPermissions() {
        return driver.State{
            Installed: true,
            Healthy:   false,
            Running:   true,
            Error:     fmt.Errorf("insufficient permissions"),
            Fix:       "Add user to mytool group: sudo usermod -aG mytool $USER",
            Doc:       "https://docs.example.com/permissions",
        }
    }

    // All checks passed
    return driver.State{
        Installed: true,
        Healthy:   true,
        Running:   true,
        Version:   getVersion(),
    }
}

func isBinaryAvailable(name string) bool {
    _, err := exec.LookPath(name)
    return err == nil
}

func isServiceRunning(name string) bool {
    // Check if service is running
    // Implementation depends on your system
    return true
}

func hasRequiredPermissions() bool {
    // Check if user has required permissions
    return true
}

func getVersion() string {
    // Get driver/tool version
    return "1.0.0"
}
```

## Step 5: Register Driver

Create registration function:

```go
// Register registers the mydriver with the global registry
func Register() error {
    return driver.Register(driver.DriverDef{
        Name:        "mydriver",
        Aliases:     []string{"my", "custom"},
        Create:      NewDriver,
        Status:      status,
        Priority:    driver.Default,
        Default:     true,
        Description: "Driver for MyTech virtualization platform",
    })
}

// Auto-register the driver
func init() {
    if err := Register(); err != nil {
        panic(fmt.Sprintf("failed to register mydriver: %v", err))
    }
}
```

## Step 6: Write Tests

Create comprehensive tests:

```go
// pkg/driver/mydriver/mydriver_test.go
package mydriver

import (
    "context"
    "testing"

    "github.com/thi-startup/spitfire/pkg/driver"
)

func TestDriver(t *testing.T) {
    config := &driver.Config{
        Name:   "test-vm",
        Memory: 512,
        CPUs:   1,
    }

    d, err := NewDriver(config)
    if err != nil {
        t.Fatalf("Failed to create driver: %v", err)
    }

    ctx := context.Background()

    // Test initial state
    state, err := d.GetState(ctx)
    if err != nil {
        t.Fatalf("Failed to get initial state: %v", err)
    }
    if state != driver.None {
        t.Errorf("Expected initial state None, got %s", state)
    }

    // Test VM lifecycle
    if err := d.Create(ctx); err != nil {
        t.Fatalf("Failed to create VM: %v", err)
    }

    if err := d.Start(ctx); err != nil {
        t.Fatalf("Failed to start VM: %v", err)
    }

    state, err = d.GetState(ctx)
    if err != nil {
        t.Fatalf("Failed to get state after start: %v", err)
    }
    if state != driver.Running {
        t.Errorf("Expected state Running, got %s", state)
    }

    if err := d.Stop(ctx); err != nil {
        t.Fatalf("Failed to stop VM: %v", err)
    }

    if err := d.Delete(ctx); err != nil {
        t.Fatalf("Failed to delete VM: %v", err)
    }
}

func TestDriverMetadata(t *testing.T) {
    config := &driver.Config{Name: "test"}
    d, err := NewDriver(config)
    if err != nil {
        t.Fatalf("Failed to create driver: %v", err)
    }

    if d.DriverName() != "mydriver" {
        t.Errorf("Expected driver name 'mydriver', got '%s'", d.DriverName())
    }

    features := d.SupportedFeatures()
    if len(features) == 0 {
        t.Error("Expected some supported features")
    }
}
```

## Step 7: Documentation

Create driver-specific documentation:

```markdown
# MyDriver Documentation

## Overview
MyDriver provides support for MyTech virtualization platform.

## Installation
1. Install MyTech from https://example.com
2. Start the daemon: `sudo systemctl start mytool-daemon`
3. Add user to group: `sudo usermod -aG mytool $USER`

## Configuration
```yaml
driver: mydriver
driver_opts:
  mydriver:
    option1: value1
    option2: value2
```

## Features
- VM lifecycle management
- Networking support
- Volume management
- SSH access

## Limitations
- Requires Linux host
- No nested virtualization support
```

## Best Practices

### Error Handling
- Use meaningful error messages
- Wrap errors with context
- Implement proper cleanup on failures

### Resource Management
- Clean up resources on deletion
- Handle partial failures gracefully
- Implement proper shutdown procedures

### Performance
- Use connection pooling where appropriate
- Implement caching for expensive operations
- Respect context timeouts

### Security
- Validate all inputs
- Use secure communication channels
- Follow principle of least privilege

### Testing
- Test all error conditions
- Verify resource cleanup
- Test concurrent operations

## Integration with Spitfire

### Configuration Options
Support standard spitfire configuration and add driver-specific options:

```go
func (d *Driver) parseDriverOpts() error {
    opts := d.config.DriverOpts
    if opts == nil {
        return nil
    }

    if val, ok := opts["custom_option"]; ok {
        d.customOption = val.(string)
    }

    return nil
}
```

### Logging
Use structured logging:

```go
import "k8s.io/klog/v2"

func (d *Driver) Create(ctx context.Context) error {
    klog.Infof("Creating VM %s with mydriver", d.config.Name)
    
    if err := d.createVM(ctx); err != nil {
        klog.Errorf("Failed to create VM %s: %v", d.config.Name, err)
        return err
    }
    
    klog.Infof("Successfully created VM %s", d.config.Name)
    return nil
}
```

## Debugging

### Enable Debug Logging
```bash
export SPITFIRE_LOG_LEVEL=debug
spitfire up --driver mydriver
```

### Common Issues
1. **Permission Denied**: Check user groups and permissions
2. **Binary Not Found**: Ensure tools are installed and in PATH
3. **Service Not Running**: Check daemon status
4. **Network Issues**: Verify network configuration

## Contributing

1. Follow the [Driver API Reference](./spitfire-driver-api.md)
2. Write comprehensive tests
3. Update documentation
4. Submit pull request

## See Also

- [Driver API Reference](./spitfire-driver-api.md)
- [Driver System Overview](./spitfire-driver-system.md)
- [Mock Driver Example](../pkg/driver/mock/mock.go)