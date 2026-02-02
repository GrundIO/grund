# Grund Init Command Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a unified `grund init` command that bootstraps global config and optionally installs AI assistant skills.

**Architecture:** Interactive CLI flow with conditional prompts. Skill content is embedded in the binary, copied to appropriate locations based on user selection.

**Tech Stack:** Go, Cobra CLI, bufio for interactive prompts

---

## Flow Diagram

```
grund init
├── Config exists? (~/.grund/config.yaml)
│   ├── Yes → Prompt: "Re-initialize config? (y/n)"
│   │   ├── y → Create fresh config
│   │   └── n → Skip config setup
│   └── No → Create config
│
└── AI skills exist? (check both locations)
    ├── All exist → Skip (print "AI skills already configured")
    └── Any missing → Prompt: "Set up AI assistant skills? (y/n)"
        ├── y → Prompt: "Which? (1=Claude, 2=Cursor, 3=Both)"
        │   └── Install selected skills
        └── n → Skip
```

---

## Task 1: Create the init command skeleton

**Files:**
- Create: `internal/cli/init.go`

**Step 1: Create basic command structure**

```go
package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/Saturn-Fintech/grund/internal/config"
	"github.com/Saturn-Fintech/grund/internal/ui"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Grund and set up AI assistant skills",
	Long: `Initialize Grund configuration and optionally set up AI assistant skills.

This command will:
1. Create global configuration at ~/.grund/config.yaml (if not exists)
2. Optionally install AI assistant skills for Claude Code and/or Cursor

Examples:
  grund init                    # Interactive setup
  grund init --skip-ai          # Skip AI assistant setup`,
	RunE: runInit,
}

var skipAI bool

func init() {
	initCmd.Flags().BoolVar(&skipAI, "skip-ai", false, "Skip AI assistant skill setup")
}

func runInit(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	// Step 1: Handle global config
	if err := handleConfigInit(reader); err != nil {
		return err
	}

	// Step 2: Handle AI skills (unless skipped)
	if !skipAI {
		if err := handleAISkillsInit(reader); err != nil {
			return err
		}
	}

	ui.Successf("Grund initialization complete!")
	return nil
}

func handleConfigInit(reader *bufio.Reader) error {
	// TODO: Implement in Task 2
	return nil
}

func handleAISkillsInit(reader *bufio.Reader) error {
	// TODO: Implement in Task 3
	return nil
}

// promptYesNo asks a yes/no question and returns true for yes
func promptYesNo(reader *bufio.Reader, question string) bool {
	fmt.Printf("%s (y/n): ", question)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}

// promptChoice asks user to select from numbered options, returns 1-based index
func promptChoice(reader *bufio.Reader, question string, options []string) int {
	fmt.Println(question)
	for i, opt := range options {
		fmt.Printf("  %d. %s\n", i+1, opt)
	}
	fmt.Print("Enter choice: ")

	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(answer)

	choice := 0
	fmt.Sscanf(answer, "%d", &choice)

	if choice < 1 || choice > len(options) {
		return 1 // Default to first option
	}
	return choice
}
```

**Step 2: Register command in root.go**

In `internal/cli/root.go`, add to the `init()` function:

```go
// Bootstrap (top-level)
rootCmd.AddCommand(initCmd)
```

**Step 3: Run build to verify syntax**

Run: `make build`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add internal/cli/init.go internal/cli/root.go
git commit -m "feat(cli): Add init command skeleton"
```

---

## Task 2: Implement config initialization with re-init prompt

**Files:**
- Modify: `internal/cli/init.go`
- Modify: `internal/config/global.go` (add ForceInitGlobalConfig)

**Step 1: Add ForceInitGlobalConfig to config package**

In `internal/config/global.go`, add:

```go
// ForceInitGlobalConfig initializes the global config, overwriting if exists
func ForceInitGlobalConfig() error {
	return SaveGlobalConfig(DefaultGlobalConfig())
}

// GlobalConfigExists checks if global config file exists
func GlobalConfigExists() bool {
	configPath, err := GetGlobalConfigPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(configPath)
	return err == nil
}
```

**Step 2: Implement handleConfigInit**

```go
func handleConfigInit(reader *bufio.Reader) error {
	configPath, _ := config.GetGlobalConfigPath()

	if config.GlobalConfigExists() {
		ui.Infof("Global config already exists at: %s", configPath)

		if promptYesNo(reader, "Do you want to re-initialize the config?") {
			if err := config.ForceInitGlobalConfig(); err != nil {
				return fmt.Errorf("failed to re-initialize config: %w", err)
			}
			ui.Successf("Config re-initialized at: %s", configPath)
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
```

**Step 3: Run test**

Run: `make test`
Expected: All tests pass

**Step 4: Manual test**

```bash
# Test fresh init
rm -f ~/.grund/config.yaml
go run . init --skip-ai
# Expected: "Config created at: ~/.grund/config.yaml"

# Test re-init prompt
go run . init --skip-ai
# Expected: Prompts for re-initialize
```

**Step 5: Commit**

```bash
git add internal/cli/init.go internal/config/global.go
git commit -m "feat(cli): Add config initialization with re-init prompt"
```

---

## Task 3: Create skill installer infrastructure

**Files:**
- Create: `internal/cli/skills/installer.go`
- Create: `internal/cli/skills/content.go` (embedded skill content)

**Step 1: Create skills package with installer**

```go
// internal/cli/skills/installer.go
package skills

import (
	"fmt"
	"os"
	"path/filepath"
)

type AIAssistant int

const (
	Claude AIAssistant = iota
	Cursor
)

// SkillPaths returns the installation paths for each assistant
func SkillPaths(assistant AIAssistant) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	switch assistant {
	case Claude:
		return filepath.Join(home, ".claude", "skills", "using-grund"), nil
	case Cursor:
		return filepath.Join(home, ".cursor", "skills-cursor", "using-grund"), nil
	default:
		return "", fmt.Errorf("unknown assistant type")
	}
}

// IsInstalled checks if skill is already installed for the given assistant
func IsInstalled(assistant AIAssistant) bool {
	path, err := SkillPaths(assistant)
	if err != nil {
		return false
	}
	skillFile := filepath.Join(path, "SKILL.md")
	_, err = os.Stat(skillFile)
	return err == nil
}

// Install installs the skill for the given assistant
func Install(assistant AIAssistant) error {
	path, err := SkillPaths(assistant)
	if err != nil {
		return err
	}

	// Create directory
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	// Write skill file
	skillFile := filepath.Join(path, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte(GrundSkillContent), 0644); err != nil {
		return fmt.Errorf("failed to write skill file: %w", err)
	}

	return nil
}

// AssistantName returns human-readable name
func AssistantName(assistant AIAssistant) string {
	switch assistant {
	case Claude:
		return "Claude Code"
	case Cursor:
		return "Cursor"
	default:
		return "Unknown"
	}
}
```

**Step 2: Create skill content file**

```go
// internal/cli/skills/content.go
package skills

// GrundSkillContent is the embedded skill content for AI assistants
const GrundSkillContent = `---
name: using-grund
description: Orchestrate local development services with Grund CLI. Use when starting services, managing dependencies, configuring infrastructure (Postgres, MongoDB, Redis, SQS, SNS, S3), or working with grund.yaml files.
allowed-tools:
  - Bash(grund *)
---

# Using Grund

Grund is a local development orchestration tool. It starts services with their dependencies (databases, queues, other services) using Docker Compose.

**Config location:** ` + "`~/.grund/`" + `
- ` + "`services.yaml`" + ` - Service registry
- ` + "`secrets.env`" + ` - Secret values (API keys)
- ` + "`config.yaml`" + ` - Global settings

## Quick Reference

### Daily Operations
` + "```bash" + `
grund up <service>              # Start service with all dependencies
grund down                      # Stop all services
grund status                    # Show all services and URLs
grund logs <service>            # View logs
grund restart <service>         # Restart specific service
` + "```" + `

### Service Configuration
` + "```bash" + `
grund service init              # Create grund.yaml interactively
grund service validate          # Validate grund.yaml
grund service add postgres      # Add PostgreSQL to service
grund service add redis         # Add Redis to service
grund service add queue <name>  # Add SQS queue
grund service add topic <name>  # Add SNS topic
grund service add bucket <name> # Add S3 bucket
` + "```" + `

### Secrets Management
` + "```bash" + `
grund secrets list <service>    # Check required secrets
grund secrets init <service>    # Generate secrets.env template
` + "```" + `

### Configuration
` + "```bash" + `
grund config show               # Show resolved config
grund config init               # Initialize global config
` + "```" + `

---

## Configuration (grund.yaml)

### Minimal Example
` + "```yaml" + `
version: "1"
service:
  name: my-service
  type: go
  port: 8080
  build:
    dockerfile: Dockerfile
    context: .
  health:
    endpoint: /health

requires:
  services: []
  infrastructure: {}

env:
  APP_ENV: development
` + "```" + `

### Adding Infrastructure
` + "```yaml" + `
requires:
  infrastructure:
    postgres:
      database: myapp_db
    redis: true
    sqs:
      queues:
        - name: events
          dlq: true
    sns:
      topics:
        - name: notifications
    s3:
      buckets:
        - name: uploads
` + "```" + `

### Environment Variables
` + "```yaml" + `
env:
  LOG_LEVEL: debug

env_refs:
  DATABASE_URL: "postgres://postgres:postgres@${postgres.host}:${postgres.port}/${self.postgres.database}"
  REDIS_URL: "redis://${redis.host}:${redis.port}"
  AWS_ENDPOINT: "${localstack.endpoint}"
  QUEUE_URL: "${sqs.events.url}"

secrets:
  API_KEY:
    description: "External API key"
    required: true
` + "```" + `

---

## Workflows

### Starting a Service
` + "```bash" + `
grund secrets list my-service   # Check secrets needed
grund secrets init my-service   # Generate template if needed
grund up my-service             # Start with dependencies
grund status                    # Verify running
` + "```" + `

### Debugging Issues
` + "```bash" + `
grund status                    # Check service states
grund logs my-service           # View logs
grund config show my-service    # Check resolved config
grund reset -v                  # Nuclear option: reset everything
` + "```" + `
`
```

**Step 3: Run build**

Run: `make build`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add internal/cli/skills/
git commit -m "feat(cli): Add skill installer infrastructure"
```

---

## Task 4: Implement AI skills setup flow

**Files:**
- Modify: `internal/cli/init.go`

**Step 1: Implement handleAISkillsInit**

```go
import (
	// ... existing imports
	"github.com/Saturn-Fintech/grund/internal/cli/skills"
)

func handleAISkillsInit(reader *bufio.Reader) error {
	fmt.Println()

	// Check what's already installed
	claudeInstalled := skills.IsInstalled(skills.Claude)
	cursorInstalled := skills.IsInstalled(skills.Cursor)

	if claudeInstalled && cursorInstalled {
		ui.Successf("AI assistant skills already configured for Claude Code and Cursor")
		return nil
	}

	// Build status message
	if claudeInstalled {
		ui.Infof("Claude Code skill already installed")
	}
	if cursorInstalled {
		ui.Infof("Cursor skill already installed")
	}

	// Ask if user wants to set up skills
	if !promptYesNo(reader, "Would you like to set up AI assistant skills for Grund?") {
		ui.Infof("Skipping AI assistant setup")
		return nil
	}

	// Determine which options to show
	var options []string
	var assistants [][]skills.AIAssistant

	if !claudeInstalled && !cursorInstalled {
		options = []string{"Claude Code", "Cursor", "Both"}
		assistants = [][]skills.AIAssistant{
			{skills.Claude},
			{skills.Cursor},
			{skills.Claude, skills.Cursor},
		}
	} else if !claudeInstalled {
		options = []string{"Claude Code"}
		assistants = [][]skills.AIAssistant{{skills.Claude}}
	} else {
		options = []string{"Cursor"}
		assistants = [][]skills.AIAssistant{{skills.Cursor}}
	}

	choice := 1
	if len(options) > 1 {
		choice = promptChoice(reader, "Which AI assistant would you like to configure?", options)
	}

	// Install selected assistants
	for _, assistant := range assistants[choice-1] {
		if err := skills.Install(assistant); err != nil {
			return fmt.Errorf("failed to install %s skill: %w", skills.AssistantName(assistant), err)
		}

		path, _ := skills.SkillPaths(assistant)
		ui.Successf("%s skill installed at: %s", skills.AssistantName(assistant), path)
	}

	return nil
}
```

**Step 2: Run build and test**

Run: `make build && make test`
Expected: All pass

**Step 3: Manual integration test**

```bash
# Clean slate test
rm -rf ~/.grund/config.yaml ~/.claude/skills/using-grund ~/.cursor/skills-cursor/using-grund

# Run init
./bin/grund init
# Expected: Prompts for config, then AI assistants

# Verify files created
ls ~/.grund/config.yaml
ls ~/.claude/skills/using-grund/SKILL.md
ls ~/.cursor/skills-cursor/using-grund/SKILL.md
```

**Step 4: Commit**

```bash
git add internal/cli/init.go
git commit -m "feat(cli): Implement AI skills setup flow"
```

---

## Task 5: Update existing commands and documentation

**Files:**
- Modify: `internal/cli/configcmd/init.go` (add deprecation note)
- Modify: `internal/cli/root.go` (reorder commands)

**Step 1: Add note to config init**

In `internal/cli/configcmd/init.go`, update the Long description:

```go
Long: `Initialize the global Grund configuration at ~/.grund/config.yaml.

NOTE: For first-time setup, consider using 'grund init' instead, which also
sets up AI assistant skills.

This command only creates the config file. For full initialization including
AI assistant setup, use 'grund init'.

Example:
  grund config init`,
```

**Step 2: Update root.go command order**

Ensure init is near the top:

```go
func init() {
	// Bootstrap
	rootCmd.AddCommand(initCmd)

	// Daily operations (most frequently used)
	rootCmd.AddCommand(upCmd)
	// ... rest
}
```

**Step 3: Test help output**

```bash
./bin/grund --help
# Verify init appears prominently

./bin/grund init --help
# Verify description is clear
```

**Step 4: Commit**

```bash
git add internal/cli/configcmd/init.go internal/cli/root.go
git commit -m "docs(cli): Update command descriptions for init"
```

---

## Task 6: Add tests

**Files:**
- Create: `internal/cli/skills/installer_test.go`

**Step 1: Write installer tests**

```go
package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSkillPaths(t *testing.T) {
	tests := []struct {
		assistant AIAssistant
		contains  string
	}{
		{Claude, ".claude/skills/using-grund"},
		{Cursor, ".cursor/skills-cursor/using-grund"},
	}

	for _, tt := range tests {
		path, err := SkillPaths(tt.assistant)
		if err != nil {
			t.Errorf("SkillPaths(%v) returned error: %v", tt.assistant, err)
		}
		if !filepath.IsAbs(path) {
			t.Errorf("SkillPaths(%v) returned non-absolute path: %s", tt.assistant, path)
		}
		if !contains(path, tt.contains) {
			t.Errorf("SkillPaths(%v) = %s, expected to contain %s", tt.assistant, path, tt.contains)
		}
	}
}

func TestIsInstalled(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Test when not installed
	// Note: This tests against actual home directory, so result depends on system state
	// In real tests, we'd mock the home directory
}

func TestAssistantName(t *testing.T) {
	tests := []struct {
		assistant AIAssistant
		expected  string
	}{
		{Claude, "Claude Code"},
		{Cursor, "Cursor"},
	}

	for _, tt := range tests {
		name := AssistantName(tt.assistant)
		if name != tt.expected {
			t.Errorf("AssistantName(%v) = %s, expected %s", tt.assistant, name, tt.expected)
		}
	}
}

func contains(s, substr string) bool {
	return filepath.Clean(s) != "" && filepath.Clean(substr) != "" &&
		len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) &&
			(s[len(s)-len(substr):] == substr ||
			 s[:len(substr)] == substr ||
			 containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

**Step 2: Run tests**

Run: `make test`
Expected: All tests pass

**Step 3: Commit**

```bash
git add internal/cli/skills/installer_test.go
git commit -m "test(cli): Add skill installer tests"
```

---

## Task 7: Final integration test and cleanup

**Step 1: Full end-to-end test**

```bash
# Clean everything
rm -rf ~/.grund/config.yaml ~/.claude/skills/using-grund ~/.cursor/skills-cursor/using-grund

# Build fresh
make build

# Run full init
./bin/grund init
# Answer: y to config, y to AI skills, 3 for both

# Verify all files
cat ~/.grund/config.yaml
cat ~/.claude/skills/using-grund/SKILL.md
cat ~/.cursor/skills-cursor/using-grund/SKILL.md

# Run again - should detect existing
./bin/grund init
# Should show "already configured" messages

# Test --skip-ai flag
rm ~/.grund/config.yaml
./bin/grund init --skip-ai
# Should only create config, no AI prompts
```

**Step 2: Run full test suite**

Run: `make test`
Expected: All tests pass

**Step 3: Final commit**

```bash
git add -A
git commit -m "feat(cli): Complete grund init command with AI skill setup"
```

---

## Summary

After completing all tasks:

1. `grund init` command provides one-stop bootstrap
2. Interactive prompts for config re-initialization
3. Optional AI skill setup for Claude Code and/or Cursor
4. `--skip-ai` flag for automation
5. Skills installed to correct locations
6. Existing installations detected and respected
