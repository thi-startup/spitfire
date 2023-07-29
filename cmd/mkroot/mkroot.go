package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

func Mkroot() *cobra.Command {
	command := &cobra.Command{
		Use:     "mkroot",
		Short:   "Create a loop device",
		Aliases: []string{"mkroot"},
		Example: `
Make a 10 megabyte loop device with the EXT4 filesystem
  $ spitfire mkroot --size 10M --fs ext4 --name file.ext4

Make loop device from an already existing directory
  $ name=$(mktemp -d) && echo "hello world!" > $file/hello && sudo spitfire mkroot --name file.ext4 --size 10M --fs ext4 --cp $file

Bundle github.com/thi-startup/init into a loop device for testing with firecracker
  $ sudo spitfire mkroot --name tmpinit --fs ext2 --size 100M --init

Also bundle with a local copy of repository
  $ sudo spitfire mkroot --name tmpinit --fs ext2 --size 100M --init --build-from <path/to/repo>
`,
		SilenceUsage: true,
	}

	command.Flags().StringP("size", "s", "", "size of the loop device")
	command.Flags().String("image", "", "name of the container image to unpack into a rootfs")
	command.Flags().StringP("name", "n", "", "name of the loop device")
	command.Flags().StringP("fs", "f", "ext4", "filesystem type. eg: ext2, ext4, btrfs, etc)")
	command.Flags().BoolP("init", "i", false, "create a loop device containing the init of the system. Default behavior is to 'go install github.com/thi-startup/init' and use the resulting binary. Use '--build-from' to specify a local repository of init code (needs go1.19+)")
	command.Flags().StringP("build-from", "b", "", "make device containing the init from a local copy of github.com/thi-startup/init (needs go1.19+)")
	command.Flags().StringP("cp", "c", "", "copy specified directory tree into loop device")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		size, _ := cmd.Flags().GetString("size")
		if size == "" {
			cmd.Help()
			return fmt.Errorf("specify size of root drive")
		}

		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			cmd.Help()
			return fmt.Errorf("specify name of root drive")
		}

		fstype, _ := cmd.Flags().GetString("fs")

		from, _ := cmd.Flags().GetString("cp")
		dirpath, err := filepath.Abs(from)
		if from != "" && err != nil {
			return fmt.Errorf("failed to get absolute path of '%s': %v", from, err)
		} else if !Exists(dirpath) {
			return fmt.Errorf("specify valid directory to copy root device files from")
		}

		init, _ := cmd.Flags().GetBool("init")
		buildDir, _ := cmd.Flags().GetString("build-from")

		mkroot, err := Mkfs()
		if err != nil {
			return err
		}

		image, _ := cmd.Flags().GetString("image")
		fromImage := len(image) != 0

		return mkroot.Name(name).Size(size).Type(fstype).DirPath(dirpath).MakeInit(init, buildDir).Image(fromImage, image).Execute()
	}

	return command
}
