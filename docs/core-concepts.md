# crew Core Concepts

## 1. Fundamental Principle

### 1 Task = 1 Worktree = 1 Session

The core design principle of crew:

- **One Task** = An independent unit of work (ID, title, description, status)
- **One Worktree** = An isolated filesystem workspace
- **One Session** = An AI agent process managed by tmux

This design enables:

- Fully parallel execution of multiple tasks
- Complete isolation between tasks
- Clear per-task state tracking
- Attach/detach from sessions at any time

---

## 2. Core Technologies

### 2.1 Git Worktree

Each task gets an isolated worktree in `<repo>-worktrees/<task-id>/`. Branch naming follows `crew-<taskID>` format. Worktrees are created lazily on first `start` and cleaned up on `merge` or `prune`.

---

### 2.2 Concurrent AI Agent Execution

AI agents run in tmux sessions, enabling background execution and attach/detach. git-crew uses a dedicated socket (`.git/crew/tmux.sock`) to isolate from system tmux sessions.

---

### 2.3 Task Data Store

Task data is stored in Git refs under `refs/crew/` namespace. This enables sharing tasks across clones and leveraging Git's built-in integrity and versioning.

---

## 3. Agent Adapter Model

git-crew is designed to work with any AI coding CLI (Claude Code, OpenCode, Codex, etc.). Each CLI has different invocation patterns, argument formats, and environment requirements.

The **Agent** abstraction handles these differences:

- **Command & Args**: How to invoke the CLI and pass prompts
- **SetupScript**: Per-worktree initialization (e.g., creating `.claude/` settings)
- **ExcludePatterns**: Files to hide from git status (e.g., `.opencode/`)

**Workers** and **Managers** reference an Agent and add role-specific configuration (prompts, model overrides). This separation allows:

1. Define CLI-specific setup once per Agent
2. Create multiple Workers with different prompts/models using the same Agent
3. Switch between AI tools without changing task workflows

---

## 4. Configuration System

Configuration is loaded from (in priority order):
1. Repository config: `.git/crew/config.toml`
2. User config: `~/.config/git-crew/config.toml`
3. Built-in defaults

---

## 5. Naming Conventions

### Resource Naming

| Resource | Format | Example |
|----------|--------|---------|
| Branch | `crew-<id>` | `crew-1` |
| Branch (with issue) | `crew-<id>-gh-<issue>` | `crew-1-gh-123` |
| Session | `crew-<id>` | `crew-1` |
| Task script | `task-<id>.sh` | `task-1.sh` |
| Prompt file | `task-<id>-prompt.txt` | `task-1-prompt.txt` |
| Task log | `task-<id>.log` | `task-1.log` |

### File Layout

```
.git/crew/
├── tasks.json              # Task store
├── config.toml             # Repository config
├── tmux.sock               # tmux socket
├── tmux.conf               # tmux config
├── scripts/
│   ├── task-1.sh           # Task 1 script
│   ├── task-1-prompt.txt   # Task 1 prompt
│   ├── review-1.sh         # Review script
│   └── review-1-prompt.txt # Review prompt
└── logs/
    ├── crew.log            # Global log
    ├── task-1.log          # Task 1 log
    └── task-2.log          # Task 2 log
```

---

## 6. Dependencies

| Tool | Purpose | Required |
|------|---------|----------|
| git | Version control, worktree | Yes |
| tmux | Session management | Yes |
| gh | GitHub integration | No |

