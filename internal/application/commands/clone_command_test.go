package commands

import (
	"context"
	"fmt"
	"testing"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
	"github.com/Saturn-Fintech/grund/internal/domain/service"
)

// --- Mocks ---

type mockGitClient struct {
	cloneCalls []struct{ RepoURL, DestPath string }
	pullCalls  []string
	cloneErr   error
	pullErr    error
	isGitRepo  map[string]bool
}

func (m *mockGitClient) Clone(_ context.Context, repoURL string, destPath string) error {
	m.cloneCalls = append(m.cloneCalls, struct{ RepoURL, DestPath string }{repoURL, destPath})
	return m.cloneErr
}

func (m *mockGitClient) Pull(_ context.Context, repoPath string) error {
	m.pullCalls = append(m.pullCalls, repoPath)
	return m.pullErr
}

func (m *mockGitClient) IsGitRepository(path string) bool {
	if m.isGitRepo == nil {
		return false
	}
	return m.isGitRepo[path]
}

type mockCloneRegistryRepo struct {
	services map[service.ServiceName]ports.ServiceEntry
	err      error
}

func (m *mockCloneRegistryRepo) GetServicePath(name service.ServiceName) (string, error) {
	entry, ok := m.services[name]
	if !ok {
		return "", fmt.Errorf("service %s not found", name)
	}
	return entry.Path, nil
}

func (m *mockCloneRegistryRepo) GetAllServices() (map[service.ServiceName]ports.ServiceEntry, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.services, nil
}

// --- Tests ---

func TestCloneCommandHandler_CloneAll(t *testing.T) {
	registry := &mockCloneRegistryRepo{
		services: map[service.ServiceName]ports.ServiceEntry{
			"svc-a": {Repo: "git@github.com:org/svc-a.git", Path: "/tmp/test-clone/svc-a"},
			"svc-b": {Repo: "git@github.com:org/svc-b.git", Path: "/tmp/test-clone/svc-b"},
		},
	}
	gitClient := &mockGitClient{}
	handler := NewCloneCommandHandler(registry, gitClient)

	cmd := CloneCommand{} // no service names = all

	results, err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Handle() returned error: %v", err)
	}

	if len(gitClient.cloneCalls) != 2 {
		t.Errorf("Expected 2 clone calls, got %d", len(gitClient.cloneCalls))
	}

	cloned := countResultsByAction(results, "cloned")
	if cloned != 2 {
		t.Errorf("Expected 2 cloned results, got %d", cloned)
	}
}

func TestCloneCommandHandler_CloneSpecific(t *testing.T) {
	registry := &mockCloneRegistryRepo{
		services: map[service.ServiceName]ports.ServiceEntry{
			"svc-a": {Repo: "git@github.com:org/svc-a.git", Path: "/tmp/test-clone/svc-a"},
			"svc-b": {Repo: "git@github.com:org/svc-b.git", Path: "/tmp/test-clone/svc-b"},
		},
	}
	gitClient := &mockGitClient{}
	handler := NewCloneCommandHandler(registry, gitClient)

	cmd := CloneCommand{ServiceNames: []string{"svc-a"}}

	results, err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Handle() returned error: %v", err)
	}

	if len(gitClient.cloneCalls) != 1 {
		t.Errorf("Expected 1 clone call, got %d", len(gitClient.cloneCalls))
	}
	if gitClient.cloneCalls[0].RepoURL != "git@github.com:org/svc-a.git" {
		t.Errorf("Expected svc-a repo URL, got %s", gitClient.cloneCalls[0].RepoURL)
	}
	if len(results) != 1 || results[0].Action != "cloned" {
		t.Errorf("Expected 1 cloned result, got %v", results)
	}
}

func TestCloneCommandHandler_SkipExistingNoSync(t *testing.T) {
	registry := &mockCloneRegistryRepo{
		services: map[service.ServiceName]ports.ServiceEntry{
			"svc-a": {Repo: "git@github.com:org/svc-a.git", Path: "/tmp"}, // /tmp exists
		},
	}
	gitClient := &mockGitClient{
		isGitRepo: map[string]bool{"/tmp": true},
	}
	handler := NewCloneCommandHandler(registry, gitClient)

	cmd := CloneCommand{Sync: false}

	results, err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Handle() returned error: %v", err)
	}

	if len(gitClient.cloneCalls) != 0 {
		t.Errorf("Expected 0 clone calls, got %d", len(gitClient.cloneCalls))
	}
	if len(gitClient.pullCalls) != 0 {
		t.Errorf("Expected 0 pull calls, got %d", len(gitClient.pullCalls))
	}
	if countResultsByAction(results, "skipped") != 1 {
		t.Errorf("Expected 1 skipped result, got %v", results)
	}
}

func TestCloneCommandHandler_SyncExisting(t *testing.T) {
	registry := &mockCloneRegistryRepo{
		services: map[service.ServiceName]ports.ServiceEntry{
			"svc-a": {Repo: "git@github.com:org/svc-a.git", Path: "/tmp"}, // /tmp exists
		},
	}
	gitClient := &mockGitClient{
		isGitRepo: map[string]bool{"/tmp": true},
	}
	handler := NewCloneCommandHandler(registry, gitClient)

	cmd := CloneCommand{Sync: true}

	results, err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Handle() returned error: %v", err)
	}

	if len(gitClient.cloneCalls) != 0 {
		t.Errorf("Expected 0 clone calls, got %d", len(gitClient.cloneCalls))
	}
	if len(gitClient.pullCalls) != 1 {
		t.Errorf("Expected 1 pull call, got %d", len(gitClient.pullCalls))
	}
	if countResultsByAction(results, "pulled") != 1 {
		t.Errorf("Expected 1 pulled result, got %v", results)
	}
}

func TestCloneCommandHandler_ExistsButNotGitRepo(t *testing.T) {
	registry := &mockCloneRegistryRepo{
		services: map[service.ServiceName]ports.ServiceEntry{
			"svc-a": {Repo: "git@github.com:org/svc-a.git", Path: "/tmp"}, // /tmp exists
		},
	}
	gitClient := &mockGitClient{
		isGitRepo: map[string]bool{"/tmp": false},
	}
	handler := NewCloneCommandHandler(registry, gitClient)

	cmd := CloneCommand{}

	results, err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Handle() returned unexpected error: %v", err)
	}

	if countResultsByAction(results, "error") != 1 {
		t.Errorf("Expected 1 error result, got %v", results)
	}
	if results[0].Error == nil {
		t.Error("Expected error in result for non-git directory")
	}
}

func TestCloneCommandHandler_ServiceNotFound(t *testing.T) {
	registry := &mockCloneRegistryRepo{
		services: map[service.ServiceName]ports.ServiceEntry{
			"svc-a": {Repo: "git@github.com:org/svc-a.git", Path: "/tmp/svc-a"},
		},
	}
	gitClient := &mockGitClient{}
	handler := NewCloneCommandHandler(registry, gitClient)

	cmd := CloneCommand{ServiceNames: []string{"nonexistent"}}

	_, err := handler.Handle(context.Background(), cmd)
	if err == nil {
		t.Fatal("Expected error for nonexistent service, got nil")
	}
}

func TestCloneCommandHandler_NoRepoField(t *testing.T) {
	registry := &mockCloneRegistryRepo{
		services: map[service.ServiceName]ports.ServiceEntry{
			"svc-a": {Repo: "", Path: "/tmp/svc-a"},
		},
	}
	gitClient := &mockGitClient{}
	handler := NewCloneCommandHandler(registry, gitClient)

	cmd := CloneCommand{ServiceNames: []string{"svc-a"}}

	_, err := handler.Handle(context.Background(), cmd)
	if err == nil {
		t.Fatal("Expected error for service with no repo, got nil")
	}
}

func TestCloneCommandHandler_NoPathField(t *testing.T) {
	registry := &mockCloneRegistryRepo{
		services: map[service.ServiceName]ports.ServiceEntry{
			"svc-a": {Repo: "git@github.com:org/svc-a.git", Path: ""},
		},
	}
	gitClient := &mockGitClient{}
	handler := NewCloneCommandHandler(registry, gitClient)

	cmd := CloneCommand{ServiceNames: []string{"svc-a"}}

	_, err := handler.Handle(context.Background(), cmd)
	if err == nil {
		t.Fatal("Expected error for service with no path, got nil")
	}
}

func TestCloneCommandHandler_PartialFailure(t *testing.T) {
	registry := &mockCloneRegistryRepo{
		services: map[service.ServiceName]ports.ServiceEntry{
			"svc-a": {Repo: "git@github.com:org/svc-a.git", Path: "/tmp/test-clone-partial/svc-a"},
			"svc-b": {Repo: "git@github.com:org/svc-b.git", Path: "/tmp/test-clone-partial/svc-b"},
		},
	}
	gitClient := &mockGitClient{
		cloneErr: fmt.Errorf("network error"),
	}
	handler := NewCloneCommandHandler(registry, gitClient)

	cmd := CloneCommand{}

	results, err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Handle() returned unexpected error: %v", err)
	}

	errorCount := countResultsByAction(results, "error")
	if errorCount != 2 {
		t.Errorf("Expected 2 error results, got %d", errorCount)
	}
}

func TestCloneCommandHandler_RegistryError(t *testing.T) {
	registry := &mockCloneRegistryRepo{
		err: fmt.Errorf("failed to read registry"),
	}
	gitClient := &mockGitClient{}
	handler := NewCloneCommandHandler(registry, gitClient)

	cmd := CloneCommand{}

	_, err := handler.Handle(context.Background(), cmd)
	if err == nil {
		t.Fatal("Expected error from registry failure, got nil")
	}
}

func TestCloneCommandHandler_NoServicesWithRepo(t *testing.T) {
	registry := &mockCloneRegistryRepo{
		services: map[service.ServiceName]ports.ServiceEntry{
			"svc-a": {Repo: "", Path: "/tmp/svc-a"},
		},
	}
	gitClient := &mockGitClient{}
	handler := NewCloneCommandHandler(registry, gitClient)

	// Clone all — but none have repo
	cmd := CloneCommand{}

	results, err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Handle() returned error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
	if len(gitClient.cloneCalls) != 0 {
		t.Errorf("Expected 0 clone calls, got %d", len(gitClient.cloneCalls))
	}
}

func TestCloneCommandHandler_PullError(t *testing.T) {
	registry := &mockCloneRegistryRepo{
		services: map[service.ServiceName]ports.ServiceEntry{
			"svc-a": {Repo: "git@github.com:org/svc-a.git", Path: "/tmp"}, // /tmp exists
		},
	}
	gitClient := &mockGitClient{
		isGitRepo: map[string]bool{"/tmp": true},
		pullErr:   fmt.Errorf("merge conflict"),
	}
	handler := NewCloneCommandHandler(registry, gitClient)

	cmd := CloneCommand{Sync: true}

	results, err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Handle() returned unexpected error: %v", err)
	}

	if countResultsByAction(results, "error") != 1 {
		t.Errorf("Expected 1 error result, got %v", results)
	}
}

func TestExpandTilde(t *testing.T) {
	// Test non-tilde path stays unchanged
	result := expandTilde("/usr/local/bin")
	if result != "/usr/local/bin" {
		t.Errorf("Expected /usr/local/bin, got %s", result)
	}

	// Test empty path
	result = expandTilde("")
	if result != "" {
		t.Errorf("Expected empty string, got %s", result)
	}
}

// countResultsByAction counts results with the given action
func countResultsByAction(results []ports.CloneResult, action string) int {
	count := 0
	for _, r := range results {
		if r.Action == action {
			count++
		}
	}
	return count
}
