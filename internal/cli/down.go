package cli

import (
	"github.com/GrundIO/grund/internal/application/commands"
	"github.com/GrundIO/grund/internal/application/wiring"
	"github.com/GrundIO/grund/internal/cli/shared"
	"github.com/GrundIO/grund/internal/config"
	"github.com/GrundIO/grund/internal/infrastructure/docker"
	"github.com/GrundIO/grund/internal/ui"
	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop all running services",
	Long: `Stop all services and infrastructure that were started by grund.

If run from a project directory (with services.yaml), stops that project.
If run without a valid project context, stops all projects in ~/.grund/tmp/.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Try to initialize with current directory context
		resolver, err := config.NewConfigResolver(configFile)
		if err == nil {
			servicesPath, orchestrationRoot, err := resolver.ResolveServicesFile()
			if err == nil {
				// Valid project context - stop this project
				ui.Debug("Using config: %s", servicesPath)
				shared.Container, err = wiring.NewContainerWithConfig(orchestrationRoot, servicesPath, resolver)
				if err == nil {
					downCmd := commands.DownCommand{}
					return shared.Container.DownCommandHandler.Handle(cmd.Context(), downCmd)
				}
			}
		}

		// No valid project context - stop all projects
		ui.Infof("No project context found, stopping all projects...")
		return docker.StopAllProjects(cmd.Context())
	},
}
