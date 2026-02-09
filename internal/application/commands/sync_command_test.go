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

type mockSyncRegistryRepo struct {
	services map[service.ServiceName]ports.ServiceEntry
	err      error
}

func (m *mockSyncRegistryRepo) GetServicePath(name service.ServiceName) (string, error) {
	entry, ok := m.services[name]
	if !ok {
		return "", fmt.Errorf("service %s not found", name)
	}
	return entry.Path, nil
}

func (m *mockSyncRegistryRepo) GetAllServices() (map[service.ServiceName]ports.ServiceEntry, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.services, nil
}

// --- Tests ---

func TestSyncCommandHandler_SyncAll(t *testing.T) {
	registry := &mockSyncRegistryRepo{
		services: map[service.ServiceName]ports.ServiceEntry{
			"svc-a": {Repo: "git@github.com:org/svc-a.git", Path: "/tmp/test-sync/svc-a"},
			"svc-b": {Repo: "git@github.com:org/svc-b.git", Path: "/tmp/test-sync/svc-b"},
		},
	}
	gitClient := &mockGitClient{}
	handler := NewSyncCommandHandler(registry, gitClient)

	cmd := SyncCommand{} // no service names = all

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

func TestSyncCommandHandler_SyncSpecific(t *testing.T) {
	registry := &mockSyncRegistryRepo{
		services: map[service.ServiceName]ports.ServiceEntry{
			"svc-a": {Repo: "git@github.com:org/svc-a.git", Path: "/tmp/test-sync/svc-a"},
			"svc-b": {Repo: "git@github.com:org/svc-b.git", Path: "/tmp/test-sync/svc-b"},
		},
	}
	gitClient := &mockGitClient{}
	handler := NewSyncCommandHandler(registry, gitClient)

	cmd := SyncCommand{ServiceNames: []string{"svc-a"}}

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

func TestSyncCommandHandler_PullExisting(t *testing.T) {
	registry := &mockSyncRegistryRepo{
		services: map[service.ServiceName]ports.ServiceEntry{
			"svc-a": {Repo: "git@github.com:org/svc-a.git", Path: "/tmp"}, // /tmp exists
		},
	}
	gitClient := &mockGitClient{
		isGitRepo: map[string]bool{"/tmp": true},
	}
	handler := NewSyncCommandHandler(registry, gitClient)

	cmd := SyncCommand{} // default: pull existing

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

func TestSyncCommandHandler_NoPullSkipsExisting(t *testing.T) {
	registry := &mockSyncRegistryRepo{
		services: map[service.ServiceName]ports.ServiceEntry{
			"svc-a": {Repo: "git@github.com:org/svc-a.git", Path: "/tmp"}, // /tmp exists
		},
	}
	gitClient := &mockGitClient{
		isGitRepo: map[string]bool{"/tmp": true},
	}
	handler := NewSyncCommandHandler(registry, gitClient)

	cmd := SyncCommand{NoPull: true}

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

func TestSyncCommandHandler_ExistsButNotGitRepo(t *testing.T) {
	registry := &mockSyncRegistryRepo{
		services: map[service.ServiceName]ports.ServiceEntry{
			"svc-a": {Repo: "git@github.com:org/svc-a.git", Path: "/tmp"}, // /tmp exists
		},
	}
	gitClient := &mockGitClient{
		isGitRepo: map[string]bool{"/tmp": false},
	}
	handler := NewSyncCommandHandler(registry, gitClient)

	cmd := SyncCommand{}

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

func TestSyncCommandHandler_ServiceNotFound(t *testing.T) {
	registry := &mockSyncRegistryRepo{
		services: map[service.ServiceName]ports.ServiceEntry{
			"svc-a": {Repo: "git@github.com:org/svc-a.git", Path: "/tmp/svc-a"},
		},
	}
	gitClient := &mockGitClient{}
	handler := NewSyncCommandHandler(registry, gitClient)

	cmd := SyncCommand{ServiceNames: []string{"nonexistent"}}

	_, err := handler.Handle(context.Background(), cmd)
	if err == nil {
		t.Fatal("Expected error for nonexistent service, got nil")
	}
}

func TestSyncCommandHandler_NoRepoField(t *testing.T) {
	registry := &mockSyncRegistryRepo{
		services: map[service.ServiceName]ports.ServiceEntry{
			"svc-a": {Repo: "", Path: "/tmp/svc-a"},
		},
	}
	gitClient := &mockGitClient{}
	handler := NewSyncCommandHandler(registry, gitClient)

	cmd := SyncCommand{ServiceNames: []string{"svc-a"}}

	_, err := handler.Handle(context.Background(), cmd)
	if err == nil {
		t.Fatal("Expected error for service with no repo, got nil")
	}
}

func TestSyncCommandHandler_NoPathField(t *testing.T) {
	registry := &mockSyncRegistryRepo{
		services: map[service.ServiceName]ports.ServiceEntry{
			"svc-a": {Repo: "git@github.com:org/svc-a.git", Path: ""},
		},
	}
	gitClient := &mockGitClient{}
	handler := NewSyncCommandHandler(registry, gitClient)

	cmd := SyncCommand{ServiceNames: []string{"svc-a"}}

	_, err := handler.Handle(context.Background(), cmd)
	if err == nil {
		t.Fatal("Expected error for service with no path, got nil")
	}
}

func TestSyncCommandHandler_PartialFailure(t *testing.T) {
	registry := &mockSyncRegistryRepo{
		services: map[service.ServiceName]ports.ServiceEntry{
			"svc-a": {Repo: "git@github.com:org/svc-a.git", Path: "/tmp/test-sync-partial/svc-a"},
			"svc-b": {Repo: "git@github.com:org/svc-b.git", Path: "/tmp/test-sync-partial/svc-b"},
		},
	}
	gitClient := &mockGitClient{
		cloneErr: fmt.Errorf("network error"),
	}
	handler := NewSyncCommandHandler(registry, gitClient)

	cmd := SyncCommand{}

	results, err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Handle() returned unexpected error: %v", err)
	}

	errorCount := countResultsByAction(results, "error")
	if errorCount != 2 {
		t.Errorf("Expected 2 error results, got %d", errorCount)
	}
}

func TestSyncCommandHandler_RegistryError(t *testing.T) {
	registry := &mockSyncRegistryRepo{
		err: fmt.Errorf("failed to read registry"),
	}
	gitClient := &mockGitClient{}
	handler := NewSyncCommandHandler(registry, gitClient)

	cmd := SyncCommand{}

	_, err := handler.Handle(context.Background(), cmd)
	if err == nil {
		t.Fatal("Expected error from registry failure, got nil")
	}
}

func TestSyncCommandHandler_NoServicesWithRepo(t *testing.T) {
	registry := &mockSyncRegistryRepo{
		services: map[service.ServiceName]ports.ServiceEntry{
			"svc-a": {Repo: "", Path: "/tmp/svc-a"},
		},
	}
	gitClient := &mockGitClient{}
	handler := NewSyncCommandHandler(registry, gitClient)

	cmd := SyncCommand{}

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

func TestSyncCommandHandler_PullError(t *testing.T) {
	registry := &mockSyncRegistryRepo{
		services: map[service.ServiceName]ports.ServiceEntry{
			"svc-a": {Repo: "git@github.com:org/svc-a.git", Path: "/tmp"}, // /tmp exists
		},
	}
	gitClient := &mockGitClient{
		isGitRepo: map[string]bool{"/tmp": true},
		pullErr:   fmt.Errorf("merge conflict"),
	}
	handler := NewSyncCommandHandler(registry, gitClient)

	cmd := SyncCommand{}

	results, err := handler.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Handle() returned unexpected error: %v", err)
	}

	if countResultsByAction(results, "error") != 1 {
		t.Errorf("Expected 1 error result, got %v", results)
	}
}

func TestExpandTilde(t *testing.T) {
	result := expandTilde("/usr/local/bin")
	if result != "/usr/local/bin" {
		t.Errorf("Expected /usr/local/bin, got %s", result)
	}

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
