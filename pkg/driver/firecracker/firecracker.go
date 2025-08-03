package firecracker

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
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
