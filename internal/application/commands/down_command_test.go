package commands

import (
	"context"
	"fmt"
	"testing"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
	"github.com/Saturn-Fintech/grund/internal/domain/infrastructure"
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

type mockDownServiceRepo struct{}

func (m *mockDownServiceRepo) FindByName(name service.ServiceName) (*service.Service, error) {
	return &service.Service{Name: string(name)}, nil
}

func (m *mockDownServiceRepo) FindAll() ([]*service.Service, error) {
	return []*service.Service{}, nil
}

func (m *mockDownServiceRepo) Save(svc *service.Service) error {
	return nil
}

func (m *mockDownServiceRepo) ExtractTunnelConfig(name service.ServiceName) (*infrastructure.TunnelRequirement, error) {
	return nil, nil
}

type mockDownRegistryRepo struct{}

func (m *mockDownRegistryRepo) GetServicePath(name service.ServiceName) (string, error) {
	return "/tmp/test", nil
}

func (m *mockDownRegistryRepo) GetAllServices() (map[service.ServiceName]ports.ServiceEntry, error) {
	return map[service.ServiceName]ports.ServiceEntry{}, nil
}

type mockDownHookExecutor struct{}

func (m *mockDownHookExecutor) Execute(ctx context.Context, hook service.Hook, execCtx ports.HookExecutionContext) error {
	return nil
}

func (m *mockDownHookExecutor) ExecuteAll(ctx context.Context, hooks []service.Hook, execCtx ports.HookExecutionContext) error {
	return nil
}

func TestDownCommandHandler_Handle_Success(t *testing.T) {
	orchestrator := &mockDownOrchestrator{}
	handler := NewDownCommandHandler(&mockDownServiceRepo{}, &mockDownRegistryRepo{}, orchestrator, &mockDownHookExecutor{})

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
	handler := NewDownCommandHandler(&mockDownServiceRepo{}, &mockDownRegistryRepo{}, orchestrator, &mockDownHookExecutor{})

	cmd := DownCommand{}

	err := handler.Handle(context.Background(), cmd)
	if err == nil {
		t.Fatal("Expected error when orchestrator fails, got nil")
	}
}

func TestDownCommandHandler_Handle_MultipleCalls(t *testing.T) {
	orchestrator := &mockDownOrchestrator{}
	handler := NewDownCommandHandler(&mockDownServiceRepo{}, &mockDownRegistryRepo{}, orchestrator, &mockDownHookExecutor{})

	// Call Handle twice
	_ = handler.Handle(context.Background(), DownCommand{})
	_ = handler.Handle(context.Background(), DownCommand{})

	if orchestrator.stopCalls != 2 {
		t.Errorf("Expected 2 StopServices calls, got %d", orchestrator.stopCalls)
	}
}
