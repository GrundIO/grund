package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
)

// Client implements ports.GitClient using os/exec
type Client struct{}

// NewClient creates a new git client
func NewClient() *Client {
	return &Client{}
}

// Clone clones a git repository to the destination path
func (c *Client) Clone(ctx context.Context, repoURL string, destPath string) error {
	// Ensure parent directory exists
	parentDir := filepath.Dir(destPath)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return fmt.Errorf("failed to create parent directory %s: %w", parentDir, err)
	}

	cmd := exec.CommandContext(ctx, "git", "clone", repoURL, destPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed for %s: %w", repoURL, err)
	}

	return nil
}

// Pull pulls the latest changes in the given git repository
func (c *Client) Pull(ctx context.Context, repoPath string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "pull", "--ff-only")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git pull failed in %s: %w", repoPath, err)
	}

	return nil
}

// IsGitRepository checks if the given path contains a .git directory
func (c *Client) IsGitRepository(path string) bool {
	info, err := os.Stat(filepath.Join(path, ".git"))
	if err != nil {
		return false
	}
	return info.IsDir()
}

// Ensure Client implements the interface
var _ ports.GitClient = (*Client)(nil)
