package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
	"github.com/Saturn-Fintech/grund/internal/domain/service"
)

// mockRegistryRepo implements ports.ServiceRegistryRepository for testing
type mockRegistryRepo struct {
	paths map[string]string
}

func (m *mockRegistryRepo) GetServicePath(name service.ServiceName) (string, error) {
	return m.paths[name.String()], nil
}

func (m *mockRegistryRepo) GetAllServices() (map[service.ServiceName]ports.ServiceEntry, error) {
	result := make(map[service.ServiceName]ports.ServiceEntry)
	for name, path := range m.paths {
		result[service.ServiceName(name)] = ports.ServiceEntry{Path: path}
	}
	return result, nil
}

func createTestGrundYaml(t *testing.T, dir string) {
	t.Helper()

	content := `version: "1"

service:
  name: test-service
  type: go
  port: 8080
  build:
    dockerfile: Dockerfile
    context: .
  health:
    endpoint: /health
    interval: 5s
    timeout: 3s
    retries: 10

requires:
  services:
    - dep-service
  infrastructure:
    postgres:
      database: testdb
      migrations: ./migrations
    redis: true
    sqs:
      queues:
        - name: test-queue
          dlq: true

env:
  APP_ENV: test
  DATABASE_URL: "postgres://localhost:5432/${self.postgres.database}"
`

	err := os.MkdirAll(dir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	err = os.WriteFile(filepath.Join(dir, "grund.yaml"), []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
}

func TestServiceRepository_FindByName(t *testing.T) {
	// Create temp directory with test config
	tmpDir := t.TempDir()
	svcDir := filepath.Join(tmpDir, "test-service")
	createTestGrundYaml(t, svcDir)

	registry := &mockRegistryRepo{
		paths: map[string]string{
			"test-service": svcDir,
		},
	}

	repo := NewServiceRepository(registry)

	svc, err := repo.FindByName(service.ServiceName("test-service"))
	if err != nil {
		t.Fatalf("FindByName() returned error: %v", err)
	}

	// Verify basic service info
	if svc.Name != "test-service" {
		t.Errorf("Expected name 'test-service', got %q", svc.Name)
	}
	if svc.Type != service.ServiceTypeGo {
		t.Errorf("Expected type 'go', got %q", svc.Type)
	}
	if svc.Port.Value() != 8080 {
		t.Errorf("Expected port 8080, got %d", svc.Port.Value())
	}

	// Verify build config
	if svc.Build == nil {
		t.Fatal("Expected build config, got nil")
	}
	if svc.Build.Dockerfile != "Dockerfile" {
		t.Errorf("Expected dockerfile 'Dockerfile', got %q", svc.Build.Dockerfile)
	}

	// Verify health config
	if svc.Health.Endpoint != "/health" {
		t.Errorf("Expected health endpoint '/health', got %q", svc.Health.Endpoint)
	}
	if svc.Health.Retries != 10 {
		t.Errorf("Expected 10 retries, got %d", svc.Health.Retries)
	}

	// Verify service dependencies
	if len(svc.Dependencies.Services) != 1 {
		t.Fatalf("Expected 1 service dependency, got %d", len(svc.Dependencies.Services))
	}
	if svc.Dependencies.Services[0] != "dep-service" {
		t.Errorf("Expected dependency 'dep-service', got %q", svc.Dependencies.Services[0])
	}

	// Verify infrastructure requirements
	if svc.Dependencies.Infrastructure.Postgres == nil {
		t.Error("Expected postgres config")
	} else if svc.Dependencies.Infrastructure.Postgres.Database != "testdb" {
		t.Errorf("Expected postgres database 'testdb', got %q", svc.Dependencies.Infrastructure.Postgres.Database)
	}

	if svc.Dependencies.Infrastructure.Redis == nil {
		t.Error("Expected redis config")
	}

	if svc.Dependencies.Infrastructure.SQS == nil {
		t.Error("Expected SQS config")
	} else if len(svc.Dependencies.Infrastructure.SQS.Queues) != 1 {
		t.Errorf("Expected 1 SQS queue, got %d", len(svc.Dependencies.Infrastructure.SQS.Queues))
	}

	// Verify environment
	if svc.Environment.Variables["APP_ENV"] != "test" {
		t.Errorf("Expected APP_ENV='test', got %q", svc.Environment.Variables["APP_ENV"])
	}
}

func TestServiceRepository_FindByName_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	registry := &mockRegistryRepo{
		paths: map[string]string{
			"test-service": tmpDir, // Directory exists but no grund.yaml
		},
	}

	repo := NewServiceRepository(registry)

	_, err := repo.FindByName(service.ServiceName("test-service"))
	if err == nil {
		t.Error("Expected error for missing config, got nil")
	}
}

func TestServiceRepository_FindAll(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two service directories
	svc1Dir := filepath.Join(tmpDir, "service-1")
	svc2Dir := filepath.Join(tmpDir, "service-2")

	createTestGrundYaml(t, svc1Dir)
	createTestGrundYaml(t, svc2Dir)

	registry := &mockRegistryRepo{
		paths: map[string]string{
			"service-1": svc1Dir,
			"service-2": svc2Dir,
		},
	}

	repo := NewServiceRepository(registry)

	services, err := repo.FindAll()
	if err != nil {
		t.Fatalf("FindAll() returned error: %v", err)
	}

	if len(services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(services))
	}
}

func TestServiceRepository_InvalidYaml(t *testing.T) {
	tmpDir := t.TempDir()

	// Write invalid YAML
	err := os.WriteFile(filepath.Join(tmpDir, "grund.yaml"), []byte("invalid: yaml: content:"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	registry := &mockRegistryRepo{
		paths: map[string]string{
			"test-service": tmpDir,
		},
	}

	repo := NewServiceRepository(registry)

	_, err = repo.FindByName(service.ServiceName("test-service"))
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestServiceRepository_InvalidPort(t *testing.T) {
	tmpDir := t.TempDir()

	// YAML with invalid port
	content := `version: "1"
service:
  name: test-service
  type: go
  port: 0
  build:
    dockerfile: Dockerfile
    context: .
  health:
    endpoint: /health
`

	err := os.WriteFile(filepath.Join(tmpDir, "grund.yaml"), []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	registry := &mockRegistryRepo{
		paths: map[string]string{
			"test-service": tmpDir,
		},
	}

	repo := NewServiceRepository(registry)

	_, err = repo.FindByName(service.ServiceName("test-service"))
	if err == nil {
		t.Error("Expected error for invalid port, got nil")
	}
}
