package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/Saturn-Fintech/grund/internal/application/queries"
	"github.com/Saturn-Fintech/grund/internal/cli/shared"
	"github.com/Saturn-Fintech/grund/internal/ui"
	"github.com/goccy/go-graphviz"
	"github.com/spf13/cobra"
)

var (
	graphShowInfra bool
	graphFormat    string
	graphOutput    string
)

var graphCmd = &cobra.Command{
	Use:   "graph [services...]",
	Short: "Visualize service dependency graph",
	Long: `Render the service dependency graph as an image or DOT text.

Supported output formats: svg (default), dot.
When using svg, specify an output file with --output.

Examples:
  grund graph                                 Render graph to graph.svg
  grund graph --output deps.svg               Render to custom file
  grund graph --format dot                    Print DOT to stdout
  grund graph --infra                         Include infrastructure nodes
  grund graph user-service                    Graph for a specific service`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if shared.Container == nil {
			return fmt.Errorf("container not initialized")
		}

		query := queries.GraphQuery{
			ServiceNames: args,
			ShowInfra:    graphShowInfra,
		}

		result, err := shared.Container.GraphQueryHandler.Handle(query)
		if err != nil {
			return err
		}

		if len(result.Nodes) == 0 {
			ui.Infof("No services found. Run 'grund up <service>' to start services.")
			return nil
		}

		switch graphFormat {
		case "dot":
			printDot(result)
		default:
			return renderGraph(cmd.Context(), result, withDefaultOutput("graph.svg"))
		}

		return nil
	},
}

func init() {
	graphCmd.Flags().BoolVar(&graphShowInfra, "infra", false, "Show infrastructure dependencies (postgres, redis, etc.)")
	graphCmd.Flags().StringVar(&graphFormat, "format", "svg", "Output format: svg (default) or dot")
	graphCmd.Flags().StringVarP(&graphOutput, "output", "o", "", "Output file path (default: graph.<format>)")
}

// withDefaultOutput returns graphOutput if set, otherwise the fallback
func withDefaultOutput(fallback string) string {
	if graphOutput != "" {
		return graphOutput
	}
	return fallback
}

// renderGraph builds a Graphviz graph from the query result and renders it to a file
func renderGraph(ctx context.Context, result *queries.GraphResult, outputPath string) error {
	gv, err := graphviz.New(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize graphviz: %w", err)
	}
	defer gv.Close()

	graph, err := gv.Graph(
		graphviz.WithDirectedType(graphviz.Directed),
		graphviz.WithName("dependencies"),
	)
	if err != nil {
		return fmt.Errorf("failed to create graph: %w", err)
	}
	defer graph.Close()

	graph.SetRankDir(graphviz.LRRank)
	graph.SetBackgroundColor("white")
	graph.SetNodeSeparator(0.8)
	graph.SetRankSeparator(1.0)

	// Sort names for deterministic layout
	names := sortedNodeNames(result)

	// Create all service nodes
	gvNodes := make(map[string]*graphviz.Node)
	for _, name := range names {
		node, err := graph.CreateNodeByName(name)
		if err != nil {
			return fmt.Errorf("failed to create node %s: %w", name, err)
		}
		node.SetShape(graphviz.BoxShape)
		node.SetStyle(graphviz.NodeStyle("filled,rounded"))
		node.SetFillColor("#E8F4FD")
		node.SetColor("#4A90D9")
		node.SetFontName("Helvetica")
		node.SetFontSize(11)
		node.SetFontColor("#2D3748")
		node.SetMargin(0.15)
		node.SetPenWidth(1.5)

		gvNodes[name] = node
	}

	// Create infrastructure nodes (deduplicated across services)
	infraNodes := make(map[string]*graphviz.Node)
	for _, name := range names {
		graphNode := result.Nodes[name]
		for _, infra := range graphNode.Infrastructure {
			if _, exists := infraNodes[infra]; exists {
				continue
			}
			node, err := graph.CreateNodeByName(infra)
			if err != nil {
				return fmt.Errorf("failed to create infra node %s: %w", infra, err)
			}
			node.SetShape(graphviz.CylinderShape)
			node.SetStyle(graphviz.FilledNodeStyle)
			node.SetFillColor("#FEF3C7")
			node.SetColor("#D97706")
			node.SetFontName("Helvetica")
			node.SetFontSize(10)
			node.SetFontColor("#92400E")
			node.SetPenWidth(1.5)

			infraNodes[infra] = node
		}
	}

	// Create service-to-service edges with bidirectional detection.
	// When A→B and B→A both exist, draw a single bidirectional edge.
	type edgePair struct{ from, to string }
	created := make(map[edgePair]bool)
	edgeIndex := 0

	for _, name := range names {
		graphNode := result.Nodes[name]
		for _, dep := range graphNode.Dependencies {
			if _, ok := gvNodes[dep]; !ok {
				continue
			}

			if created[edgePair{dep, name}] {
				continue
			}

			edgeName := fmt.Sprintf("e%d", edgeIndex)
			edgeIndex++
			edge, err := graph.CreateEdgeByName(edgeName, gvNodes[name], gvNodes[dep])
			if err != nil {
				return fmt.Errorf("failed to create edge %s -> %s: %w", name, dep, err)
			}
			edge.SetColor("#718096")
			edge.SetPenWidth(1.2)

			if depNode, ok := result.Nodes[dep]; ok {
				for _, revDep := range depNode.Dependencies {
					if revDep == name {
						edge.SetDir(graphviz.BothDir)
						break
					}
				}
			}

			created[edgePair{name, dep}] = true
		}

		// Create service-to-infrastructure edges
		for _, infra := range graphNode.Infrastructure {
			edgeName := fmt.Sprintf("e%d", edgeIndex)
			edgeIndex++
			edge, err := graph.CreateEdgeByName(edgeName, gvNodes[name], infraNodes[infra])
			if err != nil {
				return fmt.Errorf("failed to create infra edge %s -> %s: %w", name, infra, err)
			}
			edge.SetColor("#D97706")
			edge.SetStyle(graphviz.DashedEdgeStyle)
			edge.SetPenWidth(1.0)
		}
	}

	gv.SetLayout(graphviz.DOT)

	// Render to buffer first, then write to file.
	// RenderFilename with SVG crashes in the WASM runtime,
	// so we use Render to a buffer as a workaround.
	var buf bytes.Buffer
	if err := gv.Render(ctx, graph, graphviz.SVG, &buf); err != nil {
		return fmt.Errorf("failed to render graph: %w", err)
	}

	if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outputPath, err)
	}

	ui.Successf("Graph written to %s", outputPath)

	return nil
}

// printDot renders the graph in Graphviz DOT format to stdout
func printDot(result *queries.GraphResult) {
	fmt.Println("digraph dependencies {")
	fmt.Println("  rankdir=LR;")
	fmt.Println("  nodesep=0.8;")
	fmt.Println("  ranksep=1.0;")
	fmt.Println("  node [shape=box, style=\"filled,rounded\", fillcolor=\"#E8F4FD\", color=\"#4A90D9\", fontname=\"Helvetica\", fontsize=11, fontcolor=\"#2D3748\", margin=0.15, penwidth=1.5];")
	fmt.Println("  edge [color=\"#718096\", penwidth=1.2];")

	names := sortedNodeNames(result)

	// Declare infrastructure nodes
	infraSeen := make(map[string]bool)
	for _, name := range names {
		for _, infra := range result.Nodes[name].Infrastructure {
			if infraSeen[infra] {
				continue
			}
			infraSeen[infra] = true
			fmt.Printf("  %q [shape=cylinder, style=filled, fillcolor=\"#FEF3C7\", color=\"#D97706\", fontsize=10, fontcolor=\"#92400E\", penwidth=1.5];\n", infra)
		}
	}

	type edgePair struct{ from, to string }
	created := make(map[edgePair]bool)

	for _, name := range names {
		node := result.Nodes[name]

		// Service-to-service edges
		for _, dep := range node.Dependencies {
			if created[edgePair{dep, name}] {
				continue
			}

			isMutual := false
			if depNode, ok := result.Nodes[dep]; ok {
				for _, revDep := range depNode.Dependencies {
					if revDep == name {
						isMutual = true
						break
					}
				}
			}

			if isMutual {
				fmt.Printf("  %q -> %q [dir=both];\n", name, dep)
			} else {
				fmt.Printf("  %q -> %q;\n", name, dep)
			}
			created[edgePair{name, dep}] = true
		}

		// Service-to-infrastructure edges
		for _, infra := range node.Infrastructure {
			fmt.Printf("  %q -> %q [color=\"#D97706\", style=dashed, penwidth=1.0];\n", name, infra)
		}

		// Isolated nodes still need to appear
		if len(node.Dependencies) == 0 && len(node.Dependents) == 0 && len(node.Infrastructure) == 0 {
			fmt.Printf("  %q;\n", name)
		}
	}

	fmt.Println("}")
}

// sortedNodeNames returns node names in sorted order for deterministic output
func sortedNodeNames(result *queries.GraphResult) []string {
	names := make([]string, 0, len(result.Nodes))
	for name := range result.Nodes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
