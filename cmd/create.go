package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/thi-startup/spitfire/internal/config"
	"github.com/thi-startup/spitfire/internal/drive"
	"github.com/thi-startup/spitfire/internal/get"
	"github.com/thi-startup/spitfire/utils"
	"golang.org/x/net/context"
)

type rootfs struct {
	size   string
	fstype string
	name   string
	image  string
	init   bool
	path   string
}

func MakeCreateCmd() *cobra.Command {
	command := &cobra.Command{
		Use:          "create [flags] <image>",
		Short:        "create a microvm",
		Aliases:      []string{"c"},
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
	}

	command.Flags().StringP("size", "s", "", "size of the rootfs")
	command.Flags().StringP("fstype", "f", "ext4", "filesystem type. eg: ext2, ext4, btrfs, etc")
	command.Flags().StringP("name", "n", "rootfs.ext4", "name of the loop device (defaults to rootfs.ext4).")
	command.Flags().BoolP("init", "", false, "create a loop drive with the init flashed unto it")
	command.Flags().StringP("image", "i", "", "docker image to create rootfs from")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		fstype, _ := cmd.Flags().GetString("fstype")
		size, _ := cmd.Flags().GetString("size")
		init, _ := cmd.Flags().GetBool("init")
		image, _ := cmd.Flags().GetString("image")

		if len(image) == 0 {
			return fmt.Errorf("image flag must be specified")
		}

		vmName := args[0]

		vmCache, err := utils.MicroVmCache()
		if err != nil {
			return err
		}

		microvm := filepath.Join(vmCache, vmName)

		if err := os.Mkdir(microvm, 0775); err != nil {
			return err
		}

		rfs := &rootfs{
			size:   size,
			fstype: fstype,
			name:   name,
			image:  image,
			init:   init,
			path:   microvm,
		}

		ctx := context.Background()

		if err := CreateRootfsDrive(ctx, rfs); err != nil {
			return fmt.Errorf("error creating rootfs drive: %w", err)
		}

		log.Info("creating symlink of vmlinux")
		if err := utils.SymlinkVmlinux(microvm); err != nil {
			return fmt.Errorf("error creating symlink of vmlinux: %w", err)
		}

		return nil
	}

	return command
}

func CreateRootfsDrive(ctx context.Context, cfg *rootfs) error {
	tmp, err := utils.Mktemp("rootfs")
	if err != nil {
		return fmt.Errorf("error making temp dir: %w", err)
	}

	rootfs := drive.NewDrive(
		drive.WithTarget(tmp),
		drive.WithFstype(cfg.fstype),
		drive.WithPath(cfg.path),
		drive.WithName(cfg.name),
		drive.WithSize(cfg.size),
	)

	if err := rootfs.CreateLoopDrive(ctx); err != nil {
		return fmt.Errorf("error creating loop drive: %w", err)
	}
	log.Info("loop drive created")

	if err := rootfs.MountLoopDrive(ctx); err != nil {
		return fmt.Errorf("error mounting init drive: %w", err)
	}
	log.Info("mounted loop drive")

	defer func() {
		if err := rootfs.UmountLoopDrive(ctx); err != nil {
			log.Fatal(err)
		}
	}()

	img, err := get.NewImage()
	if err != nil {
		return fmt.Errorf("error getting image puller: %w", err)
	}

	imageConfig, err := img.Pull(cfg.image, tmp)
	if err != nil {
		return fmt.Errorf("error pulling image: %w", err)
	}

	if cfg.init {
		log.Info("creating init drive")
		imageCfg := config.NewInitFromImageConfig(imageConfig)

		if err := CreateInitDrive(ctx, cfg.path, imageCfg); err != nil {
			return fmt.Errorf("error creating init drive: %w", err)
		}

		if err := config.WriteInitConfig(cfg.path, imageCfg); err != nil {
			return fmt.Errorf("error writing init config: %w", err)
		}
	}

	return nil
}

func CreateInitDrive(ctx context.Context, path string, cfg *config.InitConfig) error {
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	tmp, err := utils.Mktemp("init")
	if err != nil {
		return fmt.Errorf("error making temp dir: %w", err)
	}

	tmpinit := drive.NewDrive(
		drive.WithPath(path),
		drive.WithName("tmpinit"),
		drive.WithSize("40M"),
		drive.WithFstype("ext2"),
		drive.WithTarget(tmp),
	)

	if err := tmpinit.CreateLoopDrive(ctx); err != nil {
		return fmt.Errorf("error creating loop drive: %w", err)
	}
	log.Info("loop drive created")

	if err := tmpinit.MountLoopDrive(ctx); err != nil {
		return fmt.Errorf("error mounting init drive: %w", err)
	}
	log.Info("mounted loop drive")

	defer func() {
		if err := tmpinit.UmountLoopDrive(ctx); err != nil {
			log.Fatal(err)
		}
	}()

	thiDir := filepath.Join(tmp, "thi")

	if err := os.Mkdir(thiDir, 0775); err != nil {
		return fmt.Errorf("error creating thi directory: %w", err)
	}

	if err := createInitDrive(thiDir, cfg); err != nil {
		return fmt.Errorf("error creating init drive: %w", err)
	}

	return nil
}

func createInitDrive(dst string, cfg *config.InitConfig) error {
	if err := copyInitToDrive(dst); err != nil {
		return fmt.Errorf("error copying init to drive: %w", err)
	}

	if err := config.WriteInitConfig(dst, cfg); err != nil {
		return fmt.Errorf("error writing config to drive: %w", err)
	}

	return nil
}

func copyInitToDrive(dstDir string) error {
	assetsDir, err := utils.AssetsCache()
	if err != nil {
		return fmt.Errorf("error getting assets directory: %w", err)
	}

	src := filepath.Join(assetsDir, "init")
	dst := filepath.Join(dstDir, "init")

	log.Info("copying contents to drive")
	if err := utils.CopyFile(src, dst); err != nil {
		return fmt.Errorf("error copying init binary: %w", err)
	}

	return nil
}

func getImagePath(name string) (string, error) {
	vmCache, err := utils.MicroVmCache()
	if err != nil {
		return "", fmt.Errorf("error getting microvm cache: %w", err)
	}

	image := filepath.Join(vmCache, name)

	_, err = os.Stat(image)
	if err == nil {
		return "", fmt.Errorf("image already exitsts")
	}

	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	if err := os.Mkdir(image, 0775); err != nil {
		return "", fmt.Errorf("error creating image directory: %w", err)
	}

	return "", nil
}
