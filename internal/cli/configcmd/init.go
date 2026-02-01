package configcmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vivekkundariya/grund/internal/config"
	"github.com/vivekkundariya/grund/internal/ui"
)

var initCmd = &cobra.Command{
	Use:        "init",
	Short:      "[DEPRECATED] Use 'grund init' instead",
	Long:       `DEPRECATED: This command is deprecated and will be removed in a future version.

Use 'grund init' instead, which provides a complete interactive setup including:
  - Global configuration
  - Services registration
  - AI assistant skills installation

For manual setup, create ~/.grund/config.yaml directly.`,
	Deprecated: "use 'grund init' for interactive setup or manually create ~/.grund/config.yaml",
	RunE:       runConfigInit,
}

func init() {
	Cmd.AddCommand(initCmd)
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	ui.Warnf("'grund config init' is deprecated. Use 'grund init' instead.")
	fmt.Println()

	if err := config.InitGlobalConfig(); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	configPath, _ := config.GetGlobalConfigPath()
	fmt.Printf("Global config initialized at: %s\n", configPath)
	fmt.Println("\nYou can customize this file to set default paths and options.")
	return nil
}
