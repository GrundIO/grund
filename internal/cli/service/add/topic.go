package add

import (
	"fmt"

	"github.com/Saturn-Fintech/grund/internal/ui"
	"github.com/spf13/cobra"
)

var topicCmd = &cobra.Command{
	Use:   "topic <name>",
	Short: "Add SNS topic",
	Long: `Add SNS topic requirement to grund.yaml.

Examples:
  grund service add topic notifications
  grund service add topic events`,
	Args: cobra.ExactArgs(1),
	RunE: runAddTopic,
}

func runAddTopic(cmd *cobra.Command, args []string) error {
	topicName := args[0]

	config, configPath, err := loadConfig()
	if err != nil {
		return err
	}

	requires := getRequires(config)
	infra := getInfrastructure(requires)

	sns, ok := infra["sns"].(map[string]any)
	if !ok {
		sns = map[string]any{"topics": []any{}}
		infra["sns"] = sns
	}

	topics, ok := sns["topics"].([]any)
	if !ok {
		topics = []any{}
	}

	// Check if topic already exists
	for _, t := range topics {
		if tMap, ok := t.(map[string]any); ok {
			if tMap["name"] == topicName {
				return fmt.Errorf("topic %s already configured", topicName)
			}
		}
	}

	topics = append(topics, map[string]any{"name": topicName})
	sns["topics"] = topics

	if err := writeConfig(config, configPath); err != nil {
		return err
	}

	ui.Successf("Added SNS topic: %s", topicName)
	return nil
}
