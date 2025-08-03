package state

import (
	"time"

	"github.com/thi-startup/spitfire/pkg/config"
	"github.com/thi-startup/spitfire/pkg/driver"
)

// ProjectState represents the complete state of a spitfire project
// It wraps existing config types with runtime state information
type ProjectState struct {
	Name           string                `json:"name"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
	Status         ProjectStatus         `json:"status"`
	ConfigFile     string                `json:"config_file"`     // Path to original config file
	ConfigChecksum string                `json:"config_checksum"` // Hash of original config
	ResolvedConfig *config.Config        `json:"resolved_config"` // Reuses existing config.Config
	VMs            map[string]*VMRuntime `json:"vms"`             // VM runtime state
	Environment    map[string]string     `json:"environment"`     // Project environment vars
}

// ProjectStatus represents the current state of a project
type ProjectStatus string

const (
	ProjectStatusUnknown  ProjectStatus = "unknown"
	ProjectStatusStopped  ProjectStatus = "stopped"
	ProjectStatusStarting ProjectStatus = "starting"
	ProjectStatusRunning  ProjectStatus = "running"
	ProjectStatusPaused   ProjectStatus = "paused"
	ProjectStatusError    ProjectStatus = "error"
)

// VMRuntime wraps driver.VMInfo with additional state management
type VMRuntime struct {
	// Reuse existing driver types
	driver.VMInfo // Embeds: ID, Name, State, IP, Created, Driver, Metadata

	// Add driver configuration separately (not part of VMInfo)
	DriverConfig *driver.Config `json:"driver_config"`

	// Add state management fields
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	StoppedAt    *time.Time             `json:"stopped_at,omitempty"`
	PID          int                    `json:"pid,omitempty"`
	LastError    string                 `json:"last_error,omitempty"`
	RestartCount int                    `json:"restart_count"`
	DriverState  map[string]interface{} `json:"driver_state,omitempty"` // Driver-specific runtime state
}

// Event represents an operation event in the project
type Event struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Type      EventType              `json:"type"`
	Resource  string                 `json:"resource"` // VM name, network name, etc.
	Action    string                 `json:"action"`   // start, stop, create, delete
	Status    EventStatus            `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Duration  time.Duration          `json:"duration,omitempty"`
	User      string                 `json:"user,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// EventType categorizes different types of events
type EventType string

const (
	EventTypeProject EventType = "project"
	EventTypeVM      EventType = "vm"
	EventTypeNetwork EventType = "network"
	EventTypeVolume  EventType = "volume"
	EventTypeDriver  EventType = "driver"
	EventTypeSystem  EventType = "system"
)

// EventStatus represents the outcome of an event
type EventStatus string

const (
	EventStatusStarted   EventStatus = "started"
	EventStatusSuccess   EventStatus = "success"
	EventStatusFailed    EventStatus = "failed"
	EventStatusCancelled EventStatus = "cancelled"
)

// GlobalConfig represents global spitfire configuration
type GlobalConfig struct {
	DefaultDriver    string                            `json:"default_driver,omitempty"`
	StateDir         string                            `json:"state_dir,omitempty"`
	LogLevel         string                            `json:"log_level,omitempty"`
	CheckForUpdates  bool                              `json:"check_for_updates"`
	TelemetryEnabled bool                              `json:"telemetry_enabled"`
	DriverDefaults   map[string]map[string]interface{} `json:"driver_defaults,omitempty"`
	User             string                            `json:"user,omitempty"`
	MaxEventHistory  int                               `json:"max_event_history"`
	ConfigVersion    string                            `json:"config_version"`
}

// RuntimeInfo provides additional runtime state for resources
// This can be embedded or used as needed for networks and volumes
type RuntimeInfo struct {
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	State       map[string]interface{} `json:"state,omitempty"`        // Driver-specific runtime state
	AttachedVMs []string               `json:"attached_vms,omitempty"` // VMs using this resource
}
