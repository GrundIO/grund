package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GrundIO/grund/internal/application/queries"
	"github.com/GrundIO/grund/internal/cli/shared"
	"github.com/GrundIO/grund/internal/domain/service"
	"github.com/GrundIO/grund/internal/infrastructure/generator"
	"github.com/GrundIO/grund/internal/ui"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
)

var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage service secrets",
	Long: `Manage secrets required by services.

Secrets are stored in ~/.grund/secrets.env and are injected into containers
when running services with 'grund up'.

Examples:
  grund secrets list user-service    Show secrets required by service
  grund secrets init user-service    Generate secrets.env template`,
}

var secretsListCmd = &cobra.Command{
	Use:   "list [service...]",
	Short: "List secrets required by services",
	Long: `Show all secrets required by the specified services and their dependencies.

Indicates which secrets are found, missing, or optional.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSecretsList,
}

var secretsInitCmd = &cobra.Command{
	Use:   "init [service...]",
	Short: "Generate secrets.env template",
	Long: `Generate ~/.grund/secrets.env with placeholders for missing secrets.

If the file already exists, only missing secrets are appended.
Existing values are preserved.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSecretsInit,
}

func init() {
	secretsCmd.AddCommand(secretsListCmd)
	secretsCmd.AddCommand(secretsInitCmd)
}

func runSecretsList(cmd *cobra.Command, args []string) error {
	if shared.Container == nil {
		return fmt.Errorf("container not initialized")
	}

	// Get services and their dependencies
	services, err := getServicesWithDependencies(cmd, args)
	if err != nil {
		return err
	}

	if len(services) == 0 {
		ui.Infof("No services found")
		return nil
	}

	// Load and validate secrets
	loader := generator.NewSecretsLoader()
	statuses, err := loader.ValidateSecrets(services)
	if err != nil {
		return fmt.Errorf("failed to validate secrets: %w", err)
	}

	if len(statuses) == 0 {
		ui.Infof("No secrets declared for these services")
		return nil
	}

	// Sort statuses by name for consistent output
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].Name < statuses[j].Name
	})

	// Print service names
	serviceNames := make([]string, len(args))
	copy(serviceNames, args)
	fmt.Printf("\nSecrets for %s (and dependencies):\n\n", strings.Join(serviceNames, ", "))

	// Build table
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)
	t.AppendHeader(table.Row{"Secret", "Status", "Description"})

	missingRequired := 0
	for _, s := range statuses {
		var statusIcon string
		var statusColor text.Color
		var statusText string

		if s.Found {
			statusIcon = "✓"
			statusColor = text.FgGreen
			statusText = "found"
			if s.Source == "file" {
				statusText = "found (file)"
			} else {
				statusText = "found (env)"
			}
		} else if s.Required {
			statusIcon = "✗"
			statusColor = text.FgRed
			statusText = "missing"
			missingRequired++
		} else {
			statusIcon = "○"
			statusColor = text.FgYellow
			statusText = "optional"
		}

		t.AppendRow(table.Row{
			s.Name,
			statusColor.Sprintf("%s %s", statusIcon, statusText),
			s.Description,
		})
	}

	t.Render()

	// Print summary
	fmt.Println()
	if err := loader.Load(); err == nil {
		fmt.Printf("Source: %s (%d keys loaded)\n", loader.GetSecretsFilePath(), loader.GetLoadedCount())
	}

	if missingRequired > 0 {
		fmt.Println()
		ui.Errorf("Missing required secrets: %d", missingRequired)
		fmt.Println("Run 'grund secrets init <service>' to generate a template.")
		return fmt.Errorf("missing required secrets")
	}

	return nil
}

func runSecretsInit(cmd *cobra.Command, args []string) error {
	if shared.Container == nil {
		return fmt.Errorf("container not initialized")
	}

	// Get services and their dependencies
	services, err := getServicesWithDependencies(cmd, args)
	if err != nil {
		return err
	}

	if len(services) == 0 {
		ui.Infof("No services found")
		return nil
	}

	// Load and validate secrets
	loader := generator.NewSecretsLoader()
	statuses, err := loader.ValidateSecrets(services)
	if err != nil {
		return fmt.Errorf("failed to validate secrets: %w", err)
	}

	// Filter to only missing secrets
	var missing []generator.SecretStatus
	for _, s := range statuses {
		if !s.Found {
			missing = append(missing, s)
		}
	}

	if len(missing) == 0 {
		ui.Successf("All secrets are already configured!")
		return nil
	}

	// Sort by name
	sort.Slice(missing, func(i, j int) bool {
		return missing[i].Name < missing[j].Name
	})

	// Get secrets file path
	secretsPath := loader.GetSecretsFilePath()

	// Ensure directory exists
	dir := filepath.Dir(secretsPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Read existing content if file exists
	existingContent := ""
	if data, err := os.ReadFile(secretsPath); err == nil {
		existingContent = string(data)
	}

	// Build new content to append
	var newLines []string
	existingKeys := parseExistingKeys(existingContent)

	for _, s := range missing {
		// Skip if key already exists in file (even if empty)
		if _, exists := existingKeys[s.Name]; exists {
			continue
		}

		comment := ""
		if s.Description != "" {
			comment = fmt.Sprintf("  # %s", s.Description)
		}
		if !s.Required {
			comment += " (optional)"
		}
		newLines = append(newLines, fmt.Sprintf("%s=%s", s.Name, comment))
	}

	if len(newLines) == 0 {
		ui.Successf("All secrets already have placeholders in %s", secretsPath)
		return nil
	}

	// Append to file
	file, err := os.OpenFile(secretsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open secrets file: %w", err)
	}
	defer file.Close()

	// Add newline before appending if file has content and doesn't end with newline
	if existingContent != "" && !strings.HasSuffix(existingContent, "\n") {
		file.WriteString("\n")
	}

	// Add a comment header if this is new content being added
	if existingContent == "" {
		file.WriteString("# Grund secrets - Auto-generated template\n")
		file.WriteString("# Add your secret values and remove the comments\n\n")
	} else {
		file.WriteString("\n# Added by grund secrets init\n")
	}

	for _, line := range newLines {
		file.WriteString(line + "\n")
	}

	// Print summary
	if existingContent == "" {
		fmt.Printf("\nCreated %s with %d placeholders:\n\n", secretsPath, len(newLines))
	} else {
		fmt.Printf("\nAppended %d placeholders to %s:\n\n", len(newLines), secretsPath)
	}

	for _, s := range missing {
		if _, exists := existingKeys[s.Name]; exists {
			continue
		}
		fmt.Printf("  %s\n", s.Name)
	}

	fmt.Println("\nEdit the file and add your secret values.")
	return nil
}

// getServicesWithDependencies loads services and their dependencies
func getServicesWithDependencies(cmd *cobra.Command, serviceNames []string) ([]*service.Service, error) {
	var allServices []*service.Service
	seen := make(map[string]bool)

	for _, name := range serviceNames {
		// Get service config
		query := queries.ConfigQuery{ServiceName: name}
		result, err := shared.Container.ConfigQueryHandler.Handle(query)
		if err != nil {
			return nil, fmt.Errorf("failed to get service %s: %w", name, err)
		}

		// Add the service itself
		if !seen[result.Service.Name] {
			seen[result.Service.Name] = true
			allServices = append(allServices, result.Service)
		}

		// Add dependencies (recursively load each)
		for _, depName := range result.Dependencies {
			if seen[depName] {
				continue
			}
			depQuery := queries.ConfigQuery{ServiceName: depName}
			depResult, err := shared.Container.ConfigQueryHandler.Handle(depQuery)
			if err != nil {
				ui.Warnf("Could not load dependency %s: %v", depName, err)
				continue
			}
			seen[depName] = true
			allServices = append(allServices, depResult.Service)
		}
	}

	return allServices, nil
}

// parseExistingKeys extracts key names from existing secrets file content
func parseExistingKeys(content string) map[string]bool {
	keys := make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) >= 1 {
			keys[strings.TrimSpace(parts[0])] = true
		}
	}
	return keys
}
