package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thi-startup/spitfire/pkg/log"

	// Import drivers to trigger their init() functions
	_ "github.com/thi-startup/spitfire/pkg/driver/firecracker"
	_ "github.com/thi-startup/spitfire/pkg/driver/mock"
)

var (
	version    string
	commitHash string
	buildTime  string
)

func main() {
	root := &cobra.Command{
		Use:              "spitfire",
		Short:            "Manage Firecracker microVMs with a Docker Compose-like experience",
		Version:          fmt.Sprintf("Version:\t%s+%s\nBuildTime:\t%s\n", version, commitHash, buildTime),
		PersistentPreRun: func(cmd *cobra.Command, args []string) { initializeLogging(cmd) },
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	root.SetVersionTemplate(`{{printf "%s" .Version}}`)
	root.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	root.PersistentFlags().Bool("debug", false, "Enable debug output")
	root.PersistentFlags().BoolP("quiet", "q", false, "Suppress non-error output")
	root.PersistentFlags().String("log-level", "", "Set log level (trace|debug|info|warn|error)")
	root.PersistentFlags().String("log-file", "", "Write logs to file instead of stderr")
	root.AddCommand(
		newVMCommands(),
		newVolumeCommands(),
		newConfigCommands(),
		newSetupCommand(),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// initializeLogging sets up the global logger based on command flags
func initializeLogging(cmd *cobra.Command) {
	verbose, _ := cmd.Flags().GetBool("verbose")
	debug, _ := cmd.Flags().GetBool("debug")
	quiet, _ := cmd.Flags().GetBool("quiet")
	logLevel, _ := cmd.Flags().GetString("log-level")
	logFile, _ := cmd.Flags().GetString("log-file")

	config := log.Config{
		Verbose: verbose,
		Debug:   debug,
		Quiet:   quiet,
		LogFile: logFile,
		Format:  "text", // Default to text format for CLI
	}

	// Parse explicit log level if provided
	if logLevel != "" {
		config.Level = log.ParseLogLevel(logLevel)
	}

	log.InitializeGlobalLogger(config)
}
