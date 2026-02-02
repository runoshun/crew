package tui

import "github.com/runoshun/git-crew/v2/internal/domain"

// Msg is the sealed interface for all TUI messages.
// All message types must implement the sealed() method.
// Messages intended for repo routing must implement Msg; non-Msg tea.Msg
// are treated as program-level messages by workspace.
//
// go-sumtype:decl Msg
type Msg interface {
	sealed()
}

// MsgTasksLoaded is sent when tasks are loaded from the repository.
type MsgTasksLoaded struct {
	Tasks []*domain.Task
}

func (MsgTasksLoaded) sealed() {}

// MsgTaskStarted is sent when a task session is started.
type MsgTaskStarted struct {
	SessionName string
	TaskID      int
}

func (MsgTaskStarted) sealed() {}

// MsgTaskStopped is sent when a task session is stopped.
type MsgTaskStopped struct {
	SessionName string
	TaskID      int
}

func (MsgTaskStopped) sealed() {}

// MsgTaskCreated is sent when a new task is created.
type MsgTaskCreated struct {
	TaskID int
}

func (MsgTaskCreated) sealed() {}

// MsgTaskDeleted is sent when a task is deleted.
type MsgTaskDeleted struct {
	TaskID int
}

func (MsgTaskDeleted) sealed() {}

// MsgTaskClosed is sent when a task is closed.
type MsgTaskClosed struct {
	TaskID int
}

func (MsgTaskClosed) sealed() {}

// MsgTaskMerged is sent when a task is merged.
type MsgTaskMerged struct {
	TaskID int
}

func (MsgTaskMerged) sealed() {}

// MsgTaskStatusUpdated is sent when a task status is updated.
type MsgTaskStatusUpdated struct {
	Status domain.Status
	TaskID int
}

func (MsgTaskStatusUpdated) sealed() {}

// MsgTaskCopied is sent when a task is copied.
type MsgTaskCopied struct {
	OriginalID int
	NewID      int
}

func (MsgTaskCopied) sealed() {}

// MsgError is sent when an error occurs.
type MsgError struct {
	Err error
}

func (MsgError) sealed() {}

// MsgClearError is sent to clear the current error message.
type MsgClearError struct{}

func (MsgClearError) sealed() {}

// MsgConfigLoaded is sent when configuration is loaded.
type MsgConfigLoaded struct {
	Config *domain.Config
}

func (MsgConfigLoaded) sealed() {}

// MsgAttachSession is sent to trigger session attachment.
// This is handled specially to suspend the TUI.
type MsgAttachSession struct {
	TaskID int
}

func (MsgAttachSession) sealed() {}

// MsgReloadTasks is sent after external commands complete to reload the task list.
type MsgReloadTasks struct{}

func (MsgReloadTasks) sealed() {}

// MsgShowDiff is sent to trigger diff display for a task.
// This is handled specially to suspend the TUI and run an external command.
type MsgShowDiff struct {
	TaskID int
}

func (MsgShowDiff) sealed() {}

// MsgTick is sent periodically for auto-refresh.
type MsgTick struct{}

func (MsgTick) sealed() {}

// MsgCommentsLoaded is sent when comments are loaded for a task.
type MsgCommentsLoaded struct {
	Comments []domain.Comment
	TaskID   int
}

func (MsgCommentsLoaded) sealed() {}

// MsgCommentCountsLoaded is sent when comment counts are loaded for all tasks.
type MsgCommentCountsLoaded struct {
	CommentCounts map[int]int // taskID -> comment count
}

func (MsgCommentCountsLoaded) sealed() {}

// MsgReviewActionCompleted is sent when a review action is completed.
type MsgReviewActionCompleted struct {
	TaskID int
}

func (MsgReviewActionCompleted) sealed() {}

// MsgPrepareEditComment is sent when preparing to edit a review comment.
type MsgPrepareEditComment struct {
	Message string
	TaskID  int
	Index   int
}

func (MsgPrepareEditComment) sealed() {}

// MsgReviewResultLoaded is sent when review result (reviewer comment) is loaded for display.
type MsgReviewResultLoaded struct {
	Review string
	TaskID int
}

func (MsgReviewResultLoaded) sealed() {}

// MsgManagerSessionStarted is sent when manager session is started.
type MsgManagerSessionStarted struct {
	SessionName string
}

func (MsgManagerSessionStarted) sealed() {}

// MsgAttachManagerSession is sent to trigger manager session attachment.
type MsgAttachManagerSession struct{}

func (MsgAttachManagerSession) sealed() {}

// MsgShowManagerSelect is sent to show manager agent selection UI.
type MsgShowManagerSelect struct{}

func (MsgShowManagerSelect) sealed() {}

// MsgFocusWorkspace is sent when TUI wants to return focus to the workspace pane.
// This is used in workspace mode when Tab is pressed from the detail panel.
type MsgFocusWorkspace struct{}

func (MsgFocusWorkspace) sealed() {}
