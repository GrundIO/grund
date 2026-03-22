package wiring

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GrundIO/grund/internal/application/commands"
	"github.com/GrundIO/grund/internal/application/queries"
	appconfig "github.com/GrundIO/grund/internal/config"
	"github.com/GrundIO/grund/internal/infrastructure/aws"
	"github.com/GrundIO/grund/internal/infrastructure/config"
	"github.com/GrundIO/grund/internal/infrastructure/docker"
	"github.com/GrundIO/grund/internal/infrastructure/generator"
	"github.com/GrundIO/grund/internal/infrastructure/git"
	"github.com/GrundIO/grund/internal/infrastructure/hooks"
	"github.com/GrundIO/grund/internal/infrastructure/tunnel"
)

// Container holds all dependencies (Dependency Injection Container)
// This follows the Dependency Inversion Principle
type Container struct {
	// Configuration
	ConfigResolver    *appconfig.ConfigResolver
	OrchestrationRoot string
	ServicesPath      string

	// Repositories
	ServiceRepo  interface{} // ports.ServiceRepository
	RegistryRepo interface{} // ports.ServiceRegistryRepository

	// Infrastructure
	Orchestrator     interface{} // ports.ContainerOrchestrator
	Provisioner      interface{} // ports.InfrastructureProvisioner
	ComposeGenerator interface{} // ports.ComposeGenerator
	EnvResolver      interface{} // ports.EnvironmentResolver
	HealthChecker    interface{} // ports.HealthChecker
	HookExecutor     interface{} // ports.HookExecutor

	// Command Handlers
	UpCommandHandler      *commands.UpCommandHandler
	DownCommandHandler    *commands.DownCommandHandler
	RestartCommandHandler *commands.RestartCommandHandler
	SyncCommandHandler    *commands.SyncCommandHandler

	// Query Handlers
	StatusQueryHandler *queries.StatusQueryHandler
	ConfigQueryHandler *queries.ConfigQueryHandler
	GraphQueryHandler  *queries.GraphQueryHandler
}

// NewContainer creates a new dependency injection container
// This is the legacy function for backward compatibility
func NewContainer(orchestrationRoot string) (*Container, error) {
	servicesPath := filepath.Join(orchestrationRoot, "services.yaml")
	if _, err := os.Stat(servicesPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("services.yaml not found at %s", servicesPath)
	}

	return NewContainerWithConfig(orchestrationRoot, servicesPath, nil)
}

// NewContainerWithConfig creates a new dependency injection container with config resolver
func NewContainerWithConfig(orchestrationRoot, servicesPath string, configResolver *appconfig.ConfigResolver) (*Container, error) {
	// Validate services file exists
	if _, err := os.Stat(servicesPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("services file not found at %s", servicesPath)
	}

	// Initialize repositories
	registryRepo, err := config.NewServiceRegistryRepository(servicesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry repository: %w", err)
	}

	serviceRepo := config.NewServiceRepository(registryRepo)

	// Get LocalStack endpoint from config or use default
	localstackEndpoint := "http://localhost:4566"
	if configResolver != nil {
		localstackEndpoint = configResolver.GetLocalStackEndpoint()
	}

	// Get grund tmp directory for compose file generation
	grundTmpDir, err := docker.GetGrundTmpDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get grund tmp directory: %w", err)
	}

	// Initialize infrastructure adapters
	orchestrator := docker.NewDockerOrchestrator(orchestrationRoot)
	healthChecker := docker.NewHTTPHealthChecker()

	// Discover existing compose files and set them on the orchestrator
	// This allows status, logs, etc. to work without running 'up' first
	if existingFiles, err := docker.DiscoverComposeFiles(); err == nil {
		orchestrator.SetComposeFiles(existingFiles.AllPaths())
	}

	// Initialize provisioners
	postgresProvisioner := docker.NewPostgresProvisioner()
	mongodbProvisioner := docker.NewMongoDBProvisioner()
	redisProvisioner := docker.NewRedisProvisioner()
	localstackProvisioner := aws.NewLocalStackProvisioner(localstackEndpoint)
	provisioner := docker.NewCompositeInfrastructureProvisioner(
		postgresProvisioner,
		mongodbProvisioner,
		redisProvisioner,
		localstackProvisioner,
	)

	// Initialize generators
	composeGenerator := generator.NewComposeGenerator(grundTmpDir)
	envResolver := generator.NewEnvironmentResolver()

	// Initialize tunnel manager
	tunnelManager := tunnel.NewManager()

	// Initialize hook executor
	hookExecutor := hooks.NewHookExecutor()

	// Initialize command handlers
	upHandler := commands.NewUpCommandHandler(
		serviceRepo,
		registryRepo,
		orchestrator,
		provisioner,
		composeGenerator,
		healthChecker,
		tunnelManager,
		hookExecutor,
	)

	downHandler := commands.NewDownCommandHandler(
		serviceRepo,
		registryRepo,
		orchestrator,
		hookExecutor,
		tunnelManager,
	)
	restartHandler := commands.NewRestartCommandHandler(orchestrator)

	// Initialize git client and sync handler
	gitClient := git.NewClient()
	syncHandler := commands.NewSyncCommandHandler(registryRepo, gitClient)

	// Initialize query handlers
	statusHandler := queries.NewStatusQueryHandler(orchestrator)
	configHandler := queries.NewConfigQueryHandler(
		serviceRepo,
		registryRepo,
		envResolver,
	)
	graphHandler := queries.NewGraphQueryHandler(serviceRepo, registryRepo)

	return &Container{
		ConfigResolver:        configResolver,
		OrchestrationRoot:     orchestrationRoot,
		ServicesPath:          servicesPath,
		ServiceRepo:           serviceRepo,
		RegistryRepo:          registryRepo,
		Orchestrator:          orchestrator,
		Provisioner:           provisioner,
		ComposeGenerator:      composeGenerator,
		EnvResolver:           envResolver,
		HealthChecker:         healthChecker,
		HookExecutor:          hookExecutor,
		UpCommandHandler:      upHandler,
		DownCommandHandler:    downHandler,
		RestartCommandHandler: restartHandler,
		SyncCommandHandler:    syncHandler,
		StatusQueryHandler:    statusHandler,
		ConfigQueryHandler:    configHandler,
		GraphQueryHandler:     graphHandler,
	}, nil
}
