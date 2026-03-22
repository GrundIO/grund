package add

import (
	"fmt"

	"github.com/GrundIO/grund/internal/ui"
	"github.com/spf13/cobra"
)

var bucketSeed string

var bucketCmd = &cobra.Command{
	Use:   "bucket <name>",
	Short: "Add S3 bucket",
	Long: `Add S3 bucket requirement to grund.yaml.

Examples:
  grund service add bucket uploads
  grund service add bucket documents --seed ./fixtures/`,
	Args: cobra.ExactArgs(1),
	RunE: runAddBucket,
}

func init() {
	bucketCmd.Flags().StringVar(&bucketSeed, "seed", "", "Path to seed data directory")
}

func runAddBucket(cmd *cobra.Command, args []string) error {
	bucketName := args[0]

	config, configPath, err := loadConfig()
	if err != nil {
		return err
	}

	requires := getRequires(config)
	infra := getInfrastructure(requires)

	s3, ok := infra["s3"].(map[string]any)
	if !ok {
		s3 = map[string]any{"buckets": []any{}}
		infra["s3"] = s3
	}

	buckets, ok := s3["buckets"].([]any)
	if !ok {
		buckets = []any{}
	}

	// Check if bucket already exists
	for _, b := range buckets {
		if bMap, ok := b.(map[string]any); ok {
			if bMap["name"] == bucketName {
				return fmt.Errorf("bucket %s already configured", bucketName)
			}
		}
	}

	newBucket := map[string]any{"name": bucketName}
	if bucketSeed != "" {
		newBucket["seed"] = bucketSeed
	}
	buckets = append(buckets, newBucket)
	s3["buckets"] = buckets

	if err := writeConfig(config, configPath); err != nil {
		return err
	}

	ui.Successf("Added S3 bucket: %s", bucketName)
	return nil
}
