package workspace

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/runoshun/git-crew/v2/internal/domain"
)

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
	Info domain.WorkspaceRepoInfo
	Path string
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

// MsgTick is sent periodically for auto-refresh.
type MsgTick struct{}

func (MsgTick) sealed() {}

// RepoMsg wraps a message with a repo path for routing.
//
//nolint:govet // Logical field order preferred
type RepoMsg struct {
	Path string
	Msg  tea.Msg
}

func (RepoMsg) sealed() {}
