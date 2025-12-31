package domain

import (
	"context"
	"time"
)

// StoreInitializer initializes the data store.
type StoreInitializer interface {
	// Initialize creates the store if it doesn't exist.
	Initialize() error
}

// TaskRepository manages task persistence.
type TaskRepository interface {
	// Get retrieves a task by ID. Returns nil if not found.
	Get(id int) (*Task, error)

	// List retrieves tasks matching the filter.
	List(filter TaskFilter) ([]*Task, error)

	// GetChildren retrieves direct children of a task.
	GetChildren(parentID int) ([]*Task, error)

	// Save creates or updates a task.
	Save(task *Task) error

	// Delete removes a task by ID.
	Delete(id int) error

	// NextID returns the next available task ID.
	NextID() (int, error)

	// GetComments retrieves comments for a task.
	GetComments(taskID int) ([]Comment, error)

	// AddComment adds a comment to a task.
	AddComment(taskID int, comment Comment) error
}

// TaskFilter specifies criteria for listing tasks.
// Fields are ordered to minimize memory padding.
type TaskFilter struct {
	ParentID *int     // nil = all tasks, set = only children of this parent
	Labels   []string // Filter by labels (AND condition)
}

// SessionManager manages tmux sessions.
type SessionManager interface {
	// Start creates and starts a new session.
	Start(ctx context.Context, opts StartSessionOptions) error

	// Stop terminates a session.
	Stop(sessionName string) error

	// Attach attaches to a running session.
	Attach(sessionName string) error

	// Peek captures the last N lines from a session.
	Peek(sessionName string, lines int) (string, error)

	// Send sends keys to a session.
	Send(sessionName string, keys string) error

	// IsRunning checks if a session is running.
	IsRunning(sessionName string) (bool, error)
}

// StartSessionOptions configures session creation.
type StartSessionOptions struct {
	Name    string // Session name
	Dir     string // Working directory
	Command string // Command to run
	TaskID  int    // Associated task ID
}

// WorktreeManager manages git worktrees.
type WorktreeManager interface {
	// Create creates a new worktree for the given branch.
	Create(branch, baseBranch string) (path string, err error)

	// Resolve returns the path of an existing worktree for the branch.
	Resolve(branch string) (path string, err error)

	// Remove deletes a worktree.
	Remove(branch string) error

	// Exists checks if a worktree exists for the branch.
	Exists(branch string) (bool, error)

	// List returns all worktrees.
	List() ([]WorktreeInfo, error)
}

// WorktreeInfo contains information about a worktree.
type WorktreeInfo struct {
	Path   string // Absolute path to worktree
	Branch string // Branch name
}

// Git provides git operations.
type Git interface {
	// CurrentBranch returns the name of the current branch.
	CurrentBranch() (string, error)

	// BranchExists checks if a branch exists.
	BranchExists(branch string) (bool, error)

	// HasUncommittedChanges checks for uncommitted changes in a directory.
	HasUncommittedChanges(dir string) (bool, error)

	// HasMergeConflict checks if merging branch into target would conflict.
	HasMergeConflict(branch, target string) (bool, error)

	// Merge merges a branch into the current branch.
	Merge(branch string, noFF bool) error

	// DeleteBranch deletes a branch.
	// If force is true, it uses -D (force delete), otherwise -d.
	DeleteBranch(branch string, force bool) error

	// ListBranches returns a list of all local branches.
	ListBranches() ([]string, error)
}

// GitHub provides GitHub integration via gh CLI.
type GitHub interface {
	// GetIssue retrieves issue information.
	GetIssue(number int) (*Issue, error)

	// CreatePR creates a pull request.
	CreatePR(opts CreatePROptions) (int, error)

	// UpdatePR updates an existing pull request.
	UpdatePR(number int, opts UpdatePROptions) error

	// FindPRByBranch finds a PR by branch name.
	FindPRByBranch(branch string) (int, error)

	// Push pushes a branch to remote.
	Push(branch string) error
}

// CreatePROptions configures PR creation.
type CreatePROptions struct {
	Title  string
	Body   string
	Branch string
	Base   string
}

// UpdatePROptions configures PR updates.
type UpdatePROptions struct {
	Title string
	Body  string
}

// ConfigLoader loads configuration from files.
type ConfigLoader interface {
	// Load returns the merged configuration (repo + global).
	Load() (*Config, error)

	// LoadGlobal returns only the global configuration.
	LoadGlobal() (*Config, error)

	// LoadRepo returns only the repository configuration.
	LoadRepo() (*Config, error)

	// LoadWithOptions returns the merged configuration with options to ignore sources.
	LoadWithOptions(opts LoadConfigOptions) (*Config, error)
}

// LoadConfigOptions specifies options for loading configuration.
type LoadConfigOptions struct {
	IgnoreGlobal bool // Skip loading global config
	IgnoreRepo   bool // Skip loading repo config
}

// ConfigInfo holds information about a config file.
type ConfigInfo struct {
	Path    string // Path to the config file
	Content string // Raw content (empty if not found)
	Exists  bool   // Whether the file exists
}

// ConfigManager manages configuration files.
type ConfigManager interface {
	// GetRepoConfigInfo returns information about the repository config file.
	GetRepoConfigInfo() ConfigInfo

	// GetGlobalConfigInfo returns information about the global config file.
	GetGlobalConfigInfo() ConfigInfo

	// InitRepoConfig creates a repository config file with default template.
	// Returns error if file already exists.
	InitRepoConfig() error

	// InitGlobalConfig creates a global config file with default template.
	// Returns error if file already exists.
	InitGlobalConfig() error
}

// Clock provides time operations for testability.
type Clock interface {
	// Now returns the current time.
	Now() time.Time
}

// RealClock implements Clock using the system clock.
type RealClock struct{}

// Now returns the current time.
func (RealClock) Now() time.Time {
	return time.Now()
}
