package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/thi-startup/spitfire/pkg/driver"
)

// ToDriverConfig converts a VM configuration to a driver.Config
func (vm *VM) ToDriverConfig(name string, globalConfig *Config) (*driver.Config, error) {
	// Determine effective driver
	effectiveDriver := vm.GetEffectiveDriver(globalConfig.Driver)
	if effectiveDriver == "" {
		effectiveDriver = "auto" // Default to auto-selection
	}

	// Get effective driver options
	driverOpts := vm.GetEffectiveDriverOpts(globalConfig.DriverOpts, effectiveDriver)

	// Parse memory size
	memory, err := ParseMemoryToMB(vm.getEffectiveMemory(globalConfig))
	if err != nil {
		return nil, fmt.Errorf("invalid memory size: %w", err)
	}

	// Get effective CPU count
	cpus := vm.getEffectiveCPUs(globalConfig)

	// Convert volume mounts
	volumes := make([]driver.VolumeMount, len(vm.Volumes))
	for i, vol := range vm.Volumes {
		volumes[i] = driver.VolumeMount{
			Type:     vol.Type,
			Source:   vol.Source,
			Target:   vol.Target,
			ReadOnly: false, // TODO: Add ReadOnly support to config
		}
	}

	// Merge environment variables (globals + VM-specific)
	env := make(map[string]string)
	if globalConfig.Globals != nil && globalConfig.Globals.Env != nil {
		for k, v := range globalConfig.Globals.Env {
			env[k] = v
		}
	}
	if vm.Env != nil {
		for k, v := range vm.Env {
			env[k] = v
		}
	}

	return &driver.Config{
		Name:       name,
		Image:      vm.Image,
		Rootfs:     vm.Rootfs,
		Memory:     memory,
		CPUs:       cpus,
		DiskSize:   0, // TODO: Add disk size support to config
		Networks:   vm.getEffectiveNetworks(globalConfig),
		Ports:      []string{}, // TODO: Add port mapping support
		Volumes:    volumes,
		Env:        env,
		WorkingDir: "", // TODO: Add working directory support
		User:       "", // TODO: Add user support
		DriverOpts: driverOpts,
		StorePath:  "", // Will be set by the caller
		SSHKey:     "", // Will be set by the caller
	}, nil
}

// getEffectiveMemory returns the memory setting for this VM
func (vm *VM) getEffectiveMemory(globalConfig *Config) string {
	if vm.Resources != nil && vm.Resources.Memory != "" {
		return vm.Resources.Memory
	}
	if globalConfig.Globals != nil && globalConfig.Globals.Resources != nil && globalConfig.Globals.Resources.Memory != "" {
		return globalConfig.Globals.Resources.Memory
	}
	return "512MB" // Default
}

// getEffectiveCPUs returns the CPU count for this VM
func (vm *VM) getEffectiveCPUs(globalConfig *Config) int {
	if vm.Resources != nil && vm.Resources.VCPU > 0 {
		return vm.Resources.VCPU
	}
	if globalConfig.Globals != nil && globalConfig.Globals.Resources != nil && globalConfig.Globals.Resources.VCPU > 0 {
		return globalConfig.Globals.Resources.VCPU
	}
	return 1 // Default
}

// getEffectiveKernel returns the kernel path for this VM
func (vm *VM) getEffectiveKernel(globalConfig *Config) string {
	if vm.Kernel != "" {
		return vm.Kernel
	}
	if globalConfig.Globals != nil && globalConfig.Globals.Kernel != "" {
		return globalConfig.Globals.Kernel
	}
	return "" // Will use driver default
}

// getEffectiveKernelArgs returns the kernel arguments for this VM
func (vm *VM) getEffectiveKernelArgs(globalConfig *Config) string {
	if vm.KernelArgs != "" {
		return vm.KernelArgs
	}
	if globalConfig.Globals != nil && globalConfig.Globals.KernelArgs != "" {
		return globalConfig.Globals.KernelArgs
	}
	return "" // Will use driver default
}

// getEffectiveNetworks returns the networks for this VM
func (vm *VM) getEffectiveNetworks(globalConfig *Config) []string {
	if len(vm.Networks) > 0 {
		return vm.Networks
	}
	if globalConfig.Globals != nil && len(globalConfig.Globals.Networks) > 0 {
		return globalConfig.Globals.Networks
	}
	return []string{} // No networks
}

// ParseMemoryToMB converts memory size strings to MB as int64
func ParseMemoryToMB(size string) (int64, error) {
	if size == "" {
		return 512, nil // Default 512MB
	}

	// Normalize and parse
	size = strings.ToUpper(strings.TrimSpace(size))
	if strings.HasSuffix(size, "MB") || strings.HasSuffix(size, "M") {
		numStr := strings.TrimSuffix(strings.TrimSuffix(size, "MB"), "M")
		mb, err := strconv.ParseInt(numStr, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid memory size format: %s", size)
		}
		return mb, nil
	}

	if strings.HasSuffix(size, "GB") || strings.HasSuffix(size, "G") {
		numStr := strings.TrimSuffix(strings.TrimSuffix(size, "GB"), "G")
		gb, err := strconv.ParseInt(numStr, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid memory size format: %s", size)
		}
		return gb * 1024, nil // Convert GB to MB
	}

	return 0, fmt.Errorf("unsupported memory size format: %s (use MB or GB)", size)
}

// GetVMDriverConfigs returns driver configs for all VMs in the configuration
func (c *Config) GetVMDriverConfigs() (map[string]*driver.Config, error) {
	configs := make(map[string]*driver.Config)

	for name, vm := range c.VMs {
		driverConfig, err := vm.ToDriverConfig(name, c)
		if err != nil {
			return nil, fmt.Errorf("failed to create driver config for VM '%s': %w", name, err)
		}
		configs[name] = driverConfig
	}

	return configs, nil
}
