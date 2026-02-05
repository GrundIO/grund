// internal/application/ports/hook_executor.go
package ports

import (
	"context"

	"github.com/Saturn-Fintech/grund/internal/domain/service"
)

// HookExecutionContext provides context for hook execution
type HookExecutionContext struct {
	ServiceName   string            // Name of the service
	ServicePath   string            // Path to service directory (for host execution)
	ContainerName string            // Docker container name (for container execution)
	Environment   map[string]string // Resolved environment variables
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
