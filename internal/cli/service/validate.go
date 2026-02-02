package service

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/Saturn-Fintech/grund/internal/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate grund.yaml configuration",
	Long: `Validates the grund.yaml file in the current directory.

Checks:
  - YAML syntax
  - Required fields (service.name, service.type, service.port)
  - Infrastructure configuration structure

Example:
  cd my-service
  grund service validate`,
	RunE: runValidate,
}

func init() {
	Cmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	configPath := filepath.Join(".", "grund.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("cannot read grund.yaml: %w", err)
	}

	var config map[string]any
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("invalid YAML syntax: %w", err)
	}

	// Validate required fields
	errors := validateConfig(config)

	if len(errors) > 0 {
		ui.Error("Validation failed:")
		for _, e := range errors {
			fmt.Printf("  - %s\n", e)
		}
		return fmt.Errorf("%d validation error(s)", len(errors))
	}

	ui.Success("grund.yaml is valid")
	return nil
}

func validateConfig(config map[string]any) []string {
	var errors []string

	// Check service section
	svc, ok := config["service"].(map[string]any)
	if !ok {
		errors = append(errors, "missing required section: service")
		return errors
	}

	// Required service fields
	requiredFields := []string{"name", "type", "port"}
	for _, field := range requiredFields {
		if _, ok := svc[field]; !ok {
			errors = append(errors, fmt.Sprintf("missing required field: service.%s", field))
		}
	}

	// Validate service type if present
	if svcType, ok := svc["type"].(string); ok {
		validTypes := []string{"go", "python", "node", "ruby", "java"}
		if !slices.Contains(validTypes, svcType) {
			errors = append(errors, fmt.Sprintf("invalid service.type: %s (valid: go, python, node, ruby, java)", svcType))
		}
	}

	// Validate port if present
	if port, ok := svc["port"]; ok {
		switch p := port.(type) {
		case int:
			if p < 1 || p > 65535 {
				errors = append(errors, fmt.Sprintf("invalid service.port: %d (must be 1-65535)", p))
			}
		case float64:
			if p < 1 || p > 65535 {
				errors = append(errors, fmt.Sprintf("invalid service.port: %.0f (must be 1-65535)", p))
			}
		default:
			errors = append(errors, "service.port must be a number")
		}
	}

	return errors
}
