package driver

import (
	"context"
	"net"
	"time"
)

//go:generate stringer -type=Priority,VMState

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

	// Setup operations - driver-specific host preparation
	Setup(ctx context.Context, opts *SetupOptions) (*SetupResult, error)
	VerifySetup(ctx context.Context) (*SetupStatus, error)
	GetSetupInstructions(ctx context.Context) (*SetupInstructions, error)

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

// Setup-related types for driver host preparation

// SetupOptions configures how setup should be performed
type SetupOptions struct {
	Interactive bool     // Allow interactive prompts for user input
	DryRun      bool     // Show what would be done without actually doing it
	Force       bool     // Skip confirmation prompts and proceed automatically
	Components  []string // Specific components to setup (e.g., "networking", "kvm", "permissions")
}

// SetupResult contains the outcome of a setup operation
type SetupResult struct {
	Success   bool          // Whether the overall setup succeeded
	Actions   []SetupAction // List of actions that were actually performed
	Warnings  []string      // Non-fatal issues encountered during setup
	NextSteps []string      // Manual steps still required after automated setup
}

// SetupAction represents a single setup action that was performed
type SetupAction struct {
	Component   string // Component being set up (e.g., "kvm", "networking")
	Description string // Human-readable description of what was done
	Command     string // Command that was executed (if applicable)
	Success     bool   // Whether this specific action succeeded
	Error       string // Error message if action failed
}

// SetupStatus represents the current setup state of a driver
type SetupStatus struct {
	Ready      bool              // Whether the driver is ready to use
	Components []ComponentStatus // Status of individual components
	Issues     []SetupIssue      // Problems that need to be resolved
	Summary    string            // Human-readable summary of overall status
}

// ComponentStatus represents the status of a single setup component
type ComponentStatus struct {
	Name        string // Component name (e.g., "kvm", "networking", "permissions")
	Ready       bool   // Whether this component is properly configured
	Description string // What this component provides
	Status      string // Human-readable status message
	Required    bool   // Whether this component is required for basic functionality
}

// SetupIssue represents a problem found during setup verification
type SetupIssue struct {
	Component   string   // Component affected by this issue
	Severity    Severity // How critical this issue is
	Description string   // Human-readable description of the problem
	Resolution  string   // Suggested steps to fix the issue
	Commands    []string // Specific commands to run to fix the issue
	DocLinks    []string // Links to relevant documentation
}

// Severity represents how critical a setup issue is
type Severity string

const (
	SeverityInfo     Severity = "info"     // Informational, doesn't block functionality
	SeverityWarning  Severity = "warning"  // May cause reduced functionality
	SeverityError    Severity = "error"    // Blocks core functionality
	SeverityCritical Severity = "critical" // Completely prevents driver from working
)

// SetupInstructions provides comprehensive setup guidance
type SetupInstructions struct {
	Overview        string      // High-level explanation of what setup involves
	Prerequisites   []string    // Things that must be true before setup can begin
	AutomatedSteps  []SetupStep // Steps the driver can perform automatically
	ManualSteps     []SetupStep // Steps that require manual intervention
	PostSetup       []string    // Things to do after setup is complete
	Documentation   []DocLink   // Links to relevant documentation
	Troubleshooting []DocLink   // Links to troubleshooting guides
}

// SetupStep represents a single step in the setup process
type SetupStep struct {
	Name         string   // Short name for this step
	Description  string   // Detailed description of what this step does
	Commands     []string // Commands to execute for this step
	Verification string   // How to verify this step worked
	Required     bool     // Whether this step is mandatory
	Sudo         bool     // Whether this step requires root privileges
}

// DocLink represents a link to documentation
type DocLink struct {
	Title       string // Human-readable title for the link
	URL         string // URL to the documentation
	Description string // Brief description of what this link contains
}
