package docker

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
	"github.com/Saturn-Fintech/grund/internal/domain/infrastructure"
)

// ErrNotSupported indicates a provisioner doesn't support a specific infrastructure type
var ErrNotSupported = errors.New("not supported")

// CompositeInfrastructureProvisioner combines multiple provisioners
// This follows the Composite pattern
type CompositeInfrastructureProvisioner struct {
	provisioners []ports.InfrastructureProvisioner
}

// NewCompositeInfrastructureProvisioner creates a composite provisioner
func NewCompositeInfrastructureProvisioner(provisioners ...ports.InfrastructureProvisioner) ports.InfrastructureProvisioner {
	return &CompositeInfrastructureProvisioner{
		provisioners: provisioners,
	}
}

// ProvisionPostgres delegates to the first provisioner that supports it
func (c *CompositeInfrastructureProvisioner) ProvisionPostgres(ctx context.Context, config *infrastructure.PostgresConfig) error {
	var lastErr error
	for _, p := range c.provisioners {
		err := p.ProvisionPostgres(ctx, config)
		if err == nil {
			return nil
		}
		// If it's a "not supported" error, try next provisioner
		if isNotSupportedError(err) {
			continue
		}
		// Real error - save it
		lastErr = err
	}
	if lastErr != nil {
		return fmt.Errorf("postgres provisioning failed: %w", lastErr)
	}
	return fmt.Errorf("no provisioner supports postgres")
}

// ProvisionMongoDB delegates to the first provisioner that supports it
func (c *CompositeInfrastructureProvisioner) ProvisionMongoDB(ctx context.Context, config *infrastructure.MongoDBConfig) error {
	var lastErr error
	for _, p := range c.provisioners {
		err := p.ProvisionMongoDB(ctx, config)
		if err == nil {
			return nil
		}
		if isNotSupportedError(err) {
			continue
		}
		lastErr = err
	}
	if lastErr != nil {
		return fmt.Errorf("mongodb provisioning failed: %w", lastErr)
	}
	return fmt.Errorf("no provisioner supports mongodb")
}

// ProvisionRedis delegates to the first provisioner that supports it
func (c *CompositeInfrastructureProvisioner) ProvisionRedis(ctx context.Context, config *infrastructure.RedisConfig) error {
	var lastErr error
	for _, p := range c.provisioners {
		err := p.ProvisionRedis(ctx, config)
		if err == nil {
			return nil
		}
		if isNotSupportedError(err) {
			continue
		}
		lastErr = err
	}
	if lastErr != nil {
		return fmt.Errorf("redis provisioning failed: %w", lastErr)
	}
	return fmt.Errorf("no provisioner supports redis")
}

// ProvisionLocalStack delegates to LocalStack provisioner
func (c *CompositeInfrastructureProvisioner) ProvisionLocalStack(ctx context.Context, req infrastructure.InfrastructureRequirements) (*ports.ProvisionedAWSResources, error) {
	var lastErr error
	for _, p := range c.provisioners {
		result, err := p.ProvisionLocalStack(ctx, req)
		if err == nil {
			return result, nil
		}
		if isNotSupportedError(err) {
			continue
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, fmt.Errorf("localstack provisioning failed: %w", lastErr)
	}
	return nil, fmt.Errorf("no provisioner supports localstack")
}

// isNotSupportedError checks if an error indicates the provisioner doesn't support this type
func isNotSupportedError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "not supported by") ||
		errors.Is(err, ErrNotSupported)
}
