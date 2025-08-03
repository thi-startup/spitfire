package main

import (
	"github.com/spf13/cobra"
	"github.com/thi-startup/spitfire/pkg/config"
	"github.com/thi-startup/spitfire/pkg/log"
)

// newConfigCommands creates the config subcommand with all its sub-commands
func newConfigCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile, _ := cmd.Flags().GetString("file")
			quiet, _ := cmd.Flags().GetBool("quiet")
			resolved, _ := cmd.Flags().GetBool("resolved")

			return runConfigValidate(configFile, quiet, resolved)
		},
	}
	validateCmd.Flags().StringP("file", "f", "", "Configuration file to validate")
	validateCmd.Flags().BoolP("quiet", "q", false, "Only output errors")
	validateCmd.Flags().Bool("resolved", false, "Output the resolved configuration file")

	cmd.AddCommand(validateCmd)
	return cmd
}

// runConfigValidate implements the config validate command
func runConfigValidate(configFile string, quiet bool, resolved bool) error {
	configFile, err := config.FindConfigFile(configFile)
	if err != nil {
		return err
	}

	if !quiet {
		log.Progress("Validating configuration file: %s", configFile)
	}

	// Load and validate the configuration
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return err
	}

	if !quiet {
		log.Success("Configuration file %s is valid", configFile)
	}

	// Output resolved configuration if requested
	if resolved {
		return config.OutputResolvedConfig(cfg)
	}

	return nil
}