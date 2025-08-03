# Spitfire Driver API Reference

## Overview

This document provides a comprehensive reference for the Spitfire Driver API. All drivers must implement the `Driver` interface to integrate with the Spitfire ecosystem.

## Core Interface

### Driver Interface

```go
type Driver interface {
    // Lifecycle management
    Create(ctx context.Context) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Delete(ctx context.Context) error

    // State management
    GetState(ctx context.Context) (VMState, error)
    GetIP(ctx context.Context) (net.IP, error)
    GetInfo(ctx context.Context) (*VMInfo, error)

    // Operations
    RunSSH(ctx context.Context, command string) (string, error)
    CopyFile(ctx context.Context, src, dest string) error

    // Network operations
    CreateNetwork(ctx context.Context, network *NetworkConfig) error
    DeleteNetwork(ctx context.Context, networkName string) error

    // Volume operations
    CreateVolume(ctx context.Context, volume *VolumeConfig) error
    DeleteVolume(ctx context.Context, volumeName string) error
    AttachVolume(ctx context.Context, volumeName, mountPath string) error

    // Metadata
    DriverName() string
    RequiresRoot() bool
    SupportedFeatures() []Feature
}
```

## Method Documentation

### Lifecycle Management

#### Create(ctx context.Context) error
Creates a new virtual machine instance but does not start it.

**Parameters:**
- `ctx`: Context for cancellation and timeouts

**Returns:**
- `error`: nil on success, error on failure

**Behavior:**
- Creates VM with configuration provided during driver initialization
- Allocates resources (disk, network interfaces, etc.)
- VM should be in `Stopped` state after successful creation
- Should be idempotent (safe to call multiple times)

**Example:**
```go
err := driver.Create(ctx)
if err != nil {
    return fmt.Errorf("failed to create VM: %w", err)
}
```

#### Start(ctx context.Context) error
Starts a created virtual machine.

**Parameters:**
- `ctx`: Context for cancellation and timeouts

**Returns:**
- `error`: nil on success, error on failure

**Behavior:**
- VM must be in `Stopped` state before calling
- Transitions VM to `Running` state
- May go through intermediate `Starting` state
- Should handle VM already running gracefully

#### Stop(ctx context.Context) error
Stops a running virtual machine gracefully.

**Parameters:**
- `ctx`: Context for cancellation and timeouts

**Returns:**
- `error`: nil on success, error on failure

**Behavior:**
- Attempts graceful shutdown first
- May force shutdown if graceful fails
- VM transitions to `Stopped` state
- Should handle VM already stopped gracefully

#### Delete(ctx context.Context) error
Permanently removes the virtual machine and its resources.

**Parameters:**
- `ctx`: Context for cancellation and timeouts

**Returns:**
- `error`: nil on success, error on failure

**Behavior:**
- Stops VM if running
- Removes all VM resources (disk, config, etc.)
- VM transitions to `None` state
- Should be idempotent

### State Management

#### GetState(ctx context.Context) (VMState, error)
Returns the current state of the virtual machine.

**Parameters:**
- `ctx`: Context for cancellation and timeouts

**Returns:**
- `VMState`: Current VM state
- `error`: nil on success, error on failure

**VM States:**
```go
const (
    None VMState = iota    // VM does not exist
    Starting               // VM is starting up
    Running               // VM is running
    Paused                // VM is paused
    Stopping              // VM is shutting down
    Stopped               // VM exists but is stopped
    Error                 // VM is in an error state
)
```

#### GetIP(ctx context.Context) (net.IP, error)
Returns the IP address of the running virtual machine.

**Parameters:**
- `ctx`: Context for cancellation and timeouts

**Returns:**
- `net.IP`: VM's IP address
- `error`: nil on success, error on failure

**Behavior:**
- Should return error if VM is not running
- May block briefly waiting for IP assignment
- Return nil IP if no network interface is available

#### GetInfo(ctx context.Context) (*VMInfo, error)
Returns comprehensive information about the virtual machine.

**Parameters:**
- `ctx`: Context for cancellation and timeouts

**Returns:**
- `*VMInfo`: VM information structure
- `error`: nil on success, error on failure

**VMInfo Structure:**
```go
type VMInfo struct {
    ID       string            // Unique VM identifier
    Name     string            // VM name
    State    VMState           // Current state
    IP       net.IP            // IP address (if available)
    Created  time.Time         // Creation timestamp
    Driver   string            // Driver name
    Metadata map[string]string // Driver-specific metadata
}
```

### Operations

#### RunSSH(ctx context.Context, command string) (string, error)
Executes a command on the virtual machine via SSH.

**Parameters:**
- `ctx`: Context for cancellation and timeouts
- `command`: Command to execute

**Returns:**
- `string`: Command output (stdout)
- `error`: nil on success, error on failure

**Behavior:**
- VM must be running and accessible via SSH
- Should return stdout as string
- stderr should be included in error if command fails
- Should respect context timeout

#### CopyFile(ctx context.Context, src, dest string) error
Copies a file between host and virtual machine.

**Parameters:**
- `ctx`: Context for cancellation and timeouts
- `src`: Source file path
- `dest`: Destination file path

**Returns:**
- `error`: nil on success, error on failure

**Behavior:**
- Supports host-to-VM and VM-to-host copying
- Should create destination directories if needed
- Should preserve file permissions when possible

### Network Operations

#### CreateNetwork(ctx context.Context, network *NetworkConfig) error
Creates a network that VMs can connect to.

**Parameters:**
- `ctx`: Context for cancellation and timeouts
- `network`: Network configuration

**NetworkConfig Structure:**
```go
type NetworkConfig struct {
    Name    string            // Network name
    Driver  string            // Network driver (bridge, host, etc.)
    Subnet  string            // CIDR subnet (e.g., "192.168.1.0/24")
    Gateway string            // Gateway IP address
    Options map[string]string // Driver-specific options
}
```

#### DeleteNetwork(ctx context.Context, networkName string) error
Removes a network.

**Parameters:**
- `ctx`: Context for cancellation and timeouts
- `networkName`: Name of network to delete

**Returns:**
- `error`: nil on success, error on failure

### Volume Operations

#### CreateVolume(ctx context.Context, volume *VolumeConfig) error
Creates a persistent volume for data storage.

**Parameters:**
- `ctx`: Context for cancellation and timeouts
- `volume`: Volume configuration

**VolumeConfig Structure:**
```go
type VolumeConfig struct {
    Name    string            // Volume name
    Driver  string            // Volume driver (local, etc.)
    Size    string            // Volume size (e.g., "10GB")
    Options map[string]string // Driver-specific options
}
```

#### DeleteVolume(ctx context.Context, volumeName string) error
Removes a volume and its data.

**Parameters:**
- `ctx`: Context for cancellation and timeouts
- `volumeName`: Name of volume to delete

#### AttachVolume(ctx context.Context, volumeName, mountPath string) error
Attaches a volume to the virtual machine.

**Parameters:**
- `ctx`: Context for cancellation and timeouts
- `volumeName`: Name of volume to attach
- `mountPath`: Path where volume should be mounted in VM

### Metadata

#### DriverName() string
Returns the name of the driver.

**Returns:**
- `string`: Driver name (e.g., "firecracker", "qemu")

#### RequiresRoot() bool
Indicates whether the driver requires root privileges.

**Returns:**
- `bool`: true if root required, false otherwise

#### SupportedFeatures() []Feature
Returns list of features supported by this driver.

**Returns:**
- `[]Feature`: List of supported features

**Feature Constants:**
```go
const (
    FeatureNetworking    Feature = "networking"
    FeatureVolumes       Feature = "volumes"
    FeatureSnapshots     Feature = "snapshots"
    FeaturePortForward   Feature = "port_forward"
    FeatureFileSync      Feature = "file_sync"
    FeatureGPU           Feature = "gpu"
    FeatureNested        Feature = "nested_virt"
)
```

## Driver Configuration

### Config Structure
Drivers receive configuration through the `Config` structure:

```go
type Config struct {
    // Basic VM configuration
    Name     string // VM name
    Image    string // Container image or VM image path
    Rootfs   string // Path to rootfs (alternative to Image)
    Memory   int64  // Memory in MB
    CPUs     int    // Number of CPUs
    DiskSize int64  // Disk size in MB

    // Networking
    Networks []string // Networks to connect to
    Ports    []string // Port mappings (host:guest)

    // Storage
    Volumes []VolumeMount // Volume mounts

    // Environment
    Env        map[string]string // Environment variables
    WorkingDir string           // Working directory
    User       string           // User to run as

    // Driver-specific options
    DriverOpts map[string]interface{} // Driver-specific configuration

    // Paths
    StorePath string // Path for driver storage
    SSHKey    string // SSH key for access
}

type VolumeMount struct {
    Type     string // "volume" or "bind"
    Source   string // Source path/volume name
    Target   string // Target path in VM
    ReadOnly bool   // Read-only mount
}
```

## Error Handling

### Error Types
Drivers should return meaningful errors:

```go
// VM does not exist
var ErrVMNotFound = errors.New("virtual machine not found")

// VM is in wrong state for operation
var ErrInvalidState = errors.New("invalid VM state for operation")

// Feature not supported by driver
var ErrNotSupported = errors.New("operation not supported by driver")

// Permission denied
var ErrPermissionDenied = errors.New("permission denied")
```

### Error Wrapping
Use error wrapping for context:

```go
if err != nil {
    return fmt.Errorf("failed to start firecracker VM: %w", err)
}
```

## Best Practices

### Context Handling
- Always respect context cancellation
- Use reasonable timeouts for operations
- Propagate context to underlying operations

### State Management
- Ensure state transitions are atomic
- Handle concurrent access safely
- Provide accurate state reporting

### Resource Cleanup
- Clean up resources on errors
- Implement proper shutdown procedures
- Handle partial failures gracefully

### Logging
- Use structured logging with context
- Log important state transitions
- Include relevant metadata in logs

### Testing
- Implement comprehensive unit tests
- Test error conditions and edge cases
- Verify state transitions and cleanup

## Example Implementation

See the [mock driver implementation](../pkg/driver/mock/mock.go) for a complete example of the Driver interface.

## Registry Integration

### Driver Definition
```go
type DriverDef struct {
    Name        string        // Driver name
    Aliases     []string      // Alternative names
    Create      DriverFactory // Factory function
    Status      StatusChecker // Health check function
    Priority    Priority      // Selection priority
    Default     bool          // Available by default
    Description string        // Human-readable description
}
```

### Registration
```go
func init() {
    err := driver.Register(driver.DriverDef{
        Name:        "mydriver",
        Aliases:     []string{"my", "custom"},
        Create:      NewMyDriver,
        Status:      checkMyDriverStatus,
        Priority:    driver.Default,
        Default:     true,
        Description: "My custom driver implementation",
    })
    if err != nil {
        panic(fmt.Sprintf("failed to register driver: %v", err))
    }
}
```

## See Also

- [Driver Development Guide](./spitfire-driver-development.md)
- [Driver System Overview](./spitfire-driver-system.md)
- [Mock Driver Example](../pkg/driver/mock/mock.go)