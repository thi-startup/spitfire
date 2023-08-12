package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	log "github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/thi-startup/spitfire/internal/config"
	"github.com/thi-startup/spitfire/internal/drive"
	"github.com/thi-startup/spitfire/internal/firectl"
	"github.com/thi-startup/spitfire/utils"
)

func MkRunCmd() *cobra.Command {
	command := &cobra.Command{
		Use:          "run <microvm>",
		Short:        "run a microvm",
		Aliases:      []string{"r"},
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
	}

	command.Flags().StringP("exec", "e", "", "command to exec in the microvm")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		name := args[0]

		mvm, err := getMicroVm(name)
		if err != nil {
			return fmt.Errorf("error getting microvm: %s", err)
		}

		initPath := filepath.Join(mvm, "tmpinit")

		if !utils.Exists(initPath) {
			return fmt.Errorf("init drive not found")
		}

		tmp, err := utils.Mktemp("init")
		if err != nil {
			return fmt.Errorf("error getting tmp dir: %w", err)
		}

		tmpinit := openExistingInitDrive(initPath, tmp)

		if err := tmpinit.MountLoopDrive(context.TODO()); err != nil {
			return fmt.Errorf("error mounting loop drive: %w", err)
		}

		defer func() {
			if err := tmpinit.UmountLoopDrive(context.TODO()); err != nil {
				log.Fatal(fmt.Errorf("error umounting init drive: %w", err))
			}
		}()

		cfgPath := filepath.Join(mvm, "run.json")

		initConfig, err := config.ReadInitConfig(cfgPath)
		if err != nil {
			return fmt.Errorf("error reading config: %w", err)
		}

		exec, _ := cmd.Flags().GetString("exec")

		if len(exec) != 0 {
			// TODO: implement this properly
			initConfig.CmdOverride = []string{exec}
		}

		thiDir := filepath.Join(tmp, "thi")

		// TODO: implement init reclaim functionality
		if initIsUsed(thiDir) {
			log.Infof("reclaiming init drive")
			if err := reclaimInitDrive(tmp); err != nil {
				return fmt.Errorf("error reclaiming init drive: %w", err)
			}

			if err := copyInitToDrive(thiDir); err != nil {
				return fmt.Errorf("error copying init to drive: %s", err)
			}
		}

		if err := config.WriteInitConfig(thiDir, initConfig); err != nil {
			return fmt.Errorf("error writing init config: %w", err)
		}

		if err := runMicroVm(context.TODO(), mvm); err != nil {
			return fmt.Errorf("error running microvm: %w", err)
		}

		return nil
	}

	return command
}

func getMicroVm(name string) (string, error) {
	vmCache, err := utils.MicroVmCache()
	if err != nil {
		return "", fmt.Errorf("error getting image cache: %w", err)
	}

	vm := filepath.Join(vmCache, name)

	if !utils.Exists(vm) {
		return "", fmt.Errorf("image %s doesn't exist, please create one", name)
	}

	return vm, nil
}

func initIsUsed(path string) bool {
	_, err := os.Stat(path)
	return errors.Is(err, fs.ErrNotExist)
}

func reclaimInitDrive(path string) error {
	newroot := filepath.Join(path, "newroot")
	if utils.Exists(newroot) {
		if err := os.RemoveAll(newroot); err != nil {
			return fmt.Errorf("error removing newroot directory: %w", err)
		}
	}

	dev := filepath.Join(path, "dev")
	if utils.Exists(dev) {
		if err := os.RemoveAll(dev); err != nil {
			return fmt.Errorf("error removing dev directory: %w", err)
		}
	}

	thi := filepath.Join(path, "thi")
	if !utils.Exists(thi) {
		if err := os.MkdirAll(thi, 0755); err != nil {
			return fmt.Errorf("error creating thi directory: %w", err)
		}
	}

	return nil
}

func openExistingInitDrive(path, target string) *drive.Drive {
	return drive.NewDrive(
		drive.WithName(path),
		drive.WithFstype("ext2"),
		drive.WithTarget(target),
		drive.WithSize(""),
	)
}

// TODO: improve this to be generic. As things stand right now this just works fine
// when booting a vm with our init.
func runMicroVm(ctx context.Context, vmName string) error {
	rootfs := filepath.Join(vmName, "rootfs.ext4:rw")
	kernel := filepath.Join(vmName, "vmlinux")
	init := filepath.Join(vmName, "tmpinit:rw")

	assets, err := utils.AssetsCache()
	if err != nil {
		return fmt.Errorf("error getting assets cache: %w", err)
	}

	binary := filepath.Join(assets, "firectl")

	err = firectl.NewFirectlCommand().
		WithBinary(binary).
		WithCNI("fcnet").
		WithKernelOpts("init=/thi/init").
		WithKernelImage(kernel).
		WithAdditionalDrives(rootfs).
		WithRootDrive(init).Build(ctx).Run()

	if err != nil {
		return fmt.Errorf("error running firectl command: %w", err)
	}

	return nil
}
