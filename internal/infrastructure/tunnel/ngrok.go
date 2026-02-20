package tunnel

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// ngrokLogEntry represents a JSON log entry from ngrok
type ngrokLogEntry struct {
	URL string `json:"url"`
	Msg string `json:"msg"`
}

// NgrokProvider implements Provider for ngrok tunnels
type NgrokProvider struct{}

// NewNgrokProvider creates a new ngrok provider
func NewNgrokProvider() *NgrokProvider {
	return &NgrokProvider{}
}

// Name returns the provider name
func (p *NgrokProvider) Name() string {
	return ProviderNgrok
}

// Start creates a tunnel using ngrok
func (p *NgrokProvider) Start(ctx context.Context, name string, localAddr string) (*Tunnel, error) {
	if _, err := exec.LookPath("ngrok"); err != nil {
		return nil, fmt.Errorf("ngrok not found in PATH: install from https://ngrok.com/download: %w", err)
	}

	parts := strings.Split(localAddr, ":")
	port := parts[len(parts)-1]

	// Write stdout to a temp file instead of a pipe.
	// Pipes cause SIGPIPE when the parent exits (read end closes),
	// killing the detached child. A file has no such coupling.
	logFile, err := os.CreateTemp("", "ngrok-*.log")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp log file: %w", err)
	}
	logPath := logFile.Name()

	cmd := exec.Command("ngrok", "http", port, "--log", "stdout", "--log-format", "json")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = logFile
	cmd.Stderr = nil
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		logFile.Close()
		os.Remove(logPath)
		return nil, fmt.Errorf("failed to start ngrok: %w", err)
	}

	// Close the write handle — the child has its own fd now.
	logFile.Close()

	// Poll the log file for the URL
	urlChan := make(chan string, 1)
	errChan := make(chan error, 1)
	go func() {
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
				var entry ngrokLogEntry
				if err := json.Unmarshal([]byte(line), &entry); err != nil {
					continue
				}
				if entry.URL != "" && strings.HasPrefix(entry.URL, "https://") {
					urlChan <- entry.URL
					return
				}
			} else {
				time.Sleep(100 * time.Millisecond)
				scanner = bufio.NewScanner(f)
			}
		}
	}()

	select {
	case url := <-urlChan:
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
		return nil, fmt.Errorf("timeout waiting for ngrok URL after 30 seconds")
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		os.Remove(logPath)
		return nil, fmt.Errorf("context cancelled while waiting for ngrok URL: %w", ctx.Err())
	}
}

// Stop terminates the tunnel
func (p *NgrokProvider) Stop(tunnel *Tunnel) error {
	if tunnel == nil || tunnel.Process == nil {
		return nil
	}
	if err := tunnel.Process.Kill(); err != nil {
		return fmt.Errorf("failed to stop ngrok tunnel %s: %w", tunnel.Name, err)
	}
	return nil
}
