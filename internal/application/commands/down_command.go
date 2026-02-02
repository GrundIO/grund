package commands

import (
	"context"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
)

// DownCommand represents the command to stop all services
type DownCommand struct{}

// DownCommandHandler handles the down command
type DownCommandHandler struct {
	orchestrator ports.ContainerOrchestrator
}

// NewDownCommandHandler creates a new down command handler
func NewDownCommandHandler(orchestrator ports.ContainerOrchestrator) *DownCommandHandler {
	return &DownCommandHandler{
		orchestrator: orchestrator,
	}
}

// Handle executes the down command
func (h *DownCommandHandler) Handle(ctx context.Context, cmd DownCommand) error {
	return h.orchestrator.StopServices(ctx)
}
