// Package tui provides the terminal user interface for git-crew.
package tui

// Mode represents the current UI mode.
type Mode int

const (
	ModeNormal     Mode = iota // Default navigation mode
	ModeFilter                 // Text filtering mode
	ModeConfirm                // Confirmation dialog mode
	ModeInputTitle             // Title input mode (for new task)
	ModeInputDesc              // Description input mode (for new task)
	ModeStart                  // Agent picker mode
	ModeHelp                   // Help overlay mode
	ModeDetail                 // Task detail view mode
)

// String returns the string representation of the mode.
func (m Mode) String() string {
	switch m {
	case ModeNormal:
		return "normal"
	case ModeFilter:
		return "filter"
	case ModeConfirm:
		return "confirm"
	case ModeInputTitle:
		return "input_title"
	case ModeInputDesc:
		return "input_desc"
	case ModeStart:
		return "start"
	case ModeHelp:
		return "help"
	case ModeDetail:
		return "detail"
	default:
		return "unknown"
	}
}

// ConfirmAction represents the type of action requiring confirmation.
type ConfirmAction int

const (
	ConfirmNone   ConfirmAction = iota
	ConfirmDelete               // Delete task
	ConfirmClose                // Close task
	ConfirmStop                 // Stop running session
	ConfirmMerge                // Merge task
)

// IsInputMode returns true if the mode accepts text input.
func (m Mode) IsInputMode() bool {
	switch m {
	case ModeFilter, ModeInputTitle, ModeInputDesc:
		return true
	case ModeNormal, ModeConfirm, ModeStart, ModeHelp, ModeDetail:
		return false
	}
	return false
}

// String returns a human-readable description of the action.
func (a ConfirmAction) String() string {
	switch a {
	case ConfirmNone:
		return ""
	case ConfirmDelete:
		return "delete"
	case ConfirmClose:
		return "close"
	case ConfirmStop:
		return "stop"
	case ConfirmMerge:
		return "merge"
	}
	return ""
}
