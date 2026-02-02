package docker

import (
	"path/filepath"
	"testing"

	"github.com/Saturn-Fintech/grund/internal/config"
)

func TestNewDockerOrchestrator(t *testing.T) {
	orchestrator := NewDockerOrchestrator("/working/dir")

	if orchestrator == nil {
		t.Fatal("NewDockerOrchestrator() returned nil")
	}

	// Verify it implements the interface by type assertion
	do, ok := orchestrator.(*DockerOrchestrator)
	if !ok {
		t.Error("NewDockerOrchestrator() should return *DockerOrchestrator")
	}

	if do.projectName != ProjectName {
		t.Errorf("Expected projectName %q, got %q", ProjectName, do.projectName)
	}
}

func TestGetGrundTmpDir(t *testing.T) {
	grundHome, err := config.GetGrundHome()
	if err != nil {
		t.Fatalf("Failed to get grund home: %v", err)
	}

	tmpDir, err := GetGrundTmpDir()
	if err != nil {
		t.Fatalf("GetGrundTmpDir() returned error: %v", err)
	}

	expected := filepath.Join(grundHome, "tmp")
	if tmpDir != expected {
		t.Errorf("GetGrundTmpDir() = %q, want %q", tmpDir, expected)
	}
}

func TestGetInfrastructureComposePath(t *testing.T) {
	grundHome, err := config.GetGrundHome()
	if err != nil {
		t.Fatalf("Failed to get grund home: %v", err)
	}

	path, err := GetInfrastructureComposePath()
	if err != nil {
		t.Fatalf("GetInfrastructureComposePath() returned error: %v", err)
	}

	expected := filepath.Join(grundHome, "tmp", "infrastructure", "docker-compose.yaml")
	if path != expected {
		t.Errorf("GetInfrastructureComposePath() = %q, want %q", path, expected)
	}
}

func TestGetServiceComposePath(t *testing.T) {
	grundHome, err := config.GetGrundHome()
	if err != nil {
		t.Fatalf("Failed to get grund home: %v", err)
	}

	tests := []struct {
		serviceName string
		expected    string
	}{
		{"mars", filepath.Join(grundHome, "tmp", "mars", "docker-compose.yaml")},
		{"shuttle", filepath.Join(grundHome, "tmp", "shuttle", "docker-compose.yaml")},
	}

	for _, tt := range tests {
		t.Run(tt.serviceName, func(t *testing.T) {
			path, err := GetServiceComposePath(tt.serviceName)
			if err != nil {
				t.Fatalf("GetServiceComposePath(%q) returned error: %v", tt.serviceName, err)
			}
			if path != tt.expected {
				t.Errorf("GetServiceComposePath(%q) = %q, want %q", tt.serviceName, path, tt.expected)
			}
		})
	}
}

func TestDockerOrchestrator_SetComposeFiles(t *testing.T) {
	orchestrator := NewDockerOrchestrator("/test/workdir").(*DockerOrchestrator)

	// Files with "infrastructure" in the path are treated as infrastructure files
	files := []string{"/path/infrastructure/docker-compose.yaml", "/path/mars/docker-compose.yaml"}
	orchestrator.SetComposeFiles(files)

	if orchestrator.infrastructureFile != "/path/infrastructure/docker-compose.yaml" {
		t.Errorf("Expected infrastructure file to be set, got %q", orchestrator.infrastructureFile)
	}

	if len(orchestrator.serviceFiles) != 1 {
		t.Errorf("Expected 1 service file, got %d", len(orchestrator.serviceFiles))
	}

	if orchestrator.serviceFiles[0] != "/path/mars/docker-compose.yaml" {
		t.Errorf("Expected service file to be /path/mars/docker-compose.yaml, got %q", orchestrator.serviceFiles[0])
	}
}

func TestDockerOrchestrator_composeArgs(t *testing.T) {
	orchestrator := NewDockerOrchestrator("/test/workdir").(*DockerOrchestrator)
	orchestrator.SetComposeFiles([]string{"/path/infrastructure/docker-compose.yaml", "/path/svc/docker-compose.yaml"})

	args := orchestrator.composeArgs()

	// Expected: compose -p grund -f /path/infrastructure/docker-compose.yaml -f /path/svc/docker-compose.yaml
	expected := []string{"compose", "-p", ProjectName, "-f", "/path/infrastructure/docker-compose.yaml", "-f", "/path/svc/docker-compose.yaml"}
	if len(args) != len(expected) {
		t.Fatalf("Expected %d args, got %d: %v", len(expected), len(args), args)
	}

	for i, arg := range expected {
		if args[i] != arg {
			t.Errorf("Arg %d: expected %q, got %q", i, arg, args[i])
		}
	}
}

func TestGetProjectName(t *testing.T) {
	// GetProjectName should always return the static "grund" project name
	result := GetProjectName()
	if result != ProjectName {
		t.Errorf("GetProjectName() = %q, want %q", result, ProjectName)
	}
}

// Note: Integration tests for StartServices, StopServices, RestartService
// would require Docker to be available. These are better suited for
// end-to-end tests or tests with mocked exec.Command.
// Below we document what those tests would verify:

// TestDockerOrchestrator_StartServices_Integration would verify:
// - Correct docker compose command is executed
// - Services are passed as arguments
// - -d flag is used for detached mode
// - Errors from docker are propagated

// TestDockerOrchestrator_StopServices_Integration would verify:
// - docker compose down is executed
// - Correct compose file is specified
// - Working directory is correct

// TestDockerOrchestrator_RestartService_Integration would verify:
// - docker compose restart is executed with service name
// - Errors are handled appropriately
