// Package tui provides the terminal user interface for git-crew.
package tui

// Mode represents the current UI mode.
type Mode int

const (
	ModeNormal            Mode = iota // Default navigation mode
	ModeFilter                        // Text filtering mode
	ModeConfirm                       // Confirmation dialog mode
	ModeInputTitle                    // Title input mode (for new task) - deprecated, use ModeNewTask
	ModeInputDesc                     // Description input mode (for new task) - deprecated, use ModeNewTask
	ModeNewTask                       // New task form mode (title, desc, parent)
	ModeStart                         // Agent picker mode
	ModeSelectManager                 // Manager picker mode
	ModeHelp                          // Help overlay mode
	ModeChangeStatus                  // Status change mode
	ModeExec                          // Execute command mode
	ModeActionMenu                    // Task action menu
	ModeReviewResult                  // Review result display mode
	ModeReviewAction                  // Review action selection mode (notify, merge, etc.)
	ModeReviewMessage                 // Review message input mode (for Request Changes)
	ModeEditReviewComment             // Edit review comment mode
	ModeBlock                         // Block task dialog mode
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
	case ModeSelectManager:
		return "select_manager"
	case ModeHelp:
		return "help"
	case ModeChangeStatus:
		return "change_status"
	case ModeExec:
		return "exec"
	case ModeActionMenu:
		return "action_menu"
	case ModeReviewResult:
		return "review_result"
	case ModeReviewAction:
		return "review_action"
	case ModeReviewMessage:
		return "review_message"
	case ModeEditReviewComment:
		return "edit_review_comment"
	case ModeBlock:
		return "block"
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
	case ModeFilter, ModeInputTitle, ModeInputDesc, ModeNewTask, ModeExec, ModeReviewMessage, ModeEditReviewComment, ModeBlock:
		return true
	case ModeNormal, ModeConfirm, ModeStart, ModeSelectManager, ModeHelp, ModeChangeStatus, ModeActionMenu, ModeReviewResult, ModeReviewAction:
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
	SortByStatusAsc SortMode = iota
	SortByStatusDesc
	SortByIDAsc
	SortByIDDesc
)

func (s SortMode) String() string {
	switch s {
	case SortByStatusAsc:
		return "status (asc)"
	case SortByStatusDesc:
		return "status (desc)"
	case SortByIDAsc:
		return "id (asc)"
	case SortByIDDesc:
		return "id (desc)"
	default:
		return "unknown"
	}
}

func (s SortMode) Next() SortMode {
	switch s {
	case SortByStatusAsc:
		return SortByStatusDesc
	case SortByStatusDesc:
		return SortByIDAsc
	case SortByIDAsc:
		return SortByIDDesc
	case SortByIDDesc:
		return SortByStatusAsc
	default:
		return SortByStatusAsc
	}
}
