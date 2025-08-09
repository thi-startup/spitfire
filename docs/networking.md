# Spitfire Networking Guide

This guide covers Spitfire's networking architecture, configuration, and troubleshooting.

## Overview

Spitfire implements dual-mode networking to support both secure rootless operation and full-featured root networking:

- **Rootless Mode**: Uses pasta or slirp4netns for userspace networking (no elevated privileges required)
- **Root Mode**: Uses TAP devices and bridges for advanced networking features (requires root/capabilities)

## Architecture

### Networking Modes

#### Rootless Mode (Default)
```
┌─────────────┐    ┌──────────────┐    ┌─────────────────┐
│   Host      │    │   pasta/     │    │   Firecracker   │
│  Network    │◄───│  slirp4netns │◄───│   VM Process    │
│             │    │              │    │                 │
└─────────────┘    └──────────────┘    └─────────────────┘
```

- No TAP devices created
- Networking tools attach to Firecracker's network namespace
- Port forwarding handled by pasta/slirp4netns
- Automatic when running as non-root user

#### Root Mode
```
┌─────────────┐    ┌──────────────┐    ┌─────────────────┐
│   Host      │    │  Bridge +    │    │   Firecracker   │
│  Network    │◄───│  TAP Device  │◄───│   VM (eth0)     │
│             │    │              │    │                 │
└─────────────┘    └──────────────┘    └─────────────────┘
```

- Creates TAP devices and bridges
- Full network interface in Firecracker configuration
- Advanced features: static IPs, custom bridges, macvlan
- Requires root privileges or CAP_NET_ADMIN

## Configuration

### Basic Network Configuration

```yaml
# spitfire.yaml
networks:
  default:
    driver: bridge    # bridge, tap, macvlan (root mode)
                     # pasta, slirp4netns (rootless mode)

vms:
  web-server:
    image: /opt/spitfire/ubuntu.ext4
    memory: 512MB
    cpus: 1
    networks:
      - default       # Attach to default network
    ports:
      - "8080:80"     # Forward host:8080 to VM:80
```

### Advanced Configuration

```yaml
networks:
  frontend:
    driver: bridge
    subnet: "10.0.1.0/24"
    options:
      bridge_name: "spitfire-frontend"
      
  backend:
    driver: bridge  
    subnet: "10.0.2.0/24"
    options:
      bridge_name: "spitfire-backend"

vms:
  web:
    image: /opt/spitfire/ubuntu.ext4
    networks:
      - frontend
    ports:
      - "8080:80"
      - "8443:443"
      
  database:
    image: /opt/spitfire/postgres.ext4
    networks:
      - backend
    # No external ports - internal only
```

## Backend Selection

### Automatic Selection
- **Non-root user**: Automatically uses rootless backends (pasta → slirp4netns)
- **Root user**: Uses root backends (bridge → tap → macvlan)
- **Tool availability**: Automatically detects available networking tools

### Manual Selection
```yaml
networks:
  custom:
    driver: pasta        # Force specific backend
    mode: rootless       # Force specific mode
```

## Testing and Verification

### Check VM Status
```bash
./bin/spitfire vm ps
```

### Verify Networking Processes
```bash
# Check for pasta processes
ps aux | grep pasta

# Check for slirp4netns processes  
ps aux | grep slirp4netns

# Check TAP devices (root mode)
ip link show | grep tap-
```

### Test Port Forwarding
```bash
# Start VM with port forwarding
./bin/spitfire vm up -f examples/networking-test/spitfire.yaml

# Test connectivity (if VM has web server)
curl http://localhost:8080
```

## Troubleshooting

### Common Issues

#### Permission Denied Errors
```
Error: unshare network namespace: operation not permitted
```
**Solution**: This indicates trying to create network namespaces without proper privileges. Spitfire should automatically use rootless mode to avoid this.

#### Missing Networking Tools
```
Error: pasta failed with exit code 1: command not found
```
**Solutions**:
1. Install pasta: `sudo apt-get install passt` (Ubuntu) or equivalent
2. Use slirp4netns: `sudo apt-get install slirp4netns`
3. Check available backends: `which pasta slirp4netns`

#### TAP Device Creation Failed (Root Mode)
```
Error: failed to create TAP device: operation not permitted
```
**Solutions**:
1. Run with sudo: `sudo ./bin/spitfire vm up`
2. Add CAP_NET_ADMIN capability
3. Add user to appropriate groups (varies by distribution)

#### VM Starts But No Network
**Troubleshooting Steps**:
1. Check if networking processes are running
2. Verify Firecracker process is in correct namespace
3. Check logs with `--debug` flag
4. Verify configuration syntax

### Debug Mode
```bash
# Enable detailed logging
./bin/spitfire vm up --debug -f config.yaml

# Check logs
./bin/spitfire --log-level=debug vm up -f config.yaml
```

## Implementation Details

### Key Components

- `pkg/networking/networking.go` - Main networking manager
- `pkg/networking/tap.go` - TAP device management (root mode)
- `pkg/driver/firecracker/config_builder.go` - Firecracker network config

### Network Namespace Handling
- **Rootless**: Never pre-create namespaces, let tools handle creation
- **Root**: Create persistent namespaces as needed
- **Cleanup**: Automatic cleanup on VM shutdown

### Integration Points
- Firecracker configuration adapts based on networking mode
- Driver integration through `SetupResult` interface
- State management tracks networking resources

## Examples

See `examples/networking-test/spitfire.yaml` for working configuration examples.

## Future Enhancements

- [ ] IPv6 support configuration
- [ ] Custom DNS configuration  
- [ ] Network policies and isolation
- [ ] Multi-host networking
- [ ] Performance optimization options