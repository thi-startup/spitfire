# Spitfire Configuration Examples

This document provides practical examples of Spitfire configuration files for common use cases and deployment scenarios.

## Basic Examples

### Minimal Configuration

The simplest possible Spitfire configuration:

```yaml
version: "1"

vms:
  app:
    image: "alpine:latest"
```

This creates a single VM named "app" using the Alpine Linux image with all default settings.

### Single Service with Firecracker

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

## Multi-Service Applications

### Web Application Stack

```yaml
version: "1"
driver: "firecracker"

globals:
  kernel: "/opt/firecracker/vmlinux"
  resources:
    vcpu: 1
    memory: "512MB"
  networks:
    - app-network

networks:
  app-network:
    driver: bridge
    ipam:
      config:
        - subnet: "172.20.0.0/16"

volumes:
  postgres-data:
    driver: local
    size: "5GB"
    persist: true
  redis-data:
    driver: local
    size: "1GB"

vms:
  web:
    image: "nginx:alpine"
    resources:
      memory: "256MB"
    volumes:
      - type: bind
        source: "./nginx.conf"
        target: "/etc/nginx/nginx.conf"
    depends_on:
      - api
    env:
      UPSTREAM_HOST: "api"

  api:
    image: "myapp:latest"
    resources:
      vcpu: 2
      memory: "1GB"
    depends_on:
      - database
      - cache
    env:
      DATABASE_URL: "postgres://user:pass@database:5432/app"
      REDIS_URL: "redis://cache:6379"
      PORT: "8080"

  database:
    image: "postgres:13-alpine"
    resources:
      memory: "1GB"
    volumes:
      - source: postgres-data
        target: /var/lib/postgresql/data
    env:
      POSTGRES_DB: "app"
      POSTGRES_USER: "user"
      POSTGRES_PASSWORD: "password"

  cache:
    image: "redis:alpine"
    resources:
      memory: "128MB"
    volumes:
      - source: redis-data
        target: /data
```

### Microservices Architecture

```yaml
version: "1"
driver: "firecracker"

driver_opts:
  firecracker:
    jailer: true
    cpu_template: "C3"

globals:
  kernel: "/opt/firecracker/vmlinux"
  kernel_args: "console=ttyS0 quiet"
  resources:
    vcpu: 1
    memory: "512MB"
  restart: "always"
  env:
    LOG_LEVEL: "info"
    ENVIRONMENT: "production"

networks:
  frontend:
    driver: bridge
  backend:
    driver: bridge
    ipam:
      config:
        - subnet: "10.0.1.0/24"
  database:
    driver: bridge
    ipam:
      config:
        - subnet: "10.0.2.0/24"

volumes:
  user-data:
    driver: local
    size: "10GB"
    persist: true
  order-data:
    driver: local
    size: "20GB"
    persist: true
  shared-logs:
    driver: local
    size: "5GB"

vms:
  gateway:
    image: "nginx:alpine"
    resources:
      memory: "256MB"
    networks:
      - frontend
      - backend
    volumes:
      - type: bind
        source: "./nginx-gateway.conf"
        target: "/etc/nginx/nginx.conf"
      - source: shared-logs
        target: /var/log/nginx
    depends_on:
      - user-service
      - order-service

  user-service:
    image: "mycompany/user-service:v1.2.0"
    resources:
      vcpu: 2
      memory: "1GB"
    networks:
      - backend
      - database
    volumes:
      - source: shared-logs
        target: /app/logs
    depends_on:
      - user-db
    env:
      SERVICE_NAME: "user-service"
      DB_HOST: "user-db"
      DB_NAME: "users"

  order-service:
    image: "mycompany/order-service:v2.1.0"
    resources:
      vcpu: 2
      memory: "1.5GB"
    networks:
      - backend
      - database
    volumes:
      - source: shared-logs
        target: /app/logs
    depends_on:
      - order-db
      - message-queue
    env:
      SERVICE_NAME: "order-service"
      DB_HOST: "order-db"
      QUEUE_HOST: "message-queue"

  user-db:
    image: "postgres:13-alpine"
    resources:
      memory: "1GB"
    networks:
      - database
    volumes:
      - source: user-data
        target: /var/lib/postgresql/data
    env:
      POSTGRES_DB: "users"
      POSTGRES_USER: "app"
      POSTGRES_PASSWORD: "secure-password"

  order-db:
    image: "postgres:13-alpine"
    resources:
      memory: "2GB"
    networks:
      - database
    volumes:
      - source: order-data
        target: /var/lib/postgresql/data
    env:
      POSTGRES_DB: "orders"
      POSTGRES_USER: "app"
      POSTGRES_PASSWORD: "secure-password"

  message-queue:
    image: "rabbitmq:management-alpine"
    resources:
      memory: "512MB"
    networks:
      - backend
    env:
      RABBITMQ_DEFAULT_USER: "app"
      RABBITMQ_DEFAULT_PASS: "queue-password"
```

## Multi-Driver Environments

### Development vs Production

```yaml
version: "1"
driver: "firecracker"  # Production default

driver_opts:
  firecracker:
    jailer: true
    cpu_template: "C3"
  qemu:
    accel: "kvm"
    display: "vnc=:0"

globals:
  resources:
    vcpu: 1
    memory: "512MB"

vms:
  # Production-like service (uses firecracker)
  api:
    image: "myapp:production"
    resources:
      vcpu: 2
      memory: "1GB"
    env:
      NODE_ENV: "production"

  # Development service (uses qemu for debugging)
  dev-api:
    image: "myapp:development"
    driver: "qemu"
    driver_opts:
      display: "vnc=:1"  # Enable VNC for debugging
      gdb: "tcp::1234"   # Enable GDB port
    resources:
      vcpu: 1
      memory: "512MB"
    env:
      NODE_ENV: "development"
      DEBUG: "true"

  # Testing service (auto-selects best driver)
  test-runner:
    image: "myapp:test"
    driver: ""  # Auto-select
    env:
      NODE_ENV: "test"
```

### Cross-Platform Development

```yaml
version: "1"

# Different drivers for different team members
driver_opts:
  firecracker:
    jailer: false  # Easier for development
  qemu:
    accel: "tcg"   # Works without KVM
  virtualbox:
    gui: false

globals:
  resources:
    vcpu: 2
    memory: "1GB"

vms:
  # Linux developers use firecracker
  linux-dev:
    image: "dev-env:latest"
    driver: "firecracker"
    
  # macOS developers use qemu
  macos-dev:
    image: "dev-env:latest"
    driver: "qemu"
    
  # Windows developers use VirtualBox
  windows-dev:
    image: "dev-env:latest"
    driver: "virtualbox"
```

## Advanced Configuration Patterns

### Environment-Specific Overrides

```yaml
version: "1"
driver: "firecracker"

globals:
  kernel: "/opt/firecracker/vmlinux"
  env:
    LOG_LEVEL: "${LOG_LEVEL:-info}"
    DATABASE_URL: "${DATABASE_URL}"
    REDIS_URL: "${REDIS_URL:-redis://localhost:6379}"

vms:
  app:
    image: "myapp:${APP_VERSION:-latest}"
    resources:
      vcpu: "${APP_CPUS:-2}"
      memory: "${APP_MEMORY:-1GB}"
    env:
      PORT: "${APP_PORT:-8080}"
      WORKERS: "${APP_WORKERS:-4}"
```

Usage with environment variables:
```bash
export APP_VERSION=v2.1.0
export APP_MEMORY=2GB
export DATABASE_URL=postgres://prod-db:5432/app
spitfire up
```

### Resource Scaling Patterns

```yaml
version: "1"

globals:
  kernel: "/opt/firecracker/vmlinux"

vms:
  # Small service
  cache:
    image: "redis:alpine"
    resources:
      vcpu: 1
      memory: "128MB"

  # Medium service  
  api:
    image: "api:latest"
    resources:
      vcpu: 2
      memory: "1GB"

  # Large service
  worker:
    image: "worker:latest"
    resources:
      vcpu: 4
      memory: "4GB"

  # Extra large service
  ml-processor:
    image: "ml-app:latest"
    resources:
      vcpu: 8
      memory: "16GB"
```

### Security-Focused Configuration

```yaml
version: "1"
driver: "firecracker"

driver_opts:
  firecracker:
    jailer: true
    cpu_template: "C3"
    chroot_base_dir: "/srv/spitfire"
    uid: 1000
    gid: 1000

globals:
  kernel: "/opt/secure/vmlinux"
  kernel_args: "console=ttyS0 quiet nokaslr noapic"
  resources:
    vcpu: 1
    memory: "512MB"

networks:
  isolated:
    driver: bridge
    options:
      com.docker.network.bridge.enable_icc: "false"
  dmz:
    driver: bridge
    ipam:
      config:
        - subnet: "192.168.100.0/24"

vms:
  # Public-facing service (DMZ)
  web:
    image: "nginx:alpine"
    networks:
      - dmz
    resources:
      memory: "256MB"
    # Read-only root filesystem
    driver_opts:
      read_only_root: true

  # Internal service (isolated)
  api:
    image: "api:hardened"
    networks:
      - isolated
    resources:
      memory: "512MB"
    # Additional security options
    driver_opts:
      seccomp_profile: "/etc/spitfire/seccomp/api.json"
      apparmor_profile: "spitfire-api"

  # Database (most isolated)
  database:
    image: "postgres:13-alpine"
    networks:
      - isolated
    resources:
      vcpu: 2
      memory: "2GB"
    volumes:
      - type: volume
        source: secure-db-data
        target: /var/lib/postgresql/data
    # Maximum isolation
    driver_opts:
      no_new_privileges: true
      drop_capabilities: "ALL"

volumes:
  secure-db-data:
    driver: local
    persist: true
    options:
      encryption: "aes256"
```

### Development Workflow

```yaml
version: "1"
driver: "qemu"  # Better for development debugging

driver_opts:
  qemu:
    accel: "kvm"
    display: "vnc=:0"

globals:
  resources:
    vcpu: 1
    memory: "512MB"
  env:
    NODE_ENV: "development"
  restart: "unless-stopped"

networks:
  dev-net:
    driver: bridge

volumes:
  node-modules:
    driver: local
    size: "2GB"
  dev-logs:
    driver: local

vms:
  # Main application with hot reloading
  app:
    image: "node:16-alpine"
    resources:
      vcpu: 2
      memory: "1GB"
    networks:
      - dev-net
    volumes:
      - type: bind
        source: "./src"
        target: /app/src
      - type: bind
        source: "./package.json"
        target: /app/package.json
      - source: node-modules
        target: /app/node_modules
      - source: dev-logs
        target: /app/logs
    env:
      CHOKIDAR_USEPOLLING: "true"  # For file watching in VM
      PORT: "3000"
    depends_on:
      - database
      - redis

  # Development database with seed data
  database:
    image: "postgres:13-alpine"
    networks:
      - dev-net
    volumes:
      - type: bind
        source: "./db/init"
        target: /docker-entrypoint-initdb.d
    env:
      POSTGRES_DB: "dev_app"
      POSTGRES_USER: "dev"
      POSTGRES_PASSWORD: "dev"

  # Development cache
  redis:
    image: "redis:alpine"
    networks:
      - dev-net
    command: "redis-server --appendonly yes"

  # Testing environment
  test:
    image: "node:16-alpine"
    networks:
      - dev-net
    volumes:
      - type: bind
        source: "./src"
        target: /app/src
      - type: bind
        source: "./tests"
        target: /app/tests
      - source: node-modules
        target: /app/node_modules
    env:
      NODE_ENV: "test"
    depends_on:
      - test-db

  test-db:
    image: "postgres:13-alpine"
    networks:
      - dev-net
    env:
      POSTGRES_DB: "test_app"
      POSTGRES_USER: "test"
      POSTGRES_PASSWORD: "test"
```

## Configuration Validation Examples

### Valid Configurations

```yaml
# Minimal valid config
version: "1"
vms:
  app:
    image: "alpine:latest"
```

```yaml
# Complex valid config
version: "1"
driver: "firecracker"
driver_opts:
  firecracker:
    jailer: true

globals:
  resources:
    vcpu: 2
    memory: "1GB"

networks:
  app-net:
    driver: bridge

volumes:
  data:
    driver: local

vms:
  web:
    image: "nginx:alpine"
    driver: "qemu"  # Override global
    resources:
      memory: "512MB"  # Override global
    networks:
      - app-net
    volumes:
      - source: data
        target: /data
```

### Common Configuration Errors

```yaml
# ERROR: Missing version
vms:
  app:
    image: "alpine:latest"
```

```yaml
# ERROR: No VMs defined
version: "1"
networks:
  test: {}
```

```yaml
# ERROR: Both image and rootfs specified
version: "1"
vms:
  app:
    image: "alpine:latest"
    rootfs: "/path/to/rootfs.ext4"
```

```yaml
# ERROR: Invalid driver name
version: "1"
driver: "Invalid-Driver!"
vms:
  app:
    image: "alpine:latest"
```

```yaml
# ERROR: Undefined network reference
version: "1"
vms:
  app:
    image: "alpine:latest"
    networks:
      - undefined-network
```

```yaml
# ERROR: Invalid restart policy
version: "1"
vms:
  app:
    image: "alpine:latest"
    restart: "invalid-policy"
```

```yaml
# ERROR: Invalid memory format
version: "1"
vms:
  app:
    image: "alpine:latest"
    resources:
      memory: "512KB"  # Unsupported unit
```

## Best Practices

### Naming Conventions

```yaml
version: "1"

# Use descriptive names
vms:
  user-api:          # Not: api1
    image: "user-service:v1.2.0"
  
  order-processor:   # Not: processor
    image: "order-service:v2.0.0"
  
  postgres-primary:  # Not: db
    image: "postgres:13"

# Use consistent network naming
networks:
  frontend-net:
    driver: bridge
  backend-net:
    driver: bridge
  database-net:
    driver: bridge

# Use descriptive volume names
volumes:
  postgres-data:
    driver: local
    persist: true
  redis-cache:
    driver: tmpfs
    size: "1GB"
```

### Resource Planning

```yaml
version: "1"

# Define resource tiers
globals:
  resources:
    vcpu: 1
    memory: "512MB"  # Default tier

vms:
  # Lightweight services
  redis:
    image: "redis:alpine"
    resources:
      memory: "128MB"
  
  # Standard services
  api:
    image: "api:latest"
    resources:
      vcpu: 2
      memory: "1GB"
  
  # Resource-intensive services
  database:
    image: "postgres:13"
    resources:
      vcpu: 4
      memory: "4GB"
```

### Environment Management

```yaml
version: "1"

# Use environment variables for configuration
globals:
  env:
    APP_ENV: "${ENVIRONMENT:-development}"
    LOG_LEVEL: "${LOG_LEVEL:-info}"
    DATABASE_URL: "${DATABASE_URL}"

vms:
  app:
    image: "myapp:${VERSION:-latest}"
    resources:
      memory: "${APP_MEMORY:-1GB}"
    env:
      FEATURE_FLAGS: "${FEATURE_FLAGS:-}"
```

These examples demonstrate various patterns and use cases for Spitfire configurations, from simple single-service setups to complex multi-tier applications with advanced networking and security requirements.