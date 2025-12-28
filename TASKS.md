# git-crew v2 Implementation Tasks

## Overview

Bootstrap strategy: Build crew with crew.

1. **Phase 1**: Foundation (environment, CI, project structure)
2. **Phase 2**: Task Management (create, list, show, edit, delete)
3. **Phase 3**: Migrate existing tasks to crew, continue development with crew

---

## Phase 1: Foundation

### 1.1 Project Setup

- [x] Create `v2/` directory structure
  ```
  v2/
  ├── cmd/crew/main.go
  ├── internal/
  │   ├── domain/
  │   ├── usecase/
  │   ├── infra/
  │   ├── app/
  │   └── cli/
  ├── go.mod
  └── go.sum
  ```
- [x] Initialize Go module (`github.com/runoshun/git-crew/v2`)
- [x] Add dependencies to go.mod
  - cobra
  - pelletier/go-toml/v2
  - goccy/go-json

### 1.2 CI Setup

- [x] Create `.golangci.yml` with exhaustive, gochecksumtype linters
- [x] Create `mise.toml` with build, test, lint, ci tasks
- [x] Create `.github/workflows/ci.yml` for CI
  - lint, test, vuln jobs

### 1.3 Domain Layer

- [x] `internal/domain/task.go` - Task, Comment entities
- [x] `internal/domain/status.go` - Status enum, CanTransitionTo
- [x] `internal/domain/errors.go` - Domain errors
- [x] `internal/domain/naming.go` - BranchName, SessionName functions
- [x] `internal/domain/ports.go` - TaskRepository interface
- [x] Unit tests for Status transitions

### 1.4 Infrastructure: JSON Store

- [x] `internal/infra/jsonstore/store.go` - TaskRepository implementation
- [x] File locking (flock)
- [x] Unit tests with temp files

### 1.5 App Container

- [x] `internal/app/container.go` - Container struct, Config
- [x] Clock interface and RealClock
- [x] UseCase factory methods (stubs for now)

### 1.6 CLI Skeleton

- [x] `internal/cli/root.go` - Root command with version
- [x] `cmd/crew/main.go` - Entry point, find git root, create container
- [x] `git crew --version` working

### 1.7 Init Command

- [x] `internal/usecase/init_repo.go` - InitRepo usecase
- [x] `internal/cli/init.go` - init command
- [x] Create `.git/crew/` directory
- [x] Create empty `tasks.json`
- [x] Integration test

**Phase 1 Milestone**: `git crew init` and `git crew --version` work, CI passes.

---

## Phase 2: Task Management

### 2.1 New Task

- [x] `internal/usecase/new_task.go` - NewTask usecase
- [x] `internal/cli/task.go` - new command
- [x] Flags: `--title`, `--desc`, `--parent`, `--issue`, `--label`
- [x] Parent validation
- [x] Unit tests for usecase
- [x] CLI tests (happy path)
- [x] Integration test

### 2.2 List Tasks

- [x] `internal/usecase/list_tasks.go` - ListTasks usecase
- [x] `internal/cli/task.go` - list command
- [x] Flags: `--parent`, `--label`
- [x] TSV output with PARENT column
- [x] Unit tests
- [x] CLI tests (happy path)
- [x] Integration test

### 2.3 Show Task

- [x] `internal/usecase/show_task.go` - ShowTask usecase
- [x] `internal/cli/task.go` - show command
- [x] Display all fields including parent and sub-tasks
- [x] Auto-detect task ID from branch name
- [x] Unit tests
- [x] CLI tests (happy path)
- [x] Integration test

### 2.4 Edit Task

- [x] `internal/usecase/edit_task.go` - EditTask usecase
- [x] `internal/cli/task.go` - edit command
- [x] Flags: `--title`, `--desc`, `--add-label`, `--rm-label`
- [x] Unit tests
- [x] CLI tests (happy path)
- [x] Integration test

### 2.5 Delete Task

- [x] `internal/usecase/delete_task.go` - DeleteTask usecase
- [x] `internal/cli/task.go` - rm command
- [x] Delete task from store (no worktree cleanup yet)
- [x] Unit tests
- [x] CLI tests (happy path)
- [x] Integration test

### 2.6 Copy Task

- [x] `internal/usecase/copy_task.go` - CopyTask usecase
- [x] `internal/cli/task.go` - cp command
- [x] Copy title (append " (copy)"), description
- [x] Set base branch to source branch
- [x] Unit tests
- [x] CLI tests (happy path)
- [x] Integration test

### 2.7 Comment

- [x] `internal/usecase/add_comment.go` - AddComment usecase
- [x] `internal/cli/task.go` - comment command
- [x] Display comments in show output
- [x] Unit tests
- [x] CLI tests (happy path)
- [x] Integration test

### 2.8 Help & Documentation

- [x] `internal/cli/help.go` - help command (using Cobra's built-in)
- [x] Embed USAGE.md or generate from cobra (Cobra auto-generates from Long/Examples)
- [x] `--help` for all commands
- [x] Command grouping in help output

**Phase 2 Milestone**: Full task CRUD working. Can manage tasks with crew.

---

## Phase 3: Migration & Bootstrap

### 3.1 Migrate to Self-Hosting

- [x] Run `git crew init` in git-crew repository
- [x] Create initial tasks for remaining features using `git crew new`
- [x] Verify task management works for real development

### 3.2 Document Remaining Work

- [x] Create parent task: "Session Management"
- [x] Create parent task: "Workflow Commands"
- [x] Create parent task: "GitHub Integration"
- [x] Create parent task: "TUI"
- [x] Break down each into sub-tasks

**Phase 3 Milestone**: git-crew is now self-hosted. Use `./crew list` to see all remaining tasks.

---

## Task Dependencies

```
1.1 Project Setup
 └── 1.2 CI Setup
 └── 1.3 Domain Layer
      └── 1.4 JSON Store
           └── 1.5 App Container
                └── 1.6 CLI Skeleton
                     └── 1.7 Init Command
                          └── 2.1 New Task
                               └── 2.2 List Tasks
                               └── 2.3 Show Task
                                    └── 2.4 Edit Task
                                    └── 2.5 Delete Task
                                    └── 2.6 Copy Task
                                    └── 2.7 Comment
                                         └── 2.8 Help
                                              └── 3.1 Migrate
```

---

## Notes

- Each task should have tests before moving to next
- Run `mise run ci` after each task to ensure quality
- Commit frequently with clear messages
- After Phase 2, use `git crew` itself to track Phase 3+ tasks
