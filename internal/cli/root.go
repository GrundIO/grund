package cli

import (
	"fmt"

	"github.com/Saturn-Fintech/grund/internal/application/wiring"
	"github.com/Saturn-Fintech/grund/internal/cli/configcmd"
	"github.com/Saturn-Fintech/grund/internal/cli/service"
	"github.com/Saturn-Fintech/grund/internal/cli/shared"
	"github.com/Saturn-Fintech/grund/internal/config"
	"github.com/Saturn-Fintech/grund/internal/ui"
	"github.com/spf13/cobra"
)

var (
	// CLI flags
	configFile string
	verbose    bool
)

var rootCmd = &cobra.Command{
	Use:   "grund",
	Short: "Grund - Local development orchestration tool",
	Long: `Grund is a CLI tool that enables developers to selectively spin up
services and their dependencies with a single command.

Quick Start:
  grund up user-service       Start a service with dependencies
  grund status                Check running services
  grund down                  Stop everything

Service Management:
  grund service init          Initialize new service
  grund service add queue X   Add SQS queue
  grund service validate      Validate configuration

Configuration:
  grund config init           Set up global config
  grund config show           View configuration

Documentation: https://github.com/Saturn-Fintech/grund`,
	Version: "0.5.0",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Set verbose mode on logger
		ui.SetVerbose(verbose)

		// Skip initialization for help and version commands
		if cmd.Name() == "help" || cmd.Name() == "version" {
			return nil
		}

		// Also skip for completion commands
		if cmd.Name() == "completion" {
			return nil
		}

		// Skip for commands that don't need existing services.yaml
		// - service init: creates new grund.yaml
		// - config init: creates global config
		// - down: can work without project context
		// - validate: only needs local grund.yaml
		skipInit := []string{"init", "down", "validate"}
		for _, skip := range skipInit {
			if cmd.Name() == skip {
				return nil
			}
		}

		// Initialize config resolver
		var err error
		shared.ConfigResolver, err = config.NewConfigResolver(configFile)
		if err != nil {
			return fmt.Errorf("failed to initialize config: %w", err)
		}

		// Resolve services file and orchestration root
		servicesPath, orchestrationRoot, err := shared.ConfigResolver.ResolveServicesFile()
		if err != nil {
			return err
		}

		ui.Debug("Using config: %s", servicesPath)
		ui.Debug("Orchestration root: %s", orchestrationRoot)

		// Initialize dependency injection container
		shared.Container, err = wiring.NewContainerWithConfig(orchestrationRoot, servicesPath, shared.ConfigResolver)
		if err != nil {
			return fmt.Errorf("failed to initialize: %w", err)
		}

		return nil
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "",
		"Path to services registry file (default: auto-detect services.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false,
		"Enable verbose output")

	// Initialization
	rootCmd.AddCommand(initCmd)

	// Daily operations (root level - most frequently used)
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(resetCmd)
	rootCmd.AddCommand(syncCmd)

	// Service management
	rootCmd.AddCommand(service.Cmd)

	// Configuration management
	rootCmd.AddCommand(configcmd.Cmd)

	// Secrets management
	rootCmd.AddCommand(secretsCmd)
}

// GetConfigResolver returns the current config resolver (for use by subcommands)
func GetConfigResolver() *config.ConfigResolver {
	return shared.ConfigResolver
}

// GetContainer returns the current DI container (for use by subcommands)
func GetContainer() *wiring.Container {
	return shared.Container
}
