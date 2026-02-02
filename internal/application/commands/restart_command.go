package commands

import (
	"context"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
	"github.com/Saturn-Fintech/grund/internal/domain/service"
)

// RestartCommand represents the command to restart services
type RestartCommand struct {
	ServiceNames []string
	Build        bool
}

// RestartCommandHandler handles the restart command
type RestartCommandHandler struct {
	orchestrator ports.ContainerOrchestrator
}

// NewRestartCommandHandler creates a new restart command handler
func NewRestartCommandHandler(orchestrator ports.ContainerOrchestrator) *RestartCommandHandler {
	return &RestartCommandHandler{
		orchestrator: orchestrator,
	}
}

// Handle executes the restart command
func (h *RestartCommandHandler) Handle(ctx context.Context, cmd RestartCommand) error {
	for _, name := range cmd.ServiceNames {
		if err := h.orchestrator.RestartService(ctx, service.ServiceName(name)); err != nil {
			return err
		}
	}
	return nil
}
