package firecracker

// FirecrackerConfig represents the complete Firecracker configuration
// Based on Flintlock's VmmConfig and Firecracker's JSON schema
type FirecrackerConfig struct {
	// BootSource contains kernel and boot configuration
	BootSource BootSource `json:"boot-source"`
	// Drives contains all block device configurations
	Drives []Drive `json:"drives"`
	// MachineConfig contains CPU and memory configuration
	MachineConfig MachineConfig `json:"machine-config"`
	// NetworkInterfaces contains network device configurations
	NetworkInterfaces []NetworkInterface `json:"network-interfaces"`
	// Logger contains logging configuration
	Logger *Logger `json:"logger,omitempty"`
	// Metrics contains metrics configuration
	Metrics *Metrics `json:"metrics,omitempty"`
	// Mmds contains metadata service configuration
	Mmds *Mmds `json:"mmds-config,omitempty"`
	// VsockDevice contains vsock configuration
	VsockDevice *VsockDevice `json:"vsock,omitempty"`
	// SocketPath is used internally for the API socket
	SocketPath string `json:"-"`
}

// BootSource configures the kernel and boot process
type BootSource struct {
	// KernelImagePath is the path to the kernel image
	KernelImagePath string `json:"kernel_image_path"`
	// InitrdPath is the path to the initrd (optional)
	InitrdPath *string `json:"initrd_path,omitempty"`
	// BootArgs contains kernel command line arguments
	BootArgs *string `json:"boot_args,omitempty"`
}

// Drive represents a block device configuration
type Drive struct {
	// DriveID is the unique identifier for the drive
	DriveID string `json:"drive_id"`
	// PathOnHost is the path to the backing file/device on host
	PathOnHost string `json:"path_on_host"`
	// IsReadOnly indicates if the drive is read-only
	IsReadOnly bool `json:"is_read_only"`
	// IsRootDevice indicates if this is the root filesystem
	IsRootDevice bool `json:"is_root_device"`
	// Partuuid specifies the partition UUID (optional)
	Partuuid *string `json:"partuuid,omitempty"`
	// CacheType specifies the cache behavior
	CacheType CacheType `json:"cache_type"`
}

// CacheType defines cache behavior for drives
type CacheType string

const (
	// CacheTypeUnsafe allows faster but less safe caching
	CacheTypeUnsafe CacheType = "Unsafe"
	// CacheTypeWriteBack uses safe write-back caching
	CacheTypeWriteBack CacheType = "WriteBack"
)

// MachineConfig contains CPU and memory settings
type MachineConfig struct {
	// VcpuCount is the number of virtual CPUs
	VcpuCount int64 `json:"vcpu_count"`
	// MemSizeMib is the memory size in MiB
	MemSizeMib int64 `json:"mem_size_mib"`
	// Smt enables/disables simultaneous multithreading
	Smt bool `json:"smt"`
	// CpuTemplate specifies CPU template (C3, T2, etc.)
	CpuTemplate *string `json:"cpu_template,omitempty"`
	// TrackDirtyPages enables dirty page tracking for snapshots
	TrackDirtyPages bool `json:"track_dirty_pages"`
}

// NetworkInterface represents a network device configuration
type NetworkInterface struct {
	// IfaceId is the interface identifier
	IfaceId string `json:"iface_id"`
	// HostDevName is the TAP device name on host
	HostDevName string `json:"host_dev_name"`
	// GuestMac is the MAC address for the guest interface
	GuestMac *string `json:"guest_mac,omitempty"`
}

// Logger configures Firecracker logging
type Logger struct {
	// LogPath is the path to the log file
	LogPath string `json:"log_path"`
	// Level is the log level
	Level LogLevel `json:"level"`
	// ShowLevel includes log level in output
	ShowLevel bool `json:"show_level"`
	// ShowLogOrigin includes source location in logs
	ShowLogOrigin bool `json:"show_log_origin"`
}

// LogLevel defines logging verbosity
type LogLevel string

const (
	LogLevelError   LogLevel = "Error"
	LogLevelWarning LogLevel = "Warning"
	LogLevelInfo    LogLevel = "Info"
	LogLevelDebug   LogLevel = "Debug"
)

// Metrics configures metrics output
type Metrics struct {
	// MetricsPath is the path to the metrics file
	MetricsPath string `json:"metrics_path"`
}

// Mmds configures the metadata service
type Mmds struct {
	// Version specifies MMDS version (V1 or V2)
	Version MmdsVersion `json:"version,omitempty"`
	// NetworkInterfaces specifies which interfaces allow MMDS access
	NetworkInterfaces []string `json:"network_interfaces,omitempty"`
	// Ipv4Address specifies the MMDS IPv4 address
	Ipv4Address *string `json:"ipv4_address,omitempty"`
}

// MmdsVersion defines MMDS protocol version
type MmdsVersion string

const (
	MmdsVersionV1 MmdsVersion = "V1"
	MmdsVersionV2 MmdsVersion = "V2"
)

// VsockDevice configures vsock communication
type VsockDevice struct {
	// VsockId is the vsock device identifier
	VsockId string `json:"vsock_id"`
	// GuestCid is the guest context ID
	GuestCid int64 `json:"guest_cid"`
	// UdsPath is the path to the Unix domain socket
	UdsPath string `json:"uds_path"`
}

// Metadata represents the metadata to be served by MMDS
type Metadata struct {
	Latest map[string]string `json:"latest"`
}
