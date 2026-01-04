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

See `.golangci.yml` for linter configuration.

### Sum Type Patterns

- Use `exhaustive` linter with const enums for Status, Mode, etc.
- Use `gosum` linter with sealed interfaces for TUI messages, domain events

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
│  │   GitStore   │  │  TmuxClient  │  │  GitClient   │               │
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
cmd/crew/         # Entry point
internal/
  domain/         # Entities, ports, errors, naming
  usecase/        # 1 file = 1 use case
  infra/          # Port implementations (git, tmux, config, etc.)
  app/            # DI container
  cli/            # Cobra commands
  tui/            # Bubbletea TUI
```

---

## Layer Design

### Domain Layer

- **Entities**: `Task`, `Comment`, `Status`
- **Ports**: Interface definitions for external dependencies
- **Errors**: Domain-specific sentinel errors
- **Naming**: Branch/session/file naming conventions

```go
// Ports define what the domain needs (domain/ports.go)
type TaskRepository interface {
    Get(id int) (*Task, error)
    Save(task *Task) error
    // ...
}

// Sentinel errors for domain rules (domain/errors.go)
var ErrTaskNotFound = errors.New("task not found")
```

### UseCase Layer

- 1 file = 1 use case
- `Input` → `Execute(ctx, input)` → `Output`
- Uses Ports, handles rollback on failure

```go
type StartTaskInput struct { ... }

type StartTaskOutput struct { ... }

type StartTask struct {
    tasks     domain.TaskRepository
    sessions  domain.SessionManager
    // ...
}

func (uc *StartTask) Execute(ctx context.Context, in StartTaskInput) (*StartTaskOutput, error) {
    // 1. Validate → 2. Do work → 3. Rollback on error
}
```

### App Layer (Container)

- DI container binding Ports to Infrastructure
- Factory methods for UseCases

```go
func (c *Container) StartTask() *usecase.StartTask {
    return usecase.NewStartTask(c.Tasks, c.Sessions, c.Worktrees, c.Config, c.Clock)
}
```

### Presentation Layer

- CLI (Cobra) / TUI (Bubbletea): thin layer calling UseCases

```go
out, err := c.StartTask().Execute(ctx, usecase.StartTaskInput{TaskID: id})
```

### Infrastructure Layer

- Port implementations: `gitstore/`, `tmux/`, `git/`, `worktree/`, `config/`
- Use `panic("not implemented")` for interface methods not yet needed

---

## Testing Strategy

### Unit Tests (Domain + UseCase)

- Table-driven tests
- Mock Port interfaces

```go
type MockTaskRepository struct {
    tasks map[int]*domain.Task
}

uc := usecase.NewStartTask(mockRepo, mockSessions, ...)
out, err := uc.Execute(ctx, input)
```

### Integration Tests (Infrastructure)

- Use `t.TempDir()` for isolated environments
- Test with real git/tmux commands

```go
func TestGitClient_Merge(t *testing.T) {
    dir := t.TempDir()
    exec.Command("git", "init", dir).Run()
    client := git.NewClient(dir)
    // test with real git commands
}
```

### End-to-End Tests (CLI)

- Execute actual CLI commands
- Verify output and side effects

```go
out := runCrew(t, dir, "new", "--title", "Test task")
assert.Contains(t, out, "Created task #1")
```


