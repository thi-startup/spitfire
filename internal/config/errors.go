package config

import "errors"

var (
	ErrInvalidConfig    = errors.New("invalid spitfire configuration")
	ErrInvalidNicConfig = errors.New("NIC config wasn't of the form DEVICE/MACADDR")

	// Error parsing blockdevices
	ErrInvalidDriveSpecificationNoSuffix = errors.New("invalid drive specification. Must have :rw or :ro suffix")
	ErrInvalidDriveSpecificationNoPath   = errors.New("invalid drive specification. Must have path")

	// Error parsing vsock
	ErrUnableToParseVsockDevices = errors.New("unable to parse vsock devices")
	ErrUnableToParseVsockCID     = errors.New("unable to parse vsock CID as a number")

	ErrConflictingLogOpts        = errors.New("vmm-log-fifo and firecracker-log cannot be used together")
	ErrUnableToCreateFifoLogFile = errors.New("failed to create fifo log file")

	// Error with firecracker config
	ErrInvalidMetadata = errors.New("invalid metadata, unable to parse as json")
)
