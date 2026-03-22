package add

import (
	"fmt"

	"github.com/GrundIO/grund/internal/ui"
	"github.com/spf13/cobra"
)

var redisCmd = &cobra.Command{
	Use:   "redis",
	Short: "Add Redis cache",
	Long: `Add Redis cache requirement to grund.yaml.

Example:
  grund service add redis`,
	Args: cobra.NoArgs,
	RunE: runAddRedis,
}

func runAddRedis(cmd *cobra.Command, args []string) error {
	config, configPath, err := loadConfig()
	if err != nil {
		return err
	}

	requires := getRequires(config)
	infra := getInfrastructure(requires)

	if _, exists := infra["redis"]; exists {
		return fmt.Errorf("redis already configured in grund.yaml")
	}

	infra["redis"] = true

	addEnvRef(config, "REDIS_URL", "redis://${redis.host}:${redis.port}")

	if err := writeConfig(config, configPath); err != nil {
		return err
	}

	ui.Successf("Added Redis")
	return nil
}
