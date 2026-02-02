package tunnel

import (
	"context"
	"fmt"
	"sync"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
	"github.com/Saturn-Fintech/grund/internal/config"
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

// StartAll starts tunnels for all targets in the config
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
		tunnel, err := provider.Start(ctx, target.Name, localAddr)
		if err != nil {
			// Cleanup any started tunnels
			for _, t := range startedTunnels {
				_ = provider.Stop(t)
			}
			return nil, fmt.Errorf("failed to start tunnel %s: %w", target.Name, err)
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

// StopAll stops all running tunnels
func (m *Manager) StopAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for name, tunnel := range m.tunnels {
		if tunnel.Process != nil {
			if err := tunnel.Process.Kill(); err != nil {
				lastErr = fmt.Errorf("failed to stop tunnel %s: %w", name, err)
			}
		}
		delete(m.tunnels, name)
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

// Ensure Manager implements ports.TunnelManager
var _ ports.TunnelManager = (*Manager)(nil)
