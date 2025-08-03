package config

import (
	"fmt"
	"regexp"
	"strings"
)

// Config represents the complete spitfire configuration
type Config struct {
	Version    string                            `yaml:"version" json:"version"`
	Driver     string                            `yaml:"driver,omitempty" json:"driver,omitempty"`           // Default driver for all VMs
	DriverOpts map[string]map[string]interface{} `yaml:"driver_opts,omitempty" json:"driver_opts,omitempty"` // Driver-specific global options
	Globals    *GlobalConfig                     `yaml:"globals,omitempty" json:"globals,omitempty"`
	VMs        map[string]*VM                    `yaml:"vms" json:"vms"`
	Networks   map[string]*Network               `yaml:"networks,omitempty" json:"networks,omitempty"`
	Volumes    map[string]*Volume                `yaml:"volumes,omitempty" json:"volumes,omitempty"`
}

// GlobalConfig represents global settings that apply to all VMs
type GlobalConfig struct {
	Kernel     string            `yaml:"kernel,omitempty" json:"kernel,omitempty"`
	KernelArgs string            `yaml:"kernel_args,omitempty" json:"kernel_args,omitempty"`
	Resources  *Resources        `yaml:"resources,omitempty" json:"resources,omitempty"`
	Networks   []string          `yaml:"networks,omitempty" json:"networks,omitempty"`
	Restart    string            `yaml:"restart,omitempty" json:"restart,omitempty"`
	Env        map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
}

// VM represents a single microVM configuration
type VM struct {
	Image      string                 `yaml:"image,omitempty" json:"image,omitempty"`
	Rootfs     string                 `yaml:"rootfs,omitempty" json:"rootfs,omitempty"`
	Driver     string                 `yaml:"driver,omitempty" json:"driver,omitempty"`           // Override global driver for this VM
	DriverOpts map[string]interface{} `yaml:"driver_opts,omitempty" json:"driver_opts,omitempty"` // VM-specific driver options
	Resources  *Resources             `yaml:"resources,omitempty" json:"resources,omitempty"`
	Volumes    []VolumeMount          `yaml:"volumes,omitempty" json:"volumes,omitempty"`
	Env        map[string]string      `yaml:"env,omitempty" json:"env,omitempty"`
	EnvFile    []string               `yaml:"env_file,omitempty" json:"env_file,omitempty"`
	DependsOn  []string               `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	Networks   []string               `yaml:"networks,omitempty" json:"networks,omitempty"`
	Restart    string                 `yaml:"restart,omitempty" json:"restart,omitempty"`
	Kernel     string                 `yaml:"kernel,omitempty" json:"kernel,omitempty"`
	KernelArgs string                 `yaml:"kernel_args,omitempty" json:"kernel_args,omitempty"`
}

// Resources defines CPU and memory allocation
type Resources struct {
	VCPU   int    `yaml:"vcpu,omitempty" json:"vcpu,omitempty"`
	Memory string `yaml:"memory,omitempty" json:"memory,omitempty"`
}

// VolumeMount represents a volume mount in a VM
type VolumeMount struct {
	Type   string `yaml:"type,omitempty" json:"type,omitempty"`     // "volume", "bind", or empty (auto-detect)
	Source string `yaml:"source,omitempty" json:"source,omitempty"` // volume name or host path
	Target string `yaml:"target,omitempty" json:"target,omitempty"` // mount point in VM

	// For parsing short syntax like "data:/app/data"
	raw string
}

// Network represents a network configuration
type Network struct {
	Driver  string            `yaml:"driver,omitempty" json:"driver,omitempty"`
	IPAM    *NetworkIPAM      `yaml:"ipam,omitempty" json:"ipam,omitempty"`
	Options map[string]string `yaml:"options,omitempty" json:"options,omitempty"`
}

// NetworkIPAM represents IP address management configuration
type NetworkIPAM struct {
	Config []NetworkIPAMConfig `yaml:"config,omitempty" json:"config,omitempty"`
}

// NetworkIPAMConfig represents a single IPAM configuration block
type NetworkIPAMConfig struct {
	Subnet  string `yaml:"subnet,omitempty" json:"subnet,omitempty"`
	Gateway string `yaml:"gateway,omitempty" json:"gateway,omitempty"`
}

// Volume represents a volume definition
type Volume struct {
	Driver  string            `yaml:"driver,omitempty" json:"driver,omitempty"`
	Persist bool              `yaml:"persist,omitempty" json:"persist,omitempty"`
	Size    string            `yaml:"size,omitempty" json:"size,omitempty"`
	Options map[string]string `yaml:"options,omitempty" json:"options,omitempty"`
}

// ValidateConfig performs basic validation on the configuration
func (c *Config) ValidateConfig() error {
	if c.Version == "" {
		return fmt.Errorf("version is required")
	}

	if c.VMs == nil || len(c.VMs) == 0 {
		return fmt.Errorf("at least one VM must be defined")
	}

	// Validate driver configuration
	if err := c.validateDriverConfig(); err != nil {
		return err
	}

	// Validate each VM
	for name, vm := range c.VMs {
		if err := vm.Validate(name, c.Driver); err != nil {
			return fmt.Errorf("VM '%s': %w", name, err)
		}
	}

	// Validate networks exist for VMs that reference them
	if err := c.validateNetworkReferences(); err != nil {
		return err
	}

	// Validate volumes exist for VMs that reference them
	if err := c.validateVolumeReferences(); err != nil {
		return err
	}

	return nil
}

// Validate checks if a VM configuration is valid
func (vm *VM) Validate(name string, globalDriver string) error {
	if vm.Image == "" && vm.Rootfs == "" {
		return fmt.Errorf("either 'image' or 'rootfs' must be specified")
	}

	if vm.Image != "" && vm.Rootfs != "" {
		return fmt.Errorf("cannot specify both 'image' and 'rootfs'")
	}

	// Validate restart policy
	if vm.Restart != "" {
		validRestartPolicies := []string{"always", "on-failure", "unless-stopped", "no"}
		valid := false
		for _, policy := range validRestartPolicies {
			if vm.Restart == policy {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid restart policy '%s', must be one of: %s",
				vm.Restart, strings.Join(validRestartPolicies, ", "))
		}
	}

	// Validate volume mounts
	for i, mount := range vm.Volumes {
		if err := mount.Validate(); err != nil {
			return fmt.Errorf("volume mount %d: %w", i, err)
		}
	}

	return nil
}

// Validate checks if a volume mount is valid
func (vm *VolumeMount) Validate() error {
	if vm.Source == "" {
		return fmt.Errorf("source cannot be empty")
	}
	if vm.Target == "" {
		return fmt.Errorf("target cannot be empty")
	}
	return nil
}

// validateNetworkReferences ensures all referenced networks exist
func (c *Config) validateNetworkReferences() error {
	allNetworks := make(map[string]bool)

	// Add defined networks
	for name := range c.Networks {
		allNetworks[name] = true
	}

	// Check global networks
	if c.Globals != nil {
		for _, network := range c.Globals.Networks {
			if !allNetworks[network] {
				return fmt.Errorf("global config references undefined network '%s'", network)
			}
		}
	}

	// Check VM networks
	for vmName, vm := range c.VMs {
		for _, network := range vm.Networks {
			if !allNetworks[network] {
				return fmt.Errorf("VM '%s' references undefined network '%s'", vmName, network)
			}
		}
	}

	return nil
}

// validateVolumeReferences ensures all referenced volumes exist (for named volumes)
func (c *Config) validateVolumeReferences() error {
	allVolumes := make(map[string]bool)

	// Add defined volumes
	for name := range c.Volumes {
		allVolumes[name] = true
	}

	// Check VM volume references
	for vmName, vm := range c.VMs {
		for _, mount := range vm.Volumes {
			// Only check named volumes (not bind mounts)
			if mount.Type == "volume" || (mount.Type == "" && !strings.Contains(mount.Source, "/")) {
				if !allVolumes[mount.Source] {
					return fmt.Errorf("VM '%s' references undefined volume '%s'", vmName, mount.Source)
				}
			}
		}
	}

	return nil
}

// GetEffectiveDriver returns the driver to use for this VM
// VM-specific driver overrides global driver
func (vm *VM) GetEffectiveDriver(globalDriver string) string {
	if vm.Driver != "" {
		return vm.Driver
	}
	return globalDriver
}

// GetEffectiveDriverOpts merges global and VM-specific driver options
// VM-specific options override global options for the same driver
func (vm *VM) GetEffectiveDriverOpts(globalDriverOpts map[string]map[string]interface{}, driver string) map[string]interface{} {
	result := make(map[string]interface{})

	// Start with global options for this driver
	if globalDriverOpts != nil && globalDriverOpts[driver] != nil {
		for k, v := range globalDriverOpts[driver] {
			result[k] = v
		}
	}

	// Override with VM-specific options
	if vm.DriverOpts != nil {
		for k, v := range vm.DriverOpts {
			result[k] = v
		}
	}

	return result
}

// ParseMemorySize validates and normalizes memory size strings like "512MB", "1GB"
func ParseMemorySize(size string) (string, error) {
	if size == "" {
		return "", fmt.Errorf("memory size cannot be empty")
	}

	// Basic regex for memory sizes
	re := regexp.MustCompile(`^(\d+)(MB|GB|M|G)$`)
	matches := re.FindStringSubmatch(strings.ToUpper(size))

	if len(matches) != 3 {
		return "", fmt.Errorf("invalid memory size format '%s', expected format like '512MB' or '1GB'", size)
	}

	return strings.ToUpper(size), nil
}

// validateDriverConfig validates driver-related configuration
func (c *Config) validateDriverConfig() error {
	// For now, we'll do basic validation without importing the driver package
	// to avoid circular dependencies. More advanced validation can be done
	// at runtime when the driver registry is available.

	// Validate global driver name format (basic check)
	if c.Driver != "" {
		if err := validateDriverName(c.Driver); err != nil {
			return fmt.Errorf("invalid global driver: %w", err)
		}
	}

	// Validate driver_opts structure
	if c.DriverOpts != nil {
		for driverName, opts := range c.DriverOpts {
			if err := validateDriverName(driverName); err != nil {
				return fmt.Errorf("invalid driver name in driver_opts '%s': %w", driverName, err)
			}
			if opts == nil {
				return fmt.Errorf("driver_opts for '%s' cannot be null", driverName)
			}
		}
	}

	return nil
}

// validateDriverName performs basic validation on driver names
func validateDriverName(name string) error {
	if name == "" {
		return fmt.Errorf("driver name cannot be empty")
	}

	// Basic format validation - driver names should be lowercase alphanumeric with hyphens
	matched, err := regexp.MatchString(`^[a-z][a-z0-9-]*[a-z0-9]$|^[a-z]$`, name)
	if err != nil {
		return err
	}
	if !matched {
		return fmt.Errorf("driver name '%s' must be lowercase alphanumeric with hyphens", name)
	}

	return nil
}
