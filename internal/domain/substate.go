package domain

// ExecutionSubstate represents CLI execution substate.
type ExecutionSubstate string

const (
	SubstateIdle               ExecutionSubstate = "idle"
	SubstateRunning            ExecutionSubstate = "running"
	SubstateAwaitingPermission ExecutionSubstate = "awaiting_permission"
	SubstateAwaitingUser       ExecutionSubstate = "awaiting_user"
)

func (s ExecutionSubstate) IsValid() bool {
	switch s {
	case SubstateIdle, SubstateRunning, SubstateAwaitingPermission, SubstateAwaitingUser:
		return true
	default:
		return false
	}
}

func (s ExecutionSubstate) Display() string {
	return string(s)
}
