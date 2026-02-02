package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/Saturn-Fintech/grund/internal/infrastructure/docker"
	"github.com/Saturn-Fintech/grund/internal/ui"
	"github.com/spf13/cobra"
)

var (
	resetVolumes bool
	resetImages  bool
)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Stop services and clean up resources",
	Long: `Stop all services and optionally remove volumes and images.

Examples:
  grund reset              # Stop all services
  grund reset --volumes    # Stop and remove volumes (database data)
  grund reset --volumes --images  # Stop, remove volumes and images (full cleanup)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Discover compose files
		fileSet, err := docker.DiscoverComposeFiles()
		if err != nil {
			return fmt.Errorf("failed to discover compose files: %w", err)
		}

		allPaths := fileSet.AllPaths()
		if len(allPaths) == 0 {
			ui.Infof("No services to reset (no compose files found)")
			return nil
		}

		// Build docker compose down command with project name and all compose files
		dockerArgs := []string{"compose", "-p", docker.ProjectName}
		for _, path := range allPaths {
			dockerArgs = append(dockerArgs, "-f", path)
		}
		dockerArgs = append(dockerArgs, "down")

		if resetVolumes {
			dockerArgs = append(dockerArgs, "-v")
			ui.Step("Stopping services and removing volumes...")
		} else {
			ui.Step("Stopping services...")
		}

		if resetImages {
			dockerArgs = append(dockerArgs, "--rmi", "local")
			ui.SubStep("Also removing locally built images")
		}

		// Execute docker compose down
		dockerCmd := exec.CommandContext(cmd.Context(), "docker", dockerArgs...)
		dockerCmd.Stdout = os.Stdout
		dockerCmd.Stderr = os.Stderr

		if err := dockerCmd.Run(); err != nil {
			return fmt.Errorf("failed to reset: %w", err)
		}

		ui.Successf("Reset complete")
		return nil
	},
}

func init() {
	resetCmd.Flags().BoolVar(&resetVolumes, "volumes", false, "Remove named volumes (database data will be lost)")
	resetCmd.Flags().BoolVar(&resetImages, "images", false, "Remove locally built images")
}
