package configcmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
	"github.com/Saturn-Fintech/grund/internal/application/queries"
	"github.com/Saturn-Fintech/grund/internal/cli/shared"
	"github.com/Saturn-Fintech/grund/internal/config"
	"github.com/Saturn-Fintech/grund/internal/domain/service"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show [service]",
	Short: "Show Grund configuration",
	Long: `Show Grund configuration and setup.

Without arguments: Shows overall setup (config file, registered services)
With service name: Shows resolved configuration for that service

Examples:
  grund config show              Show setup overview
  grund config show user-service Show service configuration`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return showSetupInfo()
		}
		return showServiceConfig(args[0])
	},
}

func init() {
	Cmd.AddCommand(showCmd)
}

func showSetupInfo() error {
	configResolver := shared.ConfigResolver
	if configResolver == nil {
		return fmt.Errorf("config not initialized")
	}

	// Get config file path
	servicesPath, orchestrationRoot, err := configResolver.ResolveServicesFile()
	if err != nil {
		return err
	}

	// Get global config path
	globalConfigPath, _ := config.GetGlobalConfigPath()

	fmt.Println()
	fmt.Println("  Grund Configuration")
	fmt.Println("  " + strings.Repeat("─", 40))
	fmt.Println()
	fmt.Printf("  Services file:      %s\n", servicesPath)
	fmt.Printf("  Orchestration root: %s\n", orchestrationRoot)
	fmt.Printf("  Global config:      %s\n", globalConfigPath)
	fmt.Println()

	// Show global config values
	fmt.Println("  Global Settings")
	fmt.Println("  " + strings.Repeat("─", 40))

	gc := configResolver.GlobalConfig

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)
	t.AppendHeader(table.Row{"Setting", "Value", "Source"})

	// Docker compose command
	dockerCmd := gc.GetDockerComposeCommand()
	dockerSource := "default"
	if gc.Docker != nil && gc.Docker.ComposeCommand != "" {
		dockerSource = "config"
	}
	t.AppendRow(table.Row{"docker.compose_command", dockerCmd, dockerSource})

	// LocalStack endpoint
	lsEndpoint := gc.GetLocalStackEndpoint()
	lsEndpointSource := "default"
	if gc.LocalStack != nil && gc.LocalStack.Endpoint != "" {
		lsEndpointSource = "config"
	}
	t.AppendRow(table.Row{"localstack.endpoint", lsEndpoint, lsEndpointSource})

	// LocalStack region
	lsRegion := gc.GetLocalStackRegion()
	lsRegionSource := "default"
	if gc.LocalStack != nil && gc.LocalStack.Region != "" {
		lsRegionSource = "config"
	}
	t.AppendRow(table.Row{"localstack.region", lsRegion, lsRegionSource})

	t.Render()
	fmt.Println()

	// Get registered services
	container := shared.Container
	if container != nil && container.RegistryRepo != nil {
		if registryRepo, ok := container.RegistryRepo.(ports.ServiceRegistryRepository); ok {
			services, err := registryRepo.GetAllServices()
			if err == nil && len(services) > 0 {
				fmt.Println("  Registered Services")
				fmt.Println("  " + strings.Repeat("─", 40))

				t := table.NewWriter()
				t.SetOutputMirror(os.Stdout)
				t.SetStyle(table.StyleRounded)
				t.AppendHeader(table.Row{"Service", "Path"})

				// Sort service names
				names := make([]string, 0, len(services))
				for name := range services {
					names = append(names, string(name))
				}
				sort.Strings(names)

				for _, name := range names {
					svc := services[service.ServiceName(name)]
					t.AppendRow(table.Row{name, svc.Path})
				}

				t.Render()
				fmt.Println()
			}
		}
	}

	fmt.Println("  Use 'grund config show <service>' to see service details")
	fmt.Println()

	return nil
}

func showServiceConfig(serviceName string) error {
	container := shared.Container
	if container == nil {
		return fmt.Errorf("container not initialized")
	}

	query := queries.ConfigQuery{ServiceName: serviceName}
	cfg, err := container.ConfigQueryHandler.Handle(query)
	if err != nil {
		return err
	}

	// Service info
	fmt.Printf("\n  Service: %s\n", cfg.Service.Name)
	fmt.Printf("  Type:    %s\n", cfg.Service.Type)
	fmt.Printf("  Port:    %d\n", cfg.Service.Port.Value())
	fmt.Println()

	// Dependencies
	if len(cfg.Dependencies) > 0 {
		fmt.Printf("  Dependencies: %s\n", strings.Join(cfg.Dependencies, ", "))
	} else {
		fmt.Println("  Dependencies: none")
	}

	// Infrastructure
	if len(cfg.Infrastructure) > 0 {
		fmt.Printf("  Infrastructure: %s\n", strings.Join(cfg.Infrastructure, ", "))
	} else {
		fmt.Println("  Infrastructure: none")
	}
	fmt.Println()

	// Environment variables table
	if len(cfg.Environment) > 0 {
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.SetStyle(table.StyleRounded)
		t.AppendHeader(table.Row{"Environment Variable", "Value"})

		// Sort keys for consistent output
		keys := make([]string, 0, len(cfg.Environment))
		for k := range cfg.Environment {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			v := cfg.Environment[k]
			// Truncate long values
			if len(v) > 60 {
				v = v[:57] + "..."
			}
			t.AppendRow(table.Row{k, v})
		}

		t.Render()
	} else {
		fmt.Println("  No environment variables configured")
	}
	fmt.Println()

	return nil
}
