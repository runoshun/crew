// Package testutil provides shared test utilities and mock implementations.
package testutil

import (
	"context"
	"io"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/infra/config"
)

// MockClock is a test double for domain.Clock.
type MockClock struct {
	NowTime time.Time
}

// Now returns the configured time.
func (m *MockClock) Now() time.Time {
	return m.NowTime
}

// MockTaskRepository is a test double for domain.TaskRepository.
// Fields are ordered to minimize memory padding.
type MockTaskRepository struct {
	Tasks    map[int]*domain.Task
	Comments map[int][]domain.Comment
	SaveErr  error
	GetErr   error
	NextIDN  int
}

// NewMockTaskRepository creates a new MockTaskRepository with initialized maps.
func NewMockTaskRepository() *MockTaskRepository {
	return &MockTaskRepository{
		Tasks:    make(map[int]*domain.Task),
		NextIDN:  1,
		Comments: make(map[int][]domain.Comment),
	}
}

// Get retrieves a task by ID.
func (m *MockTaskRepository) Get(id int) (*domain.Task, error) {
	if m.GetErr != nil {
		return nil, m.GetErr
	}
	task, ok := m.Tasks[id]
	if !ok {
		return nil, nil
	}
	return task, nil
}

// List returns all tasks (filtering not implemented in mock).
func (m *MockTaskRepository) List(_ domain.TaskFilter) ([]*domain.Task, error) {
	tasks := make([]*domain.Task, 0, len(m.Tasks))
	for _, t := range m.Tasks {
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// GetChildren returns children of a parent task.
func (m *MockTaskRepository) GetChildren(parentID int) ([]*domain.Task, error) {
	var tasks []*domain.Task
	for _, t := range m.Tasks {
		if t.ParentID != nil && *t.ParentID == parentID {
			tasks = append(tasks, t)
		}
	}
	return tasks, nil
}

// Save saves a task.
func (m *MockTaskRepository) Save(task *domain.Task) error {
	if m.SaveErr != nil {
		return m.SaveErr
	}
	m.Tasks[task.ID] = task
	return nil
}

// Delete removes a task by ID.
func (m *MockTaskRepository) Delete(id int) error {
	delete(m.Tasks, id)
	return nil
}

// NextID returns the next available task ID.
func (m *MockTaskRepository) NextID() (int, error) {
	id := m.NextIDN
	m.NextIDN++
	return id, nil
}

// GetComments returns comments for a task.
func (m *MockTaskRepository) GetComments(taskID int) ([]domain.Comment, error) {
	return m.Comments[taskID], nil
}

// AddComment adds a comment to a task.
func (m *MockTaskRepository) AddComment(taskID int, comment domain.Comment) error {
	m.Comments[taskID] = append(m.Comments[taskID], comment)
	return nil
}

// UpdateComment updates an existing comment of a task.
func (m *MockTaskRepository) UpdateComment(taskID, index int, comment domain.Comment) error {
	comments := m.Comments[taskID]
	if index < 0 || index >= len(comments) {
		return domain.ErrCommentNotFound
	}
	comments[index] = comment
	m.Comments[taskID] = comments
	return nil
}

// SaveTaskWithComments atomically saves a task and its comments.
func (m *MockTaskRepository) SaveTaskWithComments(task *domain.Task, comments []domain.Comment) error {
	if m.SaveErr != nil {
		return m.SaveErr
	}
	m.Tasks[task.ID] = task
	m.Comments[task.ID] = comments
	return nil
}

// MockStoreInitializer is a test double for domain.StoreInitializer.
type MockStoreInitializer struct {
	Initialized bool
	Repaired    bool
}

// Initialize is a no-op for testing.
func (m *MockStoreInitializer) Initialize() (bool, error) {
	return m.Repaired, nil
}

// IsInitialized returns the configured value.
func (m *MockStoreInitializer) IsInitialized() bool {
	return m.Initialized
}

// MockTaskRepositoryWithNextIDError extends MockTaskRepository to return error on NextID.
type MockTaskRepositoryWithNextIDError struct {
	*MockTaskRepository
	NextIDErr error
}

// NextID returns an error if configured.
func (m *MockTaskRepositoryWithNextIDError) NextID() (int, error) {
	if m.NextIDErr != nil {
		return 0, m.NextIDErr
	}
	return m.MockTaskRepository.NextID()
}

// MockTaskRepositoryWithListError extends MockTaskRepository to return error on List.
type MockTaskRepositoryWithListError struct {
	*MockTaskRepository
	ListErr error
}

// List returns an error if configured.
func (m *MockTaskRepositoryWithListError) List(_ domain.TaskFilter) ([]*domain.Task, error) {
	if m.ListErr != nil {
		return nil, m.ListErr
	}
	return m.MockTaskRepository.List(domain.TaskFilter{})
}

// MockTaskRepositoryWithChildrenError extends MockTaskRepository to return error on GetChildren.
type MockTaskRepositoryWithChildrenError struct {
	*MockTaskRepository
	ChildrenErr error
}

// GetChildren returns an error if configured.
func (m *MockTaskRepositoryWithChildrenError) GetChildren(_ int) ([]*domain.Task, error) {
	if m.ChildrenErr != nil {
		return nil, m.ChildrenErr
	}
	return m.MockTaskRepository.GetChildren(0)
}

// MockTaskRepositoryWithCommentsError extends MockTaskRepository to return error on GetComments.
type MockTaskRepositoryWithCommentsError struct {
	*MockTaskRepository
	CommentsErr error
}

// GetComments returns an error if configured.
func (m *MockTaskRepositoryWithCommentsError) GetComments(_ int) ([]domain.Comment, error) {
	if m.CommentsErr != nil {
		return nil, m.CommentsErr
	}
	return m.MockTaskRepository.GetComments(0)
}

// MockTaskRepositoryWithDeleteError extends MockTaskRepository to return error on Delete.
type MockTaskRepositoryWithDeleteError struct {
	*MockTaskRepository
	DeleteErr error
}

// Delete returns an error if configured.
func (m *MockTaskRepositoryWithDeleteError) Delete(_ int) error {
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	return m.MockTaskRepository.Delete(0)
}

// MockTaskRepositoryWithAddCommentError extends MockTaskRepository to return error on AddComment.
type MockTaskRepositoryWithAddCommentError struct {
	*MockTaskRepository
	AddCommentErr error
}

// AddComment returns an error if configured.
func (m *MockTaskRepositoryWithAddCommentError) AddComment(_ int, _ domain.Comment) error {
	if m.AddCommentErr != nil {
		return m.AddCommentErr
	}
	return nil
}

// MockTaskRepositoryWithUpdateCommentError extends MockTaskRepository to return error on UpdateComment.
type MockTaskRepositoryWithUpdateCommentError struct {
	*MockTaskRepository
	UpdateCommentErr error
}

// UpdateComment returns an error if configured.
func (m *MockTaskRepositoryWithUpdateCommentError) UpdateComment(_ int, _ int, _ domain.Comment) error {
	if m.UpdateCommentErr != nil {
		return m.UpdateCommentErr
	}
	return nil
}

// MockGit is a test double for domain.Git.
// Fields are ordered to minimize memory padding.
type MockGit struct {
	CurrentBranchErr       error
	HasUncommittedErr      error
	MergeErr               error
	DeleteBranchErr        error
	GetDefaultBranchErr    error
	MergeConflictErr       error
	CurrentBranchName      string
	DefaultBranchName      string
	MergeBranch            string
	DeletedBranch          string
	MergeConflictFiles     []string
	HasUncommittedChangesV bool
	MergeNoFF              bool
	MergeCalled            bool
	DeleteBranchCalled     bool
	GetDefaultBranchCalled bool
}

// Ensure MockGit implements domain.Git interface.
var _ domain.Git = (*MockGit)(nil)

// CurrentBranch returns the configured branch name or error.
func (m *MockGit) CurrentBranch() (string, error) {
	if m.CurrentBranchErr != nil {
		return "", m.CurrentBranchErr
	}
	return m.CurrentBranchName, nil
}

// BranchExists returns the configured value or error.
func (m *MockGit) BranchExists(branch string) (bool, error) {
	// Simple mock: assume true unless configured otherwise
	return true, nil
}

// ListBranches returns the configured branches or error.
func (m *MockGit) ListBranches() ([]string, error) {
	// Return a default list or configured list
	return []string{"main", "crew-1", "crew-2"}, nil
}

// HasUncommittedChanges returns the configured value or error.
func (m *MockGit) HasUncommittedChanges(_ string) (bool, error) {
	if m.HasUncommittedErr != nil {
		return false, m.HasUncommittedErr
	}
	return m.HasUncommittedChangesV, nil
}

// HasMergeConflict returns whether there are conflicts based on MergeConflictFiles.
func (m *MockGit) HasMergeConflict(_, _ string) (bool, error) {
	if m.MergeConflictErr != nil {
		return false, m.MergeConflictErr
	}
	return len(m.MergeConflictFiles) > 0, nil
}

// GetMergeConflictFiles returns the configured conflict files or error.
func (m *MockGit) GetMergeConflictFiles(_, _ string) ([]string, error) {
	if m.MergeConflictErr != nil {
		return nil, m.MergeConflictErr
	}
	return m.MergeConflictFiles, nil
}

// Merge records the call and returns configured error.
func (m *MockGit) Merge(branch string, noFF bool) error {
	m.MergeCalled = true
	m.MergeBranch = branch
	m.MergeNoFF = noFF
	return m.MergeErr
}

// DeleteBranch records the call and returns configured error.
func (m *MockGit) DeleteBranch(branch string, force bool) error {
	m.DeleteBranchCalled = true
	m.DeletedBranch = branch
	// force is ignored in mock for now, or we could add a field to verify it
	return m.DeleteBranchErr
}

// GetDefaultBranch returns the configured default branch name or error.
func (m *MockGit) GetDefaultBranch() (string, error) {
	m.GetDefaultBranchCalled = true
	if m.GetDefaultBranchErr != nil {
		return "", m.GetDefaultBranchErr
	}
	if m.DefaultBranchName != "" {
		return m.DefaultBranchName, nil
	}
	return "main", nil
}

// MockSessionManager is a test double for domain.SessionManager.
// Fields are ordered to minimize memory padding.
type MockSessionManager struct {
	IsRunningErr  error
	StartErr      error
	StopErr       error
	AttachErr     error
	SendErr       error
	PeekErr       error
	PeekOutput    string
	SentKeys      string
	IsRunningFunc func(string) (bool, error)
	SendFunc      func(string, string) error // Custom Send function for complex test scenarios
	StartOpts     domain.StartSessionOptions
	PeekLines     int
	PeekEscape    bool
	IsRunningVal  bool
	StartCalled   bool
	StopCalled    bool
	AttachCalled  bool
	SendCalled    bool
	PeekCalled    bool
}

// NewMockSessionManager creates a new MockSessionManager.
func NewMockSessionManager() *MockSessionManager {
	return &MockSessionManager{}
}

// Ensure MockSessionManager implements domain.SessionManager interface.
var _ domain.SessionManager = (*MockSessionManager)(nil)

// Start records the call and returns configured error.
func (m *MockSessionManager) Start(_ context.Context, opts domain.StartSessionOptions) error {
	m.StartCalled = true
	m.StartOpts = opts
	return m.StartErr
}

// Stop records the call and returns configured error.
func (m *MockSessionManager) Stop(_ string) error {
	m.StopCalled = true
	return m.StopErr
}

// Attach records the call and returns configured error.
func (m *MockSessionManager) Attach(_ string) error {
	m.AttachCalled = true
	return m.AttachErr
}

// Peek records the call and returns configured output or error.
func (m *MockSessionManager) Peek(_ string, lines int, escape bool) (string, error) {
	m.PeekCalled = true
	m.PeekLines = lines
	m.PeekEscape = escape
	if m.PeekErr != nil {
		return "", m.PeekErr
	}
	return m.PeekOutput, nil
}

// Send records the call and returns configured error.
func (m *MockSessionManager) Send(name string, keys string) error {
	m.SendCalled = true
	m.SentKeys = keys
	if m.SendFunc != nil {
		return m.SendFunc(name, keys)
	}
	return m.SendErr
}

// IsRunning returns the configured value or error.
func (m *MockSessionManager) IsRunning(name string) (bool, error) {
	if m.IsRunningFunc != nil {
		return m.IsRunningFunc(name)
	}
	if m.IsRunningErr != nil {
		return false, m.IsRunningErr
	}
	return m.IsRunningVal, nil
}

// GetPaneProcesses returns an empty list (mock implementation).
func (m *MockSessionManager) GetPaneProcesses(_ string) ([]domain.ProcessInfo, error) {
	return nil, nil
}

// MockWorktreeManager is a test double for domain.WorktreeManager.
// Fields are ordered to minimize memory padding.
type MockWorktreeManager struct {
	SetupWorktreeConfig *domain.WorktreeConfig // Captured config from SetupWorktree call
	CreateErr           error
	ResolveErr          error
	RemoveErr           error
	ExistsErr           error
	SetupErr            error
	CreatePath          string
	ResolvePath         string
	SetupWorktreePath   string // Captured path from SetupWorktree call
	ExistsVal           bool
	CreateCalled        bool
	RemoveCalled        bool
	SetupCalled         bool
}

// NewMockWorktreeManager creates a new MockWorktreeManager.
func NewMockWorktreeManager() *MockWorktreeManager {
	return &MockWorktreeManager{
		CreatePath: "/tmp/worktree",
	}
}

// Ensure MockWorktreeManager implements domain.WorktreeManager interface.
var _ domain.WorktreeManager = (*MockWorktreeManager)(nil)

// Create records the call and returns configured path or error.
func (m *MockWorktreeManager) Create(_, _ string) (string, error) {
	m.CreateCalled = true
	if m.CreateErr != nil {
		return "", m.CreateErr
	}
	return m.CreatePath, nil
}

// SetupWorktree captures the arguments and returns configured error.
func (m *MockWorktreeManager) SetupWorktree(path string, config *domain.WorktreeConfig) error {
	m.SetupCalled = true
	m.SetupWorktreePath = path
	m.SetupWorktreeConfig = config
	return m.SetupErr
}

// Resolve returns the configured path or error.
func (m *MockWorktreeManager) Resolve(_ string) (string, error) {
	if m.ResolveErr != nil {
		return "", m.ResolveErr
	}
	return m.ResolvePath, nil
}

// Remove records the call and returns configured error.
func (m *MockWorktreeManager) Remove(_ string) error {
	m.RemoveCalled = true
	return m.RemoveErr
}

// Exists returns the configured value or error.
func (m *MockWorktreeManager) Exists(_ string) (bool, error) {
	if m.ExistsErr != nil {
		return false, m.ExistsErr
	}
	return m.ExistsVal, nil
}

// List returns the configured worktrees or error.
func (m *MockWorktreeManager) List() ([]domain.WorktreeInfo, error) {
	return []domain.WorktreeInfo{}, nil
}

// MockConfigLoader is a test double for domain.ConfigLoader.
type MockConfigLoader struct {
	Config       *domain.Config
	GlobalConfig *domain.Config
	RepoConfig   *domain.Config
	LoadErr      error
	GlobalErr    error
	RepoErr      error
	LastOpts     domain.LoadConfigOptions // Records the last options passed to LoadWithOptions
}

// NewMockConfigLoader creates a new MockConfigLoader with default config.
// It also registers builtin agents and resolves inheritance to match the real ConfigLoader behavior.
func NewMockConfigLoader() *MockConfigLoader {
	cfg := domain.NewDefaultConfig()
	config.Register(cfg)
	// Resolve inheritance like the real loader does
	_ = cfg.ResolveInheritance()
	return &MockConfigLoader{
		Config: cfg,
	}
}

// Ensure MockConfigLoader implements domain.ConfigLoader interface.
var _ domain.ConfigLoader = (*MockConfigLoader)(nil)

// Load returns the configured config or error.
func (m *MockConfigLoader) Load() (*domain.Config, error) {
	if m.LoadErr != nil {
		return nil, m.LoadErr
	}
	return m.Config, nil
}

// LoadGlobal returns the configured config or error.
func (m *MockConfigLoader) LoadGlobal() (*domain.Config, error) {
	if m.GlobalErr != nil {
		return nil, m.GlobalErr
	}
	if m.GlobalConfig != nil {
		return m.GlobalConfig, nil
	}
	return m.Config, nil
}

// LoadRepo returns the configured repo config or error.
func (m *MockConfigLoader) LoadRepo() (*domain.Config, error) {
	if m.RepoErr != nil {
		return nil, m.RepoErr
	}
	if m.RepoConfig != nil {
		return m.RepoConfig, nil
	}
	return m.Config, nil
}

// LoadWithOptions returns config based on options.
func (m *MockConfigLoader) LoadWithOptions(opts domain.LoadConfigOptions) (*domain.Config, error) {
	m.LastOpts = opts
	if m.LoadErr != nil {
		return nil, m.LoadErr
	}
	// For testing, just return the default config
	// More sophisticated mocking can be added if needed
	return m.Config, nil
}

// MockConfigManager is a test double for domain.ConfigManager.
// Fields are ordered to minimize memory padding.
type MockConfigManager struct {
	InitRepoErr        error
	InitGlobalErr      error
	InitOverrideErr    error
	SetAutoFixErr      error
	RepoConfigInfo     domain.ConfigInfo
	GlobalConfigInfo   domain.ConfigInfo
	RootRepoConfigInfo domain.ConfigInfo
	OverrideConfigInfo domain.ConfigInfo
	RuntimeConfigInfo  domain.ConfigInfo
	InitRepoCalled     bool
	InitGlobalCalled   bool
	InitOverrideCalled bool
	SetAutoFixCalled   bool
	SetAutoFixValue    bool
}

// NewMockConfigManager creates a new MockConfigManager.
func NewMockConfigManager() *MockConfigManager {
	return &MockConfigManager{
		RepoConfigInfo: domain.ConfigInfo{
			Path:   "/test/.git/crew/config.toml",
			Exists: false,
		},
		GlobalConfigInfo: domain.ConfigInfo{
			Path:   "/home/test/.config/crew/config.toml",
			Exists: false,
		},
		RootRepoConfigInfo: domain.ConfigInfo{
			Path:   "/test/.crew.toml",
			Exists: false,
		},
		OverrideConfigInfo: domain.ConfigInfo{
			Path:   "/home/test/.config/crew/config.override.toml",
			Exists: false,
		},
		RuntimeConfigInfo: domain.ConfigInfo{
			Path:   "/test/.git/crew/config.runtime.toml",
			Exists: false,
		},
	}
}

// Ensure MockConfigManager implements domain.ConfigManager interface.
var _ domain.ConfigManager = (*MockConfigManager)(nil)

// GetRepoConfigInfo returns the configured repo config info.
func (m *MockConfigManager) GetRepoConfigInfo() domain.ConfigInfo {
	return m.RepoConfigInfo
}

// GetGlobalConfigInfo returns the configured global config info.
func (m *MockConfigManager) GetGlobalConfigInfo() domain.ConfigInfo {
	return m.GlobalConfigInfo
}

// GetRootRepoConfigInfo returns the configured root repository config info.
func (m *MockConfigManager) GetRootRepoConfigInfo() domain.ConfigInfo {
	return m.RootRepoConfigInfo
}

// GetOverrideConfigInfo returns the configured override config info.
func (m *MockConfigManager) GetOverrideConfigInfo() domain.ConfigInfo {
	return m.OverrideConfigInfo
}

// GetRuntimeConfigInfo returns the configured runtime config info.
func (m *MockConfigManager) GetRuntimeConfigInfo() domain.ConfigInfo {
	return m.RuntimeConfigInfo
}

// InitRepoConfig records the call and returns configured error.
func (m *MockConfigManager) InitRepoConfig(_ *domain.Config) error {
	m.InitRepoCalled = true
	return m.InitRepoErr
}

// InitGlobalConfig records the call and returns configured error.
func (m *MockConfigManager) InitGlobalConfig(_ *domain.Config) error {
	m.InitGlobalCalled = true
	return m.InitGlobalErr
}

// InitOverrideConfig records the call and returns configured error.
func (m *MockConfigManager) InitOverrideConfig(_ *domain.Config) error {
	m.InitOverrideCalled = true
	return m.InitOverrideErr
}

// SetAutoFix records the call and returns configured error.
func (m *MockConfigManager) SetAutoFix(enabled bool) error {
	m.SetAutoFixCalled = true
	m.SetAutoFixValue = enabled
	return m.SetAutoFixErr
}

// === Snapshot methods (no-op for mock) ===

// SaveSnapshot is a no-op.
func (m *MockTaskRepository) SaveSnapshot(mainSHA string) error {
	return nil
}

// RestoreSnapshot is a no-op.
func (m *MockTaskRepository) RestoreSnapshot(snapshotRef string) error {
	return nil
}

// ListSnapshots returns empty.
func (m *MockTaskRepository) ListSnapshots(mainSHA string) ([]domain.SnapshotInfo, error) {
	return nil, nil
}

// SyncSnapshot is a no-op.
func (m *MockTaskRepository) SyncSnapshot() error {
	return nil
}

// PruneSnapshots is a no-op.
func (m *MockTaskRepository) PruneSnapshots(keepCount int) error {
	return nil
}

// === Snapshot methods for MockTaskRepositoryWithAddCommentError ===

// SaveSnapshot is a no-op.
func (m *MockTaskRepositoryWithAddCommentError) SaveSnapshot(mainSHA string) error {
	return nil
}

// RestoreSnapshot is a no-op.
func (m *MockTaskRepositoryWithAddCommentError) RestoreSnapshot(snapshotRef string) error {
	return nil
}

// ListSnapshots returns empty.
func (m *MockTaskRepositoryWithAddCommentError) ListSnapshots(mainSHA string) ([]domain.SnapshotInfo, error) {
	return nil, nil
}

// SyncSnapshot is a no-op.
func (m *MockTaskRepositoryWithAddCommentError) SyncSnapshot() error {
	return nil
}

// PruneSnapshots is a no-op.
func (m *MockTaskRepositoryWithAddCommentError) PruneSnapshots(keepCount int) error {
	return nil
}

// === Remote sync methods (no-op for mock) ===

func (m *MockTaskRepository) Push() error                       { return nil }
func (m *MockTaskRepository) Fetch(_ string) error              { return nil }
func (m *MockTaskRepository) ListNamespaces() ([]string, error) { return nil, nil }

func (m *MockTaskRepositoryWithAddCommentError) Push() error                       { return nil }
func (m *MockTaskRepositoryWithAddCommentError) Fetch(_ string) error              { return nil }
func (m *MockTaskRepositoryWithAddCommentError) ListNamespaces() ([]string, error) { return nil, nil }
func (m *MockTaskRepositoryWithAddCommentError) SaveTaskWithComments(task *domain.Task, comments []domain.Comment) error {
	return m.MockTaskRepository.SaveTaskWithComments(task, comments)
}

func (m *MockTaskRepositoryWithUpdateCommentError) SaveSnapshot(mainSHA string) error { return nil }
func (m *MockTaskRepositoryWithUpdateCommentError) RestoreSnapshot(snapshotRef string) error {
	return nil
}
func (m *MockTaskRepositoryWithUpdateCommentError) ListSnapshots(mainSHA string) ([]domain.SnapshotInfo, error) {
	return nil, nil
}
func (m *MockTaskRepositoryWithUpdateCommentError) SyncSnapshot() error                { return nil }
func (m *MockTaskRepositoryWithUpdateCommentError) PruneSnapshots(keepCount int) error { return nil }
func (m *MockTaskRepositoryWithUpdateCommentError) Push() error                        { return nil }
func (m *MockTaskRepositoryWithUpdateCommentError) Fetch(_ string) error               { return nil }
func (m *MockTaskRepositoryWithUpdateCommentError) ListNamespaces() ([]string, error) {
	return nil, nil
}
func (m *MockTaskRepositoryWithUpdateCommentError) SaveTaskWithComments(task *domain.Task, comments []domain.Comment) error {
	return m.MockTaskRepository.SaveTaskWithComments(task, comments)
}

// MockLogger is a test double for domain.Logger.
// It captures all log calls for verification in tests.
type MockLogger struct {
	Entries []LogEntry
}

// LogEntry represents a single log entry.
// Fields are ordered to minimize memory padding.
type LogEntry struct {
	Level    string
	Category string
	Msg      string
	TaskID   int
}

// NewMockLogger creates a new MockLogger.
func NewMockLogger() *MockLogger {
	return &MockLogger{
		Entries: make([]LogEntry, 0),
	}
}

// Ensure MockLogger implements domain.Logger interface.
var _ domain.Logger = (*MockLogger)(nil)

// Info logs an info message.
func (m *MockLogger) Info(taskID int, category, msg string) {
	m.Entries = append(m.Entries, LogEntry{Level: "INFO", TaskID: taskID, Category: category, Msg: msg})
}

// Debug logs a debug message.
func (m *MockLogger) Debug(taskID int, category, msg string) {
	m.Entries = append(m.Entries, LogEntry{Level: "DEBUG", TaskID: taskID, Category: category, Msg: msg})
}

// Warn logs a warning message.
func (m *MockLogger) Warn(taskID int, category, msg string) {
	m.Entries = append(m.Entries, LogEntry{Level: "WARN", TaskID: taskID, Category: category, Msg: msg})
}

// Error logs an error message.
func (m *MockLogger) Error(taskID int, category, msg string) {
	m.Entries = append(m.Entries, LogEntry{Level: "ERROR", TaskID: taskID, Category: category, Msg: msg})
}

// Close closes the logger (no-op for mock).
func (m *MockLogger) Close() error {
	return nil
}

// MockScriptRunner is a test double for domain.ScriptRunner.
// Fields are ordered to minimize memory padding.
type MockScriptRunner struct {
	RunErr    error
	RunDir    string
	RunScript string
	RunCalled bool
}

// NewMockScriptRunner creates a new MockScriptRunner.
func NewMockScriptRunner() *MockScriptRunner {
	return &MockScriptRunner{}
}

// Ensure MockScriptRunner implements domain.ScriptRunner interface.
var _ domain.ScriptRunner = (*MockScriptRunner)(nil)

// Run records the call and returns configured error.
func (m *MockScriptRunner) Run(dir, script string) error {
	m.RunCalled = true
	m.RunDir = dir
	m.RunScript = script
	return m.RunErr
}

// MockCommandExecutor is a test double for domain.CommandExecutor.
// Fields are ordered to minimize memory padding.
type MockCommandExecutor struct {
	ExecutedCmd              *domain.ExecCommand
	ExecuteErr               error
	ExecuteInteractiveErr    error
	ExecuteWithContextErr    error
	ExecuteOutput            []byte
	ExecuteCalled            bool
	ExecuteInteractiveCalled bool
	ExecuteWithContextCalled bool
}

// NewMockCommandExecutor creates a new MockCommandExecutor.
func NewMockCommandExecutor() *MockCommandExecutor {
	return &MockCommandExecutor{}
}

// Ensure MockCommandExecutor implements domain.CommandExecutor interface.
var _ domain.CommandExecutor = (*MockCommandExecutor)(nil)

// Execute records the call and returns configured output or error.
func (m *MockCommandExecutor) Execute(cmd *domain.ExecCommand) ([]byte, error) {
	m.ExecuteCalled = true
	m.ExecutedCmd = cmd
	if m.ExecuteErr != nil {
		return m.ExecuteOutput, m.ExecuteErr
	}
	return m.ExecuteOutput, nil
}

// ExecuteInteractive records the call and returns configured error.
func (m *MockCommandExecutor) ExecuteInteractive(cmd *domain.ExecCommand) error {
	m.ExecuteInteractiveCalled = true
	m.ExecutedCmd = cmd
	return m.ExecuteInteractiveErr
}

// ExecuteWithContext records the call, writes output to writers, and returns configured error.
func (m *MockCommandExecutor) ExecuteWithContext(_ context.Context, cmd *domain.ExecCommand, stdout, stderr io.Writer) error {
	m.ExecuteWithContextCalled = true
	m.ExecutedCmd = cmd
	if m.ExecuteOutput != nil && stdout != nil {
		_, _ = stdout.Write(m.ExecuteOutput)
	}
	return m.ExecuteWithContextErr
}
