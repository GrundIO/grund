package generator

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GrundIO/grund/internal/domain/service"
	"github.com/GrundIO/grund/internal/ui"
)

// SecretsLoader loads secrets from ~/.grund/secrets.env and shell environment
type SecretsLoader struct {
	secretsFilePath string
	fileSecrets     map[string]string
	loaded          bool
}

// NewSecretsLoader creates a new secrets loader
func NewSecretsLoader() *SecretsLoader {
	homeDir, _ := os.UserHomeDir()
	return &SecretsLoader{
		secretsFilePath: filepath.Join(homeDir, ".grund", "secrets.env"),
		fileSecrets:     make(map[string]string),
	}
}

// Load loads secrets from the secrets file
func (l *SecretsLoader) Load() error {
	if l.loaded {
		return nil
	}

	l.fileSecrets = make(map[string]string)

	file, err := os.Open(l.secretsFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			ui.Debug("No secrets file found at %s, using shell environment only", l.secretsFilePath)
			l.loaded = true
			return nil
		}
		return fmt.Errorf("failed to open secrets file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			ui.Warnf("Invalid line %d in secrets file: %s", lineNum, line)
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove surrounding quotes if present
		value = strings.Trim(value, `"'`)

		l.fileSecrets[key] = value
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read secrets file: %w", err)
	}

	ui.Debug("Loaded %d secrets from %s", len(l.fileSecrets), l.secretsFilePath)
	l.loaded = true
	return nil
}

// GetSecretValue gets a secret value with priority: secrets.env > shell environment
func (l *SecretsLoader) GetSecretValue(name string) (string, bool) {
	// Priority 1: secrets.env file
	if value, ok := l.fileSecrets[name]; ok && value != "" {
		return value, true
	}

	// Priority 2: shell environment
	if value := os.Getenv(name); value != "" {
		return value, true
	}

	return "", false
}

// SecretStatus represents the status of a secret
type SecretStatus struct {
	Name        string
	Description string
	Required    bool
	Found       bool
	Source      string // "file", "env", or ""
}

// ValidateSecrets validates that all required secrets are present
// Returns list of missing required secrets
func (l *SecretsLoader) ValidateSecrets(services []*service.Service) ([]SecretStatus, error) {
	if err := l.Load(); err != nil {
		return nil, err
	}

	// Collect all unique secrets from all services
	secretsMap := make(map[string]service.SecretRequirement)
	for _, svc := range services {
		for name, secret := range svc.Environment.Secrets {
			// If same secret defined in multiple services, use the one that's required
			if existing, ok := secretsMap[name]; ok {
				if secret.Required && !existing.Required {
					secretsMap[name] = secret
				}
			} else {
				secretsMap[name] = secret
			}
		}
	}

	var statuses []SecretStatus
	for name, secret := range secretsMap {
		status := SecretStatus{
			Name:        name,
			Description: secret.Description,
			Required:    secret.Required,
		}

		// Check if secret is available
		if _, ok := l.fileSecrets[name]; ok {
			status.Found = true
			status.Source = "file"
		} else if os.Getenv(name) != "" {
			status.Found = true
			status.Source = "env"
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}

// GetMissingRequired returns only the missing required secrets
func (l *SecretsLoader) GetMissingRequired(services []*service.Service) ([]SecretStatus, error) {
	statuses, err := l.ValidateSecrets(services)
	if err != nil {
		return nil, err
	}

	var missing []SecretStatus
	for _, s := range statuses {
		if s.Required && !s.Found {
			missing = append(missing, s)
		}
	}
	return missing, nil
}

// ResolveSecrets resolves all secrets for a service and returns a map of key=value
func (l *SecretsLoader) ResolveSecrets(svc *service.Service) (map[string]string, error) {
	if err := l.Load(); err != nil {
		return nil, err
	}

	resolved := make(map[string]string)
	for name := range svc.Environment.Secrets {
		if value, ok := l.GetSecretValue(name); ok {
			resolved[name] = value
		}
	}
	return resolved, nil
}

// GetSecretsFilePath returns the path to the secrets file
func (l *SecretsLoader) GetSecretsFilePath() string {
	return l.secretsFilePath
}

// GetLoadedCount returns the number of secrets loaded from file
func (l *SecretsLoader) GetLoadedCount() int {
	return len(l.fileSecrets)
}
