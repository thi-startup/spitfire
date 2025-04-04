#!/bin/sh

KERNEL_PATH="/boot/vmlinuz-linux"
INITRD_PATH="/boot/initrd.img-$(uname -r)"

qemu-system-x86_64 \
    -nographic \
    -kernel "$KERNEL_PATH" \
    -initrd "$INITRD_PATH" \
    -append "console=ttyS0 root=/dev/sda rw" \
    -drive file="$ROOT_DRIVE",format=raw,index=0 \
    -m 256M \
    -smp 1
