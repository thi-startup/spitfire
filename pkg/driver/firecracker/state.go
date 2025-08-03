package firecracker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// VMState manages the state of a Firecracker VM
type VMState struct {
	vmID      string
	stateRoot string
}

// NewVMState creates a new VM state manager
func NewVMState(vmID, stateRoot string) *VMState {
	return &VMState{
		vmID:      vmID,
		stateRoot: filepath.Join(stateRoot, vmID),
	}
}

// Root returns the root directory for this VM's state
func (s *VMState) Root() string {
	return s.stateRoot
}

// EnsureStateDir creates the state directory if it doesn't exist
func (s *VMState) EnsureStateDir() error {
	if err := os.MkdirAll(s.stateRoot, 0755); err != nil {
		return err
	}

	// Also ensure log directory exists (in case logger paths are in subdirectories)
	logDir := filepath.Dir(s.LogPath())
	if logDir != s.stateRoot {
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return err
		}
	}

	return nil
}

// ConfigPath returns the path to the Firecracker config file
func (s *VMState) ConfigPath() string {
	return filepath.Join(s.stateRoot, "firecracker.cfg")
}

// SetConfig saves the Firecracker configuration to disk
func (s *VMState) SetConfig(config *FirecrackerConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(s.ConfigPath(), data, 0644)
}

// GetConfig loads the Firecracker configuration from disk
func (s *VMState) GetConfig() (*FirecrackerConfig, error) {
	data, err := os.ReadFile(s.ConfigPath())
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var config FirecrackerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	return &config, nil
}

// PIDPath returns the path to the PID file
func (s *VMState) PIDPath() string {
	return filepath.Join(s.stateRoot, "firecracker.pid")
}

// SetPID saves the process ID to disk
func (s *VMState) SetPID(pid int) error {
	pidStr := strconv.Itoa(pid)
	return os.WriteFile(s.PIDPath(), []byte(pidStr), 0644)
}

// GetPID reads the process ID from disk
func (s *VMState) GetPID() (int, error) {
	data, err := os.ReadFile(s.PIDPath())
	if err != nil {
		return 0, fmt.Errorf("reading PID file: %w", err)
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, fmt.Errorf("parsing PID: %w", err)
	}

	return pid, nil
}

// LogPath returns the path to the VM log file
func (s *VMState) LogPath() string {
	return filepath.Join(s.stateRoot, "firecracker.log")
}

// MetricsPath returns the path to the metrics file
func (s *VMState) MetricsPath() string {
	return filepath.Join(s.stateRoot, "firecracker.metrics")
}

// StdoutPath returns the path to the stdout file
func (s *VMState) StdoutPath() string {
	return filepath.Join(s.stateRoot, "firecracker.stdout")
}

// StderrPath returns the path to the stderr file
func (s *VMState) StderrPath() string {
	return filepath.Join(s.stateRoot, "firecracker.stderr")
}

// SocketPath returns the path to the Firecracker API socket
func (s *VMState) SocketPath() string {
	return filepath.Join(s.stateRoot, "firecracker.sock")
}

// MetadataPath returns the path to the metadata file
func (s *VMState) MetadataPath() string {
	return filepath.Join(s.stateRoot, "metadata.json")
}

// SetMetadata saves metadata to disk
func (s *VMState) SetMetadata(metadata *Metadata) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}

	return os.WriteFile(s.MetadataPath(), data, 0644)
}

// GetMetadata loads metadata from disk
func (s *VMState) GetMetadata() (*Metadata, error) {
	data, err := os.ReadFile(s.MetadataPath())
	if err != nil {
		return nil, fmt.Errorf("reading metadata file: %w", err)
	}

	var metadata Metadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("unmarshaling metadata: %w", err)
	}

	return &metadata, nil
}

// Cleanup removes all state files for this VM
func (s *VMState) Cleanup() error {
	return os.RemoveAll(s.stateRoot)
}

// Exists checks if the VM state directory exists
func (s *VMState) Exists() bool {
	_, err := os.Stat(s.stateRoot)
	return err == nil
}

// IsRunning checks if the VM process is still running
func (s *VMState) IsRunning() (bool, error) {
	pid, err := s.GetPID()
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // PID file doesn't exist
		}
		return false, err
	}

	// Check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false, nil // Process doesn't exist
	}

	// On Unix systems, Signal(0) can be used to check if process exists
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		return false, nil // Process doesn't exist or can't be signaled
	}

	return true, nil
}

// GetAllFiles returns a list of all state files for this VM
func (s *VMState) GetAllFiles() []string {
	return []string{
		s.ConfigPath(),
		s.PIDPath(),
		s.LogPath(),
		s.MetricsPath(),
		s.StdoutPath(),
		s.StderrPath(),
		s.SocketPath(),
		s.MetadataPath(),
	}
}
