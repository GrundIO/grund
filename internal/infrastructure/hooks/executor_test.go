package hooks

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
	"github.com/Saturn-Fintech/grund/internal/domain/service"
)

func TestHookExecutor_ExecuteHostCommand(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test script
	scriptPath := filepath.Join(tmpDir, "test.sh")
	err := os.WriteFile(scriptPath, []byte("#!/bin/sh\necho \"hello from hook\""), 0755)
	if err != nil {
		t.Fatal(err)
	}

	executor := NewHookExecutor()
	hook := service.Hook{
		Name:    "test hook",
		Command: "./test.sh",
		Target:  service.HookTargetHost,
		Timeout: 10 * time.Second,
	}

	execCtx := ports.HookExecutionContext{
		ServiceName: "test-service",
		ServicePath: tmpDir,
		Environment: map[string]string{"TEST_VAR": "test_value"},
	}

	err = executor.Execute(context.Background(), hook, execCtx)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestHookExecutor_ExecuteHostCommand_Timeout(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a script that sleeps
	scriptPath := filepath.Join(tmpDir, "slow.sh")
	err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nsleep 10"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	executor := NewHookExecutor()
	hook := service.Hook{
		Name:    "slow hook",
		Command: "./slow.sh",
		Target:  service.HookTargetHost,
		Timeout: 100 * time.Millisecond,
	}

	execCtx := ports.HookExecutionContext{
		ServiceName: "test-service",
		ServicePath: tmpDir,
	}

	err = executor.Execute(context.Background(), hook, execCtx)
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestHookExecutor_ExecuteAll_ContinueOnError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create scripts
	err := os.WriteFile(filepath.Join(tmpDir, "fail.sh"), []byte("#!/bin/sh\nexit 1"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tmpDir, "success.sh"), []byte("#!/bin/sh\necho ok"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	executor := NewHookExecutor()

	// Test with continue_on_error = false (default)
	hooks := []service.Hook{
		{Name: "fail", Command: "./fail.sh", Target: service.HookTargetHost},
		{Name: "success", Command: "./success.sh", Target: service.HookTargetHost},
	}

	execCtx := ports.HookExecutionContext{
		ServiceName: "test-service",
		ServicePath: tmpDir,
	}

	err = executor.ExecuteAll(context.Background(), hooks, execCtx)
	if err == nil {
		t.Error("expected error when continue_on_error is false")
	}

	// Test with continue_on_error = true
	hooks[0].ContinueOnError = true
	err = executor.ExecuteAll(context.Background(), hooks, execCtx)
	if err != nil {
		t.Errorf("expected no error when continue_on_error is true, got: %v", err)
	}
}

func TestHookExecutor_ExecuteAll_EmptyHooks(t *testing.T) {
	executor := NewHookExecutor()
	execCtx := ports.HookExecutionContext{
		ServiceName: "test-service",
		ServicePath: "/tmp",
	}

	err := executor.ExecuteAll(context.Background(), []service.Hook{}, execCtx)
	if err != nil {
		t.Errorf("expected no error for empty hooks, got: %v", err)
	}
}

func TestHookExecutor_ExecuteContainer_RequiresContainerName(t *testing.T) {
	executor := NewHookExecutor()
	hook := service.Hook{
		Name:    "container hook",
		Command: "echo hello",
		Target:  service.HookTargetContainer,
		Timeout: 10 * time.Second,
	}

	execCtx := ports.HookExecutionContext{
		ServiceName:   "test-service",
		ServicePath:   "/tmp",
		ContainerName: "", // Empty container name
	}

	err := executor.Execute(context.Background(), hook, execCtx)
	if err == nil {
		t.Error("expected error when container name is empty")
	}
}

func TestHookExecutor_InvalidTarget(t *testing.T) {
	executor := NewHookExecutor()
	hook := service.Hook{
		Name:    "invalid hook",
		Command: "echo hello",
		Target:  service.HookTarget("invalid"),
		Timeout: 10 * time.Second,
	}

	execCtx := ports.HookExecutionContext{
		ServiceName: "test-service",
		ServicePath: "/tmp",
	}

	err := executor.Execute(context.Background(), hook, execCtx)
	if err == nil {
		t.Error("expected error for invalid target")
	}
}

func TestHookExecutor_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a script that sleeps
	scriptPath := filepath.Join(tmpDir, "slow.sh")
	err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nsleep 10"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	executor := NewHookExecutor()
	hook := service.Hook{
		Name:    "slow hook",
		Command: "./slow.sh",
		Target:  service.HookTargetHost,
		Timeout: 30 * time.Second, // Long timeout
	}

	execCtx := ports.HookExecutionContext{
		ServiceName: "test-service",
		ServicePath: tmpDir,
	}

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err = executor.Execute(ctx, hook, execCtx)
	if err == nil {
		t.Error("expected error when context is cancelled")
	}
}
