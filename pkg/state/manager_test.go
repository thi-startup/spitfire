package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/thi-startup/spitfire/pkg/config"
	"github.com/thi-startup/spitfire/pkg/driver"
)

func TestStateManager(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "spitfire-state-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Override the spitfire home for testing
	originalHome := os.Getenv(SpitfireHome)
	os.Setenv(SpitfireHome, tempDir)
	defer os.Setenv(SpitfireHome, originalHome)

	projectName := "test-project"
	manager := NewManager(projectName)

	// Create a test config
	testConfig := &config.Config{
		Version: "1",
		Driver:  "firecracker",
		VMs: map[string]*config.VM{
			"web": {
				Image: "nginx:alpine",
				Resources: &config.Resources{
					VCPU:   1,
					Memory: "512MB",
				},
			},
			"api": {
				Image: "app:latest",
				Resources: &config.Resources{
					VCPU:   2,
					Memory: "1GB",
				},
			},
		},
	}

	// Test project initialization
	t.Run("InitializeProject", func(t *testing.T) {
		// Create a fake config file
		configPath := filepath.Join(tempDir, "spitfire.yml")
		configData := []byte("version: '1'\nvms:\n  web:\n    image: nginx:alpine")
		if err := os.WriteFile(configPath, configData, 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		state, err := manager.InitializeProject(configPath, testConfig)
		if err != nil {
			t.Fatalf("Failed to initialize project: %v", err)
		}

		// Verify project state
		if state.Name != projectName {
			t.Errorf("Expected project name %s, got %s", projectName, state.Name)
		}

		if state.Status != ProjectStatusStopped {
			t.Errorf("Expected status %s, got %s", ProjectStatusStopped, state.Status)
		}

		if len(state.VMs) != 2 {
			t.Errorf("Expected 2 VMs, got %d", len(state.VMs))
		}

		// Verify VM states
		webVM := state.VMs["web"]
		if webVM == nil {
			t.Fatal("Web VM not found")
		}

		if webVM.Name != "web" {
			t.Errorf("Expected VM name 'web', got %s", webVM.Name)
		}

		if webVM.State != driver.Stopped {
			t.Errorf("Expected VM state %s, got %s", driver.Stopped, webVM.State)
		}

		if webVM.Driver != "firecracker" {
			t.Errorf("Expected driver 'firecracker', got %s", webVM.Driver)
		}

		// Verify state file was created
		if !manager.ProjectExists() {
			t.Error("Project state file should exist")
		}
	})

	// Test loading project state
	t.Run("LoadProjectState", func(t *testing.T) {
		state, err := manager.LoadProjectState()
		if err != nil {
			t.Fatalf("Failed to load project state: %v", err)
		}

		if state.Name != projectName {
			t.Errorf("Expected project name %s, got %s", projectName, state.Name)
		}

		if len(state.VMs) != 2 {
			t.Errorf("Expected 2 VMs, got %d", len(state.VMs))
		}
	})

	// Test updating VM state
	t.Run("UpdateVMState", func(t *testing.T) {
		err := manager.UpdateVMState("web", func(vm *VMRuntime) error {
			vm.State = driver.Running
			vm.StartedAt = &time.Time{}
			*vm.StartedAt = time.Now()
			vm.PID = 12345
			return nil
		})

		if err != nil {
			t.Fatalf("Failed to update VM state: %v", err)
		}

		// Verify the update
		state, err := manager.LoadProjectState()
		if err != nil {
			t.Fatalf("Failed to load project state: %v", err)
		}

		webVM := state.VMs["web"]
		if webVM.State != driver.Running {
			t.Errorf("Expected VM state %s, got %s", driver.Running, webVM.State)
		}

		if webVM.PID != 12345 {
			t.Errorf("Expected PID 12345, got %d", webVM.PID)
		}

		if webVM.StartedAt == nil {
			t.Error("StartedAt should be set")
		}
	})

	// Test updating project status
	t.Run("UpdateProjectStatus", func(t *testing.T) {
		err := manager.UpdateProjectStatus(ProjectStatusRunning)
		if err != nil {
			t.Fatalf("Failed to update project status: %v", err)
		}

		state, err := manager.LoadProjectState()
		if err != nil {
			t.Fatalf("Failed to load project state: %v", err)
		}

		if state.Status != ProjectStatusRunning {
			t.Errorf("Expected status %s, got %s", ProjectStatusRunning, state.Status)
		}
	})

	// Test event recording
	t.Run("RecordEvent", func(t *testing.T) {
		event := &Event{
			ID:        "test-event-1",
			Timestamp: time.Now(),
			Type:      EventTypeVM,
			Resource:  "web",
			Action:    "start",
			Status:    EventStatusSuccess,
			Message:   "VM started successfully",
			Duration:  time.Second * 5,
		}

		err := manager.RecordEvent(event)
		if err != nil {
			t.Fatalf("Failed to record event: %v", err)
		}

		// Verify event was recorded
		events, err := manager.GetProjectEvents(10)
		if err != nil {
			t.Fatalf("Failed to get events: %v", err)
		}

		if len(events) != 1 {
			t.Errorf("Expected 1 event, got %d", len(events))
		}

		if events[0].ID != "test-event-1" {
			t.Errorf("Expected event ID 'test-event-1', got %s", events[0].ID)
		}
	})

	// Test project deletion
	t.Run("DeleteProject", func(t *testing.T) {
		err := manager.DeleteProject()
		if err != nil {
			t.Fatalf("Failed to delete project: %v", err)
		}

		if manager.ProjectExists() {
			t.Error("Project should not exist after deletion")
		}
	})
}

func TestGlobalConfig(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "spitfire-global-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Override the spitfire home for testing
	originalHome := os.Getenv(SpitfireHome)
	os.Setenv(SpitfireHome, tempDir)
	defer os.Setenv(SpitfireHome, originalHome)

	t.Run("LoadDefaultGlobalConfig", func(t *testing.T) {
		config, err := LoadGlobalConfig()
		if err != nil {
			t.Fatalf("Failed to load global config: %v", err)
		}

		// Verify default values
		if !config.CheckForUpdates {
			t.Error("Expected CheckForUpdates to be true by default")
		}

		if !config.TelemetryEnabled {
			t.Error("Expected TelemetryEnabled to be true by default")
		}

		if config.MaxEventHistory != 1000 {
			t.Errorf("Expected MaxEventHistory to be 1000, got %d", config.MaxEventHistory)
		}
	})

	t.Run("SaveAndLoadGlobalConfig", func(t *testing.T) {
		config := &GlobalConfig{
			DefaultDriver:    "qemu",
			LogLevel:         "debug",
			CheckForUpdates:  false,
			TelemetryEnabled: false,
			MaxEventHistory:  500,
			ConfigVersion:    "1",
		}

		err := SaveGlobalConfig(config)
		if err != nil {
			t.Fatalf("Failed to save global config: %v", err)
		}

		loadedConfig, err := LoadGlobalConfig()
		if err != nil {
			t.Fatalf("Failed to load global config: %v", err)
		}

		if loadedConfig.DefaultDriver != "qemu" {
			t.Errorf("Expected DefaultDriver 'qemu', got %s", loadedConfig.DefaultDriver)
		}

		if loadedConfig.LogLevel != "debug" {
			t.Errorf("Expected LogLevel 'debug', got %s", loadedConfig.LogLevel)
		}

		if loadedConfig.CheckForUpdates {
			t.Error("Expected CheckForUpdates to be false")
		}
	})
}

func TestListProjects(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "spitfire-list-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Override the spitfire home for testing
	originalHome := os.Getenv(SpitfireHome)
	os.Setenv(SpitfireHome, tempDir)
	defer os.Setenv(SpitfireHome, originalHome)

	// Create some test projects
	testConfig := &config.Config{
		Version: "1",
		VMs: map[string]*config.VM{
			"test": {Image: "alpine:latest"},
		},
	}

	projects := []string{"project1", "project2", "project3"}
	for _, projectName := range projects {
		manager := NewManager(projectName)
		configPath := filepath.Join(tempDir, projectName+".yml")
		configData := []byte("version: '1'\nvms:\n  test:\n    image: alpine:latest")
		if err := os.WriteFile(configPath, configData, 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		_, err := manager.InitializeProject(configPath, testConfig)
		if err != nil {
			t.Fatalf("Failed to initialize project %s: %v", projectName, err)
		}
	}

	// Test listing projects
	listedProjects, err := ListProjects()
	if err != nil {
		t.Fatalf("Failed to list projects: %v", err)
	}

	if len(listedProjects) != 3 {
		t.Errorf("Expected 3 projects, got %d", len(listedProjects))
	}

	// Verify all projects are listed
	projectMap := make(map[string]bool)
	for _, project := range listedProjects {
		projectMap[project] = true
	}

	for _, expected := range projects {
		if !projectMap[expected] {
			t.Errorf("Expected project %s not found in list", expected)
		}
	}
}

func TestPaths(t *testing.T) {
	// Test with custom spitfire home
	tempDir, err := os.MkdirTemp("", "spitfire-paths-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalHome := os.Getenv(SpitfireHome)
	os.Setenv(SpitfireHome, tempDir)
	defer os.Setenv(SpitfireHome, originalHome)

	t.Run("SpitfirePath", func(t *testing.T) {
		path := SpitfirePath()
		expected := filepath.Join(tempDir, ".spitfire")
		if path != expected {
			t.Errorf("Expected path %s, got %s", expected, path)
		}
	})

	t.Run("ProjectPaths", func(t *testing.T) {
		projectName := "test-project"

		projectPath := ProjectPath(projectName)
		expected := filepath.Join(tempDir, ".spitfire", "projects", projectName)
		if projectPath != expected {
			t.Errorf("Expected project path %s, got %s", expected, projectPath)
		}

		stateFile := ProjectStateFile(projectName)
		expected = filepath.Join(projectPath, "state.json")
		if stateFile != expected {
			t.Errorf("Expected state file %s, got %s", expected, stateFile)
		}
	})

	t.Run("EnsureDirectories", func(t *testing.T) {
		projectName := "test-ensure"

		err := EnsureProjectDirs(projectName)
		if err != nil {
			t.Fatalf("Failed to ensure project dirs: %v", err)
		}

		// Verify directories were created
		expectedDirs := []string{
			ProjectPath(projectName),
			filepath.Join(ProjectPath(projectName), "vms"),
			NetworkPath(projectName),
			VolumePath(projectName),
		}

		for _, dir := range expectedDirs {
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				t.Errorf("Directory %s was not created", dir)
			}
		}
	})
}
