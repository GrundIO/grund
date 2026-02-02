package tunnel

import (
	"testing"

	"github.com/Saturn-Fintech/grund/internal/config"
)

func TestManagerCreation(t *testing.T) {
	manager := NewManager()
	if manager == nil {
		t.Fatal("expected manager to be created")
	}
}

func TestManagerGetProvider(t *testing.T) {
	manager := NewManager()

	provider, err := manager.GetProvider(ProviderCloudflared)
	if err != nil {
		t.Fatalf("failed to get cloudflared provider: %v", err)
	}
	if provider.Name() != ProviderCloudflared {
		t.Errorf("expected cloudflared, got %s", provider.Name())
	}

	provider, err = manager.GetProvider(ProviderNgrok)
	if err != nil {
		t.Fatalf("failed to get ngrok provider: %v", err)
	}
	if provider.Name() != ProviderNgrok {
		t.Errorf("expected ngrok, got %s", provider.Name())
	}

	_, err = manager.GetProvider("invalid")
	if err == nil {
		t.Error("expected error for invalid provider")
	}
}

func TestManagerValidateConfig(t *testing.T) {
	manager := NewManager()

	// Valid config
	validConfig := &config.TunnelConfig{
		Provider: ProviderCloudflared,
		Targets: []config.TunnelTarget{
			{Name: "test", Host: "localhost", Port: "8080"},
		},
	}
	if err := manager.ValidateConfig(validConfig); err != nil {
		t.Errorf("expected valid config, got error: %v", err)
	}

	// Nil config is valid (no tunnels)
	if err := manager.ValidateConfig(nil); err != nil {
		t.Errorf("expected nil config to be valid, got error: %v", err)
	}

	// Invalid provider
	invalidProvider := &config.TunnelConfig{
		Provider: "invalid",
		Targets:  []config.TunnelTarget{{Name: "test", Host: "localhost", Port: "8080"}},
	}
	if err := manager.ValidateConfig(invalidProvider); err == nil {
		t.Error("expected error for invalid provider")
	}

	// Duplicate names
	duplicateNames := &config.TunnelConfig{
		Provider: ProviderCloudflared,
		Targets: []config.TunnelTarget{
			{Name: "test", Host: "localhost", Port: "8080"},
			{Name: "test", Host: "localhost", Port: "9090"},
		},
	}
	if err := manager.ValidateConfig(duplicateNames); err == nil {
		t.Error("expected error for duplicate target names")
	}

	// Missing host
	missingHost := &config.TunnelConfig{
		Provider: ProviderCloudflared,
		Targets:  []config.TunnelTarget{{Name: "test", Port: "8080"}},
	}
	if err := manager.ValidateConfig(missingHost); err == nil {
		t.Error("expected error for missing host")
	}

	// Missing name
	missingName := &config.TunnelConfig{
		Provider: ProviderCloudflared,
		Targets:  []config.TunnelTarget{{Host: "localhost", Port: "8080"}},
	}
	if err := manager.ValidateConfig(missingName); err == nil {
		t.Error("expected error for missing name")
	}

	// Missing port
	missingPort := &config.TunnelConfig{
		Provider: ProviderCloudflared,
		Targets:  []config.TunnelTarget{{Name: "test", Host: "localhost"}},
	}
	if err := manager.ValidateConfig(missingPort); err == nil {
		t.Error("expected error for missing port")
	}
}
