package workspace

import "github.com/runoshun/git-crew/v2/internal/domain"

// Msg is the interface for all workspace TUI messages.
// All message types implement this sealed interface.
//
//sumtype:decl
type Msg interface {
	sealed()
}

// MsgReposLoaded is sent when repos are loaded from the workspace file.
//
//nolint:govet // Logical field order preferred
type MsgReposLoaded struct {
	Repos []domain.WorkspaceRepo
	Err   error
}

func (MsgReposLoaded) sealed() {}

// MsgSummaryLoaded is sent when a repo's task summary is loaded.
//
//nolint:govet // Logical field order preferred
type MsgSummaryLoaded struct {
	Info    domain.WorkspaceRepoInfo
	Path    string
	Loading bool // true if still loading
}

func (MsgSummaryLoaded) sealed() {}

// MsgRepoAdded is sent when a repo is successfully added.
type MsgRepoAdded struct {
	Err  error
	Path string
}

func (MsgRepoAdded) sealed() {}

// MsgRepoRemoved is sent when a repo is successfully removed.
type MsgRepoRemoved struct {
	Err  error
	Path string
}

func (MsgRepoRemoved) sealed() {}

// MsgError is sent when an error occurs.
type MsgError struct {
	Err error
}

func (MsgError) sealed() {}

// MsgOpenRepo is sent when the user wants to open a repo.
type MsgOpenRepo struct {
	Path string
}

func (MsgOpenRepo) sealed() {}

// MsgRepoExited is sent when returning from a repo TUI.
type MsgRepoExited struct{}

func (MsgRepoExited) sealed() {}

// MsgTick is sent periodically for auto-refresh.
type MsgTick struct{}

func (MsgTick) sealed() {}
