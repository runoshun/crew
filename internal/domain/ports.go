package domain

import (
	"context"
	"io"
	"time"
)

// StoreInitializer initializes the data store.
type StoreInitializer interface {
	// Initialize creates the store if it doesn't exist.
	// Returns true if any repair was performed (e.g., NextTaskID was updated).
	Initialize() (repaired bool, err error)
	// IsInitialized checks if the store has been initialized.
	IsInitialized() bool
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

	// UpdateComment updates an existing comment of a task.
	UpdateComment(taskID, index int, comment Comment) error

	// SaveTaskWithComments atomically saves a task and its comments.
	// This is used when both task and comments need to be updated together.
	SaveTaskWithComments(task *Task, comments []Comment) error

	// === Snapshot operations ===

	// SaveSnapshot saves the current task state as a snapshot.
	// mainSHA is the git commit SHA to associate with this snapshot.
	SaveSnapshot(mainSHA string) error

	// RestoreSnapshot restores tasks from a snapshot.
	// Before restoring, the current state is saved as a new snapshot.
	RestoreSnapshot(snapshotRef string) error

	// ListSnapshots returns all snapshots for a given main SHA.
	// If mainSHA is empty, returns all snapshots.
	ListSnapshots(mainSHA string) ([]SnapshotInfo, error)

	// SyncSnapshot syncs task state with the current git HEAD.
	// If a snapshot exists for the current HEAD, restore it.
	SyncSnapshot() error

	// PruneSnapshots removes old snapshots, keeping the most recent keepCount per mainSHA.
	PruneSnapshots(keepCount int) error

	// === Remote sync operations ===

	// Push pushes task refs to remote.
	Push() error

	// Fetch fetches task refs from remote for a given namespace.
	Fetch(namespace string) error

	// ListNamespaces returns available namespaces on remote.
	ListNamespaces() ([]string, error)
}

// SnapshotInfo contains information about a snapshot.
// Fields are ordered to minimize memory padding.
type SnapshotInfo struct {
	CreatedAt time.Time // When the snapshot was created
	Ref       string    // Full ref name (e.g., refs/crew-xxx/snapshots/abc123_001)
	MainSHA   string    // Git commit SHA this snapshot is associated with
	Seq       int       // Sequence number within the same mainSHA
}

// TaskFilter specifies criteria for listing tasks.
// Fields are ordered to minimize memory padding.
type TaskFilter struct {
	ParentID *int     // nil = all tasks, set = only children of this parent
	Labels   []string // Filter by labels (AND condition)
}

// ProcessInfo contains information about a process.
// Fields are ordered to minimize memory padding.
type ProcessInfo struct {
	Command string // Process command
	State   string // Process state (S=Sleep, R=Running, Z=Zombie)
	PID     int    // Process ID
	PPID    int    // Parent process ID
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
	Peek(sessionName string, lines int, escape bool) (string, error)

	// Send sends keys to a session.
	Send(sessionName string, keys string) error

	// IsRunning checks if a session is running.
	IsRunning(sessionName string) (bool, error)

	// GetPaneProcesses retrieves process information for a session.
	GetPaneProcesses(sessionName string) ([]ProcessInfo, error)
}

// StartSessionOptions configures session creation.
type StartSessionOptions struct {
	Name      string // Session name
	Dir       string // Working directory
	Command   string // Command to run
	TaskTitle string // Task title for status bar
	TaskAgent string // Agent name for status bar
	TaskID    int    // Associated task ID
}

// WorktreeManager manages git worktrees.
type WorktreeManager interface {
	// Create creates a new worktree for the given branch.
	Create(branch, baseBranch string) (path string, err error)

	// SetupWorktree performs post-creation setup tasks (file copying and command execution).
	SetupWorktree(wtPath string, config *WorktreeConfig) error

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

	// GetMergeConflictFiles returns the list of files that would conflict
	// when merging branch into target. Returns empty slice if no conflicts.
	GetMergeConflictFiles(branch, target string) ([]string, error)

	// Merge merges a branch into the current branch.
	Merge(branch string, noFF bool) error

	// DeleteBranch deletes a branch.
	// If force is true, it uses -D (force delete), otherwise -d.
	DeleteBranch(branch string, force bool) error

	// ListBranches returns a list of all local branches.
	ListBranches() ([]string, error)

	// GetDefaultBranch returns the default branch name.
	// Priority: git config crew.defaultBranch > refs/remotes/origin/HEAD > "main"
	GetDefaultBranch() (string, error)
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
	// Load returns the merged configuration.
	// Priority (later takes precedence): global < override < .crew.toml < config.toml < config.runtime.toml
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
	IgnoreGlobal   bool // Skip loading global config
	IgnoreRepo     bool // Skip loading repo config (.git/crew/config.toml)
	IgnoreRootRepo bool // Skip loading root repo config (.crew.toml)
	IgnoreOverride bool // Skip loading override config (config.override.toml)
	IgnoreRuntime  bool // Skip loading runtime config (.git/crew/config.runtime.toml)
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

	// GetRootRepoConfigInfo returns information about the root repository config file (.crew.toml).
	GetRootRepoConfigInfo() ConfigInfo

	// GetOverrideConfigInfo returns information about the global override config file (config.override.toml).
	GetOverrideConfigInfo() ConfigInfo

	// GetRuntimeConfigInfo returns information about the runtime config file (.git/crew/config.runtime.toml).
	GetRuntimeConfigInfo() ConfigInfo

	// InitRepoConfig creates a repository config file with default template.
	// The cfg parameter should have builtin agents registered (via builtin.Register).
	// Returns error if file already exists.
	InitRepoConfig(cfg *Config) error

	// InitGlobalConfig creates a global config file with default template.
	// The cfg parameter should have builtin agents registered (via builtin.Register).
	// Returns error if file already exists.
	InitGlobalConfig(cfg *Config) error

	// InitOverrideConfig creates a global override config file with default template.
	// The cfg parameter should have builtin agents registered (via builtin.Register).
	// Returns error if file already exists.
	InitOverrideConfig(cfg *Config) error

	// SetAutoFix updates the auto_fix setting in the runtime config file (config.runtime.toml).
	// Creates the [complete] section if it doesn't exist.
	// Preserves other existing settings in the file.
	SetAutoFix(enabled bool) error
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

// Logger provides structured logging with task-aware output.
// Logs are written to both a global log file and task-specific log files.
type Logger interface {
	// Info logs an info message.
	Info(taskID int, category, msg string)

	// Debug logs a debug message.
	Debug(taskID int, category, msg string)

	// Warn logs a warning message.
	Warn(taskID int, category, msg string)

	// Error logs an error message.
	Error(taskID int, category, msg string)

	// Close closes any open log files.
	Close() error
}

// ScriptRunner executes shell scripts in a specified directory.
type ScriptRunner interface {
	// Run executes a script in the given directory.
	// Returns an error if the script execution fails.
	Run(dir, script string) error
}

// CommandExecutor executes external commands.
// This interface abstracts the execution of shell commands, enabling
// different implementations for CLI (direct execution) and testing (mocking).
type CommandExecutor interface {
	// Execute runs a command and returns its combined output.
	// The command is executed in the specified working directory.
	Execute(cmd *ExecCommand) (output []byte, err error)

	// ExecuteInteractive runs a command with stdin/stdout/stderr connected.
	// This is used for interactive commands that need terminal access.
	ExecuteInteractive(cmd *ExecCommand) error

	// ExecuteWithContext runs a command with context and custom stdout/stderr writers.
	// This is used for commands that need streaming output.
	ExecuteWithContext(ctx context.Context, cmd *ExecCommand, stdout, stderr io.Writer) error
}
