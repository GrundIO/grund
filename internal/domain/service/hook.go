package service

import "time"

// HookStage represents when a hook executes in the service lifecycle
type HookStage string

const (
	HookStagePreUp              HookStage = "pre_up"
	HookStagePostInfrastructure HookStage = "post_infrastructure"
	HookStagePostUp             HookStage = "post_up"
	HookStagePreDown            HookStage = "pre_down"
	HookStagePostDown           HookStage = "post_down"
)

// AllowsTarget returns true if the stage allows the given execution target
func (s HookStage) AllowsTarget(target HookTarget) bool {
	switch s {
	case HookStagePreUp, HookStagePostInfrastructure, HookStagePostDown:
		return target == HookTargetHost
	case HookStagePostUp, HookStagePreDown:
		return target == HookTargetHost || target == HookTargetContainer
	default:
		return false
	}
}

// HookTarget represents where a hook executes
type HookTarget string

const (
	HookTargetHost      HookTarget = "host"
	HookTargetContainer HookTarget = "container"
)

// IsValid returns true if the target is a valid hook target
func (t HookTarget) IsValid() bool {
	return t == HookTargetHost || t == HookTargetContainer
}

// Hook represents a lifecycle hook configuration
type Hook struct {
	Name            string
	Command         string
	Target          HookTarget
	Timeout         time.Duration
	ContinueOnError bool
}

// GetTimeout returns the timeout, using default if not set
func (h Hook) GetTimeout() time.Duration {
	if h.Timeout == 0 {
		return 10 * time.Minute
	}
	return h.Timeout
}

// Hooks contains all lifecycle hooks for a service
type Hooks struct {
	PreUp              []Hook
	PostInfrastructure []Hook
	PostUp             []Hook
	PreDown            []Hook
	PostDown           []Hook
}

// GetHooksForStage returns the hooks for a given stage
func (h Hooks) GetHooksForStage(stage HookStage) []Hook {
	switch stage {
	case HookStagePreUp:
		return h.PreUp
	case HookStagePostInfrastructure:
		return h.PostInfrastructure
	case HookStagePostUp:
		return h.PostUp
	case HookStagePreDown:
		return h.PreDown
	case HookStagePostDown:
		return h.PostDown
	default:
		return nil
	}
}

// HasHooks returns true if any hooks are defined
func (h Hooks) HasHooks() bool {
	return len(h.PreUp) > 0 ||
		len(h.PostInfrastructure) > 0 ||
		len(h.PostUp) > 0 ||
		len(h.PreDown) > 0 ||
		len(h.PostDown) > 0
}
