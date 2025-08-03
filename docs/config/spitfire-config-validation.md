# Spitfire Configuration Validation

This document describes the validation system for Spitfire configuration files, including validation rules, error handling, and troubleshooting guidance.

## Overview

Spitfire performs comprehensive validation of configuration files to ensure they are syntactically correct, semantically valid, and contain all required information for successful VM deployment. Validation occurs at multiple levels:

1. **YAML Syntax Validation** - Ensures the file is valid YAML
2. **Schema Validation** - Validates the structure and data types
3. **Semantic Validation** - Checks business rules and cross-references
4. **Driver Validation** - Driver-specific option validation

## Validation Levels

### Level 1: YAML Syntax Validation

Before any configuration processing, Spitfire validates that the configuration file contains valid YAML syntax.

**Common YAML Errors:**
```yaml
# ERROR: Invalid indentation
version: "1"
vms:
web:  # Missing proper indentation
  image: "nginx:alpine"

# ERROR: Missing quotes for strings with special characters
version: "1"
vms:
  app:
    env:
      VAR: value with spaces  # Should be "value with spaces"

# ERROR: Inconsistent indentation (mixing tabs and spaces)
version: "1"
vms:
	app:  # Tab character
  image: "nginx:alpine"  # Spaces
```

### Level 2: Schema Validation

Validates that the configuration structure matches the expected schema and all required fields are present.

#### Required Fields Validation

```go
func (c *Config) ValidateConfig() error {
    if c.Version == "" {
        return fmt.Errorf("version is required")
    }
    
    if c.VMs == nil || len(c.VMs) == 0 {
        return fmt.Errorf("at least one VM must be defined")
    }
    
    // Continue validation...
}
```

**Examples:**
```yaml
# ERROR: Missing version
vms:
  app:
    image: "nginx:alpine"

# ERROR: No VMs defined
version: "1"
globals:
  memory: "1GB"
# Missing 'vms' section

# ERROR: Empty VMs section
version: "1"
vms: {}
```

#### Data Type Validation

```go
// Memory must be a string with size suffix
resources:
  memory: 512  # ERROR: Should be "512MB"

// VCPU must be an integer
resources:
  vcpu: "2"  # ERROR: Should be 2 (integer)

// Environment variables must be string-to-string map
env:
  PORT: 8080  # ERROR: Should be "8080" (string)
```

### Level 3: Semantic Validation

Validates business rules and logical consistency of the configuration.

#### VM Configuration Validation

```go
func (vm *VM) Validate(name string, globalDriver string) error {
    // Either image or rootfs must be specified, but not both
    if vm.Image == "" && vm.Rootfs == "" {
        return fmt.Errorf("either 'image' or 'rootfs' must be specified")
    }
    
    if vm.Image != "" && vm.Rootfs != "" {
        return fmt.Errorf("cannot specify both 'image' and 'rootfs'")
    }
    
    // Validate restart policy
    if vm.Restart != "" {
        validPolicies := []string{"always", "on-failure", "unless-stopped", "no"}
        if !contains(validPolicies, vm.Restart) {
            return fmt.Errorf("invalid restart policy '%s'", vm.Restart)
        }
    }
    
    return nil
}
```

**Examples:**
```yaml
# ERROR: Neither image nor rootfs specified
version: "1"
vms:
  app:
    env:
      PORT: "8080"

# ERROR: Both image and rootfs specified
version: "1"
vms:
  app:
    image: "nginx:alpine"
    rootfs: "/path/to/rootfs.ext4"

# ERROR: Invalid restart policy
version: "1"
vms:
  app:
    image: "nginx:alpine"
    restart: "invalid-policy"
```

#### Driver Name Validation

```go
func validateDriverName(name string) error {
    if name == "" {
        return fmt.Errorf("driver name cannot be empty")
    }
    
    // Driver names must be lowercase alphanumeric with hyphens
    matched, err := regexp.MatchString(`^[a-z][a-z0-9-]*[a-z0-9]$|^[a-z]$`, name)
    if err != nil {
        return err
    }
    if !matched {
        return fmt.Errorf("driver name '%s' must be lowercase alphanumeric with hyphens", name)
    }
    
    return nil
}
```

**Examples:**
```yaml
# Valid driver names
driver: "firecracker"
driver: "qemu"
driver: "qemu-kvm"
driver: "virtualbox"
driver: "a"

# Invalid driver names
driver: "Firecracker"      # ERROR: Contains uppercase
driver: "qemu_kvm"         # ERROR: Contains underscore
driver: "qemu-"            # ERROR: Ends with hyphen
driver: "-qemu"            # ERROR: Starts with hyphen
driver: "qemu kvm"         # ERROR: Contains space
driver: "123abc"           # ERROR: Starts with number
driver: ""                 # ERROR: Empty string
```

#### Reference Validation

Ensures all references to networks, volumes, and VMs are valid.

```go
func (c *Config) validateNetworkReferences() error {
    allNetworks := make(map[string]bool)
    
    // Collect all defined networks
    for name := range c.Networks {
        allNetworks[name] = true
    }
    
    // Check VM network references
    for vmName, vm := range c.VMs {
        for _, network := range vm.Networks {
            if !allNetworks[network] {
                return fmt.Errorf("VM '%s' references undefined network '%s'", vmName, network)
            }
        }
    }
    
    return nil
}
```

**Examples:**
```yaml
# ERROR: Undefined network reference
version: "1"
networks:
  frontend:
    driver: bridge

vms:
  app:
    image: "nginx:alpine"
    networks:
      - frontend     # OK: network exists
      - backend      # ERROR: network not defined

# ERROR: Undefined volume reference
version: "1"
volumes:
  app-data:
    driver: local

vms:
  app:
    image: "nginx:alpine"
    volumes:
      - source: app-data     # OK: volume exists
        target: /data
      - source: cache-data   # ERROR: volume not defined
        target: /cache
```

#### Dependency Validation

```go
func (c *Config) validateDependencies() error {
    for vmName, vm := range c.VMs {
        for _, dep := range vm.DependsOn {
            if _, exists := c.VMs[dep]; !exists {
                return fmt.Errorf("VM '%s' depends on undefined VM '%s'", vmName, dep)
            }
        }
    }
    
    // TODO: Check for circular dependencies
    return nil
}
```

### Level 4: Resource Validation

Validates resource specifications and constraints.

#### Memory Size Validation

```go
func parseMemoryToMB(size string) (int64, error) {
    if size == "" {
        return 512, nil // Default
    }
    
    size = strings.ToUpper(strings.TrimSpace(size))
    
    if strings.HasSuffix(size, "MB") || strings.HasSuffix(size, "M") {
        numStr := strings.TrimSuffix(strings.TrimSuffix(size, "MB"), "M")
        mb, err := strconv.ParseInt(numStr, 10, 64)
        if err != nil {
            return 0, fmt.Errorf("invalid memory size format: %s", size)
        }
        return mb, nil
    }
    
    if strings.HasSuffix(size, "GB") || strings.HasSuffix(size, "G") {
        numStr := strings.TrimSuffix(strings.TrimSuffix(size, "GB"), "G")
        gb, err := strconv.ParseInt(numStr, 10, 64)
        if err != nil {
            return 0, fmt.Errorf("invalid memory size format: %s", size)
        }
        return gb * 1024, nil
    }
    
    return 0, fmt.Errorf("unsupported memory size format: %s (use MB or GB)", size)
}
```

**Valid Memory Formats:**
```yaml
memory: "512MB"
memory: "1GB"
memory: "2G"
memory: "256M"
memory: ""      # Uses default (512MB)
```

**Invalid Memory Formats:**
```yaml
memory: "512KB"    # ERROR: Unsupported unit
memory: "1TB"      # ERROR: Unsupported unit
memory: "512"      # ERROR: Missing unit
memory: "abc MB"   # ERROR: Invalid number
memory: "0MB"      # ERROR: Zero memory not allowed
```

#### CPU Validation

```yaml
# Valid CPU configurations
resources:
  vcpu: 1
  vcpu: 2
  vcpu: 4
  vcpu: 8

# Invalid CPU configurations
resources:
  vcpu: 0     # ERROR: Must be at least 1
  vcpu: -1    # ERROR: Cannot be negative
  vcpu: "2"   # ERROR: Must be integer, not string
```

## Driver-Specific Validation

Driver-specific options are validated when the driver is loaded. This validation is performed by the driver implementation itself.

### Firecracker Driver Validation

```yaml
driver_opts:
  firecracker:
    jailer: true                    # Valid: boolean
    cpu_template: "C3"              # Valid: C3 or T2
    log_level: "Debug"              # Valid: Debug, Info, Warn, Error
    invalid_option: "value"         # ERROR: Unknown option
```

### QEMU Driver Validation

```yaml
driver_opts:
  qemu:
    accel: "kvm"                    # Valid: kvm, tcg, hvf
    display: "vnc=:1"               # Valid: VNC display string
    machine: "q35"                  # Valid: QEMU machine type
    invalid_accel: "invalid"        # ERROR: Unknown accelerator
```

## Error Messages and Troubleshooting

### Error Message Format

Spitfire provides detailed error messages with context:

```
Error validating configuration:
  VM 'web': invalid restart policy 'invalid-policy', must be one of: always, on-failure, unless-stopped, no
    at line 15, column 5 in spitfire.yml
```

### Common Errors and Solutions

#### Configuration Structure Errors

**Error**: `version is required`
```yaml
# Missing version field
vms:
  app:
    image: "nginx:alpine"
```
**Solution**: Add version field at the root level:
```yaml
version: "1"
vms:
  app:
    image: "nginx:alpine"
```

**Error**: `at least one VM must be defined`
```yaml
version: "1"
networks:
  default:
    driver: bridge
```
**Solution**: Add at least one VM definition:
```yaml
version: "1"
vms:
  app:
    image: "nginx:alpine"
```

#### VM Configuration Errors

**Error**: `either 'image' or 'rootfs' must be specified`
```yaml
version: "1"
vms:
  app:
    env:
      PORT: "8080"
```
**Solution**: Add either image or rootfs:
```yaml
version: "1"
vms:
  app:
    image: "nginx:alpine"
    env:
      PORT: "8080"
```

**Error**: `cannot specify both 'image' and 'rootfs'`
```yaml
version: "1"
vms:
  app:
    image: "nginx:alpine"
    rootfs: "/path/to/rootfs.ext4"
```
**Solution**: Choose either image or rootfs:
```yaml
version: "1"
vms:
  app:
    image: "nginx:alpine"
```

#### Driver Configuration Errors

**Error**: `invalid driver name 'Invalid-Driver!' must be lowercase alphanumeric with hyphens`
```yaml
version: "1"
driver: "Invalid-Driver!"
```
**Solution**: Use valid driver name format:
```yaml
version: "1"
driver: "firecracker"
```

**Error**: `invalid driver name in driver_opts 'FIRECRACKER'`
```yaml
version: "1"
driver_opts:
  FIRECRACKER:
    jailer: true
```
**Solution**: Use lowercase driver name:
```yaml
version: "1"
driver_opts:
  firecracker:
    jailer: true
```

#### Reference Errors

**Error**: `VM 'web' references undefined network 'backend'`
```yaml
version: "1"
networks:
  frontend:
    driver: bridge

vms:
  web:
    image: "nginx:alpine"
    networks:
      - backend
```
**Solution**: Define the referenced network:
```yaml
version: "1"
networks:
  frontend:
    driver: bridge
  backend:
    driver: bridge

vms:
  web:
    image: "nginx:alpine"
    networks:
      - backend
```

#### Resource Specification Errors

**Error**: `unsupported memory size format: 512KB (use MB or GB)`
```yaml
version: "1"
vms:
  app:
    image: "nginx:alpine"
    resources:
      memory: "512KB"
```
**Solution**: Use supported memory units:
```yaml
version: "1"
vms:
  app:
    image: "nginx:alpine"
    resources:
      memory: "512MB"
```

## Validation Best Practices

### 1. Use Schema Validation Tools

Before running Spitfire, validate your configuration:

```bash
# Validate configuration syntax
spitfire config validate spitfire.yml

# Dry run to check for runtime issues
spitfire up --dry-run
```

### 2. Start Simple

Begin with minimal configurations and add complexity incrementally:

```yaml
# Start with this
version: "1"
vms:
  app:
    image: "nginx:alpine"

# Then add features
version: "1"
driver: "firecracker"
vms:
  app:
    image: "nginx:alpine"
    resources:
      memory: "512MB"

# Finally add complexity
version: "1"
driver: "firecracker"
driver_opts:
  firecracker:
    jailer: true
globals:
  resources:
    memory: "512MB"
networks:
  app-net:
    driver: bridge
vms:
  app:
    image: "nginx:alpine"
    networks:
      - app-net
```

### 3. Use Consistent Naming

Follow consistent naming conventions to avoid reference errors:

```yaml
# Good: Consistent naming pattern
networks:
  frontend-net:
    driver: bridge
  backend-net:
    driver: bridge

volumes:
  app-data:
    driver: local
  cache-data:
    driver: tmpfs

vms:
  web-server:
    image: "nginx:alpine"
    networks:
      - frontend-net
  api-server:
    image: "api:latest"
    networks:
      - backend-net
    volumes:
      - source: app-data
        target: /data
```

### 4. Environment-Specific Validation

Use different validation approaches for different environments:

```bash
# Development: Permissive validation
export SPITFIRE_VALIDATION_LEVEL=warn

# Staging: Strict validation
export SPITFIRE_VALIDATION_LEVEL=error

# Production: Maximum validation
export SPITFIRE_VALIDATION_LEVEL=strict
```

## Testing Configuration Files

### Unit Testing Approach

```go
func TestConfigValidation(t *testing.T) {
    tests := []struct {
        name    string
        config  string
        wantErr bool
        errMsg  string
    }{
        {
            name: "valid minimal config",
            config: `
version: "1"
vms:
  app:
    image: "nginx:alpine"
`,
            wantErr: false,
        },
        {
            name: "missing version",
            config: `
vms:
  app:
    image: "nginx:alpine"
`,
            wantErr: true,
            errMsg:  "version is required",
        },
        // Add more test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var config Config
            err := yaml.Unmarshal([]byte(tt.config), &config)
            require.NoError(t, err)
            
            err = config.ValidateConfig()
            if tt.wantErr {
                assert.Error(t, err)
                if tt.errMsg != "" {
                    assert.Contains(t, err.Error(), tt.errMsg)
                }
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Integration Testing

```bash
#!/bin/bash
# test-configs.sh

# Test valid configurations
for config in tests/valid/*.yml; do
    echo "Testing valid config: $config"
    spitfire config validate "$config" || exit 1
done

# Test invalid configurations
for config in tests/invalid/*.yml; do
    echo "Testing invalid config: $config"
    if spitfire config validate "$config"; then
        echo "ERROR: Expected validation failure for $config"
        exit 1
    fi
done

echo "All configuration tests passed!"
```

This comprehensive validation system ensures that Spitfire configurations are correct, complete, and ready for deployment across different virtualization drivers.