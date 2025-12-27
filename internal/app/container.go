// Package app provides the dependency injection container for the application.
package app

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/infra/jsonstore"
)

// Config holds the application configuration paths.
type Config struct {
	RepoRoot   string // Root directory of the git repository
	GitDir     string // Path to .git directory
	CrewDir    string // Path to .git/crew directory
	SocketPath string // Path to tmux socket
	StorePath  string // Path to tasks.json
}

// NewConfig creates a new Config from the given git directory.
func NewConfig(repoRoot, gitDir string) Config {
	crewDir := filepath.Join(gitDir, "crew")
	return Config{
		RepoRoot:   repoRoot,
		GitDir:     gitDir,
		CrewDir:    crewDir,
		SocketPath: filepath.Join(crewDir, "tmux.sock"),
		StorePath:  filepath.Join(crewDir, "tasks.json"),
	}
}

// Container provides dependency injection for the application.
// It holds all port implementations and provides factory methods for use cases.
// Fields are ordered to minimize memory padding.
type Container struct {
	// Ports (interfaces bound to implementations)
	// Interfaces are 16 bytes (2 words), so group them together.
	Tasks domain.TaskRepository
	Clock domain.Clock
	// Sessions  domain.SessionManager  // TODO: implement in later phase
	// Worktrees domain.WorktreeManager // TODO: implement in later phase
	// Git       domain.Git             // TODO: implement in later phase
	// GitHub    domain.GitHub          // TODO: implement in later phase
	// ConfigLoader domain.ConfigLoader // TODO: implement in later phase

	// Pointer fields (8 bytes each)
	Logger *slog.Logger

	// Configuration (value type, placed last)
	Config Config
}

// New creates a new Container with the given configuration.
func New(cfg Config) (*Container, error) {
	// Create task repository
	store := jsonstore.New(cfg.StorePath)

	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	return &Container{
		Tasks:  store,
		Clock:  domain.RealClock{},
		Logger: logger,
		Config: cfg,
	}, nil
}

// NewWithDeps creates a new Container with custom dependencies for testing.
func NewWithDeps(cfg Config, tasks domain.TaskRepository, clock domain.Clock, logger *slog.Logger) *Container {
	return &Container{
		Tasks:  tasks,
		Clock:  clock,
		Logger: logger,
		Config: cfg,
	}
}

// UseCase factory methods
// These will be implemented as use cases are added.

// NewTaskUseCase returns a new NewTask use case.
// func (c *Container) NewTaskUseCase() *usecase.NewTask {
//     return usecase.NewNewTask(c.Tasks, c.Clock)
// }

// ListTasksUseCase returns a new ListTasks use case.
// func (c *Container) ListTasksUseCase() *usecase.ListTasks {
//     return usecase.NewListTasks(c.Tasks)
// }

// ShowTaskUseCase returns a new ShowTask use case.
// func (c *Container) ShowTaskUseCase() *usecase.ShowTask {
//     return usecase.NewShowTask(c.Tasks)
// }
