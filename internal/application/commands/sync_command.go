package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
	"github.com/Saturn-Fintech/grund/internal/domain/service"
	"github.com/Saturn-Fintech/grund/internal/ui"
)

// SyncCommand represents the command to sync service repositories
type SyncCommand struct {
	ServiceNames []string // empty = all services with repo
	NoPull       bool     // --no-pull flag: skip pulling existing repos
}

// SyncCommandHandler handles the sync command
type SyncCommandHandler struct {
	registryRepo ports.ServiceRegistryRepository
	gitClient    ports.GitClient
}

// NewSyncCommandHandler creates a new sync command handler
func NewSyncCommandHandler(
	registryRepo ports.ServiceRegistryRepository,
	gitClient ports.GitClient,
) *SyncCommandHandler {
	return &SyncCommandHandler{
		registryRepo: registryRepo,
		gitClient:    gitClient,
	}
}

// Handle executes the sync command and returns results for display
func (h *SyncCommandHandler) Handle(ctx context.Context, cmd SyncCommand) ([]ports.CloneResult, error) {
	allServices, err := h.registryRepo.GetAllServices()
	if err != nil {
		return nil, fmt.Errorf("failed to get services from registry: %w", err)
	}

	// Build the list of services to process
	targets, err := h.resolveTargets(allServices, cmd.ServiceNames)
	if err != nil {
		return nil, err
	}

	if len(targets) == 0 {
		ui.Infof("No services with repo field found")
		return nil, nil
	}

	ui.Infof("Syncing %d service(s)...", len(targets))

	// Process each service
	var results []ports.CloneResult
	for name, entry := range targets {
		result := h.processService(ctx, name, entry, cmd.NoPull)
		results = append(results, result)
	}

	return results, nil
}

// resolveTargets filters services to the requested set (or all with repo field)
func (h *SyncCommandHandler) resolveTargets(
	allServices map[service.ServiceName]ports.ServiceEntry,
	requestedNames []string,
) (map[service.ServiceName]ports.ServiceEntry, error) {
	if len(requestedNames) == 0 {
		// All services with a repo field
		targets := make(map[service.ServiceName]ports.ServiceEntry)
		for name, entry := range allServices {
			if entry.Repo != "" {
				targets[name] = entry
			}
		}
		return targets, nil
	}

	// Specific services requested
	targets := make(map[service.ServiceName]ports.ServiceEntry)
	for _, name := range requestedNames {
		svcName := service.ServiceName(name)
		entry, ok := allServices[svcName]
		if !ok {
			return nil, fmt.Errorf("service %q not found in registry", name)
		}
		if entry.Repo == "" {
			return nil, fmt.Errorf("service %q has no repo field in registry", name)
		}
		if entry.Path == "" {
			return nil, fmt.Errorf("service %q has no path field in registry", name)
		}
		targets[svcName] = entry
	}

	return targets, nil
}

// processService handles clone/pull/skip logic for a single service
func (h *SyncCommandHandler) processService(
	ctx context.Context,
	name service.ServiceName,
	entry ports.ServiceEntry,
	noPull bool,
) ports.CloneResult {
	path := expandTilde(entry.Path)

	result := ports.CloneResult{
		ServiceName: string(name),
		Path:        path,
	}

	// Check if path already exists
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		// Path exists — check if it's a git repo
		if !h.gitClient.IsGitRepository(path) {
			result.Action = "error"
			result.Error = fmt.Errorf("path %s exists but is not a git repository", path)
			return result
		}

		if noPull {
			result.Action = "skipped"
			ui.Step("%s: already cloned, skipping pull (--no-pull)", name)
			return result
		}

		// Pull latest
		ui.Step("%s: pulling latest...", name)
		if err := h.gitClient.Pull(ctx, path); err != nil {
			result.Action = "error"
			result.Error = fmt.Errorf("failed to pull %s: %w", name, err)
			return result
		}
		result.Action = "pulled"
		return result
	}

	// Path does not exist — clone
	ui.Step("%s: cloning %s...", name, entry.Repo)
	if err := h.gitClient.Clone(ctx, entry.Repo, path); err != nil {
		result.Action = "error"
		result.Error = fmt.Errorf("failed to clone %s: %w", name, err)
		return result
	}
	result.Action = "cloned"
	return result
}

// expandTilde expands ~ to the user's home directory
func expandTilde(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
