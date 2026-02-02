package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
	"github.com/Saturn-Fintech/grund/internal/domain/infrastructure"
	"github.com/Saturn-Fintech/grund/internal/domain/service"
	"gopkg.in/yaml.v3"
)

// ServiceRepositoryImpl implements ServiceRepository using file system
// This is an infrastructure adapter following Dependency Inversion Principle
type ServiceRepositoryImpl struct {
	registryRepo ports.ServiceRegistryRepository
}

// NewServiceRepository creates a new service repository
func NewServiceRepository(registryRepo ports.ServiceRegistryRepository) ports.ServiceRepository {
	return &ServiceRepositoryImpl{
		registryRepo: registryRepo,
	}
}

// FindByName finds a service by name
func (r *ServiceRepositoryImpl) FindByName(name service.ServiceName) (*service.Service, error) {
	servicePath, err := r.registryRepo.GetServicePath(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get service path: %w", err)
	}

	configPath := filepath.Join(servicePath, "grund.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var configDTO ServiceConfigDTO
	if err := yaml.Unmarshal(data, &configDTO); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return r.toDomainService(configDTO, name, servicePath)
}

// FindAll finds all services
func (r *ServiceRepositoryImpl) FindAll() ([]*service.Service, error) {
	entries, err := r.registryRepo.GetAllServices()
	if err != nil {
		return nil, err
	}

	var services []*service.Service
	for name := range entries {
		svc, err := r.FindByName(name)
		if err != nil {
			return nil, err
		}
		services = append(services, svc)
	}

	return services, nil
}

// Save saves a service configuration
func (r *ServiceRepositoryImpl) Save(svc *service.Service) error {
	path, err := r.registryRepo.GetServicePath(service.ServiceName(svc.Name))
	if err != nil {
		return err
	}

	configPath := filepath.Join(path, "grund.yaml")
	configDTO := r.toConfigDTO(svc)

	data, err := yaml.Marshal(configDTO)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	return os.WriteFile(configPath, data, 0644)
}

// ServiceConfigDTO is the DTO for YAML serialization
// This separates infrastructure concerns from domain
type ServiceConfigDTO struct {
	Version  string                     `yaml:"version"`
	Service  ServiceInfoDTO             `yaml:"service"`
	Requires RequirementsDTO            `yaml:"requires"`
	Env      map[string]string          `yaml:"env"` // All values support ${placeholder} syntax
	Secrets  map[string]SecretConfigDTO `yaml:"secrets,omitempty"`
}

// SecretConfigDTO is the DTO for secret configuration
type SecretConfigDTO struct {
	Description string `yaml:"description"`
	Required    *bool  `yaml:"required,omitempty"` // nil defaults to true
}

// IsRequired returns true if the secret is required (default: true)
func (s SecretConfigDTO) IsRequired() bool {
	if s.Required == nil {
		return true
	}
	return *s.Required
}

type ServiceInfoDTO struct {
	Name   string          `yaml:"name"`
	Type   string          `yaml:"type"`
	Port   int             `yaml:"port"`
	Build  *BuildConfigDTO `yaml:"build,omitempty"`
	Run    *RunConfigDTO   `yaml:"run,omitempty"`
	Health HealthConfigDTO `yaml:"health"`
}

type BuildConfigDTO struct {
	Dockerfile string `yaml:"dockerfile"`
	Context    string `yaml:"context"`
}

type RunConfigDTO struct {
	Command   string `yaml:"command"`
	HotReload bool   `yaml:"hot_reload"`
}

type HealthConfigDTO struct {
	Endpoint string `yaml:"endpoint"`
	Interval string `yaml:"interval"`
	Timeout  string `yaml:"timeout"`
	Retries  int    `yaml:"retries"`
}

type RequirementsDTO struct {
	Services       []string                `yaml:"services"`
	Infrastructure InfrastructureConfigDTO `yaml:"infrastructure"`
}

type InfrastructureConfigDTO struct {
	Postgres *PostgresConfigDTO `yaml:"postgres,omitempty"`
	MongoDB  *MongoDBConfigDTO  `yaml:"mongodb,omitempty"`
	Redis    interface{}        `yaml:"redis,omitempty"`
	SQS      *SQSConfigDTO      `yaml:"sqs,omitempty"`
	SNS      *SNSConfigDTO      `yaml:"sns,omitempty"`
	S3       *S3ConfigDTO       `yaml:"s3,omitempty"`
	Tunnel   *TunnelConfigDTO   `yaml:"tunnel,omitempty"`
}

type TunnelConfigDTO struct {
	Provider string            `yaml:"provider"` // "cloudflared" or "ngrok"
	Targets  []TunnelTargetDTO `yaml:"targets"`
}

type TunnelTargetDTO struct {
	Name string `yaml:"name"`
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

type PostgresConfigDTO struct {
	Database   string `yaml:"database"`
	Migrations string `yaml:"migrations,omitempty"`
	Seed       string `yaml:"seed,omitempty"`
}

type MongoDBConfigDTO struct {
	Database string `yaml:"database"`
	Seed     string `yaml:"seed,omitempty"`
}

type RedisConfigDTO struct{}

type SQSConfigDTO struct {
	Queues []QueueConfigDTO `yaml:"queues"`
}

type QueueConfigDTO struct {
	Name string `yaml:"name"`
	DLQ  bool   `yaml:"dlq,omitempty"`
}

type SNSConfigDTO struct {
	Topics []TopicConfigDTO `yaml:"topics"`
}

type TopicConfigDTO struct {
	Name          string                  `yaml:"name"`
	Subscriptions []SubscriptionConfigDTO `yaml:"subscriptions,omitempty"`
}

type SubscriptionConfigDTO struct {
	Protocol   string            `yaml:"protocol"`
	Endpoint   string            `yaml:"endpoint"`
	Attributes map[string]string `yaml:"attributes,omitempty"`
}

type S3ConfigDTO struct {
	Buckets []BucketConfigDTO `yaml:"buckets"`
}

type BucketConfigDTO struct {
	Name string `yaml:"name"`
	Seed string `yaml:"seed,omitempty"`
}

// toDomainService converts DTO to domain model
func (r *ServiceRepositoryImpl) toDomainService(dto ServiceConfigDTO, name service.ServiceName, servicePath string) (*service.Service, error) {
	port, err := service.NewPort(dto.Service.Port)
	if err != nil {
		return nil, err
	}

	var build *service.BuildConfig
	if dto.Service.Build != nil {
		// Resolve build context: if it's "." or relative, use the service path
		buildContext := servicePath
		if dto.Service.Build.Context != "" && dto.Service.Build.Context != "." {
			// If context is a subdirectory, join it with service path
			buildContext = filepath.Join(servicePath, dto.Service.Build.Context)
		}
		build = &service.BuildConfig{
			Dockerfile: dto.Service.Build.Dockerfile,
			Context:    buildContext,
		}
	}

	var run *service.RunConfig
	if dto.Service.Run != nil {
		run = &service.RunConfig{
			Command:   dto.Service.Run.Command,
			HotReload: dto.Service.Run.HotReload,
		}
	}

	// Parse health config interval and timeout
	interval, _ := parseDuration(dto.Service.Health.Interval)
	timeout, _ := parseDuration(dto.Service.Health.Timeout)

	health := service.HealthConfig{
		Endpoint: dto.Service.Health.Endpoint,
		Interval: interval,
		Timeout:  timeout,
		Retries:  dto.Service.Health.Retries,
	}

	// Convert service dependencies
	var serviceDeps []service.ServiceName
	for _, dep := range dto.Requires.Services {
		serviceDeps = append(serviceDeps, service.ServiceName(dep))
	}

	// Convert infrastructure requirements
	infraReqs := r.toInfrastructureRequirements(dto.Requires.Infrastructure)

	deps := service.ServiceDependencies{
		Services:       serviceDeps,
		Infrastructure: infraReqs,
	}

	// Convert secrets
	secrets := make(map[string]service.SecretRequirement)
	for name, secretDTO := range dto.Secrets {
		secrets[name] = service.SecretRequirement{
			Description: secretDTO.Description,
			Required:    secretDTO.IsRequired(),
		}
	}

	env := service.Environment{
		Variables: dto.Env,
		Secrets:   secrets,
	}

	svc := &service.Service{
		Name:         dto.Service.Name,
		Type:         service.ServiceType(dto.Service.Type),
		Port:         port,
		Build:        build,
		Run:          run,
		Health:       health,
		Dependencies: deps,
		Environment:  env,
	}

	return svc, svc.Validate()
}

func (r *ServiceRepositoryImpl) toInfrastructureRequirements(dto InfrastructureConfigDTO) infrastructure.InfrastructureRequirements {
	var req infrastructure.InfrastructureRequirements

	if dto.Postgres != nil {
		req.Postgres = &infrastructure.PostgresConfig{
			Database:   dto.Postgres.Database,
			Migrations: dto.Postgres.Migrations,
			Seed:       dto.Postgres.Seed,
		}
	}

	if dto.MongoDB != nil {
		req.MongoDB = &infrastructure.MongoDBConfig{
			Database: dto.MongoDB.Database,
			Seed:     dto.MongoDB.Seed,
		}
	}

	if dto.Redis != nil {
		req.Redis = &infrastructure.RedisConfig{}
	}

	if dto.SQS != nil {
		var queues []infrastructure.QueueConfig
		for _, q := range dto.SQS.Queues {
			queues = append(queues, infrastructure.QueueConfig{
				Name: q.Name,
				DLQ:  q.DLQ,
			})
		}
		req.SQS = &infrastructure.SQSConfig{Queues: queues}
	}

	if dto.SNS != nil {
		var topics []infrastructure.TopicConfig
		for _, t := range dto.SNS.Topics {
			var subs []infrastructure.SubscriptionConfig
			for _, s := range t.Subscriptions {
				subs = append(subs, infrastructure.SubscriptionConfig{
					Protocol:   s.Protocol,
					Endpoint:   s.Endpoint,
					Attributes: s.Attributes,
				})
			}
			topics = append(topics, infrastructure.TopicConfig{
				Name:          t.Name,
				Subscriptions: subs,
			})
		}
		req.SNS = &infrastructure.SNSConfig{Topics: topics}
	}

	if dto.S3 != nil {
		var buckets []infrastructure.BucketConfig
		for _, b := range dto.S3.Buckets {
			buckets = append(buckets, infrastructure.BucketConfig{
				Name: b.Name,
				Seed: b.Seed,
			})
		}
		req.S3 = &infrastructure.S3Config{Buckets: buckets}
	}

	if dto.Tunnel != nil {
		var targets []infrastructure.TunnelTargetRequirement
		for _, t := range dto.Tunnel.Targets {
			targets = append(targets, infrastructure.TunnelTargetRequirement{
				Name: t.Name,
				Host: t.Host,
				Port: t.Port,
			})
		}
		req.Tunnel = &infrastructure.TunnelRequirement{
			Provider: dto.Tunnel.Provider,
			Targets:  targets,
		}
	}

	return req
}

func (r *ServiceRepositoryImpl) toConfigDTO(svc *service.Service) ServiceConfigDTO {
	// Convert secrets
	var secretsDTO map[string]SecretConfigDTO
	if len(svc.Environment.Secrets) > 0 {
		secretsDTO = make(map[string]SecretConfigDTO)
		for name, secret := range svc.Environment.Secrets {
			required := secret.Required
			secretsDTO[name] = SecretConfigDTO{
				Description: secret.Description,
				Required:    &required,
			}
		}
	}

	dto := ServiceConfigDTO{
		Version: "1",
		Service: ServiceInfoDTO{
			Name: svc.Name,
			Type: string(svc.Type),
			Port: svc.Port.Value(),
			Health: HealthConfigDTO{
				Endpoint: svc.Health.Endpoint,
				Interval: svc.Health.Interval.String(),
				Timeout:  svc.Health.Timeout.String(),
				Retries:  svc.Health.Retries,
			},
		},
		Requires: RequirementsDTO{
			Services: make([]string, len(svc.Dependencies.Services)),
		},
		Env:     svc.Environment.Variables,
		Secrets: secretsDTO,
	}

	for i, dep := range svc.Dependencies.Services {
		dto.Requires.Services[i] = dep.String()
	}

	if svc.Build != nil {
		dto.Service.Build = &BuildConfigDTO{
			Dockerfile: svc.Build.Dockerfile,
			Context:    svc.Build.Context,
		}
	}

	if svc.Run != nil {
		dto.Service.Run = &RunConfigDTO{
			Command:   svc.Run.Command,
			HotReload: svc.Run.HotReload,
		}
	}

	// Convert infrastructure requirements
	infraDTO := InfrastructureConfigDTO{}
	if svc.Dependencies.Infrastructure.Postgres != nil {
		infraDTO.Postgres = &PostgresConfigDTO{
			Database:   svc.Dependencies.Infrastructure.Postgres.Database,
			Migrations: svc.Dependencies.Infrastructure.Postgres.Migrations,
			Seed:       svc.Dependencies.Infrastructure.Postgres.Seed,
		}
	}
	// ... similar for other infrastructure types

	dto.Requires.Infrastructure = infraDTO

	return dto
}

func parseDuration(s string) (time.Duration, error) {
	// Simple parser for duration strings like "5s", "10m"
	// Can be enhanced
	return time.ParseDuration(s)
}

// ExtractTunnelConfig extracts just the tunnel configuration from a service path
// This is a lightweight operation that doesn't resolve environment placeholders
func (r *ServiceRepositoryImpl) ExtractTunnelConfig(name service.ServiceName) (*infrastructure.TunnelRequirement, error) {
	servicePath, err := r.registryRepo.GetServicePath(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get service path: %w", err)
	}

	configPath := filepath.Join(servicePath, "grund.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var configDTO ServiceConfigDTO
	if err := yaml.Unmarshal(data, &configDTO); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Only extract tunnel config
	if configDTO.Requires.Infrastructure.Tunnel == nil {
		return nil, nil
	}

	tunnelDTO := configDTO.Requires.Infrastructure.Tunnel
	var targets []infrastructure.TunnelTargetRequirement
	for _, t := range tunnelDTO.Targets {
		targets = append(targets, infrastructure.TunnelTargetRequirement{
			Name: t.Name,
			Host: t.Host,
			Port: t.Port,
		})
	}

	return &infrastructure.TunnelRequirement{
		Provider: tunnelDTO.Provider,
		Targets:  targets,
	}, nil
}
