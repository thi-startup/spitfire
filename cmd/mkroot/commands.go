package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/docker/docker/daemon/graphdriver/copy"
)

const (
	mkfsPath      = "/sbin/mkfs"
	fallocatePath = "/usr/bin/fallocate"
	mountPath     = "/usr/bin/mount"
	umountPath    = "/usr/bin/umount"
)

type mkfs struct {
	fscmd     string
	size      string
	name      string
	mountArgs string
	from      string
}

func (m *mkfs) Size(s string) *mkfs {
	m.size = s
	return m
}

// specify file to copy from
func (m *mkfs) DirPath(f string) *mkfs {
	m.from = f
	return m
}

func (m *mkfs) Type(t string) *mkfs {
	m.fscmd = strings.Join([]string{m.fscmd, t}, ".")
	return m
}

func (m *mkfs) MountFlags(s string) *mkfs {
	return m
}

func (m *mkfs) Name(name string) *mkfs {
	m.name = name
	return m
}

func mktemp() (string, func() error, error) {
	name, err := os.MkdirTemp(".", "tmpdir")
	if err != nil {
		return "", nil, err
	}

	rm := func() error {
		if err := os.RemoveAll(name); err != nil {
			return err
		}
		return nil
	}

	return name, rm, nil
}

func (m *mkfs) Execute() error {
	// fallocate -l 10M name
	fallocate := exec.Command(fallocatePath, "-l", m.size, m.name)
	if err := fallocate.Run(); err != nil {
		return nil
	}

	// mkfs.ext2 name
	if !Exists(m.fscmd) {
		return fmt.Errorf("could not find '%s'", m.fscmd)
	}
	mkfs := exec.Command(m.fscmd, m.name)
	if err := mkfs.Run(); err != nil {
		return fmt.Errorf("failed to create filesystem: %v", err)
	}

	// if there is a file to copy from
	if Exists(m.from) {
		temp, rmDir, err := mktemp()
		if err != nil {
			return fmt.Errorf("failed to make temporary directory: %v", err)
		}

		defer func() {
			if err := rmDir(); err != nil {
				log.Fatalf("failed to remove temp dir: '%s': %v", temp, err)
			}
		}()

		mount := exec.Command(mountPath, "-o", m.mountArgs, m.name, temp)
		if err := mount.Run(); err != nil {
			return fmt.Errorf("failed to mount '%s': %v", m.name, err)
		}

		if err := copy.DirCopy(m.from, temp, copy.Content, true); err != nil {
			return fmt.Errorf("failed to copy from '%s' to '%s': %v", m.from, temp, err)
		}

		umount := exec.Command(umountPath, temp)
		if err := umount.Run(); err != nil {
			return fmt.Errorf("failed to unmount '%s': %v", temp, err)
		}
	}

	return nil
}

func Exists(file string) bool {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	}
	return true
}

func Mkfs() (*mkfs, error) {
	if !Exists(fallocatePath) {
		return nil, fmt.Errorf("could not find '%s'", fallocatePath)
	}
	if !Exists(mountPath) {
		return nil, fmt.Errorf("could not find '%s'", mountPath)
	}
	return &mkfs{
		fscmd:     mkfsPath,
		mountArgs: "loop,noatime",
	}, nil
}
