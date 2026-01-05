package domain

import "errors"

// Domain errors.
var (
	ErrTaskNotFound          = errors.New("task not found")
	ErrParentNotFound        = errors.New("parent task not found")
	ErrInvalidTransition     = errors.New("invalid status transition")
	ErrSessionRunning        = errors.New("session already running")
	ErrNoSession             = errors.New("no running session")
	ErrWorktreeNotFound      = errors.New("worktree not found")
	ErrNoAgent               = errors.New("no agent specified")
	ErrUncommittedChanges    = errors.New("uncommitted changes exist")
	ErrMergeConflict         = errors.New("merge conflict exists")
	ErrAlreadyInitialized    = errors.New("crew already initialized")
	ErrNotInitialized        = errors.New("crew not initialized (run 'git crew init' first)")
	ErrEmptyTitle            = errors.New("title cannot be empty")
	ErrEmptyMessage          = errors.New("message cannot be empty")
	ErrNotOnMainBranch       = errors.New("not on main branch")
	ErrNotGitRepository      = errors.New("not a git repository (or any of the parent directories)")
	ErrNoFieldsToUpdate      = errors.New("no fields to update")
	ErrConfigExists          = errors.New("config file already exists")
	ErrInvalidStatus         = errors.New("invalid status")
	ErrCircularInheritance   = errors.New("circular inheritance detected in worker configuration")
	ErrInheritParentNotFound = errors.New("inherit parent worker not found")
	ErrCommentNotFound       = errors.New("comment not found")
	ErrAgentNotFound         = errors.New("agent not found")
)
