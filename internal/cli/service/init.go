package service

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vivekkundariya/grund/internal/cli/prompts"
	"github.com/vivekkundariya/grund/internal/ui"
	"gopkg.in/yaml.v3"
)

// initConfig holds the collected configuration during init
type initConfig struct {
	Name       string
	Type       string
	Port       int
	Dockerfile string
	HealthPath string
	Postgres   *postgresInit
	MongoDB    *mongoInit
	Redis      bool
	SQSQueues  []string
	SNSTopics  []string
	S3Buckets  []string
	Services   []string
}

type postgresInit struct {
	Database   string
	Migrations string
}

type mongoInit struct {
	Database string
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize grund.yaml in current directory",
	Long: `Create a grund.yaml file in the current directory with an interactive setup.

This command walks you through configuring:
  - Service name, type, and port
  - Build configuration (Dockerfile)
  - Health check endpoint
  - Infrastructure requirements (Postgres, MongoDB, Redis, SQS, SNS, S3)
  - Service dependencies

Example:
  cd my-service
  grund service init`,
	RunE: runInit,
}

func init() {
	Cmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	configPath := filepath.Join(wd, "grund.yaml")
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("grund.yaml already exists in %s", wd)
	}

	ui.Header("Grund Service Configuration")
	fmt.Println()

	cfg := &initConfig{}

	// Service basics
	defaultName := filepath.Base(wd)
	cfg.Name, err = prompts.Text("Service name", defaultName)
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}

	cfg.Type, err = prompts.Select("Service type", []string{"go", "python", "node"}, "go")
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}

	cfg.Port, err = prompts.Int("Port", 8080)
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}

	// Build config
	cfg.Dockerfile, err = prompts.Text("Dockerfile path", "Dockerfile")
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}

	cfg.HealthPath, err = prompts.Text("Health check endpoint", "/health")
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}

	// Infrastructure
	fmt.Println()
	ui.Step("Infrastructure Requirements")

	needPostgres, err := prompts.Confirm("Need PostgreSQL?", false)
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	if needPostgres {
		dbName, err := prompts.Text("Database name", cfg.Name+"_db")
		if err != nil {
			return fmt.Errorf("prompt failed: %w", err)
		}
		migrations, err := prompts.Text("Migrations path (leave empty if none)", "")
		if err != nil {
			return fmt.Errorf("prompt failed: %w", err)
		}
		cfg.Postgres = &postgresInit{Database: dbName, Migrations: migrations}
	}

	needMongo, err := prompts.Confirm("Need MongoDB?", false)
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	if needMongo {
		dbName, err := prompts.Text("Database name", cfg.Name+"_db")
		if err != nil {
			return fmt.Errorf("prompt failed: %w", err)
		}
		cfg.MongoDB = &mongoInit{Database: dbName}
	}

	cfg.Redis, err = prompts.Confirm("Need Redis?", false)
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}

	needSQS, err := prompts.Confirm("Need SQS queues?", false)
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	if needSQS {
		queues, err := prompts.Text("Queue names (comma-separated)", "")
		if err != nil {
			return fmt.Errorf("prompt failed: %w", err)
		}
		if queues != "" {
			cfg.SQSQueues = splitAndTrim(queues)
		}
	}

	needSNS, err := prompts.Confirm("Need SNS topics?", false)
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	if needSNS {
		topics, err := prompts.Text("Topic names (comma-separated)", "")
		if err != nil {
			return fmt.Errorf("prompt failed: %w", err)
		}
		if topics != "" {
			cfg.SNSTopics = splitAndTrim(topics)
		}
	}

	needS3, err := prompts.Confirm("Need S3 buckets?", false)
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	if needS3 {
		buckets, err := prompts.Text("Bucket names (comma-separated)", "")
		if err != nil {
			return fmt.Errorf("prompt failed: %w", err)
		}
		if buckets != "" {
			cfg.S3Buckets = splitAndTrim(buckets)
		}
	}

	// Service dependencies
	fmt.Println()
	ui.Step("Service Dependencies")
	deps, err := prompts.Text("Dependent services (comma-separated, or empty)", "")
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	if deps != "" {
		cfg.Services = splitAndTrim(deps)
	}

	// Generate YAML
	yamlContent := generateGrundYAML(cfg)

	// Preview
	fmt.Println()
	ui.Step("Generated grund.yaml")
	fmt.Println()
	fmt.Println(yamlContent)

	writeConfig, err := prompts.Confirm("Write this configuration?", true)
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}

	if !writeConfig {
		ui.Infof("Aborted.")
		return nil
	}

	// Write file
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		return fmt.Errorf("failed to write grund.yaml: %w", err)
	}

	fmt.Println()
	ui.Successf("Created: %s", configPath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Add this service to your services.yaml registry")
	fmt.Println("  2. Run: grund up " + cfg.Name)

	return nil
}

// splitAndTrim splits a comma-separated string and trims whitespace
// It also filters out invalid resource names like "n", "N", "no", "yes", etc.
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if isInvalidResourceName(p) {
			ui.Warnf("Skipping invalid resource name: %q (looks like a yes/no response)", p)
			continue
		}
		result = append(result, p)
	}
	return result
}

// isInvalidResourceName checks if a string looks like a yes/no response
// rather than a valid resource name
func isInvalidResourceName(name string) bool {
	lower := strings.ToLower(name)
	invalidNames := []string{"n", "no", "y", "yes"}
	return slices.Contains(invalidNames, lower)
}

// generateGrundYAML creates the YAML content from config
func generateGrundYAML(cfg *initConfig) string {
	// Build the config structure for YAML marshaling
	config := map[string]any{
		"version": "1",
		"service": map[string]any{
			"name": cfg.Name,
			"type": cfg.Type,
			"port": cfg.Port,
			"build": map[string]any{
				"dockerfile": cfg.Dockerfile,
				"context":    ".",
			},
			"health": map[string]any{
				"endpoint": cfg.HealthPath,
				"interval": "5s",
				"timeout":  "3s",
				"retries":  10,
			},
		},
	}

	// Build requires section
	requires := map[string]any{
		"services": cfg.Services,
	}

	infrastructure := map[string]any{}

	if cfg.Postgres != nil {
		pg := map[string]any{
			"database": cfg.Postgres.Database,
		}
		if cfg.Postgres.Migrations != "" {
			pg["migrations"] = cfg.Postgres.Migrations
		}
		infrastructure["postgres"] = pg
	}

	if cfg.MongoDB != nil {
		infrastructure["mongodb"] = map[string]any{
			"database": cfg.MongoDB.Database,
		}
	}

	if cfg.Redis {
		infrastructure["redis"] = true
	}

	if len(cfg.SQSQueues) > 0 {
		queues := make([]map[string]any, len(cfg.SQSQueues))
		for i, q := range cfg.SQSQueues {
			queues[i] = map[string]any{
				"name": q,
				"dlq":  true,
			}
		}
		infrastructure["sqs"] = map[string]any{
			"queues": queues,
		}
	}

	if len(cfg.SNSTopics) > 0 {
		topics := make([]map[string]any, len(cfg.SNSTopics))
		for i, t := range cfg.SNSTopics {
			topics[i] = map[string]any{
				"name": t,
			}
		}
		infrastructure["sns"] = map[string]any{
			"topics": topics,
		}
	}

	if len(cfg.S3Buckets) > 0 {
		buckets := make([]map[string]any, len(cfg.S3Buckets))
		for i, b := range cfg.S3Buckets {
			buckets[i] = map[string]any{
				"name": b,
			}
		}
		infrastructure["s3"] = map[string]any{
			"buckets": buckets,
		}
	}

	if len(infrastructure) > 0 {
		requires["infrastructure"] = infrastructure
	}

	config["requires"] = requires

	// Environment variables (all values support ${placeholder} syntax)
	env := map[string]string{
		"APP_ENV":   "development",
		"LOG_LEVEL": "debug",
	}

	if cfg.Postgres != nil {
		env["DATABASE_URL"] = "postgres://postgres:postgres@${postgres.host}:${postgres.port}/${self.postgres.database}"
	}

	if cfg.MongoDB != nil {
		env["MONGO_URL"] = "mongodb://${mongodb.host}:${mongodb.port}/${self.mongodb.database}"
	}

	if cfg.Redis {
		env["REDIS_URL"] = "redis://${redis.host}:${redis.port}"
	}

	// Add service dependency URLs
	for _, svc := range cfg.Services {
		envKey := strings.ToUpper(strings.ReplaceAll(svc, "-", "_")) + "_URL"
		env[envKey] = fmt.Sprintf("http://${%s.host}:${%s.port}", svc, svc)
	}

	config["env"] = env

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Sprintf("# Error generating YAML: %v", err)
	}

	return string(data)
}
