package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/Saturn-Fintech/grund/internal/infrastructure/docker"
	"github.com/spf13/cobra"
)

var (
	logsFollow bool
	logsTail   int
)

var logsCmd = &cobra.Command{
	Use:   "logs [services...]",
	Short: "View aggregated or per-service logs",
	Long:  `View logs from all services or specific services. Multiple services can be specified.`,
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Discover compose files
		fileSet, err := docker.DiscoverComposeFiles()
		if err != nil {
			return fmt.Errorf("failed to discover compose files: %w", err)
		}

		allPaths := fileSet.AllPaths()
		if len(allPaths) == 0 {
			return fmt.Errorf("no services are running. Run 'grund up <service>' first")
		}

		// Build docker compose logs command with project name and all compose files
		dockerArgs := []string{"compose", "-p", docker.ProjectName}
		for _, path := range allPaths {
			dockerArgs = append(dockerArgs, "-f", path)
		}
		dockerArgs = append(dockerArgs, "logs")

		if logsFollow {
			dockerArgs = append(dockerArgs, "-f")
		}

		if logsTail > 0 {
			dockerArgs = append(dockerArgs, "--tail", strconv.Itoa(logsTail))
		}

		// Add service names if specified
		if len(args) > 0 {
			dockerArgs = append(dockerArgs, args...)
		}

		// Execute docker compose logs with stdout/stderr connected
		dockerCmd := exec.CommandContext(cmd.Context(), "docker", dockerArgs...)
		dockerCmd.Stdout = os.Stdout
		dockerCmd.Stderr = os.Stderr
		dockerCmd.Stdin = os.Stdin

		return dockerCmd.Run()
	},
}

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
	logsCmd.Flags().IntVar(&logsTail, "tail", 100, "Number of lines to show from the end (default 100)")
}
