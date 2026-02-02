package ports

import (
	"github.com/Saturn-Fintech/grund/internal/domain/infrastructure"
	"github.com/Saturn-Fintech/grund/internal/domain/service"
)

// ServiceRepository defines the interface for service persistence
// This follows the Repository pattern and Dependency Inversion Principle
type ServiceRepository interface {
	FindByName(name service.ServiceName) (*service.Service, error)
	FindAll() ([]*service.Service, error)
	Save(svc *service.Service) error
	// ExtractTunnelConfig extracts just the tunnel configuration without full service loading
	// This is used to start infrastructure tunnels before loading services
	ExtractTunnelConfig(name service.ServiceName) (*infrastructure.TunnelRequirement, error)
}

// ServiceRegistryRepository defines the interface for service registry
type ServiceRegistryRepository interface {
	GetServicePath(name service.ServiceName) (string, error)
	GetAllServices() (map[service.ServiceName]ServiceEntry, error)
}

// ServiceEntry represents a service entry in the registry
type ServiceEntry struct {
	Repo string
	Path string
}
