// Package app provides the dependency injection container for the application.
package app

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/infra/git"
	"github.com/runoshun/git-crew/v2/internal/infra/jsonstore"
	"github.com/runoshun/git-crew/v2/internal/infra/worktree"
	"github.com/runoshun/git-crew/v2/internal/usecase"
)

// Config holds the application configuration paths.
type Config struct {
	RepoRoot    string // Root directory of the git repository
	GitDir      string // Path to .git directory
	CrewDir     string // Path to .git/crew directory
	SocketPath  string // Path to tmux socket
	StorePath   string // Path to tasks.json
	WorktreeDir string // Path to worktrees directory
}

// newConfig creates a new Config from the git client.
func newConfig(gitClient *git.Client) Config {
	gitDir := gitClient.GitDir()
	crewDir := filepath.Join(gitDir, "crew")
	return Config{
		RepoRoot:    gitClient.RepoRoot(),
		GitDir:      gitDir,
		CrewDir:     crewDir,
		SocketPath:  filepath.Join(crewDir, "tmux.sock"),
		StorePath:   filepath.Join(crewDir, "tasks.json"),
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
	// Sessions  domain.SessionManager  // TODO: implement in later phase
	// GitHub    domain.GitHub          // TODO: implement in later phase
	// ConfigLoader domain.ConfigLoader // TODO: implement in later phase

	// Pointer fields
	Logger *slog.Logger

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

	// Create task repository
	store := jsonstore.New(cfg.StorePath)

	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create worktree manager
	worktreeClient := worktree.NewClient(cfg.RepoRoot, cfg.WorktreeDir)

	return &Container{
		Tasks:            store,
		StoreInitializer: store,
		Clock:            domain.RealClock{},
		Git:              gitClient,
		Worktrees:        worktreeClient,
		Logger:           logger,
		Config:           cfg,
	}, nil
}

// NewWithDeps creates a new Container with custom dependencies for testing.
func NewWithDeps(cfg Config, tasks domain.TaskRepository, storeInit domain.StoreInitializer, clock domain.Clock, logger *slog.Logger) *Container {
	return &Container{
		Tasks:            tasks,
		StoreInitializer: storeInit,
		Clock:            clock,
		Logger:           logger,
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
	return usecase.NewNewTask(c.Tasks, c.Clock)
}

// ListTasksUseCase returns a new ListTasks use case.
func (c *Container) ListTasksUseCase() *usecase.ListTasks {
	return usecase.NewListTasks(c.Tasks)
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
	return usecase.NewCopyTask(c.Tasks, c.Clock)
}

// AddCommentUseCase returns a new AddComment use case.
func (c *Container) AddCommentUseCase() *usecase.AddComment {
	return usecase.NewAddComment(c.Tasks, c.Clock)
}

// CloseTaskUseCase returns a new CloseTask use case.
func (c *Container) CloseTaskUseCase() *usecase.CloseTask {
	return usecase.NewCloseTask(c.Tasks)
}
