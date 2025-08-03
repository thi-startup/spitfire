package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/thi-startup/spitfire/pkg/log"
)

// LoadConfig loads and parses a spitfire configuration file
func LoadConfig(configPath string) (*Config, error) {
	// Read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	// Substitute environment variables
	expanded := ExpandEnvironmentVariables(string(data))

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal([]byte(expanded), &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	// Post-process the configuration
	if err := config.postProcess(); err != nil {
		return nil, fmt.Errorf("failed to post-process config: %w", err)
	}

	// Validate the configuration
	if err := config.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

// ExpandEnvironmentVariables replaces ${VAR} and $VAR with environment variable values
func ExpandEnvironmentVariables(input string) string {
	// Pattern for ${VAR} and $VAR
	re := regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}|\$([A-Za-z_][A-Za-z0-9_]*)`)

	return re.ReplaceAllStringFunc(input, func(match string) string {
		var varName string
		if strings.HasPrefix(match, "${") {
			// ${VAR} format
			varName = strings.TrimSuffix(strings.TrimPrefix(match, "${"), "}")
		} else {
			// $VAR format
			varName = strings.TrimPrefix(match, "$")
		}

		value := os.Getenv(varName)
		if value == "" {
			// Return the original match if no environment variable found
			return match
		}
		return value
	})
}

// postProcess handles parsing of special syntax and inheritance
func (c *Config) postProcess() error {
	// Apply global defaults to VMs
	if err := c.applyGlobalDefaults(); err != nil {
		return err
	}

	// Parse volume mounts (handle short syntax like "data:/app/data")
	if err := c.parseVolumeMounts(); err != nil {
		return err
	}

	// Load env_file contents
	if err := c.loadEnvFiles(); err != nil {
		return err
	}

	return nil
}

// applyGlobalDefaults applies global configuration to VMs where not overridden
func (c *Config) applyGlobalDefaults() error {
	if c.Globals == nil {
		return nil
	}

	for _, vm := range c.VMs {
		// Apply global kernel if not set
		if vm.Kernel == "" && c.Globals.Kernel != "" {
			vm.Kernel = c.Globals.Kernel
		}

		// Apply global kernel args if not set
		if vm.KernelArgs == "" && c.Globals.KernelArgs != "" {
			vm.KernelArgs = c.Globals.KernelArgs
		}

		// Apply global resources if not set
		if vm.Resources == nil && c.Globals.Resources != nil {
			vm.Resources = &Resources{
				VCPU:   c.Globals.Resources.VCPU,
				Memory: c.Globals.Resources.Memory,
			}
		}

		// Apply global restart policy if not set
		if vm.Restart == "" && c.Globals.Restart != "" {
			vm.Restart = c.Globals.Restart
		}

		// Merge global networks with VM networks
		if len(c.Globals.Networks) > 0 {
			networkSet := make(map[string]bool)

			// Add global networks first
			for _, network := range c.Globals.Networks {
				networkSet[network] = true
			}

			// Add VM-specific networks
			for _, network := range vm.Networks {
				networkSet[network] = true
			}

			// Convert back to slice
			vm.Networks = make([]string, 0, len(networkSet))
			for network := range networkSet {
				vm.Networks = append(vm.Networks, network)
			}
		}

		// Merge global environment variables (VM env takes precedence)
		if len(c.Globals.Env) > 0 {
			if vm.Env == nil {
				vm.Env = make(map[string]string)
			}

			for key, value := range c.Globals.Env {
				if _, exists := vm.Env[key]; !exists {
					vm.Env[key] = value
				}
			}
		}
	}

	return nil
}

// parseVolumeMounts handles different volume mount syntaxes
func (c *Config) parseVolumeMounts() error {
	for vmName, vm := range c.VMs {
		for i, mount := range vm.Volumes {
			if err := mount.parse(); err != nil {
				return fmt.Errorf("VM '%s' volume mount %d: %w", vmName, i, err)
			}
		}
	}
	return nil
}

// parse handles parsing of volume mount syntax
func (vm *VolumeMount) parse() error {
	// If we have explicit type, source, and target, we're done
	if vm.Type != "" && vm.Source != "" && vm.Target != "" {
		return nil
	}

	// Handle short syntax parsing (this would need custom YAML unmarshaling in real implementation)
	// For now, we'll assume the long syntax is always used
	// In a full implementation, you'd handle cases like:
	// - "data:/app/data" -> Type: "volume", Source: "data", Target: "/app/data"
	// - "/host/path:/vm/path" -> Type: "bind", Source: "/host/path", Target: "/vm/path"

	return nil
}

// loadEnvFiles loads environment variables from files
func (c *Config) loadEnvFiles() error {
	for vmName, vm := range c.VMs {
		for _, envFile := range vm.EnvFile {
			env, err := loadEnvFile(envFile)
			if err != nil {
				return fmt.Errorf("VM '%s' env_file '%s': %w", vmName, envFile, err)
			}

			// Merge loaded env vars (existing ones in vm.Env take precedence)
			if vm.Env == nil {
				vm.Env = make(map[string]string)
			}

			for key, value := range env {
				if _, exists := vm.Env[key]; !exists {
					vm.Env[key] = value
				}
			}
		}
	}
	return nil
}

// loadEnvFile loads environment variables from a file
func loadEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	env := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // Skip malformed lines
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Handle quoted values
		if len(value) >= 2 {
			if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
				(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				value = value[1 : len(value)-1]
			}
		}

		env[key] = value
	}

	return env, scanner.Err()
}

// FindConfigFile finds a configuration file in the current directory or specified path
func FindConfigFile(configPath string) (string, error) {
	// If a specific path is provided, use it
	if configPath != "" {
		if _, err := os.Stat(configPath); err != nil {
			return "", fmt.Errorf("config file %s not found: %w", configPath, err)
		}
		return configPath, nil
	}

	// Look for default config files
	defaultNames := []string{"spitfire.yaml", "spitfire.yml", "compose.yaml", "compose.yml"}

	for _, name := range defaultNames {
		if _, err := os.Stat(name); err == nil {
			return name, nil
		}
	}

	return "", fmt.Errorf("no configuration file found, tried: %s", strings.Join(defaultNames, ", "))
}

// GetProjectName derives a project name from the config file path or directory
func GetProjectName(configPath string) string {
	if configPath == "" {
		// Use current directory name
		wd, _ := os.Getwd()
		return filepath.Base(wd)
	}

	// Use the directory name of the config file
	dir := filepath.Dir(configPath)
	return filepath.Base(dir)
}

// OutputResolvedConfig outputs the fully resolved configuration in YAML format
func OutputResolvedConfig(cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal resolved config: %w", err)
	}

	log.UserLn(string(data))
	return nil
}
