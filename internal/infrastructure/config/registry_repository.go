package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
	"github.com/Saturn-Fintech/grund/internal/domain/service"
	"gopkg.in/yaml.v3"
)

// ServiceRegistryRepositoryImpl implements ServiceRegistryRepository
type ServiceRegistryRepositoryImpl struct {
	servicesPath string
	registry     *ServiceRegistryDTO
}

// NewServiceRegistryRepository creates a new registry repository
func NewServiceRegistryRepository(servicesPath string) (ports.ServiceRegistryRepository, error) {
	data, err := os.ReadFile(servicesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read services.yaml: %w", err)
	}

	var registry ServiceRegistryDTO
	if err := yaml.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("failed to parse services.yaml: %w", err)
	}

	return &ServiceRegistryRepositoryImpl{
		servicesPath: servicesPath,
		registry:     &registry,
	}, nil
}

// GetServicePath returns the local path for a service
func (r *ServiceRegistryRepositoryImpl) GetServicePath(name service.ServiceName) (string, error) {
	entry, ok := r.registry.Services[string(name)]
	if !ok {
		return "", fmt.Errorf("service %s not found in registry", name)
	}

	// Expand ~ to home directory
	path := entry.Path
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}

	return path, nil
}

// GetAllServices returns all service entries
func (r *ServiceRegistryRepositoryImpl) GetAllServices() (map[service.ServiceName]ports.ServiceEntry, error) {
	result := make(map[service.ServiceName]ports.ServiceEntry)
	for name, entry := range r.registry.Services {
		result[service.ServiceName(name)] = ports.ServiceEntry{
			Repo: entry.Repo,
			Path: entry.Path,
		}
	}
	return result, nil
}

// ServiceRegistryDTO is the DTO for services.yaml
type ServiceRegistryDTO struct {
	Version      string                     `yaml:"version"`
	Services     map[string]ServiceEntryDTO `yaml:"services"`
	PathDefaults PathDefaultsDTO            `yaml:"path_defaults"`
}

type ServiceEntryDTO struct {
	Repo string `yaml:"repo"`
	Path string `yaml:"path"`
}

type PathDefaultsDTO struct {
	Base string `yaml:"base"`
}
