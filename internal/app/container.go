// Package app provides the dependency injection container for the application.
package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/runoshun/git-crew/v2/internal/domain"
	acpinfra "github.com/runoshun/git-crew/v2/internal/infra/acp"
	"github.com/runoshun/git-crew/v2/internal/infra/config"
	"github.com/runoshun/git-crew/v2/internal/infra/executor"
	"github.com/runoshun/git-crew/v2/internal/infra/filestore"
	"github.com/runoshun/git-crew/v2/internal/infra/git"
	"github.com/runoshun/git-crew/v2/internal/infra/gitstore"
	"github.com/runoshun/git-crew/v2/internal/infra/jsonstore"
	"github.com/runoshun/git-crew/v2/internal/infra/logging"
	"github.com/runoshun/git-crew/v2/internal/infra/runner"
	"github.com/runoshun/git-crew/v2/internal/infra/tmux"
	"github.com/runoshun/git-crew/v2/internal/infra/worktree"
	"github.com/runoshun/git-crew/v2/internal/usecase"
)

// Config holds the application configuration paths.
type Config struct {
	RepoRoot    string // Root directory of the git repository
	GitDir      string // Path to .git directory
	CrewDir     string // Path to .crew directory
	SocketPath  string // Path to tmux socket
	StorePath   string // Path to tasks directory
	WorktreeDir string // Path to worktrees directory
}

// newConfig creates a new Config from the git client.
func newConfig(gitClient *git.Client) Config {
	repoRoot := gitClient.RepoRoot()
	crewDir := domain.RepoCrewDir(repoRoot)
	return Config{
		RepoRoot:    repoRoot,
		GitDir:      gitClient.GitDir(),
		CrewDir:     crewDir,
		SocketPath:  filepath.Join(crewDir, "tmux.sock"),
		StorePath:   filepath.Join(crewDir, "tasks"),
		WorktreeDir: filepath.Join(crewDir, "worktrees"),
	}
}

// Container provides dependency injection for the application.
// It holds all port implementations and provides factory methods for use cases.
type Container struct {
	// Ports (interfaces bound to implementations)
	Tasks            domain.TaskRepository
	StoreInitializer domain.StoreInitializer
	Clock            domain.Clock
	Git              domain.Git
	Worktrees        domain.WorktreeManager
	Sessions         domain.SessionManager
	ConfigLoader     domain.ConfigLoader
	ConfigManager    domain.ConfigManager
	Logger           domain.Logger
	Runner           domain.ScriptRunner
	Executor         domain.CommandExecutor
	ACPIPCFactory    domain.ACPIPCFactory
	// GitHub    domain.GitHub          // TODO: implement in later phase

	// Configuration
	Config Config
}

// New creates a new Container by detecting the git repository from the given directory.
func New(dir string) (*Container, error) {
	// Detect git repository
	gitClient, err := git.NewClient(dir)
	if err != nil {
		return nil, err
	}

	// Create configuration from git client
	cfg := newConfig(gitClient)

	// Load app config to determine namespace and defaults
	configLoader := config.NewLoader(cfg.CrewDir, cfg.RepoRoot)
	appConfig, err := configLoader.Load()
	if err != nil {
		// Warn about config error and use defaults
		fmt.Fprintf(os.Stderr, "warning: config error: %v (using defaults)\n", err)
		appConfig = domain.NewDefaultConfig()
	}

	// Create task repository (file store)
	namespace := resolveNamespace(appConfig, gitClient)
	fileStore := filestore.New(cfg.CrewDir, namespace)
	var taskRepo domain.TaskRepository = fileStore
	var storeInit domain.StoreInitializer = fileStore

	// Create logger
	logLevel := logging.ParseLevel(appConfig.Log.Level)
	logger := logging.New(cfg.CrewDir, logLevel)

	// Create worktree manager
	worktreeClient := worktree.NewClient(cfg.RepoRoot, cfg.WorktreeDir)

	// Create session manager
	sessionClient := tmux.NewClient(cfg.SocketPath, cfg.CrewDir)

	// Create config manager
	configManager := config.NewManager(cfg.CrewDir, cfg.RepoRoot)

	// Create script runner
	scriptRunner := runner.NewClient()

	// Create command executor
	commandExecutor := executor.NewClient()

	// Create ACP IPC factory
	acpIPCFactory := acpinfra.NewFileIPCFactory(cfg.CrewDir, logger)

	return &Container{
		Tasks:            taskRepo,
		StoreInitializer: storeInit,
		Clock:            domain.RealClock{},
		Git:              gitClient,
		Worktrees:        worktreeClient,
		Sessions:         sessionClient,
		ConfigLoader:     configLoader,
		ConfigManager:    configManager,
		Logger:           logger,
		Runner:           scriptRunner,
		Executor:         commandExecutor,
		ACPIPCFactory:    acpIPCFactory,
		Config:           cfg,
	}, nil
}

// NewWithDeps creates a new Container with custom dependencies for testing.
func NewWithDeps(cfg Config, tasks domain.TaskRepository, storeInit domain.StoreInitializer, clock domain.Clock, logger domain.Logger, executor domain.CommandExecutor) *Container {
	return &Container{
		Tasks:            tasks,
		StoreInitializer: storeInit,
		Clock:            clock,
		Logger:           logger,
		Executor:         executor,
		Config:           cfg,
	}
}

// UseCase factory methods

// InitRepoUseCase returns a new InitRepo use case.
func (c *Container) InitRepoUseCase() *usecase.InitRepo {
	return usecase.NewInitRepo(c.StoreInitializer)
}

// NewTaskUseCase returns a new NewTask use case.
func (c *Container) NewTaskUseCase() *usecase.NewTask {
	return usecase.NewNewTask(c.Tasks, c.Git, c.ConfigLoader, c.Clock, c.Logger)
}

// CreateTasksFromFileUseCase returns a new CreateTasksFromFile use case.
func (c *Container) CreateTasksFromFileUseCase() *usecase.CreateTasksFromFile {
	return usecase.NewCreateTasksFromFile(c.Tasks, c.Git, c.ConfigLoader, c.Clock, c.Logger)
}

// ListTasksUseCase returns a new ListTasks use case.
func (c *Container) ListTasksUseCase() *usecase.ListTasks {
	return usecase.NewListTasks(c.Tasks, c.Sessions)
}

// ShowTaskUseCase returns a new ShowTask use case.
func (c *Container) ShowTaskUseCase() *usecase.ShowTask {
	return usecase.NewShowTask(c.Tasks)
}

// EditTaskUseCase returns a new EditTask use case.
func (c *Container) EditTaskUseCase() *usecase.EditTask {
	return usecase.NewEditTask(c.Tasks)
}

// DeleteTaskUseCase returns a new DeleteTask use case.
func (c *Container) DeleteTaskUseCase() *usecase.DeleteTask {
	return usecase.NewDeleteTask(c.Tasks)
}

// CopyTaskUseCase returns a new CopyTask use case.
func (c *Container) CopyTaskUseCase() *usecase.CopyTask {
	return usecase.NewCopyTask(c.Tasks, c.Clock, c.Worktrees, c.Git)
}

// AddCommentUseCase returns a new AddComment use case with session starter.
func (c *Container) AddCommentUseCase() *usecase.AddComment {
	uc := usecase.NewAddComment(c.Tasks, c.Sessions, c.Clock)
	// Set up session starter to auto-start sessions when needed
	startTask := c.StartTaskUseCase()
	adapter := usecase.NewStartTaskAdapter(startTask)
	return uc.WithSessionStarter(adapter)
}

// EditCommentUseCase returns a new EditComment use case.
func (c *Container) EditCommentUseCase() *usecase.EditComment {
	return usecase.NewEditComment(c.Tasks, c.Clock)
}

// CloseTaskUseCase returns a new CloseTask use case.
func (c *Container) CloseTaskUseCase() *usecase.CloseTask {
	return usecase.NewCloseTask(c.Tasks, c.Sessions, c.Worktrees)
}

// StartTaskUseCase returns a new StartTask use case.
func (c *Container) StartTaskUseCase() *usecase.StartTask {
	return usecase.NewStartTask(c.Tasks, c.Sessions, c.Worktrees, c.ConfigLoader, c.Git, c.Clock, c.Logger, c.Runner, c.Config.CrewDir, c.Config.RepoRoot)
}

// AttachSessionUseCase returns a new AttachSession use case.
func (c *Container) AttachSessionUseCase() *usecase.AttachSession {
	return usecase.NewAttachSession(c.Tasks, c.Sessions)
}

// SendKeysUseCase returns a new SendKeys use case.
func (c *Container) SendKeysUseCase() *usecase.SendKeys {
	return usecase.NewSendKeys(c.Tasks, c.Sessions)
}

// PeekSessionUseCase returns a new PeekSession use case.
func (c *Container) PeekSessionUseCase() *usecase.PeekSession {
	return usecase.NewPeekSession(c.Tasks, c.Sessions)
}

// SessionEndedUseCase returns a new SessionEnded use case.
func (c *Container) SessionEndedUseCase() *usecase.SessionEnded {
	return usecase.NewSessionEnded(c.Tasks, c.Config.CrewDir)
}

// ShowConfigUseCase returns a new ShowConfig use case.
func (c *Container) ShowConfigUseCase() *usecase.ShowConfig {
	return usecase.NewShowConfig(c.ConfigManager, c.ConfigLoader)
}

// InitConfigUseCase returns a new InitConfig use case.
func (c *Container) InitConfigUseCase() *usecase.InitConfig {
	return usecase.NewInitConfig(c.ConfigManager)
}

// ShowConfigTemplateUseCase returns a new ShowConfigTemplate use case.
func (c *Container) ShowConfigTemplateUseCase() *usecase.ShowConfigTemplate {
	return usecase.NewShowConfigTemplate()
}

// CompleteTaskUseCase returns a new CompleteTask use case.
// stdout and stderr are used for review output when auto-starting review.
func (c *Container) CompleteTaskUseCase(stdout, stderr io.Writer) *usecase.CompleteTask {
	return usecase.NewCompleteTask(c.Tasks, c.Sessions, c.Worktrees, c.Git, c.ConfigLoader, c.Clock, c.Logger, c.Executor, stderr, c.Config.CrewDir, c.Config.RepoRoot)
}

// MergeTaskUseCase returns a new MergeTask use case.
func (c *Container) MergeTaskUseCase() *usecase.MergeTask {
	return usecase.NewMergeTask(c.Tasks, c.Sessions, c.Worktrees, c.Git, c.Clock, c.Config.CrewDir)
}

// ShowDiffUseCase returns a new ShowDiff use case.
// stdout and stderr are the writers for command output.
func (c *Container) ShowDiffUseCase(stdout, stderr io.Writer) *usecase.ShowDiff {
	return usecase.NewShowDiff(c.Tasks, c.Worktrees, c.Git, c.ConfigLoader, c.Executor, stdout, stderr)
}

// ShowDiffUseCaseForCommand returns a new ShowDiff use case for GetCommand() only.
// This is used by TUI which executes the command via tea.Exec.
func (c *Container) ShowDiffUseCaseForCommand() *usecase.ShowDiff {
	return usecase.NewShowDiff(c.Tasks, c.Worktrees, c.Git, c.ConfigLoader, c.Executor, nil, nil)
}

// StopTaskUseCase returns a new StopTask use case.
func (c *Container) StopTaskUseCase() *usecase.StopTask {
	return usecase.NewStopTask(c.Tasks, c.Sessions, c.Config.CrewDir)
}

// PruneTasksUseCase returns a new PruneTasks use case.
func (c *Container) PruneTasksUseCase() *usecase.PruneTasks {
	return usecase.NewPruneTasks(c.Tasks, c.Worktrees, c.Git)
}

// ExecCommandUseCase returns a new ExecCommand use case.
func (c *Container) ExecCommandUseCase() *usecase.ExecCommand {
	return usecase.NewExecCommand(c.Tasks, c.Worktrees)
}

// ACPControlUseCase returns a new ACPControl use case.
func (c *Container) ACPControlUseCase() *usecase.ACPControl {
	return usecase.NewACPControl(
		c.Tasks,
		c.ConfigLoader,
		c.Git,
		c.ACPIPCFactory,
	)
}

// ACPRunUseCase returns a new ACPRun use case.
func (c *Container) ACPRunUseCase(stdout, stderr io.Writer) *usecase.ACPRun {
	return usecase.NewACPRun(
		c.Tasks,
		c.Worktrees,
		c.ConfigLoader,
		c.Git,
		c.Runner,
		c.ACPIPCFactory,
		c.Config.RepoRoot,
		stdout,
		stderr,
	)
}

// StartManagerUseCase returns a new StartManager use case.
func (c *Container) StartManagerUseCase() *usecase.StartManager {
	return usecase.NewStartManager(c.Sessions, c.ConfigLoader, c.Config.RepoRoot, c.Config.GitDir, c.Config.CrewDir)
}

// ReviewTaskUseCase returns a new ReviewTask use case.
// stderr is the writer for warning messages.
func (c *Container) ReviewTaskUseCase(stderr io.Writer) *usecase.ReviewTask {
	return usecase.NewReviewTask(c.Tasks, c.Worktrees, c.ConfigLoader, c.Executor, c.Clock, c.Config.RepoRoot, stderr)
}

// PollTaskUseCase returns a new PollTask use case.
// stdout and stderr are the writers for command output.
func (c *Container) PollTaskUseCase(stdout, stderr io.Writer) *usecase.PollTask {
	return usecase.NewPollTask(c.Tasks, c.Clock, c.Executor, stdout, stderr)
}

// PollStatusUseCase returns a new PollStatus use case.
// stdout is the writer for output.
func (c *Container) PollStatusUseCase(stdout io.Writer) *usecase.PollStatus {
	return usecase.NewPollStatus(c.Tasks, stdout)
}

// ShowLogsUseCase returns a new ShowLogs use case.
func (c *Container) ShowLogsUseCase() *usecase.ShowLogs {
	return usecase.NewShowLogs(c.Tasks, c.Config.CrewDir)
}

// MigrateStoreUseCase returns a new MigrateStore use case.
func (c *Container) MigrateStoreUseCase(source domain.TaskRepository, dest domain.TaskRepository, destInit domain.StoreInitializer) *usecase.MigrateStore {
	return usecase.NewMigrateStore(source, dest, destInit)
}

// FileStore returns a file-based task store for a namespace.
func (c *Container) FileStore(namespace string) (domain.TaskRepository, domain.StoreInitializer) {
	store := filestore.New(c.Config.CrewDir, namespace)
	return store, store
}

// GitStore returns a git ref-based task store for a namespace.
func (c *Container) GitStore(namespace string) (domain.TaskRepository, domain.StoreInitializer, error) {
	store, err := gitstore.New(c.Config.RepoRoot, namespace)
	if err != nil {
		return nil, nil, err
	}
	return store, store, nil
}

// JSONStore returns a JSON file-based task store for a path.
func (c *Container) JSONStore(path string) (domain.TaskRepository, domain.StoreInitializer) {
	store := jsonstore.New(path)
	return store, store
}

func resolveNamespace(cfg *domain.Config, gitClient domain.Git) string {
	if cfg != nil && cfg.Tasks.Namespace != "" {
		namespace := domain.SanitizeNamespace(cfg.Tasks.Namespace)
		if namespace != "" {
			return namespace
		}
	}
	if gitClient != nil {
		email, err := gitClient.UserEmail()
		if err == nil {
			namespace := domain.NamespaceFromEmail(email)
			if namespace != "" {
				return namespace
			}
		}
	}
	return "default"
}
