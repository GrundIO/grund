package queries

import (
	"context"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
	"github.com/Saturn-Fintech/grund/internal/domain/service"
)

// StatusQuery represents a query for service status
type StatusQuery struct {
	ServiceName *string // nil means all services
}

// StatusQueryHandler handles status queries
// This follows the Query pattern (CQRS)
type StatusQueryHandler struct {
	orchestrator ports.ContainerOrchestrator
}

// NewStatusQueryHandler creates a new status query handler
func NewStatusQueryHandler(orchestrator ports.ContainerOrchestrator) *StatusQueryHandler {
	return &StatusQueryHandler{
		orchestrator: orchestrator,
	}
}

// Handle executes the status query
func (h *StatusQueryHandler) Handle(ctx context.Context, query StatusQuery) ([]ports.ServiceStatus, error) {
	if query.ServiceName != nil {
		status, err := h.orchestrator.GetServiceStatus(ctx, service.ServiceName(*query.ServiceName))
		if err != nil {
			return nil, err
		}
		return []ports.ServiceStatus{status}, nil
	}

	// Get all services status
	return h.orchestrator.GetAllServiceStatuses(ctx)
}
