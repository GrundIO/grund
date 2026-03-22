package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GrundIO/grund/internal/cli/prompts"
	"github.com/GrundIO/grund/internal/cli/skills"
	"github.com/GrundIO/grund/internal/config"
	"github.com/GrundIO/grund/internal/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Grund and set up AI assistant skills",
	Long: `Initialize Grund global configuration and set up AI assistant skills.

This command performs three tasks:
  1. Creates global config at ~/.grund/config.yaml
  2. Sets up your projects folder and registers services
  3. Installs AI assistant skills for Claude Code and/or Cursor

Example:
  grund init`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	ui.Header("Grund Initialization")
	fmt.Println()

	// Step 1: Handle config initialization
	if err := handleConfigInit(); err != nil {
		return fmt.Errorf("config initialization failed: %w", err)
	}

	// Step 2: Handle services setup
	fmt.Println()
	if err := handleServicesInit(); err != nil {
		return fmt.Errorf("services setup failed: %w", err)
	}

	// Step 3: Handle AI skills installation
	fmt.Println()
	if err := handleAISkillsInit(); err != nil {
		return fmt.Errorf("AI skills setup failed: %w", err)
	}

	fmt.Println()
	ui.Successf("Grund initialization complete!")
	return nil
}

// handleConfigInit handles the global config initialization.
func handleConfigInit() error {
	ui.Step("Step 1: Global Configuration")

	configPath, _ := config.GetGlobalConfigPath()

	if config.GlobalConfigExists() {
		ui.Infof("Config exists at: %s", configPath)

		reinit, err := prompts.Confirm("Re-initialize config?", false)
		if err != nil {
			return fmt.Errorf("prompt failed: %w", err)
		}

		if reinit {
			if err := config.ForceInitGlobalConfig(); err != nil {
				return fmt.Errorf("failed to re-initialize config: %w", err)
			}
			ui.Successf("Config re-initialized")
		} else {
			ui.Infof("Keeping existing config")
		}
	} else {
		if err := config.InitGlobalConfig(); err != nil {
			return fmt.Errorf("failed to initialize config: %w", err)
		}
		ui.Successf("Config created at: %s", configPath)
	}

	return nil
}

// handleServicesInit handles setting up the services folder and registering projects
func handleServicesInit() error {
	ui.Step("Step 2: Services Setup")

	// Get home directory for default path
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Check if services.yaml already exists
	grundHome, _ := config.GetGrundHome()
	servicesPath := filepath.Join(grundHome, "services.yaml")
	servicesExist := fileExists(servicesPath)

	if servicesExist {
		ui.Infof("Services registry exists at: %s", servicesPath)
	}

	// Ask for projects folder
	defaultFolder := filepath.Join(home, "projects")

	projectsFolder, err := prompts.Text("Projects folder (where your services live)", defaultFolder)
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	projectsFolder = expandPath(projectsFolder)

	// Check if folder exists
	if !dirExists(projectsFolder) {
		ui.Warnf("Folder does not exist: %s", projectsFolder)
		create, err := prompts.Confirm("Create this folder?", true)
		if err != nil {
			return fmt.Errorf("prompt failed: %w", err)
		}
		if create {
			if err := os.MkdirAll(projectsFolder, 0755); err != nil {
				return fmt.Errorf("failed to create folder: %w", err)
			}
			ui.Successf("Created folder: %s", projectsFolder)
		}
		return nil
	}

	// Scan for projects (directories with grund.yaml or any directory)
	projects, err := scanProjects(projectsFolder)
	if err != nil {
		return fmt.Errorf("failed to scan projects: %w", err)
	}

	if len(projects) == 0 {
		ui.Infof("No projects found in %s", projectsFolder)
		return nil
	}

	// Ask if user wants to add projects
	addProjects, err := prompts.Confirm(fmt.Sprintf("Found %d projects. Add them to services registry?", len(projects)), true)
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}

	if !addProjects {
		ui.Infof("Skipping project registration")
		return nil
	}

	// Let user select which projects to add
	selectedProjects, err := prompts.MultiSelect("Select projects to register", projects)
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}

	if len(selectedProjects) == 0 {
		ui.Infof("No projects selected")
		return nil
	}

	// Load or create services.yaml
	services := loadServicesYAML(servicesPath)

	// Add selected projects
	for _, project := range selectedProjects {
		serviceName := filepath.Base(project)
		projectPath := filepath.Join(projectsFolder, project)

		services.Services[serviceName] = ServiceEntry{
			Path: contractPath(projectPath),
		}
		ui.Successf("Added: %s", serviceName)
	}

	// Update path defaults
	services.PathDefaults.Base = contractPath(projectsFolder)

	// Save services.yaml
	if err := saveServicesYAML(servicesPath, services); err != nil {
		return fmt.Errorf("failed to save services.yaml: %w", err)
	}

	ui.Successf("Services registry updated: %s", servicesPath)
	return nil
}

// handleAISkillsInit handles the AI assistant skills installation.
func handleAISkillsInit() error {
	ui.Step("Step 3: AI Assistant Skills")

	// Show current status
	claudeInstalled := skills.IsInstalled(skills.Claude)
	cursorInstalled := skills.IsInstalled(skills.Cursor)

	if claudeInstalled {
		ui.Infof("Claude Code: installed")
	}
	if cursorInstalled {
		ui.Infof("Cursor: installed")
	}

	// Ask if user wants to set up skills
	setupSkills, err := prompts.Confirm("Set up AI assistant skills?", true)
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}

	if !setupSkills {
		ui.Infof("Skipping AI assistant setup")
		return nil
	}

	// Ask which assistants to install
	choice, err := prompts.Select("Which AI assistant?", []string{"Claude Code", "Cursor", "Both"}, "Both")
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}

	var toInstall []skills.AIAssistant
	switch choice {
	case "Claude Code":
		toInstall = []skills.AIAssistant{skills.Claude}
	case "Cursor":
		toInstall = []skills.AIAssistant{skills.Cursor}
	default: // "Both"
		toInstall = []skills.AIAssistant{skills.Claude, skills.Cursor}
	}

	// Install selected assistants
	for _, assistant := range toInstall {
		if err := skills.Install(assistant); err != nil {
			return fmt.Errorf("failed to install %s skill: %w", skills.AssistantName(assistant), err)
		}
		path, _ := skills.SkillPaths(assistant)
		ui.Successf("%s installed at: %s", skills.AssistantName(assistant), path)
	}

	return nil
}

// Helper types for services.yaml
type ServicesYAML struct {
	Version      string                  `yaml:"version"`
	Services     map[string]ServiceEntry `yaml:"services"`
	PathDefaults PathDefaults            `yaml:"path_defaults,omitempty"`
}

type ServiceEntry struct {
	Path string `yaml:"path"`
	Repo string `yaml:"repo,omitempty"`
}

type PathDefaults struct {
	Base string `yaml:"base,omitempty"`
}

// Helper functions

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func contractPath(path string) string {
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

func scanProjects(folder string) ([]string, error) {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil, err
	}

	var projects []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Skip hidden directories
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		projects = append(projects, entry.Name())
	}
	return projects, nil
}

func loadServicesYAML(path string) *ServicesYAML {
	services := &ServicesYAML{
		Version:  "1",
		Services: make(map[string]ServiceEntry),
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return services
	}

	yaml.Unmarshal(data, services)
	if services.Services == nil {
		services.Services = make(map[string]ServiceEntry)
	}
	return services
}

func saveServicesYAML(path string, services *ServicesYAML) error {
	data, err := yaml.Marshal(services)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
