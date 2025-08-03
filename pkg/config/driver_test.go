package config

import (
	"testing"
)

func TestDriverConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config with global driver",
			config: &Config{
				Version: "1",
				Driver:  "firecracker",
				DriverOpts: map[string]map[string]interface{}{
					"firecracker": {
						"jailer": true,
					},
				},
				VMs: map[string]*VM{
					"test": {
						Image: "alpine:latest",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with VM-specific driver",
			config: &Config{
				Version: "1",
				Driver:  "firecracker",
				VMs: map[string]*VM{
					"test": {
						Image:  "alpine:latest",
						Driver: "qemu",
						DriverOpts: map[string]interface{}{
							"accel": "kvm",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid driver name",
			config: &Config{
				Version: "1",
				Driver:  "Invalid-Driver-Name!",
				VMs: map[string]*VM{
					"test": {
						Image: "alpine:latest",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid driver in driver_opts",
			config: &Config{
				Version: "1",
				Driver:  "firecracker",
				DriverOpts: map[string]map[string]interface{}{
					"INVALID": {
						"option": "value",
					},
				},
				VMs: map[string]*VM{
					"test": {
						Image: "alpine:latest",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVMGetEffectiveDriver(t *testing.T) {
	tests := []struct {
		name         string
		vm           *VM
		globalDriver string
		want         string
	}{
		{
			name:         "VM driver overrides global",
			vm:           &VM{Driver: "qemu"},
			globalDriver: "firecracker",
			want:         "qemu",
		},
		{
			name:         "Global driver used when VM has none",
			vm:           &VM{},
			globalDriver: "firecracker",
			want:         "firecracker",
		},
		{
			name:         "Empty when both are empty",
			vm:           &VM{},
			globalDriver: "",
			want:         "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.vm.GetEffectiveDriver(tt.globalDriver)
			if got != tt.want {
				t.Errorf("GetEffectiveDriver() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVMGetEffectiveDriverOpts(t *testing.T) {
	vm := &VM{
		DriverOpts: map[string]interface{}{
			"vm_option":     "vm_value",
			"shared_option": "vm_override",
		},
	}

	globalOpts := map[string]map[string]interface{}{
		"firecracker": {
			"global_option": "global_value",
			"shared_option": "global_value",
		},
	}

	result := vm.GetEffectiveDriverOpts(globalOpts, "firecracker")

	// Check that VM options are present
	if result["vm_option"] != "vm_value" {
		t.Errorf("Expected vm_option=vm_value, got %v", result["vm_option"])
	}

	// Check that global options are present
	if result["global_option"] != "global_value" {
		t.Errorf("Expected global_option=global_value, got %v", result["global_option"])
	}

	// Check that VM options override global options
	if result["shared_option"] != "vm_override" {
		t.Errorf("Expected shared_option=vm_override, got %v", result["shared_option"])
	}
}

func TestToDriverConfig(t *testing.T) {
	globalConfig := &Config{
		Version: "1",
		Driver:  "firecracker",
		Globals: &GlobalConfig{
			Kernel: "/global/kernel",
			Resources: &Resources{
				VCPU:   2,
				Memory: "1GB",
			},
			Env: map[string]string{
				"GLOBAL_VAR": "global_value",
			},
		},
		DriverOpts: map[string]map[string]interface{}{
			"firecracker": {
				"jailer": true,
			},
		},
	}

	vm := &VM{
		Image: "alpine:latest",
		Env: map[string]string{
			"VM_VAR":     "vm_value",
			"GLOBAL_VAR": "vm_override", // Should override global
		},
		Resources: &Resources{
			Memory: "512MB", // Should override global
		},
	}

	driverConfig, err := vm.ToDriverConfig("test-vm", globalConfig)
	if err != nil {
		t.Fatalf("ToDriverConfig() error = %v", err)
	}

	// Check basic fields
	if driverConfig.Name != "test-vm" {
		t.Errorf("Expected Name=test-vm, got %s", driverConfig.Name)
	}

	if driverConfig.Image != "alpine:latest" {
		t.Errorf("Expected Image=alpine:latest, got %s", driverConfig.Image)
	}

	// Check resource inheritance/override
	if driverConfig.CPUs != 2 {
		t.Errorf("Expected CPUs=2 (from globals), got %d", driverConfig.CPUs)
	}

	if driverConfig.Memory != 512 {
		t.Errorf("Expected Memory=512 (VM override), got %d", driverConfig.Memory)
	}

	// Check environment variable merging
	if driverConfig.Env["GLOBAL_VAR"] != "vm_override" {
		t.Errorf("Expected GLOBAL_VAR=vm_override, got %s", driverConfig.Env["GLOBAL_VAR"])
	}

	if driverConfig.Env["VM_VAR"] != "vm_value" {
		t.Errorf("Expected VM_VAR=vm_value, got %s", driverConfig.Env["VM_VAR"])
	}

	// Check driver options
	if driverConfig.DriverOpts["jailer"] != true {
		t.Errorf("Expected jailer=true, got %v", driverConfig.DriverOpts["jailer"])
	}
}

func TestParseMemoryToMB(t *testing.T) {
	tests := []struct {
		input   string
		want    int64
		wantErr bool
	}{
		{"512MB", 512, false},
		{"1GB", 1024, false},
		{"2G", 2048, false},
		{"256M", 256, false},
		{"", 512, false}, // Default
		{"invalid", 0, true},
		{"512KB", 0, true}, // Unsupported unit
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseMemoryToMB(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMemoryToMB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseMemoryToMB() = %v, want %v", got, tt.want)
			}
		})
	}
}
