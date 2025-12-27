# git-crew v2 Documentation

## What is git-crew?

git-crew is a CLI tool for managing AI coding agent tasks. It combines git worktree + tmux to achieve a model where **1 task = 1 worktree = 1 AI session**, enabling fully parallel and isolated task execution.

## Documentation Structure

| Document | Description |
|----------|-------------|
| [core-concepts.md](./core-concepts.md) | Fundamental principles, technologies, and design decisions |
| [spec-cli.md](./spec-cli.md) | CLI command specifications and data model |
| [spec-tui.md](./spec-tui.md) | TUI screen specifications and keybindings |
| [architecture.md](./architecture.md) | Technical architecture, code structure, and implementation details |

## Implementation Plan

See [../TASKS.md](../TASKS.md) for the implementation task list.

## Quick Overview

### Core Principle

```
1 Task = 1 Worktree = 1 Session
```

- **Task**: Unit of work with ID, title, description, status
- **Worktree**: Isolated git working directory
- **Session**: tmux-managed AI agent process

### Key Features

- Parallel task execution with complete isolation
- Parent-child task hierarchy (like GitHub Sub-issues)
- Attach/detach from agent sessions anytime
- JSON-based task storage in `.git/crew/`

### Dependencies

| Tool | Required | Purpose |
|------|----------|---------|
| git | Yes | Version control, worktree management |
| tmux | Yes | Session management |
| gh | No | GitHub integration |

### Architecture

```
CLI/TUI (Presentation)
    ↓
Container (DI)
    ↓
UseCases (1 file = 1 use case)
    ↓
Domain (Entities + Ports)
    ↓
Infrastructure (JSON Store, tmux, git)
```

## Reading Order for New Developers

1. **[core-concepts.md](./core-concepts.md)** - Understand the "why" and core principles
2. **[spec-cli.md](./spec-cli.md)** - Learn what commands exist and data model
3. **[architecture.md](./architecture.md)** - Understand how code is organized
4. **[../TASKS.md](../TASKS.md)** - See what needs to be implemented
5. **[spec-tui.md](./spec-tui.md)** - TUI details (when implementing TUI)

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| JSON storage | Simple, single file, no external DB needed |
| tmux for sessions | Robust, widely available, attach/detach support |
| Native git worktree | No external dependency (removed git-gtr) |
| Clean Architecture | Testable, maintainable, clear boundaries |
| 1 UseCase = 1 File | Easy to find, focused, testable |
| Parent-child tasks | GitHub Sub-issues compatible, keeps context visible |

## Status

v2 is a complete rewrite of v1 with:
- Clean architecture
- Better code organization
- Parent-child task hierarchy
- Improved documentation
