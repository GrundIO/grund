package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/Saturn-Fintech/grund/internal/application/commands"
	"github.com/Saturn-Fintech/grund/internal/application/ports"
	"github.com/Saturn-Fintech/grund/internal/cli/shared"
	"github.com/Saturn-Fintech/grund/internal/ui"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
)

var cloneSync bool

var cloneCmd = &cobra.Command{
	Use:   "clone [services...]",
	Short: "Clone service repositories",
	Long: `Clone all (or specific) service repositories defined in services.yaml.

By default, services that are already cloned are skipped.
Use --sync to pull the latest changes for existing repositories.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if shared.Container == nil {
			return fmt.Errorf("container not initialized")
		}

		cloneCmd := commands.CloneCommand{
			ServiceNames: args,
			Sync:         cloneSync,
		}

		results, err := shared.Container.CloneCommandHandler.Handle(cmd.Context(), cloneCmd)
		if err != nil {
			return err
		}

		if len(results) == 0 {
			return nil
		}

		return renderCloneResults(results)
	},
}

func init() {
	cloneCmd.Flags().BoolVar(&cloneSync, "sync", false, "Pull latest changes for already cloned repositories")
}

func renderCloneResults(results []ports.CloneResult) error {
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
			comment = "use --sync to pull"
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
		return fmt.Errorf("clone completed with %d error(s): %s", errored, strings.Join(errors, "; "))
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
