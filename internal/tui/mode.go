// Package tui provides the terminal user interface for git-crew.
package tui

// Mode represents the current UI mode.
type Mode int

const (
	ModeNormal       Mode = iota // Default navigation mode
	ModeFilter                   // Text filtering mode
	ModeConfirm                  // Confirmation dialog mode
	ModeInputTitle               // Title input mode (for new task) - deprecated, use ModeNewTask
	ModeInputDesc                // Description input mode (for new task) - deprecated, use ModeNewTask
	ModeNewTask                  // New task form mode (title, desc, parent)
	ModeStart                    // Agent picker mode
	ModeHelp                     // Help overlay mode
	ModeDetail                   // Task detail view mode
	ModeChangeStatus             // Status change mode
	ModeExec                     // Execute command mode
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
	case ModeNewTask:
		return "new_task"
	case ModeStart:
		return "start"
	case ModeHelp:
		return "help"
	case ModeDetail:
		return "detail"
	case ModeChangeStatus:
		return "change_status"
	case ModeExec:
		return "exec"
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
	case ModeFilter, ModeInputTitle, ModeInputDesc, ModeNewTask, ModeExec:
		return true
	case ModeNormal, ModeConfirm, ModeStart, ModeHelp, ModeDetail, ModeChangeStatus:
		return false
	}
	return false
}

// NewTaskField represents the currently focused field in the new task form.
type NewTaskField int

const (
	FieldTitle NewTaskField = iota
	FieldDesc
	FieldParent
)

func (f NewTaskField) Next() NewTaskField {
	switch f {
	case FieldTitle:
		return FieldDesc
	case FieldDesc:
		return FieldParent
	case FieldParent:
		return FieldTitle
	default:
		return FieldTitle
	}
}

func (f NewTaskField) Prev() NewTaskField {
	switch f {
	case FieldTitle:
		return FieldParent
	case FieldDesc:
		return FieldTitle
	case FieldParent:
		return FieldDesc
	default:
		return FieldTitle
	}
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

type SortMode int

const (
	SortByStatus SortMode = iota
	SortByID
)

func (s SortMode) String() string {
	switch s {
	case SortByStatus:
		return "status"
	case SortByID:
		return "id"
	default:
		return "unknown"
	}
}

func (s SortMode) Next() SortMode {
	switch s {
	case SortByStatus:
		return SortByID
	case SortByID:
		return SortByStatus
	default:
		return SortByStatus
	}
}
