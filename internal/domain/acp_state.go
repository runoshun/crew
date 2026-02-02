package domain

import "context"

// ACPExecutionSubstate represents ACP-specific execution state.
type ACPExecutionSubstate string

const (
	ACPExecutionIdle               ACPExecutionSubstate = "idle"
	ACPExecutionRunning            ACPExecutionSubstate = "running"
	ACPExecutionAwaitingPermission ACPExecutionSubstate = "awaiting_permission"
	ACPExecutionAwaitingUser       ACPExecutionSubstate = "awaiting_user"
)

func (s ACPExecutionSubstate) IsValid() bool {
	switch s {
	case ACPExecutionIdle, ACPExecutionRunning, ACPExecutionAwaitingPermission, ACPExecutionAwaitingUser:
		return true
	default:
		return false
	}
}

func (s ACPExecutionSubstate) Display() string {
	return string(s)
}

// ACPExecutionState holds persisted ACP execution state.
type ACPExecutionState struct {
	ExecutionSubstate ACPExecutionSubstate `json:"execution_substate"`
	SessionID         string               `json:"session_id,omitempty"`
}

// ACPStateStore persists ACP execution state.
type ACPStateStore interface {
	Load(ctx context.Context, namespace string, taskID int) (ACPExecutionState, error)
	Save(ctx context.Context, namespace string, taskID int, state ACPExecutionState) error
}
