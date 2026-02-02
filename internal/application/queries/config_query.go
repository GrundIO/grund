package queries

import (
	"github.com/vivekkundariya/grund/internal/application/ports"
	"github.com/vivekkundariya/grund/internal/domain/service"
)

// ConfigQuery represents a query for service configuration
type ConfigQuery struct {
	ServiceName string
}

// ConfigQueryHandler handles config queries
type ConfigQueryHandler struct {
	serviceRepo  ports.ServiceRepository
	registryRepo ports.ServiceRegistryRepository
	envResolver  ports.EnvironmentResolver
}

// NewConfigQueryHandler creates a new config query handler
func NewConfigQueryHandler(
	serviceRepo ports.ServiceRepository,
	registryRepo ports.ServiceRegistryRepository,
	envResolver ports.EnvironmentResolver,
) *ConfigQueryHandler {
	return &ConfigQueryHandler{
		serviceRepo:  serviceRepo,
		registryRepo: registryRepo,
		envResolver:  envResolver,
	}
}

// ResolvedConfig represents the resolved configuration
type ResolvedConfig struct {
	Service        *service.Service
	Environment    map[string]string
	Infrastructure []string
	Dependencies   []string
}

// Handle executes the config query
func (h *ConfigQueryHandler) Handle(query ConfigQuery) (*ResolvedConfig, error) {
	svc, err := h.serviceRepo.FindByName(service.ServiceName(query.ServiceName))
	if err != nil {
		return nil, err
	}

	// Build environment context based on service requirements
	envContext := h.buildEnvironmentContext(svc)

	// Resolve all environment variables (all values support ${placeholder} syntax)
	env, err := h.envResolver.Resolve(svc.Environment.Variables, envContext)
	if err != nil {
		return nil, err
	}

	// Collect infrastructure types
	var infraTypes []string
	if svc.RequiresInfrastructure("postgres") {
		infraTypes = append(infraTypes, "postgres")
	}
	if svc.RequiresInfrastructure("mongodb") {
		infraTypes = append(infraTypes, "mongodb")
	}
	if svc.RequiresInfrastructure("redis") {
		infraTypes = append(infraTypes, "redis")
	}
	if svc.RequiresInfrastructure("localstack") {
		infraTypes = append(infraTypes, "localstack")
	}

	// Collect service dependencies
	var deps []string
	for _, dep := range svc.Dependencies.Services {
		deps = append(deps, dep.String())
	}

	return &ResolvedConfig{
		Service:        svc,
		Environment:    env,
		Infrastructure: infraTypes,
		Dependencies:   deps,
	}, nil
}

// buildEnvironmentContext creates an environment context based on the service's requirements
func (h *ConfigQueryHandler) buildEnvironmentContext(svc *service.Service) ports.EnvironmentContext {
	ctx := ports.NewDefaultEnvironmentContext()

	// Set self context
	ctx.Self = ports.ServiceContext{
		Host:   svc.Name,
		Port:   svc.Port.Value(),
		Config: make(map[string]interface{}),
	}

	// Add postgres if required
	if svc.Dependencies.Infrastructure.Postgres != nil {
		ctx.Infrastructure["postgres"] = ports.InfrastructureContext{
			Host:     "postgres",
			Port:     5432,
			Database: svc.Dependencies.Infrastructure.Postgres.Database,
			Username: "postgres",
			Password: "postgres",
		}
		ctx.Self.Config["postgres.database"] = svc.Dependencies.Infrastructure.Postgres.Database
	}

	// Add mongodb if required
	if svc.Dependencies.Infrastructure.MongoDB != nil {
		ctx.Infrastructure["mongodb"] = ports.InfrastructureContext{
			Host:     "mongodb",
			Port:     27017,
			Database: svc.Dependencies.Infrastructure.MongoDB.Database,
		}
		ctx.Self.Config["mongodb.database"] = svc.Dependencies.Infrastructure.MongoDB.Database
	}

	// Add redis if required
	if svc.Dependencies.Infrastructure.Redis != nil {
		ctx.Infrastructure["redis"] = ports.InfrastructureContext{
			Host: "redis",
			Port: 6379,
		}
	}

	// Add SQS queues if required
	if svc.Dependencies.Infrastructure.SQS != nil {
		for _, queue := range svc.Dependencies.Infrastructure.SQS.Queues {
			ctx.SQS[queue.Name] = ports.QueueContext{
				Name: queue.Name,
				URL:  ctx.LocalStack.Endpoint + "/000000000000/" + queue.Name,
				ARN:  "arn:aws:sqs:" + ctx.LocalStack.Region + ":000000000000:" + queue.Name,
				DLQ:  ctx.LocalStack.Endpoint + "/000000000000/" + queue.Name + "-dlq",
			}
		}
	}

	// Add SNS topics if required
	if svc.Dependencies.Infrastructure.SNS != nil {
		for _, topic := range svc.Dependencies.Infrastructure.SNS.Topics {
			ctx.SNS[topic.Name] = ports.TopicContext{
				Name: topic.Name,
				ARN:  "arn:aws:sns:" + ctx.LocalStack.Region + ":000000000000:" + topic.Name,
			}
		}
	}

	// Add S3 buckets if required
	if svc.Dependencies.Infrastructure.S3 != nil {
		for _, bucket := range svc.Dependencies.Infrastructure.S3.Buckets {
			ctx.S3[bucket.Name] = ports.BucketContext{
				Name: bucket.Name,
				URL:  ctx.LocalStack.Endpoint + "/" + bucket.Name,
			}
		}
	}

	// Add service dependencies
	for _, dep := range svc.Dependencies.Services {
		// For preview purposes, use container name as host
		// In actual runtime, this would come from the resolved service
		depSvc, err := h.serviceRepo.FindByName(dep)
		if err == nil {
			ctx.Services[dep.String()] = ports.ServiceContext{
				Host:   dep.String(),
				Port:   depSvc.Port.Value(),
				Config: make(map[string]interface{}),
			}
		}
	}

	return ctx
}
