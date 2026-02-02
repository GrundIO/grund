package service

import (
	"fmt"
	"time"

	"github.com/Saturn-Fintech/grund/internal/domain/infrastructure"
)

// Service represents a service in the domain
type Service struct {
	Name         string
	Type         ServiceType
	Port         Port
	Build        *BuildConfig
	Run          *RunConfig
	Health       HealthConfig
	Dependencies ServiceDependencies
	Environment  Environment
}

// ServiceType represents the type of service
type ServiceType string

const (
	ServiceTypeGo     ServiceType = "go"
	ServiceTypePython ServiceType = "python"
	ServiceTypeNode   ServiceType = "node"
)

// Port represents a network port
type Port struct {
	value int
}

// NewPort creates a new Port value object
func NewPort(value int) (Port, error) {
	if value <= 0 || value > 65535 {
		return Port{}, fmt.Errorf("port must be between 1 and 65535, got %d", value)
	}
	return Port{value: value}, nil
}

// Value returns the port value
func (p Port) Value() int {
	return p.value
}

// BuildConfig represents build configuration
type BuildConfig struct {
	Dockerfile string
	Context    string
}

// RunConfig represents runtime configuration
type RunConfig struct {
	Command   string
	HotReload bool
}

// HealthConfig represents health check configuration
type HealthConfig struct {
	Endpoint string
	Interval time.Duration
	Timeout  time.Duration
	Retries  int
}

// ServiceDependencies represents what a service depends on
type ServiceDependencies struct {
	Services       []ServiceName
	Infrastructure infrastructure.InfrastructureRequirements
}

// ServiceName is a value object for service names
type ServiceName string

func (n ServiceName) String() string {
	return string(n)
}

// Environment represents environment variables
// All values support ${placeholder} syntax for dynamic resolution
type Environment struct {
	Variables map[string]string
	Secrets   map[string]SecretRequirement
}

// SecretRequirement defines a secret required by the service
type SecretRequirement struct {
	Description string
	Required    bool // already resolved from config (defaults applied)
}

// Validate validates the service configuration
func (s *Service) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("service name is required")
	}

	if s.Build == nil && s.Run == nil {
		return fmt.Errorf("either build or run configuration must be provided")
	}

	if s.Health.Endpoint == "" {
		return fmt.Errorf("health check endpoint is required")
	}

	return nil
}

// RequiresInfrastructure checks if service requires specific infrastructure
func (s *Service) RequiresInfrastructure(infraType string) bool {
	return s.Dependencies.Infrastructure.Has(infraType)
}
