package docker

import (
	"context"
	"fmt"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
	"github.com/Saturn-Fintech/grund/internal/domain/infrastructure"
)

// RedisProvisioner implements infrastructure provisioning for Redis
type RedisProvisioner struct{}

// NewRedisProvisioner creates a new Redis provisioner
func NewRedisProvisioner() ports.InfrastructureProvisioner {
	return &RedisProvisioner{}
}

// ProvisionRedis provisions Redis
func (r *RedisProvisioner) ProvisionRedis(ctx context.Context, config *infrastructure.RedisConfig) error {
	// Redis is started via docker-compose, no additional provisioning needed
	// Redis is ready to use once the container is healthy
	return nil
}

// ProvisionPostgres not applicable
func (r *RedisProvisioner) ProvisionPostgres(ctx context.Context, config *infrastructure.PostgresConfig) error {
	return fmt.Errorf("postgres provisioning not supported by redis provisioner")
}

// ProvisionMongoDB not applicable
func (r *RedisProvisioner) ProvisionMongoDB(ctx context.Context, config *infrastructure.MongoDBConfig) error {
	return fmt.Errorf("mongodb provisioning not supported by redis provisioner")
}

// ProvisionLocalStack not applicable
func (r *RedisProvisioner) ProvisionLocalStack(ctx context.Context, req infrastructure.InfrastructureRequirements) (*ports.ProvisionedAWSResources, error) {
	return nil, fmt.Errorf("localstack provisioning not supported by redis provisioner")
}
