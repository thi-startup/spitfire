# Spitfire Project Roadmap

## Project Vision

Spitfire aims to be a comprehensive tool for managing Firecracker microVMs with a Docker Compose-like experience. The goal is to enable developers to easily create, configure, and manage clusters of microVMs running both container images and traditional VM images, with fine-grained control over networking, storage, and resource allocation.

## Current Status

The current implementation of Spitfire focuses primarily on:
- Creating loop devices with various filesystems
- Populating filesystems from directories or container images
- Bundling a custom init system

## Requirements

Based on discussions, Spitfire should:

1. Support a declarative configuration format (YAML/JSON) for defining microVM environments
2. Support both container images and VM images
3. Potentially integrate with existing container tools while maintaining the ability to work independently
4. Support complex network topologies
5. Provide options for custom init capabilities or standard container init systems
6. Have minimal dependencies and setup requirements
7. Enable simulation of entire clusters with intercommunication capabilities

## Initial MVP Scope

For the initial MVP release, we will focus on these core capabilities:

1. **System Setup**
   - One-time privileged setup via `spitfire setup` command
   - User-friendly initialization with proper permissions

2. **Basic VM Management**
   - Run containers as microVMs via declarative config
   - Basic lifecycle management (start, stop, restart)
   - Simple resource controls (CPU, memory)

3. **Basic Networking**
   - Single bridge network for VM connectivity
   - Host-to-VM and VM-to-VM communication
   - Simple IP allocation

4. **Basic Storage**
   - Container-to-rootfs conversion
   - Basic volume mounting
   - Support for persisting data

5. **Configuration Format**
   - Support for globals and VM-specific configurations
   - Environment variable support
   - Basic dependency ordering

This MVP will enable users to:
- Define and run a set of interconnected services
- Use familiar container images
- Persist data between restarts
- Configure networking between services

## Development Roadmap

### Phase 1: Core Infrastructure & Basic VM Management

1. **Configuration Format Development**
   - Design YAML/JSON schema for microVM declarations
   - Support for basic VM settings (CPU, memory, kernel)
   - Simple networking and storage definitions
   - Validation layer for configurations
   - Global configuration for defaults and project-wide settings

2. **Firecracker API Integration**
   - Direct integration with Firecracker API for VM lifecycle management
   - Support for starting, stopping, and monitoring microVMs
   - API client library development
   - Socket management for Firecracker instances

3. **Kernel Management**
   - Local kernel registry in `~/.spitfire/kernels/`
   - Metadata tracking for kernels (version, features, build options)
   - CLI commands for kernel management:
     - `spitfire kernel list` - List available kernels
     - `spitfire kernel get [version]` - Download specific kernel
     - `spitfire kernel inspect [kernel]` - Show kernel details
     - `spitfire kernel default [kernel]` - Set default kernel
   - Integration with VM configuration
   - Kernel acquisition from trusted repositories
   - Verification of kernel signatures and checksums

   **Implementation Approach**

   Initially, we'll maintain `spitfire-build-kernel` as a companion tool that shares the kernel registry format with Spitfire. This provides:
   - Cleaner separation of concerns
   - Simpler initial implementation
   - Flexibility for future integration if needed

   The tools will share a common kernel registry format, allowing seamless interoperation while keeping responsibilities clearly defined.

4. **Basic Networking**
   - Implement TAP device creation and management
   - Network bridge for VM connectivity
   - DHCP service for IP assignment
   - Host-to-VM communication channels
   - Basic DNS resolution

### Networking Approach

#### Overview

Spitfire will implement a networking model that emulates Docker's networking approach while being specifically optimized for Firecracker microVMs.

#### Network Types

Initially, we'll support the following network types:

1. **Bridge Networks** (Default)
   - Isolated virtual networks with their own subnet
   - VMs on the same bridge network can communicate
   - Similar to Docker's bridge networks

2. **Host Network**
   - VMs share the host's network namespace
   - Direct access to host network interfaces
   - Higher performance, lower isolation

3. **None**
   - No network connectivity
   - Used for completely isolated VMs

#### Implementation Approach

1. **Network Creation Process**
   - Create Linux bridge interfaces for each defined network
   - Configure bridges with IP addresses (gateway)
   - Create and connect TAP devices for each VM to appropriate bridges
   - Pass TAP devices to Firecracker as network interfaces

2. **IP Address Management**
   - Built-in DHCP server for each bridge network
   - Automatic IP allocation from subnet
   - Optional static IP assignment

3. **Service Discovery**
   - Local DNS server for VM name resolution
   - Automatic registration of VMs in DNS
   - Support for custom DNS entries

4. **Network Isolation and Security**
   - Separation between networks by default
   - VMs can only communicate with VMs on the same networks
   - Optional routes between networks when needed
   - Default deny-all policy for maximum security

5. **Performance Considerations**
   - MTU configuration for optimal packet size
   - TX/RX queue sizing
   - TCP optimizations
   - Minimize context switching

These implementations will be built directly on Linux networking primitives rather than using CNI plugins, ensuring simplicity and minimizing external dependencies.

## Storage Design

### Overview

Spitfire needs a robust storage system to manage rootfs volumes and additional data volumes for Firecracker microVMs, building upon the existing `mkroot` functionality while expanding capabilities for different volume types and use cases.

### Volume Types

1. **Named Volumes**
   - Managed by Spitfire
   - Persist between VM restarts
   - Located in a central storage location

2. **Bind Mounts**
   - Link to directories on the host filesystem
   - Useful for development and configuration files
   - Direct access to host files

3. **Rootfs Volumes**
   - Special volume containing the root filesystem
   - Can be created from container images or VM images
   - Required for each VM

### Implementation Approach

#### Volume Management

1. **Volume Creation**
   - Create loop devices or files for each volume
   - Format with specified filesystem (ext4 default)
   - Mount for population as needed

2. **Volume Storage**
   - Store volumes in `~/.spitfire/volumes/<project>/<volume-name>`
   - Metadata in JSON files alongside volumes
   - Naming convention to ensure uniqueness

3. **Volume Lifecycle**
   - Create volumes during `spitfire up`
   - Clean up non-persistent volumes during `spitfire down`
   - Maintain persistent volumes across restarts

#### Rootfs Preparation

1. **Container Image Conversion**
   - Pull container images from registries
   - Extract layers to construct rootfs
   - Configure for Firecracker compatibility
   - Set up init system and networking

2. **VM Image Support**
   - Support raw disk images
   - Optional conversion from other formats (qcow2, etc.)
   - Kernel modules and driver compatibility checks

#### Mount Process

1. **Attaching Volumes to VMs**
   - Present volumes as block devices to Firecracker
   - Support multiple volumes per VM
   - Configure mount points inside guest

2. **Performance Optimization**
   - Cache settings for different workloads
   - Block size optimization
   - I/O schedulers tuning

### Volume Drivers

Initially, we'll support:

1. **Local Driver** (Default)
   - Loop devices on the local filesystem
   - Simple, reliable, no dependencies

2. **Tmpfs Driver**
   - Memory-backed temporary storage
   - High performance but volatile
   - Size-limited by available RAM

Future drivers could include network storage options.

### Phase 2: Container Integration & Advanced Features

## Container Runtime Design

### Overview

A key feature of Spitfire is the ability to run container images within Firecracker microVMs. This section outlines our approach to container image handling, conversion to VM-compatible rootfs, and runtime management.

### Core Challenges

Running containers in microVMs presents several unique challenges:

1. **Image Conversion**: Converting OCI/Docker container images to bootable root filesystems
2. **Init System**: Selecting an appropriate init system for the VM
3. **Entrypoint Management**: Preserving container entrypoints and command handling
4. **Environment and Configuration**: Mapping container env vars, volumes, etc. to VM configuration
5. **Resource Limits**: Translating container resource constraints to VM constraints

### Implementation Approach

#### Container Image Handling

1. **Image Pulling**:
   - Support for Docker Hub and other OCI-compatible registries
   - Authentication for private registries
   - Local caching of pulled images to avoid redundant downloads
   - Layer deduplication for storage efficiency

2. **Image to Rootfs Conversion**:
   - Extract all layers and merge to form a complete root filesystem
   - Preserve file metadata (permissions, ownership)
   - Handle special cases like symlinks and hardlinks correctly
   - Optimize conversion for frequently used base images

3. **Bootable Filesystem Creation**:
   - Enhance the rootfs with necessary system files
   - Configure networking support in the image
   - Set up the init system
   - Inject any required configuration files

#### Runtime Environment

1. **Minimal Init System**:
   - Use a lightweight init system (e.g., tini, dumb-init, or a custom solution)
   - Configure to start the container's entrypoint process
   - Handle zombie process reaping and signal forwarding
   - Support for graceful shutdown

2. **Container Configuration Mapping**:
   - Map Docker/OCI container configuration to VM configuration
   - Support for entrypoint, command, and arguments
   - Environment variable passing
   - Working directory configuration
   - User/group ID mapping

3. **Multi-container Support** (Future):
   - Option to run multiple containers in a single VM (like Docker Compose pods)
   - Container-to-container communication
   - Shared volume access

### Technical Approaches

For the initial implementation, we will use a **Direct Conversion** approach:

1. Pull and extract container image layers
2. Merge layers into a cohesive rootfs
3. Inject an init system that executes the container entrypoint
4. Configure the resulting filesystem for Firecracker

This gives us:
- A simpler initial implementation
- Faster time-to-market with basic functionality
- A clear path to supporting more complex container features in the future

2. **Advanced Networking**
   - Multiple network definitions and topologies
   - Network isolation between VM groups
   - Custom routing and firewall rules
   - DNS service for inter-VM discovery
   - Network performance tuning
   - Traffic shaping and bandwidth limits
   - Network monitoring and statistics

3. **Configuration Management**
   - Environment variables and secrets handling
   - Service dependency resolution
   - Health checking mechanisms
   - Automatic restart policies
   - Configuration templating

4. **Resource Control**
   - CPU and memory limits enforcement
   - Disk I/O throttling and monitoring
   - Network bandwidth controls
   - Resource usage statistics and reporting

### Phase 3: Cluster Simulation & Advanced Use Cases

1. **Multi-VM Orchestration**
   - Service groups and scaling
   - Coordinated VM startup and shutdown sequences
   - Service discovery integration
   - Rolling updates of VM clusters

## Init System Design

### Overview

The init system is a critical component in Firecracker microVMs, responsible for system initialization, process management, and graceful shutdown. This section outlines Spitfire's approach to init system management and integration.

### Requirements

An effective init system for Spitfire needs to:

1. **Be lightweight**: Minimal memory and CPU overhead
2. **Start quickly**: Support fast boot times
3. **Handle process supervision**: Properly manage child processes
4. **Support container workloads**: Execute container entrypoints and commands
5. **Manage networking**: Configure network interfaces
6. **Handle signals**: Properly propagate signals to child processes
7. **Support shutdown**: Graceful shutdown of services
8. **Be configurable**: Easy to configure for different workloads

### Implementation Approach

We recommend a **Hybrid/Registry Approach** which allows us to:

1. Enhance the existing `thi-startup/init` as our default init
2. Support other popular init systems
3. Allow users to specify which init to use

### Init System Registry

#### Registry Structure

1. **Local Registry**
   - Stored in `~/.spitfire/inits/`
   - Each init stored with version information
   - Metadata for compatibility and requirements

2. **Remote Sources**
   - GitHub repositories
   - Container registries
   - Custom HTTP endpoints

#### Naming Convention and References

Inits can be referenced in multiple ways:

1. **Local Reference**
   - Simple name with optional tag: `thi-init:latest`
   - Refers to an init already downloaded to local registry

2. **URL Reference**
   - Direct download URL: `https://github.com/thi-startup/init/releases/download/v0.1.0/init`
   - Will be downloaded and cached on first use

3. **GitHub Reference**
   - GitHub repo format: `github.com/username/repo@tag`
   - Will be fetched from GitHub and built/cached

4. **Container Image Reference**
   - OCI image format: `docker.io/username/init-image:tag`
   - Will extract init binary from container image

### Integration with Spitfire

#### Init Lifecycle

1. **Download/Verification**
   - Pull init from source if not cached
   - Verify integrity and compatibility
   - Cache for future use

2. **Configuration**
   - Generate init-specific configuration
   - Map container/VM config to init config

3. **Integration into Rootfs**
   - Inject init into the rootfs
   - Configure as PID 1
   - Set up necessary files and directories

4. **Execution**
   - Boot VM with configured init as PID 1
   - Monitor init status
   - Handle shutdown signals

3. **Developer Experience**
   - Live reload capabilities for development
   - Development workflow tooling
   - Debugging and introspection enhancements
   - IDE integrations (if applicable)

4. **Advanced Storage Features**
   - Snapshot and restore capabilities
   - Storage migration between hosts
   - Backup and restore workflows
   - Storage encryption options
   - VM filesystem mounting and direct access
   - File synchronization between host and VM

### Future Enhancements

1. **Advanced Firecracker Integration**
   - Support for full range of Firecracker API primitives
   - Fine-grained resource controls (CPU templates, huge pages, I/O throttling)
   - Memory hotplug and dynamic resource adjustment
   - Rate limiters for network and I/O
   - Balloon device driver support for memory management
   - Custom entropy sources
   - Advanced metrics collection and monitoring

## Proposed Configuration Format

```yaml
version: "1"

# Global configurations that apply across resources
globals:
  kernel: "/path/to/vmlinux"
  kernel_args: "console=ttyS0 reboot=k panic=1 pci=off"
  resources:
    vcpu: 1
    memory: "512MB"
  networks:
    - backend
  restart: "always"
  env:
    LOG_LEVEL: "info"
    ENVIRONMENT: "development"

vms:
  api-service:
    image: "docker.io/myapp/api:latest"  # Container image
    # Alternatively, use rootfs for VM images
    # rootfs: "/path/to/rootfs"

    # Override global settings as needed
    resources:
      vcpu: 2  # Overrides the global setting
    volumes:
      - data:/app/data
      - type: bind
        source: ./configs
        target: /etc/configs
    env:
      DB_HOST: "db-service"
      API_KEY: "secret-key"
    depends_on:
      - db-service

  db-service:
    image: "docker.io/postgres:13"
    resources:
      memory: "1GB"  # Overrides the global setting
    volumes:
      - db-data:/var/lib/postgresql/data
    env_file:
      - ./database.env

networks:
  backend:
    driver: bridge
    ipam:
      config:
        - subnet: "172.28.0.0/16"
          gateway: "172.28.0.1"
    options:
      mtu: 1500

volumes:
  data:
    driver: local
  db-data:
    driver: local
    persist: true
    size: "10GB"
```

## Configuration Management

The configuration system will follow these principles:

1. **Global Configuration**: A global configuration file stored at `~/.spitfire/config.yaml` will manage default settings, credentials, and paths.

2. **Project Configuration**: Each project can have its own configuration file (typically `spitfire.yaml` or specified via `-f` flag).

3. **Inheritance Model**: Resources inherit from globals, but local definitions override global ones.

4. **Variable Substitution**: Support for environment variable substitution and internal reference resolution.

5. **State Management**: Running environment state will be stored in `~/.spitfire/state/` with subdirectories for each project.

### Example Global Config

```yaml
defaults:
  kernel: "/usr/local/share/spitfire/kernels/vmlinux"
  kernel_args: "console=ttyS0 reboot=k panic=1 pci=off"
  resources:
    vcpu: 1
    memory: "512MB"

paths:
  kernels: "/usr/local/share/spitfire/kernels"
  volumes: "~/.spitfire/volumes"

registry:
  auth:
    dockerhub:
      username: "${DOCKER_USERNAME}"
      password: "${DOCKER_PASSWORD}"
```

## Proposed CLI Interface

```
# System Setup
spitfire setup                       # Initial system setup (requires sudo)

# VM Lifecycle Management
spitfire up -f compose.yaml          # Start services defined in compose file
spitfire down -f compose.yaml        # Stop services
spitfire restart [service]           # Restart specific service or all
spitfire ps                          # List running VMs
spitfire logs service                # Get logs from a VM
spitfire exec service command        # Execute command in a running VM
spitfire mount service [path]        # Mount VM filesystem to local directory

# Resource Management
spitfire network create name [opts]  # Create a network
spitfire network ls                  # List networks
spitfire network rm name             # Remove a network

spitfire volume create name [opts]   # Create a volume
spitfire volume ls                   # List volumes
spitfire volume rm name              # Remove a volume

# Kernel Management
spitfire kernel list                 # List available kernels
spitfire kernel get [version]        # Download a specific kernel
spitfire kernel inspect [kernel]     # Show details about a kernel
spitfire kernel default [kernel]     # Set as default kernel

# Init System Management
spitfire init ls                     # List available init systems
spitfire init pull name[:tag]        # Download an init system
spitfire init inspect name[:tag]     # Show details about an init
spitfire init rm name[:tag]          # Remove an init system

# Utilities
spitfire inspect service             # Show detailed info about a service
spitfire config validate -f file.yaml # Validate config file
spitfire version                     # Show version info
```

## Key Technical Challenges

1. **Firecracker API Integration**
   - Efficient management of Firecracker process lifecycle
   - Jailer integration for additional security
   - Metrics collection and monitoring
   - Balancing simplicity with access to advanced Firecracker features

2. **Networking**
   - Establishing complex network topologies
   - Performance tuning for VM-to-VM communication
   - Proper isolation while allowing defined connectivity
   - DNS resolution between services

3. **Container Integration**
   - Efficient transformation of container configs to VM configs
   - Handling container-specific features in a VM context
   - Performance considerations for container workloads

4. **Resource Management**
   - Accurate tracking and enforcement of resource limits
   - Handling resource contention
   - Graceful degradation under resource pressure

5. **Kernel Management**
   - Providing optimized kernels for different use cases
   - Maintaining compatibility with Firecracker's requirements
   - Ensuring secure distribution and verification
   - Balancing between custom and standard kernel options

## Implementation Approach

The implementation will be modular, with clearly separated concerns:

1. **Core Library**
   - Firecracker API client
   - Configuration parsing and validation
   - Resource management primitives

2. **CLI Interface**
   - Command handlers
   - User feedback and logging
   - Configuration file management
   - System setup and permissions management

3. **Resource Providers**
   - Network implementation
   - Storage implementation
   - VM lifecycle management
   - Kernel management

4. **Orchestration Layer**
   - Service dependency resolution
   - Coordinated operations (start/stop)
   - Health checking and recovery

Each module should be developed with clear interfaces to allow for future extensions and alternative implementations.

## System Setup and Permissions

### Overview

To improve user experience, Spitfire will include a setup mechanism to handle privileged operations while allowing normal usage without sudo.

### Setup Process

#### `spitfire setup` Command

This command will:
- Run with sudo privileges once during initial setup
- Create necessary user groups (e.g., `spitfire`)
- Configure udev rules for loop device and network access
- Set up filesystem permissions for VM access
- Store setup state to avoid repeated execution

#### Implementation Details

1. **User Group Management**
   - Add the current user to the `spitfire` group
   - Set appropriate group permissions on key files and directories

2. **Udev Rules**
   - Create rules to allow `spitfire` group members to access:
     - Loop devices
     - TAP devices
     - Other required system resources

3. **Filesystem Permissions**
   - Set up the Spitfire working directory structure
   - Configure appropriate permissions on:
     - Configuration directory (`~/.spitfire/config`)
     - Volume storage (`~/.spitfire/volumes`)
     - Init registry (`~/.spitfire/inits`)
     - State directory (`~/.spitfire/state`)
     - Kernel registry (`~/.spitfire/kernels`)

4. **Setup Verification**
   - Create a state file tracking setup completion
   - Verify setup state before operations
   - Provide clear error messages if setup is incomplete

### Privilege Separation

After setup, normal Spitfire operations will:
- Run without sudo privileges
- Use the configured user group permissions
- Access resources through proper udev rules
- Provide clear error messages if permissions issues occur

### Future Improvements

In future versions, we will research:
- User-space alternatives to loop devices (e.g., FUSE-based implementations)
- Further reduction of privileged operations
- VM filesystem access via SSHFS or similar mechanisms

## Next Steps

1. Begin implementation of the MVP:
   - Implement the `spitfire setup` command for initial system preparation
   - Create the configuration parser with validation
   - Build the Firecracker API integration for basic VM operations
   - Implement simple bridge networking
   - Enhance container-to-rootfs conversion
   - Develop kernel management subsystem
   - Design and implement minimal CLI for core operations

2. Create developer documentation:
   - Installation guide
   - Basic usage examples
   - Configuration reference

3. Test with key use cases:
   - Multi-tier application (web + database)
   - Development environment
   - Simple container workloads

4. Gather feedback and refine:
   - Identify pain points and usability issues
   - Prioritize post-MVP features
   - Create issues for future enhancements

5. Plan next phase of development based on user feedback and roadmap

## MVP Implementation Plan

### Week 1-2: Foundation
- Set up project structure and core libraries
- Implement configuration parser and validation
- Create the setup command
- Basic Firecracker API integration
- Initial kernel management implementation

### Week 3-4: Core Functionality
- Simple bridge networking
- Basic volume management
- Container image handling
- Init system integration
- Kernel registry implementation

### Week 5-6: User Experience
- CLI implementation for core commands
- Error handling and user feedback
- Documentation
- Example configurations

### Week 7-8: Testing and Refinement
- End-to-end testing with sample workloads
- Performance testing and optimization
- Bug fixes and usability improvements
- Prepare for initial release
