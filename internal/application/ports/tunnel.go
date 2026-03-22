package ports

import (
	"context"

	"github.com/GrundIO/grund/internal/config"
)

// TunnelInfo represents a running tunnel
type TunnelInfo struct {
	Name      string
	PublicURL string
	LocalAddr string
}

// ResolvedTunnelTarget represents a target with placeholders resolved
type ResolvedTunnelTarget struct {
	Name string
	Host string
	Port string
}

// TunnelManager manages tunnel lifecycle
type TunnelManager interface {
	// ValidateConfig validates tunnel configuration
	ValidateConfig(cfg *config.TunnelConfig) error

	// StartAll starts tunnels for all targets in the config
	StartAll(ctx context.Context, cfg *config.TunnelConfig, resolvedTargets []ResolvedTunnelTarget) ([]TunnelInfo, error)

	// StopAll stops all running tunnels
	StopAll() error

	// GetTunnels returns all running tunnels
	GetTunnels() map[string]TunnelInfo
}
