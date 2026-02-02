package resolver

import (
	"fmt"

	"github.com/Saturn-Fintech/grund/internal/config"
)

// DependencyGraph represents the dependency graph of services
type DependencyGraph struct {
	Nodes map[string]*Node
}

// Node represents a service node in the dependency graph
type Node struct {
	Name         string
	Config       *config.ServiceConfig
	Dependencies []string // Service names this depends on
	Dependents   []string // Services that depend on this
}

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		Nodes: make(map[string]*Node),
	}
}

// AddNode adds a service node to the graph
func (g *DependencyGraph) AddNode(name string, serviceConfig *config.ServiceConfig) {
	node := &Node{
		Name:         name,
		Config:       serviceConfig,
		Dependencies: serviceConfig.Requires.Services,
		Dependents:   []string{},
	}
	g.Nodes[name] = node
}

// BuildGraph builds the complete dependency graph from service configs
func (g *DependencyGraph) BuildGraph() error {
	// Build dependents list
	for name, node := range g.Nodes {
		for _, dep := range node.Dependencies {
			if depNode, ok := g.Nodes[dep]; ok {
				depNode.Dependents = append(depNode.Dependents, name)
			}
		}
	}

	return nil
}

// GetNode returns a node by name
func (g *DependencyGraph) GetNode(name string) (*Node, error) {
	node, ok := g.Nodes[name]
	if !ok {
		return nil, fmt.Errorf("node %s not found in graph", name)
	}
	return node, nil
}

// GetAllDependencies returns all dependencies (transitive) for a service
func (g *DependencyGraph) GetAllDependencies(serviceName string) ([]string, error) {
	visited := make(map[string]bool)
	var deps []string

	var collectDeps func(string) error
	collectDeps = func(name string) error {
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
