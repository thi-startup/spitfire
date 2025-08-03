# Spitfire Configuration System

The Spitfire configuration system provides a Docker Compose-like YAML interface for defining and managing microVMs across different virtualization drivers. This system bridges the gap between user-friendly configuration files and the underlying driver implementations.

## Overview

The configuration system consists of two main components:

1. **Configuration Types** (`pkg/config/types.go`) - Define the structure and validation of configuration files
2. **Driver Mapping** (`pkg/config/mapping.go`) - Convert configuration to driver-specific formats

The system supports:
- Global defaults with VM-specific overrides
- Driver selection and driver-specific options
- Resource management (CPU, memory, storage)
- Network and volume configuration
- Environment variable management

## Configuration Structure

### Root Configuration

```yaml
version: "1"
driver: firecracker                    # Global default driver
driver_opts:                          # Global driver options
  firecracker:
    jailer: true
    cpu_template: "C3"
  qemu:
    accel: "kvm"

globals:                              # Global VM defaults
  kernel: "/path/to/vmlinux"
  kernel_args: "console=ttyS0"
  resources:
    vcpu: 2
    memory: "1GB"
  networks:
    - default
  env:
    GLOBAL_VAR: "value"

vms:                                  # VM definitions
  web:
    image: "nginx:alpine"
    # VM-specific configuration...

networks:                             # Network definitions
  default:
    driver: bridge

volumes:                              # Volume definitions
  data:
    driver: local
    size: "10GB"
```

### VM Configuration

Each VM can override global settings and specify driver-specific options:

```yaml
vms:
  web:
    image: "nginx:alpine"             # Container image or rootfs
    driver: "qemu"                    # Override global driver
    driver_opts:                      # VM-specific driver options
      accel: "tcg"                    # Override global qemu.accel
      display: "vnc=:1"
    
    resources:
      vcpu: 4                         # Override global CPU count
      memory: "2GB"                   # Override global memory
    
    volumes:
      - type: "bind"
        source: "./html"
        target: "/var/www/html"
      - type: "volume"
        source: "data"
        target: "/app/data"
    
    networks:
      - web-net
      - database-net
    
    env:
      NGINX_PORT: "80"
      GLOBAL_VAR: "overridden"        # Override global env var
    
    restart: "always"
    depends_on:
      - database
```

## Driver Integration

### Driver Selection

The configuration system supports flexible driver selection:

1. **Global Default**: Set in the root `driver` field
2. **VM Override**: Set in the VM's `driver` field
3. **Auto-Selection**: When no driver is specified, the system uses driver registry priorities

```yaml
# Example: Mixed driver usage
version: "1"
driver: "firecracker"                 # Global default

vms:
  microservice:
    image: "app:latest"               # Uses firecracker (global default)
  
  development:
    image: "dev:latest"
    driver: "qemu"                    # Uses qemu (VM override)
  
  testing:
    image: "test:latest"
    driver: ""                        # Uses auto-selection
```

### Driver Options

Driver-specific options follow a hierarchical override pattern:

```yaml
driver_opts:
  firecracker:                        # Global options for firecracker
    jailer: true
    cpu_template: "C3"
    log_level: "Info"

vms:
  secure-vm:
    driver: "firecracker"
    driver_opts:                      # VM-specific options
      log_level: "Debug"              # Overrides global log_level
      kernel_args: "quiet"            # Adds new option
    # Final options: jailer=true, cpu_template="C3", log_level="Debug", kernel_args="quiet"
```

### Driver Config Mapping

The system automatically converts configuration to driver-compatible formats:

```go
// Configuration is mapped to driver.Config
driverConfig := &driver.Config{
    Name:       "vm-name",
    Image:      vm.Image,
    Memory:     512,                  // Parsed from "512MB"
    CPUs:       2,
    DriverOpts: mergedOptions,        // Global + VM options
    Env:        mergedEnvVars,        // Global + VM env vars
    Networks:   effectiveNetworks,   // Global + VM networks
    Volumes:    convertedVolumes,    // Converted volume mounts
}
```

## Resource Management

### Memory Configuration

Memory can be specified in various formats:

```yaml
resources:
  memory: "512MB"    # Megabytes
  memory: "1GB"      # Gigabytes  
  memory: "2G"       # Short format
  memory: "256M"     # Short format
```

The system converts all formats to megabytes for driver consumption.

### CPU Configuration

```yaml
resources:
  vcpu: 4            # Number of virtual CPUs
```

### Resource Inheritance

Resources follow the global → VM inheritance pattern:

```yaml
globals:
  resources:
    vcpu: 2
    memory: "1GB"

vms:
  app1:
    # Inherits: vcpu=2, memory="1GB"
    
  app2:
    resources:
      memory: "512MB"  # Overrides memory, inherits vcpu=2
```

## Validation System

The configuration system provides comprehensive validation:

### Driver Validation

```go
// Driver names must be lowercase alphanumeric with hyphens
validateDriverName("firecracker")     // ✓ Valid
validateDriverName("qemu-kvm")        // ✓ Valid
validateDriverName("Invalid-Name!")   // ✗ Invalid
```

### VM Validation

- Either `image` or `rootfs` must be specified (but not both)
- Restart policies must be valid: `always`, `on-failure`, `unless-stopped`, `no`
- Volume mounts must have valid source and target paths
- Network and volume references must exist in the configuration

### Reference Validation

The system ensures referential integrity:

```yaml
networks:
  web-net:
    driver: bridge

volumes:
  app-data:
    driver: local

vms:
  app:
    networks:
      - web-net      # ✓ Valid - network exists
      - missing-net  # ✗ Invalid - network not defined
    volumes:
      - source: app-data  # ✓ Valid - volume exists
        target: /data
```

## Environment Variables

### Variable Inheritance

Environment variables merge from global to VM-specific:

```yaml
globals:
  env:
    APP_ENV: "production"
    DEBUG: "false"

vms:
  service:
    env:
      DEBUG: "true"        # Overrides global DEBUG
      SERVICE_PORT: "8080" # Adds new variable
    # Final env: APP_ENV="production", DEBUG="true", SERVICE_PORT="8080"
```

### Variable Expansion

The system supports environment variable expansion:

```yaml
env:
  HOME_DIR: "${HOME}/app"
  CONFIG_PATH: "${HOME_DIR}/config"
```

## Usage Examples

### Basic Firecracker Setup

```yaml
version: "1"
driver: "firecracker"

globals:
  kernel: "/opt/firecracker/vmlinux"
  resources:
    memory: "512MB"
    vcpu: 1

vms:
  web:
    image: "nginx:alpine"
    env:
      NGINX_PORT: "80"
```

### Multi-Driver Environment

```yaml
version: "1"
driver: "firecracker"

driver_opts:
  firecracker:
    jailer: true
  qemu:
    accel: "kvm"
    display: "none"

vms:
  microservice:
    image: "app:latest"        # Uses firecracker
    
  development:
    image: "dev:latest"
    driver: "qemu"             # Uses qemu for development
    driver_opts:
      display: "vnc=:1"        # Override for development
```

### Complex Resource Configuration

```yaml
version: "1"

globals:
  resources:
    vcpu: 2
    memory: "1GB"
  networks:
    - default

networks:
  default:
    driver: bridge
  isolated:
    driver: bridge
    ipam:
      config:
        - subnet: "192.168.100.0/24"

volumes:
  database:
    driver: local
    size: "20GB"
    persist: true

vms:
  web:
    image: "nginx:alpine"
    resources:
      memory: "512MB"          # Override global memory
    networks:
      - default
      - isolated
    
  database:
    image: "postgres:13"
    resources:
      vcpu: 4                  # Override global CPU
      memory: "2GB"            # Override global memory
    volumes:
      - source: database
        target: /var/lib/postgresql/data
    env:
      POSTGRES_DB: "app"
      POSTGRES_USER: "user"
    networks:
      - isolated               # Isolated network only
```

## Integration with Driver System

The configuration system seamlessly integrates with the driver registry:

1. **Driver Selection**: Uses registry priorities when no driver specified
2. **Option Validation**: Driver-specific options are passed through for driver validation
3. **Resource Mapping**: Configuration resources are mapped to driver.Config format
4. **Error Handling**: Configuration errors are reported with context

This design allows the configuration system to remain driver-agnostic while providing rich integration with the underlying driver implementations.