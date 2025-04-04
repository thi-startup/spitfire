#!/bin/bash
# scripts/test-init.sh
set -ex

# Configuration
BUILD_DIR="./bin"
INIT_BINARY="$BUILD_DIR/spitfire-init"
INIT_DRIVE="./tmp/init.ext4"
ROOT_DRIVE="./tmp/rootfs.ext4"
DATA_DRIVE="./tmp/data.ext4"
MOUNT_DIR="./tmp/mnt"
FIRECRACKER_PATH="${FIRECRACKER_PATH:-/usr/bin/firecracker}"
KERNEL_PATH="${KERNEL_PATH:-/home/joe/.spitfire/kernels/6.1-x86_64/vmlinux}"

# Create directories
mkdir -p "$BUILD_DIR" "./tmp" "$MOUNT_DIR"

# Create the init drive
# echo "Creating init drive..."
# fallocate -l 20M "$INIT_DRIVE"
# mkfs.ext4 "$INIT_DRIVE"
# mount -o loop,noatime "$INIT_DRIVE" "$MOUNT_DIR"
# mkdir -p "$MOUNT_DIR/dev" "$MOUNT_DIR/proc" "$MOUNT_DIR/sys" "$MOUNT_DIR/newroot"
# umount "$MOUNT_DIR"

# Create a simple rootfs if it doesn't exist
if [ ! -f "$ROOT_DRIVE" ]; then
    echo "Creating root drive from Alpine Linux container..."
    fallocate -l 200M "$ROOT_DRIVE"
    mkfs.ext4 "$ROOT_DRIVE"
    mount -o loop,noatime "$ROOT_DRIVE" "$MOUNT_DIR"
    
    # Use Docker to extract a minimal Alpine filesystem
    docker run --rm -v "$MOUNT_DIR:/mnt" alpine:latest sh -c "tar -C / -c bin etc lib root sbin usr var | tar -C /mnt -xf -"
    
    mkdir -p "$MOUNT_DIR/spitfire"
    cp "$INIT_BINARY" "$MOUNT_DIR/spitfire/spitfire-init"
    cat > "$MOUNT_DIR/spitfire/run.json" <<EOF
{
    "RootDevice": "/dev/vda",
    "Hostname": "spitfire-test",
    "CmdOverride": ["/bin/sh"],
    "ImageConfig": {
        "Env": ["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],
        "WorkingDir": "/",
        "User": "root"
    },
    "ExtraEnv": ["TERM=xterm"],
    "Mounts": [
      {
        "DevicePath": "/dev/vdb",
        "MountPath": "/data"
      }
    ],
    "EtcResolv": {
      "Nameservers": ["8.8.8.8", "8.8.4.4"]
    }
}
EOF
    umount "$MOUNT_DIR"
fi

# Create a data drive if it doesn't exist
if [ ! -f "$DATA_DRIVE" ]; then
    echo "Creating data drive..."
    fallocate -l 100M "$DATA_DRIVE"
    mkfs.ext4 "$DATA_DRIVE"
    mount -o loop,noatime "$DATA_DRIVE" "$MOUNT_DIR"
    echo "This is a test file in the data volume" > "$MOUNT_DIR/test.txt"
    umount "$MOUNT_DIR"
fi

# {
#   "drive_id": "init",
#   "path_on_host": "$INIT_DRIVE",
#   "is_root_device": true,
#   "is_read_only": false
# },

# Create Firecracker config
FC_CONFIG="./tmp/firecracker.json"
cat > "$FC_CONFIG" <<EOF
{
  "boot-source": {
    "kernel_image_path": "$KERNEL_PATH",
    "boot_args": "console=ttyS0 reboot=k panic=1 pci=off init=/spitfire/spitfire-init"
  },
  "drives": [
    {
      "drive_id": "rootfs",
      "path_on_host": "$ROOT_DRIVE",
      "is_root_device": true,
      "is_read_only": false
    },
    {
      "drive_id": "data",
      "path_on_host": "$DATA_DRIVE",
      "is_root_device": false,
      "is_read_only": false
    }
  ],
  "machine-config": {
    "vcpu_count": 1,
    "mem_size_mib": 256
  }
}
EOF

# Clean up any existing Firecracker process and socket
SOCKET_PATH="./tmp/firecracker.sock"
if [ -S "$SOCKET_PATH" ]; then
    echo "Removing existing socket..."
    rm -f "$SOCKET_PATH"
fi

echo "Starting Firecracker VM..."
$FIRECRACKER_PATH --api-sock "$SOCKET_PATH" --config-file "$FC_CONFIG"
