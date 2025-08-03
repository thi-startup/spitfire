package state

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/util/homedir"
)

const (
	// SpitfireHome is the environment variable for spitfire home directory
	SpitfireHome = "SPITFIRE_HOME"

	// DefaultDirName is the default directory name under user home
	DefaultDirName = ".spitfire"
)

// SpitfirePath returns the path to the user's spitfire directory
// Follows minikube's pattern: SPITFIRE_HOME env var or ~/.spitfire
func SpitfirePath() string {
	spitfireHomeEnv := os.Getenv(SpitfireHome)
	if spitfireHomeEnv != "" {
		if filepath.Base(spitfireHomeEnv) == DefaultDirName {
			return spitfireHomeEnv
		}
		return filepath.Join(spitfireHomeEnv, DefaultDirName)
	}
	return filepath.Join(homedir.HomeDir(), DefaultDirName)
}

// MakeSpitfirePath constructs a path relative to spitfire home
func MakeSpitfirePath(segments ...string) string {
	args := []string{SpitfirePath()}
	args = append(args, segments...)
	return filepath.Join(args...)
}

// GlobalConfigFile returns the path to the global configuration file
func GlobalConfigFile() string {
	return MakeSpitfirePath("config", "config.json")
}

// ProjectPath returns the path to a project's directory
func ProjectPath(name string) string {
	return MakeSpitfirePath("projects", name)
}

// ProjectStateFile returns the path to a project's state file
func ProjectStateFile(name string) string {
	return filepath.Join(ProjectPath(name), "state.json")
}

// ProjectEventsFile returns the path to a project's events log
func ProjectEventsFile(name string) string {
	return filepath.Join(ProjectPath(name), "events.json")
}

// ProjectConfigFile returns the path to a project's resolved config
func ProjectConfigFile(name string) string {
	return filepath.Join(ProjectPath(name), "config.json")
}

// VMPath returns the path to a VM's directory within a project
func VMPath(projectName, vmName string) string {
	return filepath.Join(ProjectPath(projectName), "vms", vmName)
}

// VMStateFile returns the path to a VM's state file
func VMStateFile(projectName, vmName string) string {
	return filepath.Join(VMPath(projectName, vmName), "state.json")
}

// VMLogFile returns the path to a VM's log file
func VMLogFile(projectName, vmName string) string {
	return filepath.Join(VMPath(projectName, vmName), "console.log")
}

// NetworkPath returns the path to network state
func NetworkPath(projectName string) string {
	return filepath.Join(ProjectPath(projectName), "networks")
}

// VolumePath returns the path to volume state
func VolumePath(projectName string) string {
	return filepath.Join(ProjectPath(projectName), "volumes")
}

// CertificatePath returns the path to certificates
func CertificatePath() string {
	return MakeSpitfirePath("certs")
}

// GlobalCACert returns the path to the global CA certificate
func GlobalCACert() string {
	return MakeSpitfirePath("ca.crt")
}

// GlobalCAKey returns the path to the global CA private key
func GlobalCAKey() string {
	return MakeSpitfirePath("ca.key")
}

// LogsPath returns the path to logs directory
func LogsPath() string {
	return MakeSpitfirePath("logs")
}

// AuditLogFile returns the path to the audit log
func AuditLogFile() string {
	return filepath.Join(LogsPath(), "audit.json")
}

// CachePath returns the path to cache directory
func CachePath() string {
	return MakeSpitfirePath("cache")
}

// DriverCachePath returns the path to driver-specific cache
func DriverCachePath(driver string) string {
	return filepath.Join(CachePath(), "drivers", driver)
}

// ImageCachePath returns the path to image cache
func ImageCachePath() string {
	return filepath.Join(CachePath(), "images")
}

// EnsureDir creates a directory if it doesn't exist
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// EnsureProjectDirs creates all necessary directories for a project
func EnsureProjectDirs(projectName string) error {
	dirs := []string{
		ProjectPath(projectName),
		VMPath(projectName, ""), // Creates the vms subdirectory
		NetworkPath(projectName),
		VolumePath(projectName),
	}

	for _, dir := range dirs {
		if err := EnsureDir(dir); err != nil {
			return err
		}
	}

	return nil
}

// EnsureGlobalDirs creates all necessary global directories
func EnsureGlobalDirs() error {
	dirs := []string{
		filepath.Dir(GlobalConfigFile()), // config directory
		CertificatePath(),
		LogsPath(),
		CachePath(),
		DriverCachePath(""), // drivers cache parent
		ImageCachePath(),
	}

	for _, dir := range dirs {
		if err := EnsureDir(dir); err != nil {
			return err
		}
	}

	return nil
}

// Additional utility functions consolidated from pkg/utils

// Exists checks if a file or directory exists
func Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// VerifyBinary checks if a binary exists and is executable
func VerifyBinary(path string) error {
	finfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("binary %q does not exist", path)
	}
	if err != nil {
		return fmt.Errorf("stat binary %q: %w", path, err)
	}
	if finfo.IsDir() {
		return fmt.Errorf("binary %q is a directory", path)
	}
	const executableMask = 0111 // File permission bits for executable
	if finfo.Mode()&executableMask == 0 {
		return fmt.Errorf("binary %q is not executable", path)
	}
	return nil
}

// Additional convenience functions

// KernelsPath returns the path to kernel directory
func KernelsPath() string {
	return MakeSpitfirePath("kernels")
}

// InitsPath returns the path to init systems directory
func InitsPath() string {
	return MakeSpitfirePath("inits")
}

// TempPath returns the path to temporary files directory
func TempPath() string {
	return MakeSpitfirePath("tmp")
}

// EnsureVMDirs creates all necessary directories for a VM
func EnsureVMDirs(projectName, vmName string) error {
	return EnsureDir(VMPath(projectName, vmName))
}
