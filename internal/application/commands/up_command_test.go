package commands

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
	"github.com/Saturn-Fintech/grund/internal/domain/infrastructure"
	"github.com/Saturn-Fintech/grund/internal/domain/service"
)

// Mock implementations for testing
type mockServiceRepository struct {
	services map[service.ServiceName]*service.Service
	findErr  error
}

func (m *mockServiceRepository) FindByName(name service.ServiceName) (*service.Service, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	svc, ok := m.services[name]
	if !ok {
		return nil, fmt.Errorf("service %s not found", name)
	}
	return svc, nil
}

func (m *mockServiceRepository) FindAll() ([]*service.Service, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	result := make([]*service.Service, 0, len(m.services))
	for _, svc := range m.services {
		result = append(result, svc)
	}
	return result, nil
}

func (m *mockServiceRepository) Save(svc *service.Service) error {
	return nil
}

func (m *mockServiceRepository) ExtractTunnelConfig(name service.ServiceName) (*infrastructure.TunnelRequirement, error) {
	return nil, nil
}

type mockRegistryRepository struct{}

func (m *mockRegistryRepository) GetServicePath(name service.ServiceName) (string, error) {
	return "/path/to/" + name.String(), nil
}

func (m *mockRegistryRepository) GetAllServices() (map[service.ServiceName]ports.ServiceEntry, error) {
	return map[service.ServiceName]ports.ServiceEntry{}, nil
}

type mockOrchestrator struct {
	startErr   error
	startCalls [][]service.ServiceName
}

func (m *mockOrchestrator) StartInfrastructure(ctx context.Context) error {
	return nil
}

func (m *mockOrchestrator) StartServices(ctx context.Context, services []service.ServiceName) error {
	m.startCalls = append(m.startCalls, services)
	return m.startErr
}

func (m *mockOrchestrator) StopServices(ctx context.Context) error {
	return nil
}

func (m *mockOrchestrator) RestartService(ctx context.Context, name service.ServiceName) error {
	return nil
}

func (m *mockOrchestrator) GetServiceStatus(ctx context.Context, name service.ServiceName) (ports.ServiceStatus, error) {
	return ports.ServiceStatus{Name: name.String(), Status: "running", Health: "healthy"}, nil
}

func (m *mockOrchestrator) GetLogs(ctx context.Context, name service.ServiceName, follow bool, tail int) (ports.LogStream, error) {
	return nil, nil
}

func (m *mockOrchestrator) GetAllServiceStatuses(ctx context.Context) ([]ports.ServiceStatus, error) {
	return nil, nil
}

func (m *mockOrchestrator) SetComposeFiles(files []string) {
	// no-op for tests
}

type mockProvisioner struct {
	provisionErr    error
	postgresCalls   []*infrastructure.PostgresConfig
	mongoCalls      []*infrastructure.MongoDBConfig
	redisCalls      []*infrastructure.RedisConfig
	localstackCalls []infrastructure.InfrastructureRequirements
}

func (m *mockProvisioner) ProvisionPostgres(ctx context.Context, config *infrastructure.PostgresConfig) error {
	m.postgresCalls = append(m.postgresCalls, config)
	return m.provisionErr
}

func (m *mockProvisioner) ProvisionMongoDB(ctx context.Context, config *infrastructure.MongoDBConfig) error {
	m.mongoCalls = append(m.mongoCalls, config)
	return m.provisionErr
}

func (m *mockProvisioner) ProvisionRedis(ctx context.Context, config *infrastructure.RedisConfig) error {
	m.redisCalls = append(m.redisCalls, config)
	return m.provisionErr
}

func (m *mockProvisioner) ProvisionLocalStack(ctx context.Context, req infrastructure.InfrastructureRequirements) error {
	m.localstackCalls = append(m.localstackCalls, req)
	return m.provisionErr
}

type mockComposeGenerator struct {
	generateErr error
}

func (m *mockComposeGenerator) Generate(services []*service.Service, infra infrastructure.InfrastructureRequirements) (*ports.ComposeFileSet, error) {
	if m.generateErr != nil {
		return nil, m.generateErr
	}
	return &ports.ComposeFileSet{
		InfrastructurePath: "/tmp/infrastructure/docker-compose.yaml",
		ServicePaths:       map[string]string{},
	}, nil
}

func (m *mockComposeGenerator) GenerateWithTunnels(services []*service.Service, infra infrastructure.InfrastructureRequirements, tunnelCtx map[string]ports.TunnelContext) (*ports.ComposeFileSet, error) {
	return m.Generate(services, infra)
}

type mockHealthChecker struct{}

func (m *mockHealthChecker) CheckHealth(ctx context.Context, endpoint string, timeout int) error {
	return nil
}

func (m *mockHealthChecker) WaitForHealthy(ctx context.Context, endpoint string, interval, timeout int, retries int) error {
	return nil
}

// Helper to create a test service
func createTestService(name string, deps []string) *service.Service {
	port, _ := service.NewPort(8080)
	serviceDeps := make([]service.ServiceName, len(deps))
	for i, d := range deps {
		serviceDeps[i] = service.ServiceName(d)
	}

	return &service.Service{
		Name: name,
		Type: service.ServiceTypeGo,
		Port: port,
		Build: &service.BuildConfig{
			Dockerfile: "Dockerfile",
			Context:    ".",
		},
		Health: service.HealthConfig{
			Endpoint: "/health",
			Interval: 5 * time.Second,
			Timeout:  3 * time.Second,
			Retries:  10,
		},
		Dependencies: service.ServiceDependencies{
			Services:       serviceDeps,
			Infrastructure: infrastructure.InfrastructureRequirements{},
		},
	}
}

func TestUpCommandHandler_Handle_Success(t *testing.T) {
	// Setup services: A -> B (A depends on B)
	svcB := createTestService("service-b", []string{})
	svcA := createTestService("service-a", []string{"service-b"})

	repo := &mockServiceRepository{
		services: map[service.ServiceName]*service.Service{
			"service-a": svcA,
			"service-b": svcB,
		},
	}
	registry := &mockRegistryRepository{}
	orchestrator := &mockOrchestrator{}
	provisioner := &mockProvisioner{}
	composeGen := &mockComposeGenerator{}
	healthChecker := &mockHealthChecker{}

	handler := NewUpCommandHandler(repo, registry, orchestrator, provisioner, composeGen, healthChecker, nil)

	cmd := UpCommand{
		ServiceNames: []string{"service-a", "service-b"},
		NoDeps:       false,
		InfraOnly:    false,
	}

	err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Handle() returned error: %v", err)
	}

	// Verify orchestrator was called
	if len(orchestrator.startCalls) != 1 {
		t.Fatalf("Expected 1 StartServices call, got %d", len(orchestrator.startCalls))
	}

	// Verify both services are started (order is not enforced - services start in parallel)
	startedServices := orchestrator.startCalls[0]
	if len(startedServices) != 2 {
		t.Fatalf("Expected 2 services to start, got %d", len(startedServices))
	}

	// Check both services are present (order doesn't matter)
	hasA, hasB := false, false
	for _, name := range startedServices {
		if name == "service-a" {
			hasA = true
		}
		if name == "service-b" {
			hasB = true
		}
	}

	if !hasA || !hasB {
		t.Errorf("Expected both service-a and service-b to be started, got: %v", startedServices)
	}
}

func TestUpCommandHandler_Handle_ServiceNotFound(t *testing.T) {
	repo := &mockServiceRepository{
		services: map[service.ServiceName]*service.Service{},
	}
	registry := &mockRegistryRepository{}
	orchestrator := &mockOrchestrator{}
	provisioner := &mockProvisioner{}
	composeGen := &mockComposeGenerator{}
	healthChecker := &mockHealthChecker{}

	handler := NewUpCommandHandler(repo, registry, orchestrator, provisioner, composeGen, healthChecker, nil)

	cmd := UpCommand{
		ServiceNames: []string{"nonexistent-service"},
	}

	err := handler.Handle(context.Background(), cmd)
	if err == nil {
		t.Fatal("Expected error for nonexistent service, got nil")
	}
}

func TestUpCommandHandler_Handle_CircularDependency(t *testing.T) {
	// Create circular dependency: A -> B -> C -> A
	// Circular dependencies are now allowed - services start in parallel
	// and handle reconnection themselves
	svcC := createTestService("service-c", []string{"service-a"})
	svcB := createTestService("service-b", []string{"service-c"})
	svcA := createTestService("service-a", []string{"service-b"})

	repo := &mockServiceRepository{
		services: map[service.ServiceName]*service.Service{
			"service-a": svcA,
			"service-b": svcB,
			"service-c": svcC,
		},
	}
	registry := &mockRegistryRepository{}
	orchestrator := &mockOrchestrator{}
	provisioner := &mockProvisioner{}
	composeGen := &mockComposeGenerator{}
	healthChecker := &mockHealthChecker{}

	handler := NewUpCommandHandler(repo, registry, orchestrator, provisioner, composeGen, healthChecker, nil)

	cmd := UpCommand{
		ServiceNames: []string{"service-a", "service-b", "service-c"},
	}

	// Circular dependencies should now succeed - services start in parallel
	err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Handle() should succeed with circular dependencies, got error: %v", err)
	}

	// Verify all services were started
	if len(orchestrator.startCalls) != 1 {
		t.Fatalf("Expected 1 StartServices call, got %d", len(orchestrator.startCalls))
	}

	startedServices := orchestrator.startCalls[0]
	if len(startedServices) != 3 {
		t.Fatalf("Expected 3 services to start, got %d", len(startedServices))
	}
}

func TestUpCommandHandler_Handle_TransitiveDependencies(t *testing.T) {
	// A depends on B, B depends on C
	// Running `grund up service-a` should load all three
	svcC := createTestService("service-c", []string{})
	svcB := createTestService("service-b", []string{"service-c"})
	svcA := createTestService("service-a", []string{"service-b"})

	repo := &mockServiceRepository{
		services: map[service.ServiceName]*service.Service{
			"service-a": svcA,
			"service-b": svcB,
			"service-c": svcC,
		},
	}
	registry := &mockRegistryRepository{}
	orchestrator := &mockOrchestrator{}
	provisioner := &mockProvisioner{}
	composeGen := &mockComposeGenerator{}
	healthChecker := &mockHealthChecker{}

	handler := NewUpCommandHandler(repo, registry, orchestrator, provisioner, composeGen, healthChecker, nil)

	// Only request service-a, but B and C should be loaded as dependencies
	cmd := UpCommand{
		ServiceNames: []string{"service-a"},
	}

	err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Handle() returned error: %v", err)
	}

	// Verify all three services were started (transitive deps loaded)
	if len(orchestrator.startCalls) != 1 {
		t.Fatalf("Expected 1 StartServices call, got %d", len(orchestrator.startCalls))
	}

	startedServices := orchestrator.startCalls[0]
	if len(startedServices) != 3 {
		t.Fatalf("Expected 3 services (including transitive deps), got %d: %v", len(startedServices), startedServices)
	}
}

func TestUpCommandHandler_Handle_InfraOnly(t *testing.T) {
	svc := createTestService("service-a", []string{})
	svc.Dependencies.Infrastructure = infrastructure.InfrastructureRequirements{
		Postgres: &infrastructure.PostgresConfig{Database: "testdb"},
	}

	repo := &mockServiceRepository{
		services: map[service.ServiceName]*service.Service{
			"service-a": svc,
		},
	}
	registry := &mockRegistryRepository{}
	orchestrator := &mockOrchestrator{}
	provisioner := &mockProvisioner{}
	composeGen := &mockComposeGenerator{}
	healthChecker := &mockHealthChecker{}

	handler := NewUpCommandHandler(repo, registry, orchestrator, provisioner, composeGen, healthChecker, nil)

	cmd := UpCommand{
		ServiceNames: []string{"service-a"},
		InfraOnly:    true, // Only provision infrastructure
	}

	err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Handle() returned error: %v", err)
	}

	// Verify postgres was provisioned
	if len(provisioner.postgresCalls) != 1 {
		t.Errorf("Expected 1 ProvisionPostgres call, got %d", len(provisioner.postgresCalls))
	}

	// Verify services were NOT started
	if len(orchestrator.startCalls) != 0 {
		t.Errorf("Expected 0 StartServices calls for infra-only, got %d", len(orchestrator.startCalls))
	}
}

func TestUpCommandHandler_Handle_NoDeps(t *testing.T) {
	// A depends on B, but we use NoDeps flag
	// Since we no longer enforce dependency ordering, this should work
	svcA := createTestService("service-a", []string{"service-b"})

	repo := &mockServiceRepository{
		services: map[service.ServiceName]*service.Service{
			"service-a": svcA,
		},
	}
	registry := &mockRegistryRepository{}
	orchestrator := &mockOrchestrator{}
	provisioner := &mockProvisioner{}
	composeGen := &mockComposeGenerator{}
	healthChecker := &mockHealthChecker{}

	handler := NewUpCommandHandler(repo, registry, orchestrator, provisioner, composeGen, healthChecker, nil)

	// Only start service-a, even though it depends on service-b
	cmd := UpCommand{
		ServiceNames: []string{"service-a"},
		NoDeps:       true,
	}

	// This should succeed - we no longer build dependency graphs
	// Service-a will start and handle reconnection to service-b itself
	err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Handle() should succeed with NoDeps, got error: %v", err)
	}

	// Verify service-a was started
	if len(orchestrator.startCalls) != 1 {
		t.Fatalf("Expected 1 StartServices call, got %d", len(orchestrator.startCalls))
	}

	if len(orchestrator.startCalls[0]) != 1 || orchestrator.startCalls[0][0] != "service-a" {
		t.Errorf("Expected only service-a to start, got: %v", orchestrator.startCalls[0])
	}
}

func TestUpCommandHandler_Handle_ProvisioningFails(t *testing.T) {
	svc := createTestService("service-a", []string{})
	svc.Dependencies.Infrastructure = infrastructure.InfrastructureRequirements{
		Postgres: &infrastructure.PostgresConfig{Database: "testdb"},
	}

	repo := &mockServiceRepository{
		services: map[service.ServiceName]*service.Service{
			"service-a": svc,
		},
	}
	registry := &mockRegistryRepository{}
	orchestrator := &mockOrchestrator{}
	provisioner := &mockProvisioner{
		provisionErr: fmt.Errorf("postgres connection failed"),
	}
	composeGen := &mockComposeGenerator{}
	healthChecker := &mockHealthChecker{}

	handler := NewUpCommandHandler(repo, registry, orchestrator, provisioner, composeGen, healthChecker, nil)

	cmd := UpCommand{
		ServiceNames: []string{"service-a"},
	}

	err := handler.Handle(context.Background(), cmd)
	if err == nil {
		t.Fatal("Expected error for provisioning failure, got nil")
	}
	if len(orchestrator.startCalls) > 0 {
		t.Error("Services should not have started after provisioning failure")
	}
}

func TestUpCommandHandler_Handle_WithLocalStack(t *testing.T) {
	svc := createTestService("service-a", []string{})
	svc.Dependencies.Infrastructure = infrastructure.InfrastructureRequirements{
		SQS: &infrastructure.SQSConfig{
			Queues: []infrastructure.QueueConfig{{Name: "test-queue", DLQ: true}},
		},
		SNS: &infrastructure.SNSConfig{
			Topics: []infrastructure.TopicConfig{{Name: "test-topic"}},
		},
	}

	repo := &mockServiceRepository{
		services: map[service.ServiceName]*service.Service{
			"service-a": svc,
		},
	}
	registry := &mockRegistryRepository{}
	orchestrator := &mockOrchestrator{}
	provisioner := &mockProvisioner{}
	composeGen := &mockComposeGenerator{}
	healthChecker := &mockHealthChecker{}

	handler := NewUpCommandHandler(repo, registry, orchestrator, provisioner, composeGen, healthChecker, nil)

	cmd := UpCommand{
		ServiceNames: []string{"service-a"},
	}

	err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Handle() returned error: %v", err)
	}

	// Verify LocalStack was provisioned
	if len(provisioner.localstackCalls) != 1 {
		t.Errorf("Expected 1 ProvisionLocalStack call, got %d", len(provisioner.localstackCalls))
	}

	// Verify the requirements were passed correctly
	req := provisioner.localstackCalls[0]
	if req.SQS == nil || len(req.SQS.Queues) != 1 {
		t.Error("Expected SQS queue in LocalStack requirements")
	}
	if req.SNS == nil || len(req.SNS.Topics) != 1 {
		t.Error("Expected SNS topic in LocalStack requirements")
	}
}
