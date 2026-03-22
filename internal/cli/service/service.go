package service

import (
	"github.com/GrundIO/grund/internal/cli/service/add"
	"github.com/spf13/cobra"
)

// Cmd is the parent command for service management
var Cmd = &cobra.Command{
	Use:   "service",
	Short: "Manage service configuration",
	Long: `Commands for managing service configuration files (grund.yaml).

Examples:
  grund service init              Initialize a new service
  grund service add postgres mydb Add PostgreSQL database
  grund service add queue orders  Add SQS queue
  grund service validate          Validate configuration
  grund service list              List registered services`,
}

func init() {
	Cmd.AddCommand(add.Cmd)
}
