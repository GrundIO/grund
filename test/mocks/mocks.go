package mocks

import (
	"context"
	"fmt"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
	"github.com/Saturn-Fintech/grund/internal/domain/infrastructure"
	"github.com/Saturn-Fintech/grund/internal/domain/service"
)

// MockServiceRepository is a mock implementation of ports.ServiceRepository
type MockServiceRepository struct {
	FindByNameFunc func(name service.ServiceName) (*service.Service, error)
	FindAllFunc    func() ([]*service.Service, error)
	SaveFunc       func(svc *service.Service) error

	// Track calls for assertions
	FindByNameCalls []service.ServiceName
	SaveCalls       []*service.Service
}

func (m *MockServiceRepository) FindByName(name service.ServiceName) (*service.Service, error) {
	m.FindByNameCalls = append(m.FindByNameCalls, name)
	if m.FindByNameFunc != nil {
		return m.FindByNameFunc(name)
	}
	return nil, fmt.Errorf("FindByName not implemented")
}

func (m *MockServiceRepository) FindAll() ([]*service.Service, error) {
	if m.FindAllFunc != nil {
		return m.FindAllFunc()
	}
	return nil, fmt.Errorf("FindAll not implemented")
}

func (m *MockServiceRepository) Save(svc *service.Service) error {
	m.SaveCalls = append(m.SaveCalls, svc)
	if m.SaveFunc != nil {
		return m.SaveFunc(svc)
	}
	return nil
}

// MockServiceRegistryRepository is a mock implementation of ports.ServiceRegistryRepository
type MockServiceRegistryRepository struct {
	GetServicePathFunc func(name service.ServiceName) (string, error)
	GetAllServicesFunc func() (map[service.ServiceName]ports.ServiceEntry, error)

	// Track calls
	GetServicePathCalls []service.ServiceName
}

func (m *MockServiceRegistryRepository) GetServicePath(name service.ServiceName) (string, error) {
	m.GetServicePathCalls = append(m.GetServicePathCalls, name)
	if m.GetServicePathFunc != nil {
		return m.GetServicePathFunc(name)
	}
	return "", fmt.Errorf("GetServicePath not implemented")
}

func (m *MockServiceRegistryRepository) GetAllServices() (map[service.ServiceName]ports.ServiceEntry, error) {
	if m.GetAllServicesFunc != nil {
		return m.GetAllServicesFunc()
	}
	return nil, fmt.Errorf("GetAllServices not implemented")
}

// MockContainerOrchestrator is a mock implementation of ports.ContainerOrchestrator
type MockContainerOrchestrator struct {
	StartServicesFunc    func(ctx context.Context, services []service.ServiceName) error
	StopServicesFunc     func(ctx context.Context) error
	RestartServiceFunc   func(ctx context.Context, name service.ServiceName) error
	GetServiceStatusFunc func(ctx context.Context, name service.ServiceName) (ports.ServiceStatus, error)
	GetLogsFunc          func(ctx context.Context, name service.ServiceName, follow bool, tail int) (ports.LogStream, error)

	// Track calls
	StartServicesCalls  [][]service.ServiceName
	StopServicesCalls   int
	RestartServiceCalls []service.ServiceName
}

func (m *MockContainerOrchestrator) StartServices(ctx context.Context, services []service.ServiceName) error {
	m.StartServicesCalls = append(m.StartServicesCalls, services)
	if m.StartServicesFunc != nil {
		return m.StartServicesFunc(ctx, services)
	}
	return nil
}

func (m *MockContainerOrchestrator) StopServices(ctx context.Context) error {
	m.StopServicesCalls++
	if m.StopServicesFunc != nil {
		return m.StopServicesFunc(ctx)
	}
	return nil
}

func (m *MockContainerOrchestrator) RestartService(ctx context.Context, name service.ServiceName) error {
	m.RestartServiceCalls = append(m.RestartServiceCalls, name)
	if m.RestartServiceFunc != nil {
		return m.RestartServiceFunc(ctx, name)
	}
	return nil
}

func (m *MockContainerOrchestrator) GetServiceStatus(ctx context.Context, name service.ServiceName) (ports.ServiceStatus, error) {
	if m.GetServiceStatusFunc != nil {
		return m.GetServiceStatusFunc(ctx, name)
	}
	return ports.ServiceStatus{
		Name:   name.String(),
		Status: "running",
		Health: "healthy",
	}, nil
}

func (m *MockContainerOrchestrator) GetLogs(ctx context.Context, name service.ServiceName, follow bool, tail int) (ports.LogStream, error) {
	if m.GetLogsFunc != nil {
		return m.GetLogsFunc(ctx, name, follow, tail)
	}
	return nil, fmt.Errorf("GetLogs not implemented")
}

// MockInfrastructureProvisioner is a mock implementation of ports.InfrastructureProvisioner
type MockInfrastructureProvisioner struct {
	ProvisionPostgresFunc   func(ctx context.Context, config *infrastructure.PostgresConfig) error
	ProvisionMongoDBFunc    func(ctx context.Context, config *infrastructure.MongoDBConfig) error
	ProvisionRedisFunc      func(ctx context.Context, config *infrastructure.RedisConfig) error
	ProvisionLocalStackFunc func(ctx context.Context, req infrastructure.InfrastructureRequirements) (*ports.ProvisionedAWSResources, error)

	// Track calls
	ProvisionPostgresCalls   []*infrastructure.PostgresConfig
	ProvisionMongoDBCalls    []*infrastructure.MongoDBConfig
	ProvisionRedisCalls      []*infrastructure.RedisConfig
	ProvisionLocalStackCalls []infrastructure.InfrastructureRequirements
}

func (m *MockInfrastructureProvisioner) ProvisionPostgres(ctx context.Context, config *infrastructure.PostgresConfig) error {
	m.ProvisionPostgresCalls = append(m.ProvisionPostgresCalls, config)
	if m.ProvisionPostgresFunc != nil {
		return m.ProvisionPostgresFunc(ctx, config)
	}
	return nil
}

func (m *MockInfrastructureProvisioner) ProvisionMongoDB(ctx context.Context, config *infrastructure.MongoDBConfig) error {
	m.ProvisionMongoDBCalls = append(m.ProvisionMongoDBCalls, config)
	if m.ProvisionMongoDBFunc != nil {
		return m.ProvisionMongoDBFunc(ctx, config)
	}
	return nil
}

func (m *MockInfrastructureProvisioner) ProvisionRedis(ctx context.Context, config *infrastructure.RedisConfig) error {
	m.ProvisionRedisCalls = append(m.ProvisionRedisCalls, config)
	if m.ProvisionRedisFunc != nil {
		return m.ProvisionRedisFunc(ctx, config)
	}
	return nil
}

func (m *MockInfrastructureProvisioner) ProvisionLocalStack(ctx context.Context, req infrastructure.InfrastructureRequirements) (*ports.ProvisionedAWSResources, error) {
	m.ProvisionLocalStackCalls = append(m.ProvisionLocalStackCalls, req)
	if m.ProvisionLocalStackFunc != nil {
		return m.ProvisionLocalStackFunc(ctx, req)
	}
	return nil, nil
}

// MockComposeGenerator is a mock implementation of ports.ComposeGenerator
type MockComposeGenerator struct {
	GenerateFunc                 func(services []*service.Service, infra infrastructure.InfrastructureRequirements) (*ports.ComposeFileSet, error)
	GenerateWithTunnelsFunc      func(services []*service.Service, infra infrastructure.InfrastructureRequirements, tunnelCtx map[string]ports.TunnelContext) (*ports.ComposeFileSet, error)
	GenerateWithAWSResourcesFunc func(services []*service.Service, infra infrastructure.InfrastructureRequirements, tunnelCtx map[string]ports.TunnelContext, awsResources *ports.ProvisionedAWSResources) (*ports.ComposeFileSet, error)

	// Track calls
	GenerateCalls []struct {
		Services []*service.Service
		Infra    infrastructure.InfrastructureRequirements
	}
}

func (m *MockComposeGenerator) Generate(services []*service.Service, infra infrastructure.InfrastructureRequirements) (*ports.ComposeFileSet, error) {
	m.GenerateCalls = append(m.GenerateCalls, struct {
		Services []*service.Service
		Infra    infrastructure.InfrastructureRequirements
	}{Services: services, Infra: infra})
	if m.GenerateFunc != nil {
		return m.GenerateFunc(services, infra)
	}
	return &ports.ComposeFileSet{
		InfrastructurePath: "/tmp/infrastructure/docker-compose.yaml",
		ServicePaths:       make(map[string]string),
	}, nil
}

func (m *MockComposeGenerator) GenerateWithTunnels(services []*service.Service, infra infrastructure.InfrastructureRequirements, tunnelCtx map[string]ports.TunnelContext) (*ports.ComposeFileSet, error) {
	if m.GenerateWithTunnelsFunc != nil {
		return m.GenerateWithTunnelsFunc(services, infra, tunnelCtx)
	}
	return &ports.ComposeFileSet{
		InfrastructurePath: "/tmp/infrastructure/docker-compose.yaml",
		ServicePaths:       make(map[string]string),
	}, nil
}

func (m *MockComposeGenerator) GenerateWithAWSResources(services []*service.Service, infra infrastructure.InfrastructureRequirements, tunnelCtx map[string]ports.TunnelContext, awsResources *ports.ProvisionedAWSResources) (*ports.ComposeFileSet, error) {
	if m.GenerateWithAWSResourcesFunc != nil {
		return m.GenerateWithAWSResourcesFunc(services, infra, tunnelCtx, awsResources)
	}
	return &ports.ComposeFileSet{
		InfrastructurePath: "/tmp/infrastructure/docker-compose.yaml",
		ServicePaths:       make(map[string]string),
	}, nil
}

// MockHealthChecker is a mock implementation of ports.HealthChecker
type MockHealthChecker struct {
	CheckHealthFunc    func(ctx context.Context, endpoint string, timeout int) error
	WaitForHealthyFunc func(ctx context.Context, endpoint string, interval, timeout int, retries int) error

	// Track calls
	CheckHealthCalls    []string
	WaitForHealthyCalls []string
}

func (m *MockHealthChecker) CheckHealth(ctx context.Context, endpoint string, timeout int) error {
	m.CheckHealthCalls = append(m.CheckHealthCalls, endpoint)
	if m.CheckHealthFunc != nil {
		return m.CheckHealthFunc(ctx, endpoint, timeout)
	}
	return nil
}

func (m *MockHealthChecker) WaitForHealthy(ctx context.Context, endpoint string, interval, timeout int, retries int) error {
	m.WaitForHealthyCalls = append(m.WaitForHealthyCalls, endpoint)
	if m.WaitForHealthyFunc != nil {
		return m.WaitForHealthyFunc(ctx, endpoint, interval, timeout, retries)
	}
	return nil
}

// MockEnvironmentResolver is a mock implementation of ports.EnvironmentResolver
type MockEnvironmentResolver struct {
	ResolveFunc func(envRefs map[string]string, context ports.EnvironmentContext) (map[string]string, error)
}

func (m *MockEnvironmentResolver) Resolve(envRefs map[string]string, context ports.EnvironmentContext) (map[string]string, error) {
	if m.ResolveFunc != nil {
		return m.ResolveFunc(envRefs, context)
	}
	// Return envRefs as-is for simple mock
	return envRefs, nil
}

// MockLogStream is a mock implementation of ports.LogStream
type MockLogStream struct {
	ReadFunc  func() ([]byte, error)
	CloseFunc func() error

	logs   []byte
	closed bool
}

func NewMockLogStream(logs string) *MockLogStream {
	return &MockLogStream{logs: []byte(logs)}
}

func (m *MockLogStream) Read() ([]byte, error) {
	if m.ReadFunc != nil {
		return m.ReadFunc()
	}
	if m.closed {
		return nil, fmt.Errorf("stream closed")
	}
	return m.logs, nil
}

func (m *MockLogStream) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	m.closed = true
	return nil
}
