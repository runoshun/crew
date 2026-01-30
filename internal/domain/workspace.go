package domain

import "time"

// WorkspaceRepo represents a repository registered in the workspace.
// Fields are ordered to minimize memory padding.
//
//nolint:govet // Field order follows TOML convention for readability
type WorkspaceRepo struct {
	Path       string    `toml:"path"`                 // Absolute path to the repository root
	Name       string    `toml:"name,omitempty"`       // Display name (defaults to directory basename)
	Pinned     bool      `toml:"pinned,omitempty"`     // Whether the repo is pinned to top
	LastOpened time.Time `toml:"last_opened,omitzero"` // Last time the repo was opened (omitzero for time.Time)
}

// WorkspaceFile represents the workspaces.toml file structure.
// Fields are ordered to minimize memory padding.
type WorkspaceFile struct {
	Repos   []WorkspaceRepo `toml:"repos"`   // List of registered repositories
	Version int             `toml:"version"` // File format version (currently 1)
}

// RepoState represents the state of a repository when loading.
type RepoState int

const (
	// RepoStateOK indicates the repository is accessible and initialized.
	RepoStateOK RepoState = iota
	// RepoStateNotGitRepo indicates the path is not a git repository.
	RepoStateNotGitRepo
	// RepoStateNotInitialized indicates crew has not been initialized (.crew not found).
	RepoStateNotInitialized
	// RepoStateConfigError indicates there was an error loading the repo config.
	RepoStateConfigError
	// RepoStateLoadError indicates there was an error loading tasks.
	RepoStateLoadError
	// RepoStateNotFound indicates the path does not exist.
	RepoStateNotFound
)

// String returns a human-readable description of the repo state.
func (s RepoState) String() string {
	switch s {
	case RepoStateOK:
		return "OK"
	case RepoStateNotGitRepo:
		return "Not a git repo"
	case RepoStateNotInitialized:
		return "Not initialized"
	case RepoStateConfigError:
		return "Config error"
	case RepoStateLoadError:
		return "Load error"
	case RepoStateNotFound:
		return "Not found"
	default:
		return "Unknown"
	}
}

// TaskSummary holds the count of tasks by status for a repository.
type TaskSummary struct {
	Todo        int
	InProgress  int
	NeedsInput  int
	ForReview   int
	Reviewing   int
	Reviewed    int
	Stopped     int
	Error       int
	Closed      int
	TotalActive int // Sum of all non-terminal statuses
}

// NewTaskSummary creates a TaskSummary from a list of tasks.
func NewTaskSummary(tasks []*Task) TaskSummary {
	var s TaskSummary
	for _, t := range tasks {
		switch t.Status { //nolint:exhaustive // statusDoneLegacy handled in default
		case StatusTodo:
			s.Todo++
		case StatusInProgress:
			s.InProgress++
		case StatusNeedsInput:
			s.NeedsInput++
		case StatusForReview:
			s.ForReview++
		case StatusReviewing:
			s.Reviewing++
		case StatusReviewed:
			s.Reviewed++
		case StatusStopped:
			s.Stopped++
		case StatusError:
			s.Error++
		case StatusClosed:
			s.Closed++
		default:
			// Handle legacy "done" status as closed
			if t.Status.IsLegacyDone() {
				s.Closed++
			}
		}
	}
	s.TotalActive = s.Todo + s.InProgress + s.NeedsInput + s.ForReview + s.Reviewing + s.Reviewed + s.Stopped + s.Error
	return s
}

// WorkspaceRepoInfo holds all information about a repo in the workspace view.
// Fields are ordered to minimize memory padding.
type WorkspaceRepoInfo struct {
	Repo     WorkspaceRepo // The repo configuration
	ErrorMsg string        // Error message if State indicates an error
	Summary  TaskSummary   // Task counts (only valid if State == RepoStateOK)
	State    RepoState     // Current state of the repo
}

// DisplayName returns the name to display for this repo.
// Returns Name if set, otherwise the basename of the path.
func (r *WorkspaceRepo) DisplayName() string {
	if r.Name != "" {
		return r.Name
	}
	// Return the last path component
	for i := len(r.Path) - 1; i >= 0; i-- {
		if r.Path[i] == '/' {
			return r.Path[i+1:]
		}
	}
	return r.Path
}

// WorkspaceRepository defines the interface for workspace persistence.
type WorkspaceRepository interface {
	// Load reads the workspace file and returns the repos list.
	// Returns empty list if file doesn't exist.
	Load() (*WorkspaceFile, error)

	// Save writes the workspace file.
	Save(file *WorkspaceFile) error

	// AddRepo adds a repository to the workspace.
	// Returns error if the path is already registered or if path is invalid.
	AddRepo(path string) error

	// RemoveRepo removes a repository from the workspace by path.
	RemoveRepo(path string) error

	// UpdateLastOpened updates the last_opened timestamp for a repo.
	UpdateLastOpened(path string) error
}
