package cli

import (
	"fmt"
	"os"

	"github.com/GrundIO/grund/internal/application/queries"
	"github.com/GrundIO/grund/internal/cli/shared"
	"github.com/GrundIO/grund/internal/ui"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show running services and health",
	Long:  `Display the status of all running services and infrastructure, including health checks.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if shared.Container == nil {
			return fmt.Errorf("container not initialized")
		}

		query := queries.StatusQuery{ServiceName: nil}
		statuses, err := shared.Container.StatusQueryHandler.Handle(cmd.Context(), query)
		if err != nil {
			return err
		}

		if len(statuses) == 0 {
			ui.Infof("No services found. Run 'grund up <service>' to start services.")
			return nil
		}

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.SetStyle(table.StyleRounded)

		t.AppendHeader(table.Row{"Service", "Status", "URL"})

		for _, s := range statuses {
			var statusIcon string
			var statusColor text.Color
			switch s.Status {
			case "running":
				statusIcon = "●"
				statusColor = text.FgGreen
			case "exited":
				statusIcon = "●"
				statusColor = text.FgRed
			default:
				statusIcon = "○"
				statusColor = text.FgYellow
			}

			statusText := fmt.Sprintf("%s %s", statusIcon, s.Status)

			// Format URL - show "-" if not available or not running
			url := "-"
			if s.Endpoint != "" && s.Status == "running" {
				url = text.FgCyan.Sprint(s.Endpoint)
			}

			t.AppendRow(table.Row{
				s.Name,
				statusColor.Sprint(statusText),
				url,
			})
		}

		fmt.Println()
		t.Render()
		fmt.Println()

		return nil
	},
}
