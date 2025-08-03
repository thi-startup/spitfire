package driver

import (
	"testing"
)

func TestRegistry(t *testing.T) {
	// Create a new registry for testing
	registry := NewRegistry()

	// Test driver definition
	mockDef := DriverDef{
		Name:        "test-driver",
		Aliases:     []string{"test", "mock"},
		Create:      func(*Config) (Driver, error) { return nil, nil },
		Status:      func() State { return State{Installed: true, Healthy: true} },
		Priority:    Default,
		Default:     true,
		Description: "Test driver",
	}

	t.Run("Register", func(t *testing.T) {
		err := registry.Register(mockDef)
		if err != nil {
			t.Fatalf("Failed to register driver: %v", err)
		}

		// Try to register the same driver again
		err = registry.Register(mockDef)
		if err == nil {
			t.Error("Expected error when registering duplicate driver")
		}
	})

	t.Run("Register with empty name", func(t *testing.T) {
		emptyDef := DriverDef{}
		err := registry.Register(emptyDef)
		if err == nil {
			t.Error("Expected error when registering driver with empty name")
		}
	})

	t.Run("Register without Create function", func(t *testing.T) {
		invalidDef := DriverDef{
			Name:   "invalid",
			Status: func() State { return State{} },
		}
		err := registry.Register(invalidDef)
		if err == nil {
			t.Error("Expected error when registering driver without Create function")
		}
	})

	t.Run("Register without Status function", func(t *testing.T) {
		invalidDef := DriverDef{
			Name:   "invalid2",
			Create: func(*Config) (Driver, error) { return nil, nil },
		}
		err := registry.Register(invalidDef)
		if err == nil {
			t.Error("Expected error when registering driver without Status function")
		}
	})

	t.Run("Driver", func(t *testing.T) {
		// Test getting driver by name
		def := registry.Driver("test-driver")
		if def.Empty() {
			t.Error("Expected to find registered driver")
		}
		if def.Name != "test-driver" {
			t.Errorf("Expected driver name 'test-driver', got '%s'", def.Name)
		}

		// Test getting driver by alias
		def = registry.Driver("test")
		if def.Empty() {
			t.Error("Expected to find driver by alias")
		}
		if def.Name != "test-driver" {
			t.Errorf("Expected driver name 'test-driver', got '%s'", def.Name)
		}

		def = registry.Driver("mock")
		if def.Empty() {
			t.Error("Expected to find driver by alias")
		}

		// Test getting non-existent driver
		def = registry.Driver("non-existent")
		if !def.Empty() {
			t.Error("Expected empty driver for non-existent name")
		}
	})

	t.Run("List", func(t *testing.T) {
		drivers := registry.List()
		if len(drivers) != 1 {
			t.Errorf("Expected 1 driver, got %d", len(drivers))
		}
		if drivers[0].Name != "test-driver" {
			t.Errorf("Expected driver name 'test-driver', got '%s'", drivers[0].Name)
		}
	})

	t.Run("Available", func(t *testing.T) {
		states := registry.Available()
		if len(states) != 1 {
			t.Errorf("Expected 1 driver state, got %d", len(states))
		}

		state := states[0]
		if state.Name != "test-driver" {
			t.Errorf("Expected driver name 'test-driver', got '%s'", state.Name)
		}
		if !state.State.Installed {
			t.Error("Expected driver to be installed")
		}
		if !state.State.Healthy {
			t.Error("Expected driver to be healthy")
		}
	})

	t.Run("Status", func(t *testing.T) {
		state := registry.Status("test-driver")
		if !state.Installed {
			t.Error("Expected driver to be installed")
		}
		if !state.Healthy {
			t.Error("Expected driver to be healthy")
		}

		// Test status for non-existent driver
		state = registry.Status("non-existent")
		if state.Error == nil {
			t.Error("Expected error for non-existent driver")
		}
	})

	t.Run("Aliases conflict", func(t *testing.T) {
		conflictDef := DriverDef{
			Name:    "conflict-driver",
			Aliases: []string{"test"}, // This alias is already used
			Create:  func(*Config) (Driver, error) { return nil, nil },
			Status:  func() State { return State{} },
		}
		err := registry.Register(conflictDef)
		if err == nil {
			t.Error("Expected error when registering driver with conflicting alias")
		}
	})
}

func TestGlobalRegistry(t *testing.T) {
	// Test that global functions work
	originalDrivers := List()

	testDef := DriverDef{
		Name:        "global-test",
		Create:      func(*Config) (Driver, error) { return nil, nil },
		Status:      func() State { return State{Installed: true, Healthy: true} },
		Priority:    Default,
		Default:     true,
		Description: "Global test driver",
	}

	err := Register(testDef)
	if err != nil {
		t.Fatalf("Failed to register driver globally: %v", err)
	}

	newDrivers := List()
	if len(newDrivers) != len(originalDrivers)+1 {
		t.Errorf("Expected %d drivers after registration, got %d", len(originalDrivers)+1, len(newDrivers))
	}

	def := GetDriver("global-test")
	if def.Empty() {
		t.Error("Expected to find globally registered driver")
	}

	if !Supported("global-test") {
		t.Error("Expected global-test to be supported")
	}

	if Supported("non-existent") {
		t.Error("Expected non-existent driver to not be supported")
	}

	supportedNames := SupportedDrivers()
	found := false
	for _, name := range supportedNames {
		if name == "global-test" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected global-test to be in supported drivers list")
	}
}

func TestPriorityOrdering(t *testing.T) {
	registry := NewRegistry()

	// Register drivers with different priorities
	drivers := []DriverDef{
		{
			Name:     "low-priority",
			Create:   func(*Config) (Driver, error) { return nil, nil },
			Status:   func() State { return State{Installed: true, Healthy: true} },
			Priority: Fallback,
		},
		{
			Name:     "high-priority",
			Create:   func(*Config) (Driver, error) { return nil, nil },
			Status:   func() State { return State{Installed: true, Healthy: true} },
			Priority: HighlyPreferred,
		},
		{
			Name:     "medium-priority",
			Create:   func(*Config) (Driver, error) { return nil, nil },
			Status:   func() State { return State{Installed: true, Healthy: true} },
			Priority: Preferred,
		},
	}

	for _, def := range drivers {
		err := registry.Register(def)
		if err != nil {
			t.Fatalf("Failed to register driver %s: %v", def.Name, err)
		}
	}

	available := registry.Available()
	if len(available) != 3 {
		t.Fatalf("Expected 3 drivers, got %d", len(available))
	}

	// Check that they're ordered by priority (highest first)
	expectedOrder := []string{"high-priority", "medium-priority", "low-priority"}
	for i, expected := range expectedOrder {
		if available[i].Name != expected {
			t.Errorf("Expected driver %d to be '%s', got '%s'", i, expected, available[i].Name)
		}
	}
}

func TestDriverDefValidation(t *testing.T) {
	def := DriverDef{}
	if !def.Empty() {
		t.Error("Expected empty driver def to be empty")
	}

	def.Name = "test"
	if def.Empty() {
		t.Error("Expected non-empty driver def to not be empty")
	}

	str := def.String()
	if str != "test" {
		t.Errorf("Expected string representation to be 'test', got '%s'", str)
	}
}
