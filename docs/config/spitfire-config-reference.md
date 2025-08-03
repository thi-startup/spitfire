# Spitfire Configuration Reference

This document provides a comprehensive reference for all configuration options available in Spitfire configuration files.

## File Format

Spitfire uses YAML configuration files with a structure inspired by Docker Compose. The configuration file typically named `spitfire.yml` or `spitfire.yaml`.

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

Sets the default virtualization driver for all VMs.

```yaml
driver: "firecracker"
```

**Type**: String  
**Required**: No  
**Default**: Auto-selected based on driver registry priorities  
**Valid Values**: Any registered driver name (`firecracker`, `qemu`, `virtualbox`, etc.)

### `driver_opts` (optional)

Global driver-specific configuration options.

```yaml
driver_opts:
  firecracker:
    jailer: true
    cpu_template: "C3"
  qemu:
    accel: "kvm"
    display: "none"
```

**Type**: Map of driver name to options map  
**Required**: No  
**Structure**: `driver_name: { option: value }`

### `globals` (optional)

Global settings that apply to all VMs unless overridden.

```yaml
globals:
  kernel: "/path/to/vmlinux"
  kernel_args: "console=ttyS0 quiet"
  resources:
    vcpu: 2
    memory: "1GB"
  networks:
    - default
  restart: "always"
  env:
    GLOBAL_VAR: "value"
```

See [Global Configuration](#global-configuration) for detailed options.

### `vms` (required)

Defines the microVMs to be managed.

```yaml
vms:
  web:
    image: "nginx:alpine"
  api:
    image: "app:latest"
```

**Type**: Map of VM name to VM configuration  
**Required**: Yes (at least one VM must be defined)

See [VM Configuration](#vm-configuration) for detailed options.

### `networks` (optional)

Defines custom networks for VM connectivity.

```yaml
networks:
  web-tier:
    driver: bridge
  isolated:
    driver: bridge
    ipam:
      config:
        - subnet: "192.168.100.0/24"
          gateway: "192.168.100.1"
```

See [Network Configuration](#network-configuration) for detailed options.

### `volumes` (optional)

Defines named volumes for persistent storage.

```yaml
volumes:
  database:
    driver: local
    size: "20GB"
    persist: true
  cache:
    driver: tmpfs
    size: "1GB"
```

See [Volume Configuration](#volume-configuration) for detailed options.

## Global Configuration

The `globals` section defines default settings applied to all VMs.

### `kernel` (optional)

Path to the kernel image for VMs.

```yaml
globals:
  kernel: "/opt/kernels/vmlinux"
```

**Type**: String (file path)  
**Required**: No  
**Default**: Driver-specific default

### `kernel_args` (optional)

Kernel command-line arguments.

```yaml
globals:
  kernel_args: "console=ttyS0 noapic reboot=k panic=1"
```

**Type**: String  
**Required**: No  
**Default**: Driver-specific default

### `resources` (optional)

Default resource allocation for VMs.

```yaml
globals:
  resources:
    vcpu: 2
    memory: "1GB"
```

See [Resource Configuration](#resource-configuration) for details.

### `networks` (optional)

Default networks for VMs.

```yaml
globals:
  networks:
    - default
    - monitoring
```

**Type**: Array of network names  
**Required**: No  
**Default**: No networks attached

### `restart` (optional)

Default restart policy for VMs.

```yaml
globals:
  restart: "always"
```

**Type**: String  
**Required**: No  
**Valid Values**: `always`, `on-failure`, `unless-stopped`, `no`  
**Default**: `no`

### `env` (optional)

Global environment variables for all VMs.

```yaml
globals:
  env:
    APP_ENV: "production"
    LOG_LEVEL: "info"
```

**Type**: Map of string to string  
**Required**: No

## VM Configuration

Each VM is defined as a key-value pair under the `vms` section, where the key is the VM name and the value is the configuration.

### `image` (conditionally required)

Container image or VM image to use.

```yaml
vms:
  web:
    image: "nginx:alpine"
```

**Type**: String  
**Required**: Yes (unless `rootfs` is specified)  
**Mutually Exclusive**: Cannot be used with `rootfs`

### `rootfs` (conditionally required)

Path to a rootfs filesystem image.

```yaml
vms:
  custom:
    rootfs: "/path/to/rootfs.ext4"
```

**Type**: String (file path)  
**Required**: Yes (unless `image` is specified)  
**Mutually Exclusive**: Cannot be used with `image`

### `driver` (optional)

Override the global driver for this specific VM.

```yaml
vms:
  development:
    image: "dev:latest"
    driver: "qemu"
```

**Type**: String  
**Required**: No  
**Default**: Uses global `driver` setting

### `driver_opts` (optional)

VM-specific driver options that override global driver options.

```yaml
vms:
  debug-vm:
    image: "app:debug"
    driver_opts:
      log_level: "Debug"
      enable_tracing: true
```

**Type**: Map of string to any  
**Required**: No

### `resources` (optional)

Resource allocation for this VM.

```yaml
vms:
  database:
    image: "postgres:13"
    resources:
      vcpu: 4
      memory: "4GB"
```

See [Resource Configuration](#resource-configuration) for details.

### `volumes` (optional)

Volume mounts for this VM.

```yaml
vms:
  app:
    image: "app:latest"
    volumes:
      - type: "bind"
        source: "./config"
        target: "/app/config"
      - type: "volume"
        source: "app-data"
        target: "/data"
```

See [Volume Mount Configuration](#volume-mount-configuration) for details.

### `env` (optional)

Environment variables for this VM.

```yaml
vms:
  api:
    image: "api:latest"
    env:
      API_PORT: "8080"
      DB_HOST: "database"
```

**Type**: Map of string to string  
**Required**: No

### `env_file` (optional)

Load environment variables from files.

```yaml
vms:
  app:
    image: "app:latest"
    env_file:
      - ".env"
      - "production.env"
```

**Type**: Array of file paths  
**Required**: No

### `depends_on` (optional)

Specify VM startup dependencies.

```yaml
vms:
  web:
    image: "nginx:alpine"
    depends_on:
      - api
      - database
```

**Type**: Array of VM names  
**Required**: No

### `networks` (optional)

Networks to attach this VM to.

```yaml
vms:
  web:
    image: "nginx:alpine"
    networks:
      - frontend
      - backend
```

**Type**: Array of network names  
**Required**: No

### `restart` (optional)

Restart policy for this VM.

```yaml
vms:
  worker:
    image: "worker:latest"
    restart: "on-failure"
```

**Type**: String  
**Required**: No  
**Valid Values**: `always`, `on-failure`, `unless-stopped`, `no`

### `kernel` (optional)

Override global kernel for this VM.

```yaml
vms:
  special:
    image: "special:latest"
    kernel: "/opt/special-kernel/vmlinux"
```

**Type**: String (file path)  
**Required**: No

### `kernel_args` (optional)

Override global kernel arguments for this VM.

```yaml
vms:
  debug:
    image: "app:debug"
    kernel_args: "console=ttyS0 loglevel=8 debug"
```

**Type**: String  
**Required**: No

## Resource Configuration

Resource configuration defines CPU and memory allocation.

### `vcpu` (optional)

Number of virtual CPUs.

```yaml
resources:
  vcpu: 4
```

**Type**: Integer  
**Required**: No  
**Default**: 1  
**Range**: 1-64 (driver dependent)

### `memory` (optional)

Memory allocation with size suffix.

```yaml
resources:
  memory: "2GB"
```

**Type**: String with size suffix  
**Required**: No  
**Default**: "512MB"  
**Valid Suffixes**: `MB`, `M`, `GB`, `G`  
**Examples**: `"512MB"`, `"1GB"`, `"2G"`, `"256M"`

## Volume Mount Configuration

Volume mounts attach storage to VMs.

### Short Syntax

```yaml
volumes:
  - "./host/path:/container/path"
  - "volume-name:/container/path"
```

### Long Syntax

```yaml
volumes:
  - type: "bind"
    source: "./host/path"
    target: "/container/path"
  - type: "volume"
    source: "volume-name"
    target: "/data"
```

### `type` (optional)

Mount type.

**Type**: String  
**Required**: No (auto-detected)  
**Valid Values**: `bind`, `volume`  
**Auto-detection**: Paths with `/` are treated as `bind`, otherwise `volume`

### `source` (required)

Source of the mount.

**Type**: String  
**Required**: Yes  
**For bind mounts**: Host filesystem path  
**For volume mounts**: Named volume reference

### `target` (required)

Mount point inside the VM.

**Type**: String  
**Required**: Yes

## Network Configuration

Networks define isolated network segments for VMs.

### `driver` (optional)

Network driver type.

```yaml
networks:
  web-net:
    driver: bridge
```

**Type**: String  
**Required**: No  
**Default**: `bridge`  
**Valid Values**: Driver-dependent (`bridge`, `host`, `none`)

### `ipam` (optional)

IP Address Management configuration.

```yaml
networks:
  custom:
    ipam:
      config:
        - subnet: "192.168.100.0/24"
          gateway: "192.168.100.1"
```

#### `config` (optional)

Array of IPAM configuration blocks.

##### `subnet` (optional)

Network subnet in CIDR notation.

**Type**: String (CIDR)  
**Example**: `"192.168.100.0/24"`

##### `gateway` (optional)

Gateway IP address.

**Type**: String (IP address)  
**Example**: `"192.168.100.1"`

### `options` (optional)

Driver-specific network options.

```yaml
networks:
  custom:
    options:
      com.docker.network.bridge.name: "spitfire0"
      com.docker.network.driver.mtu: "1450"
```

**Type**: Map of string to string

## Volume Configuration

Volumes define persistent storage that can be shared between VMs.

### `driver` (optional)

Volume driver type.

```yaml
volumes:
  data:
    driver: local
```

**Type**: String  
**Required**: No  
**Default**: `local`  
**Valid Values**: Driver-dependent (`local`, `tmpfs`, `nfs`)

### `persist` (optional)

Whether the volume should persist after VM deletion.

```yaml
volumes:
  database:
    persist: true
```

**Type**: Boolean  
**Required**: No  
**Default**: `false`

### `size` (optional)

Volume size with suffix.

```yaml
volumes:
  cache:
    size: "10GB"
```

**Type**: String with size suffix  
**Required**: No  
**Valid Suffixes**: `MB`, `M`, `GB`, `G`, `TB`, `T`

### `options` (optional)

Driver-specific volume options.

```yaml
volumes:
  nfs-data:
    driver: nfs
    options:
      server: "192.168.1.100"
      share: "/exports/data"
```

**Type**: Map of string to string

## Environment Variable Expansion

Environment variables can reference other variables or system environment variables.

### Syntax

```yaml
env:
  HOME_DIR: "${HOME}/app"
  CONFIG_PATH: "${HOME_DIR}/config"
  PORT: "${PORT:-8080}"  # Default value syntax
```

### Supported Formats

- `${VAR}` - Variable substitution
- `${VAR:-default}` - Variable with default value
- `$VAR` - Short form variable substitution

## Validation Rules

### Required Fields

- `version` must be specified
- At least one VM must be defined in `vms`
- Each VM must have either `image` or `rootfs` (but not both)

### Name Restrictions

- VM names must be valid identifiers
- Network names must be unique within the configuration
- Volume names must be unique within the configuration

### Reference Validation

- All network references in VMs must exist in `networks` section
- All volume references in VMs must exist in `volumes` section
- All `depends_on` references must point to existing VMs

### Driver Validation

- Driver names must be lowercase alphanumeric with hyphens
- Driver options are validated by the specific driver implementation

## Configuration Examples

See the [Configuration System Documentation](spitfire-config-system.md) for comprehensive examples and usage patterns.