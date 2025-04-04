package config

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	models "github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/opentracing/opentracing-go/log"
	"gopkg.in/yaml.v2"
)

const (
	FirecrackerDefaultPath = "firecracker"
	ExecutableMask         = 0b111
)

type Config struct {
	Version    string               `yaml:"version"`
	VMMConfigs map[string]VMMOption `yaml:"vms"`
}

type VMMOption struct {
	FcBinary           string   `yaml:"firecracker-binary" long:"firecracker-binary" description:"Path to firecracker binary"`
	FcKernelImage      string   `yaml:"kernel" long:"kernel" description:"Path to the kernel image" default:"./vmlinux"`
	FcKernelCmdLine    string   `yaml:"kernel-opts" long:"kernel-opts" description:"Kernel commandline" default:"ro console=ttyS0 noapic reboot=k panic=1 pci=off nomodules"`
	FcInitrd           string   `yaml:"initrd-path" long:"initrd-path" description:"Path to initrd"`
	FcRootDrivePath    string   `yaml:"root-drive" long:"root-drive" description:"Path to root disk image"`
	FcRootPartUUID     string   `yaml:"root-partition" long:"root-partition" description:"Root partition UUID"`
	FcAdditionalDrives []string `yaml:"add-drive" long:"add-drive" description:"Path to additional drive, suffixed with :ro or :rw, can be specified multiple times"`
	FcNicConfig        []string `yaml:"tap-device" long:"tap-device" description:"NIC info, specified as DEVICE/MAC, can be specified multiple times"`
	FcVsockDevices     []string `yaml:"vsock-device" long:"vsock-device" description:"Vsock interface, specified as PATH:CID. Multiple OK"`
	FcLogFifo          string   `yaml:"vmm-log-fifo" long:"vmm-log-fifo" description:"FIFO for firecracker logs"`
	FcLogLevel         string   `yaml:"log-level" long:"log-level" description:"vmm log level" default:"Debug"`
	FcMetricsFifo      string   `yaml:"metrics-fifo" long:"metrics-fifo" description:"FIFO for firecracker metrics"`
	FcDisableSmt       bool     `yaml:"disable-smt" long:"disable-smt" short:"t" description:"Disable CPU Simultaneous Multithreading"`
	FcCPUCount         int64    `yaml:"ncpus" long:"ncpus" description:"Number of CPUs" default:"1"`
	FcCPUTemplate      string   `yaml:"cpu-template" long:"cpu-template" description:"Firecracker CPU Template (C3 or T2)"`
	FcMemSz            int64    `yaml:"memory" long:"memory" short:"m" description:"VM memory, in MiB" default:"512"`
	FcMetadata         string   `yaml:"metadata" long:"metadata" description:"Firecracker Metadata for MMDS (json)"`
	FcFifoLogFile      string   `yaml:"firecracker-log" long:"firecracker-log" short:"l" description:"pipes the fifo contents to the specified file"`
	FcSocketPath       string   `yaml:"socket-path" long:"socket-path" short:"s" description:"path to use for firecracker socket, defaults to a unique file in in the first existing directory from {$HOME, $TMPDIR, or /tmp}"`
	Debug              bool     `yaml:"debug" long:"debug" short:"d" description:"Enable debug output"`

	Id           string `yaml:"id" long:"id" description:"Jailer VMM id"`
	ExecFile     string `yaml:"exec-file" long:"exec-file" description:"Jailer executable"`
	JailerBinary string `yaml:"jailer" long:"jailer" description:"Jailer binary"`

	Uid      int `yaml:"uid" long:"uid" description:"Jailer uid for dropping privileges"`
	Gid      int `yaml:"gid" long:"gid" description:"Jailer gid for dropping privileges"`
	NumaNode int `yaml:"node" long:"node" description:"Jailer numa node"`

	ChrootBaseDir string `yaml:"chroot-base-dir" long:"chroot-base-dir" description:"Jailer chroot base directory"`
	Daemonize     bool   `yaml:"daemonize" long:"daemonize" description:"Run jailer as daemon"`

	closers       []func() error
	ValidMetadata interface{}

	createFifoFileLogs func(fifoPath string) (*os.File, error)
}

func NewVMMOption() *VMMOption {
	return &VMMOption{
		createFifoFileLogs: createFifoFileLogs,
	}
}

func ParseYAML(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("%w, while parsing yaml: %w", ErrInvalidConfig, err)
	}
	return &config, nil
}

// Converts options to a usable firecracker config
func (opts *VMMOption) GetFirecrackerConfig() (firecracker.Config, error) {
	// validate metadata json
	if opts.FcMetadata != "" {
		if err := json.Unmarshal([]byte(opts.FcMetadata), &opts.ValidMetadata); err != nil {
			return firecracker.Config{}, fmt.Errorf("%s: %v", ErrInvalidMetadata.Error(), err)
		}
	}
	//setup NICs
	NICs, err := opts.getNetwork()
	if err != nil {
		return firecracker.Config{}, err
	}
	// BlockDevices
	blockDevices, err := opts.getBlockDevices()
	if err != nil {
		return firecracker.Config{}, err
	}

	// vsocks
	vsocks, err := parseVsocks(opts.FcVsockDevices)
	if err != nil {
		return firecracker.Config{}, err
	}

	//fifos
	fifo, err := opts.handleFifos()
	if err != nil {
		return firecracker.Config{}, err
	}

	var (
		socketPath string
		jail       *firecracker.JailerConfig
	)

	if opts.JailerBinary != "" {
		jail = &firecracker.JailerConfig{
			GID:            firecracker.Int(opts.Gid),
			UID:            firecracker.Int(opts.Uid),
			ID:             opts.Id,
			NumaNode:       firecracker.Int(opts.NumaNode),
			ExecFile:       opts.ExecFile,
			JailerBinary:   opts.JailerBinary,
			ChrootBaseDir:  opts.ChrootBaseDir,
			Daemonize:      opts.Daemonize,
			ChrootStrategy: firecracker.NewNaiveChrootStrategy(opts.FcKernelImage),
			Stdout:         os.Stdout,
			Stderr:         os.Stderr,
			Stdin:          os.Stdin,
		}
	} else {

		// if no jail is active, either use the path from the arguments
		if opts.FcSocketPath != "" {
			socketPath = opts.FcSocketPath
		} else {
			// or generate a default socket path
			socketPath = getSocketPath()
		}
	}

	return firecracker.Config{
		SocketPath:        socketPath,
		LogFifo:           opts.FcLogFifo,
		LogLevel:          opts.FcLogLevel,
		MetricsFifo:       opts.FcMetricsFifo,
		FifoLogWriter:     fifo,
		KernelImagePath:   opts.FcKernelImage,
		KernelArgs:        opts.FcKernelCmdLine,
		InitrdPath:        opts.FcInitrd,
		Drives:            blockDevices,
		NetworkInterfaces: NICs,
		VsockDevices:      vsocks,
		MachineCfg: models.MachineConfiguration{
			VcpuCount:   firecracker.Int64(opts.FcCPUCount),
			CPUTemplate: models.CPUTemplate(opts.FcCPUTemplate),
			Smt:         firecracker.Bool(!opts.FcDisableSmt),
			MemSizeMib:  firecracker.Int64(opts.FcMemSz),
		},
		JailerCfg: jail,
		VMID:      opts.Id,
	}, nil
}

func (opts *VMMOption) getNetwork() ([]firecracker.NetworkInterface, error) {
	var NICs []firecracker.NetworkInterface
	if len(opts.FcNicConfig) > 0 {
		for _, nicConfig := range opts.FcNicConfig {
			tapDev, tapMacAddr, err := parseNicConfig(nicConfig)
			if err != nil {
				return nil, err
			}
			allowMMDS := opts.ValidMetadata != nil
			nic := firecracker.NetworkInterface{
				StaticConfiguration: &firecracker.StaticNetworkConfiguration{
					MacAddress:  tapMacAddr,
					HostDevName: tapDev,
				},
				AllowMMDS: allowMMDS,
			}
			NICs = append(NICs, nic)
		}
	}
	return NICs, nil
}

// constructs a list of drives from the options config
func (opts *VMMOption) getBlockDevices() ([]models.Drive, error) {
	blockDevices, err := parseBlockDevices(opts.FcAdditionalDrives)
	if err != nil {
		return nil, err
	}

	rootDrivePath, readOnly := parseDevice(opts.FcRootDrivePath)
	rootDrive := models.Drive{
		DriveID:      firecracker.String("1"),
		PathOnHost:   firecracker.String(rootDrivePath),
		IsReadOnly:   firecracker.Bool(readOnly),
		IsRootDevice: firecracker.Bool(true),
		Partuuid:     opts.FcRootPartUUID,
	}
	blockDevices = append(blockDevices, rootDrive)
	return blockDevices, nil
}

// handleFifos will see if any fifos need to be generated and if a fifo log
// file should be created.
func (opts *VMMOption) handleFifos() (io.Writer, error) {
	// these booleans are used to check whether or not the fifo queue or metrics
	// fifo queue needs to be generated. If any which need to be generated, then
	// we know we need to create a temporary directory. Otherwise, a temporary
	// directory does not need to be created.
	generateFifoFilename := false
	generateMetricFifoFilename := false
	var err error
	var fifo io.WriteCloser

	if len(opts.FcFifoLogFile) > 0 {
		if len(opts.FcLogFifo) > 0 {
			return nil, ErrConflictingLogOpts
		}
		generateFifoFilename = true
		// if a fifo log file was specified via the CLI then we need to check if
		// metric fifo was also specified. If not, we will then generate that fifo
		if len(opts.FcMetricsFifo) == 0 {
			generateMetricFifoFilename = true
		}
		if fifo, err = opts.createFifoFileLogs(opts.FcFifoLogFile); err != nil {
			return nil, fmt.Errorf("%s: %v", ErrUnableToCreateFifoLogFile.Error(), err)
		}
		opts.addCloser(func() error {
			return fifo.Close()
		})

	} else if len(opts.FcLogFifo) > 0 || len(opts.FcMetricsFifo) > 0 {
		// this checks to see if either one of the fifos was set. If at least one
		// has been set we check to see if any of the others were not set. If one
		// isn't set, we will generate the proper file path.
		if len(opts.FcLogFifo) == 0 {
			generateFifoFilename = true
		}

		if len(opts.FcMetricsFifo) == 0 {
			generateMetricFifoFilename = true
		}
	}

	if generateFifoFilename || generateMetricFifoFilename {
		dir, err := os.MkdirTemp(os.TempDir(), "fcfifo")
		if err != nil {
			return fifo, fmt.Errorf("fail to create temporary directory: %v", err)
		}
		opts.addCloser(func() error {
			return os.RemoveAll(dir)
		})
		if generateFifoFilename {
			opts.FcLogFifo = filepath.Join(dir, "fc_fifo")
		}

		if generateMetricFifoFilename {
			opts.FcMetricsFifo = filepath.Join(dir, "fc_metrics_fifo")
		}
	}

	return fifo, nil
}

func (opts *VMMOption) addCloser(c func() error) {
	opts.closers = append(opts.closers, c)
}

func (opts *VMMOption) Close() {
	for _, closer := range opts.closers {
		err := closer()
		if err != nil {
			log.Error(err)
		}
	}
}

const (
	rwDeviceSuffix = ":rw"
	roDeviceSuffix = ":ro"
)

// Given a string in the form of path:suffix return the path and read-only marker
func parseDevice(entry string) (path string, readOnly bool) {
	if strings.HasSuffix(entry, roDeviceSuffix) {
		return strings.TrimSuffix(entry, roDeviceSuffix), true
	}

	return strings.TrimSuffix(entry, rwDeviceSuffix), false
}

// given a []string in the form of path:suffix converts to []models.Drive
func parseBlockDevices(entries []string) ([]models.Drive, error) {
	devices := []models.Drive{}

	for i, entry := range entries {
		path := ""
		readOnly := true

		if strings.HasSuffix(entry, rwDeviceSuffix) {
			readOnly = false
			path = strings.TrimSuffix(entry, rwDeviceSuffix)
		} else if strings.HasSuffix(entry, roDeviceSuffix) {
			path = strings.TrimSuffix(entry, roDeviceSuffix)
		} else {
			return nil, ErrInvalidDriveSpecificationNoSuffix
		}

		if path == "" {
			return nil, ErrInvalidDriveSpecificationNoPath
		}

		if _, err := os.Stat(path); err != nil {
			return nil, err
		}

		e := models.Drive{
			// i + 2 represents the drive ID. We will reserve 1 for root.
			DriveID:      firecracker.String(strconv.Itoa(i + 2)),
			PathOnHost:   firecracker.String(path),
			IsReadOnly:   firecracker.Bool(readOnly),
			IsRootDevice: firecracker.Bool(false),
		}
		devices = append(devices, e)
	}
	return devices, nil
}

// Given a string of the form DEVICE/MACADDR, return the device name and the mac address, or an error
func parseNicConfig(cfg string) (string, string, error) {
	fields := strings.Split(cfg, "/")
	if len(fields) != 2 || len(fields[0]) == 0 || len(fields[1]) == 0 {
		return "", "", ErrInvalidNicConfig
	}
	return fields[0], fields[1], nil
}

// Given a list of string representations of vsock devices,
// return a corresponding slice of machine.VsockDevice objects
func parseVsocks(devices []string) ([]firecracker.VsockDevice, error) {
	var result []firecracker.VsockDevice
	for _, entry := range devices {
		fields := strings.Split(entry, ":")
		if len(fields) != 2 || len(fields[0]) == 0 || len(fields[1]) == 0 {
			return []firecracker.VsockDevice{}, ErrUnableToParseVsockDevices
		}
		CID, err := strconv.ParseUint(fields[1], 10, 32)
		if err != nil {
			return []firecracker.VsockDevice{}, ErrUnableToParseVsockCID
		}
		dev := firecracker.VsockDevice{
			Path: fields[0],
			CID:  uint32(CID),
		}
		result = append(result, dev)
	}
	return result, nil
}

func createFifoFileLogs(fifoPath string) (*os.File, error) {
	return os.OpenFile(fifoPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
}

// getSocketPath provides a randomized socket path by building a unique filename
// and searching for the existence of directories {$HOME, os.TempDir()} and returning
// the path with the first directory joined with the unique filename. If we can't
// find a good path panics.
func getSocketPath() string {
	filename := strings.Join([]string{
		".firecracker.sock",
		strconv.Itoa(os.Getpid()),
		strconv.Itoa(rand.Intn(1000))},
		"-",
	)
	var dir string
	if d := os.Getenv("HOME"); checkExistsAndDir(d) {
		dir = d
	} else if checkExistsAndDir(os.TempDir()) {
		dir = os.TempDir()
	} else {
		panic("Unable to find a location for firecracker socket. 'It's not going to do any good to land on mars if we're stupid.' --Ray Bradbury")
	}

	return filepath.Join(dir, filename)
}

// checkExistsAndDir returns true if path exists and is a Dir
func checkExistsAndDir(path string) bool {
	// empty
	if path == "" {
		return false
	}
	// does it exist?
	if info, err := os.Stat(path); err == nil {
		// is it a directory?
		return info.IsDir()
	}
	return false
}
