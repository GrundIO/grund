package helpers

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/GrundIO/grund/internal/domain/infrastructure"
	"github.com/GrundIO/grund/internal/domain/service"
)

// GetFixturePath returns the absolute path to the test fixtures directory
func GetFixturePath(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get current file path")
	}
	return filepath.Join(filepath.Dir(filename), "..", "fixtures")
}

// GetFixtureServicesPath returns the path to the services fixtures
func GetFixtureServicesPath(t *testing.T, serviceName string) string {
	t.Helper()
	return filepath.Join(GetFixturePath(t), "services", serviceName)
}

// CreateTempDir creates a temporary directory for tests
func CreateTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "grund-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	return dir
}

// CreateTestService creates a service for testing
func CreateTestService(name string, port int, deps []string) *service.Service {
	p, _ := service.NewPort(port)

	serviceDeps := make([]service.ServiceName, len(deps))
	for i, d := range deps {
		serviceDeps[i] = service.ServiceName(d)
	}

	return &service.Service{
		Name: name,
		Type: service.ServiceTypeGo,
		Port: p,
		Build: &service.BuildConfig{
			Dockerfile: "Dockerfile",
			Context:    ".",
		},
		Health: service.HealthConfig{
			Endpoint: "/health",
			Interval: 5 * time.Second,
			Timeout:  3 * time.Second,
			Retries:  10,
		},
		Dependencies: service.ServiceDependencies{
			Services:       serviceDeps,
			Infrastructure: infrastructure.InfrastructureRequirements{},
		},
		Environment: service.Environment{
			Variables: map[string]string{"APP_ENV": "test"},
		},
	}
}

// CreateTestServiceWithInfra creates a service with infrastructure dependencies
func CreateTestServiceWithInfra(name string, port int, deps []string, infra infrastructure.InfrastructureRequirements) *service.Service {
	svc := CreateTestService(name, port, deps)
	svc.Dependencies.Infrastructure = infra
	return svc
}

// AssertNoError fails the test if err is not nil
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// AssertError fails the test if err is nil
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// AssertEqual fails the test if expected != actual
func AssertEqual(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

// AssertContains fails the test if slice does not contain item
func AssertContains(t *testing.T, slice []string, item string) {
	t.Helper()
	for _, s := range slice {
		if s == item {
			return
		}
	}
	t.Fatalf("slice %v does not contain %s", slice, item)
}

// AssertLen fails the test if len(slice) != expected
func AssertLen(t *testing.T, slice interface{}, expected int) {
	t.Helper()
	// This is a simplified version - in production you'd use reflection
	// For now, we'll just check common types
	switch v := slice.(type) {
	case []string:
		if len(v) != expected {
			t.Fatalf("expected length %d, got %d", expected, len(v))
		}
	case []service.ServiceName:
		if len(v) != expected {
			t.Fatalf("expected length %d, got %d", expected, len(v))
		}
	case []*service.Service:
		if len(v) != expected {
			t.Fatalf("expected length %d, got %d", expected, len(v))
		}
	default:
		t.Fatalf("AssertLen: unsupported type %T", slice)
	}
}
