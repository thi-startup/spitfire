package firecracker

import (
	"fmt"
	"strings"
)

// KernelCmdLine represents kernel command line parameters
type KernelCmdLine map[string]string

// DefaultKernelCmdLine returns the default recommended kernel parameters for Firecracker
// Based on Flintlock's defaults with security and performance optimizations
func DefaultKernelCmdLine() KernelCmdLine {
	return KernelCmdLine{
		"console":       "ttyS0", // Serial console output
		"reboot":        "k",     // Keyboard reboot
		"panic":         "1",     // Reboot after 1 second on panic
		"pci":           "off",   // Disable PCI bus probing
		"i8042.noaux":   "",      // Disable auxiliary port (mouse)
		"i8042.nomux":   "",      // Disable multiplexing controller
		"i8042.nopnp":   "",      // Disable PnP discovery
		"i8042.dumbkbd": "",      // Simple keyboard mode
		"8250.nr_uarts": "0",     // Disable extra UARTs
		"ipv6.disable":  "1",     // Disable IPv6 for simplicity
	}
}

// Set adds or updates a kernel parameter
func (k KernelCmdLine) Set(key, value string) {
	k[key] = value
}

// Delete removes a kernel parameter
func (k KernelCmdLine) Delete(key string) {
	delete(k, key)
}

// String converts the kernel command line to a string
func (k KernelCmdLine) String() string {
	if len(k) == 0 {
		return ""
	}

	var parts []string
	for key, value := range k {
		if value == "" {
			parts = append(parts, key)
		} else {
			parts = append(parts, fmt.Sprintf("%s=%s", key, value))
		}
	}

	return strings.Join(parts, " ")
}

// Clone creates a copy of the kernel command line
func (k KernelCmdLine) Clone() KernelCmdLine {
	clone := make(KernelCmdLine)
	for key, value := range k {
		clone[key] = value
	}
	return clone
}

// MergeFrom merges parameters from another kernel command line
// Parameters in 'other' will override existing parameters
func (k KernelCmdLine) MergeFrom(other KernelCmdLine) {
	for key, value := range other {
		k[key] = value
	}
}

// GetNetworkKernelArgs returns kernel arguments for network configuration
// This can be used when we want the kernel to configure networking
func GetNetworkKernelArgs(ip, gateway, netmask string) KernelCmdLine {
	args := make(KernelCmdLine)

	if ip != "" && gateway != "" && netmask != "" {
		// Format: ip=<client-ip>:<server-ip>:<gw-ip>:<netmask>:<hostname>:<device>:<autoconf>
		ipConfig := fmt.Sprintf("%s::%s:%s:::off", ip, gateway, netmask)
		args.Set("ip", ipConfig)
	}

	return args
}

// GetCloudInitKernelArgs returns kernel arguments for cloud-init
func GetCloudInitKernelArgs(datasourceURL string) KernelCmdLine {
	args := make(KernelCmdLine)

	if datasourceURL != "" {
		args.Set("ds", fmt.Sprintf("nocloud-net;s=%s", datasourceURL))
	}

	return args
}

// GetDebugKernelArgs returns kernel arguments useful for debugging
func GetDebugKernelArgs() KernelCmdLine {
	return KernelCmdLine{
		"loglevel":        "8",                   // Maximum verbosity
		"debug":           "",                    // Enable debug output
		"ignore_loglevel": "",                    // Show all kernel messages
		"earlyprintk":     "serial,ttyS0,115200", // Early serial output
	}
}

// GetSecurityKernelArgs returns security-focused kernel arguments
func GetSecurityKernelArgs() KernelCmdLine {
	return KernelCmdLine{
		"nokaslr":                   "",    // Disable KASLR for consistent memory layout
		"nosmap":                    "",    // Disable SMAP for compatibility
		"nosmep":                    "",    // Disable SMEP for compatibility
		"spectre_v2":                "off", // Disable spectre v2 mitigations
		"spec_store_bypass_disable": "off", // Disable speculative store bypass
		"l1tf":                      "off", // Disable L1TF mitigations
		"mds":                       "off", // Disable MDS mitigations
		"tsx_async_abort":           "off", // Disable TSX async abort mitigations
	}
}
