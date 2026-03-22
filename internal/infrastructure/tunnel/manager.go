package tunnel

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/GrundIO/grund/internal/application/ports"
	"github.com/GrundIO/grund/internal/config"
)

// Manager handles tunnel lifecycle
type Manager struct {
	tunnels map[string]*Tunnel
	mu      sync.Mutex
}

// NewManager creates a new tunnel manager
func NewManager() *Manager {
	return &Manager{
		tunnels: make(map[string]*Tunnel),
	}
}

// GetProvider returns the appropriate provider for the given name
func (m *Manager) GetProvider(name string) (Provider, error) {
	switch name {
	case ProviderCloudflared:
		return NewCloudflaredProvider(), nil
	case ProviderNgrok:
		return NewNgrokProvider(), nil
	default:
		return nil, fmt.Errorf("unknown tunnel provider: %s (supported: cloudflared, ngrok)", name)
	}
}

// ValidateConfig validates tunnel configuration
func (m *Manager) ValidateConfig(cfg *config.TunnelConfig) error {
	if cfg == nil {
		return nil
	}

	// Validate provider
	if cfg.Provider != ProviderCloudflared && cfg.Provider != ProviderNgrok {
		return fmt.Errorf("invalid tunnel provider: %s (must be 'cloudflared' or 'ngrok')", cfg.Provider)
	}

	// Validate targets
	names := make(map[string]bool)
	for _, target := range cfg.Targets {
		if target.Name == "" {
			return fmt.Errorf("tunnel target missing name")
		}
		if target.Host == "" {
			return fmt.Errorf("tunnel target %s missing host", target.Name)
		}
		if target.Port == "" {
			return fmt.Errorf("tunnel target %s missing port", target.Name)
		}
		if names[target.Name] {
			return fmt.Errorf("duplicate tunnel target name: %s", target.Name)
		}
		names[target.Name] = true
	}

	return nil
}

// StartAll starts tunnels for all targets in the config.
// Before starting a new tunnel, it checks for an existing PID file.
// If the process is still alive, it reuses the existing tunnel.
func (m *Manager) StartAll(ctx context.Context, cfg *config.TunnelConfig, resolvedTargets []ports.ResolvedTunnelTarget) ([]ports.TunnelInfo, error) {
	if cfg == nil || len(cfg.Targets) == 0 {
		return nil, nil
	}

	provider, err := m.GetProvider(cfg.Provider)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var tunnelInfos []ports.TunnelInfo
	var startedTunnels []*Tunnel
	for _, target := range resolvedTargets {
		localAddr := fmt.Sprintf("%s:%s", target.Host, target.Port)

		// Check for existing PID file — reuse if process is still alive
		if state, err := readPIDFile(target.Name); err == nil && isProcessAlive(state.PID) {
			m.tunnels[target.Name] = &Tunnel{
				Name:      state.Name,
				PublicURL: state.PublicURL,
				LocalAddr: state.LocalAddr,
				Process:   nil, // we don't own the process handle
			}
			tunnelInfos = append(tunnelInfos, ports.TunnelInfo{
				Name:      state.Name,
				PublicURL: state.PublicURL,
				LocalAddr: state.LocalAddr,
			})
			continue
		}

		tunnel, err := provider.Start(ctx, target.Name, localAddr)
		if err != nil {
			// Cleanup any started tunnels
			for _, t := range startedTunnels {
				_ = provider.Stop(t)
				_ = removePIDFile(t.Name)
			}
			return nil, fmt.Errorf("failed to start tunnel %s: %w", target.Name, err)
		}

		// Persist PID file so the tunnel can be tracked across CLI invocations
		if tunnel.Process != nil {
			state := TunnelState{
				PID:       tunnel.Process.Pid,
				Name:      tunnel.Name,
				PublicURL: tunnel.PublicURL,
				LocalAddr: tunnel.LocalAddr,
				Provider:  cfg.Provider,
			}
			if writeErr := writePIDFile(tunnel.Name, state); writeErr != nil {
				// Non-fatal: tunnel works, just can't be tracked later
				fmt.Fprintf(os.Stderr, "warning: failed to write PID file for tunnel %s: %v\n", tunnel.Name, writeErr)
			}
		}

		startedTunnels = append(startedTunnels, tunnel)
		m.tunnels[target.Name] = tunnel
		tunnelInfos = append(tunnelInfos, ports.TunnelInfo{
			Name:      tunnel.Name,
			PublicURL: tunnel.PublicURL,
			LocalAddr: tunnel.LocalAddr,
		})
	}

	return tunnelInfos, nil
}

// StopAll stops all running tunnels by reading PID files from disk.
// This works across CLI invocations — it does not require the same
// Manager instance that started the tunnels.
func (m *Manager) StopAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error

	// Kill in-memory tunnels
	for name, tun := range m.tunnels {
		if tun.Process != nil {
			if err := tun.Process.Kill(); err != nil {
				lastErr = fmt.Errorf("failed to stop tunnel %s: %w", name, err)
			}
		}
		_ = removePIDFile(name)
		delete(m.tunnels, name)
	}

	// Also kill tunnels tracked only on disk (started by a previous CLI run)
	dir, err := getTunnelsDir()
	if err != nil {
		return lastErr
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return lastErr
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		name := entry.Name()[:len(entry.Name())-len(".json")]
		state, err := readPIDFile(name)
		if err != nil {
			_ = removePIDFile(name)
			continue
		}
		if isProcessAlive(state.PID) {
			if err := syscall.Kill(state.PID, syscall.SIGTERM); err != nil {
				lastErr = fmt.Errorf("failed to stop tunnel %s (pid %d): %w", name, state.PID, err)
			}
		}
		_ = removePIDFile(name)
	}

	return lastErr
}

// GetTunnels returns all running tunnels
func (m *Manager) GetTunnels() map[string]ports.TunnelInfo {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make(map[string]ports.TunnelInfo)
	for k, v := range m.tunnels {
		result[k] = ports.TunnelInfo{
			Name:      v.Name,
			PublicURL: v.PublicURL,
			LocalAddr: v.LocalAddr,
		}
	}
	return result
}

// getTunnelsDir returns the directory for tunnel PID files (~/.grund/tmp/tunnels)
func getTunnelsDir() (string, error) {
	grundHome, err := config.GetGrundHome()
	if err != nil {
		return "", fmt.Errorf("failed to get grund home: %w", err)
	}
	dir := filepath.Join(grundHome, "tmp", "tunnels")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create tunnels directory: %w", err)
	}
	return dir, nil
}

// writePIDFile persists tunnel state to a JSON file
func writePIDFile(name string, state TunnelState) error {
	dir, err := getTunnelsDir()
	if err != nil {
		return err
	}
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal tunnel state: %w", err)
	}
	path := filepath.Join(dir, name+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}
	return nil
}

// readPIDFile reads tunnel state from a JSON file
func readPIDFile(name string) (*TunnelState, error) {
	dir, err := getTunnelsDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read PID file: %w", err)
	}
	var state TunnelState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tunnel state: %w", err)
	}
	return &state, nil
}

// removePIDFile deletes the PID file for a tunnel
func removePIDFile(name string) error {
	dir, err := getTunnelsDir()
	if err != nil {
		return err
	}
	return os.Remove(filepath.Join(dir, name+".json"))
}

// isProcessAlive checks if a process with the given PID is still running
func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	// Signal 0 checks if the process exists without sending a signal
	return syscall.Kill(pid, 0) == nil
}

// Ensure Manager implements ports.TunnelManager
var _ ports.TunnelManager = (*Manager)(nil)
