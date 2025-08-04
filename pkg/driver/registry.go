package driver

import (
	"fmt"
	"sort"
	"sync"
)

// DriverFactory creates a new driver instance with the given configuration
type DriverFactory func(*Config) (Driver, error)

// StatusChecker checks if a driver is available and healthy
type StatusChecker func() State

// DriverDef defines how to initialize and check a driver
type DriverDef struct {
	Name        string
	Aliases     []string
	Create      DriverFactory
	Status      StatusChecker
	Priority    Priority
	Default     bool
	Description string
}

// Empty returns true if the driver definition is empty
func (d DriverDef) Empty() bool {
	return d.Name == ""
}

func (d DriverDef) String() string {
	return d.Name
}

// DriverState combines driver definition with current state
type DriverState struct {
	DriverDef
	State      State
	Rejection  string
	Suggestion string
}

// Registry manages available drivers
type Registry interface {
	Register(def DriverDef) error
	Driver(name string) DriverDef
	List() []DriverDef
	Available() []DriverState
	Status(name string) State
}

type driverRegistry struct {
	drivers        map[string]DriverDef
	driversByAlias map[string]DriverDef
	mutex          sync.RWMutex
}

// NewRegistry creates a new driver registry
func NewRegistry() Registry {
	return &driverRegistry{
		drivers:        make(map[string]DriverDef),
		driversByAlias: make(map[string]DriverDef),
	}
}

// Register registers a driver definition
func (r *driverRegistry) Register(def DriverDef) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if def.Name == "" {
		return fmt.Errorf("driver name cannot be empty")
	}

	if def.Create == nil {
		return fmt.Errorf("driver %q must have a Create function", def.Name)
	}

	if def.Status == nil {
		return fmt.Errorf("driver %q must have a Status function", def.Name)
	}

	if _, exists := r.drivers[def.Name]; exists {
		return fmt.Errorf("driver %q is already registered", def.Name)
	}

	r.drivers[def.Name] = def
	for _, alias := range def.Aliases {
		if _, exists := r.driversByAlias[alias]; exists {
			return fmt.Errorf("alias %q is already registered", alias)
		}
		r.driversByAlias[alias] = def
	}

	return nil
}

// Driver returns a driver definition by name or alias
func (r *driverRegistry) Driver(name string) DriverDef {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if def, exists := r.drivers[name]; exists {
		return def
	}

	if def, exists := r.driversByAlias[name]; exists {
		return def
	}

	return DriverDef{}
}

// List returns all registered drivers
func (r *driverRegistry) List() []DriverDef {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make([]DriverDef, 0, len(r.drivers))
	for _, def := range r.drivers {
		result = append(result, def)
	}

	return result
}

// Available returns all drivers with their current state
func (r *driverRegistry) Available() []DriverState {
	defs := r.List()
	states := make([]DriverState, len(defs))

	for i, def := range defs {
		states[i] = DriverState{
			DriverDef: def,
			State:     r.Status(def.Name),
		}
	}

	// Sort by priority (highest first)
	sort.Slice(states, func(i, j int) bool {
		return states[i].Priority > states[j].Priority
	})

	return states
}

// Status returns the current status of a driver
func (r *driverRegistry) Status(name string) State {
	def := r.Driver(name)
	if def.Empty() {
		return State{
			Error:  fmt.Errorf("driver %q not found", name),
			Reason: "DRIVER_NOT_FOUND",
		}
	}

	if def.Status == nil {
		return State{
			Error:  fmt.Errorf("driver %q has no status checker", name),
			Reason: "DRIVER_NO_STATUS_CHECKER",
		}
	}

	return def.Status()
}

// Global registry instance
var globalRegistry Registry = NewRegistry()

// Register registers a driver in the global registry
func Register(def DriverDef) error {
	return globalRegistry.Register(def)
}

// GetDriver returns a driver from the global registry
func GetDriver(name string) DriverDef {
	return globalRegistry.Driver(name)
}

// List returns all drivers from the global registry
func List() []DriverDef {
	return globalRegistry.List()
}

// Available returns all available drivers from the global registry
func Available() []DriverState {
	return globalRegistry.Available()
}

// Status returns driver status from the global registry
func Status(name string) State {
	return globalRegistry.Status(name)
}

// Supported returns true if the driver is registered
func Supported(name string) bool {
	return !GetDriver(name).Empty()
}

// SupportedDrivers returns a list of supported driver names
func SupportedDrivers() []string {
	defs := List()
	names := make([]string, len(defs))
	for i, def := range defs {
		names[i] = def.Name
	}
	return names
}

// IsAlias checks if an alias belongs to a driver
func IsAlias(driverName, alias string) bool {
	def := GetDriver(driverName)
	if def.Empty() {
		return false
	}

	for _, a := range def.Aliases {
		if a == alias {
			return true
		}
	}
	return false
}
