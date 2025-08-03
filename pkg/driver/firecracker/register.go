package firecracker

import (
	"fmt"

	"github.com/thi-startup/spitfire/pkg/driver"
)

// Register registers the Firecracker driver with the global registry
func Register() error {
	return driver.Register(driver.DriverDef{
		Name:        "firecracker",
		Aliases:     []string{"fc", "microvm"},
		Create:      NewFirecrackerDriver,
		Status:      status,
		Priority:    driver.HighlyPreferred,
		Default:     true,
		Description: "AWS Firecracker microVM driver for secure, fast, and lightweight virtualization",
	})
}

// init automatically registers the firecracker driver
func init() {
	if err := Register(); err != nil {
		panic(fmt.Sprintf("failed to register firecracker driver: %v", err))
	}
}
