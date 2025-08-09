package firecracker

import (
	"fmt"

	"github.com/thi-startup/spitfire/pkg/driver"
)

// ConfigBuilder helps build Firecracker configurations from driver.Config
type ConfigBuilder struct {
	driverConfig *driver.Config
	vmState      *VMState
	kernelArgs   KernelCmdLine
	netResult    interface{} // networking.SetupResult - using interface{} to avoid circular import
}

// NewConfigBuilder creates a new configuration builder
func NewConfigBuilder(driverConfig *driver.Config, vmState *VMState) *ConfigBuilder {
	return &ConfigBuilder{
		driverConfig: driverConfig,
		vmState:      vmState,
		kernelArgs:   DefaultKernelCmdLine(),
	}
}

// WithKernelArgs sets custom kernel arguments
func (b *ConfigBuilder) WithKernelArgs(args KernelCmdLine) *ConfigBuilder {
	b.kernelArgs = args
	return b
}

// WithCloudInit adds cloud-init datasource configuration
func (b *ConfigBuilder) WithCloudInit(datasourceURL string) *ConfigBuilder {
	cloudInitArgs := GetCloudInitKernelArgs(datasourceURL)
	b.kernelArgs.MergeFrom(cloudInitArgs)
	return b
}

// WithDebug enables debug kernel arguments
func (b *ConfigBuilder) WithDebug() *ConfigBuilder {
	debugArgs := GetDebugKernelArgs()
	b.kernelArgs.MergeFrom(debugArgs)
	return b
}

// WithNetworking sets the networking setup result
func (b *ConfigBuilder) WithNetworking(netResult interface{}) *ConfigBuilder {
	b.netResult = netResult
	return b
}

// Build creates the complete Firecracker configuration
func (b *ConfigBuilder) Build() (*FirecrackerConfig, error) {
	config := &FirecrackerConfig{
		BootSource:        b.buildBootSource(),
		MachineConfig:     b.buildMachineConfig(),
		Drives:            b.buildDrives(),
		NetworkInterfaces: b.buildNetworkInterfaces(),
		Logger:            b.buildLogger(),
		Metrics:           b.buildMetrics(),
		SocketPath:        b.vmState.SocketPath(),
	}

	// Disable MMDS for now since we disabled networking
	// TODO: Re-enable when networking is working
	// if len(b.driverConfig.Env) > 0 {
	//     config.Mmds = b.buildMmds()
	// }

	return config, nil
}

// buildBootSource creates the boot source configuration
func (b *ConfigBuilder) buildBootSource() BootSource {
	// Get kernel path from driver options or use default
	kernelPath := "/opt/spitfire/vmlinux" // Default kernel path
	if kernel, ok := b.driverConfig.DriverOpts["kernel"].(string); ok && kernel != "" {
		kernelPath = kernel
	}

	// Build kernel command line
	kernelCmdLine := b.kernelArgs.String()

	bootSource := BootSource{
		KernelImagePath: kernelPath,
		BootArgs:        &kernelCmdLine,
	}

	// Add initrd if specified in driver options
	if initrd, ok := b.driverConfig.DriverOpts["initrd"].(string); ok && initrd != "" {
		bootSource.InitrdPath = &initrd
	}

	return bootSource
}

// buildMachineConfig creates the machine configuration
func (b *ConfigBuilder) buildMachineConfig() MachineConfig {
	config := MachineConfig{
		VcpuCount:       int64(b.driverConfig.CPUs),
		MemSizeMib:      b.driverConfig.Memory,
		Smt:             false, // Disable SMT for security
		TrackDirtyPages: false, // Disable for better performance
	}

	// Set CPU template if specified in driver options
	if cpuTemplate, ok := b.driverConfig.DriverOpts["cpu_template"].(string); ok {
		config.CpuTemplate = &cpuTemplate
	}

	return config
}

// buildDrives creates the block device configuration
func (b *ConfigBuilder) buildDrives() []Drive {
	var drives []Drive

	// Add root drive if rootfs or image is specified
	rootfsPath := b.driverConfig.Rootfs
	if rootfsPath == "" {
		rootfsPath = b.driverConfig.Image
	}
	if rootfsPath != "" {
		rootDrive := Drive{
			DriveID:      "root",
			PathOnHost:   rootfsPath,
			IsReadOnly:   false,
			IsRootDevice: true,
			CacheType:    CacheTypeUnsafe, // Use unsafe for better performance
		}
		drives = append(drives, rootDrive)
	}

	// Add additional volumes
	for i, volume := range b.driverConfig.Volumes {
		if volume.Type == "bind" {
			// For bind mounts, we need to create a disk image
			// This is a simplified approach - in practice, you'd want to
			// create proper disk images for bind mounts
			continue
		}

		drive := Drive{
			DriveID:      fmt.Sprintf("drive_%d", i+1),
			PathOnHost:   volume.Source,
			IsReadOnly:   volume.ReadOnly,
			IsRootDevice: false,
			CacheType:    CacheTypeWriteBack,
		}
		drives = append(drives, drive)
	}

	return drives
}

// buildNetworkInterfaces creates network interface configuration
func (b *ConfigBuilder) buildNetworkInterfaces() []NetworkInterface {
	// Check if we have networking setup from the driver
	if b.netResult == nil {
		return []NetworkInterface{}
	}

	// Type assert the network result to determine networking mode
	// For now, we'll use a simple check - if we can't determine the mode,
	// assume rootless and return no interfaces (pasta handles networking externally)
	
	// In rootless mode (pasta/slirp4netns), Firecracker doesn't need network interfaces
	// configured because the networking tools attach to the process namespace
	
	// TODO: When root mode is implemented, check the result and create TAP interfaces
	// if result.Mode == "root" && result.TAPDevice != "" {
	//   return []NetworkInterface{{
	//     IfaceId:      "eth0",
	//     HostDevName:  &result.TAPDevice,
	//     GuestMac:     stringPtr("02:FC:00:00:00:01"),
	//   }}
	// }
	
	// For rootless mode, return no interfaces - networking handled externally
	return []NetworkInterface{}
}

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}

// buildLogger creates the logger configuration
func (b *ConfigBuilder) buildLogger() *Logger {
	// Disable logger for now to avoid file creation issues
	// TODO: Fix logger file creation permissions
	return nil
}

// buildMetrics creates the metrics configuration
func (b *ConfigBuilder) buildMetrics() *Metrics {
	// Disable metrics for now to avoid file creation issues
	// TODO: Fix metrics file creation permissions
	return nil
}

// buildMmds creates the metadata service configuration
func (b *ConfigBuilder) buildMmds() *Mmds {
	return &Mmds{
		Version:           MmdsVersionV1,
		NetworkInterfaces: []string{"eth0"}, // Allow MMDS on first interface
	}
}

// BuildFromDriverConfig is a convenience function to build configuration
// directly from a driver.Config
func BuildFromDriverConfig(driverConfig *driver.Config, stateRoot string) (*FirecrackerConfig, *VMState, error) {
	// Create VM state manager
	vmState := NewVMState(driverConfig.Name, stateRoot)

	// Ensure state directory exists
	if err := vmState.EnsureStateDir(); err != nil {
		return nil, nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	// Create builder and build configuration
	builder := NewConfigBuilder(driverConfig, vmState)

	// Apply driver-specific options
	if err := applyDriverOptions(builder, driverConfig.DriverOpts); err != nil {
		return nil, nil, fmt.Errorf("failed to apply driver options: %w", err)
	}

	config, err := builder.Build()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build config: %w", err)
	}

	return config, vmState, nil
}

// applyDriverOptions applies Firecracker-specific driver options
func applyDriverOptions(builder *ConfigBuilder, opts map[string]interface{}) error {
	if opts == nil {
		return nil
	}

	// Apply debug mode
	if debug, ok := opts["debug"].(bool); ok && debug {
		builder.WithDebug()
	}

	// Apply cloud-init datasource
	if datasource, ok := opts["cloud_init_datasource"].(string); ok && datasource != "" {
		builder.WithCloudInit(datasource)
	}

	// Apply custom kernel arguments
	if kernelArgs, ok := opts["kernel_args"].(map[string]interface{}); ok {
		customArgs := make(KernelCmdLine)
		for key, value := range kernelArgs {
			if valueStr, ok := value.(string); ok {
				customArgs[key] = valueStr
			} else {
				customArgs[key] = ""
			}
		}
		builder.kernelArgs.MergeFrom(customArgs)
	}

	return nil
}

// ValidateFirecrackerConfig validates a Firecracker configuration
func ValidateFirecrackerConfig(config *FirecrackerConfig) error {
	// Validate boot source
	if config.BootSource.KernelImagePath == "" {
		return fmt.Errorf("kernel image path is required")
	}

	// Validate machine config
	if config.MachineConfig.VcpuCount <= 0 {
		return fmt.Errorf("CPU count must be greater than 0")
	}
	if config.MachineConfig.MemSizeMib <= 0 {
		return fmt.Errorf("memory size must be greater than 0")
	}

	// Validate that we have at least one drive or rootfs
	hasRootDevice := false
	for _, drive := range config.Drives {
		if drive.IsRootDevice {
			hasRootDevice = true
			break
		}
	}
	if !hasRootDevice {
		return fmt.Errorf("at least one root device is required")
	}

	// Validate network interfaces
	for _, iface := range config.NetworkInterfaces {
		if iface.IfaceId == "" {
			return fmt.Errorf("network interface ID is required")
		}
		if iface.HostDevName == "" {
			return fmt.Errorf("host device name is required for interface %s", iface.IfaceId)
		}
	}

	return nil
}
