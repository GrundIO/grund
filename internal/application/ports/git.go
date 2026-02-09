package ports

import "context"

// CloneResult represents the outcome of a clone/pull operation for a service
type CloneResult struct {
	ServiceName string
	Path        string
	Action      string // "cloned", "pulled", "skipped"
	Error       error
}

// GitClient defines the interface for git operations
type GitClient interface {
	Clone(ctx context.Context, repoURL string, destPath string) error
	Pull(ctx context.Context, repoPath string) error
	IsGitRepository(path string) bool
}
