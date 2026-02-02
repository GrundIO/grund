package config

import "time"

// ServiceConfig represents the grund.yaml in each service
// All values in Env support ${placeholder} syntax for dynamic resolution
type ServiceConfig struct {
	Version  string                  `yaml:"version"`
	Service  ServiceInfo             `yaml:"service"`
	Requires Requirements            `yaml:"requires"`
	Env      map[string]string       `yaml:"env"` // All values support ${placeholder} syntax
	Secrets  map[string]SecretConfig `yaml:"secrets,omitempty"`
}

// SecretConfig defines a secret required by the service
type SecretConfig struct {
	Description string `yaml:"description"`
	Required    *bool  `yaml:"required,omitempty"` // nil defaults to true
}

// IsRequired returns true if the secret is required (default: true)
func (s SecretConfig) IsRequired() bool {
	if s.Required == nil {
		return true // default to required
	}
	return *s.Required
}

type ServiceInfo struct {
	Name   string       `yaml:"name"`
	Type   string       `yaml:"type"` // go, python, node
	Port   int          `yaml:"port"`
	Build  *BuildConfig `yaml:"build,omitempty"`
	Run    *RunConfig   `yaml:"run,omitempty"`
	Health HealthConfig `yaml:"health"`
}

type BuildConfig struct {
	Dockerfile string `yaml:"dockerfile"`
	Context    string `yaml:"context"`
}

type RunConfig struct {
	Command   string `yaml:"command"`
	HotReload bool   `yaml:"hot_reload"`
}

type HealthConfig struct {
	Endpoint string        `yaml:"endpoint"`
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
	Retries  int           `yaml:"retries"`
}

type Requirements struct {
	Services       []string             `yaml:"services"`
	Infrastructure InfrastructureConfig `yaml:"infrastructure"`
}

type InfrastructureConfig struct {
	Postgres *PostgresConfig `yaml:"postgres,omitempty"`
	MongoDB  *MongoDBConfig  `yaml:"mongodb,omitempty"`
	Redis    interface{}     `yaml:"redis,omitempty"` // bool or RedisConfig
	SQS      *SQSConfig      `yaml:"sqs,omitempty"`
	SNS      *SNSConfig      `yaml:"sns,omitempty"`
	S3       *S3Config       `yaml:"s3,omitempty"`
	Tunnel   *TunnelConfig   `yaml:"tunnel,omitempty"`
}

type PostgresConfig struct {
	Database   string `yaml:"database"`
	Migrations string `yaml:"migrations,omitempty"`
	Seed       string `yaml:"seed,omitempty"`
}

type MongoDBConfig struct {
	Database string `yaml:"database"`
	Seed     string `yaml:"seed,omitempty"`
}

type RedisConfig struct {
	// Future: specific redis config if needed
}

type SQSConfig struct {
	Queues []QueueConfig `yaml:"queues"`
}

type QueueConfig struct {
	Name string `yaml:"name"`
	DLQ  bool   `yaml:"dlq,omitempty"`
}

type SNSConfig struct {
	Topics []TopicConfig `yaml:"topics"`
}

type TopicConfig struct {
	Name          string               `yaml:"name"`
	Subscriptions []SubscriptionConfig `yaml:"subscriptions,omitempty"`
}

type SubscriptionConfig struct {
	Protocol   string            `yaml:"protocol"`
	Endpoint   string            `yaml:"endpoint"`
	Attributes map[string]string `yaml:"attributes,omitempty"`
}

type S3Config struct {
	Buckets []BucketConfig `yaml:"buckets"`
}

type BucketConfig struct {
	Name string `yaml:"name"`
	Seed string `yaml:"seed,omitempty"`
}

type TunnelConfig struct {
	Provider string         `yaml:"provider"` // "cloudflared" or "ngrok"
	Targets  []TunnelTarget `yaml:"targets"`
}

type TunnelTarget struct {
	Name string `yaml:"name"` // identifier for ${tunnel.<name>.url}
	Host string `yaml:"host"` // supports placeholders
	Port string `yaml:"port"` // string to support placeholders
}

// ServiceRegistry represents the services.yaml in the orchestration repo
// Note: This is the legacy format - new configs use GlobalConfig.Services directly
type ServiceRegistry struct {
	Version      string                  `yaml:"version"`
	Services     map[string]ServiceEntry `yaml:"services"`
	PathDefaults PathDefaults            `yaml:"path_defaults"`
}

// PathDefaults contains default path settings
type PathDefaults struct {
	Base string `yaml:"base"`
}
