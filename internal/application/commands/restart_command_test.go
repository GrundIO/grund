package commands

import (
	"context"
	"fmt"
	"testing"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
	"github.com/Saturn-Fintech/grund/internal/domain/service"
)

type mockRestartOrchestrator struct {
	restartErr   error
	restartCalls []service.ServiceName
}

func (m *mockRestartOrchestrator) StartInfrastructure(ctx context.Context) error {
	return nil
}

func (m *mockRestartOrchestrator) StartServices(ctx context.Context, services []service.ServiceName) error {
	return nil
}

func (m *mockRestartOrchestrator) StopServices(ctx context.Context) error {
	return nil
}

func (m *mockRestartOrchestrator) RestartService(ctx context.Context, name service.ServiceName) error {
	m.restartCalls = append(m.restartCalls, name)
	return m.restartErr
}

func (m *mockRestartOrchestrator) GetServiceStatus(ctx context.Context, name service.ServiceName) (ports.ServiceStatus, error) {
	return ports.ServiceStatus{}, nil
}

func (m *mockRestartOrchestrator) GetLogs(ctx context.Context, name service.ServiceName, follow bool, tail int) (ports.LogStream, error) {
	return nil, nil
}

func (m *mockRestartOrchestrator) GetAllServiceStatuses(ctx context.Context) ([]ports.ServiceStatus, error) {
	return nil, nil
}

func (m *mockRestartOrchestrator) SetComposeFiles(files []string) {
	// no-op for tests
}

func TestRestartCommandHandler_Handle_SingleService(t *testing.T) {
	orchestrator := &mockRestartOrchestrator{}
	handler := NewRestartCommandHandler(orchestrator)

	cmd := RestartCommand{
		ServiceNames: []string{"service-a"},
	}

	err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Handle() returned error: %v", err)
	}

	if len(orchestrator.restartCalls) != 1 {
		t.Fatalf("Expected 1 RestartService call, got %d", len(orchestrator.restartCalls))
	}

	if orchestrator.restartCalls[0] != "service-a" {
		t.Errorf("Expected restart of 'service-a', got %q", orchestrator.restartCalls[0])
	}
}

func TestRestartCommandHandler_Handle_MultipleServices(t *testing.T) {
	orchestrator := &mockRestartOrchestrator{}
	handler := NewRestartCommandHandler(orchestrator)

	cmd := RestartCommand{
		ServiceNames: []string{"service-a", "service-b", "service-c"},
	}

	err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Handle() returned error: %v", err)
	}

	if len(orchestrator.restartCalls) != 3 {
		t.Fatalf("Expected 3 RestartService calls, got %d", len(orchestrator.restartCalls))
	}

	// Verify order
	expected := []service.ServiceName{"service-a", "service-b", "service-c"}
	for i, name := range orchestrator.restartCalls {
		if name != expected[i] {
			t.Errorf("Call %d: expected %q, got %q", i, expected[i], name)
		}
	}
}

func TestRestartCommandHandler_Handle_ErrorStopsProcessing(t *testing.T) {
	orchestrator := &mockRestartOrchestrator{
		restartErr: fmt.Errorf("restart failed"),
	}
	handler := NewRestartCommandHandler(orchestrator)

	cmd := RestartCommand{
		ServiceNames: []string{"service-a", "service-b"},
	}

	err := handler.Handle(context.Background(), cmd)
	if err == nil {
		t.Fatal("Expected error when restart fails, got nil")
	}

	// Should have only tried to restart the first service
	if len(orchestrator.restartCalls) != 1 {
		t.Errorf("Expected 1 RestartService call (stopped on error), got %d", len(orchestrator.restartCalls))
	}
}

func TestRestartCommandHandler_Handle_EmptyServices(t *testing.T) {
	orchestrator := &mockRestartOrchestrator{}
	handler := NewRestartCommandHandler(orchestrator)

	cmd := RestartCommand{
		ServiceNames: []string{},
	}

	err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Handle() returned error for empty services: %v", err)
	}

	if len(orchestrator.restartCalls) != 0 {
		t.Errorf("Expected 0 RestartService calls for empty list, got %d", len(orchestrator.restartCalls))
	}
}
