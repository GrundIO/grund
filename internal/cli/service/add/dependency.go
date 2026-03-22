package add

import (
	"fmt"

	"github.com/GrundIO/grund/internal/ui"
	"github.com/spf13/cobra"
)

var dependencyCmd = &cobra.Command{
	Use:   "dependency <service>",
	Short: "Add service dependency",
	Long: `Add another service as a dependency to grund.yaml.

This service will be started before your service when running 'grund up'.

Examples:
  grund service add dependency auth-service
  grund service add dependency user-service`,
	Args: cobra.ExactArgs(1),
	RunE: runAddDependency,
}

func runAddDependency(cmd *cobra.Command, args []string) error {
	serviceName := args[0]

	config, configPath, err := loadConfig()
	if err != nil {
		return err
	}

	requires := getRequires(config)

	services, ok := requires["services"].([]any)
	if !ok {
		services = []any{}
	}

	// Check if already exists
	for _, s := range services {
		if str, ok := s.(string); ok && str == serviceName {
			return fmt.Errorf("service %s already in dependencies", serviceName)
		}
	}

	services = append(services, serviceName)
	requires["services"] = services

	// Add env_ref for the service URL
	envKey := toEnvKey(serviceName) + "_URL"
	envVal := fmt.Sprintf("http://${%s.host}:${%s.port}", serviceName, serviceName)
	addEnvRef(config, envKey, envVal)

	if err := writeConfig(config, configPath); err != nil {
		return err
	}

	ui.Successf("Added service dependency: %s", serviceName)
	return nil
}
