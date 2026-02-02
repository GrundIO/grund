package cli

import (
	"fmt"

	"github.com/Saturn-Fintech/grund/internal/application/commands"
	"github.com/Saturn-Fintech/grund/internal/cli/shared"
	"github.com/spf13/cobra"
)

var restartBuild bool

var restartCmd = &cobra.Command{
	Use:   "restart [services...]",
	Short: "Restart specific service(s)",
	Long:  `Restart one or more services while keeping infrastructure running.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if shared.Container == nil {
			return fmt.Errorf("container not initialized")
		}

		restartCmd := commands.RestartCommand{
			ServiceNames: args,
			Build:        restartBuild,
		}
		return shared.Container.RestartCommandHandler.Handle(cmd.Context(), restartCmd)
	},
}

func init() {
	restartCmd.Flags().BoolVar(&restartBuild, "build", false, "Rebuild containers before restarting")
}
