# Spitfire Configuration Reference

This document provides a comprehensive reference for all configuration options available in Spitfire configuration files.

## File Format

Spitfire uses YAML configuration files with a structure inspired by Docker Compose. The configuration file is typically named `spitfire.yaml` or `spitfire.yml`.

## Configuration Loading

Spitfire VM commands (`spitfire vm up/down/ps`) operate on project-based configurations:

- **Default Behavior**: Looks for `spitfire.yaml` in the current directory
- **Custom Config**: Use `-f` flag to specify alternate config file
- **Project-Based**: Each directory with a config file represents a project
- **State Management**: VM state is persisted per project in `~/.spitfire/projects/`

## Root Level Properties

### `version` (required)

Specifies the configuration format version.

```yaml
version: "1"
```

**Type**: String  
**Required**: Yes  
**Values**: Currently only `"1"` is supported

### `driver` (optional)

Sets the default virtualization driver for all VMs in this project.

```yaml
driver: "firecracker"
```

**Type**: String  
**Required**: No  
**Default**: Auto-selected based on driver registry priorities  
**Valid Values**: Any registered driver name (use `spitfire driver ls` to see available drivers)

### `vms` (required)

Defines the virtual machines for this project. Each VM will be created and managed as part of the project lifecycle.

```yaml
vms:
  web:
    image: /path/to/rootfs.ext4
    resources:
      vcpu: 2
      memory: 1GB
  database:
    image: /path/to/db-rootfs.ext4
    resources:
      vcpu: 1
      memory: 512MB
```

**Type**: Map of VM name to VM configuration  
**Required**: Yes (at least one VM must be defined)  
**Structure**: `vm_name: { vm_config }`

## VM Configuration Properties

Each VM in the `vms` section supports the following configuration options:

### `image` (required)

Path to the rootfs image file for the VM.

```yaml
vms:
  web:
    image: /opt/spitfire/ubuntu.ext4
```

**Type**: String  
**Required**: Yes  
**Description**: Must be a valid path to an ext4 filesystem image that will serve as the VM's root filesystem

### `resources` (optional)

CPU and memory resource allocation for the VM.

```yaml
vms:
  web:
    resources:
      vcpu: 2
      memory: 1GB
```

**Type**: Object  
**Required**: No  
**Properties**:
- `vcpu` (int): Number of virtual CPUs (default: 1)  
- `memory` (string): Memory allocation with units (MB, GB) (default: "512MB")

### `env` (optional)

Environment variables to set in the VM.

```yaml
vms:
  web:
    env:
      NODE_ENV: production
      PORT: "3000"
```

**Type**: Map of string to string  
**Required**: No  
**Description**: Environment variables passed to processes running in the VM

### `driver_opts` (optional)

Driver-specific configuration options for this VM.

```yaml
vms:
  web:
    driver_opts:
      kernel: /opt/spitfire/vmlinux
      debug: true
      cpu_template: "C3"
```

**Type**: Map of option name to value  
**Required**: No  
**Description**: Options specific to the driver being used. Available options depend on the selected driver.

#### Firecracker Driver Options
- `kernel` (string): Path to kernel binary (vmlinux file)
- `debug` (bool): Enable debug logging
- `cpu_template` (string): CPU template for optimization ("C3", "T2", etc.)
- `jailer` (bool): Run with jailer for additional security

## Complete Example

```yaml
version: "1"
driver: firecracker

vms:
  web:
    image: /opt/spitfire/ubuntu.ext4
    resources:
      vcpu: 2
      memory: 1GB
    env:
      NODE_ENV: production
      PORT: "3000"
    driver_opts:
      kernel: /opt/spitfire/hello-vmlinux.bin
      debug: false

  database:
    image: /opt/spitfire/postgres.ext4
    resources:
      vcpu: 1
      memory: 512MB
    env:
      POSTGRES_DB: myapp
      POSTGRES_USER: admin
    driver_opts:
      kernel: /opt/spitfire/hello-vmlinux.bin
```

## Project Lifecycle

When you run VM commands, Spitfire:

1. **`spitfire vm up`**: Creates and starts all VMs defined in the config
2. **`spitfire vm ps`**: Shows status of all VMs in the current project  
3. **`spitfire vm down`**: Stops and removes all VMs in the current project

Each project maintains its own state in `~/.spitfire/projects/<project-name>/` for persistence across restarts.

## Usage Commands

The configuration file drives the following VM lifecycle commands:

```bash
# Start all VMs defined in the config
spitfire vm up

# List running VMs for this project  
spitfire vm ps

# Stop and remove all VMs in this project
spitfire vm down
```

## See Also

- [Configuration Examples](./spitfire-config-examples.md) - Real-world configuration examples
- [Configuration System](./spitfire-config-system.md) - Advanced configuration patterns
- [Driver System](../driver/spitfire-driver-system.md) - Understanding driver selection and configuration
