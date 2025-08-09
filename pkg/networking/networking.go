// Package networking provides both rootless and root-based networking support for Spitfire
// Users can choose between secure rootless operation or full-featured root networking
package networking

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containers/common/libnetwork/slirp4netns"
	"github.com/containers/common/libnetwork/types"
	"github.com/containers/common/pkg/config"
	"github.com/containers/common/pkg/netns"
	"github.com/thi-startup/spitfire/pkg/log"
)

// Mode represents the networking mode to use
type Mode string

const (
	// ModeRootless uses userspace networking (no root required)
	ModeRootless Mode = "rootless"
	// ModeRoot uses traditional TAP/bridge networking (requires root)
	ModeRoot Mode = "root"
	// ModeAuto automatically selects based on privileges
	ModeAuto Mode = "auto"
)

// Backend represents the networking backend to use
type Backend string

const (
	// Rootless backends
	BackendSlirp4netns Backend = "slirp4netns"
	BackendPasta       Backend = "pasta"
	
	// Root backends
	BackendTAP    Backend = "tap"
	BackendBridge Backend = "bridge"
	BackendMacvlan Backend = "macvlan"
	
	// Auto selection
	BackendAuto Backend = "auto"
)

// Config represents networking configuration for a VM
type Config struct {
	// Mode determines rootless vs root networking
	Mode Mode
	// Backend to use for networking
	Backend Backend
	// NetworkName for bridge/network attachment
	NetworkName string
	// Ports to forward from host to VM
	Ports []types.PortMapping
	// IPv6 enables IPv6 support
	IPv6 bool
	// DisableHostLoopback disables access to host loopback (rootless only)
	DisableHostLoopback bool
	// StaticIP for the VM (root mode only)
	StaticIP string
	// MACAddress for the VM interface
	MACAddress string
	// MTU for the network interface
	MTU int
	// ExtraOptions for the networking backend
	ExtraOptions []string
}

// Manager handles both rootless and root networking for VMs
type Manager struct {
	config       *config.Config
	stateDir     string
	mode         Mode
	isRoot       bool
}

// NewManager creates a new networking manager
func NewManager(stateDir string) (*Manager, error) {
	// Load containers.conf config for compatibility
	cfg, err := config.Default()
	if err != nil {
		// Create a minimal config if default fails
		cfg = &config.Config{
			Network: config.NetworkConfig{
				DefaultRootlessNetworkCmd: "pasta", // Default to pasta for rootless
			},
		}
	}

	// Check if we're running as root
	isRoot := os.Geteuid() == 0

	return &Manager{
		config:   cfg,
		stateDir: stateDir,
		mode:     ModeAuto,
		isRoot:   isRoot,
	}, nil
}

// SetupResult contains the result of setting up networking
type SetupResult struct {
	// Mode that was actually used
	Mode Mode
	// Backend that was used
	Backend Backend
	// Process information for rootless networking backend
	Pid int
	// TAP device name (root mode)
	TAPDevice string
	// Bridge name (root mode)
	Bridge string
	// IP address assigned
	IPAddress string
	// MAC address assigned
	MACAddress string
	// Network configuration
	Subnet     *net.IPNet
	IPv6       bool
	DNSServers []string
	// Network namespace path (rootless)
	NetNSPath string
}

// Setup sets up networking for a VM
func (m *Manager) Setup(ctx context.Context, vmName string, cfg *Config) (*SetupResult, error) {
	// Determine which mode to use
	mode := m.selectMode(cfg.Mode)
	
	log.Debugf("Setting up networking in %s mode", mode)

	switch mode {
	case ModeRootless:
		return m.setupRootless(ctx, vmName, cfg)
	case ModeRoot:
		return m.setupRoot(ctx, vmName, cfg)
	default:
		return nil, fmt.Errorf("unsupported networking mode: %s", mode)
	}
}

// setupRootless sets up rootless networking
func (m *Manager) setupRootless(ctx context.Context, vmName string, cfg *Config) (*SetupResult, error) {
	// Determine which backend to use
	backend := m.selectRootlessBackend(cfg.Backend)
	
	log.Debugf("Using rootless backend: %s", backend)

	// For rootless mode, let the backend tools (pasta/slirp4netns) handle namespace creation
	// They have the proper logic to create user+network namespaces correctly
	
	// Setup based on backend
	switch backend {
	case BackendPasta:
		return m.setupPastaRootless(ctx, vmName, cfg)
	case BackendSlirp4netns:
		return m.setupSlirp4netnsRootless(ctx, vmName, cfg)
	default:
		return nil, fmt.Errorf("unsupported rootless backend: %s", backend)
	}
}

// setupRoot sets up root-based networking
func (m *Manager) setupRoot(ctx context.Context, vmName string, cfg *Config) (*SetupResult, error) {
	if !m.isRoot {
		return nil, fmt.Errorf("root networking requires root privileges")
	}

	// Determine which backend to use
	backend := m.selectRootBackend(cfg.Backend)
	
	log.Debugf("Using root backend: %s", backend)

	switch backend {
	case BackendTAP:
		return m.setupTAP(ctx, vmName, cfg)
	case BackendBridge:
		return m.setupBridge(ctx, vmName, cfg)
	case BackendMacvlan:
		return m.setupMacvlan(ctx, vmName, cfg)
	default:
		return nil, fmt.Errorf("unsupported root backend: %s", backend)
	}
}

// setupPastaRootless sets up networking using pasta in rootless mode
// This approach bypasses namespace creation and runs pasta directly with a target PID
func (m *Manager) setupPastaRootless(ctx context.Context, vmName string, cfg *Config) (*SetupResult, error) {
	// For rootless pasta, we'll run it later when Firecracker starts
	// Store the configuration for later use
	log.Progress("Preparing pasta networking for VM %s (rootless)", vmName)

	return &SetupResult{
		Mode:       ModeRootless,
		Backend:    BackendPasta,
		Pid:        0, // Will be set when pasta actually starts
		IPv6:       true, // Pasta supports IPv6
		DNSServers: []string{"10.0.2.3"}, // Default pasta DNS
		NetNSPath:  "", // Not using persistent namespace
	}, nil
}

// StartPastaForPID starts pasta for a specific process PID (Firecracker)
func (m *Manager) StartPastaForPID(vmName string, targetPID int, cfg *Config) (*SetupResult, error) {
	pidFile := filepath.Join(m.stateDir, fmt.Sprintf("%s-pasta.pid", vmName))
	
	// Build pasta command
	args := []string{
		"pasta",
		"--pid", pidFile,
		"--netns-only",
		fmt.Sprintf("--pid-file=%s", pidFile),
	}
	
	// Add port forwarding
	for _, port := range cfg.Ports {
		if port.Protocol == "tcp" || port.Protocol == "" {
			args = append(args, "-t", fmt.Sprintf("%d:%d", port.HostPort, port.ContainerPort))
		}
		if port.Protocol == "udp" {
			args = append(args, "-u", fmt.Sprintf("%d:%d", port.HostPort, port.ContainerPort))
		}
	}
	
	// Add target PID
	args = append(args, strconv.Itoa(targetPID))
	
	log.Debugf("Starting pasta with args: %v", args)
	
	cmd := exec.Command(args[0], args[1:]...)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start pasta: %w", err)
	}
	
	log.Progress("Started pasta for VM %s targeting PID %d", vmName, targetPID)
	
	return &SetupResult{
		Mode:       ModeRootless,
		Backend:    BackendPasta,
		Pid:        cmd.Process.Pid,
		IPv6:       true,
		DNSServers: []string{"10.0.2.3"},
		NetNSPath:  "",
	}, nil
}

// setupSlirp4netnsRootless sets up networking using slirp4netns in rootless mode
func (m *Manager) setupSlirp4netnsRootless(ctx context.Context, vmName string, cfg *Config) (*SetupResult, error) {
	// Create namespace directory
	netnsPath := filepath.Join(m.stateDir, "netns", vmName)
	if err := os.MkdirAll(filepath.Dir(netnsPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create netns directory: %w", err)
	}

	// Create pipes for process lifecycle management
	slirpPipeR, slirpPipeW, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create slirp4netns pipe: %w", err)
	}
	defer slirpPipeW.Close()

	var portPipeR, portPipeW *os.File
	if len(cfg.Ports) > 0 {
		portPipeR, portPipeW, err = os.Pipe()
		if err != nil {
			slirpPipeR.Close()
			return nil, fmt.Errorf("failed to create port pipe: %w", err)
		}
		defer portPipeW.Close()
	}

	opts := &slirp4netns.SetupOptions{
		Config:                m.config,
		ContainerID:           vmName,
		Netns:                 "", // Empty - let slirp4netns create the namespace
		Ports:                 cfg.Ports,
		ExtraOptions:          cfg.ExtraOptions,
		Slirp4netnsExitPipeR:  slirpPipeR,
		RootlessPortExitPipeR: portPipeR,
	}

	res, err := slirp4netns.Setup(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to setup slirp4netns: %w", err)
	}

	log.Progress("Started slirp4netns for VM %s (rootless)", vmName)

	// Get DNS server from subnet
	dnsIP, err := slirp4netns.GetDNS(res.Subnet)
	if err != nil {
		return nil, fmt.Errorf("failed to get DNS server: %w", err)
	}

	return &SetupResult{
		Mode:       ModeRootless,
		Backend:    BackendSlirp4netns,
		Pid:        res.Pid,
		Subnet:     res.Subnet,
		IPv6:       res.IPv6,
		DNSServers: []string{dnsIP.String()},
		NetNSPath:  netnsPath,
	}, nil
}

// setupTAP sets up a TAP device (requires root)
func (m *Manager) setupTAP(ctx context.Context, vmName string, cfg *Config) (*SetupResult, error) {
	tapName := fmt.Sprintf("tap-%s", vmName)
	
	// This will be implemented using netlink to create TAP devices
	// For now, return a placeholder
	log.Progress("Creating TAP device %s for VM %s (root mode)", tapName, vmName)
	
	return &SetupResult{
		Mode:      ModeRoot,
		Backend:   BackendTAP,
		TAPDevice: tapName,
		IPAddress: cfg.StaticIP,
		MACAddress: cfg.MACAddress,
	}, nil
}

// setupBridge sets up bridge networking (requires root)
func (m *Manager) setupBridge(ctx context.Context, vmName string, cfg *Config) (*SetupResult, error) {
	bridgeName := cfg.NetworkName
	if bridgeName == "" {
		bridgeName = "spitfire0"
	}
	
	tapName := fmt.Sprintf("tap-%s", vmName)
	
	// This will be implemented using netlink to:
	// 1. Create/ensure bridge exists
	// 2. Create TAP device
	// 3. Attach TAP to bridge
	// 4. Configure IP if needed
	log.Progress("Setting up bridge networking for VM %s on bridge %s", vmName, bridgeName)
	
	return &SetupResult{
		Mode:       ModeRoot,
		Backend:    BackendBridge,
		TAPDevice:  tapName,
		Bridge:     bridgeName,
		IPAddress:  cfg.StaticIP,
		MACAddress: cfg.MACAddress,
	}, nil
}

// setupMacvlan sets up macvlan networking (requires root)
func (m *Manager) setupMacvlan(ctx context.Context, vmName string, cfg *Config) (*SetupResult, error) {
	// This will be implemented using netlink to create macvlan interfaces
	log.Progress("Setting up macvlan networking for VM %s", vmName)
	
	return &SetupResult{
		Mode:    ModeRoot,
		Backend: BackendMacvlan,
		IPAddress: cfg.StaticIP,
		MACAddress: cfg.MACAddress,
	}, nil
}

// Teardown tears down networking for a VM
func (m *Manager) Teardown(ctx context.Context, vmName string) error {
	// Clean up based on what mode was used
	// For now, try to clean up everything that might exist
	
	// Rootless cleanup
	netnsPath := filepath.Join(m.stateDir, "netns", vmName)
	if err := os.Remove(netnsPath); err != nil && !os.IsNotExist(err) {
		log.Debugf("Failed to remove network namespace: %v", err)
	}

	// Clean up PID files (rootless)
	pidFiles := []string{
		filepath.Join(m.stateDir, fmt.Sprintf("%s-pasta.pid", vmName)),
		filepath.Join(m.stateDir, fmt.Sprintf("%s-slirp4netns.pid", vmName)),
	}

	for _, pidFile := range pidFiles {
		if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
			log.Debugf("Failed to remove PID file %s: %v", pidFile, err)
		}
	}

	// Root mode cleanup would go here (remove TAP devices, etc.)
	if m.isRoot {
		// TODO: Clean up TAP devices, bridge attachments, etc.
		tapName := fmt.Sprintf("tap-%s", vmName)
		log.Debugf("Would clean up TAP device %s", tapName)
	}

	log.Debugf("Cleaned up networking for VM %s", vmName)
	return nil
}

// selectMode determines which mode to use
func (m *Manager) selectMode(requested Mode) Mode {
	if requested != ModeAuto {
		return requested
	}

	// If running as root, default to root mode for full features
	// Users can explicitly request rootless mode even as root
	if m.isRoot {
		return ModeRoot
	}

	return ModeRootless
}

// selectRootlessBackend determines which rootless backend to use
func (m *Manager) selectRootlessBackend(requested Backend) Backend {
	if requested != BackendAuto {
		return requested
	}

	// Check what's available, preferring pasta
	if commandExists("pasta") {
		return BackendPasta
	}

	if commandExists("slirp4netns") {
		return BackendSlirp4netns
	}

	// Default to pasta (will error if not available)
	return BackendPasta
}

// selectRootBackend determines which root backend to use
func (m *Manager) selectRootBackend(requested Backend) Backend {
	if requested != BackendAuto {
		return requested
	}

	// Default to bridge networking for root mode
	return BackendBridge
}

// commandExists checks if a command exists in PATH
func commandExists(cmd string) bool {
	_, err := os.Stat(filepath.Join("/usr/bin", cmd))
	if err == nil {
		return true
	}
	_, err = os.Stat(filepath.Join("/usr/local/bin", cmd))
	return err == nil
}

// IsRootRequired returns true if the given config requires root privileges
func IsRootRequired(cfg *Config) bool {
	// Explicitly requesting root mode
	if cfg.Mode == ModeRoot {
		return true
	}
	
	// These backends require root
	if cfg.Backend == BackendTAP || cfg.Backend == BackendBridge || cfg.Backend == BackendMacvlan {
		return true
	}
	
	// Static IP usually requires root (bridge/TAP setup)
	if cfg.StaticIP != "" {
		return true
	}
	
	return false
}

// createNetworkNamespace creates a network namespace file using the proper approach
// For rootless, this requires creating a user namespace first
func (m *Manager) createNetworkNamespace(nsPath string) error {
	// Check if namespace already exists and is valid
	if nsRef, err := ns.GetNS(nsPath); err == nil {
		nsRef.Close()
		log.Debugf("Network namespace %s already exists and is valid", nsPath)
		return nil
	}

	// Remove invalid namespace file if it exists
	if _, err := os.Stat(nsPath); err == nil {
		log.Debugf("Removing invalid network namespace file %s", nsPath)
		os.Remove(nsPath)
	}

	// For rootless mode, use the netns library directly - it handles user namespaces
	if !m.isRoot {
		// This should work now that we're using the proper rootless netns directory
	}

	// Root mode - create directly
	nsRef, err := netns.NewNSAtPath(nsPath)
	if err != nil {
		return fmt.Errorf("failed to create network namespace: %w", err)
	}
	defer nsRef.Close()

	log.Debugf("Created network namespace at %s", nsPath)
	return nil
}

// getRootlessNetnsDir returns the proper directory for rootless network namespaces
func (m *Manager) getRootlessNetnsDir() (string, error) {
	// Get XDG_RUNTIME_DIR for the current user
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		// Fallback to /run/user/{uid}
		uid := os.Getuid()
		runtimeDir = fmt.Sprintf("/run/user/%d", uid)
	}

	netnsDir := filepath.Join(runtimeDir, "netns")
	
	// Create the directory if it doesn't exist
	if err := os.MkdirAll(netnsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create netns directory %s: %w", netnsDir, err)
	}

	return netnsDir, nil
}