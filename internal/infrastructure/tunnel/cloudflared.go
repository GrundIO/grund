package tunnel

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"syscall"
	"time"
)

// Compile-time interface check
var _ Provider = (*CloudflaredProvider)(nil)

var cloudflaredURLPattern = regexp.MustCompile(`(https://[a-z0-9-]+\.trycloudflare\.com)`)

// CloudflaredProvider implements Provider for cloudflared tunnels
type CloudflaredProvider struct{}

// NewCloudflaredProvider creates a new cloudflared provider
func NewCloudflaredProvider() *CloudflaredProvider {
	return &CloudflaredProvider{}
}

// Name returns the provider name
func (p *CloudflaredProvider) Name() string {
	return ProviderCloudflared
}

// Start creates a tunnel using cloudflared
func (p *CloudflaredProvider) Start(ctx context.Context, name string, localAddr string) (*Tunnel, error) {
	if _, err := exec.LookPath("cloudflared"); err != nil {
		return nil, fmt.Errorf("cloudflared not found in PATH: install with 'brew install cloudflared': %w", err)
	}

	// Write stderr to a temp file instead of a pipe.
	// Pipes cause SIGPIPE when the parent exits (read end closes),
	// killing the detached child. A file has no such coupling.
	logFile, err := os.CreateTemp("", "cloudflared-*.log")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp log file: %w", err)
	}
	logPath := logFile.Name()

	cmd := exec.Command("cloudflared", "tunnel", "--url", fmt.Sprintf("http://%s", localAddr))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stderr = logFile
	cmd.Stdout = nil
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		logFile.Close()
		os.Remove(logPath)
		return nil, fmt.Errorf("failed to start cloudflared: %w", err)
	}

	// Close the write handle — the child has its own fd now.
	// This fully severs the parent from the child's I/O.
	logFile.Close()

	// Poll the log file for the URL
	urlChan := make(chan string, 1)
	errChan := make(chan error, 1)
	go func() {
		// Open our own read handle
		f, err := os.Open(logPath)
		if err != nil {
			errChan <- fmt.Errorf("failed to open log file for reading: %w", err)
			return
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for {
			if scanner.Scan() {
				line := scanner.Text()
				if matches := cloudflaredURLPattern.FindStringSubmatch(line); len(matches) > 1 {
					urlChan <- matches[1]
					return
				}
			} else {
				// No more data yet — wait and retry (file is still being written)
				time.Sleep(100 * time.Millisecond)
				// Reset scanner to pick up new content
				scanner = bufio.NewScanner(f)
			}
		}
	}()

	select {
	case url := <-urlChan:
		// Clean up the temp log file — child keeps running independently
		os.Remove(logPath)
		return &Tunnel{
			Name:      name,
			PublicURL: url,
			LocalAddr: localAddr,
			Process:   cmd.Process,
		}, nil
	case err := <-errChan:
		_ = cmd.Process.Kill()
		os.Remove(logPath)
		return nil, err
	case <-time.After(30 * time.Second):
		_ = cmd.Process.Kill()
		os.Remove(logPath)
		return nil, fmt.Errorf("timeout waiting for cloudflared URL after 30 seconds")
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		os.Remove(logPath)
		return nil, fmt.Errorf("context cancelled while waiting for cloudflared URL: %w", ctx.Err())
	}
}

// Stop terminates the tunnel
func (p *CloudflaredProvider) Stop(tunnel *Tunnel) error {
	if tunnel == nil || tunnel.Process == nil {
		return nil
	}
	if err := tunnel.Process.Kill(); err != nil {
		return fmt.Errorf("failed to stop cloudflared tunnel %s: %w", tunnel.Name, err)
	}
	return nil
}
