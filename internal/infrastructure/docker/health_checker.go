package docker

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
)

// HTTPHealthChecker implements HealthChecker using HTTP
type HTTPHealthChecker struct {
	client *http.Client
}

// NewHTTPHealthChecker creates a new HTTP health checker
func NewHTTPHealthChecker() ports.HealthChecker {
	return &HTTPHealthChecker{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// CheckHealth checks if an endpoint is healthy
func (h *HTTPHealthChecker) CheckHealth(ctx context.Context, endpoint string, timeout int) error {
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}

// WaitForHealthy waits for a service to become healthy
func (h *HTTPHealthChecker) WaitForHealthy(ctx context.Context, endpoint string, interval, timeout, retries int) error {
	intervalDuration := time.Duration(interval) * time.Second
	timeoutDuration := time.Duration(timeout) * time.Second

	deadline := time.Now().Add(timeoutDuration * time.Duration(retries))

	for i := 0; i < retries; i++ {
		if time.Now().After(deadline) {
			return fmt.Errorf("health check timeout after %d retries", retries)
		}

		if err := h.CheckHealth(ctx, endpoint, timeout); err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(intervalDuration):
		}
	}

	return fmt.Errorf("health check failed after %d retries", retries)
}
