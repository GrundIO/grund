package add

import (
	"fmt"

	"github.com/Saturn-Fintech/grund/internal/ui"
	"github.com/spf13/cobra"
)

var mongodbSeed string

var mongodbCmd = &cobra.Command{
	Use:   "mongodb <database>",
	Short: "Add MongoDB database",
	Long: `Add MongoDB database requirement to grund.yaml.

Examples:
  grund service add mongodb mydb
  grund service add mongodb users_db --seed ./fixtures/seed.js`,
	Args: cobra.ExactArgs(1),
	RunE: runAddMongoDB,
}

func init() {
	mongodbCmd.Flags().StringVar(&mongodbSeed, "seed", "", "Path to seed script")
}

func runAddMongoDB(cmd *cobra.Command, args []string) error {
	database := args[0]

	config, configPath, err := loadConfig()
	if err != nil {
		return err
	}

	requires := getRequires(config)
	infra := getInfrastructure(requires)

	if _, exists := infra["mongodb"]; exists {
		return fmt.Errorf("mongodb already configured in grund.yaml")
	}

	mongo := map[string]any{"database": database}
	if mongodbSeed != "" {
		mongo["seed"] = mongodbSeed
	}
	infra["mongodb"] = mongo

	addEnvRef(config, "MONGO_URL", "mongodb://${mongodb.host}:${mongodb.port}/${self.mongodb.database}")

	if err := writeConfig(config, configPath); err != nil {
		return err
	}

	ui.Successf("Added MongoDB (database: %s)", database)
	return nil
}
