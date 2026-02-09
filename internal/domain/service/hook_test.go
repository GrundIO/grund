package service

import (
	"testing"
	"time"
)

func TestHook_Defaults(t *testing.T) {
	hook := Hook{
		Name:    "test hook",
		Command: "echo hello",
		Target:  HookTargetHost,
	}

	if hook.GetTimeout() != 10*time.Minute {
		t.Errorf("expected default timeout 10m, got %v", hook.GetTimeout())
	}
	if hook.ContinueOnError != false {
		t.Errorf("expected default continue_on_error false")
	}
}

func TestHook_CustomTimeout(t *testing.T) {
	hook := Hook{
		Name:    "test hook",
		Command: "echo hello",
		Target:  HookTargetHost,
		Timeout: 30 * time.Second,
	}

	if hook.GetTimeout() != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", hook.GetTimeout())
	}
}

func TestHookTarget_Validation(t *testing.T) {
	if !HookTargetHost.IsValid() {
		t.Error("host should be valid")
	}
	if !HookTargetContainer.IsValid() {
		t.Error("container should be valid")
	}
	if HookTarget("invalid").IsValid() {
		t.Error("invalid should not be valid")
	}
}

func TestHookStage_AllowsTarget(t *testing.T) {
	tests := []struct {
		stage         HookStage
		target        HookTarget
		shouldBeValid bool
	}{
		{HookStagePreUp, HookTargetHost, true},
		{HookStagePreUp, HookTargetContainer, false},
		{HookStagePostInfrastructure, HookTargetHost, true},
		{HookStagePostInfrastructure, HookTargetContainer, false},
		{HookStagePostUp, HookTargetHost, true},
		{HookStagePostUp, HookTargetContainer, true},
		{HookStagePreDown, HookTargetHost, true},
		{HookStagePreDown, HookTargetContainer, true},
		{HookStagePostDown, HookTargetHost, true},
		{HookStagePostDown, HookTargetContainer, false},
	}

	for _, tt := range tests {
		result := tt.stage.AllowsTarget(tt.target)
		if result != tt.shouldBeValid {
			t.Errorf("stage %s with target %s: expected %v, got %v",
				tt.stage, tt.target, tt.shouldBeValid, result)
		}
	}
}

func TestHookStage_AllowsTarget_InvalidStage(t *testing.T) {
	invalidStage := HookStage("invalid_stage")
	if invalidStage.AllowsTarget(HookTargetHost) {
		t.Error("invalid stage should not allow any target")
	}
	if invalidStage.AllowsTarget(HookTargetContainer) {
		t.Error("invalid stage should not allow any target")
	}
}

func TestHooks_HasHooks(t *testing.T) {
	tests := []struct {
		name     string
		hooks    Hooks
		expected bool
	}{
		{
			name:     "empty hooks",
			hooks:    Hooks{},
			expected: false,
		},
		{
			name: "has pre_up hook",
			hooks: Hooks{
				PreUp: []Hook{{Name: "test", Command: "echo test", Target: HookTargetHost}},
			},
			expected: true,
		},
		{
			name: "has post_infrastructure hook",
			hooks: Hooks{
				PostInfrastructure: []Hook{{Name: "test", Command: "echo test", Target: HookTargetHost}},
			},
			expected: true,
		},
		{
			name: "has post_up hook",
			hooks: Hooks{
				PostUp: []Hook{{Name: "test", Command: "echo test", Target: HookTargetHost}},
			},
			expected: true,
		},
		{
			name: "has pre_down hook",
			hooks: Hooks{
				PreDown: []Hook{{Name: "test", Command: "echo test", Target: HookTargetHost}},
			},
			expected: true,
		},
		{
			name: "has post_down hook",
			hooks: Hooks{
				PostDown: []Hook{{Name: "test", Command: "echo test", Target: HookTargetHost}},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.hooks.HasHooks() != tt.expected {
				t.Errorf("HasHooks() = %v, want %v", tt.hooks.HasHooks(), tt.expected)
			}
		})
	}
}

func TestHooks_GetHooksForStage(t *testing.T) {
	preUpHook := Hook{Name: "pre_up", Command: "echo pre_up", Target: HookTargetHost}
	postInfraHook := Hook{Name: "post_infra", Command: "echo post_infra", Target: HookTargetHost}
	postUpHook := Hook{Name: "post_up", Command: "echo post_up", Target: HookTargetHost}
	preDownHook := Hook{Name: "pre_down", Command: "echo pre_down", Target: HookTargetHost}
	postDownHook := Hook{Name: "post_down", Command: "echo post_down", Target: HookTargetHost}

	hooks := Hooks{
		PreUp:              []Hook{preUpHook},
		PostInfrastructure: []Hook{postInfraHook},
		PostUp:             []Hook{postUpHook},
		PreDown:            []Hook{preDownHook},
		PostDown:           []Hook{postDownHook},
	}

	tests := []struct {
		stage        HookStage
		expectedName string
	}{
		{HookStagePreUp, "pre_up"},
		{HookStagePostInfrastructure, "post_infra"},
		{HookStagePostUp, "post_up"},
		{HookStagePreDown, "pre_down"},
		{HookStagePostDown, "post_down"},
	}

	for _, tt := range tests {
		t.Run(string(tt.stage), func(t *testing.T) {
			result := hooks.GetHooksForStage(tt.stage)
			if len(result) != 1 {
				t.Fatalf("expected 1 hook, got %d", len(result))
			}
			if result[0].Name != tt.expectedName {
				t.Errorf("expected hook name %q, got %q", tt.expectedName, result[0].Name)
			}
		})
	}
}

func TestHooks_GetHooksForStage_InvalidStage(t *testing.T) {
	hooks := Hooks{
		PreUp: []Hook{{Name: "test", Command: "echo test", Target: HookTargetHost}},
	}

	result := hooks.GetHooksForStage(HookStage("invalid"))
	if result != nil {
		t.Errorf("expected nil for invalid stage, got %v", result)
	}
}
