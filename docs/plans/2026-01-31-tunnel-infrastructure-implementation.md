# Tunnel Infrastructure Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add tunnel infrastructure support to expose local endpoints to the internet via cloudflared or ngrok.

**Architecture:** New `tunnel` package under `internal/infrastructure/tunnel/` with provider abstraction. Config structs in `schema.go`, env resolution in `env_resolver.go`, lifecycle integration in `up_command.go`.

**Tech Stack:** Go, os/exec for process management, regexp for URL parsing, context for cancellation.

---

## Task 1: Add Tunnel Config Structs

**Files:**
- Modify: `internal/config/schema.go`
- Test: `internal/config/schema_test.go` (create)

**Step 1: Write the failing test**

Create `internal/config/schema_test.go`:

```go
package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestTunnelConfigParsing(t *testing.T) {
	yamlContent := `
version: "1"
service:
  name: test-service
  type: go
  port: 8080
  health:
    endpoint: /health
    interval: 5s
    timeout: 3s
    retries: 10
requires:
  infrastructure:
    tunnel:
      provider: cloudflared
      targets:
        - name: localstack
          host: localhost
          port: "4566"
        - name: api
          host: "${self.host}"
          port: "${self.port}"
`
	var config ServiceConfig
	err := yaml.Unmarshal([]byte(yamlContent), &config)
	if err != nil {
		t.Fatalf("failed to parse yaml: %v", err)
	}

	if config.Requires.Infrastructure.Tunnel == nil {
		t.Fatal("expected tunnel config to be parsed")
	}
	if config.Requires.Infrastructure.Tunnel.Provider != "cloudflared" {
		t.Errorf("expected provider cloudflared, got %s", config.Requires.Infrastructure.Tunnel.Provider)
	}
	if len(config.Requires.Infrastructure.Tunnel.Targets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(config.Requires.Infrastructure.Tunnel.Targets))
	}
	if config.Requires.Infrastructure.Tunnel.Targets[0].Name != "localstack" {
		t.Errorf("expected target name localstack, got %s", config.Requires.Infrastructure.Tunnel.Targets[0].Name)
	}
	if config.Requires.Infrastructure.Tunnel.Targets[0].Port != "4566" {
		t.Errorf("expected port 4566, got %s", config.Requires.Infrastructure.Tunnel.Targets[0].Port)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config/... -run TestTunnelConfigParsing -v`
Expected: FAIL - Tunnel field doesn't exist

**Step 3: Write minimal implementation**

Add to `internal/config/schema.go` after `S3Config`:

```go
type TunnelConfig struct {
	Provider string         `yaml:"provider"` // "cloudflared" or "ngrok"
	Targets  []TunnelTarget `yaml:"targets"`
}

type TunnelTarget struct {
	Name string `yaml:"name"` // identifier for ${tunnel.<name>.url}
	Host string `yaml:"host"` // supports placeholders
	Port string `yaml:"port"` // string to support placeholders
}
```

Add `Tunnel` field to `InfrastructureConfig`:

```go
type InfrastructureConfig struct {
	Postgres *PostgresConfig `yaml:"postgres,omitempty"`
	MongoDB  *MongoDBConfig  `yaml:"mongodb,omitempty"`
	Redis    interface{}     `yaml:"redis,omitempty"`
	SQS      *SQSConfig      `yaml:"sqs,omitempty"`
	SNS      *SNSConfig      `yaml:"sns,omitempty"`
	S3       *S3Config       `yaml:"s3,omitempty"`
	Tunnel   *TunnelConfig   `yaml:"tunnel,omitempty"`
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/config/... -run TestTunnelConfigParsing -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/schema.go internal/config/schema_test.go
git commit -m "feat(config): add TunnelConfig and TunnelTarget structs"
```

---

## Task 2: Add TunnelContext to EnvironmentContext

**Files:**
- Modify: `internal/application/ports/compose.go`

**Step 1: Write the failing test**

Add to existing test file or create `internal/application/ports/compose_test.go`:

```go
package ports

import "testing"

func TestEnvironmentContextHasTunnel(t *testing.T) {
	ctx := NewDefaultEnvironmentContext()

	// Tunnel map should exist
	if ctx.Tunnel == nil {
		t.Fatal("expected Tunnel map to be initialized")
	}

	// Should be able to add tunnel context
	ctx.Tunnel["localstack"] = TunnelContext{
		Name:      "localstack",
		PublicURL: "https://abc.trycloudflare.com",
	}

	if ctx.Tunnel["localstack"].PublicURL != "https://abc.trycloudflare.com" {
		t.Error("expected tunnel context to be stored")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/application/ports/... -run TestEnvironmentContextHasTunnel -v`
Expected: FAIL - Tunnel field doesn't exist

**Step 3: Write minimal implementation**

Add to `internal/application/ports/compose.go` after `LocalStackContext`:

```go
// TunnelContext provides tunnel connection details
type TunnelContext struct {
	Name      string
	PublicURL string // Full URL like https://abc.trycloudflare.com
}
```

Add `Tunnel` field to `EnvironmentContext`:

```go
type EnvironmentContext struct {
	Infrastructure map[string]InfrastructureContext
	Services       map[string]ServiceContext
	Self           ServiceContext
	SQS            map[string]QueueContext
	SNS            map[string]TopicContext
	S3             map[string]BucketContext
	LocalStack     LocalStackContext
	Tunnel         map[string]TunnelContext // NEW
}
```

Update `NewDefaultEnvironmentContext()`:

```go
func NewDefaultEnvironmentContext() EnvironmentContext {
	return EnvironmentContext{
		Infrastructure: make(map[string]InfrastructureContext),
		Services:       make(map[string]ServiceContext),
		SQS:            make(map[string]QueueContext),
		SNS:            make(map[string]TopicContext),
		S3:             make(map[string]BucketContext),
		Tunnel:         make(map[string]TunnelContext), // NEW
		LocalStack: LocalStackContext{
			Endpoint:        "http://localstack:4566",
			Region:          "us-east-1",
			AccessKeyID:     "test",
			SecretAccessKey: "test",
			AccountID:       "000000000000",
		},
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/application/ports/... -run TestEnvironmentContextHasTunnel -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/application/ports/compose.go internal/application/ports/compose_test.go
git commit -m "feat(ports): add TunnelContext to EnvironmentContext"
```

---

## Task 3: Add Tunnel Placeholder Resolution

**Files:**
- Modify: `internal/infrastructure/generator/env_resolver.go`
- Modify: `internal/infrastructure/generator/env_resolver_test.go`

**Step 1: Write the failing test**

Add to `internal/infrastructure/generator/env_resolver_test.go`:

```go
func TestResolveTunnelPlaceholders(t *testing.T) {
	resolver := NewEnvironmentResolver()

	ctx := ports.NewDefaultEnvironmentContext()
	ctx.Tunnel["localstack"] = ports.TunnelContext{
		Name:      "localstack",
		PublicURL: "https://abc-xyz.trycloudflare.com",
	}
	ctx.Tunnel["api"] = ports.TunnelContext{
		Name:      "api",
		PublicURL: "https://def-123.trycloudflare.com",
	}

	envRefs := map[string]string{
		"PUBLIC_S3_ENDPOINT": "${tunnel.localstack.url}",
		"PUBLIC_API_URL":     "${tunnel.api.url}",
		"PUBLIC_S3_HOST":     "${tunnel.localstack.host}",
	}

	resolved, err := resolver.Resolve(envRefs, ctx)
	if err != nil {
		t.Fatalf("failed to resolve: %v", err)
	}

	if resolved["PUBLIC_S3_ENDPOINT"] != "https://abc-xyz.trycloudflare.com" {
		t.Errorf("expected https://abc-xyz.trycloudflare.com, got %s", resolved["PUBLIC_S3_ENDPOINT"])
	}
	if resolved["PUBLIC_API_URL"] != "https://def-123.trycloudflare.com" {
		t.Errorf("expected https://def-123.trycloudflare.com, got %s", resolved["PUBLIC_API_URL"])
	}
	if resolved["PUBLIC_S3_HOST"] != "abc-xyz.trycloudflare.com" {
		t.Errorf("expected abc-xyz.trycloudflare.com, got %s", resolved["PUBLIC_S3_HOST"])
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/infrastructure/generator/... -run TestResolveTunnelPlaceholders -v`
Expected: FAIL - unknown placeholder tunnel

**Step 3: Write minimal implementation**

Add case in `resolvePlaceholder` function in `env_resolver.go`:

```go
case "tunnel":
	return r.resolveTunnel(parts[1:], context)
```

Add new method:

```go
func (r *EnvironmentResolverImpl) resolveTunnel(parts []string, context ports.EnvironmentContext) (string, error) {
	if len(parts) < 2 {
		return "", fmt.Errorf("tunnel reference must be ${tunnel.<name>.<property>}")
	}

	tunnelName := parts[0]
	property := parts[1]

	tunnel, ok := context.Tunnel[tunnelName]
	if !ok {
		return "", fmt.Errorf("tunnel %s not found", tunnelName)
	}

	switch property {
	case "url":
		return tunnel.PublicURL, nil
	case "host":
		// Extract host from URL
		url := tunnel.PublicURL
		url = strings.TrimPrefix(url, "https://")
		url = strings.TrimPrefix(url, "http://")
		// Remove any path
		if idx := strings.Index(url, "/"); idx != -1 {
			url = url[:idx]
		}
		return url, nil
	default:
		return "", fmt.Errorf("unknown tunnel property %s", property)
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/infrastructure/generator/... -run TestResolveTunnelPlaceholders -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/infrastructure/generator/env_resolver.go internal/infrastructure/generator/env_resolver_test.go
git commit -m "feat(env): add tunnel placeholder resolution"
```

---

## Task 4: Create Tunnel Provider Interface

**Files:**
- Create: `internal/infrastructure/tunnel/provider.go`
- Create: `internal/infrastructure/tunnel/provider_test.go`

**Step 1: Write the test**

Create `internal/infrastructure/tunnel/provider_test.go`:

```go
package tunnel

import (
	"context"
	"testing"
)

func TestTunnelStructure(t *testing.T) {
	tunnel := &Tunnel{
		Name:      "localstack",
		PublicURL: "https://abc.trycloudflare.com",
		LocalAddr: "localhost:4566",
	}

	if tunnel.Name != "localstack" {
		t.Errorf("expected name localstack, got %s", tunnel.Name)
	}
	if tunnel.PublicURL != "https://abc.trycloudflare.com" {
		t.Errorf("expected URL https://abc.trycloudflare.com, got %s", tunnel.PublicURL)
	}
}

func TestProviderConstants(t *testing.T) {
	if ProviderCloudflared != "cloudflared" {
		t.Errorf("expected cloudflared constant")
	}
	if ProviderNgrok != "ngrok" {
		t.Errorf("expected ngrok constant")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/infrastructure/tunnel/... -v`
Expected: FAIL - package doesn't exist

**Step 3: Write minimal implementation**

Create `internal/infrastructure/tunnel/provider.go`:

```go
package tunnel

import (
	"context"
	"os"
)

const (
	ProviderCloudflared = "cloudflared"
	ProviderNgrok       = "ngrok"
)

// Tunnel represents a running tunnel
type Tunnel struct {
	Name      string      // identifier from config
	PublicURL string      // public URL like https://abc.trycloudflare.com
	LocalAddr string      // local address being tunneled like localhost:4566
	Process   *os.Process // the tunnel process
}

// Provider defines the interface for tunnel providers
type Provider interface {
	// Start creates a tunnel to the given local address
	Start(ctx context.Context, name string, localAddr string) (*Tunnel, error)
	// Stop terminates the tunnel
	Stop(tunnel *Tunnel) error
	// Name returns the provider name
	Name() string
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/infrastructure/tunnel/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/infrastructure/tunnel/
git commit -m "feat(tunnel): add Provider interface and Tunnel struct"
```

---

## Task 5: Implement Cloudflared Provider

**Files:**
- Create: `internal/infrastructure/tunnel/cloudflared.go`
- Create: `internal/infrastructure/tunnel/cloudflared_test.go`

**Step 1: Write the failing test**

Create `internal/infrastructure/tunnel/cloudflared_test.go`:

```go
package tunnel

import (
	"regexp"
	"testing"
)

func TestCloudflaredURLParsing(t *testing.T) {
	// Sample cloudflared output lines
	testCases := []struct {
		line     string
		expected string
	}{
		{
			line:     "2024-01-15T10:00:00Z INF |  https://random-words-here.trycloudflare.com",
			expected: "https://random-words-here.trycloudflare.com",
		},
		{
			line:     "INF +--------------------------------------------------------------------------------------------+",
			expected: "",
		},
		{
			line:     "https://abc-def-ghi.trycloudflare.com",
			expected: "https://abc-def-ghi.trycloudflare.com",
		},
	}

	pattern := regexp.MustCompile(`(https://[a-z0-9-]+\.trycloudflare\.com)`)

	for _, tc := range testCases {
		matches := pattern.FindStringSubmatch(tc.line)
		var result string
		if len(matches) > 1 {
			result = matches[1]
		}
		if result != tc.expected {
			t.Errorf("line %q: expected %q, got %q", tc.line, tc.expected, result)
		}
	}
}

func TestCloudflaredProviderName(t *testing.T) {
	provider := NewCloudflaredProvider()
	if provider.Name() != ProviderCloudflared {
		t.Errorf("expected %s, got %s", ProviderCloudflared, provider.Name())
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/infrastructure/tunnel/... -run TestCloudflared -v`
Expected: FAIL - NewCloudflaredProvider doesn't exist

**Step 3: Write minimal implementation**

Create `internal/infrastructure/tunnel/cloudflared.go`:

```go
package tunnel

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"time"
)

var cloudflaredURLPattern = regexp.MustCompile(`(https://[a-z0-9-]+\.trycloudflare\.com)`)

// CloudflaredProvider implements Provider for cloudflared
type CloudflaredProvider struct{}

// NewCloudflaredProvider creates a new cloudflared provider
func NewCloudflaredProvider() *CloudflaredProvider {
	return &CloudflaredProvider{}
}

// Name returns the provider name
func (p *CloudflaredProvider) Name() string {
	return ProviderCloudflared
}

// Start creates a tunnel using cloudflared
func (p *CloudflaredProvider) Start(ctx context.Context, name string, localAddr string) (*Tunnel, error) {
	// Check if cloudflared is installed
	if _, err := exec.LookPath("cloudflared"); err != nil {
		return nil, fmt.Errorf("cloudflared not found in PATH. Install with: brew install cloudflared")
	}

	// Start cloudflared tunnel
	cmd := exec.CommandContext(ctx, "cloudflared", "tunnel", "--url", fmt.Sprintf("http://%s", localAddr))

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start cloudflared: %w", err)
	}

	// Wait for URL with timeout
	urlChan := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if matches := cloudflaredURLPattern.FindStringSubmatch(line); len(matches) > 1 {
				urlChan <- matches[1]
				return
			}
		}
	}()

	select {
	case url := <-urlChan:
		return &Tunnel{
			Name:      name,
			PublicURL: url,
			LocalAddr: localAddr,
			Process:   cmd.Process,
		}, nil
	case <-time.After(30 * time.Second):
		cmd.Process.Kill()
		return nil, fmt.Errorf("timeout waiting for cloudflared URL")
	case <-ctx.Done():
		cmd.Process.Kill()
		return nil, ctx.Err()
	}
}

// Stop terminates the tunnel
func (p *CloudflaredProvider) Stop(tunnel *Tunnel) error {
	if tunnel == nil || tunnel.Process == nil {
		return nil
	}
	return tunnel.Process.Kill()
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/infrastructure/tunnel/... -run TestCloudflared -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/infrastructure/tunnel/cloudflared.go internal/infrastructure/tunnel/cloudflared_test.go
git commit -m "feat(tunnel): implement cloudflared provider"
```

---

## Task 6: Implement ngrok Provider

**Files:**
- Create: `internal/infrastructure/tunnel/ngrok.go`
- Create: `internal/infrastructure/tunnel/ngrok_test.go`

**Step 1: Write the failing test**

Create `internal/infrastructure/tunnel/ngrok_test.go`:

```go
package tunnel

import (
	"encoding/json"
	"testing"
)

func TestNgrokURLParsing(t *testing.T) {
	// Sample ngrok JSON log line
	logLine := `{"lvl":"info","msg":"started tunnel","obj":"tunnels","name":"command_line","addr":"http://localhost:4566","url":"https://abc123.ngrok-free.app"}`

	var logEntry struct {
		URL string `json:"url"`
	}

	err := json.Unmarshal([]byte(logLine), &logEntry)
	if err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if logEntry.URL != "https://abc123.ngrok-free.app" {
		t.Errorf("expected https://abc123.ngrok-free.app, got %s", logEntry.URL)
	}
}

func TestNgrokProviderName(t *testing.T) {
	provider := NewNgrokProvider()
	if provider.Name() != ProviderNgrok {
		t.Errorf("expected %s, got %s", ProviderNgrok, provider.Name())
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/infrastructure/tunnel/... -run TestNgrok -v`
Expected: FAIL - NewNgrokProvider doesn't exist

**Step 3: Write minimal implementation**

Create `internal/infrastructure/tunnel/ngrok.go`:

```go
package tunnel

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// NgrokProvider implements Provider for ngrok
type NgrokProvider struct{}

// NewNgrokProvider creates a new ngrok provider
func NewNgrokProvider() *NgrokProvider {
	return &NgrokProvider{}
}

// Name returns the provider name
func (p *NgrokProvider) Name() string {
	return ProviderNgrok
}

// ngrokLogEntry represents a JSON log entry from ngrok
type ngrokLogEntry struct {
	URL string `json:"url"`
	Msg string `json:"msg"`
}

// Start creates a tunnel using ngrok
func (p *NgrokProvider) Start(ctx context.Context, name string, localAddr string) (*Tunnel, error) {
	// Check if ngrok is installed
	if _, err := exec.LookPath("ngrok"); err != nil {
		return nil, fmt.Errorf("ngrok not found in PATH. Install from: https://ngrok.com/download")
	}

	// Extract port from localAddr
	parts := strings.Split(localAddr, ":")
	port := parts[len(parts)-1]

	// Start ngrok with JSON logging
	cmd := exec.CommandContext(ctx, "ngrok", "http", port, "--log", "stdout", "--log-format", "json")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ngrok: %w", err)
	}

	// Wait for URL with timeout
	urlChan := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			var entry ngrokLogEntry
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				continue
			}
			if entry.URL != "" && strings.HasPrefix(entry.URL, "https://") {
				urlChan <- entry.URL
				return
			}
		}
	}()

	select {
	case url := <-urlChan:
		return &Tunnel{
			Name:      name,
			PublicURL: url,
			LocalAddr: localAddr,
			Process:   cmd.Process,
		}, nil
	case <-time.After(30 * time.Second):
		cmd.Process.Kill()
		return nil, fmt.Errorf("timeout waiting for ngrok URL")
	case <-ctx.Done():
		cmd.Process.Kill()
		return nil, ctx.Err()
	}
}

// Stop terminates the tunnel
func (p *NgrokProvider) Stop(tunnel *Tunnel) error {
	if tunnel == nil || tunnel.Process == nil {
		return nil
	}
	return tunnel.Process.Kill()
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/infrastructure/tunnel/... -run TestNgrok -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/infrastructure/tunnel/ngrok.go internal/infrastructure/tunnel/ngrok_test.go
git commit -m "feat(tunnel): implement ngrok provider"
```

---

## Task 7: Create Tunnel Manager

**Files:**
- Create: `internal/infrastructure/tunnel/manager.go`
- Create: `internal/infrastructure/tunnel/manager_test.go`

**Step 1: Write the failing test**

Create `internal/infrastructure/tunnel/manager_test.go`:

```go
package tunnel

import (
	"testing"

	"github.com/Saturn-Fintech/grund/internal/config"
)

func TestManagerCreation(t *testing.T) {
	manager := NewManager()
	if manager == nil {
		t.Fatal("expected manager to be created")
	}
}

func TestManagerGetProvider(t *testing.T) {
	manager := NewManager()

	provider, err := manager.GetProvider(ProviderCloudflared)
	if err != nil {
		t.Fatalf("failed to get cloudflared provider: %v", err)
	}
	if provider.Name() != ProviderCloudflared {
		t.Errorf("expected cloudflared, got %s", provider.Name())
	}

	provider, err = manager.GetProvider(ProviderNgrok)
	if err != nil {
		t.Fatalf("failed to get ngrok provider: %v", err)
	}
	if provider.Name() != ProviderNgrok {
		t.Errorf("expected ngrok, got %s", provider.Name())
	}

	_, err = manager.GetProvider("invalid")
	if err == nil {
		t.Error("expected error for invalid provider")
	}
}

func TestManagerValidateConfig(t *testing.T) {
	manager := NewManager()

	// Valid config
	validConfig := &config.TunnelConfig{
		Provider: ProviderCloudflared,
		Targets: []config.TunnelTarget{
			{Name: "test", Host: "localhost", Port: "8080"},
		},
	}
	if err := manager.ValidateConfig(validConfig); err != nil {
		t.Errorf("expected valid config, got error: %v", err)
	}

	// Invalid provider
	invalidProvider := &config.TunnelConfig{
		Provider: "invalid",
		Targets:  []config.TunnelTarget{{Name: "test", Host: "localhost", Port: "8080"}},
	}
	if err := manager.ValidateConfig(invalidProvider); err == nil {
		t.Error("expected error for invalid provider")
	}

	// Duplicate names
	duplicateNames := &config.TunnelConfig{
		Provider: ProviderCloudflared,
		Targets: []config.TunnelTarget{
			{Name: "test", Host: "localhost", Port: "8080"},
			{Name: "test", Host: "localhost", Port: "9090"},
		},
	}
	if err := manager.ValidateConfig(duplicateNames); err == nil {
		t.Error("expected error for duplicate target names")
	}

	// Missing host
	missingHost := &config.TunnelConfig{
		Provider: ProviderCloudflared,
		Targets:  []config.TunnelTarget{{Name: "test", Port: "8080"}},
	}
	if err := manager.ValidateConfig(missingHost); err == nil {
		t.Error("expected error for missing host")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/infrastructure/tunnel/... -run TestManager -v`
Expected: FAIL - NewManager doesn't exist

**Step 3: Write minimal implementation**

Create `internal/infrastructure/tunnel/manager.go`:

```go
package tunnel

import (
	"context"
	"fmt"
	"sync"

	"github.com/Saturn-Fintech/grund/internal/config"
)

// Manager handles tunnel lifecycle
type Manager struct {
	tunnels  map[string]*Tunnel
	mu       sync.Mutex
}

// NewManager creates a new tunnel manager
func NewManager() *Manager {
	return &Manager{
		tunnels: make(map[string]*Tunnel),
	}
}

// GetProvider returns the appropriate provider for the given name
func (m *Manager) GetProvider(name string) (Provider, error) {
	switch name {
	case ProviderCloudflared:
		return NewCloudflaredProvider(), nil
	case ProviderNgrok:
		return NewNgrokProvider(), nil
	default:
		return nil, fmt.Errorf("unknown tunnel provider: %s (supported: cloudflared, ngrok)", name)
	}
}

// ValidateConfig validates tunnel configuration
func (m *Manager) ValidateConfig(cfg *config.TunnelConfig) error {
	if cfg == nil {
		return nil
	}

	// Validate provider
	if cfg.Provider != ProviderCloudflared && cfg.Provider != ProviderNgrok {
		return fmt.Errorf("invalid tunnel provider: %s (must be 'cloudflared' or 'ngrok')", cfg.Provider)
	}

	// Validate targets
	names := make(map[string]bool)
	for _, target := range cfg.Targets {
		if target.Name == "" {
			return fmt.Errorf("tunnel target missing name")
		}
		if target.Host == "" {
			return fmt.Errorf("tunnel target %s missing host", target.Name)
		}
		if target.Port == "" {
			return fmt.Errorf("tunnel target %s missing port", target.Name)
		}
		if names[target.Name] {
			return fmt.Errorf("duplicate tunnel target name: %s", target.Name)
		}
		names[target.Name] = true
	}

	return nil
}

// StartAll starts tunnels for all targets in the config
func (m *Manager) StartAll(ctx context.Context, cfg *config.TunnelConfig, resolvedTargets []ResolvedTarget) ([]*Tunnel, error) {
	if cfg == nil || len(cfg.Targets) == 0 {
		return nil, nil
	}

	provider, err := m.GetProvider(cfg.Provider)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var tunnels []*Tunnel
	for _, target := range resolvedTargets {
		localAddr := fmt.Sprintf("%s:%s", target.Host, target.Port)
		tunnel, err := provider.Start(ctx, target.Name, localAddr)
		if err != nil {
			// Cleanup any started tunnels
			for _, t := range tunnels {
				provider.Stop(t)
			}
			return nil, fmt.Errorf("failed to start tunnel %s: %w", target.Name, err)
		}
		tunnels = append(tunnels, tunnel)
		m.tunnels[target.Name] = tunnel
	}

	return tunnels, nil
}

// StopAll stops all running tunnels
func (m *Manager) StopAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for name, tunnel := range m.tunnels {
		if tunnel.Process != nil {
			if err := tunnel.Process.Kill(); err != nil {
				lastErr = fmt.Errorf("failed to stop tunnel %s: %w", name, err)
			}
		}
		delete(m.tunnels, name)
	}
	return lastErr
}

// GetTunnels returns all running tunnels
func (m *Manager) GetTunnels() map[string]*Tunnel {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make(map[string]*Tunnel)
	for k, v := range m.tunnels {
		result[k] = v
	}
	return result
}

// ResolvedTarget represents a target with placeholders resolved
type ResolvedTarget struct {
	Name string
	Host string
	Port string
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/infrastructure/tunnel/... -run TestManager -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/infrastructure/tunnel/manager.go internal/infrastructure/tunnel/manager_test.go
git commit -m "feat(tunnel): add Manager for tunnel lifecycle"
```

---

## Task 8: Add Tunnel to Domain Infrastructure

**Files:**
- Modify: `internal/domain/infrastructure/infrastructure.go`

**Step 1: Write the failing test**

Add to `internal/domain/infrastructure/infrastructure_test.go`:

```go
func TestInfrastructureRequirementsTunnel(t *testing.T) {
	reqs := InfrastructureRequirements{
		Tunnel: &TunnelRequirement{
			Provider: "cloudflared",
			Targets: []TunnelTargetRequirement{
				{Name: "localstack", Host: "${localstack.host}", Port: "${localstack.port}"},
			},
		},
	}

	if reqs.Tunnel == nil {
		t.Fatal("expected tunnel requirement")
	}
	if reqs.Tunnel.Provider != "cloudflared" {
		t.Errorf("expected cloudflared, got %s", reqs.Tunnel.Provider)
	}
}

func TestAggregateTunnel(t *testing.T) {
	req1 := InfrastructureRequirements{
		Tunnel: &TunnelRequirement{
			Provider: "cloudflared",
			Targets: []TunnelTargetRequirement{
				{Name: "localstack", Host: "localhost", Port: "4566"},
			},
		},
	}
	req2 := InfrastructureRequirements{
		Tunnel: &TunnelRequirement{
			Provider: "cloudflared",
			Targets: []TunnelTargetRequirement{
				{Name: "api", Host: "localhost", Port: "8080"},
			},
		},
	}

	result := Aggregate(req1, req2)

	if result.Tunnel == nil {
		t.Fatal("expected aggregated tunnel")
	}
	if len(result.Tunnel.Targets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(result.Tunnel.Targets))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/infrastructure/... -run TestInfrastructureRequirementsTunnel -v`
Expected: FAIL - Tunnel field doesn't exist

**Step 3: Write minimal implementation**

Add to `internal/domain/infrastructure/infrastructure.go`:

```go
// TunnelRequirement represents tunnel infrastructure needs
type TunnelRequirement struct {
	Provider string
	Targets  []TunnelTargetRequirement
}

// TunnelTargetRequirement represents a single tunnel target
type TunnelTargetRequirement struct {
	Name string
	Host string
	Port string
}
```

Add `Tunnel` field to `InfrastructureRequirements`:

```go
type InfrastructureRequirements struct {
	Postgres *PostgresRequirement
	MongoDB  *MongoDBRequirement
	Redis    *RedisRequirement
	SQS      *SQSRequirement
	SNS      *SNSRequirement
	S3       *S3Requirement
	Tunnel   *TunnelRequirement // NEW
}
```

Update `Aggregate` function to handle tunnel:

```go
// In the Aggregate function, add tunnel aggregation:
if req.Tunnel != nil {
	if result.Tunnel == nil {
		result.Tunnel = &TunnelRequirement{
			Provider: req.Tunnel.Provider,
			Targets:  make([]TunnelTargetRequirement, 0),
		}
	}
	// Add targets, avoiding duplicates by name
	existingNames := make(map[string]bool)
	for _, t := range result.Tunnel.Targets {
		existingNames[t.Name] = true
	}
	for _, t := range req.Tunnel.Targets {
		if !existingNames[t.Name] {
			result.Tunnel.Targets = append(result.Tunnel.Targets, t)
			existingNames[t.Name] = true
		}
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/domain/infrastructure/... -run TestInfrastructureRequirementsTunnel -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/domain/infrastructure/infrastructure.go internal/domain/infrastructure/infrastructure_test.go
git commit -m "feat(domain): add TunnelRequirement to infrastructure"
```

---

## Task 9: Integrate Tunnel into Up Command

**Files:**
- Modify: `internal/application/commands/up_command.go`
- Modify: `internal/application/ports/ports.go` (if exists, or add interface)

**Step 1: Add TunnelManager port interface**

Add to `internal/application/ports/tunnel.go`:

```go
package ports

import (
	"context"

	"github.com/Saturn-Fintech/grund/internal/config"
)

// TunnelInfo represents a running tunnel
type TunnelInfo struct {
	Name      string
	PublicURL string
	LocalAddr string
}

// TunnelManager manages tunnel lifecycle
type TunnelManager interface {
	ValidateConfig(cfg *config.TunnelConfig) error
	StartAll(ctx context.Context, cfg *config.TunnelConfig, resolvedTargets []ResolvedTunnelTarget) ([]TunnelInfo, error)
	StopAll() error
	GetTunnels() map[string]TunnelInfo
}

// ResolvedTunnelTarget represents a target with placeholders resolved
type ResolvedTunnelTarget struct {
	Name string
	Host string
	Port string
}
```

**Step 2: Update UpCommandHandler**

Modify `internal/application/commands/up_command.go` to include tunnel manager and integrate into lifecycle. Add tunnel manager to handler struct and update Handle method to:

1. After infrastructure starts, resolve tunnel target placeholders
2. Start tunnels
3. Add tunnel URLs to environment context
4. On shutdown, stop tunnels

**Step 3: Run all tests**

Run: `make test`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/application/commands/up_command.go internal/application/ports/tunnel.go
git commit -m "feat(up): integrate tunnel lifecycle into up command"
```

---

## Task 10: Update Documentation

**Files:**
- Modify: `README.md`
- Modify: `docs/wiki/configuration.md`

**Step 1: Add tunnel documentation to README**

Add tunnel section to README.md after S3 section:

```markdown
#### Tunnel (via cloudflared or ngrok)
```bash
grund add tunnel --provider cloudflared --target localstack
```

This exposes LocalStack to the internet, enabling presigned S3 URLs that cloud LLMs can access.
```

**Step 2: Add tunnel configuration reference**

Add to `docs/wiki/configuration.md`:

```markdown
### Tunnel Configuration

Expose local endpoints to the internet:

```yaml
requires:
  infrastructure:
    tunnel:
      provider: cloudflared  # or "ngrok"
      targets:
        - name: localstack
          host: ${localstack.host}
          port: ${localstack.port}
        - name: api
          host: localhost
          port: 8080

env_refs:
  AWS_PUBLIC_ENDPOINT: "${tunnel.localstack.url}"
```

**Environment Placeholders:**

| Placeholder | Description |
|-------------|-------------|
| `${tunnel.<name>.url}` | Public HTTPS URL |
| `${tunnel.<name>.host}` | Hostname only |
```

**Step 3: Commit**

```bash
git add README.md docs/wiki/configuration.md
git commit -m "docs: add tunnel configuration documentation"
```

---

## Task 11: Run Full Test Suite

**Step 1: Run all tests**

Run: `make test`
Expected: All tests PASS

**Step 2: Run linter**

Run: `make lint`
Expected: No errors

**Step 3: Build**

Run: `make build`
Expected: Build succeeds

**Step 4: Final commit (if any fixes needed)**

```bash
git add -A
git commit -m "fix: address test/lint issues"
```

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | Config structs | `schema.go` |
| 2 | TunnelContext | `compose.go` |
| 3 | Placeholder resolution | `env_resolver.go` |
| 4 | Provider interface | `tunnel/provider.go` |
| 5 | Cloudflared provider | `tunnel/cloudflared.go` |
| 6 | ngrok provider | `tunnel/ngrok.go` |
| 7 | Tunnel manager | `tunnel/manager.go` |
| 8 | Domain infrastructure | `infrastructure.go` |
| 9 | Up command integration | `up_command.go` |
| 10 | Documentation | `README.md`, `configuration.md` |
| 11 | Final validation | All tests pass |
