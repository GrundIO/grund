package queries

import (
	"context"
	"fmt"
	"testing"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
	"github.com/Saturn-Fintech/grund/internal/domain/service"
)

type mockStatusOrchestrator struct {
	statusFunc func(name service.ServiceName) (ports.ServiceStatus, error)
}

func (m *mockStatusOrchestrator) StartInfrastructure(ctx context.Context) error {
	return nil
}

func (m *mockStatusOrchestrator) StartServices(ctx context.Context, services []service.ServiceName) error {
	return nil
}

func (m *mockStatusOrchestrator) StopServices(ctx context.Context) error {
	return nil
}

func (m *mockStatusOrchestrator) RestartService(ctx context.Context, name service.ServiceName) error {
	return nil
}

func (m *mockStatusOrchestrator) GetServiceStatus(ctx context.Context, name service.ServiceName) (ports.ServiceStatus, error) {
	if m.statusFunc != nil {
		return m.statusFunc(name)
	}
	return ports.ServiceStatus{
		Name:   name.String(),
		Status: "running",
		Health: "healthy",
	}, nil
}

func (m *mockStatusOrchestrator) GetLogs(ctx context.Context, name service.ServiceName, follow bool, tail int) (ports.LogStream, error) {
	return nil, nil
}

func (m *mockStatusOrchestrator) GetAllServiceStatuses(ctx context.Context) ([]ports.ServiceStatus, error) {
	return []ports.ServiceStatus{
		{Name: "service-a", Status: "running", Health: "healthy"},
		{Name: "postgres", Status: "running", Health: "healthy"},
	}, nil
}

func (m *mockStatusOrchestrator) SetComposeFiles(files []string) {
	// no-op for tests
}

func TestStatusQueryHandler_Handle_SingleService(t *testing.T) {
	orchestrator := &mockStatusOrchestrator{
		statusFunc: func(name service.ServiceName) (ports.ServiceStatus, error) {
			return ports.ServiceStatus{
				Name:     name.String(),
				Status:   "running",
				Endpoint: "http://localhost:8080",
				Health:   "healthy",
			}, nil
		},
	}

	handler := NewStatusQueryHandler(orchestrator)

	serviceName := "service-a"
	query := StatusQuery{
		ServiceName: &serviceName,
	}

	result, err := handler.Handle(context.Background(), query)
	if err != nil {
		t.Fatalf("Handle() returned error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 status result, got %d", len(result))
	}

	status := result[0]
	if status.Name != "service-a" {
		t.Errorf("Expected name 'service-a', got %q", status.Name)
	}
	if status.Status != "running" {
		t.Errorf("Expected status 'running', got %q", status.Status)
	}
	if status.Health != "healthy" {
		t.Errorf("Expected health 'healthy', got %q", status.Health)
	}
}

func TestStatusQueryHandler_Handle_AllServices(t *testing.T) {
	orchestrator := &mockStatusOrchestrator{}
	handler := NewStatusQueryHandler(orchestrator)

	query := StatusQuery{
		ServiceName: nil, // nil means all services
	}

	result, err := handler.Handle(context.Background(), query)
	if err != nil {
		t.Fatalf("Handle() returned error: %v", err)
	}

	// Current implementation returns empty for all services
	// This test documents expected behavior
	if result == nil {
		t.Error("Expected non-nil result")
	}
}

func TestStatusQueryHandler_Handle_ServiceNotFound(t *testing.T) {
	orchestrator := &mockStatusOrchestrator{
		statusFunc: func(name service.ServiceName) (ports.ServiceStatus, error) {
			return ports.ServiceStatus{}, fmt.Errorf("service %s not found", name)
		},
	}

	handler := NewStatusQueryHandler(orchestrator)

	serviceName := "nonexistent"
	query := StatusQuery{
		ServiceName: &serviceName,
	}

	_, err := handler.Handle(context.Background(), query)
	if err == nil {
		t.Fatal("Expected error for nonexistent service, got nil")
	}
}

func TestStatusQueryHandler_Handle_DifferentStatuses(t *testing.T) {
	tests := []struct {
		name           string
		expectedStatus string
		expectedHealth string
	}{
		{"running-healthy", "running", "healthy"},
		{"running-unhealthy", "running", "unhealthy"},
		{"stopped", "stopped", ""},
		{"starting", "starting", "pending"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orchestrator := &mockStatusOrchestrator{
				statusFunc: func(name service.ServiceName) (ports.ServiceStatus, error) {
					return ports.ServiceStatus{
						Name:   name.String(),
						Status: tt.expectedStatus,
						Health: tt.expectedHealth,
					}, nil
				},
			}

			handler := NewStatusQueryHandler(orchestrator)

			serviceName := "test-service"
			query := StatusQuery{
				ServiceName: &serviceName,
			}

			result, err := handler.Handle(context.Background(), query)
			if err != nil {
				t.Fatalf("Handle() returned error: %v", err)
			}

			if len(result) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(result))
			}

			if result[0].Status != tt.expectedStatus {
				t.Errorf("Expected status %q, got %q", tt.expectedStatus, result[0].Status)
			}
			if result[0].Health != tt.expectedHealth {
				t.Errorf("Expected health %q, got %q", tt.expectedHealth, result[0].Health)
			}
		})
	}
}
