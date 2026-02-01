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

### 1.1 Task Status & Review Flow

Task statuses are normalized to the following lifecycle:

| Status | Meaning |
|--------|---------|
| `todo` | Created, awaiting start |
| `in_progress` | Work in progress (includes input waiting and paused states) |
| `done` | Implementation complete, awaiting merge or close |
| `merged` | Merged to base branch (terminal) |
| `closed` | Closed without merge (terminal) |
| `error` | Session terminated unexpectedly or manually stopped (restartable) |

Main flow:

```
todo -> in_progress -> done -> merged
                 \-> closed
error -> in_progress
```

Reviews run synchronously inside `crew complete` and do not change task status unless completion succeeds. Review count increments only when the review result is recorded. `crew complete` requires the review count to satisfy `min_reviews` (unless `skip_review` is enabled) and then transitions the task to `done`.

---

## 2. Core Technologies

### 2.1 Git Worktree

Each task gets an isolated worktree in `<repo>-worktrees/<task-id>/`. Branch naming follows `crew-<taskID>` format. Worktrees are created lazily on first `start` and cleaned up on `merge` or `prune`.

---

### 2.2 Concurrent AI Agent Execution

AI agents run in tmux sessions, enabling background execution and attach/detach. git-crew uses a dedicated socket (`.crew/tmux.sock`) to isolate from system tmux sessions.

---

### 2.3 Task Data Store

Task data is stored as files under `.crew/tasks/<namespace>/`.
Each task is split into:

- `<id>.md` (frontmatter, description, comments)
- `<id>.meta.json` (status and lifecycle metadata)
- `meta.json` (namespace schema and next_id)

Operational note: when committing changes under `.crew/`, only include `.crew/tasks/**`.
Other paths under `.crew/` (logs, runtime state, tmux socket) are local and should not be versioned.
To prevent accidental staging, keep `.crew/logs/`, `.crew/config.runtime.toml`, and `.crew/tmux.sock` out of VCS via `.git/info/exclude` or `.gitignore` per repo policy.

---

## 3. Agent Adapter Model

git-crew is designed to work with any AI coding CLI (Claude Code, OpenCode, Codex, etc.). Each CLI has different invocation patterns, argument formats, and environment requirements.

### Agent, Worker, Manager

The system uses three distinct concepts:

#### **Agent**: Base AI CLI Configuration

An **Agent** defines the core command execution pattern for an AI tool, independent of its role. Agents are reusable templates that specify:

- **Command**: Base executable name (e.g., `"claude"`, `"opencode"`)
- **CommandTemplate**: How to assemble arguments (e.g., `"{{.Command}} {{.SystemArgs}} {{.Args}} {{.Prompt}}"`)
- **DefaultModel**: Default model for this agent (e.g., `"sonnet"`, `"gpt-4o"`)
- **SetupScript**: Per-worktree initialization script (e.g., creating `.claude/` settings)
- **ExcludePatterns**: Files to hide from git status (e.g., `.opencode/`)

Agents are defined in the `[agents.<name>]` configuration section and do **not** include role-specific settings like `system_args` or prompts.

#### **Worker**: Task Execution Agent

A **Worker** is a task execution agent that performs actual code implementation. Workers:

- Reference an Agent (or define their own command)
- Add role-specific configuration (`system_args`, `system_prompt`, `prompt`)
- Run in isolated worktrees with full read-write access
- Can be customized per task using `--worker` flag

Workers are configured in `[workers.<name>]` sections and automatically inherit from their Agent's settings. Multiple workers can share the same Agent with different prompts or models.

#### **Manager**: Task Orchestration Agent

A **Manager** is a read-only orchestration agent that:

- Creates, monitors, and manages tasks using crew commands
- Delegates code implementation to Workers (does not edit files directly)
- Runs in the repository root (not a worktree)
- Has access to all crew commands for task management

Managers are configured in `[managers.<name>]` sections and also reference Agents for their base configuration.

### Separation of Concerns

This three-tier design provides clear separation:

1. **Agents** - Define CLI-specific setup once (command, setup script, exclusions)
2. **Workers/Managers** - Add role-specific configuration (prompts, args, model overrides)
3. **Tasks** - Reference Workers for execution

This allows you to:
- Define CLI-specific setup once per Agent
- Create multiple Workers with different prompts/models using the same Agent
- Create multiple Managers with different orchestration styles
- Switch between AI tools without changing task workflows

---

## 4. Configuration System

Configuration is loaded and merged from multiple sources (later takes precedence):
1. Built-in defaults
2. User config: `~/.config/crew/config.toml`
3. User override: `~/.config/crew/config.override.toml`
4. Root repo config: `.crew.toml`
5. Repository config: `.crew/config.toml`
6. Runtime config: `.crew/config.runtime.toml` (TUI/system state)

### Configuration Structure

The configuration format has been unified under the `[agents]` table, replacing the older `[workers]` and `[managers]` separation.

```toml
[agents]
worker_default = "opencode"      # Default agent for starting tasks
worker_prompt = ""               # Default prompt for all worker agents

manager_default = "opencode-manager" # Default agent for starting manager tasks
manager_prompt = ""                  # Default prompt for all manager agents

# Define a new agent or override an existing one
[agents.my-agent]
inherit = "opencode"             # Inherit settings from another agent
default_model = "gpt-4o"         # Override model
description = "My custom agent"  # Description for agent selection
role = "worker"                  # "worker" or "manager"
# system_prompt = "..."          # Add custom system prompt
# prompt = "..."                 # Add custom user prompt
# args = "--verbose"             # Add CLI arguments

# Complex custom agent definition
[agents.custom-cli]
command_template = "my-cli --model {{.Model}} {{.Args}} --prompt {{.Prompt}}"
setup_script = """
#!/bin/bash
cd {{.Worktree}}
echo "Setting up custom environment..."
"""
exclude_patterns = [".cache/"]

# Worktree initialization settings
[worktree]
setup_command = "npm install"    # Run after creation
copy = [".env.example"]          # Copy files from main repo

# CI gate on completion
[complete]
command = "mise run ci"

# Diff display
[diff]
command = "git diff {{.BaseBranch}}...HEAD{{if .Args}} {{.Args}}{{end}}"

# Logging
[log]
level = "info"

# Task settings
[tasks]
new_task_base = "current"  # Base branch for new tasks: "current" (default) or "default"

# Git Configuration (via git config)
# These settings are configured using `git config` rather than TOML

# crew.defaultBranch
#   Override the default branch name used for merging and diffs
#   Priority: 1) git config crew.defaultBranch, 2) origin/HEAD, 3) "main"
#   Example: git config crew.defaultBranch develop

# TUI Customization
[tui.keybindings]
"ctrl+r" = { command = "crew run-review {{.TaskID}}", description = "Run review script" }
```

### Agent Inheritance

The `inherit` field allows you to create specialized variations of agents without redefining everything.

1. **Base Agent**: Defines the CLI command and argument template (e.g., `claude`, `opencode`).
2. **Specialized Agent**: Inherits from Base, overrides model or adds prompts (e.g., `claude-architect`).

This allows you to easily switch models or create role-specific agents (QA, Security, Frontend) while reusing the underlying CLI integration.


---

## 5. Worktree Setup and Exclusions

### Worktree Setup Script

Agents can define a `worktree_setup_script` that runs automatically after worktree creation. This script is useful for:

- Creating agent-specific configuration files (e.g., `.claude/`, `.opencode/`)
- Copying or linking shared resources
- Setting up agent-specific environment

The script is template-expanded with the following variables:
- `{{.Worktree}}` - Worktree path
- `{{.TaskID}}` - Task ID
- `{{.Branch}}` - Branch name
- `{{.GitDir}}` - .git directory path
- `{{.RepoRoot}}` - Repository root path

**Example:**
```bash
#!/bin/bash
cd {{.Worktree}}
mkdir -p .opencode/skill
echo "Worktree setup complete for task {{.TaskID}}"
```

### Exclude Patterns

Agents can define `exclude_patterns` to automatically hide agent-specific files from git status. These patterns are added to `.git/info/exclude` in the worktree.

This prevents clutter from:
- Agent cache directories (e.g., `.opencode/`, `.claude/cache/`)
- Temporary agent files
- Agent-specific build artifacts

**Example:**
```toml
[agents.opencode]
exclude_patterns = [".opencode/", ".codex/"]
```

These exclusions are applied automatically when starting a task with the agent, ensuring a clean git status output without polluting the repository's `.gitignore`.

---

## 6. Naming Conventions

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
.crew/
├── tasks/                  # Task store (namespaced)
│   ├── default/
│   │   ├── meta.json
│   │   ├── 1.md
│   │   └── 1.meta.json
├── config.toml             # Repository config
├── config.runtime.toml     # Runtime config (TUI/system state)
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

## 7. Dependencies

| Tool | Purpose | Required |
|------|---------|----------|
| git | Version control, worktree | Yes |
| tmux | Session management | Yes |
| gh | GitHub integration | No |
