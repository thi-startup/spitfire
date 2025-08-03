package state

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/thi-startup/spitfire/pkg/config"
	"github.com/thi-startup/spitfire/pkg/driver"
)

// Manager handles project state persistence and operations
type Manager struct {
	projectName string
}

// NewManager creates a new state manager for a project
func NewManager(projectName string) *Manager {
	return &Manager{
		projectName: projectName,
	}
}

// InitializeProject creates initial project state
func (m *Manager) InitializeProject(configFile string, resolvedConfig *config.Config) (*ProjectState, error) {
	if err := EnsureProjectDirs(m.projectName); err != nil {
		return nil, fmt.Errorf("failed to create project directories: %w", err)
	}

	checksum, err := calculateFileChecksum(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate config checksum: %w", err)
	}

	now := time.Now()
	state := &ProjectState{
		Name:           m.projectName,
		CreatedAt:      now,
		UpdatedAt:      now,
		Status:         ProjectStatusStopped,
		ConfigFile:     configFile,
		ConfigChecksum: checksum,
		ResolvedConfig: resolvedConfig,
		VMs:            make(map[string]*VMRuntime),
		Environment:    make(map[string]string),
	}

	// Initialize VM runtime states from config
	for vmName, vmConfig := range resolvedConfig.VMs {
		driverConfig, err := vmConfig.ToDriverConfig(vmName, resolvedConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create driver config for VM %s: %w", vmName, err)
		}

		vmRuntime := &VMRuntime{
			VMInfo: driver.VMInfo{
				Name:    vmName,
				State:   driver.Stopped,
				Driver:  vmConfig.GetEffectiveDriver(resolvedConfig.Driver),
				Created: now,
			},
			DriverConfig: driverConfig,
			CreatedAt:    now,
			UpdatedAt:    now,
			RestartCount: 0,
			DriverState:  make(map[string]interface{}),
		}
		state.VMs[vmName] = vmRuntime
	}

	return state, m.SaveProjectState(state)
}

// LoadProjectState loads project state from disk
func (m *Manager) LoadProjectState() (*ProjectState, error) {
	statePath := ProjectStateFile(m.projectName)

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("project %s does not exist", m.projectName)
		}
		return nil, fmt.Errorf("failed to read project state: %w", err)
	}

	var state ProjectState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse project state: %w", err)
	}

	return &state, nil
}

// SaveProjectState saves project state to disk atomically
func (m *Manager) SaveProjectState(state *ProjectState) error {
	state.UpdatedAt = time.Now()

	statePath := ProjectStateFile(m.projectName)
	return writeJSONAtomic(statePath, state)
}

// UpdateVMState updates the state of a specific VM
func (m *Manager) UpdateVMState(vmName string, updateFn func(*VMRuntime) error) error {
	state, err := m.LoadProjectState()
	if err != nil {
		return err
	}

	vmRuntime, exists := state.VMs[vmName]
	if !exists {
		return fmt.Errorf("VM %s not found in project", vmName)
	}

	if err := updateFn(vmRuntime); err != nil {
		return err
	}

	vmRuntime.UpdatedAt = time.Now()
	return m.SaveProjectState(state)
}

// UpdateProjectStatus updates the overall project status
func (m *Manager) UpdateProjectStatus(status ProjectStatus) error {
	state, err := m.LoadProjectState()
	if err != nil {
		return err
	}

	state.Status = status
	return m.SaveProjectState(state)
}

// RecordEvent records an event in the project's event log
func (m *Manager) RecordEvent(event *Event) error {
	eventsPath := ProjectEventsFile(m.projectName)

	// Load existing events
	var events []Event
	if data, err := os.ReadFile(eventsPath); err == nil {
		json.Unmarshal(data, &events)
	}

	// Add new event
	events = append(events, *event)

	// Keep only recent events (configurable limit)
	maxEvents := 1000 // TODO: make configurable
	if len(events) > maxEvents {
		events = events[len(events)-maxEvents:]
	}

	return writeJSONAtomic(eventsPath, events)
}

// GetProjectEvents returns recent events for the project
func (m *Manager) GetProjectEvents(limit int) ([]Event, error) {
	eventsPath := ProjectEventsFile(m.projectName)

	data, err := os.ReadFile(eventsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Event{}, nil
		}
		return nil, err
	}

	var events []Event
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, err
	}

	// Return most recent events up to limit
	if limit > 0 && len(events) > limit {
		events = events[len(events)-limit:]
	}

	return events, nil
}

// ProjectExists checks if a project exists
func (m *Manager) ProjectExists() bool {
	statePath := ProjectStateFile(m.projectName)
	_, err := os.Stat(statePath)
	return err == nil
}

// DeleteProject removes all project state and data
func (m *Manager) DeleteProject() error {
	projectPath := ProjectPath(m.projectName)
	return os.RemoveAll(projectPath)
}

// ListProjects returns all existing projects
func ListProjects() ([]string, error) {
	projectsPath := MakeSpitfirePath("projects")

	entries, err := os.ReadDir(projectsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var projects []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Verify it's a valid project by checking for state file
			statePath := ProjectStateFile(entry.Name())
			if _, err := os.Stat(statePath); err == nil {
				projects = append(projects, entry.Name())
			}
		}
	}

	return projects, nil
}

// Global config management

// LoadGlobalConfig loads global spitfire configuration
func LoadGlobalConfig() (*GlobalConfig, error) {
	configPath := GlobalConfigFile()

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config
			return &GlobalConfig{
				CheckForUpdates:  true,
				TelemetryEnabled: true,
				MaxEventHistory:  1000,
				ConfigVersion:    "1",
			}, nil
		}
		return nil, err
	}

	var config GlobalConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveGlobalConfig saves global spitfire configuration
func SaveGlobalConfig(config *GlobalConfig) error {
	if err := EnsureGlobalDirs(); err != nil {
		return err
	}

	configPath := GlobalConfigFile()
	return writeJSONAtomic(configPath, config)
}

// Utility functions

// writeJSONAtomic writes JSON data to a file atomically using temp file + rename
func writeJSONAtomic(path string, data interface{}) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	// Marshal to JSON with indentation
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	// Write to temporary file
	tempFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tempFile.Name()) // Clean up temp file if we fail

	if _, err := tempFile.Write(jsonData); err != nil {
		tempFile.Close()
		return err
	}

	if err := tempFile.Close(); err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tempFile.Name(), path)
}

// calculateFileChecksum calculates SHA256 checksum of a file
func calculateFileChecksum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
