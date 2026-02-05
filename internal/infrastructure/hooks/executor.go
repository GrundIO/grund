package hooks

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
	"github.com/Saturn-Fintech/grund/internal/domain/service"
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
				// Log warning but continue (for now just continue silently)
				continue
			}
			return fmt.Errorf("hook '%s' failed: %w", hook.Name, err)
		}
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
