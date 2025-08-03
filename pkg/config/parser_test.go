package config

import (
	"os"
	"strings"
	"testing"
)

func TestExpandEnvironmentVariables(t *testing.T) {
	// Set up test environment variables
	os.Setenv("TEST_VAR", "test_value")
	os.Setenv("PORT", "8080")
	defer os.Unsetenv("TEST_VAR")
	defer os.Unsetenv("PORT")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple variable substitution",
			input:    "image: nginx:${TEST_VAR}",
			expected: "image: nginx:test_value",
		},
		{
			name:     "multiple variables",
			input:    "env:\n  HOST: ${TEST_VAR}\n  PORT: ${PORT}",
			expected: "env:\n  HOST: test_value\n  PORT: 8080",
		},
		{
			name:     "short syntax variable",
			input:    "image: nginx:$TEST_VAR",
			expected: "image: nginx:test_value",
		},
		{
			name:     "nonexistent variable",
			input:    "image: nginx:${NONEXISTENT}",
			expected: "image: nginx:${NONEXISTENT}",
		},
		{
			name:     "no variables",
			input:    "image: nginx:latest",
			expected: "image: nginx:latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandEnvironmentVariables(tt.input)
			if result != tt.expected {
				t.Errorf("ExpandEnvironmentVariables() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid minimal config",
			config: Config{
				Version: "1",
				VMs: map[string]*VM{
					"test": {
						Image: "nginx:latest",
					},
				},
			},
			expectError: false,
		},
		{
			name: "missing version",
			config: Config{
				VMs: map[string]*VM{
					"test": {
						Image: "nginx:latest",
					},
				},
			},
			expectError: true,
			errorMsg:    "version is required",
		},
		{
			name: "no VMs defined",
			config: Config{
				Version: "1",
				VMs:     map[string]*VM{},
			},
			expectError: true,
			errorMsg:    "at least one VM must be defined",
		},
		{
			name: "VM with neither image nor rootfs",
			config: Config{
				Version: "1",
				VMs: map[string]*VM{
					"test": {},
				},
			},
			expectError: true,
			errorMsg:    "either 'image' or 'rootfs' must be specified",
		},
		{
			name: "VM with both image and rootfs",
			config: Config{
				Version: "1",
				VMs: map[string]*VM{
					"test": {
						Image:  "nginx:latest",
						Rootfs: "/path/to/rootfs",
					},
				},
			},
			expectError: true,
			errorMsg:    "cannot specify both 'image' and 'rootfs'",
		},
		{
			name: "invalid restart policy",
			config: Config{
				Version: "1",
				VMs: map[string]*VM{
					"test": {
						Image:   "nginx:latest",
						Restart: "invalid",
					},
				},
			},
			expectError: true,
			errorMsg:    "invalid restart policy",
		},
		{
			name: "undefined network reference",
			config: Config{
				Version: "1",
				VMs: map[string]*VM{
					"test": {
						Image:    "nginx:latest",
						Networks: []string{"undefined"},
					},
				},
			},
			expectError: true,
			errorMsg:    "references undefined network",
		},
		{
			name: "undefined volume reference",
			config: Config{
				Version: "1",
				VMs: map[string]*VM{
					"test": {
						Image: "nginx:latest",
						Volumes: []VolumeMount{
							{
								Type:   "volume",
								Source: "undefined",
								Target: "/data",
							},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "references undefined volume",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateConfig()
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestVolumeMount_Validate(t *testing.T) {
	tests := []struct {
		name        string
		mount       VolumeMount
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid volume mount",
			mount: VolumeMount{
				Source: "data",
				Target: "/app/data",
			},
			expectError: false,
		},
		{
			name: "empty source",
			mount: VolumeMount{
				Target: "/app/data",
			},
			expectError: true,
			errorMsg:    "source cannot be empty",
		},
		{
			name: "empty target",
			mount: VolumeMount{
				Source: "data",
			},
			expectError: true,
			errorMsg:    "target cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.mount.Validate()
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestParseMemorySize(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name:     "valid MB format",
			input:    "512MB",
			expected: "512MB",
		},
		{
			name:     "valid GB format",
			input:    "1GB",
			expected: "1GB",
		},
		{
			name:     "valid M format",
			input:    "512M",
			expected: "512M",
		},
		{
			name:     "valid G format",
			input:    "2G",
			expected: "2G",
		},
		{
			name:     "lowercase input",
			input:    "512mb",
			expected: "512MB",
		},
		{
			name:        "empty string",
			input:       "",
			expectError: true,
		},
		{
			name:        "invalid format",
			input:       "512",
			expectError: true,
		},
		{
			name:        "invalid unit",
			input:       "512KB",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseMemorySize(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("ParseMemorySize() = %q, want %q", result, tt.expected)
				}
			}
		})
	}
}

func TestApplyGlobalDefaults(t *testing.T) {
	config := Config{
		Globals: &GlobalConfig{
			Kernel:     "/path/to/kernel",
			KernelArgs: "console=ttyS0",
			Resources: &Resources{
				VCPU:   2,
				Memory: "1GB",
			},
			Networks: []string{"default"},
			Restart:  "always",
			Env: map[string]string{
				"GLOBAL_VAR": "global_value",
			},
		},
		VMs: map[string]*VM{
			"vm1": {
				Image: "nginx:latest",
				// Will inherit all global settings
			},
			"vm2": {
				Image:    "redis:latest",
				Kernel:   "/custom/kernel",   // Should override global
				Networks: []string{"custom"}, // Should merge with global
				Env: map[string]string{
					"GLOBAL_VAR": "overridden", // Should take precedence
					"VM_VAR":     "vm_value",
				},
			},
		},
	}

	err := config.applyGlobalDefaults()
	if err != nil {
		t.Fatalf("applyGlobalDefaults() failed: %v", err)
	}

	// Check vm1 inherited all global settings
	vm1 := config.VMs["vm1"]
	if vm1.Kernel != "/path/to/kernel" {
		t.Errorf("vm1.Kernel = %q, want %q", vm1.Kernel, "/path/to/kernel")
	}
	if vm1.KernelArgs != "console=ttyS0" {
		t.Errorf("vm1.KernelArgs = %q, want %q", vm1.KernelArgs, "console=ttyS0")
	}
	if vm1.Resources.VCPU != 2 {
		t.Errorf("vm1.Resources.VCPU = %d, want %d", vm1.Resources.VCPU, 2)
	}
	if vm1.Restart != "always" {
		t.Errorf("vm1.Restart = %q, want %q", vm1.Restart, "always")
	}
	if vm1.Env["GLOBAL_VAR"] != "global_value" {
		t.Errorf("vm1.Env[GLOBAL_VAR] = %q, want %q", vm1.Env["GLOBAL_VAR"], "global_value")
	}

	// Check vm2 overrides work correctly
	vm2 := config.VMs["vm2"]
	if vm2.Kernel != "/custom/kernel" {
		t.Errorf("vm2.Kernel = %q, want %q", vm2.Kernel, "/custom/kernel")
	}
	if vm2.Env["GLOBAL_VAR"] != "overridden" {
		t.Errorf("vm2.Env[GLOBAL_VAR] = %q, want %q", vm2.Env["GLOBAL_VAR"], "overridden")
	}
	if vm2.Env["VM_VAR"] != "vm_value" {
		t.Errorf("vm2.Env[VM_VAR] = %q, want %q", vm2.Env["VM_VAR"], "vm_value")
	}

	// Check network merging (should contain both default and custom)
	expectedNetworks := map[string]bool{"default": true, "custom": true}
	actualNetworks := make(map[string]bool)
	for _, network := range vm2.Networks {
		actualNetworks[network] = true
	}
	for expected := range expectedNetworks {
		if !actualNetworks[expected] {
			t.Errorf("vm2.Networks missing expected network %q", expected)
		}
	}
}
