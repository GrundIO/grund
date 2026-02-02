package add

import (
	"fmt"

	"github.com/Saturn-Fintech/grund/internal/ui"
	"github.com/spf13/cobra"
)

var (
	tunnelHost     string
	tunnelPort     string
	tunnelProvider string
)

var tunnelCmd = &cobra.Command{
	Use:   "tunnel <name>",
	Short: "Add tunnel for external access",
	Long: `Add tunnel configuration for exposing local services to internet.

Tunnels are useful for:
  - S3 presigned URLs that need public access
  - Webhook callbacks from external services
  - Sharing local development with teammates

Examples:
  grund service add tunnel localstack --port 4566
  grund service add tunnel api --host localhost --port 8080
  grund service add tunnel webhook --port 3000 --provider ngrok`,
	Args: cobra.ExactArgs(1),
	RunE: runAddTunnel,
}

func init() {
	tunnelCmd.Flags().StringVar(&tunnelHost, "host", "localhost", "Local host to tunnel")
	tunnelCmd.Flags().StringVar(&tunnelPort, "port", "", "Local port to tunnel (required)")
	tunnelCmd.Flags().StringVar(&tunnelProvider, "provider", "cloudflared", "Tunnel provider (cloudflared, ngrok)")
	tunnelCmd.MarkFlagRequired("port")
}

func runAddTunnel(cmd *cobra.Command, args []string) error {
	tunnelName := args[0]

	config, configPath, err := loadConfig()
	if err != nil {
		return err
	}

	requires := getRequires(config)
	infra := getInfrastructure(requires)

	tunnel, ok := infra["tunnel"].(map[string]any)
	if !ok {
		tunnel = map[string]any{
			"provider": tunnelProvider,
			"targets":  []any{},
		}
		infra["tunnel"] = tunnel
	}

	targets, ok := tunnel["targets"].([]any)
	if !ok {
		targets = []any{}
	}

	// Check if tunnel target already exists
	for _, t := range targets {
		if tMap, ok := t.(map[string]any); ok {
			if tMap["name"] == tunnelName {
				return fmt.Errorf("tunnel target %s already configured", tunnelName)
			}
		}
	}

	newTarget := map[string]any{
		"name": tunnelName,
		"host": tunnelHost,
		"port": tunnelPort,
	}
	targets = append(targets, newTarget)
	tunnel["targets"] = targets

	// Update provider if different from default
	if tunnelProvider != "cloudflared" {
		tunnel["provider"] = tunnelProvider
	}

	// Add env_ref for the tunnel URL
	envKey := fmt.Sprintf("%s_PUBLIC_URL", toEnvKey(tunnelName))
	addEnvRef(config, envKey, fmt.Sprintf("${tunnel.%s.url}", tunnelName))

	if err := writeConfig(config, configPath); err != nil {
		return err
	}

	ui.Successf("Added tunnel: %s (%s:%s via %s)", tunnelName, tunnelHost, tunnelPort, tunnelProvider)
	return nil
}
