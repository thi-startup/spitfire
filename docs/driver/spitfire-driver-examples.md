# Spitfire Driver Examples and Best Practices

## Overview

This document provides practical examples and best practices for working with the Spitfire driver system. Whether you're using existing drivers or developing new ones, these examples will help you get the most out of the driver architecture.

## Basic Usage Examples

### Automatic Driver Selection

Let spitfire choose the best driver for your platform:

```bash
# Automatic selection based on availability and health
spitfire up

# View selection reasoning
spitfire driver suggest --verbose
```

Example output:
```
Evaluating drivers for selection...
✓ firecracker: Healthy (priority: HighlyPreferred)
✓ qemu: Healthy (priority: Preferred) 
✗ virtualbox: Not installed
✗ vmware: Permission denied

Selected: firecracker
Reason: Highest priority healthy driver
```

### Explicit Driver Selection

Choose a specific driver when you need specific features:

```bash
# Use Firecracker for production workloads
spitfire up --driver firecracker

# Use QEMU for development with snapshots
spitfire up --driver qemu --dev-mode

# Use VirtualBox for cross-platform compatibility
spitfire up --driver virtualbox
```

### Configuration-Based Selection

Define driver preferences in your project configuration:

```yaml
# spitfire.yaml
version: "1"

# Primary driver selection
driver: firecracker

# Fallback driver chain
driver_fallback:
  - qemu
  - virtualbox

# Driver-specific configuration
driver_opts:
  firecracker:
    jailer: true
    kernel_args: "console=ttyS0 reboot=k panic=1"
    seccomp: true
  qemu:
    accel: kvm
    snapshot: true
    display: none
  virtualbox:
    gui: false
    nat_network: true

vms:
  webapp:
    image: "nginx:alpine"
    ports:
      - "8080:80"
    resources:
      memory: "512MB"
      vcpu: 1
```

## Environment-Specific Examples

### Development Environment

Optimized for fast iteration and debugging:

```yaml
# dev-environment.yaml
version: "1"
driver: qemu

driver_opts:
  qemu:
    accel: kvm
    snapshot: true     # Fast VM resets
    gdb_port: 1234    # Debug support
    monitor: stdio    # Interactive monitor

globals:
  resources:
    memory: "1GB"
    vcpu: 2
  volumes:
    - type: bind
      source: ./src
      target: /app/src  # Live code reloading

vms:
  app:
    image: "node:18-alpine"
    command: ["npm", "run", "dev"]
    ports:
      - "3000:3000"
    env:
      NODE_ENV: development
      DEBUG: "*"
```

Usage:
```bash
# Start development environment
spitfire up -f dev-environment.yaml

# Attach to running app for debugging
spitfire exec app /bin/sh

# Reset VM quickly with snapshots
spitfire vm reset app
```

### Production Environment

Security and performance focused:

```yaml
# production.yaml
version: "1"
driver: firecracker

driver_opts:
  firecracker:
    jailer: true           # Enable jailer for isolation
    seccomp: true         # Enable seccomp filtering
    kernel_args: "console=ttyS0 reboot=k panic=1 pci=off"
    cpu_template: T2      # AWS T2 CPU template
    
globals:
  restart: always
  networks:
    - production-net
  resources:
    memory: "2GB"
    vcpu: 2

networks:
  production-net:
    driver: bridge
    ipam:
      config:
        - subnet: "10.0.1.0/24"
          gateway: "10.0.1.1"

vms:
  api:
    image: "myapp/api:v1.2.0"
    replicas: 3          # Load balancing
    ports:
      - "80:8080"
    env:
      NODE_ENV: production
      LOG_LEVEL: info
    volumes:
      - api-data:/app/data
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  database:
    image: "postgres:15-alpine"
    resources:
      memory: "4GB"
      vcpu: 4
    volumes:
      - db-data:/var/lib/postgresql/data
    env:
      POSTGRES_DB: myapp
      POSTGRES_USER: dbuser
      POSTGRES_PASSWORD_FILE: /run/secrets/db_password
    secrets:
      - db_password

volumes:
  api-data:
    driver: local
    persist: true
  db-data:
    driver: local
    persist: true
    size: "100GB"

secrets:
  db_password:
    file: ./secrets/db_password.txt
```

### CI/CD Environment

Fast and reliable for automated testing:

```yaml
# ci.yaml
version: "1"
driver: mock  # Fast, no real VMs for testing

driver_opts:
  mock:
    simulate_delay: false  # Instant operations

globals:
  resources:
    memory: "512MB"
    vcpu: 1

vms:
  test-app:
    image: "myapp:test"
    command: ["npm", "test"]
    env:
      NODE_ENV: test
      CI: "true"
  
  test-db:
    image: "postgres:15-alpine"
    env:
      POSTGRES_DB: testdb
      POSTGRES_USER: test
      POSTGRES_PASSWORD: test
```

CI script:
```bash
#!/bin/bash
# .github/workflows/test.yml

# Validate configuration
spitfire config validate -f ci.yaml

# Run tests
spitfire up -f ci.yaml
spitfire exec test-app npm test
spitfire down -f ci.yaml
```

## Driver-Specific Examples

### Firecracker Examples

#### Minimal Security-Focused Setup
```yaml
driver: firecracker
driver_opts:
  firecracker:
    jailer: true
    jailer_uid: 123
    jailer_gid: 456
    chroot_base: /srv/jailer
    seccomp: true
    kernel: /opt/spitfire/vmlinux
    
vms:
  secure-app:
    image: "alpine:latest"
    command: ["/bin/sh", "-c", "while true; do sleep 30; done"]
    resources:
      memory: "128MB"
      vcpu: 1
```

#### High-Performance Setup
```yaml
driver: firecracker
driver_opts:
  firecracker:
    cpu_template: C3      # High-performance template
    kernel_args: "console=ttyS0 reboot=k panic=1 pci=off nokaslr"
    rate_limiters:
      network:
        bandwidth: 1000000000  # 1 Gbps
        ops: 10000
      disk:
        bandwidth: 100000000   # 100 MBps
        ops: 1000

vms:
  high-perf-app:
    image: "myapp:optimized"
    resources:
      memory: "8GB"
      vcpu: 8
```

### QEMU Examples

#### Development with Debugging
```yaml
driver: qemu
driver_opts:
  qemu:
    accel: kvm
    gdb_port: 1234
    monitor: telnet:localhost:4444,server,nowait
    snapshot: true
    display: vnc:localhost:5900

vms:
  debug-app:
    image: "ubuntu:22.04"
    command: ["/usr/bin/gdbserver", "0.0.0.0:1234", "/app/myapp"]
    ports:
      - "1234:1234"  # GDB port
      - "5900:5900"  # VNC port
```

#### Cross-Architecture Emulation
```yaml
driver: qemu
driver_opts:
  qemu:
    arch: aarch64
    machine: virt
    cpu: cortex-a57
    accel: tcg  # Software emulation

vms:
  arm-app:
    image: "arm64v8/alpine:latest"
    command: ["./my-arm-binary"]
```

### VirtualBox Examples

#### Cross-Platform Development
```yaml
driver: virtualbox
driver_opts:
  virtualbox:
    gui: false
    vrde: true
    vrde_port: 5555
    nat_network: spitfire-net
    shared_folders:
      - name: project
        path: .
        mount: /project

vms:
  cross-platform-app:
    image: "ubuntu:22.04"
    ports:
      - "8080:80"
    volumes:
      - type: shared
        source: project
        target: /project
```

## Advanced Configuration Patterns

### Multi-Driver Setup

Use different drivers for different components:

```yaml
version: "1"

# Default driver
driver: firecracker

vms:
  # Use Firecracker for production app
  api:
    driver: firecracker
    image: "myapp/api:latest"
    
  # Use QEMU for database (needs more features)
  database:
    driver: qemu
    image: "postgres:15"
    driver_opts:
      qemu:
        snapshot: true  # Easy backup/restore
        
  # Use mock for testing components
  test-runner:
    driver: mock
    image: "test:latest"
```

### Environment-Based Driver Selection

```yaml
version: "1"

# Use environment variable for driver selection
driver: ${SPITFIRE_DRIVER:-firecracker}

driver_opts:
  firecracker:
    jailer: ${FIRECRACKER_JAILER:-true}
  qemu:
    accel: ${QEMU_ACCEL:-kvm}
  virtualbox:
    gui: ${VBOX_GUI:-false}

vms:
  app:
    image: "myapp:${APP_VERSION:-latest}"
    resources:
      memory: ${APP_MEMORY:-1GB}
      vcpu: ${APP_CPUS:-2}
```

Usage:
```bash
# Production
export SPITFIRE_DRIVER=firecracker
export FIRECRACKER_JAILER=true
spitfire up

# Development  
export SPITFIRE_DRIVER=qemu
export QEMU_ACCEL=kvm
spitfire up

# CI/CD
export SPITFIRE_DRIVER=mock
spitfire up
```

### Conditional Driver Features

```yaml
version: "1"
driver: auto  # Let spitfire choose

# Feature-based configuration
features:
  - name: gpu_support
    required: false
    drivers: [qemu, virtualbox]
  - name: nested_virt
    required: true
    drivers: [qemu]

vms:
  ml-app:
    image: "tensorflow:latest"
    features:
      - gpu_support
    driver_opts:
      qemu:
        gpu: virtio-vga
      virtualbox:
        gpu: vmsvga
```

## Error Handling and Troubleshooting

### Driver Health Diagnostics

```bash
# Check all driver status
spitfire driver status

# Detailed health check for specific driver
spitfire driver diagnose firecracker

# Test driver with minimal VM
spitfire driver test qemu --image alpine:latest
```

### Configuration Validation

```bash
# Validate driver configuration
spitfire config validate -f spitfire.yaml

# Check driver compatibility
spitfire config check-drivers -f spitfire.yaml

# Dry run to see what would happen
spitfire up --dry-run
```

### Fallback Strategies

```yaml
# Automatic fallback chain
version: "1"
driver: firecracker

fallback:
  drivers: [qemu, virtualbox, mock]
  strategy: best_available
  
# Manual fallback handling
error_handling:
  driver_unavailable:
    action: fallback
    notify: true
  driver_unhealthy:
    action: retry
    max_attempts: 3
    fallback_after: 3
```

## Performance Optimization

### Driver-Specific Optimizations

#### Firecracker Performance Tuning
```yaml
driver: firecracker
driver_opts:
  firecracker:
    # CPU optimization
    cpu_template: C3
    hyperthreading: false
    
    # Memory optimization  
    balloon: true
    balloon_stats: true
    
    # I/O optimization
    rate_limiters:
      disk:
        bandwidth: 500000000  # 500 MBps
        ops: 5000
      network:
        bandwidth: 1000000000 # 1 Gbps
        ops: 10000
        
    # Boot optimization
    kernel_args: "console=ttyS0 reboot=k panic=1 pci=off nokaslr quiet"
```

#### QEMU Performance Tuning
```yaml
driver: qemu
driver_opts:
  qemu:
    # Hardware acceleration
    accel: kvm
    cpu: host
    
    # Memory optimization
    memory_prealloc: true
    memory_backend: hugepages
    
    # I/O optimization
    cache: none
    aio: native
    
    # Network optimization
    netdev: virtio
    
    # Multi-queue support
    queues: 4
```

### Resource Monitoring

```yaml
version: "1"
driver: firecracker

monitoring:
  enabled: true
  metrics:
    - cpu_usage
    - memory_usage
    - disk_io
    - network_io
  export:
    prometheus: true
    endpoint: "http://localhost:9090"

vms:
  monitored-app:
    image: "myapp:latest"
    monitoring:
      alerts:
        - metric: cpu_usage
          threshold: 80
          action: scale_up
        - metric: memory_usage
          threshold: 90
          action: restart
```

## Security Best Practices

### Firecracker Security Configuration

```yaml
driver: firecracker
driver_opts:
  firecracker:
    # Enable all security features
    jailer: true
    jailer_uid: 1000
    jailer_gid: 1000
    chroot_base: /srv/jailer
    
    # Resource limits
    seccomp: true
    seccomp_filter: /etc/spitfire/seccomp.json
    
    # Network isolation
    network_namespace: true
    
    # Disable unnecessary features
    entropy: /dev/urandom
    logger_level: Error

vms:
  secure-service:
    image: "alpine:latest"
    user: "1000:1000"  # Non-root user
    read_only: true    # Read-only filesystem
    cap_drop:
      - ALL
    cap_add:
      - NET_BIND_SERVICE
```

### Multi-Layer Security

```yaml
version: "1"
driver: firecracker

security:
  network:
    default_deny: true
    policies:
      - from: webapp
        to: database
        ports: [5432]
      - from: webapp  
        to: external
        ports: [80, 443]
        
  storage:
    encrypt: true
    key_rotation: 30d
    
  secrets:
    backend: vault
    path: secret/spitfire/

vms:
  webapp:
    image: "myapp:latest"
    security_context:
      user: 1000
      group: 1000
      capabilities: []
    networks:
      - web-tier
      
  database:
    image: "postgres:15"
    security_context:
      user: 999
      group: 999
    networks:
      - db-tier
    volumes:
      - type: encrypted
        source: db-data
        target: /var/lib/postgresql/data
```

## Testing Strategies

### Unit Testing Driver Configurations

```bash
#!/bin/bash
# test-drivers.sh

drivers=("firecracker" "qemu" "virtualbox" "mock")

for driver in "${drivers[@]}"; do
    echo "Testing driver: $driver"
    
    # Test basic functionality
    spitfire driver test "$driver" \
        --image alpine:latest \
        --timeout 60s \
        --cleanup
        
    if [ $? -eq 0 ]; then
        echo "✓ $driver: PASS"
    else
        echo "✗ $driver: FAIL"
    fi
done
```

### Integration Testing

```yaml
# test-suite.yaml
version: "1"
driver: ${TEST_DRIVER:-mock}

test_scenarios:
  - name: basic_lifecycle
    vms:
      test-vm:
        image: "alpine:latest"
        command: ["echo", "hello world"]
    assertions:
      - vm_starts: true
      - vm_stops: true
      - exit_code: 0
      
  - name: networking
    vms:
      client:
        image: "alpine:latest"
        command: ["wget", "-O-", "http://server:8080"]
      server:
        image: "nginx:alpine"
        ports: ["8080:80"]
    assertions:
      - connectivity: true
      - response_code: 200
      
  - name: persistence
    vms:
      data-vm:
        image: "alpine:latest"
        command: ["sh", "-c", "echo data > /data/file.txt"]
        volumes:
          - test-vol:/data
    assertions:
      - file_exists: /data/file.txt
      - file_content: "data"
```

## Migration Patterns

### Driver Migration

Migrate between drivers while preserving state:

```bash
# Export current state
spitfire export --driver firecracker --output state.tar.gz

# Change driver configuration
sed -i 's/driver: firecracker/driver: qemu/' spitfire.yaml

# Import state to new driver
spitfire import --driver qemu --input state.tar.gz

# Verify migration
spitfire up
spitfire status
```

### Cross-Platform Migration

```yaml
# Source: Linux with Firecracker
source:
  driver: firecracker
  platform: linux
  
# Target: macOS with QEMU
target:
  driver: qemu
  platform: darwin
  
migration:
  preserve:
    - volumes
    - networks
    - env_vars
  convert:
    - filesystem_format
    - network_config
    - volume_drivers
```

## Contributing Examples

### Custom Driver Template

```go
// pkg/driver/mydriver/mydriver.go
package mydriver

import (
    "context"
    "github.com/thi-startup/spitfire/pkg/driver"
)

// Use the development guide template
func NewMyDriver(config *driver.Config) (driver.Driver, error) {
    // Implementation following the development guide
    return &MyDriver{config: config}, nil
}

func status() driver.State {
    // Health check implementation
    return driver.State{
        Installed: true,
        Healthy:   true,
        Running:   true,
    }
}

func init() {
    driver.Register(driver.DriverDef{
        Name:        "mydriver",
        Create:      NewMyDriver,
        Status:      status,
        Priority:    driver.Default,
        Default:     true,
        Description: "My custom driver",
    })
}
```

### Testing Your Driver

```go
// pkg/driver/mydriver/mydriver_test.go
package mydriver

import (
    "testing"
    "github.com/thi-startup/spitfire/pkg/driver"
)

func TestMyDriver(t *testing.T) {
    config := &driver.Config{
        Name: "test-vm",
        Memory: 512,
        CPUs: 1,
    }
    
    d, err := NewMyDriver(config)
    if err != nil {
        t.Fatalf("Failed to create driver: %v", err)
    }
    
    // Test driver interface methods
    // Follow the patterns in mock/mock_test.go
}
```

## See Also

- [Driver System Overview](./spitfire-driver-system.md)
- [Driver API Reference](./spitfire-driver-api.md)
- [Driver Development Guide](./spitfire-driver-development.md)
- [Driver Registry Documentation](./spitfire-driver-registry.md)
- [Mock Driver Implementation](../pkg/driver/mock/mock.go)