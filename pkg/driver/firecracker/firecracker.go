package firecracker

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/thi-startup/spitfire/pkg/driver"
	"github.com/thi-startup/spitfire/pkg/log"
)

const (
	// DefaultStateRoot is the default directory for VM state
	DefaultStateRoot = "/tmp/spitfire/firecracker"
	// DefaultFirecrackerBinary is the default firecracker binary name
	DefaultFirecrackerBinary = "firecracker"
)

// Driver implements the spitfire driver interface for Firecracker
// Uses configuration-based approach inspired by Flintlock
type Driver struct {
	config         *driver.Config
	stateRoot      string
	firecrackerBin string
	vmState        *VMState
	fcConfig       *FirecrackerConfig
	logger         *logrus.Logger
}

// NewFirecrackerDriver creates a new Firecracker driver instance
func NewFirecrackerDriver(config *driver.Config) (driver.Driver, error) {
	if config.Name == "" {
		return nil, fmt.Errorf("VM name is required")
	}

	// Determine state root from config or use default
	stateRoot := DefaultStateRoot
	if config.StorePath != "" {
		stateRoot = config.StorePath
	}

	// Determine firecracker binary path
	firecrackerBin := DefaultFirecrackerBinary
	if bin, ok := config.DriverOpts["firecracker_bin"].(string); ok && bin != "" {
		firecrackerBin = bin
	}

	// Find the firecracker binary
	if !filepath.IsAbs(firecrackerBin) {
		var err error
		firecrackerBin, err = exec.LookPath(firecrackerBin)
		if err != nil {
			return nil, fmt.Errorf("firecracker binary not found: %w", err)
		}
	}

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	d := &Driver{
		config:         config,
		stateRoot:      stateRoot,
		firecrackerBin: firecrackerBin,
		logger:         logger,
	}

	// Build Firecracker configuration
	if err := d.buildFirecrackerConfig(); err != nil {
		return nil, fmt.Errorf("failed to build firecracker config: %w", err)
	}

	return d, nil
}

// buildFirecrackerConfig converts driver config to firecracker config
func (d *Driver) buildFirecrackerConfig() error {
	// Build configuration using our new config builder
	fcConfig, vmState, err := BuildFromDriverConfig(d.config, d.stateRoot)
	if err != nil {
		return fmt.Errorf("building firecracker config: %w", err)
	}

	// Validate the configuration
	if err := ValidateFirecrackerConfig(fcConfig); err != nil {
		return fmt.Errorf("invalid firecracker config: %w", err)
	}

	d.fcConfig = fcConfig
	d.vmState = vmState

	return nil
}

// Create implements driver.Driver
func (d *Driver) Create(ctx context.Context) error {
	d.logger.WithField("vm", d.config.Name).Info("creating Firecracker VM")

	// Check if VM is already properly configured
	if d.vmState.Exists() {
		// Check if config file exists - if not, we need to create it
		if _, err := os.Stat(d.vmState.ConfigPath()); err == nil {
			d.logger.WithField("vm", d.config.Name).Debug("VM config already exists")
			return nil // Already created
		}
		d.logger.WithField("vm", d.config.Name).Debug("VM directory exists but no config, recreating")
	}

	log.Debugf("About to save config to %s", d.vmState.ConfigPath())
	// Save the Firecracker configuration to disk
	if err := d.vmState.SetConfig(d.fcConfig); err != nil {
		log.Debugf("Failed to save config: %v", err)
		return fmt.Errorf("failed to save firecracker config: %w", err)
	}
	log.Debugf("Config saved successfully")

	// Create metadata if we have environment variables
	if len(d.config.Env) > 0 {
		metadata := &Metadata{
			Latest: d.config.Env,
		}
		if err := d.vmState.SetMetadata(metadata); err != nil {
			return fmt.Errorf("failed to save metadata: %w", err)
		}
	}

	d.logger.WithField("vm", d.config.Name).Info("VM created successfully")
	return nil
}

// Start implements driver.Driver
func (d *Driver) Start(ctx context.Context) error {
	d.logger.WithField("vm", d.config.Name).Info("starting Firecracker VM")

	// Check if VM is already running (only if PID file exists)
	if running, err := d.vmState.IsRunning(); err == nil && running {
		d.logger.WithField("vm", d.config.Name).Debug("VM is already running")
		return nil
	}

	// Ensure state directory exists
	if err := d.vmState.EnsureStateDir(); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Get the configuration file path
	configPath := d.vmState.ConfigPath()

	// Build firecracker command
	args := []string{
		"--api-sock", d.vmState.SocketPath(),
		"--config-file", configPath,
	}

	// Create the command
	cmd := exec.CommandContext(ctx, d.firecrackerBin, args...)

	// Set up stdout and stderr files
	stdoutFile, err := os.Create(d.vmState.StdoutPath())
	if err != nil {
		return fmt.Errorf("failed to create stdout file: %w", err)
	}
	defer stdoutFile.Close()

	stderrFile, err := os.Create(d.vmState.StderrPath())
	if err != nil {
		return fmt.Errorf("failed to create stderr file: %w", err)
	}
	defer stderrFile.Close()

	cmd.Stdout = stdoutFile
	cmd.Stderr = stderrFile

	// Set process group to make signal handling easier
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start firecracker process: %w", err)
	}

	// Save the PID
	if err := d.vmState.SetPID(cmd.Process.Pid); err != nil {
		// If we can't save PID, kill the process to avoid orphans
		cmd.Process.Kill()
		return fmt.Errorf("failed to save PID: %w", err)
	}

	d.logger.WithField("vm", d.config.Name).WithField("pid", cmd.Process.Pid).Info("Firecracker VM started")

	// Start a goroutine to wait for the process and clean up PID file when it exits
	go func() {
		cmd.Wait()
		os.Remove(d.vmState.PIDPath()) // Clean up PID file when process exits
	}()

	return nil
}

// Stop implements driver.Driver
func (d *Driver) Stop(ctx context.Context) error {
	d.logger.WithField("vm", d.config.Name).Info("stopping Firecracker VM")

	// Check if VM is running
	running, err := d.vmState.IsRunning()
	if err != nil {
		// If PID file doesn't exist, VM is not running
		d.logger.WithField("vm", d.config.Name).Debug("VM is not running (no PID file)")
		return nil
	}
	if !running {
		d.logger.WithField("vm", d.config.Name).Debug("VM is not running")
		return nil
	}

	// Get the PID
	pid, err := d.vmState.GetPID()
	if err != nil {
		return fmt.Errorf("failed to get VM PID: %w", err)
	}

	// Find the process
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Try graceful shutdown first with SIGTERM
	d.logger.WithField("vm", d.config.Name).WithField("pid", pid).Debug("sending SIGTERM")
	if err := process.Signal(syscall.SIGTERM); err != nil {
		d.logger.WithField("vm", d.config.Name).WithField("pid", pid).Warn("failed to send SIGTERM, process may already be dead")
		// Clean up PID file and return success - process is already gone
		os.Remove(d.vmState.PIDPath())
		return nil
	}

	// Wait for graceful shutdown with timeout
	gracefulTimeout := 10 * time.Second
	shutdownCtx, cancel := context.WithTimeout(ctx, gracefulTimeout)
	defer cancel()

	// Poll for process termination
	shutdownComplete := make(chan bool, 1)
	go func() {
		for {
			if running, _ := d.vmState.IsRunning(); !running {
				shutdownComplete <- true
				return
			}
			select {
			case <-shutdownCtx.Done():
				return
			default:
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	select {
	case <-shutdownComplete:
		d.logger.WithField("vm", d.config.Name).Info("VM stopped gracefully")
		return nil
	case <-shutdownCtx.Done():
		// Graceful shutdown timed out, force kill
		d.logger.WithField("vm", d.config.Name).WithField("pid", pid).Warn("graceful shutdown timed out, sending SIGKILL")
		if err := process.Signal(syscall.SIGKILL); err != nil {
			return fmt.Errorf("failed to force kill process: %w", err)
		}

		// Wait a bit more for force kill to take effect
		time.Sleep(2 * time.Second)

		// Clean up PID file
		os.Remove(d.vmState.PIDPath())
		d.logger.WithField("vm", d.config.Name).Info("VM stopped forcefully")
		return nil
	}
}

// Delete implements driver.Driver
func (d *Driver) Delete(ctx context.Context) error {
	d.logger.WithField("vm", d.config.Name).Info("deleting Firecracker VM")

	// Stop the VM first
	if err := d.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop VM before deletion: %w", err)
	}

	// Clean up all state files and directories
	if err := d.vmState.Cleanup(); err != nil {
		d.logger.WithField("vm", d.config.Name).WithError(err).Warn("failed to cleanup some state files")
		// Don't return error - continue with cleanup
	}

	// Clean up any remaining socket files (they might be outside state directory)
	socketPath := d.vmState.SocketPath()
	if _, err := os.Stat(socketPath); err == nil {
		if err := os.Remove(socketPath); err != nil {
			d.logger.WithField("vm", d.config.Name).WithError(err).Warn("failed to remove socket file")
		}
	}

	d.logger.WithField("vm", d.config.Name).Info("VM deleted successfully")
	return nil
}

// GetState implements driver.Driver
func (d *Driver) GetState(ctx context.Context) (driver.VMState, error) {
	// Check if VM state exists
	if !d.vmState.Exists() {
		return driver.None, nil
	}

	// Check if VM process is running
	running, err := d.vmState.IsRunning()
	if err != nil {
		// If we can't determine state, assume stopped
		d.logger.WithField("vm", d.config.Name).WithError(err).Debug("failed to check running state")
		return driver.Stopped, nil
	}

	if running {
		return driver.Running, nil
	}

	return driver.Stopped, nil
}

// GetIP implements driver.Driver
func (d *Driver) GetIP(ctx context.Context) (net.IP, error) {
	// Check if VM is running
	state, err := d.GetState(ctx)
	if err != nil {
		return nil, err
	}
	if state != driver.Running {
		return nil, fmt.Errorf("VM is not running")
	}

	// TODO: Implement IP retrieval from firecracker
	// This would typically involve:
	// 1. Checking DHCP leases file
	// 2. Querying network interface status
	// 3. Using cloud-init metadata if available
	// 4. Reading from guest agent if available
	return nil, fmt.Errorf("GetIP not yet implemented")
}

// GetInfo implements driver.Driver
func (d *Driver) GetInfo(ctx context.Context) (*driver.VMInfo, error) {
	state, err := d.GetState(ctx)
	if err != nil {
		return nil, err
	}

	ip, _ := d.GetIP(ctx) // Ignore error if VM not running

	// Try to get creation time from state directory
	createdTime := time.Now()
	if stat, err := os.Stat(d.vmState.Root()); err == nil {
		createdTime = stat.ModTime()
	}

	// Build metadata
	metadata := map[string]string{
		"memory":     fmt.Sprintf("%dMB", d.config.Memory),
		"cpus":       fmt.Sprintf("%d", d.config.CPUs),
		"socket":     d.vmState.SocketPath(),
		"state_root": d.vmState.Root(),
	}

	// Add PID if running
	if state == driver.Running {
		if pid, err := d.vmState.GetPID(); err == nil {
			metadata["pid"] = fmt.Sprintf("%d", pid)
		}
	}

	// Add kernel and rootfs info if available
	if kernel, ok := d.config.DriverOpts["kernel"].(string); ok && kernel != "" {
		metadata["kernel"] = kernel
	}
	if d.config.Rootfs != "" {
		metadata["rootfs"] = d.config.Rootfs
	}

	return &driver.VMInfo{
		ID:       d.config.Name,
		Name:     d.config.Name,
		State:    state,
		IP:       ip,
		Created:  createdTime,
		Driver:   d.DriverName(),
		Metadata: metadata,
	}, nil
}

// RunSSH implements driver.Driver
func (d *Driver) RunSSH(ctx context.Context, command string) (string, error) {
	ip, err := d.GetIP(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get VM IP: %w", err)
	}

	// TODO: Implement SSH execution
	_ = ip
	return "", fmt.Errorf("RunSSH not yet implemented")
}

// CopyFile implements driver.Driver
func (d *Driver) CopyFile(ctx context.Context, src, dest string) error {
	// TODO: Implement file copying via SCP or other method
	return fmt.Errorf("CopyFile not yet implemented")
}

// CreateNetwork implements driver.Driver
func (d *Driver) CreateNetwork(ctx context.Context, network *driver.NetworkConfig) error {
	// TODO: Implement network creation (TAP devices, bridges)
	return fmt.Errorf("CreateNetwork not yet implemented")
}

// DeleteNetwork implements driver.Driver
func (d *Driver) DeleteNetwork(ctx context.Context, networkName string) error {
	// TODO: Implement network deletion
	return fmt.Errorf("DeleteNetwork not yet implemented")
}

// CreateVolume implements driver.Driver
func (d *Driver) CreateVolume(ctx context.Context, volume *driver.VolumeConfig) error {
	// TODO: Implement volume creation (block devices for firecracker)
	return fmt.Errorf("CreateVolume not yet implemented")
}

// DeleteVolume implements driver.Driver
func (d *Driver) DeleteVolume(ctx context.Context, volumeName string) error {
	// TODO: Implement volume deletion
	return fmt.Errorf("DeleteVolume not yet implemented")
}

// AttachVolume implements driver.Driver
func (d *Driver) AttachVolume(ctx context.Context, volumeName, mountPath string) error {
	// TODO: Implement volume attachment to firecracker VM
	return fmt.Errorf("AttachVolume not yet implemented")
}

// DriverName implements driver.Driver
func (d *Driver) DriverName() string {
	return "firecracker"
}

// RequiresRoot implements driver.Driver
func (d *Driver) RequiresRoot() bool {
	// Firecracker typically requires access to /dev/kvm which needs special permissions
	return false // With proper setup (user in kvm group), root is not required
}

// SupportedFeatures implements driver.Driver
func (d *Driver) SupportedFeatures() []driver.Feature {
	return []driver.Feature{
		driver.FeatureNetworking,
		driver.FeatureVolumes,
		driver.FeatureFileSync,
		// Firecracker doesn't support GPU passthrough in standard mode
		// driver.FeatureGPU,
	}
}

// Setup implements driver.Driver for Firecracker-specific host setup
func (d *Driver) Setup(ctx context.Context, opts *driver.SetupOptions) (*driver.SetupResult, error) {
	result := &driver.SetupResult{
		Success: true,
		Actions: []driver.SetupAction{},
	}

	log.Progress("Starting Firecracker driver setup...")

	// 1. Verify system requirements and binaries
	if err := d.verifySystemRequirements(opts, result); err != nil {
		result.Success = false
		return result, err
	}

	// 2. Setup KVM access and permissions
	if err := d.setupKVMAccess(opts, result); err != nil {
		result.Success = false
		return result, err
	}

	// 3. Setup networking infrastructure
	if err := d.setupNetworking(opts, result); err != nil {
		result.Success = false
		return result, err
	}

	// 4. Setup firewall rules
	if err := d.setupFirewall(opts, result); err != nil {
		result.Success = false
		return result, err
	}

	// 5. Create required directories
	if err := d.setupDirectories(opts, result); err != nil {
		result.Success = false
		return result, err
	}

	// Add final success message
	if result.Success {
		result.NextSteps = []string{
			"Log out and back in (or reboot) if user was added to kvm group",
			"Verify setup with: spitfire driver verify firecracker",
			"Test with a simple VM: spitfire up --driver firecracker",
		}
	}

	return result, nil
}

// verifySystemRequirements checks system prerequisites and required binaries
func (d *Driver) verifySystemRequirements(opts *driver.SetupOptions, result *driver.SetupResult) error {
	log.Debugf("Verifying system requirements...")

	// Check required binaries
	requiredBinaries := []struct {
		name     string
		required bool
		desc     string
	}{
		{"firecracker", true, "Firecracker hypervisor binary"},
		{"ip", true, "Network configuration utility"},
		{"iptables", true, "Firewall configuration utility"},
		{"brctl", false, "Bridge control utility (optional, ip can substitute)"},
	}

	for _, binary := range requiredBinaries {
		path, err := exec.LookPath(binary.name)
		action := driver.SetupAction{
			Component:   "binaries",
			Description: fmt.Sprintf("Checking %s: %s", binary.name, binary.desc),
			Success:     err == nil,
		}

		if err != nil {
			if binary.required {
				action.Error = fmt.Sprintf("Required binary %s not found in PATH", binary.name)
				result.Actions = append(result.Actions, action)
				return fmt.Errorf("required binary %s not found", binary.name)
			} else {
				action.Description = fmt.Sprintf("Optional binary %s not found (skipping)", binary.name)
				result.Warnings = append(result.Warnings, fmt.Sprintf("Optional binary %s not found", binary.name))
			}
		} else {
			action.Description = fmt.Sprintf("Found %s at %s", binary.name, path)
		}

		if !opts.DryRun {
			result.Actions = append(result.Actions, action)
		}
	}

	return nil
}

// setupKVMAccess ensures KVM device access and user permissions
func (d *Driver) setupKVMAccess(opts *driver.SetupOptions, result *driver.SetupResult) error {
	log.Debugf("Setting up KVM access...")

	// Check if /dev/kvm exists
	if _, err := os.Stat("/dev/kvm"); err != nil {
		action := driver.SetupAction{
			Component:   "kvm",
			Description: "Checking KVM device access",
			Success:     false,
			Error:       "KVM device /dev/kvm not found - hardware virtualization not supported",
		}
		result.Actions = append(result.Actions, action)
		return fmt.Errorf("KVM not supported on this system")
	}

	// Check if user is in kvm group
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	groups, err := currentUser.GroupIds()
	if err != nil {
		return fmt.Errorf("failed to get user groups: %w", err)
	}

	inKVMGroup := false
	for _, gid := range groups {
		if group, err := user.LookupGroupId(gid); err == nil && group.Name == "kvm" {
			inKVMGroup = true
			break
		}
	}

	action := driver.SetupAction{
		Component:   "kvm-permissions",
		Description: "Checking KVM group membership",
		Success:     inKVMGroup,
	}

	if inKVMGroup {
		action.Description = "User is already in kvm group"
	} else {
		action.Description = "User needs to be added to kvm group"
		action.Command = fmt.Sprintf("sudo usermod -a -G kvm %s", currentUser.Username)

		if !opts.DryRun {
			// Try to add user to kvm group
			cmd := exec.Command("sudo", "usermod", "-a", "-G", "kvm", currentUser.Username)
			if err := cmd.Run(); err != nil {
				action.Success = false
				action.Error = fmt.Sprintf("Failed to add user to kvm group: %v", err)
			} else {
				action.Success = true
				action.Description = "Successfully added user to kvm group"
				result.NextSteps = append(result.NextSteps, "Log out and back in for group membership to take effect")
			}
		}
	}

	result.Actions = append(result.Actions, action)
	return nil
}

// setupNetworking creates network bridge and configures IP forwarding
func (d *Driver) setupNetworking(opts *driver.SetupOptions, result *driver.SetupResult) error {
	log.Debugf("Setting up networking...")

	bridgeName := "spitfire0"
	bridgeIP := "172.16.0.1/24"

	// Check if bridge already exists
	_, err := net.InterfaceByName(bridgeName)
	bridgeExists := err == nil

	if !bridgeExists {
		// Create bridge
		action := driver.SetupAction{
			Component:   "networking",
			Description: fmt.Sprintf("Creating bridge interface %s", bridgeName),
			Command:     fmt.Sprintf("ip link add %s type bridge", bridgeName),
		}

		if !opts.DryRun {
			cmd := exec.Command("sudo", "ip", "link", "add", bridgeName, "type", "bridge")
			if err := cmd.Run(); err != nil {
				action.Success = false
				action.Error = fmt.Sprintf("Failed to create bridge: %v", err)
				result.Actions = append(result.Actions, action)
				return fmt.Errorf("failed to create bridge interface: %w", err)
			}
			action.Success = true
		} else {
			action.Success = true
		}
		result.Actions = append(result.Actions, action)

		// Set bridge IP
		action = driver.SetupAction{
			Component:   "networking",
			Description: fmt.Sprintf("Configuring bridge IP %s", bridgeIP),
			Command:     fmt.Sprintf("ip addr add %s dev %s", bridgeIP, bridgeName),
		}

		if !opts.DryRun {
			cmd := exec.Command("sudo", "ip", "addr", "add", bridgeIP, "dev", bridgeName)
			if err := cmd.Run(); err != nil {
				action.Success = false
				action.Error = fmt.Sprintf("Failed to set bridge IP: %v", err)
				result.Actions = append(result.Actions, action)
				return fmt.Errorf("failed to set bridge IP: %w", err)
			}
			action.Success = true
		} else {
			action.Success = true
		}
		result.Actions = append(result.Actions, action)

		// Bring bridge up
		action = driver.SetupAction{
			Component:   "networking",
			Description: fmt.Sprintf("Bringing up bridge interface %s", bridgeName),
			Command:     fmt.Sprintf("ip link set %s up", bridgeName),
		}

		if !opts.DryRun {
			cmd := exec.Command("sudo", "ip", "link", "set", bridgeName, "up")
			if err := cmd.Run(); err != nil {
				action.Success = false
				action.Error = fmt.Sprintf("Failed to bring up bridge: %v", err)
				result.Actions = append(result.Actions, action)
				return fmt.Errorf("failed to bring up bridge: %w", err)
			}
			action.Success = true
		} else {
			action.Success = true
		}
		result.Actions = append(result.Actions, action)
	} else {
		// Bridge already exists
		action := driver.SetupAction{
			Component:   "networking",
			Description: fmt.Sprintf("Bridge %s already exists", bridgeName),
			Success:     true,
		}
		result.Actions = append(result.Actions, action)
	}

	// Enable IP forwarding
	forwardingBytes, err := os.ReadFile("/proc/sys/net/ipv4/ip_forward")
	if err != nil {
		return fmt.Errorf("failed to read IP forwarding status: %w", err)
	}

	forwardingEnabled := strings.TrimSpace(string(forwardingBytes)) == "1"
	
	action := driver.SetupAction{
		Component:   "networking",
		Description: "Enabling IP forwarding",
		Command:     "echo 1 | sudo tee /proc/sys/net/ipv4/ip_forward",
	}

	if !forwardingEnabled {
		if !opts.DryRun {
			cmd := exec.Command("sudo", "tee", "/proc/sys/net/ipv4/ip_forward")
			cmd.Stdin = strings.NewReader("1")
			if err := cmd.Run(); err != nil {
				action.Success = false
				action.Error = fmt.Sprintf("Failed to enable IP forwarding: %v", err)
				result.Actions = append(result.Actions, action)
				return fmt.Errorf("failed to enable IP forwarding: %w", err)
			}
			action.Success = true
		} else {
			action.Success = true
		}
	} else {
		action.Description = "IP forwarding already enabled"
		action.Success = true
	}
	result.Actions = append(result.Actions, action)

	return nil
}

// setupFirewall configures iptables rules for VM networking
func (d *Driver) setupFirewall(opts *driver.SetupOptions, result *driver.SetupResult) error {
	log.Debugf("Setting up firewall rules...")

	bridgeName := "spitfire0"
	subnet := "172.16.0.0/24"

	firewallRules := []struct {
		description string
		command     []string
	}{
		{
			description: fmt.Sprintf("Allow forwarding from %s", bridgeName),
			command:     []string{"iptables", "-A", "FORWARD", "-i", bridgeName, "-j", "ACCEPT"},
		},
		{
			description: fmt.Sprintf("Allow forwarding to %s", bridgeName),  
			command:     []string{"iptables", "-A", "FORWARD", "-o", bridgeName, "-j", "ACCEPT"},
		},
		{
			description: fmt.Sprintf("Enable NAT for %s", subnet),
			command:     []string{"iptables", "-t", "nat", "-A", "POSTROUTING", "-s", subnet, "!", "-d", subnet, "-j", "MASQUERADE"},
		},
	}

	for _, rule := range firewallRules {
		action := driver.SetupAction{
			Component:   "firewall",
			Description: rule.description,
			Command:     "sudo " + strings.Join(rule.command, " "),
		}

		if !opts.DryRun {
			args := append([]string{rule.command[0]}, rule.command[1:]...)
			cmd := exec.Command("sudo", args...)
			if err := cmd.Run(); err != nil {
				// Don't fail on firewall rules - they might already exist
				action.Success = false
				action.Error = fmt.Sprintf("Rule may already exist: %v", err)
				result.Warnings = append(result.Warnings, fmt.Sprintf("Firewall rule failed (may already exist): %s", rule.description))
			} else {
				action.Success = true
			}
		} else {
			action.Success = true
		}
		result.Actions = append(result.Actions, action)
	}

	return nil
}

// setupDirectories creates required state directories
func (d *Driver) setupDirectories(opts *driver.SetupOptions, result *driver.SetupResult) error {
	log.Debugf("Setting up directories...")

	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	directories := []string{
		filepath.Join(currentUser.HomeDir, ".spitfire"),
		filepath.Join(currentUser.HomeDir, ".spitfire", "vm"),
		filepath.Join(currentUser.HomeDir, ".spitfire", "network"), 
		filepath.Join(currentUser.HomeDir, ".spitfire", "kernels"),
		"/tmp/spitfire",
		"/tmp/spitfire/sockets",
	}

	for _, dir := range directories {
		action := driver.SetupAction{
			Component:   "directories",
			Description: fmt.Sprintf("Creating directory %s", dir),
		}

		if !opts.DryRun {
			if err := os.MkdirAll(dir, 0755); err != nil {
				action.Success = false
				action.Error = fmt.Sprintf("Failed to create directory: %v", err)
				result.Actions = append(result.Actions, action)
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
			action.Success = true
		} else {
			action.Success = true
		}
		result.Actions = append(result.Actions, action)
	}

	return nil
}

// VerifySetup implements driver.Driver for Firecracker setup verification
func (d *Driver) VerifySetup(ctx context.Context) (*driver.SetupStatus, error) {
	status := &driver.SetupStatus{
		Ready:      true,
		Components: []driver.ComponentStatus{},
		Issues:     []driver.SetupIssue{},
	}

	log.Debugf("Verifying Firecracker driver setup...")

	// Verify each component
	components := []func() driver.ComponentStatus{
		d.verifyFirecrackerBinary,
		d.verifyKVMAccess,
		d.verifyNetworking,
		d.verifyDirectories,
		d.verifyIPForwarding,
	}

	for _, verifyFunc := range components {
		compStatus := verifyFunc()
		status.Components = append(status.Components, compStatus)

		// If a required component is not ready, mark overall status as not ready
		if !compStatus.Ready && compStatus.Required {
			status.Ready = false
			// Create issue for this component
			issue := d.createIssueForComponent(compStatus)
			status.Issues = append(status.Issues, issue)
		}
	}

	// Set summary message
	if status.Ready {
		status.Summary = "Firecracker driver is ready to use"
	} else {
		status.Summary = fmt.Sprintf("Firecracker driver has %d setup issues", len(status.Issues))
	}

	return status, nil
}

// verifyFirecrackerBinary checks if Firecracker binary is available
func (d *Driver) verifyFirecrackerBinary() driver.ComponentStatus {
	path, err := exec.LookPath("firecracker")
	if err != nil {
		return driver.ComponentStatus{
			Name:        "firecracker-binary",
			Ready:       false,
			Required:    true,
			Description: "Firecracker hypervisor binary",
			Status:      "Not found in PATH",
		}
	}

	// Check version
	output, err := exec.Command(path, "--version").Output()
	version := "unknown"
	if err == nil {
		version = strings.TrimSpace(string(output))
	}

	return driver.ComponentStatus{
		Name:        "firecracker-binary",
		Ready:       true,
		Required:    true,
		Description: "Firecracker hypervisor binary",
		Status:      fmt.Sprintf("Ready at %s (version: %s)", path, version),
	}
}

// verifyKVMAccess checks KVM device access and permissions
func (d *Driver) verifyKVMAccess() driver.ComponentStatus {
	// Check if /dev/kvm exists
	if _, err := os.Stat("/dev/kvm"); err != nil {
		return driver.ComponentStatus{
			Name:        "kvm-device",
			Ready:       false,
			Required:    true,
			Description: "KVM virtualization device access",
			Status:      "KVM device /dev/kvm not found",
		}
	}

	// Check if user is in kvm group
	currentUser, err := user.Current()
	if err != nil {
		return driver.ComponentStatus{
			Name:        "kvm-permissions",
			Ready:       false,
			Required:    true,
			Description: "User permissions for KVM access",
			Status:      "Unable to determine user information",
		}
	}

	groups, err := currentUser.GroupIds()
	if err != nil {
		return driver.ComponentStatus{
			Name:        "kvm-permissions",
			Ready:       false,
			Required:    true,
			Description: "User permissions for KVM access",
			Status:      "Unable to determine user groups",
		}
	}

	inKVMGroup := false
	for _, gid := range groups {
		if group, err := user.LookupGroupId(gid); err == nil && group.Name == "kvm" {
			inKVMGroup = true
			break
		}
	}

	if !inKVMGroup {
		return driver.ComponentStatus{
			Name:        "kvm-permissions",
			Ready:       false,
			Required:    true,
			Description: "User permissions for KVM access",
			Status:      "User not in kvm group",
		}
	}

	return driver.ComponentStatus{
		Name:        "kvm-permissions",
		Ready:       true,
		Required:    true,
		Description: "User permissions for KVM access",
		Status:      "User is in kvm group",
	}
}

// verifyNetworking checks network bridge setup
func (d *Driver) verifyNetworking() driver.ComponentStatus {
	bridgeName := "spitfire0"
	
	// Check if bridge exists
	iface, err := net.InterfaceByName(bridgeName)
	if err != nil {
		return driver.ComponentStatus{
			Name:        "network-bridge",
			Ready:       false,
			Required:    true,
			Description: "Spitfire network bridge interface",
			Status:      fmt.Sprintf("Bridge %s not found", bridgeName),
		}
	}

	// Check if bridge is up
	isUp := iface.Flags&net.FlagUp != 0

	return driver.ComponentStatus{
		Name:        "network-bridge",
		Ready:       isUp,
		Required:    true,
		Description: "Spitfire network bridge interface",
		Status:      fmt.Sprintf("Bridge %s exists and is %s", bridgeName, map[bool]string{true: "up", false: "down"}[isUp]),
	}
}

// verifyDirectories checks required state directories
func (d *Driver) verifyDirectories() driver.ComponentStatus {
	currentUser, err := user.Current()
	if err != nil {
		return driver.ComponentStatus{
			Name:        "directories",
			Ready:       false,
			Required:    true,
			Description: "Required state directories",
			Status:      "Unable to determine user home directory",
		}
	}

	requiredDirs := []string{
		filepath.Join(currentUser.HomeDir, ".spitfire"),
		"/tmp/spitfire",
		"/tmp/spitfire/sockets",
	}

	missingDirs := []string{}
	for _, dir := range requiredDirs {
		if _, err := os.Stat(dir); err != nil {
			missingDirs = append(missingDirs, dir)
		}
	}

	if len(missingDirs) > 0 {
		return driver.ComponentStatus{
			Name:        "directories",
			Ready:       false,
			Required:    true,
			Description: "Required state directories",
			Status:      fmt.Sprintf("Missing directories: %s", strings.Join(missingDirs, ", ")),
		}
	}

	return driver.ComponentStatus{
		Name:        "directories",
		Ready:       true,
		Required:    true,
		Description: "Required state directories",
		Status:      "All required directories exist",
	}
}

// verifyIPForwarding checks if IP forwarding is enabled
func (d *Driver) verifyIPForwarding() driver.ComponentStatus {
	data, err := os.ReadFile("/proc/sys/net/ipv4/ip_forward")
	if err != nil {
		return driver.ComponentStatus{
			Name:        "ip-forwarding",
			Ready:       false,
			Required:    true,
			Description: "IP forwarding for VM networking",
			Status:      "Unable to check IP forwarding status",
		}
	}

	enabled := strings.TrimSpace(string(data)) == "1"

	return driver.ComponentStatus{
		Name:        "ip-forwarding",
		Ready:       enabled,
		Required:    true,
		Description: "IP forwarding for VM networking",
		Status:      fmt.Sprintf("IP forwarding is %s", map[bool]string{true: "enabled", false: "disabled"}[enabled]),
	}
}

// createIssueForComponent creates a SetupIssue for a failed component
func (d *Driver) createIssueForComponent(component driver.ComponentStatus) driver.SetupIssue {
	issue := driver.SetupIssue{
		Component:   component.Name,
		Description: component.Status,
		Severity:    driver.SeverityError,
	}

	// Provide specific resolution instructions based on component
	switch component.Name {
	case "firecracker-binary":
		issue.Resolution = "Install Firecracker binary"
		issue.Commands = []string{
			"# On Arch Linux:",
			"sudo pacman -S firecracker",
			"# On Ubuntu/Debian:",
			"curl -LOJ https://github.com/firecracker-microvm/firecracker/releases/latest/download/firecracker-v1.4.0-x86_64.tgz",
			"tar xzf firecracker-v1.4.0-x86_64.tgz",
			"sudo cp release-v1.4.0-x86_64/firecracker-v1.4.0-x86_64 /usr/local/bin/firecracker",
		}

	case "kvm-device":
		issue.Resolution = "Enable hardware virtualization in BIOS and load KVM modules"
		issue.Commands = []string{
			"# Load KVM modules",
			"sudo modprobe kvm-intel  # For Intel CPUs",
			"sudo modprobe kvm-amd    # For AMD CPUs", 
		}
		issue.Severity = driver.SeverityCritical

	case "kvm-permissions":
		issue.Resolution = "Add user to kvm group and log out/in"
		issue.Commands = []string{
			"sudo usermod -a -G kvm $USER",
			"# Then log out and back in, or reboot",
		}

	case "network-bridge":
		issue.Resolution = "Create and configure spitfire bridge"
		issue.Commands = []string{
			"sudo ip link add spitfire0 type bridge",
			"sudo ip addr add 172.16.0.1/24 dev spitfire0",
			"sudo ip link set spitfire0 up",
		}

	case "directories":
		issue.Resolution = "Create required state directories"
		issue.Commands = []string{
			"mkdir -p ~/.spitfire/{vm,network,kernels}",
			"sudo mkdir -p /tmp/spitfire/sockets",
		}

	case "ip-forwarding":
		issue.Resolution = "Enable IP forwarding"
		issue.Commands = []string{
			"echo 1 | sudo tee /proc/sys/net/ipv4/ip_forward",
			"# Make permanent:",
			"echo 'net.ipv4.ip_forward = 1' | sudo tee -a /etc/sysctl.conf",
		}
	}

	return issue
}

// GetSetupInstructions implements driver.Driver for Firecracker setup instructions
func (d *Driver) GetSetupInstructions(ctx context.Context) (*driver.SetupInstructions, error) {
	return &driver.SetupInstructions{
		Overview: `The Firecracker driver provides lightweight microVM virtualization using Amazon's Firecracker VMM.
This setup configures KVM permissions, network infrastructure, and security settings required for Firecracker to operate.`,

		Prerequisites: []string{
			"Linux system with KVM support (Intel VT-x or AMD-V required)",
			"Kernel version 4.14+ with KVM modules loaded",
			"At least 1GB available memory for VMs",
			"Root or sudo access for system configuration",
			"Firecracker binary installed (v1.0.0 or later recommended)",
		},

		AutomatedSteps: []driver.SetupStep{
			{
				Name:        "Install Firecracker Binary",
				Description: "Download and install the Firecracker hypervisor binary",
				Commands: []string{
					"curl -LOJ https://github.com/firecracker-microvm/firecracker/releases/latest/download/firecracker-v1.4.1-x86_64.tgz",
					"tar xvf firecracker-v1.4.1-x86_64.tgz",
					"sudo mv release-v1.4.1-x86_64/firecracker-v1.4.1-x86_64 /usr/local/bin/firecracker",
					"sudo chmod +x /usr/local/bin/firecracker",
				},
				Verification: "firecracker --version",
				Required:     true,
				Sudo:         true,
			},
			{
				Name:        "Configure KVM Access",
				Description: "Set up KVM device permissions and add user to kvm group",
				Commands: []string{
					"sudo usermod -aG kvm $USER",
					"sudo chmod 666 /dev/kvm",
					"sudo chown root:kvm /dev/kvm",
				},
				Verification: "ls -la /dev/kvm && groups | grep kvm",
				Required:     true,
				Sudo:         true,
			},
			{
				Name:        "Setup Network Bridge",
				Description: "Create and configure network bridge for VM networking",
				Commands: []string{
					"sudo ip link add spitfire-br type bridge",
					"sudo ip addr add 172.16.0.1/24 dev spitfire-br",
					"sudo ip link set spitfire-br up",
				},
				Verification: "ip link show spitfire-br",
				Required:     true,
				Sudo:         true,
			},
			{
				Name:        "Configure IP Forwarding",
				Description: "Enable IP forwarding for VM internet access",
				Commands: []string{
					"echo 'net.ipv4.ip_forward = 1' | sudo tee -a /etc/sysctl.conf",
					"sudo sysctl -p",
				},
				Verification: "sysctl net.ipv4.ip_forward",
				Required:     true,
				Sudo:         true,
			},
			{
				Name:        "Setup Firewall Rules",
				Description: "Configure iptables for VM network access and NAT",
				Commands: []string{
					"sudo iptables -t nat -A POSTROUTING -s 172.16.0.0/24 -j MASQUERADE",
					"sudo iptables -A FORWARD -i spitfire-br -j ACCEPT",
					"sudo iptables -A FORWARD -o spitfire-br -j ACCEPT",
				},
				Verification: "sudo iptables -t nat -L POSTROUTING -n | grep 172.16.0.0/24",
				Required:     true,
				Sudo:         true,
			},
		},

		ManualSteps: []driver.SetupStep{
			{
				Name:        "Install Rootfs Images",
				Description: "Download and configure rootfs images for your VMs",
				Commands: []string{
					"mkdir -p ~/.spitfire/images",
					"# Download Ubuntu rootfs:",
					"curl -o ~/.spitfire/images/ubuntu.ext4 https://s3.amazonaws.com/spec.ccfc.min/img/hello/fsfiles/hello-rootfs.ext4",
				},
				Verification: "ls -la ~/.spitfire/images/",
				Required:     false,
			},
			{
				Name:        "Configure Persistent Network",
				Description: "Make network configuration persistent across reboots",
				Commands: []string{
					"# Add to /etc/systemd/network/spitfire-br.network:",
					"[Match]",
					"Name=spitfire-br",
					"[Network]",
					"Address=172.16.0.1/24",
					"IPForward=yes",
				},
				Verification: "systemctl status systemd-networkd",
				Required:     false,
			},
		},

		PostSetup: []string{
			"Log out and back in (or reboot) to activate kvm group membership",
			"Test basic functionality with: spitfire driver verify firecracker",
			"Create your first VM with: spitfire up --driver firecracker --image ubuntu",
			"Monitor VM status with: spitfire ps",
		},

		Documentation: []driver.DocLink{
			{
				Title:       "Firecracker Getting Started",
				URL:         "https://github.com/firecracker-microvm/firecracker/blob/main/docs/getting-started.md",
				Description: "Official Firecracker quickstart guide",
			},
			{
				Title:       "Firecracker Network Setup",
				URL:         "https://github.com/firecracker-microvm/firecracker/blob/main/docs/network-setup.md",
				Description: "Detailed network configuration for Firecracker VMs",
			},
			{
				Title:       "KVM Setup Guide",
				URL:         "https://help.ubuntu.com/community/KVM/Installation",
				Description: "Ubuntu KVM installation and configuration",
			},
			{
				Title:       "Spitfire Documentation",
				URL:         "https://github.com/thi-startup/spitfire/docs",
				Description: "Complete Spitfire usage documentation",
			},
		},

		Troubleshooting: []driver.DocLink{
			{
				Title:       "Firecracker Troubleshooting",
				URL:         "https://github.com/firecracker-microvm/firecracker/blob/main/docs/troubleshooting.md",
				Description: "Common Firecracker issues and solutions",
			},
			{
				Title:       "KVM Permission Issues",
				URL:         "https://askubuntu.com/questions/564910/kvm-is-not-installed-on-this-machine-dev-kvm-is-missing",
				Description: "Solving /dev/kvm access problems",
			},
			{
				Title:       "Network Bridge Issues",
				URL:         "https://wiki.archlinux.org/title/Network_bridge",
				Description: "Linux bridge configuration troubleshooting",
			},
		},
	}, nil
}
