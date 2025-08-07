# Spitfire Driver System

## Overview

Spitfire uses a modular driver system that allows it to work with multiple virtualization backends. This architecture, inspired by minikube's driver model, provides a consistent interface while supporting different underlying technologies like Firecracker, QEMU, VirtualBox, and more.

## Why a Driver System?

The driver system provides several key benefits:

1. **Technology Agnostic**: Not locked into a single virtualization platform
2. **Future Proof**: Easy to add support for new virtualization technologies
3. **Development Flexibility**: Use faster drivers for development, optimized drivers for production
4. **Platform Support**: Different drivers can optimize for different platforms and use cases
5. **Community Contributions**: Easy for the community to add new driver implementations

## Architecture Overview

The driver system consists of four main components:

### 1. Driver Interface
A standardized interface that all drivers must implement, covering:
- VM lifecycle management (create, start, stop, delete)
- State management and information retrieval
- Network and volume operations
- Feature detection and capabilities

### 2. Driver Registry
A central registry that manages:
- Driver registration and discovery
- Health checking and status reporting
- Priority-based driver selection
- Alias management for driver names

### 3. Driver Selection Engine
Intelligent driver selection based on:
- Driver health and availability
- User preferences and requirements
- Priority levels and compatibility
- Platform-specific optimizations

### 4. Configuration Integration
Seamless integration with spitfire's configuration system:
- Driver selection in project configs
- Driver-specific options and settings
- Global driver preferences

## Supported Drivers

### Current Status

| Driver | Status | Priority | Description |
|--------|--------|----------|-------------|
| **Firecracker** | ✅ Production Ready | HighlyPreferred | AWS Firecracker microVM driver with comprehensive setup automation |
| **Mock** | ✅ Available | Default | Testing and development driver |
| QEMU | 📋 Planned | Default | Full system virtualization with KVM acceleration |
| VirtualBox | 📋 Planned | Fallback | Cross-platform virtualization for development |
| Docker | 📋 Planned | Default | Container-based lightweight virtualization |

## Driver Management Commands

Spitfire provides comprehensive driver management through the `spitfire driver` command:

### Listing Drivers
```bash
# List all available drivers with status
spitfire driver ls

# Output in JSON format
spitfire driver ls -o json
```

### Driver Information
```bash
# Show detailed information about a specific driver
spitfire driver info firecracker
```

### Driver Setup
```bash
# Run automated setup for a driver
spitfire driver setup firecracker

# Dry-run to see what would be done
spitfire driver setup firecracker --dry-run

# Interactive setup with prompts
spitfire driver setup firecracker --interactive

# Force setup without confirmations
spitfire driver setup firecracker --force
```

### Setup Verification
```bash
# Verify driver setup status
spitfire driver verify firecracker

# JSON output for programmatic use
spitfire driver verify firecracker -o json
```

### Setup Instructions
```bash
# Show comprehensive setup instructions
spitfire driver instructions firecracker
```

## Driver Setup System

Each driver implements a comprehensive setup system to configure host prerequisites:

### Setup Operations
1. **Automated Setup**: `Setup()` method configures host system automatically
2. **Setup Verification**: `VerifySetup()` checks current configuration status
3. **Setup Instructions**: `GetSetupInstructions()` provides comprehensive guidance

### Setup Features
- **Dry-run Mode**: Test setup without making changes
- **Component-based Setup**: Configure specific components (networking, permissions, etc.)
- **Interactive Mode**: User prompts for configuration decisions
- **Comprehensive Verification**: Component-by-component status reporting
- **Issue Resolution**: Specific commands and steps to fix problems

### Example: Firecracker Setup
The Firecracker driver setup includes:
- KVM permissions and device access configuration
- Network bridge creation and IP configuration
- iptables firewall rules for VM networking
- Required directory structure creation
- IP forwarding enablement
- User group membership management

## VM Lifecycle Management

VMs are managed through the `spitfire vm` command, which integrates seamlessly with the driver system:

### Creating and Starting VMs
```bash
# Start all VMs defined in spitfire.yaml
spitfire vm up

# Use specific config file
spitfire vm up -f custom-spitfire.yaml

# Run in background (detached mode)
spitfire vm up --detach
```

### Viewing Running VMs
```bash
# List running VMs
spitfire vm ps

# Show all VMs (including stopped)
spitfire vm ps --all

# Quiet mode (just VM names)
spitfire vm ps --quiet

# JSON output
spitfire vm ps -o json
```

### Stopping VMs
```bash
# Stop all VMs in the current project
spitfire vm down

# Also remove volumes
spitfire vm down --volumes
```

### VM Configuration Integration
VMs are defined in `spitfire.yaml` files and the driver system automatically:
- Selects appropriate drivers based on configuration
- Converts VM configurations to driver-specific formats
- Manages VM state persistence across restarts
- Handles driver-specific optimization and features

### Driver Priorities

- **HighlyPreferred**: Best performance and integration
- **Preferred**: Good performance with broader compatibility  
- **Default**: Reliable but may have limitations
- **Fallback**: Works but not recommended for primary use
- **Experimental**: Early stage, may have issues

## Basic Usage

### Automatic Driver Selection

By default, spitfire automatically selects the best available driver:

```bash
# Spitfire chooses the best driver automatically
spitfire up
```

### Explicit Driver Selection

You can specify a driver explicitly:

```bash
# Use a specific driver
spitfire up --driver firecracker

# Or via configuration
spitfire up -f config.yaml
```

### Configuration File

```yaml
version: "1"

# Driver selection
driver: firecracker

# Driver-specific options
driver_opts:
  firecracker:
    jailer: true
    kernel_args: "console=ttyS0 reboot=k panic=1"
  qemu:
    accel: kvm
    display: none

globals:
  # Standard spitfire configuration
  kernel: "/path/to/vmlinux"
  resources:
    memory: "512MB"
    vcpu: 1

vms:
  web:
    image: "nginx:alpine"
    ports:
      - "8080:80"
```

## Driver Information Commands

```bash
# List all available drivers
spitfire driver list

# Show driver status and health
spitfire driver status

# Get detailed driver information
spitfire driver info firecracker

# Show supported features
spitfire driver features qemu
```

## Driver States and Health

Each driver reports its state:

- **Installed**: Driver binary/dependencies are available
- **Healthy**: Driver passes all health checks
- **Running**: Driver daemon/service is operational
- **NeedsImprovement**: Works but could be optimized

Example output:
```
$ spitfire driver status
NAME         STATUS    PRIORITY         VERSION    NOTES
firecracker  Healthy   HighlyPreferred  v1.4.0     Ready
qemu         Healthy   Preferred        v7.2.0     KVM acceleration enabled
virtualbox   Unhealthy Default          -          Not installed
```

## Configuration Examples

### Development Setup
```yaml
# Fast iteration with QEMU
driver: qemu
driver_opts:
  qemu:
    accel: kvm
    snapshot: true  # Fast resets

vms:
  app:
    image: "myapp:latest"
    volumes:
      - type: bind
        source: ./src
        target: /app/src
```

### Production Setup
```yaml
# Security-focused with Firecracker
driver: firecracker
driver_opts:
  firecracker:
    jailer: true
    seccomp: true

vms:
  api:
    image: "myapi:v1.0.0"
    resources:
      memory: "1GB"
      vcpu: 2
```

### Cross-Platform Setup
```yaml
# Works on Windows, macOS, Linux
driver: virtualbox
driver_opts:
  virtualbox:
    gui: false
    
vms:
  database:
    image: "postgres:13"
    volumes:
      - db-data:/var/lib/postgresql/data
```

## Error Handling and Troubleshooting

### Common Issues

1. **Driver Not Found**
   ```
   Error: driver 'firecracker' not found
   Solution: Install Firecracker or use a different driver
   ```

2. **Driver Unhealthy**
   ```
   Error: driver 'firecracker' is not healthy: permission denied
   Solution: Add user to 'kvm' group and relogin
   ```

3. **Feature Not Supported**
   ```
   Warning: driver 'basic' does not support networking
   Solution: Use a driver with networking support
   ```

### Diagnostic Commands

```bash
# Check driver health
spitfire driver diagnose firecracker

# Validate configuration
spitfire config validate -f spitfire.yaml

# Show driver selection reasoning
spitfire driver suggest --verbose
```

## Performance Considerations

### Driver Performance Characteristics

| Driver | Boot Time | Memory Overhead | CPU Overhead | Security |
|--------|-----------|-----------------|--------------|----------|
| Firecracker | ~150ms | ~5MB | Low | High |
| QEMU/KVM | ~500ms | ~50MB | Low | Medium |
| VirtualBox | ~2s | ~100MB | Medium | Medium |

### Optimization Tips

1. **Development**: Use QEMU with snapshots for fast iteration
2. **Production**: Use Firecracker for minimal overhead
3. **CI/CD**: Use mock driver for testing configuration
4. **Cross-platform**: Use VirtualBox for consistency

## Integration with Spitfire Features

### Networking
Different drivers handle networking differently:
- **Firecracker**: TAP devices with custom bridge setup
- **QEMU**: Built-in networking with bridge/NAT options
- **VirtualBox**: Host-only or NAT networking

### Storage
Volume handling varies by driver:
- **Firecracker**: Raw block devices
- **QEMU**: qcow2 images with snapshot support
- **VirtualBox**: VDI/VMDK format support

### Resource Management
Resource limit enforcement:
- **Firecracker**: Strict limits enforced by hypervisor
- **QEMU**: Cgroup-based limiting
- **VirtualBox**: VirtualBox resource controls

## Future Roadmap

### Planned Drivers
1. **Firecracker Driver** (v0.2.0)
   - Full Firecracker API integration
   - Jailer support for security
   - Custom kernel management

2. **QEMU Driver** (v0.3.0)
   - KVM acceleration support
   - Snapshot and restore capabilities
   - Cross-platform compatibility

3. **Container Drivers** (v0.4.0)
   - Docker driver for development
   - Podman driver integration
   - Lightweight alternatives

### Advanced Features
- Driver plugin system for external drivers
- Remote driver support for distributed setups
- Driver migration for moving VMs between backends
- Performance profiling and optimization tools

## See Also

- [Driver Development Guide](./spitfire-driver-development.md)
- [Driver API Reference](./spitfire-driver-api.md)
- [Configuration Reference](./spitfire-project-roadmaps.md)
- [Troubleshooting Guide](./spitfire-troubleshooting.md)