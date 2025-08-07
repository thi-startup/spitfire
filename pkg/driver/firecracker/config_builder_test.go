package firecracker

import (
	"testing"

	"github.com/thi-startup/spitfire/pkg/driver"
)

func TestConfigBuilder_buildDrives_ImageToRootfs(t *testing.T) {
	tests := []struct {
		name     string
		config   *driver.Config
		wantRoot bool
		rootPath string
	}{
		{
			name: "Rootfs field takes precedence",
			config: &driver.Config{
				Name:   "test",
				Rootfs: "/path/to/rootfs.ext4",
				Image:  "/path/to/image.ext4",
			},
			wantRoot: true,
			rootPath: "/path/to/rootfs.ext4",
		},
		{
			name: "Image field used when Rootfs is empty",
			config: &driver.Config{
				Name:   "test",
				Rootfs: "",
				Image:  "/path/to/image.ext4",
			},
			wantRoot: true,
			rootPath: "/path/to/image.ext4",
		},
		{
			name: "No root device when both are empty",
			config: &driver.Config{
				Name:   "test",
				Rootfs: "",
				Image:  "",
			},
			wantRoot: false,
		},
		{
			name: "Only Rootfs provided",
			config: &driver.Config{
				Name:   "test",
				Rootfs: "/path/to/rootfs.ext4",
				Image:  "",
			},
			wantRoot: true,
			rootPath: "/path/to/rootfs.ext4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := &ConfigBuilder{
				driverConfig: tt.config,
			}

			drives := builder.buildDrives()

			// Check if root device exists
			var rootDrive *Drive
			for _, drive := range drives {
				if drive.IsRootDevice {
					rootDrive = &drive
					break
				}
			}

			if tt.wantRoot {
				if rootDrive == nil {
					t.Errorf("Expected root device to be created, but none found")
					return
				}
				if rootDrive.PathOnHost != tt.rootPath {
					t.Errorf("Expected root device path %s, got %s", tt.rootPath, rootDrive.PathOnHost)
				}
				if rootDrive.DriveID != "root" {
					t.Errorf("Expected root device ID 'root', got %s", rootDrive.DriveID)
				}
			} else {
				if rootDrive != nil {
					t.Errorf("Expected no root device, but found one with path %s", rootDrive.PathOnHost)
				}
			}
		})
	}
}

func TestConfigBuilder_buildDrives_RegressionTest(t *testing.T) {
	// Regression test for the specific issue we fixed:
	// When config has `image` but no `rootfs`, should create root device
	config := &driver.Config{
		Name:   "web",
		Image:  "/opt/spitfire/ubuntu.ext4",
		Rootfs: "", // This was the problem - empty rootfs with non-empty image
		Memory: 512,
		CPUs:   1,
	}

	builder := &ConfigBuilder{
		driverConfig: config,
	}

	drives := builder.buildDrives()

	// Must have at least one drive
	if len(drives) == 0 {
		t.Fatal("Expected at least one drive to be created")
	}

	// Must have exactly one root device
	rootDeviceCount := 0
	var rootDrive *Drive
	for _, drive := range drives {
		if drive.IsRootDevice {
			rootDeviceCount++
			rootDrive = &drive
		}
	}

	if rootDeviceCount != 1 {
		t.Fatalf("Expected exactly 1 root device, got %d", rootDeviceCount)
	}

	// Root device should use the image path
	expectedPath := "/opt/spitfire/ubuntu.ext4"
	if rootDrive.PathOnHost != expectedPath {
		t.Errorf("Expected root device path %s, got %s", expectedPath, rootDrive.PathOnHost)
	}

	// Validate that we have at least one root device (which is what the original error checked)
	hasRootDevice := false
	for _, drive := range drives {
		if drive.IsRootDevice {
			hasRootDevice = true
			break
		}
	}

	if !hasRootDevice {
		t.Error("Expected at least one root device to be created - this would cause 'at least one root device is required' error")
	}
}