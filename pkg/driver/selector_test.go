package driver

import (
	"fmt"
	"testing"
)

func TestSelector(t *testing.T) {
	// Create a test registry
	registry := NewRegistry()

	// Register test drivers with different states
	drivers := []DriverDef{
		{
			Name:     "healthy-preferred",
			Create:   func(*Config) (Driver, error) { return nil, nil },
			Status:   func() State { return State{Installed: true, Healthy: true} },
			Priority: Preferred,
			Default:  true,
		},
		{
			Name:     "healthy-default",
			Create:   func(*Config) (Driver, error) { return nil, nil },
			Status:   func() State { return State{Installed: true, Healthy: true} },
			Priority: Default,
			Default:  true,
		},
		{
			Name:     "unhealthy",
			Create:   func(*Config) (Driver, error) { return nil, nil },
			Status:   func() State { return State{Installed: true, Healthy: false, Error: fmt.Errorf("test error")} },
			Priority: HighlyPreferred,
			Default:  true,
		},
		{
			Name:     "not-installed",
			Create:   func(*Config) (Driver, error) { return nil, nil },
			Status:   func() State { return State{Installed: false, Healthy: false} },
			Priority: Preferred,
			Default:  true,
		},
		{
			Name:     "experimental",
			Create:   func(*Config) (Driver, error) { return nil, nil },
			Status:   func() State { return State{Installed: true, Healthy: true} },
			Priority: Experimental,
			Default:  false,
		},
	}

	for _, def := range drivers {
		err := registry.Register(def)
		if err != nil {
			t.Fatalf("Failed to register driver %s: %v", def.Name, err)
		}
	}

	// Replace global registry for testing
	originalRegistry := globalRegistry
	globalRegistry = registry
	defer func() { globalRegistry = originalRegistry }()

	t.Run("Basic selection", func(t *testing.T) {
		options := DefaultSelectorOptions()
		pick, alternatives, rejects := Suggest(options)

		if pick.Name != "healthy-preferred" {
			t.Errorf("Expected 'healthy-preferred' to be picked, got '%s'", pick.Name)
		}

		// Check that healthy-default is in alternatives
		found := false
		for _, alt := range alternatives {
			if alt.Name == "healthy-default" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected 'healthy-default' to be in alternatives")
		}

		// Check that unhealthy drivers are rejected
		foundUnhealthy := false
		foundNotInstalled := false
		for _, rej := range rejects {
			if rej.Name == "unhealthy" {
				foundUnhealthy = true
			}
			if rej.Name == "not-installed" {
				foundNotInstalled = true
			}
		}
		if !foundUnhealthy {
			t.Error("Expected 'unhealthy' to be in rejects")
		}
		if !foundNotInstalled {
			t.Error("Expected 'not-installed' to be in rejects")
		}
	})

	t.Run("Preferred driver selection", func(t *testing.T) {
		options := DefaultSelectorOptions()
		options.PreferredDriver = "healthy-default"

		pick, _, _ := Suggest(options)
		if pick.Name != "healthy-default" {
			t.Errorf("Expected preferred driver 'healthy-default' to be picked, got '%s'", pick.Name)
		}
	})

	t.Run("Preferred driver unhealthy", func(t *testing.T) {
		options := DefaultSelectorOptions()
		options.PreferredDriver = "unhealthy"

		pick, _, rejects := Suggest(options)

		// Should fall back to best available
		if pick.Name != "healthy-preferred" {
			t.Errorf("Expected fallback to 'healthy-preferred', got '%s'", pick.Name)
		}

		// Check that preferred unhealthy driver is in rejects
		found := false
		for _, rej := range rejects {
			if rej.Name == "unhealthy" {
				found = true
				if rej.Rejection == "" {
					t.Error("Expected rejection reason for unhealthy preferred driver")
				}
				break
			}
		}
		if !found {
			t.Error("Expected unhealthy preferred driver to be in rejects")
		}
	})

	t.Run("Require default drivers", func(t *testing.T) {
		options := DefaultSelectorOptions()
		options.RequireDefault = true

		pick, _, rejects := Suggest(options)

		if pick.Name != "healthy-preferred" {
			t.Errorf("Expected 'healthy-preferred' to be picked, got '%s'", pick.Name)
		}

		// Check that experimental (non-default) is rejected
		found := false
		for _, rej := range rejects {
			if rej.Name == "experimental" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected 'experimental' to be rejected when requiring default")
		}
	})

	t.Run("Allow unhealthy drivers", func(t *testing.T) {
		options := DefaultSelectorOptions()
		options.RequireHealthy = false

		pick, _, _ := Suggest(options)

		// Should pick the highest priority, even if unhealthy
		if pick.Name != "unhealthy" {
			t.Errorf("Expected 'unhealthy' to be picked when allowing unhealthy, got '%s'", pick.Name)
		}
	})

	t.Run("Minimum priority filter", func(t *testing.T) {
		options := DefaultSelectorOptions()
		options.MinPriority = Default

		pick, _, rejects := Suggest(options)

		if pick.Name != "healthy-preferred" {
			t.Errorf("Expected 'healthy-preferred' to be picked, got '%s'", pick.Name)
		}

		// Check that experimental is rejected due to low priority
		found := false
		for _, rej := range rejects {
			if rej.Name == "experimental" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected 'experimental' to be rejected due to low priority")
		}
	})
}

func TestSelectorHelpers(t *testing.T) {
	registry := NewRegistry()

	// Register a test driver
	testDef := DriverDef{
		Name:        "test-helper",
		Create:      func(*Config) (Driver, error) { return nil, nil },
		Status:      func() State { return State{Installed: true, Healthy: true} },
		Priority:    Experimental,
		Default:     false,
		Description: "Test driver for helpers",
	}

	err := registry.Register(testDef)
	if err != nil {
		t.Fatalf("Failed to register test driver: %v", err)
	}

	// Replace global registry for testing
	originalRegistry := globalRegistry
	globalRegistry = registry
	defer func() { globalRegistry = originalRegistry }()

	t.Run("DisplaySupportedDrivers", func(t *testing.T) {
		display := DisplaySupportedDrivers()
		if display != "test-helper (experimental)" {
			t.Errorf("Expected 'test-helper (experimental)', got '%s'", display)
		}
	})

	t.Run("GetDriverChoices", func(t *testing.T) {
		choices := GetDriverChoices()
		if len(choices) != 1 {
			t.Errorf("Expected 1 choice, got %d", len(choices))
		}
		if choices[0].Name != "test-helper" {
			t.Errorf("Expected 'test-helper', got '%s'", choices[0].Name)
		}
	})
}

func TestIsDriverSuitable(t *testing.T) {
	tests := []struct {
		name     string
		ds       DriverState
		options  SelectorOptions
		expected bool
	}{
		{
			name: "healthy and installed",
			ds: DriverState{
				DriverDef: DriverDef{Priority: Default, Default: true},
				State:     State{Installed: true, Healthy: true},
			},
			options:  DefaultSelectorOptions(),
			expected: true,
		},
		{
			name: "not installed",
			ds: DriverState{
				DriverDef: DriverDef{Priority: Default, Default: true},
				State:     State{Installed: false, Healthy: true},
			},
			options:  DefaultSelectorOptions(),
			expected: false,
		},
		{
			name: "not healthy",
			ds: DriverState{
				DriverDef: DriverDef{Priority: Default, Default: true},
				State:     State{Installed: true, Healthy: false},
			},
			options:  DefaultSelectorOptions(),
			expected: false,
		},
		{
			name: "not default when required",
			ds: DriverState{
				DriverDef: DriverDef{Priority: Default, Default: false},
				State:     State{Installed: true, Healthy: true},
			},
			options:  SelectorOptions{RequireDefault: true, RequireHealthy: true, MinPriority: Experimental},
			expected: false,
		},
		{
			name: "low priority",
			ds: DriverState{
				DriverDef: DriverDef{Priority: Experimental, Default: true},
				State:     State{Installed: true, Healthy: true},
			},
			options:  SelectorOptions{RequireDefault: false, RequireHealthy: true, MinPriority: Default},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isDriverSuitable(tt.ds, tt.options)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsInSlice(t *testing.T) {
	slice := []string{"a", "b", "c"}

	if !isInSlice("b", slice) {
		t.Error("Expected 'b' to be in slice")
	}

	if isInSlice("d", slice) {
		t.Error("Expected 'd' to not be in slice")
	}

	if isInSlice("a", []string{}) {
		t.Error("Expected 'a' to not be in empty slice")
	}
}
