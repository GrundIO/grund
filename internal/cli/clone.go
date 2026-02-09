package cli

import (
	"fmt"

	"github.com/Saturn-Fintech/grund/internal/application/commands"
	"github.com/Saturn-Fintech/grund/internal/cli/shared"
	"github.com/spf13/cobra"
)

var cloneSync bool

var cloneCmd = &cobra.Command{
	Use:   "clone [services...]",
	Short: "Clone service repositories",
	Long: `Clone all (or specific) service repositories defined in services.yaml.

By default, services that are already cloned are skipped.
Use --sync to pull the latest changes for existing repositories.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if shared.Container == nil {
			return fmt.Errorf("container not initialized")
		}

		cloneCmd := commands.CloneCommand{
			ServiceNames: args,
			Sync:         cloneSync,
		}

		return shared.Container.CloneCommandHandler.Handle(cmd.Context(), cloneCmd)
	},
}

func init() {
	cloneCmd.Flags().BoolVar(&cloneSync, "sync", false, "Pull latest changes for already cloned repositories")
}
