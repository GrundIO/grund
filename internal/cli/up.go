package cli

import (
	"fmt"

	"github.com/GrundIO/grund/internal/application/commands"
	"github.com/GrundIO/grund/internal/cli/shared"
	"github.com/GrundIO/grund/internal/infrastructure/generator"
	"github.com/GrundIO/grund/internal/ui"
	"github.com/spf13/cobra"
)

var (
	upNoDeps    bool
	upInfraOnly bool
	upBuild     bool
	upLocal     bool
)

var upCmd = &cobra.Command{
	Use:   "up [services...]",
	Short: "Start services and dependencies",
	Long: `Start one or more services along with all their dependencies.
Infrastructure will be started first, followed by services in dependency order.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if shared.Container == nil {
			return fmt.Errorf("container not initialized")
		}

		// Validate secrets before starting
		if err := validateSecrets(cmd, args); err != nil {
			return err
		}

		upCmd := commands.UpCommand{
			ServiceNames: args,
			NoDeps:       upNoDeps,
			InfraOnly:    upInfraOnly,
			Build:        upBuild,
			Local:        upLocal,
		}

		return shared.Container.UpCommandHandler.Handle(cmd.Context(), upCmd)
	},
}

func init() {
	upCmd.Flags().BoolVar(&upNoDeps, "no-deps", false, "Only start specified services, no dependencies")
	upCmd.Flags().BoolVar(&upInfraOnly, "infra-only", false, "Only start infrastructure, no services")
	upCmd.Flags().BoolVar(&upBuild, "build", false, "Force rebuild containers")
	upCmd.Flags().BoolVar(&upLocal, "local", false, "Run service locally (not in container)")
}

// validateSecrets checks that all required secrets are available
func validateSecrets(cmd *cobra.Command, serviceNames []string) error {
	// Get services and their dependencies
	services, err := getServicesWithDependencies(cmd, serviceNames)
	if err != nil {
		return err
	}

	// Check if any services have secrets defined
	hasSecrets := false
	for _, svc := range services {
		if len(svc.Environment.Secrets) > 0 {
			hasSecrets = true
			break
		}
	}

	if !hasSecrets {
		return nil // No secrets to validate
	}

	ui.Step("Checking secrets...")

	loader := generator.NewSecretsLoader()
	missing, err := loader.GetMissingRequired(services)
	if err != nil {
		return fmt.Errorf("failed to validate secrets: %w", err)
	}

	if len(missing) > 0 {
		for _, s := range missing {
			ui.Errorf("  ✗ %s is required but not found", s.Name)
		}
		fmt.Println()
		ui.Errorf("Missing required secrets: %d", len(missing))
		fmt.Println()
		fmt.Println("Run 'grund secrets list <service>' to see all required secrets.")
		fmt.Println("Run 'grund secrets init <service>' to generate a template.")
		return fmt.Errorf("missing required secrets")
	}

	// Count loaded secrets
	if err := loader.Load(); err == nil {
		count := 0
		for _, svc := range services {
			for name := range svc.Environment.Secrets {
				if _, found := loader.GetSecretValue(name); found {
					count++
				}
			}
		}
		if count > 0 {
			ui.Successf("  ✓ %d secrets loaded", count)
		}
	}

	return nil
}
