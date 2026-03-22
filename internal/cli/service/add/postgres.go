package add

import (
	"fmt"

	"github.com/GrundIO/grund/internal/ui"
	"github.com/spf13/cobra"
)

var (
	postgresMigrations string
	postgresSeed       string
)

var postgresCmd = &cobra.Command{
	Use:   "postgres <database>",
	Short: "Add PostgreSQL database",
	Long: `Add PostgreSQL database requirement to grund.yaml.

Examples:
  grund service add postgres mydb
  grund service add postgres users_db --migrations ./db/migrations
  grund service add postgres app_db --seed ./fixtures/seed.sql`,
	Args: cobra.ExactArgs(1),
	RunE: runAddPostgres,
}

func init() {
	postgresCmd.Flags().StringVar(&postgresMigrations, "migrations", "", "Path to migrations directory")
	postgresCmd.Flags().StringVar(&postgresSeed, "seed", "", "Path to seed SQL file")
}

func runAddPostgres(cmd *cobra.Command, args []string) error {
	database := args[0]

	config, configPath, err := loadConfig()
	if err != nil {
		return err
	}

	requires := getRequires(config)
	infra := getInfrastructure(requires)

	if _, exists := infra["postgres"]; exists {
		return fmt.Errorf("postgres already configured in grund.yaml")
	}

	pg := map[string]any{"database": database}
	if postgresMigrations != "" {
		pg["migrations"] = postgresMigrations
	}
	if postgresSeed != "" {
		pg["seed"] = postgresSeed
	}
	infra["postgres"] = pg

	addEnvRef(config, "DATABASE_URL", "postgres://postgres:postgres@${postgres.host}:${postgres.port}/${self.postgres.database}")

	if err := writeConfig(config, configPath); err != nil {
		return err
	}

	ui.Successf("Added PostgreSQL (database: %s)", database)
	return nil
}
