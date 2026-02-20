package commands

import (
	"context"
	"fmt"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
	"github.com/Saturn-Fintech/grund/internal/domain/service"
	"github.com/Saturn-Fintech/grund/internal/ui"
)

// DownCommand represents the command to stop all services
type DownCommand struct{}

// DownCommandHandler handles the down command
type DownCommandHandler struct {
	serviceRepo   ports.ServiceRepository
	registryRepo  ports.ServiceRegistryRepository
	orchestrator  ports.ContainerOrchestrator
	hookExecutor  ports.HookExecutor
	tunnelManager ports.TunnelManager
}

// NewDownCommandHandler creates a new down command handler
func NewDownCommandHandler(
	serviceRepo ports.ServiceRepository,
	registryRepo ports.ServiceRegistryRepository,
	orchestrator ports.ContainerOrchestrator,
	hookExecutor ports.HookExecutor,
	tunnelManager ports.TunnelManager,
) *DownCommandHandler {
	return &DownCommandHandler{
		serviceRepo:   serviceRepo,
		registryRepo:  registryRepo,
		orchestrator:  orchestrator,
		hookExecutor:  hookExecutor,
		tunnelManager: tunnelManager,
	}
}

// Handle executes the down command
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

		servicePath, _ := h.registryRepo.GetServicePath(service.ServiceName(svc.Name))
		execCtx := ports.HookExecutionContext{
			ServiceName:   svc.Name,
			ServicePath:   servicePath,
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

	// Stop tunnels (kills processes, removes PID files)
	if err := h.tunnelManager.StopAll(); err != nil {
		ui.Warnf("Failed to stop tunnels: %v", err)
	}

	// Run post_down hooks
	for _, svc := range services {
		hooks := svc.Hooks.GetHooksForStage(service.HookStagePostDown)
		if len(hooks) == 0 {
			continue
		}

		ui.Step("Running post_down hooks for %s...", svc.Name)

		servicePath, _ := h.registryRepo.GetServicePath(service.ServiceName(svc.Name))
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
