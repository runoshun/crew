# git-crew v2 Architecture

## Overview

This document describes the technical architecture for git-crew v2, including library choices, design patterns, and external dependencies.

---

## Technology Stack

### Language

- **Go 1.25+**
  - Generics for type-safe collections
  - `log/slog` for structured logging

### Core Libraries

| Library | Purpose | Rationale |
|---------|---------|-----------|
| [cobra](https://github.com/spf13/cobra) | CLI framework | De facto standard, subcommand support, shell completion |
| [bubbletea](https://github.com/charmbracelet/bubbletea) | TUI framework | Elm architecture, testable, composable |
| [lipgloss](https://github.com/charmbracelet/lipgloss) | TUI styling | Declarative styling, works with bubbletea |
| [bubbles](https://github.com/charmbracelet/bubbles) | TUI components | List, textarea, textinput components |
| [pelletier/go-toml/v2](https://github.com/pelletier/go-toml) | TOML parsing | Fast, spec-compliant, struct tags |
| [goccy/go-json](https://github.com/goccy/go-json) | JSON handling | Drop-in replacement, faster than stdlib |

### Testing

| Library | Purpose | Rationale |
|---------|---------|-----------|
| stdlib `testing` | Unit tests | No external dependency needed |
| [testify](https://github.com/stretchr/testify) | Assertions | Only `assert` and `require` packages |
| [go-cmp](https://github.com/google/go-cmp) | Deep comparison | Better diff output than reflect.DeepEqual |

### Development Tools

| Tool | Purpose |
|------|---------|
| golangci-lint | Linting (errcheck, staticcheck, exhaustive, gosum, etc.) |
| mise | Task runner, tool version management |
| govulncheck | Vulnerability scanning |

---

## Linting Configuration

### golangci-lint

```yaml
# .golangci.yml
run:
  timeout: 5m

linters:
  enable:
    # Default
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    # Formatting
    - gofmt
    - goimports
    # Code quality
    - misspell
    - unconvert
    - unparam
    - prealloc
    # Security
    - gosec
    # Sum types / Exhaustiveness
    - exhaustive      # enum switch exhaustiveness
    - gosum           # sealed interface exhaustiveness
    # Hygiene
    - nolintlint

linters-settings:
  errcheck:
    check-type-assertions: true
  govet:
    enable-all: true
  exhaustive:
    default-signifies-exhaustive: false
    check-generated: true
```

### Sum Type Patterns

Two patterns for exhaustive type checking:

#### 1. Enum with exhaustive (for Status, Type enums)

```go
type Status string

const (
    StatusTodo       Status = "todo"
    StatusInProgress Status = "in_progress"
    StatusInReview   Status = "in_review"
    StatusError      Status = "error"
    StatusDone       Status = "done"
    StatusClosed     Status = "closed"
)

func (s Status) Display() string {
    switch s {
    case StatusTodo:
        return "To Do"
    case StatusInProgress:
        return "In Progress"
    case StatusInReview:
        return "In Review"
    case StatusError:
        return "Error"
    case StatusDone:
        return "Done"
    case StatusClosed:
        return "Closed"
    }
    // exhaustive linter ensures all cases are handled
    panic("unreachable")
}
```

#### 2. Sealed Interface with gosum (for Messages, Events)

```go
//go-sumtype:decl Msg
type Msg interface {
    sealed()
}

type MsgTaskLoaded struct {
    Tasks []*Task
}

type MsgTaskUpdated struct {
    Task *Task
}

type MsgError struct {
    Err error
}

func (MsgTaskLoaded) sealed()  {}
func (MsgTaskUpdated) sealed() {}
func (MsgError) sealed()       {}

func handleMsg(m Msg) {
    switch m := m.(type) {
    case MsgTaskLoaded:
        // handle
    case MsgTaskUpdated:
        // handle
    case MsgError:
        // handle
    }
    // gosum linter ensures all cases are handled
}
```

#### When to Use Which

| Pattern | Linter | Use Case |
|---------|--------|----------|
| const enum | `exhaustive` | Status, Mode, simple enums |
| sealed interface | `gosum` | TUI messages, domain events, command results |

---

## Architecture Pattern

### Clean Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                           Presentation                              │
│  ┌─────────────────────────────┐  ┌──────────────────────────────┐  │
│  │         CLI (cobra)         │  │       TUI (bubbletea)        │  │
│  └──────────────┬──────────────┘  └───────────────┬──────────────┘  │
└─────────────────┼─────────────────────────────────┼─────────────────┘
                  │                                 │
                  ▼                                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                             App                                     │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                  Container (DI / Composition Root)          │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────┐
│                           UseCase                                   │
│  ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────────┐            │
│  │ StartTask │ │ StopTask  │ │ NewTask   │ │ MergeTask │  ...       │
│  └───────────┘ └───────────┘ └───────────┘ └───────────┘            │
│  (1 file = 1 usecase, directly uses Ports)                          │
└─────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────┐
│                            Domain                                   │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌───────────┐   │
│  │    Task     │  │   Status    │  │   Comment   │  │  Config   │   │
│  │   Entity    │  │  Lifecycle  │  │   Entity    │  │  Entity   │   │
│  └─────────────┘  └─────────────┘  └─────────────┘  └───────────┘   │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                      Port Interfaces                        │    │
│  │  TaskRepository  SessionManager  WorktreeManager  Git  ...  │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         Infrastructure                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐               │
│  │  JSONStore   │  │  TmuxClient  │  │  GitClient   │               │
│  └──────────────┘  └──────────────┘  └──────────────┘               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐               │
│  │WorktreeClient│  │ GitHubClient │  │ ConfigLoader │               │
│  └──────────────┘  └──────────────┘  └──────────────┘               │
└─────────────────────────────────────────────────────────────────────┘
```

### Layer Responsibilities

| Layer | Responsibility | Dependencies |
|-------|----------------|--------------|
| **Presentation** | User interaction (CLI/TUI), input parsing, output formatting | App (Container) |
| **App** | DI container, UseCase factory methods | UseCase, Infra |
| **UseCase** | Business logic orchestration (1 file = 1 use case) | Domain (Ports) |
| **Domain** | Entities, value objects, port interfaces, domain errors | None |
| **Infrastructure** | External system adapters (git, tmux, file system) | Domain (implements ports) |

### Dependency Rule

- Dependencies flow **inward only**
- Domain layer has **no external dependencies**
- UseCase depends on Port interfaces (not concrete implementations)
- Infrastructure implements Port interfaces
- Container binds Infrastructure to Ports

---

## Directory Structure

```
git-crew/
├── cmd/
│   └── crew/
│       └── main.go              # Entry point, creates Container
├── internal/
│   ├── domain/
│   │   ├── task.go              # Task, Comment entities
│   │   ├── status.go            # Status enum and transitions
│   │   ├── config.go            # Config, Agent entities
│   │   ├── errors.go            # Domain errors
│   │   ├── naming.go            # Naming conventions (branch, session, files)
│   │   └── ports.go             # All port interfaces
│   │
│   ├── usecase/
│   │   ├── new_task.go          # NewTask
│   │   ├── list_tasks.go        # ListTasks
│   │   ├── show_task.go         # ShowTask
│   │   ├── edit_task.go         # EditTask
│   │   ├── delete_task.go       # DeleteTask
│   │   ├── copy_task.go         # CopyTask
│   │   ├── start_task.go        # StartTask
│   │   ├── stop_task.go         # StopTask
│   │   ├── attach_session.go    # AttachSession
│   │   ├── peek_session.go      # PeekSession
│   │   ├── send_keys.go         # SendKeys
│   │   ├── complete_task.go     # CompleteTask
│   │   ├── review_task.go       # ReviewTask
│   │   ├── merge_task.go        # MergeTask
│   │   ├── close_task.go        # CloseTask
│   │   ├── prune_tasks.go       # PruneTasks
│   │   ├── import_issue.go      # ImportIssue
│   │   ├── create_pr.go         # CreatePR
│   │   └── session_ended.go     # SessionEnded (internal)
│   │
│   ├── infra/
│   │   ├── jsonstore/
│   │   │   └── store.go         # JSON file store (implements TaskRepository)
│   │   ├── tmux/
│   │   │   └── client.go        # tmux operations (implements SessionManager)
│   │   ├── git/
│   │   │   └── client.go        # git operations (implements Git)
│   │   ├── worktree/
│   │   │   └── client.go        # git worktree (implements WorktreeManager)
│   │   ├── github/
│   │   │   └── client.go        # gh CLI wrapper (implements GitHub)
│   │   └── config/
│   │       └── loader.go        # TOML config (implements ConfigLoader)
│   │
│   ├── app/
│   │   └── container.go         # DI container, UseCase factories
│   │
│   ├── cli/
│   │   ├── root.go              # Root command
│   │   ├── task.go              # new, list, show, edit, rm, cp
│   │   ├── session.go           # start, stop, attach, peek, send
│   │   ├── workflow.go          # complete, review, merge, close, prune
│   │   ├── github.go            # import, pr
│   │   ├── config.go            # config, init
│   │   └── internal.go          # _session-ended
│   │
│   └── tui/
│       ├── app.go               # Main bubbletea model
│       ├── update.go            # Message handlers
│       ├── view.go              # Rendering
│       ├── styles.go            # lipgloss styles
│       ├── keys.go              # Keybindings
│       └── components/
│           ├── tasklist.go      # Task list component
│           ├── dialog.go        # Confirmation dialog
│           ├── input.go         # Text input dialog
│           └── agent_picker.go  # Agent selection
│
├── go.mod
├── go.sum
└── mise.toml
```

---

## Domain Layer Design

### Entities

```go
// domain/task.go
package domain

type Task struct {
    ID          int
    ParentID    *int        // nil = root task, set = sub-task
    Title       string
    Description string
    Status      Status
    Created     time.Time
    Started     time.Time
    Agent       string      // Running agent name (empty if not running)
    Session     string      // tmux session name (empty if not running)
    Issue       int         // GitHub issue number (0 = not linked)
    PR          int         // GitHub PR number (0 = not created)
    BaseBranch  string      // Base branch for worktree creation
    Labels      []string
}

// IsRoot returns true if this is a root task (no parent)
func (t *Task) IsRoot() bool {
    return t.ParentID == nil
}

type Comment struct {
    Text string
    Time time.Time
}
```

### Status Lifecycle

```go
// domain/status.go
package domain

type Status string

const (
    StatusTodo       Status = "todo"
    StatusInProgress Status = "in_progress"
    StatusInReview   Status = "in_review"
    StatusError      Status = "error"
    StatusDone       Status = "done"
    StatusClosed     Status = "closed"
)

func (s Status) CanTransitionTo(target Status) bool {
    allowed := map[Status][]Status{
        StatusTodo:       {StatusInProgress, StatusClosed},
        StatusInProgress: {StatusInReview, StatusError, StatusClosed},
        StatusInReview:   {StatusInProgress, StatusDone, StatusClosed},
        StatusError:      {StatusInProgress, StatusClosed},
        StatusDone:       {StatusClosed},
        StatusClosed:     {},
    }
    for _, t := range allowed[s] {
        if t == target {
            return true
        }
    }
    return false
}
```

### Naming Conventions

```go
// domain/naming.go
package domain

func BranchName(taskID int, issue int) string {
    if issue > 0 {
        return fmt.Sprintf("crew-%d-gh-%d", taskID, issue)
    }
    return fmt.Sprintf("crew-%d", taskID)
}

func SessionName(taskID int) string {
    return fmt.Sprintf("crew-%d", taskID)
}

func ScriptPath(crewDir string, taskID int) string {
    return filepath.Join(crewDir, "scripts", fmt.Sprintf("task-%d.sh", taskID))
}

func PromptPath(crewDir string, taskID int) string {
    return filepath.Join(crewDir, "scripts", fmt.Sprintf("task-%d-prompt.txt", taskID))
}
```

### Domain Errors

```go
// domain/errors.go
package domain

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
)
```

### Port Interfaces

```go
// domain/ports.go
package domain

type TaskRepository interface {
    Get(id int) (*Task, error)
    List(filter TaskFilter) ([]*Task, error)
    GetChildren(parentID int) ([]*Task, error)
    Save(task *Task) error
    Delete(id int) error
    NextID() (int, error)
    GetComments(taskID int) ([]Comment, error)
    AddComment(taskID int, comment Comment) error
}

type TaskFilter struct {
    Labels   []string
    ParentID *int  // nil = all tasks, set = only children of this parent
}

type SessionManager interface {
    Start(ctx context.Context, opts StartSessionOptions) error
    Stop(sessionName string) error
    Attach(sessionName string) error
    Peek(sessionName string, lines int) (string, error)
    Send(sessionName string, keys string) error
    IsRunning(sessionName string) (bool, error)
}

type StartSessionOptions struct {
    Name      string
    Dir       string
    Command   string
    TaskID    int
}

type WorktreeManager interface {
    Create(branch, baseBranch string) (path string, err error)
    Resolve(branch string) (path string, err error)
    Remove(branch string) error
    Exists(branch string) (bool, error)
    List() ([]WorktreeInfo, error)
}

type WorktreeInfo struct {
    Path   string
    Branch string
}

type Git interface {
    CurrentBranch() (string, error)
    BranchExists(branch string) (bool, error)
    HasUncommittedChanges(dir string) (bool, error)
    HasMergeConflict(branch, target string) (bool, error)
    Merge(branch string, noFF bool) error
    DeleteBranch(branch string) error
}

type GitHub interface {
    GetIssue(number int) (*Issue, error)
    CreatePR(opts CreatePROptions) (int, error)
    UpdatePR(number int, opts UpdatePROptions) error
    FindPRByBranch(branch string) (int, error)
    Push(branch string) error
}

type Issue struct {
    Number int
    Title  string
    Body   string
    Labels []string
}

type CreatePROptions struct {
    Title  string
    Body   string
    Branch string
    Base   string
}

type UpdatePROptions struct {
    Title string
    Body  string
}

type ConfigLoader interface {
    Load() (*Config, error)
    LoadGlobal() (*Config, error)
}

type Clock interface {
    Now() time.Time
}
```

---

## UseCase Layer Design

Each use case is a single struct with an `Execute` method. It directly uses Port interfaces.

### UseCase Example: StartTask

```go
// usecase/start_task.go
package usecase

import (
    "context"
    "fmt"

    "github.com/user/git-crew/internal/domain"
)

type StartTaskInput struct {
    TaskID int
    Agent  string
}

type StartTaskOutput struct {
    SessionName  string
    WorktreePath string
}

type StartTask struct {
    tasks     domain.TaskRepository
    sessions  domain.SessionManager
    worktrees domain.WorktreeManager
    config    domain.ConfigLoader
    clock     domain.Clock
}

func NewStartTask(
    tasks domain.TaskRepository,
    sessions domain.SessionManager,
    worktrees domain.WorktreeManager,
    config domain.ConfigLoader,
    clock domain.Clock,
) *StartTask {
    return &StartTask{
        tasks:     tasks,
        sessions:  sessions,
        worktrees: worktrees,
        config:    config,
        clock:     clock,
    }
}

func (uc *StartTask) Execute(ctx context.Context, in StartTaskInput) (*StartTaskOutput, error) {
    // 1. Get task
    task, err := uc.tasks.Get(in.TaskID)
    if err != nil {
        return nil, fmt.Errorf("get task: %w", err)
    }
    if task == nil {
        return nil, fmt.Errorf("task %d: %w", in.TaskID, domain.ErrTaskNotFound)
    }

    // 2. Validate status transition
    if !task.Status.CanTransitionTo(domain.StatusInProgress) {
        return nil, fmt.Errorf("cannot start task in %s status: %w", task.Status, domain.ErrInvalidTransition)
    }

    // 3. Resolve agent
    agent := in.Agent
    if agent == "" {
        cfg, _ := uc.config.Load()
        if cfg != nil {
            agent = cfg.DefaultAgent
        }
    }
    if agent == "" {
        return nil, domain.ErrNoAgent
    }

    // 4. Create worktree
    branch := domain.BranchName(task.ID, task.Issue)
    wtPath, err := uc.worktrees.Create(branch, task.BaseBranch)
    if err != nil {
        return nil, fmt.Errorf("create worktree: %w", err)
    }

    // 5. Start session
    sessionName := domain.SessionName(task.ID)
    if err := uc.sessions.Start(ctx, domain.StartSessionOptions{
        Name:    sessionName,
        Dir:     wtPath,
        Command: agent, // simplified; actual impl builds full command
        TaskID:  task.ID,
    }); err != nil {
        // Rollback: remove worktree
        uc.worktrees.Remove(branch)
        return nil, fmt.Errorf("start session: %w", err)
    }

    // 6. Update task status
    task.Status = domain.StatusInProgress
    task.Agent = agent
    task.Session = sessionName
    task.Started = uc.clock.Now()
    if err := uc.tasks.Save(task); err != nil {
        // Rollback
        uc.sessions.Stop(sessionName)
        uc.worktrees.Remove(branch)
        return nil, fmt.Errorf("save task: %w", err)
    }

    return &StartTaskOutput{
        SessionName:  sessionName,
        WorktreePath: wtPath,
    }, nil
}
```

### UseCase Example: NewTask (simple)

```go
// usecase/new_task.go
package usecase

type NewTaskInput struct {
    Title       string
    Description string
    Issue       int
    Labels      []string
}

type NewTaskOutput struct {
    TaskID int
}

type NewTask struct {
    tasks domain.TaskRepository
    clock domain.Clock
}

func NewNewTask(tasks domain.TaskRepository, clock domain.Clock) *NewTask {
    return &NewTask{tasks: tasks, clock: clock}
}

func (uc *NewTask) Execute(ctx context.Context, in NewTaskInput) (*NewTaskOutput, error) {
    id, err := uc.tasks.NextID()
    if err != nil {
        return nil, fmt.Errorf("generate task ID: %w", err)
    }

    task := &domain.Task{
        ID:          id,
        Title:       in.Title,
        Description: in.Description,
        Status:      domain.StatusTodo,
        Created:     uc.clock.Now(),
        Issue:       in.Issue,
        Labels:      in.Labels,
        BaseBranch:  "main",
    }

    if err := uc.tasks.Save(task); err != nil {
        return nil, fmt.Errorf("save task: %w", err)
    }

    return &NewTaskOutput{TaskID: id}, nil
}
```

---

## App Layer Design (Container)

The Container handles all dependency injection and provides factory methods for UseCases.

```go
// internal/app/container.go
package app

import (
    "log/slog"
    "os"

    "github.com/user/git-crew/internal/domain"
    "github.com/user/git-crew/internal/infra/config"
    "github.com/user/git-crew/internal/infra/git"
    "github.com/user/git-crew/internal/infra/github"
    "github.com/user/git-crew/internal/infra/jsonstore"
    "github.com/user/git-crew/internal/infra/tmux"
    "github.com/user/git-crew/internal/infra/worktree"
    "github.com/user/git-crew/internal/usecase"
)

type Config struct {
    RepoRoot   string
    GitDir     string
    CrewDir    string
    SocketPath string
    StorePath  string
}

type Container struct {
    // Ports (interfaces bound to implementations)
    Tasks     domain.TaskRepository
    Sessions  domain.SessionManager
    Worktrees domain.WorktreeManager
    Git       domain.Git
    GitHub    domain.GitHub
    Config    domain.ConfigLoader
    Clock     domain.Clock
    Logger    *slog.Logger
}

func NewContainer(cfg Config) (*Container, error) {
    // Create infrastructure implementations
    store, err := jsonstore.New(cfg.StorePath)
    if err != nil {
        return nil, fmt.Errorf("init store: %w", err)
    }

    return &Container{
        Tasks:     store,
        Sessions:  tmux.NewClient(cfg.SocketPath, cfg.CrewDir),
        Worktrees: worktree.NewClient(cfg.RepoRoot),
        Git:       git.NewClient(cfg.RepoRoot),
        GitHub:    github.NewClient(),
        Config:    config.NewLoader(cfg.CrewDir),
        Clock:     &RealClock{},
        Logger:    slog.New(slog.NewTextHandler(os.Stderr, nil)),
    }, nil
}

// UseCase factory methods

func (c *Container) NewTask() *usecase.NewTask {
    return usecase.NewNewTask(c.Tasks, c.Clock)
}

func (c *Container) ListTasks() *usecase.ListTasks {
    return usecase.NewListTasks(c.Tasks)
}

func (c *Container) ShowTask() *usecase.ShowTask {
    return usecase.NewShowTask(c.Tasks)
}

func (c *Container) StartTask() *usecase.StartTask {
    return usecase.NewStartTask(
        c.Tasks,
        c.Sessions,
        c.Worktrees,
        c.Config,
        c.Clock,
    )
}

func (c *Container) StopTask() *usecase.StopTask {
    return usecase.NewStopTask(c.Tasks, c.Sessions)
}

func (c *Container) CompleteTask() *usecase.CompleteTask {
    return usecase.NewCompleteTask(c.Tasks, c.Git, c.Config)
}

func (c *Container) MergeTask() *usecase.MergeTask {
    return usecase.NewMergeTask(c.Tasks, c.Sessions, c.Worktrees, c.Git)
}

func (c *Container) ImportIssue() *usecase.ImportIssue {
    return usecase.NewImportIssue(c.Tasks, c.GitHub, c.Clock)
}

func (c *Container) CreatePR() *usecase.CreatePR {
    return usecase.NewCreatePR(c.Tasks, c.GitHub, c.Config)
}

// ... other UseCase factories

type RealClock struct{}

func (RealClock) Now() time.Time {
    return time.Now()
}
```

---

## Presentation Layer Design

### CLI with Cobra

CLI commands receive the Container and call UseCase factory methods.

```go
// cli/root.go
package cli

import (
    "github.com/spf13/cobra"
    "github.com/user/git-crew/internal/app"
)

func NewRootCommand(c *app.Container) *cobra.Command {
    root := &cobra.Command{
        Use:   "crew",
        Short: "AI agent task management",
    }

    root.AddCommand(
        newNewCommand(c),
        newListCommand(c),
        newShowCommand(c),
        newStartCommand(c),
        newStopCommand(c),
        newAttachCommand(c),
        newCompleteCommand(c),
        newMergeCommand(c),
        // ...
    )

    return root
}
```

```go
// cli/task.go
package cli

func newNewCommand(c *app.Container) *cobra.Command {
    var opts struct {
        Title  string
        Desc   string
        Issue  int
        Labels []string
    }

    cmd := &cobra.Command{
        Use:   "new",
        Short: "Create a new task",
        RunE: func(cmd *cobra.Command, args []string) error {
            out, err := c.NewTask().Execute(cmd.Context(), usecase.NewTaskInput{
                Title:       opts.Title,
                Description: opts.Desc,
                Issue:       opts.Issue,
                Labels:      opts.Labels,
            })
            if err != nil {
                return err
            }
            fmt.Fprintf(cmd.OutOrStdout(), "Created task #%d\n", out.TaskID)
            return nil
        },
    }

    cmd.Flags().StringVar(&opts.Title, "title", "", "Task title (required)")
    cmd.Flags().StringVar(&opts.Desc, "desc", "", "Task description")
    cmd.Flags().IntVar(&opts.Issue, "issue", 0, "Linked GitHub issue")
    cmd.Flags().StringArrayVar(&opts.Labels, "label", nil, "Labels")
    cmd.MarkFlagRequired("title")

    return cmd
}
```

```go
// cli/session.go
package cli

func newStartCommand(c *app.Container) *cobra.Command {
    return &cobra.Command{
        Use:   "start <id> [agent]",
        Short: "Start a task",
        Args:  cobra.RangeArgs(1, 2),
        RunE: func(cmd *cobra.Command, args []string) error {
            id, _ := strconv.Atoi(args[0])
            agent := ""
            if len(args) > 1 {
                agent = args[1]
            }

            out, err := c.StartTask().Execute(cmd.Context(), usecase.StartTaskInput{
                TaskID: id,
                Agent:  agent,
            })
            if err != nil {
                return err
            }

            fmt.Fprintf(cmd.OutOrStdout(), "Started task #%d (session: %s)\n", id, out.SessionName)
            return nil
        },
    }
}
```

### TUI with Bubbletea

TUI also receives the Container and calls UseCase factory methods.

```go
// tui/app.go
package tui

import (
    "context"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/user/git-crew/internal/app"
    "github.com/user/git-crew/internal/domain"
    "github.com/user/git-crew/internal/usecase"
)

type Model struct {
    container *app.Container

    // State
    mode   Mode
    tasks  []*domain.Task
    cursor int
    err    error

    // Components
    // ...
}

func New(c *app.Container) *Model {
    return &Model{
        container: c,
        mode:      ModeNormal,
    }
}

func (m *Model) Init() tea.Cmd {
    return m.loadTasks
}

func (m *Model) loadTasks() tea.Msg {
    out, err := m.container.ListTasks().Execute(context.Background(), usecase.ListTasksInput{})
    if err != nil {
        return errMsg{err}
    }
    return tasksLoadedMsg{out.Tasks}
}

func (m *Model) startTask(id int, agent string) tea.Cmd {
    return func() tea.Msg {
        out, err := m.container.StartTask().Execute(
            context.Background(),
            usecase.StartTaskInput{TaskID: id, Agent: agent},
        )
        if err != nil {
            return errMsg{err}
        }
        return taskStartedMsg{out}
    }
}
```

### Entry Point

```go
// cmd/crew/main.go
package main

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/user/git-crew/internal/app"
    "github.com/user/git-crew/internal/cli"
)

func main() {
    repoRoot, gitDir, err := findGitRoot()
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }

    crewDir := filepath.Join(gitDir, "crew")

    container, err := app.NewContainer(app.Config{
        RepoRoot:   repoRoot,
        GitDir:     gitDir,
        CrewDir:    crewDir,
        SocketPath: filepath.Join(crewDir, "tmux.sock"),
        StorePath:  filepath.Join(crewDir, "tasks.json"),
    })
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }

    rootCmd := cli.NewRootCommand(container)
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}

func findGitRoot() (repoRoot, gitDir string, err error) {
    // Implementation: walk up to find .git
    // ...
}
```

---

## Infrastructure Layer Design

### JSON Store

```go
// infra/jsonstore/store.go
package jsonstore

type Store struct {
    path     string
    lockPath string
}

func New(path string) (*Store, error) {
    return &Store{
        path:     path,
        lockPath: path + ".lock",
    }, nil
}

func (s *Store) Get(id int) (*domain.Task, error) {
    // ...
}

func (s *Store) Save(task *domain.Task) error {
    // ...
}

func (s *Store) withLock(fn func() error) error {
    lock, err := os.OpenFile(s.lockPath, os.O_CREATE|os.O_RDWR, 0644)
    if err != nil {
        return fmt.Errorf("open lock file: %w", err)
    }
    defer lock.Close()

    if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
        return fmt.Errorf("acquire lock: %w", err)
    }
    defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)

    return fn()
}
```

### Worktree Client

```go
// infra/worktree/client.go
package worktree

type Client struct {
    repoRoot     string
    worktreeBase string
}

func NewClient(repoRoot string) *Client {
    base := filepath.Join(filepath.Dir(repoRoot), filepath.Base(repoRoot)+"-worktrees")
    return &Client{
        repoRoot:     repoRoot,
        worktreeBase: base,
    }
}

func (c *Client) Create(branch, baseBranch string) (string, error) {
    wtPath := filepath.Join(c.worktreeBase, branch)

    if _, err := os.Stat(wtPath); err == nil {
        return wtPath, nil // Already exists
    }

    // git worktree add -b <branch> <path> <base>
    cmd := exec.Command("git", "worktree", "add", "-b", branch, wtPath, baseBranch)
    cmd.Dir = c.repoRoot
    if out, err := cmd.CombinedOutput(); err != nil {
        return "", fmt.Errorf("git worktree add: %s: %w", out, err)
    }

    return wtPath, nil
}

func (c *Client) Remove(branch string) error {
    wtPath := filepath.Join(c.worktreeBase, branch)

    cmd := exec.Command("git", "worktree", "remove", wtPath)
    cmd.Dir = c.repoRoot
    if out, err := cmd.CombinedOutput(); err != nil {
        return fmt.Errorf("git worktree remove: %s: %w", out, err)
    }

    return nil
}
```

---

## Testing Strategy

### Unit Tests (Domain + UseCase)

```go
func TestStatus_CanTransitionTo(t *testing.T) {
    tests := []struct {
        from   domain.Status
        to     domain.Status
        expect bool
    }{
        {domain.StatusTodo, domain.StatusInProgress, true},
        {domain.StatusTodo, domain.StatusDone, false},
        {domain.StatusInProgress, domain.StatusInReview, true},
    }

    for _, tt := range tests {
        t.Run(fmt.Sprintf("%s->%s", tt.from, tt.to), func(t *testing.T) {
            got := tt.from.CanTransitionTo(tt.to)
            assert.Equal(t, tt.expect, got)
        })
    }
}
```

### UseCase Tests with Mocks

```go
type MockTaskRepository struct {
    tasks map[int]*domain.Task
}

func (m *MockTaskRepository) Get(id int) (*domain.Task, error) {
    return m.tasks[id], nil
}

func (m *MockTaskRepository) Save(task *domain.Task) error {
    m.tasks[task.ID] = task
    return nil
}

// ...

func TestStartTask_Execute(t *testing.T) {
    repo := &MockTaskRepository{
        tasks: map[int]*domain.Task{
            1: {ID: 1, Status: domain.StatusTodo, BaseBranch: "main"},
        },
    }
    sessions := &MockSessionManager{}
    worktrees := &MockWorktreeManager{paths: map[string]string{}}
    config := &MockConfigLoader{cfg: &domain.Config{DefaultAgent: "claude"}}
    clock := &MockClock{now: time.Now()}

    uc := usecase.NewStartTask(repo, sessions, worktrees, config, clock)

    out, err := uc.Execute(context.Background(), usecase.StartTaskInput{
        TaskID: 1,
    })

    require.NoError(t, err)
    assert.Equal(t, "crew-1", out.SessionName)
    assert.Equal(t, domain.StatusInProgress, repo.tasks[1].Status)
    assert.True(t, sessions.StartCalled)
}
```

### End-to-End Tests

```go
func TestCLI_NewAndStart(t *testing.T) {
    dir := setupTestRepo(t)
    defer os.RemoveAll(dir)

    runCrew(t, dir, "init")

    out := runCrew(t, dir, "new", "--title", "Test task")
    assert.Contains(t, out, "Created task #1")

    out = runCrew(t, dir, "list")
    assert.Contains(t, out, "Test task")
    assert.Contains(t, out, "todo")
}
```

---

## External Dependencies

### Required Tools

| Tool | Minimum Version | Purpose | Check Command |
|------|-----------------|---------|---------------|
| git | 2.20+ | Version control, worktree | `git --version` |
| tmux | 3.0+ | Session management | `tmux -V` |

### Optional Tools

| Tool | Purpose | Check Command |
|------|---------|---------------|
| gh | GitHub CLI | `gh --version` |
| delta | Diff viewer | `delta --version` |

---

## Configuration

### File Format (TOML)

Chosen over YAML/JSON for:
- Human-readable and writable
- Clear section structure
- Native multiline strings
- Widely used in Go ecosystem

### Loading Priority

1. `.git/crew/config.toml` (repository-specific)
2. `~/.config/git-crew/config.toml` (user defaults)
3. Built-in defaults

Settings are merged with later files overriding earlier ones.

---

## Logging

### Structured Logging with slog

```go
logger := slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))

logger.Info("task started",
    "task_id", task.ID,
    "agent", agent,
    "session", session,
)

logger.Error("session failed",
    "task_id", task.ID,
    "error", err,
    "exit_code", exitCode,
)
```

### Log Files

| File | Content |
|------|---------|
| `.git/crew/logs/crew.log` | Global operations |
| `.git/crew/logs/task-N.log` | Per-task operations |

---

## GitHub Actions CI

```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [main]
  pull_request:

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: jdx/mise-action@v2
      - run: mise run lint

  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: jdx/mise-action@v2
      - run: mise run test:cover
      - uses: actions/upload-artifact@v4
        with:
          name: coverage
          path: coverage.out
          retention-days: 7

  vuln:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: jdx/mise-action@v2
      - run: mise run vuln
```

---

## Summary

| Layer | Location | Responsibility |
|-------|----------|----------------|
| **Domain** | `internal/domain/` | Entities, Status, Ports, Errors, Naming |
| **UseCase** | `internal/usecase/` | 1 file = 1 use case, orchestration logic |
| **Infra** | `internal/infra/` | Port implementations (JSON, tmux, git, etc.) |
| **App** | `internal/app/` | Container (DI), UseCase factories |
| **CLI** | `internal/cli/` | Cobra commands, thin layer calling UseCases |
| **TUI** | `internal/tui/` | Bubbletea app, thin layer calling UseCases |

**Key Design Decisions**:

1. **No Service layer**: UseCases directly use Ports
2. **1 file = 1 UseCase**: Clear, focused, testable
3. **Container for DI**: All wiring in one place, UseCase factories
4. **Thin Presentation**: CLI/TUI only handle I/O, delegate to UseCases
