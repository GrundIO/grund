package queries

import (
	"fmt"
	"sort"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
	"github.com/Saturn-Fintech/grund/internal/domain/dependency"
	"github.com/Saturn-Fintech/grund/internal/domain/service"
	"github.com/Saturn-Fintech/grund/internal/ui"
)

// GraphQuery represents a query for the service dependency graph
type GraphQuery struct {
	ServiceNames []string
	ShowInfra    bool
}

// GraphNode represents a node in the query result
type GraphNode struct {
	Name           string
	Dependencies   []string
	Dependents     []string
	Infrastructure []string
}

// GraphResult holds the dependency graph query result
type GraphResult struct {
	Nodes map[string]GraphNode
}

// GraphQueryHandler handles graph queries
type GraphQueryHandler struct {
	serviceRepo  ports.ServiceRepository
	registryRepo ports.ServiceRegistryRepository
}

// NewGraphQueryHandler creates a new graph query handler
func NewGraphQueryHandler(
	serviceRepo ports.ServiceRepository,
	registryRepo ports.ServiceRegistryRepository,
) *GraphQueryHandler {
	return &GraphQueryHandler{
		serviceRepo:  serviceRepo,
		registryRepo: registryRepo,
	}
}

// Handle executes the graph query
func (h *GraphQueryHandler) Handle(query GraphQuery) (*GraphResult, error) {
	services, err := h.loadServices(query.ServiceNames)
	if err != nil {
		return nil, fmt.Errorf("failed to load services: %w", err)
	}

	if len(services) == 0 {
		return &GraphResult{
			Nodes: make(map[string]GraphNode),
		}, nil
	}

	// Build a set of loaded service names so we can strip references
	// to services whose grund.yaml couldn't be loaded
	loadedSet := make(map[service.ServiceName]bool, len(services))
	for _, svc := range services {
		loadedSet[service.ServiceName(svc.Name)] = true
	}

	// Strip dependency references that point to unloaded services
	// so graph.Build() doesn't fail on missing nodes
	for _, svc := range services {
		var filtered []service.ServiceName
		for _, dep := range svc.Dependencies.Services {
			if loadedSet[dep] {
				filtered = append(filtered, dep)
			}
		}
		svc.Dependencies.Services = filtered
	}

	// Build the dependency graph
	graph := dependency.NewGraph()
	for _, svc := range services {
		graph.AddService(svc)
	}
	if err := graph.Build(); err != nil {
		return nil, fmt.Errorf("failed to build dependency graph: %w", err)
	}

	// Extract nodes into result
	nodes := make(map[string]GraphNode)
	allNodes := graph.GetAllNodes()
	for name, node := range allNodes {
		gn := GraphNode{
			Name: name.String(),
		}
		for _, dep := range node.Dependencies {
			gn.Dependencies = append(gn.Dependencies, dep.String())
		}
		sort.Strings(gn.Dependencies)

		for _, dep := range node.Dependents {
			gn.Dependents = append(gn.Dependents, dep.String())
		}
		sort.Strings(gn.Dependents)

		if query.ShowInfra {
			gn.Infrastructure = collectInfraResources(node.Service)
		}

		nodes[name.String()] = gn
	}

	return &GraphResult{
		Nodes: nodes,
	}, nil
}

// loadServices loads requested services and their transitive dependencies.
// Services whose grund.yaml is missing are skipped with a warning.
func (h *GraphQueryHandler) loadServices(names []string) ([]*service.Service, error) {
	if len(names) == 0 {
		return h.loadAllServices()
	}

	loaded := make(map[string]*service.Service)
	skipped := make(map[string]bool)

	var loadRecursive func(name string)
	loadRecursive = func(name string) {
		if _, exists := loaded[name]; exists {
			return
		}
		if skipped[name] {
			return
		}

		svc, err := h.serviceRepo.FindByName(service.ServiceName(name))
		if err != nil {
			ui.Warnf("Skipping %s: %v", name, err)
			skipped[name] = true
			return
		}

		loaded[name] = svc

		for _, dep := range svc.Dependencies.Services {
			loadRecursive(dep.String())
		}
	}

	for _, name := range names {
		loadRecursive(name)
	}

	services := make([]*service.Service, 0, len(loaded))
	for _, svc := range loaded {
		services = append(services, svc)
	}
	return services, nil
}

// loadAllServices loads all registered services, skipping any whose grund.yaml is missing
func (h *GraphQueryHandler) loadAllServices() ([]*service.Service, error) {
	entries, err := h.registryRepo.GetAllServices()
	if err != nil {
		return nil, fmt.Errorf("failed to get service registry: %w", err)
	}

	var services []*service.Service
	for name := range entries {
		svc, err := h.serviceRepo.FindByName(name)
		if err != nil {
			ui.Warnf("Skipping %s: %v", name, err)
			continue
		}
		services = append(services, svc)
	}
	return services, nil
}

// collectInfraResources returns specific infrastructure resource identifiers.
// Each entry is "type:name" (e.g. "postgres:mydb", "sqs:orders-queue")
// so that shared resources across services map to the same graph node.
func collectInfraResources(svc *service.Service) []string {
	infra := svc.Dependencies.Infrastructure
	var resources []string

	if infra.Postgres != nil {
		resources = append(resources, "postgres:"+infra.Postgres.Database)
	}
	if infra.MongoDB != nil {
		resources = append(resources, "mongodb:"+infra.MongoDB.Database)
	}
	if infra.Redis != nil {
		resources = append(resources, "redis")
	}
	if infra.SQS != nil {
		for _, q := range infra.SQS.Queues {
			resources = append(resources, "sqs:"+q.Name)
		}
	}
	if infra.SNS != nil {
		for _, t := range infra.SNS.Topics {
			resources = append(resources, "sns:"+t.Name)
		}
	}
	if infra.S3 != nil {
		for _, b := range infra.S3.Buckets {
			resources = append(resources, "s3:"+b.Name)
		}
	}
	return resources
}
