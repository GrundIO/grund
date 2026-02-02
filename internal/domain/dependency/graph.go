package dependency

import (
	"fmt"
	"strings"

	"github.com/Saturn-Fintech/grund/internal/domain/service"
)

// Graph represents the dependency graph of services
type Graph struct {
	nodes map[service.ServiceName]*Node
}

// Node represents a service node in the dependency graph
type Node struct {
	Service      *service.Service
	Dependencies []service.ServiceName
	Dependents   []service.ServiceName
}

// NewGraph creates a new dependency graph
func NewGraph() *Graph {
	return &Graph{
		nodes: make(map[service.ServiceName]*Node),
	}
}

// AddService adds a service to the graph
func (g *Graph) AddService(svc *service.Service) {
	node := &Node{
		Service:      svc,
		Dependencies: svc.Dependencies.Services,
		Dependents:   []service.ServiceName{},
	}
	g.nodes[service.ServiceName(svc.Name)] = node
}

// Build builds the complete dependency graph
func (g *Graph) Build() error {
	// Build dependents list
	for name, node := range g.nodes {
		for _, dep := range node.Dependencies {
			if depNode, ok := g.nodes[dep]; ok {
				depNode.Dependents = append(depNode.Dependents, name)
			} else {
				return fmt.Errorf("dependency %s not found in graph", dep)
			}
		}
	}
	return nil
}

// GetNode returns a node by service name
func (g *Graph) GetNode(name service.ServiceName) (*Node, error) {
	node, ok := g.nodes[name]
	if !ok {
		return nil, fmt.Errorf("node %s not found in graph", name)
	}
	return node, nil
}

// GetAllDependencies returns all transitive dependencies for a service
func (g *Graph) GetAllDependencies(serviceName service.ServiceName) ([]service.ServiceName, error) {
	visited := make(map[service.ServiceName]bool)
	var deps []service.ServiceName

	var collectDeps func(service.ServiceName) error
	collectDeps = func(name service.ServiceName) error {
		if visited[name] {
			return nil
		}
		visited[name] = true

		node, err := g.GetNode(name)
		if err != nil {
			return err
		}

		for _, dep := range node.Dependencies {
			deps = append(deps, dep)
			if err := collectDeps(dep); err != nil {
				return err
			}
		}

		return nil
	}

	if err := collectDeps(serviceName); err != nil {
		return nil, err
	}

	return deps, nil
}

// DetectCycle detects circular dependencies
func (g *Graph) DetectCycle(startService service.ServiceName) ([]service.ServiceName, error) {
	visited := make(map[service.ServiceName]bool)
	recStack := make(map[service.ServiceName]bool)
	var cycle []service.ServiceName

	var dfs func(service.ServiceName, []service.ServiceName) error
	dfs = func(name service.ServiceName, path []service.ServiceName) error {
		visited[name] = true
		recStack[name] = true
		path = append(path, name)

		node, err := g.GetNode(name)
		if err != nil {
			return err
		}

		for _, dep := range node.Dependencies {
			if !visited[dep] {
				if err := dfs(dep, path); err != nil {
					return err
				}
			} else if recStack[dep] {
				// Found cycle
				cycleStart := -1
				for i, s := range path {
					if s == dep {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					cycle = append(path[cycleStart:], dep)
					names := make([]string, len(cycle))
					for i, n := range cycle {
						names[i] = n.String()
					}
					return fmt.Errorf("circular dependency detected: %s", strings.Join(names, " → "))
				}
			}
		}

		recStack[name] = false
		return nil
	}

	if err := dfs(startService, []service.ServiceName{}); err != nil {
		return cycle, err
	}

	return nil, nil
}

// TopologicalSort performs topological sort to determine startup order
func (g *Graph) TopologicalSort(startServices []service.ServiceName) ([]service.ServiceName, error) {
	// Get all dependencies
	allServices := make(map[service.ServiceName]bool)
	for _, svc := range startServices {
		deps, err := g.GetAllDependencies(svc)
		if err != nil {
			return nil, err
		}
		for _, dep := range deps {
			allServices[dep] = true
		}
		allServices[svc] = true
	}

	// Convert to list
	servicesList := make([]service.ServiceName, 0, len(allServices))
	for svc := range allServices {
		servicesList = append(servicesList, svc)
	}

	// Calculate in-degrees
	inDegree := make(map[service.ServiceName]int)
	for _, svc := range servicesList {
		inDegree[svc] = 0
	}

	for _, svc := range servicesList {
		node, err := g.GetNode(svc)
		if err != nil {
			continue
		}
		for _, dep := range node.Dependencies {
			if _, ok := allServices[dep]; ok {
				inDegree[svc]++
			}
		}
	}

	// Kahn's algorithm
	var queue []service.ServiceName
	for _, svc := range servicesList {
		if inDegree[svc] == 0 {
			queue = append(queue, svc)
		}
	}

	var result []service.ServiceName
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		node, err := g.GetNode(current)
		if err != nil {
			continue
		}

		for _, dependent := range node.Dependents {
			if _, ok := allServices[dependent]; ok {
				inDegree[dependent]--
				if inDegree[dependent] == 0 {
					queue = append(queue, dependent)
				}
			}
		}
	}

	// Check for cycles
	if len(result) != len(servicesList) {
		return nil, fmt.Errorf("circular dependency detected")
	}

	return result, nil
}
