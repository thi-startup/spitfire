package main

import (
	"fmt"

	"github.com/thi-startup/spitfire/pkg/config"
	"github.com/thi-startup/spitfire/pkg/driver"
	"github.com/thi-startup/spitfire/pkg/log"
)

// Helper functions shared across commands

// findConfigFile finds the configuration file to use
func findConfigFile(configFile string) (string, error) {
	return config.FindConfigFile(configFile)
}

// loadConfig loads and validates the configuration
func loadConfig(configFile string) (*config.Config, error) {
	return config.LoadConfig(configFile)
}

// convertVMToDriverConfig converts a VM configuration to driver configuration
// Uses the canonical mapping from pkg/config instead of duplicated logic
func convertVMToDriverConfig(vmName string, vm *config.VM, cfg *config.Config) (*driver.Config, error) {
	driverConfig, err := vm.ToDriverConfig(vmName, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to convert VM config: %w", err)
	}

	// Log debug information
	log.Debugf("Converting VM config: image=%s, rootfs=%s, memory=%dMB, cpus=%d",
		driverConfig.Image, driverConfig.Rootfs, driverConfig.Memory, driverConfig.CPUs)

	return driverConfig, nil
}
