package vm

import (
	"context"
	"fmt"
	"os/exec"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	log "github.com/sirupsen/logrus"
	"github.com/thi-startup/spitfire/internal/config"
	"github.com/thi-startup/spitfire/pkg/utils"
)

func StartVMM(ctx context.Context, opts *config.VMMOption) (*firecracker.Machine, error) {
	fcCfg, err := opts.GetFirecrackerConfig()
	if err != nil {
		log.Errorf("Error: %s", err)
		return nil, err
	}
	logger := log.New()

	if opts.Debug {
		log.SetLevel(log.DebugLevel)
		logger.SetLevel(log.DebugLevel)
	}

	vmmCtx, vmmCancel := context.WithCancel(ctx)
	defer vmmCancel()

	machineOpts := []firecracker.Opt{
		firecracker.WithLogger(log.NewEntry(logger)),
	}

	var firecrackerBinary string
	if len(opts.FcBinary) != 0 {
		firecrackerBinary = opts.FcBinary
	} else {
		firecrackerBinary, err = exec.LookPath(config.FirecrackerDefaultPath)
		if err != nil {
			return nil, err
		}
	}

	if err := utils.VerifyBinary(firecrackerBinary); err != nil {
		return nil, fmt.Errorf("invalid firecracker binary provided: %w", err)
	}

	// if the jailer is used, the final command will be built in NewMachine()
	// TODO(joe): we are not supposed to start the vms with stdin, stdout or stderr
	// because we might be doing multiple of them concurrently if its allowed
	if fcCfg.JailerCfg == nil {
		cmd := firecracker.VMCommandBuilder{}.
			WithBin(firecrackerBinary).
			WithSocketPath(fcCfg.SocketPath).
			Build(ctx)
			// WithStdin(os.Stdin).
			// WithStdout(os.Stdout).
			// WithStderr(os.Stderr).

		machineOpts = append(machineOpts, firecracker.WithProcessRunner(cmd))
	}

	m, err := firecracker.NewMachine(vmmCtx, fcCfg, machineOpts...)
	if err != nil {
		return nil, fmt.Errorf("Failed creating machine: %s", err)
	}

	if err := m.Start(vmmCtx); err != nil {
		return nil, fmt.Errorf("Failed to start machine: %v", err)
	}

	// defer func() {
	// 	if err := m.StopVMM(); err != nil {
	// 		log.Errorf("An error occurred while stopping Firecracker VMM: %v", err)
	// 	}
	// }()

	if opts.ValidMetadata != nil {
		if err := m.SetMetadata(vmmCtx, opts.ValidMetadata); err != nil {
			log.Errorf("An error occurred while setting Firecracker VM metadata: %v", err)
		}
	}

	// installSignalHandlers(vmmCtx, m)

	// wait for the VMM to exit
	// if err := m.Wait(vmmCtx); err != nil {
	// 	return fmt.Errorf("Wait returned an error %s", err)
	// }
	// log.Printf("Start machine was happy")
	return m, nil
}
