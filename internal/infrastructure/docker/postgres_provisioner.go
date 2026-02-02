package docker

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/vivekkundariya/grund/internal/application/ports"
	"github.com/vivekkundariya/grund/internal/domain/infrastructure"
	"github.com/vivekkundariya/grund/internal/ui"
)

// PostgresProvisioner implements infrastructure provisioning for PostgreSQL
type PostgresProvisioner struct{}

// NewPostgresProvisioner creates a new Postgres provisioner
func NewPostgresProvisioner() ports.InfrastructureProvisioner {
	return &PostgresProvisioner{}
}

// ProvisionPostgres provisions PostgreSQL databases
func (p *PostgresProvisioner) ProvisionPostgres(ctx context.Context, config *infrastructure.PostgresConfig) error {
	if config == nil {
		return nil
	}

	// Primary database is created by Docker via POSTGRES_DB env var
	// We only need to create additional databases
	for _, db := range config.Databases {
		if err := p.createDatabase(ctx, db.Name); err != nil {
			return fmt.Errorf("failed to create database %s: %w", db.Name, err)
		}
		ui.SubStep("postgres: %s ✓", db.Name)
	}

	// TODO: Implement migration and seeding logic
	// if config.Migrations != "" {
	//     if err := p.runMigrations(ctx, config.Database, config.Migrations); err != nil {
	//         return fmt.Errorf("failed to run migrations: %w", err)
	//     }
	// }

	return nil
}

// createDatabase creates a PostgreSQL database if it doesn't exist
func (p *PostgresProvisioner) createDatabase(ctx context.Context, dbName string) error {
	// Check if database already exists
	checkCmd := exec.CommandContext(ctx, "docker", "exec", "grund-postgres",
		"psql", "-U", "postgres", "-tAc",
		fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname='%s'", dbName))

	output, err := checkCmd.Output()
	if err == nil && strings.TrimSpace(string(output)) == "1" {
		// Database already exists
		return nil
	}

	// Create the database
	createCmd := exec.CommandContext(ctx, "docker", "exec", "grund-postgres",
		"psql", "-U", "postgres", "-c",
		fmt.Sprintf("CREATE DATABASE %s", dbName))

	if output, err := createCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("create database failed: %s: %w", string(output), err)
	}

	return nil
}

// ProvisionMongoDB not applicable
func (p *PostgresProvisioner) ProvisionMongoDB(ctx context.Context, config *infrastructure.MongoDBConfig) error {
	return fmt.Errorf("mongodb provisioning not supported by postgres provisioner")
}

// ProvisionRedis not applicable
func (p *PostgresProvisioner) ProvisionRedis(ctx context.Context, config *infrastructure.RedisConfig) error {
	return fmt.Errorf("redis provisioning not supported by postgres provisioner")
}

// ProvisionLocalStack not applicable
func (p *PostgresProvisioner) ProvisionLocalStack(ctx context.Context, req infrastructure.InfrastructureRequirements) error {
	return fmt.Errorf("localstack provisioning not supported by postgres provisioner")
}
