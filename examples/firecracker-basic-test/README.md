# Firecracker Basic Test

This example demonstrates a minimal Firecracker VM configuration for testing basic functionality.

## Configuration

- **Driver**: Firecracker
- **VM**: Single VM named "web"
- **Resources**: 1 vCPU, 512MB RAM
- **Image**: Ubuntu rootfs (`/opt/spitfire/ubuntu.ext4`)
- **Kernel**: Custom kernel (`/opt/spitfire/hello-vmlinux.bin`)

## Usage

```bash
cd examples/firecracker-basic-test

# Start the VM
spitfire vm up

# Check VM status
spitfire vm ps

# Stop the VM
spitfire vm down
```

## Prerequisites

- Firecracker driver must be set up: `spitfire driver setup firecracker`
- Required files must exist:
  - `/opt/spitfire/ubuntu.ext4` - Ubuntu rootfs image
  - `/opt/spitfire/hello-vmlinux.bin` - Linux kernel binary

## State Management

VM state is stored in `~/.spitfire/projects/firecracker-basic-test/` and persists across restarts.