package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/GrundIO/grund/internal/application/commands"
	"github.com/GrundIO/grund/internal/application/ports"
	"github.com/GrundIO/grund/internal/cli/shared"
	"github.com/GrundIO/grund/internal/ui"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
)

var syncNoPull bool

var syncCmd = &cobra.Command{
	Use:   "sync [services...]",
	Short: "Sync service repositories (clone missing, pull existing)",
	Long: `Sync all (or specific) service repositories defined in services.yaml.

Missing repositories are cloned. Existing repositories are pulled to the latest.
Use --no-pull to only clone missing repositories without pulling existing ones.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if shared.Container == nil {
			return fmt.Errorf("container not initialized")
		}

		syncCmd := commands.SyncCommand{
			ServiceNames: args,
			NoPull:       syncNoPull,
		}

		results, err := shared.Container.SyncCommandHandler.Handle(cmd.Context(), syncCmd)
		if err != nil {
			return err
		}

		if len(results) == 0 {
			return nil
		}

		return renderSyncResults(results)
	},
}

func init() {
	syncCmd.Flags().BoolVar(&syncNoPull, "no-pull", false, "Only clone missing repositories, skip pulling existing ones")
}

func renderSyncResults(results []ports.CloneResult) error {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)

	t.AppendHeader(table.Row{"Service", "Status", "Path", "Comment"})

	var errored int
	var errors []string

	for _, r := range results {
		var statusIcon string
		var statusColor text.Color
		var statusText string
		var comment string

		switch r.Action {
		case "cloned":
			statusIcon = "●"
			statusColor = text.FgGreen
			statusText = "cloned"
			comment = "-"
		case "pulled":
			statusIcon = "●"
			statusColor = text.FgCyan
			statusText = "pulled"
			comment = "-"
		case "skipped":
			statusIcon = "○"
			statusColor = text.FgYellow
			statusText = "skipped"
			comment = "--no-pull"
		default:
			statusIcon = "●"
			statusColor = text.FgRed
			statusText = "error"
			errored++
			errors = append(errors, fmt.Sprintf("%s: %v", r.ServiceName, r.Error))
			if r.Error != nil {
				comment = text.FgRed.Sprint(r.Error.Error())
			}
		}

		t.AppendRow(table.Row{
			r.ServiceName,
			statusColor.Sprintf("%s %s", statusIcon, statusText),
			r.Path,
			comment,
		})
	}

	fmt.Println()
	t.Render()
	fmt.Println()

	cloned := countByAction(results, "cloned")
	pulled := countByAction(results, "pulled")
	skipped := countByAction(results, "skipped")

	ui.Infof("Cloned: %d, Pulled: %d, Skipped: %d, Errors: %d", cloned, pulled, skipped, errored)

	if errored > 0 {
		return fmt.Errorf("sync completed with %d error(s): %s", errored, strings.Join(errors, "; "))
	}

	return nil
}

func countByAction(results []ports.CloneResult, action string) int {
	count := 0
	for _, r := range results {
		if r.Action == action {
			count++
		}
	}
	return count
}
