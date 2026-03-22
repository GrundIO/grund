package add

import (
	"fmt"

	"github.com/GrundIO/grund/internal/ui"
	"github.com/spf13/cobra"
)

var queueDLQ bool

var queueCmd = &cobra.Command{
	Use:   "queue <name>",
	Short: "Add SQS queue",
	Long: `Add SQS queue requirement to grund.yaml.

Examples:
  grund service add queue orders
  grund service add queue notifications --dlq`,
	Args: cobra.ExactArgs(1),
	RunE: runAddQueue,
}

func init() {
	queueCmd.Flags().BoolVar(&queueDLQ, "dlq", true, "Create dead-letter queue")
}

func runAddQueue(cmd *cobra.Command, args []string) error {
	queueName := args[0]

	config, configPath, err := loadConfig()
	if err != nil {
		return err
	}

	requires := getRequires(config)
	infra := getInfrastructure(requires)

	sqs, ok := infra["sqs"].(map[string]any)
	if !ok {
		sqs = map[string]any{"queues": []any{}}
		infra["sqs"] = sqs
	}

	queues, ok := sqs["queues"].([]any)
	if !ok {
		queues = []any{}
	}

	// Check if queue already exists
	for _, q := range queues {
		if qMap, ok := q.(map[string]any); ok {
			if qMap["name"] == queueName {
				return fmt.Errorf("queue %s already configured", queueName)
			}
		}
	}

	newQueue := map[string]any{"name": queueName}
	if queueDLQ {
		newQueue["dlq"] = true
	}
	queues = append(queues, newQueue)
	sqs["queues"] = queues

	if err := writeConfig(config, configPath); err != nil {
		return err
	}

	ui.Successf("Added SQS queue: %s", queueName)
	return nil
}
