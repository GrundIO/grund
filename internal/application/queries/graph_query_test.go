package queries

import (
	"fmt"
	"testing"
	"time"

	"github.com/GrundIO/grund/internal/application/ports"
	"github.com/GrundIO/grund/internal/domain/infrastructure"
	"github.com/GrundIO/grund/internal/domain/service"
)

type mockGraphServiceRepo struct {
	services map[service.ServiceName]*service.Service
}

func (m *mockGraphServiceRepo) FindByName(name service.ServiceName) (*service.Service, error) {
	svc, ok := m.services[name]
	if !ok {
		return nil, fmt.Errorf("service %s not found", name)
	}
	return svc, nil
}

func (m *mockGraphServiceRepo) FindAll() ([]*service.Service, error) {
	result := make([]*service.Service, 0, len(m.services))
	for _, svc := range m.services {
		result = append(result, svc)
	}
	return result, nil
}

func (m *mockGraphServiceRepo) Save(svc *service.Service) error {
	return nil
}

func (m *mockGraphServiceRepo) ExtractTunnelConfig(name service.ServiceName) (*infrastructure.TunnelRequirement, error) {
	return nil, nil
}

type mockGraphRegistryRepo struct {
	entries map[service.ServiceName]ports.ServiceEntry
}

func (m *mockGraphRegistryRepo) GetServicePath(name service.ServiceName) (string, error) {
	return "", nil
}

func (m *mockGraphRegistryRepo) GetAllServices() (map[service.ServiceName]ports.ServiceEntry, error) {
	return m.entries, nil
}

func registryFromNames(names ...string) *mockGraphRegistryRepo {
	entries := make(map[service.ServiceName]ports.ServiceEntry, len(names))
	for _, n := range names {
		entries[service.ServiceName(n)] = ports.ServiceEntry{}
	}
	return &mockGraphRegistryRepo{entries: entries}
}

func newTestService(name string, deps []string, infra infrastructure.InfrastructureRequirements) *service.Service {
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
			Infrastructure: infra,
		},
	}
}

func TestGraphQueryHandler_AllServices(t *testing.T) {
	svcC := newTestService("service-c", []string{}, infrastructure.InfrastructureRequirements{})
	svcB := newTestService("service-b", []string{"service-c"}, infrastructure.InfrastructureRequirements{})
	svcA := newTestService("service-a", []string{"service-b"}, infrastructure.InfrastructureRequirements{})

	repo := &mockGraphServiceRepo{
		services: map[service.ServiceName]*service.Service{
			"service-a": svcA,
			"service-b": svcB,
			"service-c": svcC,
		},
	}

	registry := registryFromNames("service-a", "service-b", "service-c")
	handler := NewGraphQueryHandler(repo, registry)
	result, err := handler.Handle(GraphQuery{})
	if err != nil {
		t.Fatalf("Handle() returned error: %v", err)
	}

	if len(result.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(result.Nodes))
	}

	// Verify edges
	nodeA := result.Nodes["service-a"]
	if len(nodeA.Dependencies) != 1 || nodeA.Dependencies[0] != "service-b" {
		t.Errorf("service-a deps = %v, want [service-b]", nodeA.Dependencies)
	}

	nodeC := result.Nodes["service-c"]
	if len(nodeC.Dependencies) != 0 {
		t.Errorf("service-c deps = %v, want []", nodeC.Dependencies)
	}

}

func TestGraphQueryHandler_FilteredService(t *testing.T) {
	svcC := newTestService("service-c", []string{}, infrastructure.InfrastructureRequirements{})
	svcB := newTestService("service-b", []string{"service-c"}, infrastructure.InfrastructureRequirements{})
	svcA := newTestService("service-a", []string{"service-b"}, infrastructure.InfrastructureRequirements{})
	svcD := newTestService("service-d", []string{}, infrastructure.InfrastructureRequirements{})

	repo := &mockGraphServiceRepo{
		services: map[service.ServiceName]*service.Service{
			"service-a": svcA,
			"service-b": svcB,
			"service-c": svcC,
			"service-d": svcD,
		},
	}

	registry := registryFromNames("service-a", "service-b", "service-c", "service-d")
	handler := NewGraphQueryHandler(repo, registry)
	result, err := handler.Handle(GraphQuery{ServiceNames: []string{"service-a"}})
	if err != nil {
		t.Fatalf("Handle() returned error: %v", err)
	}

	// Should include service-a and its transitive deps (service-b, service-c), but NOT service-d
	if len(result.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(result.Nodes))
	}

	if _, ok := result.Nodes["service-d"]; ok {
		t.Error("service-d should not be in the graph")
	}
}

func TestGraphQueryHandler_ShowInfra(t *testing.T) {
	svc := newTestService("service-a", []string{}, infrastructure.InfrastructureRequirements{
		Postgres: &infrastructure.PostgresConfig{Database: "mydb"},
		Redis:    &infrastructure.RedisConfig{},
	})

	repo := &mockGraphServiceRepo{
		services: map[service.ServiceName]*service.Service{
			"service-a": svc,
		},
	}

	registry := registryFromNames("service-a")
	handler := NewGraphQueryHandler(repo, registry)
	result, err := handler.Handle(GraphQuery{ShowInfra: true})
	if err != nil {
		t.Fatalf("Handle() returned error: %v", err)
	}

	node := result.Nodes["service-a"]
	if len(node.Infrastructure) != 2 {
		t.Fatalf("expected 2 infra types, got %d: %v", len(node.Infrastructure), node.Infrastructure)
	}

	infraMap := make(map[string]bool)
	for _, i := range node.Infrastructure {
		infraMap[i] = true
	}
	if !infraMap["postgres:mydb"] {
		t.Error("expected postgres:mydb in infrastructure")
	}
	if !infraMap["redis"] {
		t.Error("expected redis in infrastructure")
	}
}

func TestGraphQueryHandler_EmptyGraph(t *testing.T) {
	repo := &mockGraphServiceRepo{
		services: map[service.ServiceName]*service.Service{},
	}

	registry := &mockGraphRegistryRepo{entries: map[service.ServiceName]ports.ServiceEntry{}}
	handler := NewGraphQueryHandler(repo, registry)
	result, err := handler.Handle(GraphQuery{})
	if err != nil {
		t.Fatalf("Handle() returned error: %v", err)
	}

	if len(result.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(result.Nodes))
	}
}

func TestGraphQueryHandler_SingleServiceNoDeps(t *testing.T) {
	svc := newTestService("standalone", []string{}, infrastructure.InfrastructureRequirements{})

	repo := &mockGraphServiceRepo{
		services: map[service.ServiceName]*service.Service{
			"standalone": svc,
		},
	}

	registry := registryFromNames("standalone")
	handler := NewGraphQueryHandler(repo, registry)
	result, err := handler.Handle(GraphQuery{})
	if err != nil {
		t.Fatalf("Handle() returned error: %v", err)
	}

	if len(result.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(result.Nodes))
	}

	node := result.Nodes["standalone"]
	if len(node.Dependencies) != 0 {
		t.Errorf("expected 0 deps, got %v", node.Dependencies)
	}
	if len(node.Dependents) != 0 {
		t.Errorf("expected 0 dependents, got %v", node.Dependents)
	}

}

func TestGraphQueryHandler_MissingDependencySkipped(t *testing.T) {
	// service-a depends on service-b, but service-b has no grund.yaml (not in repo)
	svcA := newTestService("service-a", []string{"service-b"}, infrastructure.InfrastructureRequirements{})

	repo := &mockGraphServiceRepo{
		services: map[service.ServiceName]*service.Service{
			"service-a": svcA,
		},
	}

	registry := registryFromNames("service-a", "service-b")
	handler := NewGraphQueryHandler(repo, registry)
	result, err := handler.Handle(GraphQuery{ServiceNames: []string{"service-a"}})
	if err != nil {
		t.Fatalf("Handle() should not error when dependency is missing, got: %v", err)
	}

	// Only service-a should be in the graph; service-b was skipped
	if len(result.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(result.Nodes))
	}

	nodeA := result.Nodes["service-a"]
	if len(nodeA.Dependencies) != 0 {
		t.Errorf("expected 0 deps after stripping missing service-b, got %v", nodeA.Dependencies)
	}
}

func TestGraphQueryHandler_MissingDependencySkipped_FindAll(t *testing.T) {
	// service-a loads fine, service-b is in registry but FindByName fails
	svcA := newTestService("service-a", []string{"service-b"}, infrastructure.InfrastructureRequirements{})

	repo := &mockGraphServiceRepo{
		services: map[service.ServiceName]*service.Service{
			"service-a": svcA,
		},
	}

	// Both are in registry, but only service-a can be loaded
	registry := registryFromNames("service-a", "service-b")
	handler := NewGraphQueryHandler(repo, registry)
	result, err := handler.Handle(GraphQuery{})
	if err != nil {
		t.Fatalf("Handle() should not error when a registered service is missing grund.yaml, got: %v", err)
	}

	if len(result.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(result.Nodes))
	}

	nodeA := result.Nodes["service-a"]
	if len(nodeA.Dependencies) != 0 {
		t.Errorf("expected 0 deps after stripping unloadable service-b, got %v", nodeA.Dependencies)
	}
}

func TestGraphQueryHandler_CircularDependency(t *testing.T) {
	// A -> B -> C -> A (cycle)
	svcA := newTestService("service-a", []string{"service-b"}, infrastructure.InfrastructureRequirements{})
	svcB := newTestService("service-b", []string{"service-c"}, infrastructure.InfrastructureRequirements{})
	svcC := newTestService("service-c", []string{"service-a"}, infrastructure.InfrastructureRequirements{})

	repo := &mockGraphServiceRepo{
		services: map[service.ServiceName]*service.Service{
			"service-a": svcA,
			"service-b": svcB,
			"service-c": svcC,
		},
	}

	registry := registryFromNames("service-a", "service-b", "service-c")
	handler := NewGraphQueryHandler(repo, registry)
	result, err := handler.Handle(GraphQuery{ServiceNames: []string{"service-a"}})
	if err != nil {
		t.Fatalf("Handle() should not error on circular dependency, got: %v", err)
	}

	// All 3 nodes should be present
	if len(result.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(result.Nodes))
	}

	// Edges should be preserved
	nodeA := result.Nodes["service-a"]
	if len(nodeA.Dependencies) != 1 || nodeA.Dependencies[0] != "service-b" {
		t.Errorf("service-a deps = %v, want [service-b]", nodeA.Dependencies)
	}
}
