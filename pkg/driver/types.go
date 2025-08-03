package driver

import (
	"context"
	"net"
	"time"
)

// Priority determines driver selection order
type Priority int

const (
	Unknown Priority = iota
	Obsolete
	Unhealthy
	Experimental
	Discouraged
	Deprecated
	Fallback
	Default
	Preferred
	HighlyPreferred
)

// State represents the current state of a driver and its dependencies
type State struct {
	Installed        bool
	Healthy          bool
	Running          bool
	NeedsImprovement bool
	Error            error
	Reason           string
	Fix              string
	Doc              string
	Version          string
}

// VMState represents the state of a virtual machine
type VMState int

const (
	None VMState = iota
	Starting
	Running
	Paused
	Stopping
	Stopped
	Error
)

func (s VMState) String() string {
	switch s {
	case None:
		return "None"
	case Starting:
		return "Starting"
	case Running:
		return "Running"
	case Paused:
		return "Paused"
	case Stopping:
		return "Stopping"
	case Stopped:
		return "Stopped"
	case Error:
		return "Error"
	default:
		return "Unknown"
	}
}

// VMInfo contains information about a virtual machine
type VMInfo struct {
	ID       string
	Name     string
	State    VMState
	IP       net.IP
	Created  time.Time
	Driver   string
	Metadata map[string]string
}

// Driver defines the interface that all virtualization drivers must implement
type Driver interface {
	// Lifecycle management
	Create(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Delete(ctx context.Context) error

	// State management
	GetState(ctx context.Context) (VMState, error)
	GetIP(ctx context.Context) (net.IP, error)
	GetInfo(ctx context.Context) (*VMInfo, error)

	// Operations
	RunSSH(ctx context.Context, command string) (string, error)
	CopyFile(ctx context.Context, src, dest string) error

	// Network operations
	CreateNetwork(ctx context.Context, network *NetworkConfig) error
	DeleteNetwork(ctx context.Context, networkName string) error

	// Volume operations
	CreateVolume(ctx context.Context, volume *VolumeConfig) error
	DeleteVolume(ctx context.Context, volumeName string) error
	AttachVolume(ctx context.Context, volumeName, mountPath string) error

	// Metadata
	DriverName() string
	RequiresRoot() bool
	SupportedFeatures() []Feature
}

// NetworkConfig represents network configuration
type NetworkConfig struct {
	Name    string
	Driver  string
	Subnet  string
	Gateway string
	Options map[string]string
}

// VolumeConfig represents volume configuration
type VolumeConfig struct {
	Name    string
	Driver  string
	Size    string
	Options map[string]string
}

// Feature represents driver capabilities
type Feature string

const (
	FeatureNetworking  Feature = "networking"
	FeatureVolumes     Feature = "volumes"
	FeatureSnapshots   Feature = "snapshots"
	FeaturePortForward Feature = "port_forward"
	FeatureFileSync    Feature = "file_sync"
	FeatureGPU         Feature = "gpu"
	FeatureNested      Feature = "nested_virt"
)

// Config represents the configuration passed to a driver
type Config struct {
	// Basic VM configuration
	Name     string
	Image    string
	Rootfs   string
	Memory   int64 // in MB
	CPUs     int
	DiskSize int64 // in MB

	// Networking
	Networks []string
	Ports    []string

	// Storage
	Volumes []VolumeMount

	// Environment
	Env        map[string]string
	WorkingDir string
	User       string

	// Driver-specific options
	DriverOpts map[string]interface{}

	// Paths
	StorePath string
	SSHKey    string
}

// VolumeMount represents a volume mount
type VolumeMount struct {
	Type     string // "volume", "bind"
	Source   string
	Target   string
	ReadOnly bool
}
