package add

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// loadConfig loads the grund.yaml from current directory
func loadConfig() (map[string]any, string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get working directory: %w", err)
	}

	configPath := filepath.Join(wd, "grund.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, "", fmt.Errorf("grund.yaml not found. Run 'grund service init' first")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read grund.yaml: %w", err)
	}

	var config map[string]any
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, "", fmt.Errorf("failed to parse grund.yaml: %w", err)
	}

	return config, configPath, nil
}

// getServiceName extracts the service name from config
func getServiceName(config map[string]any) string {
	if svc, ok := config["service"].(map[string]any); ok {
		if name, ok := svc["name"].(string); ok {
			return name
		}
	}
	return ""
}

// getRequires gets or creates the requires section
func getRequires(config map[string]any) map[string]any {
	requires, ok := config["requires"].(map[string]any)
	if !ok {
		requires = make(map[string]any)
		config["requires"] = requires
	}
	return requires
}

// getInfrastructure gets or creates the infrastructure section
func getInfrastructure(requires map[string]any) map[string]any {
	infrastructure, ok := requires["infrastructure"].(map[string]any)
	if !ok {
		infrastructure = make(map[string]any)
		requires["infrastructure"] = infrastructure
	}
	return infrastructure
}

// addEnvRef adds an environment variable with placeholder syntax if it doesn't exist
func addEnvRef(config map[string]any, key, value string) {
	env, ok := config["env"].(map[string]any)
	if !ok {
		env = make(map[string]any)
		config["env"] = env
	}
	if _, exists := env[key]; !exists {
		env[key] = value
	}
}

// writeConfig writes the config back to grund.yaml
func writeConfig(config map[string]any, configPath string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write grund.yaml: %w", err)
	}

	return nil
}

// toEnvKey converts a service name to an environment variable key
func toEnvKey(name string) string {
	return strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
}
