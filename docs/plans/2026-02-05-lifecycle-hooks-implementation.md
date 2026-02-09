# Lifecycle Hooks Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add lifecycle hooks (`pre_up`, `post_infrastructure`, `post_up`, `pre_down`, `post_down`) to Grund services, enabling custom script execution at each stage.

**Architecture:** Domain-driven design with hooks as value objects in the service domain. A new `HookExecutor` port/adapter handles execution (host shell or docker exec). Command handlers orchestrate hook invocation at appropriate lifecycle points.

**Tech Stack:** Go, Docker, YAML parsing (gopkg.in/yaml.v3)

---

## Task 1: Add Hook Domain Models

**Files:**
- Create: `internal/domain/service/hook.go`
- Test: `internal/domain/service/hook_test.go`

**Step 1: Write the failing test**

```go
// internal/domain/service/hook_test.go
package service

import (
	"testing"
	"time"
)

func TestHook_Defaults(t *testing.T) {
	hook := Hook{
		Name:    "test hook",
		Command: "echo hello",
		Target:  HookTargetHost,
	}

	if hook.GetTimeout() != 10*time.Minute {
		t.Errorf("expected default timeout 10m, got %v", hook.GetTimeout())
	}
	if hook.ContinueOnError != false {
		t.Errorf("expected default continue_on_error false")
	}
}

func TestHook_CustomTimeout(t *testing.T) {
	hook := Hook{
		Name:    "test hook",
		Command: "echo hello",
		Target:  HookTargetHost,
		Timeout: 30 * time.Second,
	}

	if hook.GetTimeout() != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", hook.GetTimeout())
	}
}

func TestHookTarget_Validation(t *testing.T) {
	if !HookTargetHost.IsValid() {
		t.Error("host should be valid")
	}
	if !HookTargetContainer.IsValid() {
		t.Error("container should be valid")
	}
	if HookTarget("invalid").IsValid() {
		t.Error("invalid should not be valid")
	}
}

func TestHookStage_AllowsTarget(t *testing.T) {
	tests := []struct {
		stage         HookStage
		target        HookTarget
		shouldBeValid bool
	}{
		{HookStagePreUp, HookTargetHost, true},
		{HookStagePreUp, HookTargetContainer, false},
		{HookStagePostInfrastructure, HookTargetHost, true},
		{HookStagePostInfrastructure, HookTargetContainer, false},
		{HookStagePostUp, HookTargetHost, true},
		{HookStagePostUp, HookTargetContainer, true},
		{HookStagePreDown, HookTargetHost, true},
		{HookStagePreDown, HookTargetContainer, true},
		{HookStagePostDown, HookTargetHost, true},
		{HookStagePostDown, HookTargetContainer, false},
	}

	for _, tt := range tests {
		result := tt.stage.AllowsTarget(tt.target)
		if result != tt.shouldBeValid {
			t.Errorf("stage %s with target %s: expected %v, got %v",
				tt.stage, tt.target, tt.shouldBeValid, result)
		}
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/domain/service/... -run TestHook -v
```

Expected: FAIL - types not defined

**Step 3: Write minimal implementation**

```go
// internal/domain/service/hook.go
package service

import "time"

// HookStage represents when a hook executes in the service lifecycle
type HookStage string

const (
	HookStagePreUp              HookStage = "pre_up"
	HookStagePostInfrastructure HookStage = "post_infrastructure"
	HookStagePostUp             HookStage = "post_up"
	HookStagePreDown            HookStage = "pre_down"
	HookStagePostDown           HookStage = "post_down"
)

// AllowsTarget returns true if the stage allows the given execution target
func (s HookStage) AllowsTarget(target HookTarget) bool {
	switch s {
	case HookStagePreUp, HookStagePostInfrastructure, HookStagePostDown:
		// These stages only allow host execution (no containers available)
		return target == HookTargetHost
	case HookStagePostUp, HookStagePreDown:
		// These stages allow both host and container execution
		return target == HookTargetHost || target == HookTargetContainer
	default:
		return false
	}
}

// HookTarget represents where a hook executes
type HookTarget string

const (
	HookTargetHost      HookTarget = "host"
	HookTargetContainer HookTarget = "container"
)

// IsValid returns true if the target is a valid hook target
func (t HookTarget) IsValid() bool {
	return t == HookTargetHost || t == HookTargetContainer
}

// Hook represents a lifecycle hook configuration
type Hook struct {
	Name            string
	Command         string
	Target          HookTarget
	Timeout         time.Duration // 0 means use default (10m)
	ContinueOnError bool
}

// GetTimeout returns the timeout, using default if not set
func (h Hook) GetTimeout() time.Duration {
	if h.Timeout == 0 {
		return 10 * time.Minute
	}
	return h.Timeout
}

// Hooks contains all lifecycle hooks for a service
type Hooks struct {
	PreUp              []Hook
	PostInfrastructure []Hook
	PostUp             []Hook
	PreDown            []Hook
	PostDown           []Hook
}

// GetHooksForStage returns the hooks for a given stage
func (h Hooks) GetHooksForStage(stage HookStage) []Hook {
	switch stage {
	case HookStagePreUp:
		return h.PreUp
	case HookStagePostInfrastructure:
		return h.PostInfrastructure
	case HookStagePostUp:
		return h.PostUp
	case HookStagePreDown:
		return h.PreDown
	case HookStagePostDown:
		return h.PostDown
	default:
		return nil
	}
}

// HasHooks returns true if any hooks are defined
func (h Hooks) HasHooks() bool {
	return len(h.PreUp) > 0 ||
		len(h.PostInfrastructure) > 0 ||
		len(h.PostUp) > 0 ||
		len(h.PreDown) > 0 ||
		len(h.PostDown) > 0
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/domain/service/... -run TestHook -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/domain/service/hook.go internal/domain/service/hook_test.go
git commit -m "feat(domain): add Hook value objects for lifecycle hooks"
```

---

## Task 2: Add Hooks Field to Service Struct

**Files:**
- Modify: `internal/domain/service/service.go:11-20`

**Step 1: Add Hooks field to Service struct**

In `internal/domain/service/service.go`, add the `Hooks` field to the `Service` struct:

```go
// Service represents a service in the domain
type Service struct {
	Name         string
	Type         ServiceType
	Port         Port
	Build        *BuildConfig
	Run          *RunConfig
	Health       HealthConfig
	Dependencies ServiceDependencies
	Environment  Environment
	Hooks        Hooks  // NEW: lifecycle hooks
}
```

**Step 2: Run existing tests to ensure no regression**

```bash
go test ./internal/domain/service/... -v
```

Expected: PASS (all existing tests should still pass)

**Step 3: Commit**

```bash
git add internal/domain/service/service.go
git commit -m "feat(domain): add Hooks field to Service struct"
```

---

## Task 3: Add Hook DTOs for YAML Parsing

**Files:**
- Modify: `internal/infrastructure/config/service_repository.go`

**Step 1: Add HookDTO types after line 206**

Add these DTOs after the existing DTO definitions (around line 206):

```go
// HooksDTO is the DTO for hooks YAML serialization
type HooksDTO struct {
	PreUp              []HookDTO `yaml:"pre_up,omitempty"`
	PostInfrastructure []HookDTO `yaml:"post_infrastructure,omitempty"`
	PostUp             []HookDTO `yaml:"post_up,omitempty"`
	PreDown            []HookDTO `yaml:"pre_down,omitempty"`
	PostDown           []HookDTO `yaml:"post_down,omitempty"`
}

// HookDTO is the DTO for individual hook YAML serialization
type HookDTO struct {
	Name            string `yaml:"name"`
	Command         string `yaml:"command"`
	Target          string `yaml:"target"`                      // "host" or "container"
	Timeout         string `yaml:"timeout,omitempty"`           // e.g., "30s", "5m"
	ContinueOnError bool   `yaml:"continue_on_error,omitempty"` // default false
}
```

**Step 2: Add Hooks field to ServiceInfoDTO (around line 110)**

```go
type ServiceInfoDTO struct {
	Name   string          `yaml:"name"`
	Type   string          `yaml:"type"`
	Port   int             `yaml:"port"`
	Build  *BuildConfigDTO `yaml:"build,omitempty"`
	Run    *RunConfigDTO   `yaml:"run,omitempty"`
	Health HealthConfigDTO `yaml:"health"`
	Hooks  *HooksDTO       `yaml:"hooks,omitempty"` // NEW
}
```

**Step 3: Commit**

```bash
git add internal/infrastructure/config/service_repository.go
git commit -m "feat(infrastructure): add Hook DTOs for YAML parsing"
```

---

## Task 4: Parse Hooks in Service Repository

**Files:**
- Modify: `internal/infrastructure/config/service_repository.go`
- Test: `internal/infrastructure/config/service_repository_test.go`

**Step 1: Write failing test**

Add to `internal/infrastructure/config/service_repository_test.go`:

```go
func TestServiceRepository_ParsesHooks(t *testing.T) {
	// Create temp directory with test service
	tmpDir := t.TempDir()

	// Create services.yaml
	servicesYAML := `services:
  test-service:
    path: ./test-service
`
	err := os.WriteFile(filepath.Join(tmpDir, "services.yaml"), []byte(servicesYAML), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create service directory
	serviceDir := filepath.Join(tmpDir, "test-service")
	err = os.MkdirAll(serviceDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create grund.yaml with hooks
	grundYAML := `version: "1"
service:
  name: test-service
  type: go
  port: 8080
  health:
    endpoint: /health
    interval: 5s
    timeout: 3s
    retries: 3
  hooks:
    pre_up:
      - name: "Generate config"
        command: "./scripts/gen.sh"
        target: host
    post_infrastructure:
      - name: "Run migrations"
        command: "psql -f migrations.sql"
        target: host
        timeout: 5m
    post_up:
      - name: "Create admin"
        command: "bin/create-admin"
        target: container
        continue_on_error: true
requires:
  services: []
  infrastructure: {}
env: {}
`
	err = os.WriteFile(filepath.Join(serviceDir, "grund.yaml"), []byte(grundYAML), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create repository and load service
	registryRepo, err := NewServiceRegistryRepository(filepath.Join(tmpDir, "services.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	repo := NewServiceRepository(registryRepo)
	svc, err := repo.FindByName(service.ServiceName("test-service"))
	if err != nil {
		t.Fatal(err)
	}

	// Verify hooks were parsed
	if len(svc.Hooks.PreUp) != 1 {
		t.Errorf("expected 1 pre_up hook, got %d", len(svc.Hooks.PreUp))
	}
	if svc.Hooks.PreUp[0].Name != "Generate config" {
		t.Errorf("expected hook name 'Generate config', got '%s'", svc.Hooks.PreUp[0].Name)
	}
	if svc.Hooks.PreUp[0].Target != service.HookTargetHost {
		t.Errorf("expected target host, got %s", svc.Hooks.PreUp[0].Target)
	}

	if len(svc.Hooks.PostInfrastructure) != 1 {
		t.Errorf("expected 1 post_infrastructure hook, got %d", len(svc.Hooks.PostInfrastructure))
	}
	if svc.Hooks.PostInfrastructure[0].Timeout != 5*time.Minute {
		t.Errorf("expected timeout 5m, got %v", svc.Hooks.PostInfrastructure[0].Timeout)
	}

	if len(svc.Hooks.PostUp) != 1 {
		t.Errorf("expected 1 post_up hook, got %d", len(svc.Hooks.PostUp))
	}
	if svc.Hooks.PostUp[0].Target != service.HookTargetContainer {
		t.Errorf("expected target container, got %s", svc.Hooks.PostUp[0].Target)
	}
	if !svc.Hooks.PostUp[0].ContinueOnError {
		t.Error("expected continue_on_error true")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/infrastructure/config/... -run TestServiceRepository_ParsesHooks -v
```

Expected: FAIL - hooks not being parsed

**Step 3: Add hook parsing to toDomainService function**

In the `toDomainService` function (around line 275), add hooks parsing before returning the service:

```go
	// Parse hooks
	hooks := r.toHooks(dto.Service.Hooks)

	svc := &service.Service{
		Name:         dto.Service.Name,
		Type:         service.ServiceType(dto.Service.Type),
		Port:         port,
		Build:        build,
		Run:          run,
		Health:       health,
		Dependencies: deps,
		Environment:  env,
		Hooks:        hooks,  // NEW
	}

	return svc, svc.Validate()
```

**Step 4: Add toHooks helper function**

Add this function after `toInfrastructureRequirements`:

```go
// toHooks converts HooksDTO to domain Hooks
func (r *ServiceRepositoryImpl) toHooks(dto *HooksDTO) service.Hooks {
	if dto == nil {
		return service.Hooks{}
	}

	return service.Hooks{
		PreUp:              r.toHookSlice(dto.PreUp),
		PostInfrastructure: r.toHookSlice(dto.PostInfrastructure),
		PostUp:             r.toHookSlice(dto.PostUp),
		PreDown:            r.toHookSlice(dto.PreDown),
		PostDown:           r.toHookSlice(dto.PostDown),
	}
}

// toHookSlice converts a slice of HookDTO to domain Hook slice
func (r *ServiceRepositoryImpl) toHookSlice(dtos []HookDTO) []service.Hook {
	if len(dtos) == 0 {
		return nil
	}

	hooks := make([]service.Hook, len(dtos))
	for i, dto := range dtos {
		var timeout time.Duration
		if dto.Timeout != "" {
			timeout, _ = time.ParseDuration(dto.Timeout)
		}

		hooks[i] = service.Hook{
			Name:            dto.Name,
			Command:         dto.Command,
			Target:          service.HookTarget(dto.Target),
			Timeout:         timeout,
			ContinueOnError: dto.ContinueOnError,
		}
	}
	return hooks
}
```

**Step 5: Run test to verify it passes**

```bash
go test ./internal/infrastructure/config/... -run TestServiceRepository_ParsesHooks -v
```

Expected: PASS

**Step 6: Commit**

```bash
git add internal/infrastructure/config/service_repository.go internal/infrastructure/config/service_repository_test.go
git commit -m "feat(infrastructure): parse hooks from grund.yaml"
```

---

## Task 5: Create HookExecutor Port (Interface)

**Files:**
- Create: `internal/application/ports/hook_executor.go`

**Step 1: Create the port interface**

```go
// internal/application/ports/hook_executor.go
package ports

import (
	"context"

	"github.com/Saturn-Fintech/grund/internal/domain/service"
)

// HookExecutionContext provides context for hook execution
type HookExecutionContext struct {
	ServiceName    string            // Name of the service
	ServicePath    string            // Path to service directory (for host execution)
	ContainerName  string            // Docker container name (for container execution)
	Environment    map[string]string // Resolved environment variables
}

// HookExecutor defines the interface for executing lifecycle hooks
// This follows the Dependency Inversion Principle
type HookExecutor interface {
	// Execute runs a single hook and returns an error if it fails
	Execute(ctx context.Context, hook service.Hook, execCtx HookExecutionContext) error

	// ExecuteAll runs all hooks for a stage, respecting continue_on_error
	// Returns the first error encountered (if continue_on_error is false)
	ExecuteAll(ctx context.Context, hooks []service.Hook, execCtx HookExecutionContext) error
}
```

**Step 2: Commit**

```bash
git add internal/application/ports/hook_executor.go
git commit -m "feat(ports): add HookExecutor interface"
```

---

## Task 6: Implement HookExecutor Adapter

**Files:**
- Create: `internal/infrastructure/hooks/executor.go`
- Test: `internal/infrastructure/hooks/executor_test.go`

**Step 1: Write failing test**

```go
// internal/infrastructure/hooks/executor_test.go
package hooks

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
	"github.com/Saturn-Fintech/grund/internal/domain/service"
)

func TestHookExecutor_ExecuteHostCommand(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test script
	scriptPath := filepath.Join(tmpDir, "test.sh")
	err := os.WriteFile(scriptPath, []byte("#!/bin/sh\necho \"hello from hook\""), 0755)
	if err != nil {
		t.Fatal(err)
	}

	executor := NewHookExecutor()
	hook := service.Hook{
		Name:    "test hook",
		Command: "./test.sh",
		Target:  service.HookTargetHost,
		Timeout: 10 * time.Second,
	}

	execCtx := ports.HookExecutionContext{
		ServiceName: "test-service",
		ServicePath: tmpDir,
		Environment: map[string]string{"TEST_VAR": "test_value"},
	}

	err = executor.Execute(context.Background(), hook, execCtx)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestHookExecutor_ExecuteHostCommand_Timeout(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a script that sleeps
	scriptPath := filepath.Join(tmpDir, "slow.sh")
	err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nsleep 10"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	executor := NewHookExecutor()
	hook := service.Hook{
		Name:    "slow hook",
		Command: "./slow.sh",
		Target:  service.HookTargetHost,
		Timeout: 100 * time.Millisecond,
	}

	execCtx := ports.HookExecutionContext{
		ServiceName: "test-service",
		ServicePath: tmpDir,
	}

	err = executor.Execute(context.Background(), hook, execCtx)
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestHookExecutor_ExecuteAll_ContinueOnError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create scripts
	err := os.WriteFile(filepath.Join(tmpDir, "fail.sh"), []byte("#!/bin/sh\nexit 1"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tmpDir, "success.sh"), []byte("#!/bin/sh\necho ok"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	executor := NewHookExecutor()

	// Test with continue_on_error = false (default)
	hooks := []service.Hook{
		{Name: "fail", Command: "./fail.sh", Target: service.HookTargetHost},
		{Name: "success", Command: "./success.sh", Target: service.HookTargetHost},
	}

	execCtx := ports.HookExecutionContext{
		ServiceName: "test-service",
		ServicePath: tmpDir,
	}

	err = executor.ExecuteAll(context.Background(), hooks, execCtx)
	if err == nil {
		t.Error("expected error when continue_on_error is false")
	}

	// Test with continue_on_error = true
	hooks[0].ContinueOnError = true
	err = executor.ExecuteAll(context.Background(), hooks, execCtx)
	if err != nil {
		t.Errorf("expected no error when continue_on_error is true, got: %v", err)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/infrastructure/hooks/... -v
```

Expected: FAIL - package doesn't exist

**Step 3: Implement HookExecutor**

```go
// internal/infrastructure/hooks/executor.go
package hooks

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
	"github.com/Saturn-Fintech/grund/internal/domain/service"
	"github.com/Saturn-Fintech/grund/internal/ui"
)

// HookExecutor implements the ports.HookExecutor interface
type HookExecutor struct{}

// NewHookExecutor creates a new hook executor
func NewHookExecutor() *HookExecutor {
	return &HookExecutor{}
}

// Execute runs a single hook
func (e *HookExecutor) Execute(ctx context.Context, hook service.Hook, execCtx ports.HookExecutionContext) error {
	// Create context with timeout
	timeout := hook.GetTimeout()
	execContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ui.Debug("Executing hook '%s' (%s on %s)", hook.Name, hook.Command, hook.Target)

	switch hook.Target {
	case service.HookTargetHost:
		return e.executeHost(execContext, hook, execCtx)
	case service.HookTargetContainer:
		return e.executeContainer(execContext, hook, execCtx)
	default:
		return fmt.Errorf("invalid hook target: %s", hook.Target)
	}
}

// ExecuteAll runs all hooks, respecting continue_on_error
func (e *HookExecutor) ExecuteAll(ctx context.Context, hooks []service.Hook, execCtx ports.HookExecutionContext) error {
	for _, hook := range hooks {
		err := e.Execute(ctx, hook, execCtx)
		if err != nil {
			if hook.ContinueOnError {
				ui.Warnf("Hook '%s' failed (continuing): %v", hook.Name, err)
				continue
			}
			return fmt.Errorf("hook '%s' failed: %w", hook.Name, err)
		}
		ui.Successf("Hook '%s' completed", hook.Name)
	}
	return nil
}

func (e *HookExecutor) executeHost(ctx context.Context, hook service.Hook, execCtx ports.HookExecutionContext) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", hook.Command)
	cmd.Dir = execCtx.ServicePath

	// Build environment
	cmd.Env = os.Environ()
	for k, v := range execCtx.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Stream output to terminal
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("hook timed out after %v", hook.GetTimeout())
	}
	return err
}

func (e *HookExecutor) executeContainer(ctx context.Context, hook service.Hook, execCtx ports.HookExecutionContext) error {
	if execCtx.ContainerName == "" {
		return fmt.Errorf("container name required for container hook execution")
	}

	cmd := exec.CommandContext(ctx, "docker", "exec", execCtx.ContainerName, "sh", "-c", hook.Command)

	// Stream output to terminal
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("hook timed out after %v", hook.GetTimeout())
	}
	return err
}

// Ensure HookExecutor implements the interface
var _ ports.HookExecutor = (*HookExecutor)(nil)
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/infrastructure/hooks/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/infrastructure/hooks/executor.go internal/infrastructure/hooks/executor_test.go
git commit -m "feat(infrastructure): implement HookExecutor adapter"
```

---

## Task 7: Wire HookExecutor into Container

**Files:**
- Modify: `internal/application/wiring/wiring.go`

**Step 1: Add HookExecutor to Container struct (around line 35)**

```go
// Add to Container struct
HookExecutor     interface{} // ports.HookExecutor
```

**Step 2: Initialize HookExecutor in NewContainerWithConfig (around line 112)**

Add after tunnel manager initialization:

```go
	// Initialize hook executor
	hookExecutor := hooks.NewHookExecutor()
```

**Step 3: Add import for hooks package**

```go
import (
	// ... existing imports
	"github.com/Saturn-Fintech/grund/internal/infrastructure/hooks"
)
```

**Step 4: Pass hookExecutor to UpCommandHandler (around line 115)**

Update the `NewUpCommandHandler` call to include the hook executor:

```go
	upHandler := commands.NewUpCommandHandler(
		serviceRepo,
		registryRepo,
		orchestrator,
		provisioner,
		composeGenerator,
		healthChecker,
		tunnelManager,
		hookExecutor,  // NEW
	)
```

**Step 5: Add HookExecutor to Container initialization (around line 144)**

```go
	return &Container{
		// ... existing fields
		HookExecutor:          hookExecutor,
		// ... rest of fields
	}, nil
```

**Step 6: Update DownCommandHandler to receive hookExecutor**

```go
	downHandler := commands.NewDownCommandHandler(orchestrator, serviceRepo, hookExecutor)
```

**Step 7: Commit**

```bash
git add internal/application/wiring/wiring.go
git commit -m "feat(wiring): wire HookExecutor into dependency container"
```

---

## Task 8: Add Hooks to UpCommandHandler

**Files:**
- Modify: `internal/application/commands/up_command.go`
- Test: `internal/application/commands/up_command_test.go`

**Step 1: Update UpCommandHandler struct to include hookExecutor**

```go
type UpCommandHandler struct {
	serviceRepo      ports.ServiceRepository
	registryRepo     ports.ServiceRegistryRepository
	orchestrator     ports.ContainerOrchestrator
	provisioner      ports.InfrastructureProvisioner
	composeGenerator ports.ComposeGenerator
	healthChecker    ports.HealthChecker
	tunnelManager    ports.TunnelManager
	hookExecutor     ports.HookExecutor  // NEW
}
```

**Step 2: Update NewUpCommandHandler**

```go
func NewUpCommandHandler(
	serviceRepo ports.ServiceRepository,
	registryRepo ports.ServiceRegistryRepository,
	orchestrator ports.ContainerOrchestrator,
	provisioner ports.InfrastructureProvisioner,
	composeGenerator ports.ComposeGenerator,
	healthChecker ports.HealthChecker,
	tunnelManager ports.TunnelManager,
	hookExecutor ports.HookExecutor,  // NEW
) *UpCommandHandler {
	return &UpCommandHandler{
		serviceRepo:      serviceRepo,
		registryRepo:     registryRepo,
		orchestrator:     orchestrator,
		provisioner:      provisioner,
		composeGenerator: composeGenerator,
		healthChecker:    healthChecker,
		tunnelManager:    tunnelManager,
		hookExecutor:     hookExecutor,  // NEW
	}
}
```

**Step 3: Add helper method for running hooks at a stage**

```go
// runHooks executes hooks for all services at the given stage
func (h *UpCommandHandler) runHooks(ctx context.Context, services []*service.Service, stage service.HookStage, awsResources *ports.ProvisionedAWSResources) error {
	for _, svc := range services {
		hooks := svc.Hooks.GetHooksForStage(stage)
		if len(hooks) == 0 {
			continue
		}

		ui.Step("Running %s hooks for %s...", stage, svc.Name)

		// Get service path for host execution
		servicePath, err := h.registryRepo.GetServicePath(service.ServiceName(svc.Name))
		if err != nil {
			return fmt.Errorf("failed to get service path: %w", err)
		}

		// Build execution context
		execCtx := ports.HookExecutionContext{
			ServiceName:   svc.Name,
			ServicePath:   servicePath,
			ContainerName: fmt.Sprintf("grund-%s-1", svc.Name), // Docker compose naming convention
			Environment:   svc.Environment.Variables,
		}

		if err := h.hookExecutor.ExecuteAll(ctx, hooks, execCtx); err != nil {
			return fmt.Errorf("%s hooks failed for %s: %w", stage, svc.Name, err)
		}
	}
	return nil
}
```

**Step 4: Add hook execution points in Handle method**

Update the `Handle` method to call hooks at appropriate stages. After loading services (around line 73):

```go
	// 2.5 Run pre_up hooks (before anything starts)
	if err := h.runHooks(ctx, services, service.HookStagePreUp, nil); err != nil {
		return err
	}
```

After provisioning infrastructure (around line 128):

```go
	// 8.5 Run post_infrastructure hooks (after DBs/queues ready)
	if err := h.runHooks(ctx, services, service.HookStagePostInfrastructure, awsResources); err != nil {
		return err
	}
```

After starting services (around line 146):

```go
		// 9.5 Run post_up hooks (after services healthy)
		if err := h.runHooks(ctx, services, service.HookStagePostUp, awsResources); err != nil {
			return err
		}
```

**Step 5: Run all tests**

```bash
go test ./internal/application/commands/... -v
```

Expected: PASS (may need to update test mocks)

**Step 6: Commit**

```bash
git add internal/application/commands/up_command.go
git commit -m "feat(commands): execute lifecycle hooks in UpCommandHandler"
```

---

## Task 9: Add Hooks to DownCommandHandler

**Files:**
- Modify: `internal/application/commands/down_command.go`

**Step 1: Update DownCommandHandler struct**

```go
type DownCommandHandler struct {
	orchestrator ports.ContainerOrchestrator
	serviceRepo  ports.ServiceRepository
	hookExecutor ports.HookExecutor
}
```

**Step 2: Update NewDownCommandHandler**

```go
func NewDownCommandHandler(
	orchestrator ports.ContainerOrchestrator,
	serviceRepo ports.ServiceRepository,
	hookExecutor ports.HookExecutor,
) *DownCommandHandler {
	return &DownCommandHandler{
		orchestrator: orchestrator,
		serviceRepo:  serviceRepo,
		hookExecutor: hookExecutor,
	}
}
```

**Step 3: Update Handle method to run pre_down and post_down hooks**

```go
func (h *DownCommandHandler) Handle(ctx context.Context, cmd DownCommand) error {
	// Load all services to get their hooks
	services, err := h.serviceRepo.FindAll()
	if err != nil {
		// If we can't load services, just stop containers
		ui.Warnf("Could not load services for hooks: %v", err)
		return h.orchestrator.StopServices(ctx)
	}

	// Run pre_down hooks
	for _, svc := range services {
		hooks := svc.Hooks.GetHooksForStage(service.HookStagePreDown)
		if len(hooks) == 0 {
			continue
		}

		ui.Step("Running pre_down hooks for %s...", svc.Name)
		execCtx := ports.HookExecutionContext{
			ServiceName:   svc.Name,
			ContainerName: fmt.Sprintf("grund-%s-1", svc.Name),
			Environment:   svc.Environment.Variables,
		}

		if err := h.hookExecutor.ExecuteAll(ctx, hooks, execCtx); err != nil {
			ui.Warnf("pre_down hooks failed for %s: %v", svc.Name, err)
			// Continue with shutdown even if hooks fail
		}
	}

	// Stop containers
	if err := h.orchestrator.StopServices(ctx); err != nil {
		return err
	}

	// Run post_down hooks
	for _, svc := range services {
		hooks := svc.Hooks.GetHooksForStage(service.HookStagePostDown)
		if len(hooks) == 0 {
			continue
		}

		ui.Step("Running post_down hooks for %s...", svc.Name)

		// For post_down, we need to get service path since containers are stopped
		servicePath := "" // We'd need registryRepo to get this
		execCtx := ports.HookExecutionContext{
			ServiceName: svc.Name,
			ServicePath: servicePath,
			Environment: svc.Environment.Variables,
		}

		if err := h.hookExecutor.ExecuteAll(ctx, hooks, execCtx); err != nil {
			ui.Warnf("post_down hooks failed for %s: %v", svc.Name, err)
		}
	}

	return nil
}
```

**Step 4: Add registryRepo to DownCommandHandler for service path resolution**

```go
type DownCommandHandler struct {
	orchestrator ports.ContainerOrchestrator
	serviceRepo  ports.ServiceRepository
	registryRepo ports.ServiceRegistryRepository
	hookExecutor ports.HookExecutor
}

func NewDownCommandHandler(
	orchestrator ports.ContainerOrchestrator,
	serviceRepo ports.ServiceRepository,
	registryRepo ports.ServiceRegistryRepository,
	hookExecutor ports.HookExecutor,
) *DownCommandHandler {
	return &DownCommandHandler{
		orchestrator: orchestrator,
		serviceRepo:  serviceRepo,
		registryRepo: registryRepo,
		hookExecutor: hookExecutor,
	}
}
```

**Step 5: Update wiring.go to pass registryRepo to DownCommandHandler**

```go
downHandler := commands.NewDownCommandHandler(orchestrator, serviceRepo, registryRepo, hookExecutor)
```

**Step 6: Run tests**

```bash
go test ./internal/application/commands/... -v
```

**Step 7: Commit**

```bash
git add internal/application/commands/down_command.go internal/application/wiring/wiring.go
git commit -m "feat(commands): execute pre_down and post_down hooks in DownCommandHandler"
```

---

## Task 10: Update Existing Tests

**Files:**
- Modify: `internal/application/commands/up_command_test.go`
- Modify: `internal/application/commands/down_command_test.go`

**Step 1: Create mock HookExecutor**

Add to test files or a shared test helpers location:

```go
type mockHookExecutor struct{}

func (m *mockHookExecutor) Execute(ctx context.Context, hook service.Hook, execCtx ports.HookExecutionContext) error {
	return nil
}

func (m *mockHookExecutor) ExecuteAll(ctx context.Context, hooks []service.Hook, execCtx ports.HookExecutionContext) error {
	return nil
}
```

**Step 2: Update test cases to include hookExecutor in handler construction**

For each test that creates a handler, add the mock hook executor:

```go
handler := commands.NewUpCommandHandler(
	mockServiceRepo,
	mockRegistryRepo,
	mockOrchestrator,
	mockProvisioner,
	mockGenerator,
	mockHealthChecker,
	mockTunnelManager,
	&mockHookExecutor{},  // NEW
)
```

**Step 3: Run all tests**

```bash
make test
```

**Step 4: Commit**

```bash
git add internal/application/commands/up_command_test.go internal/application/commands/down_command_test.go
git commit -m "test: update command tests with mock HookExecutor"
```

---

## Task 11: Add E2E Test for Hooks

**Files:**
- Create: `test/e2e/hooks_test.go`

**Step 1: Write E2E test**

```go
// test/e2e/hooks_test.go
package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHooksE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	// Create temp directory with test service
	tmpDir := t.TempDir()

	// Create services.yaml
	servicesYAML := `services:
  hook-test-service:
    path: ./hook-test-service
`
	err := os.WriteFile(filepath.Join(tmpDir, "services.yaml"), []byte(servicesYAML), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create service directory
	serviceDir := filepath.Join(tmpDir, "hook-test-service")
	err = os.MkdirAll(serviceDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create a marker file script
	scriptContent := `#!/bin/sh
echo "hook executed at $(date)" >> /tmp/grund-hook-test.log
`
	err = os.WriteFile(filepath.Join(serviceDir, "hook.sh"), []byte(scriptContent), 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create grund.yaml with hooks
	grundYAML := `version: "1"
service:
  name: hook-test-service
  type: go
  port: 8080
  build:
    dockerfile: Dockerfile
    context: .
  health:
    endpoint: /health
    interval: 5s
    timeout: 3s
    retries: 3
  hooks:
    pre_up:
      - name: "Pre-up hook"
        command: "./hook.sh"
        target: host
requires:
  services: []
  infrastructure: {}
env: {}
`
	err = os.WriteFile(filepath.Join(serviceDir, "grund.yaml"), []byte(grundYAML), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Clean up log file
	os.Remove("/tmp/grund-hook-test.log")
	defer os.Remove("/tmp/grund-hook-test.log")

	// Run grund config (just to validate parsing works)
	cmd := exec.Command("grund", "config", "hook-test-service")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("grund config failed: %v\nOutput: %s", err, output)
	}

	// Verify hooks are shown in config output
	if !strings.Contains(string(output), "hooks") || !strings.Contains(string(output), "Pre-up hook") {
		t.Errorf("expected hooks in config output, got: %s", output)
	}
}
```

**Step 2: Run E2E test**

```bash
go test ./test/e2e/... -run TestHooksE2E -v
```

**Step 3: Commit**

```bash
git add test/e2e/hooks_test.go
git commit -m "test: add E2E test for lifecycle hooks"
```

---

## Task 12: Update Documentation

**Files:**
- Modify: `docs/wiki/configuration.md`

**Step 1: Add hooks documentation section**

Add a new section for hooks in the configuration documentation:

```markdown
## Lifecycle Hooks

Hooks allow you to run custom scripts at various stages of service startup and shutdown.

### Configuration

```yaml
service:
  name: my-service
  hooks:
    pre_up:
      - name: "Generate config"
        command: "./scripts/generate-config.sh"
        target: host
        timeout: 30s
        continue_on_error: false

    post_infrastructure:
      - name: "Run migrations"
        command: "psql -h localhost -f ./migrations/up.sql"
        target: host

    post_up:
      - name: "Create admin user"
        command: "bin/create-admin"
        target: container

    pre_down:
      - name: "Drain connections"
        command: "bin/drain"
        target: container

    post_down:
      - name: "Cleanup temp files"
        command: "rm -rf ./tmp/cache"
        target: host
```

### Hook Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `name` | Yes | - | Human-readable hook name |
| `command` | Yes | - | Command to execute |
| `target` | Yes | - | `host` or `container` |
| `timeout` | No | `10m` | Maximum execution time |
| `continue_on_error` | No | `false` | Continue if hook fails |

### Lifecycle Stages

| Stage | When | Allowed Targets |
|-------|------|-----------------|
| `pre_up` | Before anything starts | `host` only |
| `post_infrastructure` | After DBs/queues ready | `host` only |
| `post_up` | After services healthy | `host`, `container` |
| `pre_down` | Before stopping | `host`, `container` |
| `post_down` | After containers stopped | `host` only |

### Environment Variables

Hooks have access to the same environment variables as your service:

```bash
$DATABASE_URL
$REDIS_URL
$SQS_QUEUE_URL
```
```

**Step 2: Commit**

```bash
git add docs/wiki/configuration.md
git commit -m "docs: add lifecycle hooks documentation"
```

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | Add Hook domain models | `internal/domain/service/hook.go` |
| 2 | Add Hooks to Service struct | `internal/domain/service/service.go` |
| 3 | Add Hook DTOs | `internal/infrastructure/config/service_repository.go` |
| 4 | Parse hooks from YAML | `internal/infrastructure/config/service_repository.go` |
| 5 | Create HookExecutor port | `internal/application/ports/hook_executor.go` |
| 6 | Implement HookExecutor | `internal/infrastructure/hooks/executor.go` |
| 7 | Wire into container | `internal/application/wiring/wiring.go` |
| 8 | Add to UpCommandHandler | `internal/application/commands/up_command.go` |
| 9 | Add to DownCommandHandler | `internal/application/commands/down_command.go` |
| 10 | Update existing tests | `*_test.go` files |
| 11 | Add E2E test | `test/e2e/hooks_test.go` |
| 12 | Update documentation | `docs/wiki/configuration.md` |
