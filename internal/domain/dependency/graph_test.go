package dependency

import (
	"strings"
	"testing"
	"time"

	"github.com/Saturn-Fintech/grund/internal/domain/infrastructure"
	"github.com/Saturn-Fintech/grund/internal/domain/service"
)

// Helper function to create a test service
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

func TestGraph_AddService(t *testing.T) {
	graph := NewGraph()
	svc := createTestService("service-a", []string{})

	graph.AddService(svc)

	node, err := graph.GetNode(service.ServiceName("service-a"))
	if err != nil {
		t.Fatalf("GetNode() returned error: %v", err)
	}
	if node.Service.Name != "service-a" {
		t.Errorf("node.Service.Name = %q, want %q", node.Service.Name, "service-a")
	}
}

func TestGraph_GetNode_NotFound(t *testing.T) {
	graph := NewGraph()

	_, err := graph.GetNode(service.ServiceName("nonexistent"))
	if err == nil {
		t.Error("GetNode() expected error for nonexistent node, got nil")
	}
}

func TestGraph_Build_Success(t *testing.T) {
	graph := NewGraph()

	// Create a simple dependency chain: A -> B -> C
	svcC := createTestService("service-c", []string{})
	svcB := createTestService("service-b", []string{"service-c"})
	svcA := createTestService("service-a", []string{"service-b"})

	graph.AddService(svcC)
	graph.AddService(svcB)
	graph.AddService(svcA)

	err := graph.Build()
	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}

	// Check that dependents are set correctly
	nodeC, _ := graph.GetNode(service.ServiceName("service-c"))
	nodeB, _ := graph.GetNode(service.ServiceName("service-b"))

	// service-c should have service-b as dependent
	if len(nodeC.Dependents) != 1 || nodeC.Dependents[0] != service.ServiceName("service-b") {
		t.Errorf("service-c dependents = %v, want [service-b]", nodeC.Dependents)
	}

	// service-b should have service-a as dependent
	if len(nodeB.Dependents) != 1 || nodeB.Dependents[0] != service.ServiceName("service-a") {
		t.Errorf("service-b dependents = %v, want [service-a]", nodeB.Dependents)
	}
}

func TestGraph_Build_MissingDependency(t *testing.T) {
	graph := NewGraph()

	// service-a depends on service-b, but service-b is not in the graph
	svcA := createTestService("service-a", []string{"service-b"})
	graph.AddService(svcA)

	err := graph.Build()
	if err == nil {
		t.Error("Build() expected error for missing dependency, got nil")
	}
}

func TestGraph_GetAllDependencies(t *testing.T) {
	graph := NewGraph()

	// Create a dependency chain: A -> B -> C
	svcC := createTestService("service-c", []string{})
	svcB := createTestService("service-b", []string{"service-c"})
	svcA := createTestService("service-a", []string{"service-b"})

	graph.AddService(svcC)
	graph.AddService(svcB)
	graph.AddService(svcA)
	graph.Build()

	deps, err := graph.GetAllDependencies(service.ServiceName("service-a"))
	if err != nil {
		t.Fatalf("GetAllDependencies() returned error: %v", err)
	}

	// service-a should depend on service-b and service-c (transitively)
	if len(deps) != 2 {
		t.Errorf("GetAllDependencies() returned %d deps, want 2", len(deps))
	}

	depMap := make(map[service.ServiceName]bool)
	for _, d := range deps {
		depMap[d] = true
	}

	if !depMap[service.ServiceName("service-b")] {
		t.Error("GetAllDependencies() missing service-b")
	}
	if !depMap[service.ServiceName("service-c")] {
		t.Error("GetAllDependencies() missing service-c")
	}
}

func TestGraph_GetAllDependencies_NoDeps(t *testing.T) {
	graph := NewGraph()

	svcA := createTestService("service-a", []string{})
	graph.AddService(svcA)
	graph.Build()

	deps, err := graph.GetAllDependencies(service.ServiceName("service-a"))
	if err != nil {
		t.Fatalf("GetAllDependencies() returned error: %v", err)
	}

	if len(deps) != 0 {
		t.Errorf("GetAllDependencies() returned %d deps, want 0", len(deps))
	}
}

func TestGraph_DetectCycle_NoCycle(t *testing.T) {
	graph := NewGraph()

	// Linear dependency chain: A -> B -> C (no cycle)
	svcC := createTestService("service-c", []string{})
	svcB := createTestService("service-b", []string{"service-c"})
	svcA := createTestService("service-a", []string{"service-b"})

	graph.AddService(svcC)
	graph.AddService(svcB)
	graph.AddService(svcA)
	graph.Build()

	cycle, err := graph.DetectCycle(service.ServiceName("service-a"))
	if err != nil {
		t.Errorf("DetectCycle() returned error for non-cyclic graph: %v", err)
	}
	if cycle != nil {
		t.Errorf("DetectCycle() returned cycle %v for non-cyclic graph", cycle)
	}
}

func TestGraph_DetectCycle_WithCycle(t *testing.T) {
	graph := NewGraph()

	// Create a cycle: A -> B -> C -> A
	svcC := createTestService("service-c", []string{"service-a"})
	svcB := createTestService("service-b", []string{"service-c"})
	svcA := createTestService("service-a", []string{"service-b"})

	graph.AddService(svcC)
	graph.AddService(svcB)
	graph.AddService(svcA)
	// Note: Build() will succeed because it only builds dependents list
	// Cycle detection is separate

	_, err := graph.DetectCycle(service.ServiceName("service-a"))
	if err == nil {
		t.Error("DetectCycle() expected error for cyclic graph, got nil")
	}
	if !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("DetectCycle() error = %q, want to contain 'circular dependency'", err.Error())
	}
}

func TestGraph_TopologicalSort(t *testing.T) {
	graph := NewGraph()

	// Create a dependency chain: A -> B -> C
	// Expected startup order: C, B, A
	svcC := createTestService("service-c", []string{})
	svcB := createTestService("service-b", []string{"service-c"})
	svcA := createTestService("service-a", []string{"service-b"})

	graph.AddService(svcC)
	graph.AddService(svcB)
	graph.AddService(svcA)
	graph.Build()

	order, err := graph.TopologicalSort([]service.ServiceName{service.ServiceName("service-a")})
	if err != nil {
		t.Fatalf("TopologicalSort() returned error: %v", err)
	}

	if len(order) != 3 {
		t.Fatalf("TopologicalSort() returned %d services, want 3", len(order))
	}

	// service-c must come before service-b, and service-b must come before service-a
	indexMap := make(map[service.ServiceName]int)
	for i, name := range order {
		indexMap[name] = i
	}

	if indexMap[service.ServiceName("service-c")] >= indexMap[service.ServiceName("service-b")] {
		t.Error("TopologicalSort() service-c should come before service-b")
	}
	if indexMap[service.ServiceName("service-b")] >= indexMap[service.ServiceName("service-a")] {
		t.Error("TopologicalSort() service-b should come before service-a")
	}
}

func TestGraph_TopologicalSort_MultipleStartServices(t *testing.T) {
	graph := NewGraph()

	// Two independent services with shared dependency
	//   A -> C
	//   B -> C
	svcC := createTestService("service-c", []string{})
	svcB := createTestService("service-b", []string{"service-c"})
	svcA := createTestService("service-a", []string{"service-c"})

	graph.AddService(svcC)
	graph.AddService(svcB)
	graph.AddService(svcA)
	graph.Build()

	order, err := graph.TopologicalSort([]service.ServiceName{
		service.ServiceName("service-a"),
		service.ServiceName("service-b"),
	})
	if err != nil {
		t.Fatalf("TopologicalSort() returned error: %v", err)
	}

	if len(order) != 3 {
		t.Fatalf("TopologicalSort() returned %d services, want 3", len(order))
	}

	// service-c must come first
	if order[0] != service.ServiceName("service-c") {
		t.Errorf("TopologicalSort() first service = %q, want service-c", order[0])
	}
}

func TestGraph_TopologicalSort_WithCycle(t *testing.T) {
	graph := NewGraph()

	// Create a cycle: A -> B -> C -> A
	svcC := createTestService("service-c", []string{"service-a"})
	svcB := createTestService("service-b", []string{"service-c"})
	svcA := createTestService("service-a", []string{"service-b"})

	graph.AddService(svcC)
	graph.AddService(svcB)
	graph.AddService(svcA)

	_, err := graph.TopologicalSort([]service.ServiceName{service.ServiceName("service-a")})
	if err == nil {
		t.Error("TopologicalSort() expected error for cyclic graph, got nil")
	}
}

func TestGraph_TopologicalSort_SingleService(t *testing.T) {
	graph := NewGraph()

	svcA := createTestService("service-a", []string{})
	graph.AddService(svcA)
	graph.Build()

	order, err := graph.TopologicalSort([]service.ServiceName{service.ServiceName("service-a")})
	if err != nil {
		t.Fatalf("TopologicalSort() returned error: %v", err)
	}

	if len(order) != 1 {
		t.Fatalf("TopologicalSort() returned %d services, want 1", len(order))
	}
	if order[0] != service.ServiceName("service-a") {
		t.Errorf("TopologicalSort() returned %q, want service-a", order[0])
	}
}
