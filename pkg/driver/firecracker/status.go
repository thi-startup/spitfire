package firecracker

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/thi-startup/spitfire/pkg/driver"
)

// status checks if the Firecracker driver is available and healthy
func status() driver.State {
	// Check if firecracker binary exists
	firecrackerPath, err := exec.LookPath("firecracker")
	if err != nil {
		return driver.State{
			Installed: false,
			Healthy:   false,
			Error:     fmt.Errorf("firecracker binary not found in PATH"),
			Fix:       "Install Firecracker from https://github.com/firecracker-microvm/firecracker/releases",
			Doc:       "https://github.com/firecracker-microvm/firecracker/blob/main/docs/getting-started.md",
		}
	}

	// Check if binary is executable
	if err := isExecutable(firecrackerPath); err != nil {
		return driver.State{
			Installed: true,
			Healthy:   false,
			Error:     err,
			Fix:       fmt.Sprintf("Make firecracker binary executable: chmod +x %s", firecrackerPath),
		}
	}

	// Check platform support
	if runtime.GOOS != "linux" {
		return driver.State{
			Installed: true,
			Healthy:   false,
			Error:     fmt.Errorf("firecracker only supports Linux"),
			Fix:       "Use a Linux host or different driver",
			Doc:       "https://github.com/firecracker-microvm/firecracker/blob/main/docs/getting-started.md#prerequisites",
		}
	}

	// Check KVM access
	if err := checkKVMAccess(); err != nil {
		return driver.State{
			Installed: true,
			Healthy:   false,
			Error:     err,
			Fix:       "Add user to kvm group: sudo usermod -aG kvm $USER && newgrp kvm",
			Doc:       "https://www.kernel.org/doc/Documentation/virt/kvm/api.txt",
		}
	}

	// Check firecracker version
	version, err := getFirecrackerVersion(firecrackerPath)
	if err != nil {
		return driver.State{
			Installed:        true,
			Healthy:          true,
			NeedsImprovement: true,
			Error:            err,
			Fix:              "Unable to determine Firecracker version",
		}
	}

	// Check if version is too old
	if isVersionTooOld(version) {
		return driver.State{
			Installed:        true,
			Healthy:          true,
			NeedsImprovement: true,
			Fix:              fmt.Sprintf("Update Firecracker to v1.0.0 or later (current: %s)", version),
		}
	}

	return driver.State{
		Installed: true,
		Healthy:   true,
		Running:   true,
		Version:   version,
	}
}

// isExecutable checks if a file is executable
func isExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("cannot stat file %s: %w", path, err)
	}

	if info.Mode()&0111 == 0 {
		return fmt.Errorf("file %s is not executable", path)
	}

	return nil
}

// checkKVMAccess verifies that the user can access /dev/kvm
func checkKVMAccess() error {
	// Check if /dev/kvm exists
	if _, err := os.Stat("/dev/kvm"); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("/dev/kvm does not exist - KVM support not available")
		}
		return fmt.Errorf("cannot access /dev/kvm: %w", err)
	}

	// Try to open /dev/kvm for read/write
	file, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("cannot open /dev/kvm: %w", err)
	}
	file.Close()

	return nil
}

// getFirecrackerVersion gets the version of the firecracker binary
func getFirecrackerVersion(firecrackerPath string) (string, error) {
	cmd := exec.Command(firecrackerPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get firecracker version: %w", err)
	}

	// Parse version from output like "firecracker v1.4.0"
	versionStr := strings.TrimSpace(string(output))
	parts := strings.Fields(versionStr)
	if len(parts) >= 2 {
		version := strings.TrimPrefix(parts[1], "v")
		return version, nil
	}

	return versionStr, nil
}

// isVersionTooOld checks if the firecracker version is too old
func isVersionTooOld(version string) bool {
	// For now, just check if version starts with "0."
	// In a real implementation, we'd do proper semantic version comparison
	return strings.HasPrefix(version, "0.")
}
