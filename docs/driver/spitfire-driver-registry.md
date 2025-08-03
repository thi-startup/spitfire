# Spitfire Driver Registry and Selection

## Overview

The Spitfire driver registry is a central system that manages driver discovery, health checking, and selection. It provides a robust foundation for supporting multiple virtualization backends while maintaining a consistent user experience.

## Registry Architecture

### Core Components

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Driver Def    │    │    Registry     │    │    Selector     │
│                 │    │                 │    │                 │
│ • Name          │    │ • Registration  │    │ • Health Check  │
│ • Factory       │ ──▶│ • Storage       │ ──▶│ • Priority Sort │
│ • Status Check  │    │ • Lookup        │    │ • Preference    │
│ • Priority      │    │ • Aliases       │    │ • Fallback      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Driver Definition

Every driver must register with a `DriverDef` structure:

```go
type DriverDef struct {
    Name        string        // Unique driver identifier
    Aliases     []string      // Alternative names (e.g., "kvm" for "kvm2")
    Create      DriverFactory // Function to create driver instances
    Status      StatusChecker // Function to check driver health
    Priority    Priority      // Selection priority level
    Default     bool          // Available for automatic selection
    Description string        // Human-readable description
}
```

### Driver Factory

The factory function creates driver instances:

```go
type DriverFactory func(*Config) (Driver, error)

// Example factory implementation
func NewFirecrackerDriver(config *Config) (Driver, error) {
    return &FirecrackerDriver{
        config: config,
        // ... initialize driver
    }, nil
}
```

### Status Checker

Status checkers report driver health and availability:

```go
type StatusChecker func() State

type State struct {
    Installed        bool   // Driver dependencies are installed
    Healthy          bool   // Driver passes health checks
    Running          bool   // Driver services are running
    NeedsImprovement bool   // Works but could be optimized
    Error            error  // Any error encountered
    Reason           string // Error reason identifier
    Fix              string // Suggested fix for issues
    Doc              string // Documentation URL
    Version          string // Driver/tool version
}
```

## Priority System

Drivers are assigned priorities that determine selection order:

### Priority Levels

| Priority | Value | Description | Use Case |
|----------|-------|-------------|----------|
| HighlyPreferred | 10 | Best choice for platform | Firecracker on Linux |
| Preferred | 8 | Good choice with broad support | QEMU with KVM |
| Default | 6 | Reliable fallback option | VirtualBox |
| Fallback | 4 | Works but not recommended | Basic virtualization |
| Experimental | 2 | Early stage, may have issues | New drivers |
| Deprecated | 1 | Being phased out | Legacy drivers |
| Discouraged | 0 | Has significant limitations | Problematic drivers |
| Unhealthy | -1 | Currently not working | Failed health check |
| Obsolete | -2 | No longer supported | Removed drivers |

### Priority Assignment Examples

```go
// Firecracker driver on Linux
Priority: HighlyPreferred  // Native performance, security-focused

// QEMU driver with KVM
Priority: Preferred        // Good performance, broad compatibility

// VirtualBox driver  
Priority: Default          // Cross-platform but slower

// Docker driver (containers)
Priority: Fallback         // Fast but limited isolation

// New experimental driver
Priority: Experimental     // Not yet production-ready
```

## Registration Process

### Automatic Registration

Drivers typically register themselves in `init()` functions:

```go
package firecracker

import "github.com/thi-startup/spitfire/pkg/driver"

func init() {
    err := driver.Register(driver.DriverDef{
        Name:        "firecracker",
        Aliases:     []string{"fc"},
        Create:      NewFirecrackerDriver,
        Status:      checkFirecrackerStatus,
        Priority:    driver.HighlyPreferred,
        Default:     true,
        Description: "AWS Firecracker microVM driver",
    })
    if err != nil {
        panic(fmt.Sprintf("failed to register firecracker driver: %v", err))
    }
}
```

### Manual Registration

Drivers can also be registered programmatically:

```go
func RegisterCustomDriver() error {
    return driver.Register(driver.DriverDef{
        Name:     "custom",
        Create:   NewCustomDriver,
        Status:   checkCustomStatus,
        Priority: driver.Default,
        Default:  false,
    })
}
```

### Registration Validation

The registry validates registrations:

- **Name uniqueness**: No duplicate driver names
- **Alias conflicts**: Aliases must be unique across all drivers
- **Required functions**: Create and Status functions must be provided
- **Valid priority**: Priority must be a valid enum value

## Driver Selection

### Selection Algorithm

The selector uses a multi-step process:

1. **Filtering**: Remove drivers that don't meet requirements
2. **Health Check**: Verify driver status and availability  
3. **Priority Sort**: Order by priority (highest first)
4. **Preference**: Apply user preferences and overrides
5. **Selection**: Pick the best available option

### Selection Options

```go
type SelectorOptions struct {
    PreferredDriver string   // Specific driver to prefer
    RequireHealthy  bool     // Only select healthy drivers
    RequireDefault  bool     // Only select default drivers
    MinPriority     Priority // Minimum acceptable priority
}
```

### Selection Examples

#### Automatic Selection
```go
options := driver.DefaultSelectorOptions()
pick, alternatives, rejects := driver.Suggest(options)

if pick.Empty() {
    return fmt.Errorf("no suitable driver found")
}

fmt.Printf("Selected driver: %s\n", pick.Name)
```

#### Preferred Driver
```go
options := driver.DefaultSelectorOptions()
options.PreferredDriver = "firecracker"
pick, _, _ := driver.Suggest(options)
```

#### Minimum Requirements
```go
options := driver.SelectorOptions{
    RequireHealthy: true,
    RequireDefault: true,
    MinPriority:    driver.Default,
}
pick, _, _ := driver.Suggest(options)
```

## Health Checking

### Health Check Implementation

Drivers implement comprehensive health checking:

```go
func checkFirecrackerStatus() driver.State {
    // Check if firecracker binary exists
    if _, err := exec.LookPath("firecracker"); err != nil {
        return driver.State{
            Installed: false,
            Healthy:   false,
            Error:     err,
            Fix:       "Install Firecracker from https://github.com/firecracker-microvm/firecracker",
            Doc:       "https://github.com/firecracker-microvm/firecracker/blob/main/docs/getting-started.md",
        }
    }

    // Check permissions
    if !hasKVMAccess() {
        return driver.State{
            Installed: true,
            Healthy:   false,
            Error:     fmt.Errorf("no access to /dev/kvm"),
            Fix:       "Add user to kvm group: sudo usermod -aG kvm $USER",
            Doc:       "https://docs.kernel.org/virt/kvm/api.html",
        }
    }

    // Check version compatibility
    version, err := getFirecrackerVersion()
    if err != nil {
        return driver.State{
            Installed:        true,
            Healthy:          true,
            NeedsImprovement: true,
            Fix:              "Unable to determine Firecracker version",
        }
    }

    if isVersionTooOld(version) {
        return driver.State{
            Installed:        true,
            Healthy:          true,
            NeedsImprovement: true,
            Fix:              fmt.Sprintf("Update Firecracker to v%s or later", minVersion),
        }
    }

    return driver.State{
        Installed: true,
        Healthy:   true,
        Running:   true,
        Version:   version,
    }
}
```

### Common Health Checks

1. **Binary Availability**: Check if required executables exist
2. **Permissions**: Verify user has necessary privileges
3. **Services**: Ensure required daemons are running
4. **Dependencies**: Check for required libraries/modules
5. **Version Compatibility**: Verify compatible versions
6. **Hardware Support**: Check for required hardware features

### Health Status Examples

#### Healthy Driver
```
STATUS: Healthy
- Firecracker v1.4.0 installed
- KVM access available
- All dependencies satisfied
```

#### Unhealthy Driver
```
STATUS: Unhealthy
ERROR: permission denied accessing /dev/kvm
FIX: Add user to kvm group: sudo usermod -aG kvm $USER
DOC: https://docs.kernel.org/virt/kvm/api.html
```

#### Needs Improvement
```
STATUS: Healthy (Needs Improvement)
WARNING: Firecracker v0.24.0 is outdated
FIX: Update to v1.4.0 or later for better performance
```

## Driver Lookup

### Name Resolution

The registry supports multiple ways to reference drivers:

```go
// By primary name
driver := registry.Driver("firecracker")

// By alias
driver := registry.Driver("fc")

// Check if supported
if driver.Supported("firecracker") {
    // Driver is available
}
```

### Alias Management

Aliases provide convenient shortcuts:

```go
// Register with aliases
driver.Register(driver.DriverDef{
    Name:    "firecracker", 
    Aliases: []string{"fc", "microvm"},
    // ...
})

// All of these work:
driver.GetDriver("firecracker")
driver.GetDriver("fc") 
driver.GetDriver("microvm")
```

### Listing Drivers

```go
// Get all registered drivers
drivers := driver.List()

// Get drivers with status
available := driver.Available()

// Get just driver names
names := driver.SupportedDrivers()
```

## Selection Results

### Selection Output

The selector returns three categories:

1. **Pick**: The selected driver (best choice)
2. **Alternatives**: Other suitable drivers
3. **Rejects**: Unsuitable drivers with reasons

```go
pick, alternatives, rejects := driver.Suggest(options)

fmt.Printf("Selected: %s (priority: %d)\n", pick.Name, pick.Priority)

for _, alt := range alternatives {
    fmt.Printf("Alternative: %s (%s)\n", alt.Name, alt.Rejection)
}

for _, rej := range rejects {
    fmt.Printf("Rejected: %s (%s)\n", rej.Name, rej.Rejection)
}
```

### Example Output

```
Selected: firecracker (priority: 10)

Alternatives:
- qemu (firecracker is preferred)
- virtualbox (firecracker is preferred)

Rejected:
- vmware (not installed: Install VMware Workstation)
- hyperv (platform not supported: Linux required)
- docker (priority too low: 4 < 6)
```

## Error Handling

### Registration Errors

```go
// Duplicate name
err := driver.Register(duplicateDriver)
// Error: driver "firecracker" is already registered

// Missing factory
err := driver.Register(driver.DriverDef{Name: "test"})
// Error: driver "test" must have a Create function

// Alias conflict  
err := driver.Register(driver.DriverDef{
    Name: "new",
    Aliases: []string{"fc"}, // Already used by firecracker
})
// Error: alias "fc" is already registered
```

### Selection Errors

```go
// No drivers available
pick, _, _ := driver.Suggest(options)
if pick.Empty() {
    // Handle no suitable driver found
}

// Preferred driver unavailable
options.PreferredDriver = "nonexistent"
pick, _, rejects := driver.Suggest(options)
// Check rejects for reason
```

## Thread Safety

The registry is thread-safe for concurrent access:

- **Registration**: Protected by mutex during startup
- **Lookups**: Safe for concurrent reads
- **Status Checks**: Independent per driver
- **Selection**: Stateless operations

## Performance Considerations

### Lazy Loading

- Status checks are performed on-demand
- Driver instances created only when needed
- Registry caches registration data

### Optimization Tips

1. **Avoid expensive status checks**: Keep checks lightweight
2. **Cache status results**: Cache health check results when appropriate
3. **Parallel health checks**: Check multiple drivers concurrently
4. **Early termination**: Stop checking once good driver is found

## CLI Integration

### Driver Commands

```bash
# List available drivers
spitfire driver list

# Show driver status
spitfire driver status [driver]

# Get driver information  
spitfire driver info <driver>

# Show selection reasoning
spitfire driver suggest --verbose

# Force driver selection
spitfire up --driver firecracker
```

### Configuration Integration

```yaml
# Global driver preference
driver: firecracker

# Driver-specific options
driver_opts:
  firecracker:
    jailer: true
  qemu:
    accel: kvm
    
# Per-VM driver override
vms:
  database:
    driver: qemu  # Use different driver for this VM
    image: postgres:13
```

## Debugging

### Debug Output

Enable verbose driver selection:

```bash
export SPITFIRE_LOG_LEVEL=debug
spitfire up --driver-debug
```

### Common Issues

1. **Driver not found**: Check registration and imports
2. **Selection fails**: Verify health checks and priorities
3. **Status errors**: Check dependencies and permissions
4. **Alias conflicts**: Ensure unique aliases across drivers

## Extension Points

### Custom Priorities

```go
// Define custom priority for specific environments
const ProductionOptimized Priority = 15

// Register with custom priority
driver.Register(driver.DriverDef{
    Priority: ProductionOptimized,
    // ...
})
```

### Plugin System

Future support for external driver plugins:

```go
// Load external driver plugin
plugin, err := driver.LoadPlugin("custom-driver.so")
if err == nil {
    plugin.Register()
}
```

## See Also

- [Driver System Overview](./spitfire-driver-system.md)
- [Driver API Reference](./spitfire-driver-api.md)
- [Driver Development Guide](./spitfire-driver-development.md)