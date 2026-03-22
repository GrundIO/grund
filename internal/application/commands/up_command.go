package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/GrundIO/grund/internal/application/ports"
	"github.com/GrundIO/grund/internal/config"
	"github.com/GrundIO/grund/internal/domain/infrastructure"
	"github.com/GrundIO/grund/internal/domain/service"
	"github.com/GrundIO/grund/internal/ui"
)

// UpCommand represents the command to start services
type UpCommand struct {
	ServiceNames []string
	NoDeps       bool
	InfraOnly    bool
	Build        bool
	Local        bool
}

// UpCommandHandler handles the up command
// This follows the Command pattern and Single Responsibility Principle
type UpCommandHandler struct {
	serviceRepo      ports.ServiceRepository
	registryRepo     ports.ServiceRegistryRepository
	orchestrator     ports.ContainerOrchestrator
	provisioner      ports.InfrastructureProvisioner
	composeGenerator ports.ComposeGenerator
	healthChecker    ports.HealthChecker
	tunnelManager    ports.TunnelManager // optional, can be nil
	hookExecutor     ports.HookExecutor
}

// NewUpCommandHandler creates a new up command handler
func NewUpCommandHandler(
	serviceRepo ports.ServiceRepository,
	registryRepo ports.ServiceRegistryRepository,
	orchestrator ports.ContainerOrchestrator,
	provisioner ports.InfrastructureProvisioner,
	composeGenerator ports.ComposeGenerator,
	healthChecker ports.HealthChecker,
	tunnelManager ports.TunnelManager,
	hookExecutor ports.HookExecutor,
) *UpCommandHandler {
	return &UpCommandHandler{
		serviceRepo:      serviceRepo,
		registryRepo:     registryRepo,
		orchestrator:     orchestrator,
		provisioner:      provisioner,
		composeGenerator: composeGenerator,
		healthChecker:    healthChecker,
		tunnelManager:    tunnelManager,
		hookExecutor:     hookExecutor,
	}
}

// Handle executes the up command
func (h *UpCommandHandler) Handle(ctx context.Context, cmd UpCommand) error {
	ui.Infof("Starting services: %s", strings.Join(cmd.ServiceNames, ", "))

	// 1. Load services (and their transitive dependencies unless --no-deps)
	ui.Step("Loading service configurations...")
	var services []*service.Service
	var err error
	if cmd.NoDeps {
		services, err = h.loadServices(cmd.ServiceNames)
	} else {
		services, err = h.loadServicesWithDependencies(cmd.ServiceNames)
	}
	if err != nil {
		return fmt.Errorf("failed to load services: %w", err)
	}
	ui.Debug("Loaded %d service(s)", len(services))

	// Run pre_up hooks (before anything starts)
	if err := h.runHooks(ctx, services, service.HookStagePreUp); err != nil {
		return err
	}

	// 2. Build service names list from all loaded services
	// Note: We no longer enforce startup order between services.
	// Services start in parallel after infrastructure is ready.
	// They should handle reconnection to other services themselves.
	serviceNames := make([]service.ServiceName, len(services))
	for i, svc := range services {
		serviceNames[i] = service.ServiceName(svc.Name)
	}
	ui.Debug("Services to start: %v", serviceNames)

	// 5. Aggregate infrastructure requirements
	infraReqs := h.aggregateInfrastructure(services)
	ui.Debug("Infrastructure requirements: postgres=%v, mongodb=%v, redis=%v, sqs=%v",
		infraReqs.Postgres != nil, infraReqs.MongoDB != nil, infraReqs.Redis != nil, infraReqs.SQS != nil)

	// 5.5. Start tunnels FIRST if configured (so tunnel URLs are available for env)
	// Tunnels point to localhost ports that will be bound by infrastructure containers
	var tunnelContext map[string]ports.TunnelContext
	if infraReqs.Tunnel != nil && h.tunnelManager != nil {
		ui.Step("Starting tunnels...")
		var err error
		tunnelContext, err = h.startTunnels(ctx, infraReqs.Tunnel)
		if err != nil {
			return fmt.Errorf("failed to start tunnels: %w", err)
		}
		ui.Successf("Tunnels started")
	}

	// 6. Generate docker-compose (now with tunnel context for env resolution)
	ui.Step("Generating docker-compose configuration...")
	if err := h.generateCompose(services, infraReqs, tunnelContext); err != nil {
		return fmt.Errorf("failed to generate compose file: %w", err)
	}
	ui.Successf("Docker compose file generated")

	// 7. Start infrastructure containers first (postgres, redis, localstack, etc.)
	ui.Step("Starting infrastructure containers...")
	if err := h.orchestrator.StartInfrastructure(ctx); err != nil {
		return fmt.Errorf("failed to start infrastructure: %w", err)
	}
	ui.Successf("Infrastructure containers started")

	// 8. Provision infrastructure (create databases, SQS queues, etc.)
	// This runs AFTER containers are up
	if infraReqs.SQS != nil || infraReqs.SNS != nil || infraReqs.S3 != nil {
		ui.Step("Provisioning AWS resources in LocalStack...")
	}
	awsResources, err := h.provisionInfrastructure(ctx, infraReqs)
	if err != nil {
		return fmt.Errorf("failed to provision infrastructure: %w", err)
	}
	if infraReqs.SQS != nil || infraReqs.SNS != nil || infraReqs.S3 != nil {
		ui.Successf("AWS resources provisioned")
	}

	// Run post_infrastructure hooks (after DBs/queues ready)
	if err := h.runHooks(ctx, services, service.HookStagePostInfrastructure); err != nil {
		return err
	}

	// 8.5. Regenerate service compose files with real AWS resource URLs
	// Now that we have actual URLs/ARNs from LocalStack, update the compose files
	if awsResources != nil {
		ui.Step("Updating service configuration with actual AWS resource URLs...")
		if err := h.regenerateComposeWithAWSResources(services, infraReqs, tunnelContext, awsResources); err != nil {
			return fmt.Errorf("failed to update compose with AWS resources: %w", err)
		}
		ui.Successf("Service configuration updated")
	}

	// 9. Start services in parallel (no ordering enforced)
	if !cmd.InfraOnly {
		ui.Step("Starting application services...")
		if err := h.startServices(ctx, serviceNames); err != nil {
			return fmt.Errorf("failed to start services: %w", err)
		}
		ui.Successf("All services started successfully")

		// Run post_up hooks (after services healthy)
		if err := h.runHooks(ctx, services, service.HookStagePostUp); err != nil {
			return err
		}
	} else {
		ui.Infof("Infrastructure only mode - skipping service startup")
	}

	return nil
}

// loadServices loads only the explicitly requested services (no dependencies)
func (h *UpCommandHandler) loadServices(names []string) ([]*service.Service, error) {
	var services []*service.Service
	for _, name := range names {
		svc, err := h.serviceRepo.FindByName(service.ServiceName(name))
		if err != nil {
			return nil, fmt.Errorf("service %s: %w", name, err)
		}
		services = append(services, svc)
	}
	return services, nil
}

// loadServicesWithDependencies loads the requested services and all their transitive dependencies
// This handles circular dependencies by tracking visited services
func (h *UpCommandHandler) loadServicesWithDependencies(names []string) ([]*service.Service, error) {
	loaded := make(map[string]*service.Service)
	var loadOrder []string // Track order for consistent output

	var loadRecursive func(name string) error
	loadRecursive = func(name string) error {
		// Skip if already loaded (handles circular dependencies)
		if _, exists := loaded[name]; exists {
			return nil
		}

		svc, err := h.serviceRepo.FindByName(service.ServiceName(name))
		if err != nil {
			return fmt.Errorf("service %s: %w", name, err)
		}

		loaded[name] = svc
		loadOrder = append(loadOrder, name)

		// Recursively load dependencies
		for _, dep := range svc.Dependencies.Services {
			if err := loadRecursive(dep.String()); err != nil {
				return err
			}
		}

		return nil
	}

	// Load each requested service and its dependencies
	for _, name := range names {
		if err := loadRecursive(name); err != nil {
			return nil, err
		}
	}

	// Build result in load order
	services := make([]*service.Service, 0, len(loaded))
	for _, name := range loadOrder {
		services = append(services, loaded[name])
	}

	// Log if we loaded additional dependencies
	if len(services) > len(names) {
		var deps []string
		nameSet := make(map[string]bool)
		for _, n := range names {
			nameSet[n] = true
		}
		for _, svc := range services {
			if !nameSet[svc.Name] {
				deps = append(deps, svc.Name)
			}
		}
		ui.Infof("Also loading dependencies: %s", strings.Join(deps, ", "))
	}

	return services, nil
}

func (h *UpCommandHandler) aggregateInfrastructure(services []*service.Service) infrastructure.InfrastructureRequirements {
	var reqs []infrastructure.InfrastructureRequirements
	for _, svc := range services {
		reqs = append(reqs, svc.Dependencies.Infrastructure)
	}
	return infrastructure.Aggregate(reqs...)
}

func (h *UpCommandHandler) provisionInfrastructure(ctx context.Context, req infrastructure.InfrastructureRequirements) (*ports.ProvisionedAWSResources, error) {
	if req.Postgres != nil {
		if err := h.provisioner.ProvisionPostgres(ctx, req.Postgres); err != nil {
			return nil, fmt.Errorf("postgres: %w", err)
		}
	}
	if req.MongoDB != nil {
		if err := h.provisioner.ProvisionMongoDB(ctx, req.MongoDB); err != nil {
			return nil, fmt.Errorf("mongodb: %w", err)
		}
	}
	if req.Redis != nil {
		if err := h.provisioner.ProvisionRedis(ctx, req.Redis); err != nil {
			return nil, fmt.Errorf("redis: %w", err)
		}
	}
	if req.SQS != nil || req.SNS != nil || req.S3 != nil {
		awsResources, err := h.provisioner.ProvisionLocalStack(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("localstack: %w", err)
		}
		return awsResources, nil
	}
	return nil, nil
}

func (h *UpCommandHandler) generateCompose(services []*service.Service, req infrastructure.InfrastructureRequirements, tunnelCtx map[string]ports.TunnelContext) error {
	var fileSet *ports.ComposeFileSet
	var err error

	if tunnelCtx != nil && len(tunnelCtx) > 0 {
		fileSet, err = h.composeGenerator.GenerateWithTunnels(services, req, tunnelCtx)
	} else {
		fileSet, err = h.composeGenerator.Generate(services, req)
	}
	if err != nil {
		return err
	}

	// Update orchestrator with the generated compose files
	h.orchestrator.SetComposeFiles(fileSet.AllPaths())
	return nil
}

func (h *UpCommandHandler) startServices(ctx context.Context, order []service.ServiceName) error {
	return h.orchestrator.StartServices(ctx, order)
}

// regenerateComposeWithAWSResources regenerates service compose files with actual AWS resource URLs/ARNs
func (h *UpCommandHandler) regenerateComposeWithAWSResources(services []*service.Service, req infrastructure.InfrastructureRequirements, tunnelCtx map[string]ports.TunnelContext, awsResources *ports.ProvisionedAWSResources) error {
	fileSet, err := h.composeGenerator.GenerateWithAWSResources(services, req, tunnelCtx, awsResources)
	if err != nil {
		return err
	}

	// Update orchestrator with the regenerated compose files
	h.orchestrator.SetComposeFiles(fileSet.AllPaths())
	return nil
}

// startTunnels starts tunnels based on the tunnel requirement and returns tunnel context
func (h *UpCommandHandler) startTunnels(ctx context.Context, tunnelReq *infrastructure.TunnelRequirement) (map[string]ports.TunnelContext, error) {
	if tunnelReq == nil || len(tunnelReq.Targets) == 0 {
		return nil, nil
	}

	// Convert domain targets to config targets for the manager
	cfg := &config.TunnelConfig{
		Provider: tunnelReq.Provider,
		Targets:  make([]config.TunnelTarget, len(tunnelReq.Targets)),
	}
	for i, t := range tunnelReq.Targets {
		cfg.Targets[i] = config.TunnelTarget{
			Name: t.Name,
			Host: t.Host,
			Port: t.Port,
		}
	}

	// Resolve placeholders in targets (for now, just use the raw values)
	// TODO: In future, resolve ${localstack.host} etc.
	resolvedTargets := make([]ports.ResolvedTunnelTarget, len(tunnelReq.Targets))
	for i, t := range tunnelReq.Targets {
		resolvedTargets[i] = ports.ResolvedTunnelTarget{
			Name: t.Name,
			Host: t.Host,
			Port: t.Port,
		}
	}

	tunnels, err := h.tunnelManager.StartAll(ctx, cfg, resolvedTargets)
	if err != nil {
		return nil, err
	}

	// Build tunnel context for env resolution
	tunnelContext := make(map[string]ports.TunnelContext)
	for _, t := range tunnels {
		ui.Infof("  %s: %s -> %s", t.Name, t.PublicURL, t.LocalAddr)
		tunnelContext[t.Name] = ports.TunnelContext{
			Name:      t.Name,
			PublicURL: t.PublicURL,
		}
	}

	return tunnelContext, nil
}

// runHooks executes hooks for all services at the given stage
func (h *UpCommandHandler) runHooks(ctx context.Context, services []*service.Service, stage service.HookStage) error {
	for _, svc := range services {
		hooks := svc.Hooks.GetHooksForStage(stage)
		if len(hooks) == 0 {
			continue
		}

		ui.Step("Running %s hooks for %s...", stage, svc.Name)

		// Get service path for host execution
		servicePath, err := h.registryRepo.GetServicePath(service.ServiceName(svc.Name))
		if err != nil {
			return fmt.Errorf("failed to get service path: %w", err)
		}

		// Build execution context
		execCtx := ports.HookExecutionContext{
			ServiceName:   svc.Name,
			ServicePath:   servicePath,
			ContainerName: fmt.Sprintf("grund-%s-1", svc.Name),
			Environment:   svc.Environment.Variables,
		}

		if err := h.hookExecutor.ExecuteAll(ctx, hooks, execCtx); err != nil {
			return fmt.Errorf("%s hooks failed for %s: %w", stage, svc.Name, err)
		}
	}
	return nil
}
