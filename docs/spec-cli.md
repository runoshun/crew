# CLI Specification

This document specifies the command-line interface for git-crew.

## Command Structure

```
crew [flags]
crew [command] [subcommand] [flags] [arguments]
```

### Global Flags

| Flag | Description |
|------|-------------|
| `-h, --help` | Show help for any command |
| `--help-worker` | Show detailed help for worker agents |
| `--help-manager` | Show detailed help for manager agents |
| `-v, --version` | Show version information |

---

## Setup Commands

### `crew init`

Initialize a repository for git-crew.

```
crew init [flags]
```

**Behavior:**
- Creates `.git/crew/` directory
- Generates default `config.toml` if it doesn't exist
- Sets up tmux configuration

### `crew config`

Display effective configuration.

```
crew config [flags]
```

**Output:**
- Shows merged configuration from all sources
- Includes built-in defaults, global config, and repo config

### `crew gen`

Generate files (skills, etc.).

```
crew gen [type] [flags]
```

---

## Task Management Commands

### `crew new`

Create a new task.

```
crew new [flags]
```

**Flags:**
- `--title string` - Task title (required)
- `--description string` - Task description
- `--parent int` - Parent task ID
- `--label string` - Task label (can be specified multiple times)
- `--branch string` - Branch name (default: auto-generated from task ID)
- `--issue int` - Link GitHub issue

**Output:**
- Task ID and details

**Examples:**
```bash
crew new --title "Add user authentication"
crew new --title "Fix bug" --description "Detailed description" --parent 5
crew new --title "Feature X" --issue 123
```

### `crew list`

List tasks.

```
crew list [flags]
```

**Flags:**
- `--status string` - Filter by status (default: all non-terminal)
- `--label string` - Filter by label
- `--format string` - Output format: table, json

**Output:**
- Table or JSON list of tasks

### `crew show`

Display task details.

```
crew show [id] [flags]
```

**Arguments:**
- `id` - Task ID (optional; uses current task if in a worktree)

**Output:**
- Full task information including title, description, status, branch, comments

### `crew edit`

Edit task information.

```
crew edit [id] [flags]
```

**Arguments:**
- `id` - Task ID (optional; uses current task if in a worktree)

**Flags:**
- `--title string` - New title
- `--description string` - New description
- `--status string` - New status
- `--label string` - Add or remove labels

**Examples:**
```bash
crew edit 1 --title "Updated title"
crew edit --status in_review
crew edit 1 --status needs_input
```

### `crew comment`

Add a comment to a task.

```
crew comment [id] [text] [flags]
```

**Arguments:**
- `id` - Task ID (optional; uses current task if in a worktree)
- `text` - Comment text

**Examples:**
```bash
crew comment 1 "Work in progress"
crew comment "Completed implementation"
```

### `crew close`

Close task without merging.

```
crew close [id] [flags]
```

**Arguments:**
- `id` - Task ID (optional; uses current task if in a worktree)

**Behavior:**
- Sets status to `closed`
- Stops session if running
- Preserves worktree and branch (use `prune` to clean up)

### `crew rm`

Delete a task.

```
crew rm [id] [flags]
```

**Arguments:**
- `id` - Task ID (required)

**Flags:**
- `-f, --force` - Force deletion without confirmation

**Behavior:**
- Removes task from store
- Stops session if running
- Removes worktree and branch

### `crew cp`

Copy a task.

```
crew cp [source-id] [flags]
```

**Arguments:**
- `source-id` - Source task ID

**Flags:**
- `--title string` - Title for the new task (defaults to "Copy of [original title]")
- `--description string` - Description for the new task

**Output:**
- New task ID

### `crew prune`

Cleanup branches and worktrees for completed tasks.

```
crew prune [flags]
```

**Flags:**
- `--dry-run` - Show what would be pruned without actually doing it
- `-f, --force` - Skip confirmation prompt

**Behavior:**
- Removes worktrees and branches for tasks with status `done` or `closed`

---

## Session Management Commands

### `crew start`

Start an AI agent session for a task.

```
crew start [id] [flags]
```

**Arguments:**
- `id` - Task ID (required)

**Flags:**
- `-m, --model string` - Model to use (overrides worker default)
- `-w, --worker string` - Worker to use (default: "default")

**Behavior:**
1. Creates worktree if it doesn't exist
2. Runs agent's worktree setup script (if defined)
3. Launches agent in tmux session
4. Sets task status to `in_progress`

**Examples:**
```bash
crew start 1
crew start 1 --model opus
crew start 1 --worker my-custom-worker
```

### `crew manager`

Launch a manager agent for task orchestration.

```
crew manager [name] [flags]
```

**Arguments:**
- `name` - Manager name (optional; default: "default")

**Flags:**
- `-m, --model string` - Model to use (overrides manager default)

**Behavior:**
- Launches manager agent in the current directory (not a worktree)
- Manager has access to all crew commands for task management
- Manager is read-only and delegates code implementation to workers

**Examples:**
```bash
crew manager
crew manager my-manager
crew manager --model opus
```

### `crew attach`

Attach to a running session.

```
crew attach [id] [flags]
```

**Arguments:**
- `id` - Task ID (optional; uses current task if in a worktree)

**Behavior:**
- Attaches to the tmux session for the task
- Requires the session to be running

### `crew peek`

View session output non-interactively.

```
crew peek [id] [flags]
```

**Arguments:**
- `id` - Task ID (optional; uses current task if in a worktree)

**Flags:**
- `-f, --follow` - Follow output (like `tail -f`)

**Output:**
- Session output without attaching

### `crew stop`

Stop a running session.

```
crew stop [id] [flags]
```

**Arguments:**
- `id` - Task ID (optional; uses current task if in a worktree)

**Behavior:**
- Terminates the tmux session
- Sets task status to `stopped`

### `crew send`

Send key input to a session.

```
crew send [id] [keys] [flags]
```

**Arguments:**
- `id` - Task ID (optional; uses current task if in a worktree)
- `keys` - Keys to send (tmux send-keys format)

**Examples:**
```bash
crew send 1 "echo hello" Enter
crew send "y" Enter
```

### `crew complete`

Mark task as complete and merge to main.

```
crew complete [id] [flags]
```

**Arguments:**
- `id` - Task ID (optional; uses current task if in a worktree)

**Behavior:**
1. Runs CI gate command (if configured in `[complete]` section)
2. Merges task branch to main
3. Sets task status to `done`
4. Stops session if running

**Examples:**
```bash
crew complete
crew complete 1
```

### `crew merge`

Merge task branch into main without completing.

```
crew merge [id] [flags]
```

**Arguments:**
- `id` - Task ID (required)

**Flags:**
- `--no-ff` - Force merge commit (no fast-forward)

### `crew diff`

Display task change diff.

```
crew diff [id] [flags] [-- args...]
```

**Arguments:**
- `id` - Task ID (optional; uses current task if in a worktree)

**Flags:**
- Additional flags are passed to the diff command

**Behavior:**
- Runs the diff command configured in `[diff]` section
- Supports template variables: `{{.BaseBranch}}`, `{{.Args}}`

**Examples:**
```bash
crew diff
crew diff 1
crew diff -- --stat
```

### `crew exec`

Execute a command in task's worktree.

```
crew exec [id] [flags] [-- command...]
```

**Arguments:**
- `id` - Task ID (optional; uses current task if in a worktree)
- `command` - Command to execute

**Examples:**
```bash
crew exec 1 -- make test
crew exec -- git status
```

---

## Additional Commands

### `crew tui`

Launch interactive TUI.

```
crew tui [flags]
```

**Behavior:**
- Opens a terminal UI for task management
- Supports keyboard navigation and task operations

### `crew sync`

Sync tasks with remote.

```
crew sync [flags]
```

**Behavior:**
- Fetches task refs from remote
- Merges remote task changes

### `crew snapshot`

Manage task snapshots.

```
crew snapshot [subcommand] [flags]
```

---

## Configuration

git-crew uses a TOML configuration file. Configuration is loaded from (in priority order):

1. Repository config: `.git/crew/config.toml`
2. User config: `~/.config/git-crew/config.toml`
3. Built-in defaults

### Configuration Sections

#### `[agents.<name>]`

Defines a base agent configuration that Workers and Managers can reference.

**Fields:**
- `command` - Base command to execute (e.g., "claude", "opencode")
- `command_template` - Template for assembling the full command
  - Available variables: `{{.Command}}`, `{{.SystemArgs}}`, `{{.Args}}`, `{{.Prompt}}`
- `default_model` - Default model for this agent
- `description` - Description of the agent's purpose
- `worktree_setup_script` - Script to run after worktree creation (template-expanded)
  - Available variables: `{{.Worktree}}`, `{{.TaskID}}`, `{{.Branch}}`, etc.
- `exclude_patterns` - Patterns to add to `.git/info/exclude` for this agent

**Example:**
```toml
[agents.my-agent]
command = "my-custom-agent"
command_template = "{{.Command}} {{.SystemArgs}} {{.Args}} {{.Prompt}}"
default_model = "my-model"
description = "My custom AI agent"
worktree_setup_script = """
#!/bin/bash
cd {{.Worktree}}
echo "Setting up worktree for task {{.TaskID}}"
"""
exclude_patterns = [".my-agent-cache/"]
```

#### `[workers]`

Common settings for all workers.

**Fields:**
- `system_prompt` - Default system prompt template for all workers (automatically set by crew)
- `prompt` - User custom instructions added after system_prompt

#### `[workers.<name>]`

Per-worker configuration. Workers are task execution agents that reference an Agent for base settings.

**Fields:**
- `agent` - Name of the Agent to inherit from (optional)
- `inherit` - Name of worker to inherit from (optional, for worker-to-worker inheritance)
- `command_template` - Template for assembling the command
- `command` - Base command (overrides agent's command)
- `system_args` - System arguments required for crew operation (auto-applied)
- `args` - User-customizable arguments
- `model` - Default model for this worker (overrides agent's default_model)
- `system_prompt` - System prompt template for this worker (overrides common system_prompt)
- `prompt` - Prompt template for this worker (overrides common prompt)
- `description` - Description of the worker's purpose

**Example:**
```toml
[workers.default]
agent = "opencode"
model = "sonnet"

[workers.my-worker]
agent = "my-agent"
model = "my-model-large"
args = "--verbose"
```

#### `[managers]`

Common settings for all managers.

**Fields:**
- `system_prompt` - Default system prompt template for all managers
- `prompt` - User custom instructions added after system_prompt

#### `[managers.<name>]`

Manager configuration. Managers are read-only orchestration agents that can create and monitor tasks.

**Fields:**
- `agent` - Name of the Agent to inherit from (optional)
- `model` - Model override for this manager
- `system_args` - System arguments required for crew operation (auto-applied)
- `args` - Additional arguments for this manager
- `system_prompt` - System prompt template for this manager (overrides common system_prompt)
- `prompt` - Prompt template for this manager (overrides common prompt)
- `description` - Description of the manager's purpose

**Example:**
```toml
[managers.default]
agent = "opencode"
model = "opus"
```

#### `[complete]`

Completion gate settings.

**Fields:**
- `command` - Command to run as CI gate on complete

**Example:**
```toml
[complete]
command = "mise run ci"
```

#### `[diff]`

Diff display settings.

**Fields:**
- `command` - Shell command to display diff (supports template variables)
  - `{{.BaseBranch}}` - Task's base branch (e.g., "main")
  - `{{.Args}}` - Additional arguments passed to the diff command
- `tui_command` - Command for TUI diff display (optional)

**Example:**
```toml
[diff]
command = "git diff {{.BaseBranch}}...HEAD{{if .Args}} {{.Args}}{{end}}"
tui_command = "git diff --color {{.BaseBranch}}...HEAD | less -R"
```

#### `[log]`

Logging settings.

**Fields:**
- `level` - Log level: debug, info, warn, error (default: "info")

**Example:**
```toml
[log]
level = "debug"
```

#### `[tasks]`

Task storage settings.

**Fields:**
- `store` - Storage backend: "git" (default) or "json"
- `namespace` - Git namespace for refs (default: "crew")
- `encrypt` - Enable encryption for task data (default: false)

**Example:**
```toml
[tasks]
store = "git"
namespace = "crew"
encrypt = false
```

#### `[worktree]`

Worktree customization settings.

**Fields:**
- `setup_command` - Command to run after worktree creation
- `copy` - Files/directories to copy (with CoW if available)

**Example:**
```toml
[worktree]
setup_command = "mise install"
copy = ["node_modules/", ".venv/"]
```

---

## Task Status

Tasks have the following lifecycle statuses:

| Status | Description | Transitions |
|--------|-------------|-------------|
| `todo` | Created, awaiting start | → `in_progress`, `closed` |
| `in_progress` | Agent working | → `in_review`, `needs_input`, `stopped`, `error`, `closed` |
| `in_review` | Work complete, awaiting review | → `in_progress`, `done`, `closed` |
| `needs_input` | Agent waiting for user input | → `in_progress`, `closed` |
| `stopped` | Manually stopped | → `in_progress`, `closed` |
| `error` | Session terminated abnormally | → `in_progress`, `closed` |
| `done` | Merge complete | → `closed` |
| `closed` | Discarded without merge (terminal) | - |

**Status Transitions:**
- `crew start` sets status to `in_progress`
- `crew stop` sets status to `stopped`
- `crew edit --status` allows manual status changes
- `crew complete` sets status to `done` after successful merge
- `crew close` sets status to `closed`

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Invalid arguments or flags |
| 3 | Task not found |
| 4 | Session not running |
| 5 | Git operation failed |

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `XDG_CONFIG_HOME` | User config directory (default: `~/.config`) |
| `CREW_LOG_LEVEL` | Override log level (debug, info, warn, error) |
