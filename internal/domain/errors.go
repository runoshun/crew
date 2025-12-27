package domain

import "errors"

// Domain errors.
var (
	ErrTaskNotFound       = errors.New("task not found")
	ErrParentNotFound     = errors.New("parent task not found")
	ErrInvalidTransition  = errors.New("invalid status transition")
	ErrSessionRunning     = errors.New("session already running")
	ErrNoSession          = errors.New("no running session")
	ErrWorktreeNotFound   = errors.New("worktree not found")
	ErrNoAgent            = errors.New("no agent specified")
	ErrUncommittedChanges = errors.New("uncommitted changes exist")
	ErrMergeConflict      = errors.New("merge conflict exists")
	ErrAlreadyInitialized = errors.New("crew already initialized")
	ErrNotInitialized     = errors.New("crew not initialized")
	ErrEmptyTitle         = errors.New("title cannot be empty")
	ErrEmptyMessage       = errors.New("message cannot be empty")
	ErrNotOnMainBranch    = errors.New("not on main branch")
)
