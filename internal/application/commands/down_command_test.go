package commands

import (
	"context"
	"fmt"
	"testing"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
	"github.com/Saturn-Fintech/grund/internal/domain/service"
)

type mockDownOrchestrator struct {
	stopErr   error
	stopCalls int
}

func (m *mockDownOrchestrator) StartInfrastructure(ctx context.Context) error {
	return nil
}

func (m *mockDownOrchestrator) StartServices(ctx context.Context, services []service.ServiceName) error {
	return nil
}

func (m *mockDownOrchestrator) StopServices(ctx context.Context) error {
	m.stopCalls++
	return m.stopErr
}

func (m *mockDownOrchestrator) RestartService(ctx context.Context, name service.ServiceName) error {
	return nil
}

func (m *mockDownOrchestrator) GetServiceStatus(ctx context.Context, name service.ServiceName) (ports.ServiceStatus, error) {
	return ports.ServiceStatus{}, nil
}

func (m *mockDownOrchestrator) GetLogs(ctx context.Context, name service.ServiceName, follow bool, tail int) (ports.LogStream, error) {
	return nil, nil
}

func (m *mockDownOrchestrator) GetAllServiceStatuses(ctx context.Context) ([]ports.ServiceStatus, error) {
	return nil, nil
}

func (m *mockDownOrchestrator) SetComposeFiles(files []string) {
	// no-op for tests
}

func TestDownCommandHandler_Handle_Success(t *testing.T) {
	orchestrator := &mockDownOrchestrator{}
	handler := NewDownCommandHandler(orchestrator)

	cmd := DownCommand{}

	err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Handle() returned error: %v", err)
	}

	if orchestrator.stopCalls != 1 {
		t.Errorf("Expected 1 StopServices call, got %d", orchestrator.stopCalls)
	}
}

func TestDownCommandHandler_Handle_OrchestratorFails(t *testing.T) {
	orchestrator := &mockDownOrchestrator{
		stopErr: fmt.Errorf("docker compose down failed"),
	}
	handler := NewDownCommandHandler(orchestrator)

	cmd := DownCommand{}

	err := handler.Handle(context.Background(), cmd)
	if err == nil {
		t.Fatal("Expected error when orchestrator fails, got nil")
	}
}

func TestDownCommandHandler_Handle_MultipleCalls(t *testing.T) {
	orchestrator := &mockDownOrchestrator{}
	handler := NewDownCommandHandler(orchestrator)

	// Call Handle twice
	_ = handler.Handle(context.Background(), DownCommand{})
	_ = handler.Handle(context.Background(), DownCommand{})

	if orchestrator.stopCalls != 2 {
		t.Errorf("Expected 2 StopServices calls, got %d", orchestrator.stopCalls)
	}
}
