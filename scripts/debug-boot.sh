#!/bin/bash
set -ex

# _umount_all() { umount ./tmp/mnt }
# trap '_umount_all' ERR


# Configuration
INIT_DRIVE="./tmp/test-drive.ext4"
MOUNT_DIR="./tmp/mnt"
FIRECRACKER_PATH="${FIRECRACKER_PATH:-/usr/bin/firecracker}"
KERNEL_PATH="${KERNEL_PATH:-/home/joe/.spitfire/kernels/6.1-x86_64/vmlinux}"

mkdir -p "./tmp" "$MOUNT_DIR"

echo "Creating minimal init drive..."
fallocate -l 200M "$INIT_DRIVE"
mkfs.ext4 "$INIT_DRIVE"
mount -o loop,noatime "$INIT_DRIVE" "$MOUNT_DIR"

docker run --rm -v "$MOUNT_DIR:/mnt" alpine:latest sh -c "tar -C / -c bin etc lib root sbin usr var | tar -C /mnt -xf -"

chroot $MOUNT_DIR /bin/sh <<"EOT"
export PATH=/sbin:/bin:/usr/bin
apk add openrc
apk add bash
apk add util-linux
# Set up a login terminal on the serial console (ttyS0):
ln -s agetty /etc/init.d/agetty.ttyS0
echo ttyS0 > /etc/securetty
rc-update add agetty.ttyS0 default

# Make sure special file systems are mounted on boot:
rc-update add devfs boot
rc-update add procfs boot
rc-update add sysfs boot

for dir in dev proc run sys var; do mkdir -p /$dir; done
EOT
umount "$MOUNT_DIR"

# Create Firecracker config with very basic settings and verbose output
FC_CONFIG="./tmp/firecracker-debug.json"
cat > "$FC_CONFIG" <<EOF
{
  "boot-source": {
    "kernel_image_path": "$KERNEL_PATH",
    "boot_args": "console=ttyS0 rootfstype=ext4 rw debug loglevel=7"
  },
  "drives": [
    {
      "drive_id": "rootfs",
      "path_on_host": "$INIT_DRIVE",
      "is_root_device": true,
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
SOCKET_PATH="./tmp/firecracker-debug.sock"
if [ -S "$SOCKET_PATH" ]; then
    echo "Removing existing socket..."
    rm -f "$SOCKET_PATH"
fi

if [ "$1" = "qemu" ]; then
    qemu-system-x86_64 \
        -nographic \
        -kernel "$KERNEL_PATH" \
        -append "console=ttyS0 root=/dev/vda init=/init debug loglevel=7" \
        -drive file="$INIT_DRIVE",format=raw,index=0 \
        -m 256M \
        -smp 1
else
    $FIRECRACKER_PATH --api-sock "$SOCKET_PATH" --config-file "$FC_CONFIG" --level Debug
fi
