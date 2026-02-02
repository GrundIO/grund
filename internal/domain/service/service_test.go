package service

import (
	"testing"
	"time"

	"github.com/Saturn-Fintech/grund/internal/domain/infrastructure"
)

func TestNewPort_Valid(t *testing.T) {
	tests := []struct {
		name  string
		value int
	}{
		{"minimum valid port", 1},
		{"common http port", 80},
		{"common https port", 443},
		{"typical app port", 8080},
		{"maximum valid port", 65535},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port, err := NewPort(tt.value)
			if err != nil {
				t.Errorf("NewPort(%d) returned error: %v", tt.value, err)
			}
			if port.Value() != tt.value {
				t.Errorf("NewPort(%d).Value() = %d, want %d", tt.value, port.Value(), tt.value)
			}
		})
	}
}

func TestNewPort_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		value int
	}{
		{"zero port", 0},
		{"negative port", -1},
		{"port too high", 65536},
		{"very negative", -1000},
		{"very high", 100000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPort(tt.value)
			if err == nil {
				t.Errorf("NewPort(%d) expected error, got nil", tt.value)
			}
		})
	}
}

func TestService_Validate_Success(t *testing.T) {
	port, _ := NewPort(8080)
	svc := &Service{
		Name: "test-service",
		Type: ServiceTypeGo,
		Port: port,
		Build: &BuildConfig{
			Dockerfile: "Dockerfile",
			Context:    ".",
		},
		Health: HealthConfig{
			Endpoint: "/health",
			Interval: 5 * time.Second,
			Timeout:  3 * time.Second,
			Retries:  10,
		},
		Dependencies: ServiceDependencies{
			Services:       []ServiceName{},
			Infrastructure: infrastructure.InfrastructureRequirements{},
		},
	}

	if err := svc.Validate(); err != nil {
		t.Errorf("Validate() returned error for valid service: %v", err)
	}
}

func TestService_Validate_WithRunConfig(t *testing.T) {
	port, _ := NewPort(8080)
	svc := &Service{
		Name: "test-service",
		Type: ServiceTypeGo,
		Port: port,
		Run: &RunConfig{
			Command:   "go run ./cmd/main.go",
			HotReload: true,
		},
		Health: HealthConfig{
			Endpoint: "/health",
		},
	}

	if err := svc.Validate(); err != nil {
		t.Errorf("Validate() returned error for valid service with run config: %v", err)
	}
}

func TestService_Validate_MissingName(t *testing.T) {
	port, _ := NewPort(8080)
	svc := &Service{
		Name: "", // Missing name
		Port: port,
		Build: &BuildConfig{
			Dockerfile: "Dockerfile",
			Context:    ".",
		},
		Health: HealthConfig{
			Endpoint: "/health",
		},
	}

	if err := svc.Validate(); err == nil {
		t.Error("Validate() expected error for missing name, got nil")
	}
}

func TestService_Validate_MissingBuildAndRun(t *testing.T) {
	port, _ := NewPort(8080)
	svc := &Service{
		Name:  "test-service",
		Port:  port,
		Build: nil, // No build
		Run:   nil, // No run
		Health: HealthConfig{
			Endpoint: "/health",
		},
	}

	if err := svc.Validate(); err == nil {
		t.Error("Validate() expected error for missing build and run, got nil")
	}
}

func TestService_Validate_MissingHealthEndpoint(t *testing.T) {
	port, _ := NewPort(8080)
	svc := &Service{
		Name: "test-service",
		Port: port,
		Build: &BuildConfig{
			Dockerfile: "Dockerfile",
			Context:    ".",
		},
		Health: HealthConfig{
			Endpoint: "", // Missing endpoint
		},
	}

	if err := svc.Validate(); err == nil {
		t.Error("Validate() expected error for missing health endpoint, got nil")
	}
}

func TestService_RequiresInfrastructure(t *testing.T) {
	tests := []struct {
		name      string
		infra     infrastructure.InfrastructureRequirements
		infraType string
		expected  bool
	}{
		{
			name: "has postgres",
			infra: infrastructure.InfrastructureRequirements{
				Postgres: &infrastructure.PostgresConfig{Database: "testdb"},
			},
			infraType: "postgres",
			expected:  true,
		},
		{
			name:      "no postgres",
			infra:     infrastructure.InfrastructureRequirements{},
			infraType: "postgres",
			expected:  false,
		},
		{
			name: "has mongodb",
			infra: infrastructure.InfrastructureRequirements{
				MongoDB: &infrastructure.MongoDBConfig{Database: "testdb"},
			},
			infraType: "mongodb",
			expected:  true,
		},
		{
			name: "has redis",
			infra: infrastructure.InfrastructureRequirements{
				Redis: &infrastructure.RedisConfig{},
			},
			infraType: "redis",
			expected:  true,
		},
		{
			name: "has localstack via SQS",
			infra: infrastructure.InfrastructureRequirements{
				SQS: &infrastructure.SQSConfig{
					Queues: []infrastructure.QueueConfig{{Name: "test-queue"}},
				},
			},
			infraType: "localstack",
			expected:  true,
		},
		{
			name: "has localstack via SNS",
			infra: infrastructure.InfrastructureRequirements{
				SNS: &infrastructure.SNSConfig{
					Topics: []infrastructure.TopicConfig{{Name: "test-topic"}},
				},
			},
			infraType: "localstack",
			expected:  true,
		},
		{
			name: "has localstack via S3",
			infra: infrastructure.InfrastructureRequirements{
				S3: &infrastructure.S3Config{
					Buckets: []infrastructure.BucketConfig{{Name: "test-bucket"}},
				},
			},
			infraType: "localstack",
			expected:  true,
		},
		{
			name:      "unknown infra type",
			infra:     infrastructure.InfrastructureRequirements{},
			infraType: "unknown",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port, _ := NewPort(8080)
			svc := &Service{
				Name: "test-service",
				Port: port,
				Build: &BuildConfig{
					Dockerfile: "Dockerfile",
					Context:    ".",
				},
				Health: HealthConfig{
					Endpoint: "/health",
				},
				Dependencies: ServiceDependencies{
					Services:       []ServiceName{},
					Infrastructure: tt.infra,
				},
			}

			result := svc.RequiresInfrastructure(tt.infraType)
			if result != tt.expected {
				t.Errorf("RequiresInfrastructure(%q) = %v, want %v", tt.infraType, result, tt.expected)
			}
		})
	}
}

func TestServiceName_String(t *testing.T) {
	name := ServiceName("my-service")
	if name.String() != "my-service" {
		t.Errorf("ServiceName.String() = %q, want %q", name.String(), "my-service")
	}
}

func TestServiceType_Constants(t *testing.T) {
	if ServiceTypeGo != "go" {
		t.Errorf("ServiceTypeGo = %q, want %q", ServiceTypeGo, "go")
	}
	if ServiceTypePython != "python" {
		t.Errorf("ServiceTypePython = %q, want %q", ServiceTypePython, "python")
	}
	if ServiceTypeNode != "node" {
		t.Errorf("ServiceTypeNode = %q, want %q", ServiceTypeNode, "node")
	}
}
