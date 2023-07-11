package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

func Mkroot() *cobra.Command {
	command := &cobra.Command{
		Use:     "mkroot",
		Short:   "Create root drive",
		Aliases: []string{"mkroot"},
		Example: `spitfire mkroot --size 10M --fs ext4 --name file.ext4
name=$(mktemp -d) && echo "hello world!" > $file/hello && sudo spitfire mkroot --name file.ext4 --size 10M --fs ext4 --cp $file`,
		SilenceUsage: true,
	}

	command.Flags().StringP("size", "s", "", "size of the root device")
	command.Flags().StringP("name", "n", "", "name of the root device")
	command.Flags().StringP("fs", "f", "ext4", "filesystem type")
	command.Flags().StringP("cp", "c", "", "directory to copy root device files from")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		size, _ := cmd.Flags().GetString("size")
		if size == "" {
			return fmt.Errorf("specify size of root drive")
		}

		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			return fmt.Errorf("specify name of root drive")
		}

		fstype, _ := cmd.Flags().GetString("fs")

		mkroot, err := Mkfs()
		if err != nil {
			return err
		}

		from, _ := cmd.Flags().GetString("cp")
		dirpath, err := filepath.Abs(from)
		if err != nil {
			return fmt.Errorf("failed to get absolute path of '%s': %v", from, err)
		}
		if dirpath != "" && !Exists(dirpath) {
			return fmt.Errorf("specify valid directory to copy root device files from")
		}

		return mkroot.Name(name).Size(size).Type(fstype).DirPath(dirpath).Execute()
	}

	return command
}
