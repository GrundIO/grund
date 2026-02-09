package ports

import (
	"context"

	"github.com/Saturn-Fintech/grund/internal/domain/infrastructure"
	"github.com/Saturn-Fintech/grund/internal/domain/service"
)

// ContainerOrchestrator defines the interface for container orchestration
// This abstracts away Docker Compose implementation details
type ContainerOrchestrator interface {
	StartInfrastructure(ctx context.Context) error
	StartServices(ctx context.Context, services []service.ServiceName) error
	StopServices(ctx context.Context) error
	RestartService(ctx context.Context, name service.ServiceName) error
	GetServiceStatus(ctx context.Context, name service.ServiceName) (ServiceStatus, error)
	GetAllServiceStatuses(ctx context.Context) ([]ServiceStatus, error)
	GetLogs(ctx context.Context, name service.ServiceName, follow bool, tail int) (LogStream, error)
	SetComposeFiles(files []string)
}

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Name     string
	Status   string
	Endpoint string
	Health   string
}

// LogStream represents a stream of logs
type LogStream interface {
	Read() ([]byte, error)
	Close() error
}

// InfrastructureProvisioner defines the interface for infrastructure provisioning
type InfrastructureProvisioner interface {
	ProvisionPostgres(ctx context.Context, config *infrastructure.PostgresConfig) error
	ProvisionMongoDB(ctx context.Context, config *infrastructure.MongoDBConfig) error
	ProvisionRedis(ctx context.Context, config *infrastructure.RedisConfig) error
	ProvisionLocalStack(ctx context.Context, req infrastructure.InfrastructureRequirements) (*ProvisionedAWSResources, error)
}

// ProvisionedAWSResources contains the actual resource details from LocalStack
type ProvisionedAWSResources struct {
	SQS map[string]ProvisionedQueue  // queue name -> details
	SNS map[string]ProvisionedTopic  // topic name -> details
	S3  map[string]ProvisionedBucket // bucket name -> details
}

// ProvisionedQueue contains actual SQS queue details from LocalStack
type ProvisionedQueue struct {
	Name   string
	URL    string
	ARN    string
	DLQURL string // Empty if no DLQ
	DLQARN string // Empty if no DLQ
}

// ProvisionedTopic contains actual SNS topic details from LocalStack
type ProvisionedTopic struct {
	Name string
	ARN  string
}

// ProvisionedBucket contains actual S3 bucket details from LocalStack
type ProvisionedBucket struct {
	Name string
	URL  string
}

// HealthChecker defines the interface for health checking
type HealthChecker interface {
	CheckHealth(ctx context.Context, endpoint string, timeout int) error
	WaitForHealthy(ctx context.Context, endpoint string, interval, timeout int, retries int) error
}
