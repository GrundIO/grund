package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
	"github.com/Saturn-Fintech/grund/internal/domain/infrastructure"
	"github.com/Saturn-Fintech/grund/internal/domain/service"
	"github.com/Saturn-Fintech/grund/internal/ui"
	"gopkg.in/yaml.v3"
)

// ComposeGeneratorImpl implements ComposeGenerator
type ComposeGeneratorImpl struct {
	tmpDir        string // ~/.grund/tmp
	envResolver   ports.EnvironmentResolver
	secretsLoader *SecretsLoader
}

// NewComposeGenerator creates a new compose generator
// tmpDir should be ~/.grund/tmp
func NewComposeGenerator(tmpDir string) ports.ComposeGenerator {
	return &ComposeGeneratorImpl{
		tmpDir:        tmpDir,
		envResolver:   NewEnvironmentResolver(),
		secretsLoader: NewSecretsLoader(),
	}
}

// ComposeFile represents a docker-compose.yaml structure
type ComposeFile struct {
	Services map[string]ComposeService `yaml:"services"`
	Networks map[string]ComposeNetwork `yaml:"networks,omitempty"`
	Volumes  map[string]ComposeVolume  `yaml:"volumes,omitempty"`
}

// ComposeService represents a service in docker-compose
type ComposeService struct {
	Image         string            `yaml:"image,omitempty"`
	Build         *ComposeBuild     `yaml:"build,omitempty"`
	Ports         []string          `yaml:"ports,omitempty"`
	Environment   map[string]string `yaml:"environment,omitempty"`
	Volumes       []string          `yaml:"volumes,omitempty"`
	DependsOn     interface{}       `yaml:"depends_on,omitempty"`
	Networks      []string          `yaml:"networks,omitempty"`
	Healthcheck   *ComposeHealth    `yaml:"healthcheck,omitempty"`
	Command       []string          `yaml:"command,omitempty"`
	ContainerName string            `yaml:"container_name,omitempty"`
}

// ComposeBuild represents build configuration
type ComposeBuild struct {
	Context    string `yaml:"context"`
	Dockerfile string `yaml:"dockerfile"`
}

// ComposeHealth represents healthcheck configuration
type ComposeHealth struct {
	Test        []string `yaml:"test"`
	Interval    string   `yaml:"interval"`
	Timeout     string   `yaml:"timeout"`
	Retries     int      `yaml:"retries"`
	StartPeriod string   `yaml:"start_period,omitempty"`
}

// ComposeNetwork represents a network
type ComposeNetwork struct {
	Driver   string `yaml:"driver,omitempty"`
	External bool   `yaml:"external,omitempty"`
	Name     string `yaml:"name,omitempty"`
}

// ComposeVolume represents a volume
type ComposeVolume struct {
	Driver string `yaml:"driver,omitempty"`
}

// DependsOnCondition for service dependencies with conditions
type DependsOnCondition struct {
	Condition string `yaml:"condition"`
}

// portAllocator tracks used host ports and assigns available ones
type portAllocator struct {
	usedPorts map[int]string // port -> service name that uses it
}

// newPortAllocator creates a port allocator with infrastructure ports pre-reserved
func newPortAllocator() *portAllocator {
	pa := &portAllocator{
		usedPorts: make(map[int]string),
	}
	// Reserve standard infrastructure ports to prevent accidental conflicts
	reservedPorts := map[int]string{
		// Databases
		5432:  "postgres",
		3306:  "mysql",
		27017: "mongodb",
		6379:  "redis",
		9042:  "cassandra",
		7687:  "neo4j",
		8529:  "arangodb",
		// Message queues
		5672:  "rabbitmq",
		15672: "rabbitmq-management",
		9092:  "kafka",
		2181:  "zookeeper",
		4222:  "nats",
		// AWS LocalStack
		4566: "localstack",
		// Search
		9200: "elasticsearch",
		9300: "elasticsearch-transport",
		7700: "meilisearch",
		// Monitoring
		9090:  "prometheus",
		3000:  "grafana",
		9411:  "zipkin",
		16686: "jaeger",
		// Others
		8500: "consul",
		8200: "vault",
		2379: "etcd",
	}
	for port, name := range reservedPorts {
		pa.usedPorts[port] = name
	}
	return pa
}

// allocate returns an available host port for the given service and container port
// If the container port is available, it uses that. Otherwise, it finds the next available.
func (pa *portAllocator) allocate(serviceName string, containerPort int) (hostPort int, wasReassigned bool) {
	// If the desired port is available, use it
	if _, used := pa.usedPorts[containerPort]; !used {
		pa.usedPorts[containerPort] = serviceName
		return containerPort, false
	}

	// Find next available port starting from the container port
	candidate := containerPort + 1
	for {
		if _, used := pa.usedPorts[candidate]; !used {
			pa.usedPorts[candidate] = serviceName
			return candidate, true
		}
		candidate++
		// Safety limit to prevent infinite loop
		if candidate > 65535 {
			// Fall back to original port (will cause Docker error, but that's better than infinite loop)
			return containerPort, false
		}
	}
}

// Generate generates per-service docker-compose.yaml files
// Infrastructure goes in ~/.grund/tmp/infrastructure/docker-compose.yaml
// Each service goes in ~/.grund/tmp/<service>/docker-compose.yaml
// Returns ALL compose files (including existing ones from previous runs)
func (g *ComposeGeneratorImpl) Generate(services []*service.Service, infra infrastructure.InfrastructureRequirements) (*ports.ComposeFileSet, error) {
	fileSet := &ports.ComposeFileSet{
		ServicePaths: make(map[string]string),
	}

	// First, discover existing compose files from previous runs
	g.discoverExistingComposeFiles(fileSet)

	// Build environment context for variable resolution
	envContext := g.buildEnvironmentContext(services, infra)

	// Create port allocator to handle port conflicts
	portAlloc := newPortAllocator()

	// Generate infrastructure compose file if there are any infrastructure requirements
	// Note: generateInfrastructure merges with existing infrastructure
	if infra.Postgres != nil || infra.MongoDB != nil || infra.Redis != nil ||
		infra.SQS != nil || infra.SNS != nil || infra.S3 != nil {
		infraPath, err := g.generateInfrastructure(infra)
		if err != nil {
			return nil, fmt.Errorf("failed to generate infrastructure compose: %w", err)
		}
		fileSet.InfrastructurePath = infraPath
	} else if fileSet.InfrastructurePath != "" {
		// No new infrastructure requirements, but keep existing infrastructure
		ui.Debug("Keeping existing infrastructure compose file")
	}

	// Generate per-service compose files (overwrites if service already exists)
	for _, svc := range services {
		svcPath, err := g.generateService(svc, envContext, portAlloc)
		if err != nil {
			return nil, fmt.Errorf("failed to generate compose for %s: %w", svc.Name, err)
		}
		fileSet.ServicePaths[svc.Name] = svcPath
	}

	return fileSet, nil
}

// GenerateWithTunnels generates compose files with tunnel context for env resolution
func (g *ComposeGeneratorImpl) GenerateWithTunnels(services []*service.Service, infra infrastructure.InfrastructureRequirements, tunnelCtx map[string]ports.TunnelContext) (*ports.ComposeFileSet, error) {
	fileSet := &ports.ComposeFileSet{
		ServicePaths: make(map[string]string),
	}

	// First, discover existing compose files from previous runs
	g.discoverExistingComposeFiles(fileSet)

	// Build environment context for variable resolution
	envContext := g.buildEnvironmentContext(services, infra)

	// Inject tunnel context for ${tunnel.<name>.url} placeholders
	for name, tc := range tunnelCtx {
		envContext.Tunnel[name] = tc
	}

	// Create port allocator to handle port conflicts
	portAlloc := newPortAllocator()

	// Generate infrastructure compose file if there are any infrastructure requirements
	if infra.Postgres != nil || infra.MongoDB != nil || infra.Redis != nil ||
		infra.SQS != nil || infra.SNS != nil || infra.S3 != nil {
		infraPath, err := g.generateInfrastructure(infra)
		if err != nil {
			return nil, fmt.Errorf("failed to generate infrastructure compose: %w", err)
		}
		fileSet.InfrastructurePath = infraPath
	} else if fileSet.InfrastructurePath != "" {
		ui.Debug("Keeping existing infrastructure compose file")
	}

	// Generate per-service compose files
	for _, svc := range services {
		svcPath, err := g.generateService(svc, envContext, portAlloc)
		if err != nil {
			return nil, fmt.Errorf("failed to generate compose for %s: %w", svc.Name, err)
		}
		fileSet.ServicePaths[svc.Name] = svcPath
	}

	return fileSet, nil
}

// GenerateWithAWSResources generates compose files with actual AWS resource URLs/ARNs from LocalStack
// This is called after provisioning to update service configs with real URLs instead of hardcoded ones
func (g *ComposeGeneratorImpl) GenerateWithAWSResources(services []*service.Service, infra infrastructure.InfrastructureRequirements, tunnelCtx map[string]ports.TunnelContext, awsResources *ports.ProvisionedAWSResources) (*ports.ComposeFileSet, error) {
	fileSet := &ports.ComposeFileSet{
		ServicePaths: make(map[string]string),
	}

	// Discover existing compose files
	g.discoverExistingComposeFiles(fileSet)

	// Build environment context with actual AWS resource URLs
	envContext := g.buildEnvironmentContextWithAWSResources(services, infra, awsResources)

	// Inject tunnel context
	if tunnelCtx != nil {
		for name, tc := range tunnelCtx {
			envContext.Tunnel[name] = tc
		}
	}

	// Create port allocator
	portAlloc := newPortAllocator()

	// Don't regenerate infrastructure - only regenerate service compose files
	// Infrastructure compose file stays the same

	// Generate per-service compose files with real AWS URLs
	for _, svc := range services {
		svcPath, err := g.generateService(svc, envContext, portAlloc)
		if err != nil {
			return nil, fmt.Errorf("failed to generate compose for %s: %w", svc.Name, err)
		}
		fileSet.ServicePaths[svc.Name] = svcPath
	}

	return fileSet, nil
}

// discoverExistingComposeFiles scans tmpDir for existing compose files
func (g *ComposeGeneratorImpl) discoverExistingComposeFiles(fileSet *ports.ComposeFileSet) {
	// Check if tmp directory exists
	if _, err := os.Stat(g.tmpDir); os.IsNotExist(err) {
		return
	}

	// Scan all subdirectories for docker-compose.yaml
	entries, err := os.ReadDir(g.tmpDir)
	if err != nil {
		ui.Debug("Failed to read tmp directory: %v", err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		composePath := filepath.Join(g.tmpDir, entry.Name(), "docker-compose.yaml")
		if _, err := os.Stat(composePath); os.IsNotExist(err) {
			continue
		}

		if entry.Name() == "infrastructure" {
			fileSet.InfrastructurePath = composePath
		} else {
			fileSet.ServicePaths[entry.Name()] = composePath
		}
	}

	if len(fileSet.ServicePaths) > 0 || fileSet.InfrastructurePath != "" {
		svcNames := make([]string, 0, len(fileSet.ServicePaths))
		for name := range fileSet.ServicePaths {
			svcNames = append(svcNames, name)
		}
		ui.Debug("Discovered existing compose files: infra=%v, services=%v",
			fileSet.InfrastructurePath != "", svcNames)
	}
}

// generateInfrastructure generates the infrastructure docker-compose.yaml
// It merges with existing infrastructure to support incremental service startup
func (g *ComposeGeneratorImpl) generateInfrastructure(infra infrastructure.InfrastructureRequirements) (string, error) {
	infraDir := filepath.Join(g.tmpDir, "infrastructure")
	if err := os.MkdirAll(infraDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create infrastructure directory: %w", err)
	}

	outputPath := filepath.Join(infraDir, "docker-compose.yaml")

	// Try to load existing infrastructure compose file
	existingCompose := g.loadExistingInfrastructure(outputPath)

	compose := &ComposeFile{
		Services: make(map[string]ComposeService),
		Networks: map[string]ComposeNetwork{
			"grund-network": {Driver: "bridge", Name: "grund-network"},
		},
		Volumes: make(map[string]ComposeVolume),
	}

	// Merge existing services first (preserves infrastructure from previous runs)
	if existingCompose != nil {
		for name, svc := range existingCompose.Services {
			compose.Services[name] = svc
		}
		for name, vol := range existingCompose.Volumes {
			compose.Volumes[name] = vol
		}
		ui.Debug("Merged existing infrastructure: %v", getServiceNames(existingCompose.Services))
	}

	// Add/update infrastructure services from current requirements
	// This will overwrite existing services with same name (ensures config is up to date)
	g.addInfrastructureServices(compose, infra)

	if err := g.writeComposeFile(outputPath, compose); err != nil {
		return "", err
	}

	return outputPath, nil
}

// loadExistingInfrastructure reads and parses an existing infrastructure compose file
func (g *ComposeGeneratorImpl) loadExistingInfrastructure(path string) *ComposeFile {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil // File doesn't exist or can't be read
	}

	var compose ComposeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		ui.Debug("Failed to parse existing infrastructure compose: %v", err)
		return nil
	}

	return &compose
}

// getServiceNames returns the names of services in a map (for logging)
func getServiceNames(services map[string]ComposeService) []string {
	names := make([]string, 0, len(services))
	for name := range services {
		names = append(names, name)
	}
	return names
}

// generateService generates a single service's docker-compose.yaml
func (g *ComposeGeneratorImpl) generateService(svc *service.Service, envContext ports.EnvironmentContext, portAlloc *portAllocator) (string, error) {
	svcDir := filepath.Join(g.tmpDir, svc.Name)
	if err := os.MkdirAll(svcDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create service directory: %w", err)
	}

	compose := &ComposeFile{
		Services: make(map[string]ComposeService),
		Networks: map[string]ComposeNetwork{
			"grund-network": {External: true, Name: "grund-network"},
		},
	}

	// Build self context for this service
	selfContext := envContext
	selfContext.Self = ports.ServiceContext{
		Host: svc.Name,
		Port: svc.Port.Value(),
		Config: map[string]any{
			"postgres.database": getServicePostgresDB(svc),
			"mongodb.database":  getServiceMongoDB(svc),
		},
	}

	// Add this service to the compose file
	if err := g.addSingleService(compose, svc, selfContext, portAlloc); err != nil {
		return "", err
	}

	outputPath := filepath.Join(svcDir, "docker-compose.yaml")
	if err := g.writeComposeFile(outputPath, compose); err != nil {
		return "", err
	}

	return outputPath, nil
}

// writeComposeFile writes a compose file to disk
func (g *ComposeGeneratorImpl) writeComposeFile(outputPath string, compose *ComposeFile) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create compose file: %w", err)
	}
	defer file.Close()

	// Add header comment
	fmt.Fprintf(file, "# AUTO-GENERATED by grund - DO NOT EDIT\n")
	fmt.Fprintf(file, "# Regenerate with: grund up <services>\n\n")

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(compose); err != nil {
		return fmt.Errorf("failed to write compose file: %w", err)
	}

	return nil
}

// addSingleService adds a single service to a compose file
func (g *ComposeGeneratorImpl) addSingleService(compose *ComposeFile, svc *service.Service, selfContext ports.EnvironmentContext, portAlloc *portAllocator) error {
	// Resolve environment variables
	resolvedEnv := make(map[string]string)

	// Add AWS credentials FIRST if LocalStack is used (as defaults)
	// These can be overridden by user env values
	if svc.RequiresInfrastructure("localstack") {
		resolvedEnv["AWS_ENDPOINT"] = selfContext.LocalStack.Endpoint
		resolvedEnv["AWS_REGION"] = selfContext.LocalStack.Region
		resolvedEnv["AWS_ACCESS_KEY_ID"] = selfContext.LocalStack.AccessKeyID
		resolvedEnv["AWS_SECRET_ACCESS_KEY"] = selfContext.LocalStack.SecretAccessKey
		resolvedEnv["AWS_ACCOUNT_ID"] = selfContext.LocalStack.AccountID
	}

	// Resolve all environment variables (all values support ${placeholder} syntax)
	if len(svc.Environment.Variables) > 0 {
		resolved, err := g.envResolver.Resolve(svc.Environment.Variables, selfContext)
		if err != nil {
			return fmt.Errorf("failed to resolve env: %w", err)
		}
		for k, v := range resolved {
			resolvedEnv[k] = v
		}
	}

	// Add resolved secrets
	if len(svc.Environment.Secrets) > 0 {
		secrets, err := g.secretsLoader.ResolveSecrets(svc)
		if err != nil {
			return fmt.Errorf("failed to resolve secrets: %w", err)
		}
		for k, v := range secrets {
			resolvedEnv[k] = v
		}
	}

	// Build depends_on with conditions
	dependsOn := g.buildDependsOn(svc)

	// Create compose service
	composeService := ComposeService{
		ContainerName: fmt.Sprintf("grund-%s", svc.Name),
		Environment:   resolvedEnv,
		Networks:      []string{"grund-network"},
		DependsOn:     dependsOn,
	}

	// Set build or image
	if svc.Build != nil {
		composeService.Build = &ComposeBuild{
			Context:    svc.Build.Context,
			Dockerfile: svc.Build.Dockerfile,
		}
	}

	// Set ports with conflict detection
	containerPort := svc.Port.Value()
	hostPort, wasReassigned := portAlloc.allocate(svc.Name, containerPort)
	if wasReassigned {
		ui.Warnf("Port conflict: %s uses container port %d, assigned host port %d", svc.Name, containerPort, hostPort)
	}
	composeService.Ports = []string{fmt.Sprintf("%d:%d", hostPort, containerPort)}

	// Set healthcheck
	if svc.Health.Endpoint != "" {
		composeService.Healthcheck = &ComposeHealth{
			Test:     []string{"CMD-SHELL", fmt.Sprintf("curl -sf http://localhost:%d%s || exit 1", svc.Port.Value(), svc.Health.Endpoint)},
			Interval: svc.Health.Interval.String(),
			Timeout:  svc.Health.Timeout.String(),
			Retries:  svc.Health.Retries,
		}
	}

	compose.Services[svc.Name] = composeService
	return nil
}

func (g *ComposeGeneratorImpl) buildEnvironmentContext(services []*service.Service, infra infrastructure.InfrastructureRequirements) ports.EnvironmentContext {
	ctx := ports.NewDefaultEnvironmentContext()

	// Add infrastructure contexts
	if infra.Postgres != nil {
		ctx.Infrastructure["postgres"] = ports.InfrastructureContext{
			Host:     "postgres",
			Port:     5432,
			Database: infra.Postgres.Database,
			Username: "postgres",
			Password: "postgres",
		}
	}

	if infra.MongoDB != nil {
		ctx.Infrastructure["mongodb"] = ports.InfrastructureContext{
			Host:     "mongodb",
			Port:     27017,
			Database: infra.MongoDB.Database,
		}
	}

	if infra.Redis != nil {
		ctx.Infrastructure["redis"] = ports.InfrastructureContext{
			Host: "redis",
			Port: 6379,
		}
	}

	// Add SQS queue contexts (with placeholder URLs - will be updated after provisioning)
	if infra.SQS != nil {
		for _, queue := range infra.SQS.Queues {
			ctx.SQS[queue.Name] = ports.QueueContext{
				Name: queue.Name,
				URL:  fmt.Sprintf("%s/000000000000/%s", ctx.LocalStack.Endpoint, queue.Name),
				ARN:  fmt.Sprintf("arn:aws:sqs:%s:000000000000:%s", ctx.LocalStack.Region, queue.Name),
				DLQ:  fmt.Sprintf("%s/000000000000/%s-dlq", ctx.LocalStack.Endpoint, queue.Name),
			}
		}
	}

	// Add SNS topic contexts (with placeholder ARNs - will be updated after provisioning)
	if infra.SNS != nil {
		for _, topic := range infra.SNS.Topics {
			ctx.SNS[topic.Name] = ports.TopicContext{
				Name: topic.Name,
				ARN:  fmt.Sprintf("arn:aws:sns:%s:000000000000:%s", ctx.LocalStack.Region, topic.Name),
			}
		}
	}

	// Add S3 bucket contexts
	if infra.S3 != nil {
		for _, bucket := range infra.S3.Buckets {
			ctx.S3[bucket.Name] = ports.BucketContext{
				Name: bucket.Name,
				URL:  fmt.Sprintf("%s/%s", ctx.LocalStack.Endpoint, bucket.Name),
			}
		}
	}

	// Add service contexts
	for _, svc := range services {
		ctx.Services[svc.Name] = ports.ServiceContext{
			Host: svc.Name, // Container name in Docker network
			Port: svc.Port.Value(),
			Config: map[string]any{
				"postgres.database": getServicePostgresDB(svc),
				"mongodb.database":  getServiceMongoDB(svc),
			},
		}
	}

	return ctx
}

// buildEnvironmentContextWithAWSResources builds environment context using actual AWS resource URLs/ARNs
func (g *ComposeGeneratorImpl) buildEnvironmentContextWithAWSResources(services []*service.Service, infra infrastructure.InfrastructureRequirements, awsResources *ports.ProvisionedAWSResources) ports.EnvironmentContext {
	ctx := ports.NewDefaultEnvironmentContext()

	// Add infrastructure contexts
	if infra.Postgres != nil {
		ctx.Infrastructure["postgres"] = ports.InfrastructureContext{
			Host:     "postgres",
			Port:     5432,
			Database: infra.Postgres.Database,
			Username: "postgres",
			Password: "postgres",
		}
	}

	if infra.MongoDB != nil {
		ctx.Infrastructure["mongodb"] = ports.InfrastructureContext{
			Host:     "mongodb",
			Port:     27017,
			Database: infra.MongoDB.Database,
		}
	}

	if infra.Redis != nil {
		ctx.Infrastructure["redis"] = ports.InfrastructureContext{
			Host: "redis",
			Port: 6379,
		}
	}

	// Add SQS queue contexts with ACTUAL URLs from LocalStack
	if awsResources != nil && awsResources.SQS != nil {
		for name, queue := range awsResources.SQS {
			ctx.SQS[name] = ports.QueueContext{
				Name: queue.Name,
				URL:  queue.URL,
				ARN:  queue.ARN,
				DLQ:  queue.DLQURL,
			}
		}
	}

	// Add SNS topic contexts with ACTUAL ARNs from LocalStack
	if awsResources != nil && awsResources.SNS != nil {
		for name, topic := range awsResources.SNS {
			ctx.SNS[name] = ports.TopicContext{
				Name: topic.Name,
				ARN:  topic.ARN,
			}
		}
	}

	// Add S3 bucket contexts with ACTUAL URLs from LocalStack
	if awsResources != nil && awsResources.S3 != nil {
		for name, bucket := range awsResources.S3 {
			ctx.S3[name] = ports.BucketContext{
				Name: bucket.Name,
				URL:  bucket.URL,
			}
		}
	}

	// Add service contexts
	for _, svc := range services {
		ctx.Services[svc.Name] = ports.ServiceContext{
			Host: svc.Name,
			Port: svc.Port.Value(),
			Config: map[string]any{
				"postgres.database": getServicePostgresDB(svc),
				"mongodb.database":  getServiceMongoDB(svc),
			},
		}
	}

	return ctx
}

func (g *ComposeGeneratorImpl) addInfrastructureServices(compose *ComposeFile, infra infrastructure.InfrastructureRequirements) {
	// Add PostgreSQL
	if infra.Postgres != nil {
		compose.Services["postgres"] = ComposeService{
			Image:         "postgres:15-alpine",
			ContainerName: "grund-postgres",
			Ports:         []string{"5432:5432"},
			Environment: map[string]string{
				"POSTGRES_USER":     "postgres",
				"POSTGRES_PASSWORD": "postgres",
				"POSTGRES_DB":       infra.Postgres.Database,
			},
			Volumes:  []string{"postgres-data:/var/lib/postgresql/data"},
			Networks: []string{"grund-network"},
			Healthcheck: &ComposeHealth{
				Test:     []string{"CMD-SHELL", "pg_isready -U postgres"},
				Interval: "5s",
				Timeout:  "5s",
				Retries:  5,
			},
		}
		compose.Volumes["postgres-data"] = ComposeVolume{}
	}

	// Add MongoDB
	if infra.MongoDB != nil {
		compose.Services["mongodb"] = ComposeService{
			Image:         "mongo:6",
			ContainerName: "grund-mongodb",
			Ports:         []string{"27017:27017"},
			Environment: map[string]string{
				"MONGO_INITDB_DATABASE": infra.MongoDB.Database,
			},
			Volumes:  []string{"mongodb-data:/data/db"},
			Networks: []string{"grund-network"},
			Healthcheck: &ComposeHealth{
				Test:     []string{"CMD", "mongosh", "--eval", "db.adminCommand('ping')"},
				Interval: "5s",
				Timeout:  "5s",
				Retries:  5,
			},
		}
		compose.Volumes["mongodb-data"] = ComposeVolume{}
	}

	// Add Redis
	if infra.Redis != nil {
		compose.Services["redis"] = ComposeService{
			Image:         "redis:7-alpine",
			ContainerName: "grund-redis",
			Ports:         []string{"6379:6379"},
			Networks:      []string{"grund-network"},
			Healthcheck: &ComposeHealth{
				Test:     []string{"CMD", "redis-cli", "ping"},
				Interval: "5s",
				Timeout:  "5s",
				Retries:  5,
			},
		}
	}

	// Add LocalStack if any AWS services are needed
	if infra.SQS != nil || infra.SNS != nil || infra.S3 != nil {
		services := []string{}
		if infra.SQS != nil {
			services = append(services, "sqs")
		}
		if infra.SNS != nil {
			services = append(services, "sns")
		}
		if infra.S3 != nil {
			services = append(services, "s3")
		}

		compose.Services["localstack"] = ComposeService{
			Image:         "localstack/localstack:latest",
			ContainerName: "grund-localstack",
			Ports:         []string{"4566:4566"},
			Environment: map[string]string{
				"SERVICES":           strings.Join(services, ","),
				"DEBUG":              "0",
				"AWS_DEFAULT_REGION": "us-east-1",
				"AWS_ACCOUNT_ID":     "000000000000",
				"DOCKER_HOST":        "unix:///var/run/docker.sock",
			},
			Volumes: []string{
				"/var/run/docker.sock:/var/run/docker.sock",
				"localstack-data:/var/lib/localstack",
			},
			Networks: []string{"grund-network"},
			Healthcheck: &ComposeHealth{
				Test:        []string{"CMD-SHELL", "curl -sf http://localhost:4566/_localstack/health || exit 1"},
				Interval:    "10s",
				Timeout:     "5s",
				Retries:     10,
				StartPeriod: "20s",
			},
		}
		compose.Volumes["localstack-data"] = ComposeVolume{}
	}
}

func (g *ComposeGeneratorImpl) buildDependsOn(svc *service.Service) map[string]DependsOnCondition {
	dependsOn := make(map[string]DependsOnCondition)

	// Add infrastructure dependencies only
	// Service-to-service dependencies are NOT added here - services should handle
	// reconnection logic themselves. This allows circular dependencies and parallel startup.
	if svc.RequiresInfrastructure("postgres") {
		dependsOn["postgres"] = DependsOnCondition{Condition: "service_healthy"}
	}
	if svc.RequiresInfrastructure("mongodb") {
		dependsOn["mongodb"] = DependsOnCondition{Condition: "service_healthy"}
	}
	if svc.RequiresInfrastructure("redis") {
		dependsOn["redis"] = DependsOnCondition{Condition: "service_healthy"}
	}
	if svc.RequiresInfrastructure("localstack") {
		dependsOn["localstack"] = DependsOnCondition{Condition: "service_healthy"}
	}

	if len(dependsOn) == 0 {
		return nil
	}

	return dependsOn
}

// Helper functions
func getServicePostgresDB(svc *service.Service) string {
	if svc.Dependencies.Infrastructure.Postgres != nil {
		return svc.Dependencies.Infrastructure.Postgres.Database
	}
	return ""
}

func getServiceMongoDB(svc *service.Service) string {
	if svc.Dependencies.Infrastructure.MongoDB != nil {
		return svc.Dependencies.Infrastructure.MongoDB.Database
	}
	return ""
}
