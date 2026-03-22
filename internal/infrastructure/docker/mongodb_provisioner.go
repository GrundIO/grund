package docker

import (
	"context"
	"fmt"

	"github.com/GrundIO/grund/internal/application/ports"
	"github.com/GrundIO/grund/internal/domain/infrastructure"
)

// MongoDBProvisioner implements infrastructure provisioning for MongoDB
type MongoDBProvisioner struct{}

// NewMongoDBProvisioner creates a new MongoDB provisioner
func NewMongoDBProvisioner() ports.InfrastructureProvisioner {
	return &MongoDBProvisioner{}
}

// ProvisionMongoDB provisions MongoDB
func (m *MongoDBProvisioner) ProvisionMongoDB(ctx context.Context, config *infrastructure.MongoDBConfig) error {
	// MongoDB is started via docker-compose, provisioning here
	// mainly involves seeding if needed
	// TODO: Implement seeding logic
	return nil
}

// ProvisionPostgres not applicable
func (m *MongoDBProvisioner) ProvisionPostgres(ctx context.Context, config *infrastructure.PostgresConfig) error {
	return fmt.Errorf("postgres provisioning not supported by mongodb provisioner")
}

// ProvisionRedis not applicable
func (m *MongoDBProvisioner) ProvisionRedis(ctx context.Context, config *infrastructure.RedisConfig) error {
	return fmt.Errorf("redis provisioning not supported by mongodb provisioner")
}

// ProvisionLocalStack not applicable
func (m *MongoDBProvisioner) ProvisionLocalStack(ctx context.Context, req infrastructure.InfrastructureRequirements) (*ports.ProvisionedAWSResources, error) {
	return nil, fmt.Errorf("localstack provisioning not supported by mongodb provisioner")
}
