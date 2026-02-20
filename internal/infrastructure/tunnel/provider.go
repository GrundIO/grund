package tunnel

import (
	"context"
	"os"
)

const (
	ProviderCloudflared = "cloudflared"
	ProviderNgrok       = "ngrok"
)

// Tunnel represents a running tunnel
type Tunnel struct {
	Name      string      // identifier from config
	PublicURL string      // public URL like https://abc.trycloudflare.com
	LocalAddr string      // local address being tunneled like localhost:4566
	Process   *os.Process // the tunnel process
}

// TunnelState represents persisted tunnel state for PID file serialization
type TunnelState struct {
	PID       int    `json:"pid"`
	Name      string `json:"name"`
	PublicURL string `json:"public_url"`
	LocalAddr string `json:"local_addr"`
	Provider  string `json:"provider"`
}

// Provider defines the interface for tunnel providers
type Provider interface {
	// Start creates a tunnel to the given local address
	Start(ctx context.Context, name string, localAddr string) (*Tunnel, error)
	// Stop terminates the tunnel
	Stop(tunnel *Tunnel) error
	// Name returns the provider name
	Name() string
}
