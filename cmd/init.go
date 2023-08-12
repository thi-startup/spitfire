package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/thi-startup/spitfire/internal/get"
	"github.com/thi-startup/spitfire/utils"

	log "github.com/charmbracelet/log"
)

func MakeInitCmd() *cobra.Command {
	command := &cobra.Command{
		Use:          "init",
		Short:        "download assets if they don't exist",
		SilenceUsage: true,
	}

	command.RunE = func(cmd *cobra.Command, args []string) error {
		assetsDir, err := utils.AssetsCache()
		if err != nil {
			return fmt.Errorf("error getting assests cache: %w", err)
		}

		initPath := filepath.Join(assetsDir, "init")
		_, err = os.Stat(initPath)
		if err != nil {
			if os.IsNotExist(err) {
				log.Info("init binary not found, pulling one from github...")
				if err := get.DownloadInitFromGithub(assetsDir); err != nil {
					return fmt.Errorf("error downloading init binary: %w", err)
				}
			}
		}

		log.Info("init binary is ready to go")

		vmlinuxPath := filepath.Join(assetsDir, "vmlinux")
		_, err = os.Stat(vmlinuxPath)
		if err != nil {
			if os.IsNotExist(err) {
				log.Info("vmlinux not found")
				return fmt.Errorf("vmlinux doesn't exist, please manually copy one to %s", assetsDir)
			}
		}

		log.Info("vmlinux binary is ready to go")

		firectlPath := filepath.Join(assetsDir, "firectl")
		_, err = os.Stat(firectlPath)
		if err != nil {
			if os.IsNotExist(err) {
				log.Info("firectl not found, pulling one from github...")
				if err := get.DownloadFirectlFromGithub(assetsDir); err != nil {
					return fmt.Errorf("error downloading firectl binary: %w", err)
				}
			}
		}

		log.Info("firectl binary is ready to go")

		return nil
	}

	return command
}
