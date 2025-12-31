# git-crew v2 Specification

## Table of Contents

1. [Command Specification](#command-specification)
2. [Data Model](#data-model)
3. [Status Transitions](#status-transitions)
4. [Configuration System](#configuration-system)
5. [Naming Conventions](#naming-conventions)
6. [Agent Contract](#agent-contract)
7. [TUI Specification](#tui-specification)

---

## Command Specification

### Initialization & Configuration

#### `git crew init`

Initialize a repository for git-crew.

**Preconditions**:
- Current directory is inside a git repository

**Processing**:
1. Create `.git/crew/` directory
2. Generate `.git/crew/tmux.conf` (minimal tmux configuration)
3. Initialize `.git/crew/tasks.json` as empty

**Error Conditions**:
- Already initialized → "crew already initialized"
- `tmux` command not found → error

**Generated Files**:
```
.git/crew/
├── tmux.conf
└── tasks.json
```

---

#### `git crew config [--ignore-global] [--ignore-repo]`

Display configuration information including effective (merged) configuration.

**Optional Arguments**:
- `--ignore-global`: Exclude global config from effective config calculation
- `--ignore-repo`: Exclude repository config from effective config calculation

**Output**:
- Effective configuration (merged result of all config sources)
- Global config (`~/.config/git-crew/config.toml`) path and contents
- Repository config (`.git/crew/config.toml`) path and contents
- Show "(not found)" for non-existent files

**Examples**:
```bash
# Show all config (default + global + repo merged)
git crew config

# Show config without global settings
git crew config --ignore-global

# Show config without repo settings (global + default only)
git crew config --ignore-repo
```

---

#### `git crew config init [--global]`

Generate a configuration file template.

**Options**:
- `--global`: Generate global configuration

**Target Path**:
- With `--global` → `~/.config/git-crew/config.toml`
- Without `--global` → `.git/crew/config.toml`

**Error Conditions**:
- Target file already exists → error

---

#### `git crew gen review`

Auto-detect and configure an appropriate `[complete].command` for the project.

**Detection Priority**:
1. `mise.toml` has `ci` task → `mise run ci`
2. Go project + golangci-lint → `golangci-lint run ./... && go test ./...`
3. Go project → `go test ./...`
4. None matched → empty string

**Output**:
- Display the configured command and config file path

---

### Task Management

#### `git crew new --title <title> [--desc <desc>] [--issue <num>] [--label <label>]... [--parent <id>]`

Create a new task.

**Required Arguments**:
- `--title`: Task title

**Optional Arguments**:
- `--desc`: Task description
- `--issue`: Linked GitHub issue number
- `--label`: Labels (can specify multiple)
- `--parent`: Parent task ID (creates a sub-task)

**Processing**:
1. Assign new task ID (monotonic counter, no reuse)
2. If `--parent` specified, verify parent task exists
3. Save task info to store
   - Status: `todo`
   - ParentID: parent task ID (or null)
   - Created: current time

**Error Conditions**:
- Not initialized → error
- Title is empty → error
- Parent task does not exist → error

**Note**: Worktree/branch is NOT created until `start` (lazy creation)

**Examples**:
```bash
# Create a root task
git crew new --title "Auth refactoring"

# Create a sub-task
git crew new --parent 1 --title "OAuth2.0 implementation"

# Create a nested sub-task (unlimited nesting)
git crew new --parent 2 --title "Google OAuth"
```

---

#### `git crew list [--label <label>]... [--parent <id>]`

Display task list.

**Optional Arguments**:
- `--label`: Filter by labels (AND condition when multiple)
- `--parent`: Show only direct children of the specified task

**Output Format** (TSV):
```
ID	PARENT	STATUS	AGENT	LABELS	[ELAPSED]	TITLE
```

**Fields**:
- `PARENT`: Parent task ID, or `-` if root task
- `AGENT`: Running agent name, or `-` if none
- `LABELS`: `[label1,label2]` format, or `-` if none
- `ELAPSED`: Elapsed time shown only for `in_progress` (e.g., `(5m)`)

**Example Output**:
```
ID   PARENT  STATUS        AGENT   LABELS  TITLE
1    -       in_progress   -       -       Auth refactoring
2    1       done          -       -       OAuth2.0 implementation
5    2       done          -       -       Google OAuth
6    2       done          -       -       GitHub OAuth
3    1       in_progress   claude  -       Session management
4    1       todo          -       -       Token refresh
7    -       todo          -       -       Fix typo
```

---

#### `git crew show [id]`

Display task details.

**Arguments**:
- `id`: Task ID (auto-detected from current branch if omitted)

**Output Format**:
```
# Task <id>: <title>

<description>

Status: <status>
Parent: #<parent-id> (or "none")
Branch: <branch>
Labels: [<labels>]
Created: <timestamp>
Issue: #<num>
PR: #<num>
Agent: <agent> (session: <session>)

Sub-tasks:
  #<id> [<status>] <title>
  #<id> [<status>] <title>

Comments:
  [<timestamp>] <text>
```

**Notes**:
- `Parent` field shows parent task ID if this is a sub-task
- `Sub-tasks` section only shown if the task has children
- Sub-tasks are listed with their ID, status, and title

**Error Conditions**:
- Task does not exist → error
- ID omitted and not on a crew branch → error

---

#### `git crew edit <id> [--title <title>] [--desc <desc>] [--status <status>] [--labels <labels>] [--add-label <label>]... [--rm-label <label>]...`

Edit task information.

**Required Arguments**:
- `id`: Task ID

**Optional Arguments** (at least one required):
- `--title`: New title
- `--desc`: New description
- `--status`: New status (`todo`, `in_progress`, `in_review`, `done`, `closed`, `error`)
- `--labels`: Replace all labels (comma-separated, e.g., `bug,urgent`)
- `--add-label`: Labels to add (can specify multiple)
- `--rm-label`: Labels to remove (can specify multiple)

**Notes**:
- `--labels` replaces all existing labels with the specified set
- If `--labels` is specified, `--add-label` and `--rm-label` are ignored
- Use `--labels ""` to clear all labels
- `--status` allows manual status changes without following normal transition rules

**Error Conditions**:
- Task does not exist → error
- No options specified → error
- Invalid status value → error

---

#### `git crew comment <id> "message"`

Add a comment to a task.

**Arguments**:
- `id`: Task ID
- `message`: Comment text

**Processing**:
- Save comment with timestamp

**Error Conditions**:
- Task does not exist → error
- Message is empty → error

---

#### `git crew diff <id> [args...]`

Display task change diff.

**Arguments**:
- `id`: Task ID
- `args`: Additional diff arguments (e.g., `--stat`)

**Processing**:
1. Resolve task's worktree path
2. Execute diff using `diff.command` from config
3. Expand `{{.Args}}` with additional arguments

**Error Conditions**:
- Task does not exist → error
- Worktree does not exist → error

---

#### `git crew cp <id> [--title <new-title>]`

Copy a task.

**Arguments**:
- `id`: Source task ID

**Optional Arguments**:
- `--title`: New title (defaults to `<original title> (copy)`)

**Processing**:
1. Create new task (status: `todo`)
2. Copy description
3. Set base branch to source branch

**Not Inherited**:
- GitHub issue link
- PR number
- Comments

**Error Conditions**:
- Source task does not exist → error

---

#### `git crew rm <id>`

Delete a task.

**Arguments**:
- `id`: Task ID

**Processing**:
1. Stop session if running
2. Delete worktree if exists
3. Delete task script
4. Delete task from store

**Error Conditions**:
- Task does not exist → error

---

### Session Management

#### `git crew start <id> [agent] [--model <model>]`

Start an AI agent session.

**Arguments**:
- `id`: Task ID
- `agent`: Agent name (uses `default_agent` setting if omitted)

**Optional Arguments**:
- `--model`, `-m`: Model name override (uses agent's default model if omitted)

**Preconditions**:
- Status is `todo`, `in_review`, or `error`
- No session running for the same task

**Processing**:
1. Create worktree/branch if they don't exist
2. Resolve agent command and prompt
3. Resolve model (use `--model` value or fall back to agent's default)
4. Generate task script (with session termination callback)
5. Start tmux session in background
6. Update status to `in_progress`
7. Save agent info

**Examples**:
```bash
# Start with default model
git crew start 1 claude

# Start with specific model
git crew start 1 claude --model sonnet
git crew start 1 opencode -m gpt-4o
```

**Error Conditions**:
- Task does not exist → error
- Agent not specified and `default_agent` not set → error
- Session already running → error
- Status is `done` or `closed` → error

---

#### `git crew attach <id>`

Attach to a running session.

**Arguments**:
- `id`: Task ID

**Preconditions**:
- Session is running

**Error Conditions**:
- Task does not exist → error
- Session is not running → error

---

#### `git crew peek <id> [--lines <n>]`

View session output non-interactively.

**Arguments**:
- `id`: Task ID

**Optional Arguments**:
- `--lines`, `-n`: Number of lines to display (default: 30)

**Preconditions**:
- Session is running

**Error Conditions**:
- Task does not exist → error
- Session is not running → error

---

#### `git crew stop <id>`

Stop a session.

**Arguments**:
- `id`: Task ID

**Processing**:
1. Terminate tmux session
2. Delete task script
3. Clear agent info
4. Update status to `in_review`

**Error Conditions**:
- Task does not exist → error

---

#### `git crew send <id> <keys>`

Send key input to a session.

**Arguments**:
- `id`: Task ID
- `keys`: Keys to send (`Tab`, `Escape`, `Enter`, or any text)

**Preconditions**:
- Session is running

**Error Conditions**:
- Task does not exist → error
- Session is not running → error

---

#### `git crew complete [id]`

Mark task as complete (`in_progress` → `in_review`).

**Arguments**:
- `id`: Task ID (auto-detected from current branch if omitted)

**Preconditions**:
- Status is `in_progress`
- No uncommitted changes in worktree

**Processing**:
1. If `[complete].command` is configured, execute it
   - Abort on failure
2. Update status to `in_review`

**Error Conditions**:
- Task does not exist → error
- Status is not `in_progress` → error
- Uncommitted changes exist → error
- `[complete].command` fails → error

---

#### `git crew review [id]`

Run review agent.

**Arguments**:
- `id`: Task ID (auto-detected from current branch if omitted)

**Preconditions**:
- Status is `in_progress` or `in_review`
- No uncommitted changes in worktree
- No merge conflict with main branch

**Processing**:
1. If `[review].agent` is configured:
   - Build review prompt (including diff content)
   - Execute agent
2. If not configured:
   - Display "Review agent not configured"

**Error Conditions**:
- Task does not exist → error
- Status is `todo` → error (need to `start` first)
- Status is `done` or `closed` → error
- Uncommitted changes exist → error
- Merge conflict exists → error

---

#### `git crew _session-ended <id> <exit-code>` (internal command)

Handle session termination (called from task script).

**Processing**:
1. Ignore if agent info already cleared (race condition prevention)
2. Clear agent info
3. Delete task script
4. Status transition:
   - Normal exit in `in_review` state → maintain
   - Abnormal exit otherwise → transition to `error`

---

### Task Completion

#### `git crew merge <id> [--force] [--yes]`

Merge task into main and complete.

**Arguments**:
- `id`: Task ID

**Optional Arguments**:
- `--force`: Force stop session if running and merge
- `--yes`, `-y`: Skip confirmation prompt

**Preconditions**:
- Current branch is `main`
- main's working tree is clean
- No merge conflict

**Processing**:
1. If session is running:
   - Without `--force` → error
   - With `--force` → stop session
2. Display confirmation prompt (skip with `--yes`)
3. Execute `git merge --no-ff`
4. Delete worktree
5. Delete branch
6. Update status to `done`
7. Clear agent info

**Error Conditions**:
- Task does not exist → error
- Current branch is not `main` → error
- Working tree has uncommitted changes → error
- Merge conflict exists → error
- Session running and no `--force` → error

---

#### `git crew close <id>`

Close task without merging.

**Arguments**:
- `id`: Task ID

**Processing**:
1. Stop session if running
2. Delete worktree
3. Update status to `closed`
4. Clear agent info

**Error Conditions**:
- Task does not exist → error

---

#### `git crew prune [--all] [--dry-run] [--yes]`

Clean up completed tasks and orphaned resources.

**Optional Arguments**:
- `--all`: Include `done` tasks in deletion targets
- `--dry-run`: Display only, no deletion
- `--yes`, `-y`: Skip confirmation prompt

**Deletion Targets**:
- Tasks with `closed` status
- With `--all`, also tasks with `done` status
- Orphan branches: match `crew-*` pattern but task doesn't exist
- Orphan worktrees: same as above

**Processing**:
1. Collect and display deletion targets
2. If no targets, display "Nothing to prune." and exit
3. If `--dry-run`, display "Dry run: no changes made." and exit
4. Confirmation prompt (skip with `--yes`)
5. Delete each target

---

### GitHub Integration

#### `git crew import <issue> [--parent <id>]`

Create task from GitHub issue.

**Arguments**:
- `issue`: Issue number

**Optional Arguments**:
- `--parent`: Parent task ID (import as a sub-task)

**Processing**:
1. Fetch issue info with `gh issue view`
2. If `--parent` specified, verify parent task exists
3. Create task:
   - Title: issue title
   - Description: issue body
   - Issue link: issue number
   - Labels: issue labels
   - ParentID: parent task ID (or null)

**Error Conditions**:
- `gh` command not found → error
- Issue does not exist → error
- Parent task does not exist → error

**Examples**:
```bash
# Import as root task
git crew import 123

# Import as sub-task
git crew import 123 --parent 1
```

---

#### `git crew pr <id>`

Create or update PR.

**Arguments**:
- `id`: Task ID

**Processing**:
1. Push branch to remote
2. Generate PR body (`github.pr_body` template)
3. Check for existing PR:
   - Use task's PR number if exists
   - Otherwise search by branch name
4. Create or update PR
5. Save PR number to task

**Error Conditions**:
- `gh` command not found → error
- Task does not exist → error
- Worktree does not exist → error

---

### Orchestrator

#### `git crew manager [agent]`

Launch read-only manager agent.

**Arguments**:
- `agent`: Agent name (defaults to `manager.default_agent` or "claude")

**Processing**:
1. Get prompt (`manager.prompt` or built-in)
2. Build agent command
3. Execute agent in foreground at repository root

**Agent-specific Settings**:
- `claude`: `--disallowed-tools=Write,Edit,MultiEdit,NotebookEditCell`
- `opencode`: Pass prompt with `--prompt`
- `codex`: `--sandbox read-only`

---

### Debug

#### `git crew log [id] [--lines <n>] [--global]`

Display logs.

**Arguments**:
- `id`: Task ID (auto-detected from current branch if omitted)

**Optional Arguments**:
- `--lines`, `-n`: Number of lines to display (default: all)
- `--global`: Display global log

**Log Files**:
- Task log: `.git/crew/logs/task-<id>.log`
- Global log: `.git/crew/logs/crew.log`

**Error Conditions**:
- Cannot determine task ID without `--global` → error

---

### TUI

#### `git crew ui` / `git crew tui`

Launch interactive TUI.

See [TUI Specification](#tui-specification) for details.

---

## Data Model

### Task Entity

| Field | Type | Description |
|-------|------|-------------|
| ID | int | Task ID (monotonic, no reuse) |
| ParentID | int? | Parent task ID (null=root task) |
| Title | string | Title (required) |
| Description | string | Description (optional) |
| Status | string | Status |
| Created | timestamp | Creation time |
| Started | timestamp | `in_progress` start time |
| Agent | string | Running agent name |
| Session | string | tmux session name |
| Issue | int | GitHub issue number (0=not linked) |
| PR | int | GitHub PR number (0=not created) |
| BaseBranch | string | Base branch (for copy/lazy creation) |
| Labels | []string | Labels |

**Parent-Child Relationship**:
- Tasks can have parent-child relationships (unlimited nesting)
- Both parent and child tasks can have worktrees
- Parent task status is managed manually (not auto-calculated from children)
- Compatible with GitHub Sub-issues feature

### Comment Entity

| Field | Type | Description |
|-------|------|-------------|
| Text | string | Comment text |
| Time | timestamp | Creation time |

### Storage

**File Path**: `.git/crew/tasks.json`

```json
{
  "meta": {
    "nextTaskID": 100
  },
  "tasks": {
    "1": {
      "parentID": null,
      "title": "Auth refactoring",
      "description": "Refactor authentication system",
      "status": "in_progress",
      "created": "2024-01-01T00:00:00Z",
      "started": "2024-01-01T00:00:00Z",
      "agent": "",
      "session": "",
      "issue": 100,
      "pr": 0,
      "baseBranch": "main",
      "labels": ["feature"]
    },
    "2": {
      "parentID": 1,
      "title": "OAuth2.0 implementation",
      "description": "Add OAuth2.0 support",
      "status": "done",
      "created": "2024-01-01T01:00:00Z",
      "started": "2024-01-01T01:30:00Z",
      "agent": "",
      "session": "",
      "issue": 101,
      "pr": 10,
      "baseBranch": "main",
      "labels": []
    },
    "5": {
      "parentID": 2,
      "title": "Google OAuth",
      "description": "",
      "status": "done",
      "created": "2024-01-01T02:00:00Z",
      "started": "2024-01-01T02:30:00Z",
      "agent": "",
      "session": "",
      "issue": 0,
      "pr": 0,
      "baseBranch": "main",
      "labels": []
    }
  },
  "comments": {
    "1": [
      { "text": "Started planning", "time": "2024-01-01T00:00:00Z" }
    ]
  }
}
```

**Characteristics**:
- All data in a single file
- Task ID stored as string key
- `parentID` is null for root tasks, integer for sub-tasks
- Unlimited nesting depth
- File locking for concurrent access control

---

## Status Transitions

### Status List

| Status | Description |
|--------|-------------|
| `todo` | Created, awaiting start |
| `in_progress` | Agent working |
| `in_review` | Work complete, awaiting review |
| `error` | Session terminated abnormally |
| `done` | Merge complete |
| `closed` | Discarded without merge |

### Transition Diagram

```
new → [todo] → start → [in_progress] → complete → [in_review] → merge → [done]
                   ↑         │                          │
                   │   abnormal exit                    │
                   │         ↓                          │
                   └─start─ [error]                     └─ close → [closed]
```

### Transition Rules

| Current Status | Allowed Targets |
|----------------|-----------------|
| `todo` | `in_progress`, `closed` |
| `in_progress` | `in_review`, `error`, `closed` |
| `in_review` | `in_progress`, `done`, `closed` |
| `error` | `in_progress`, `closed` |
| `done` | `closed` |
| `closed` | none (terminal state) |

### Transition Triggers

| Action | Status Change |
|--------|---------------|
| `new` | → `todo` |
| `start` | `todo` / `in_review` / `error` → `in_progress` |
| `complete` | `in_progress` → `in_review` |
| `stop` | `in_progress` → `in_review` |
| Abnormal session exit | `in_progress` → `error` |
| `merge` | any → `done` |
| `close` | any → `closed` |

---

## Configuration System

### Configuration File Priority

1. Repository config: `.git/crew/config.toml`
2. User config: `~/.config/git-crew/config.toml`
3. Built-in defaults

### Configuration Items

```toml
# Default agent
default_agent = "claude"

# Common agent settings
[agent]
prompt = "When the task is complete, run 'git crew complete'."

# Log settings
[log]
level = "info"  # debug, info, warn, error

# Built-in agent argument extensions
[agents.claude]
args = "--model claude-sonnet-4-20250514 --add-dir {{.GitDir}}"

[agents.opencode]
args = "-m gpt-4"

[agents.codex]
args = "--model gpt-5.2-codex --add-dir {{.GitDir}}"

# Custom agent
[agents.my-agent]
command = 'my-custom-agent --task "{{.Title}}"'

# CI gate on completion
[complete]
command = "mise run ci"

# Diff display settings
[diff]
command = "git diff main...HEAD{{if .Args}} {{.Args}}{{end}} | delta"
tui_command = "git diff --color main...HEAD | less -R"

# Review settings
[review]
agent = "claude"
prompt = "Check for bugs and code quality"

# GitHub integration
[github]
pr_body = '''
## Summary
{{.Title}}

{{.Description}}

{{if .Issue}}Closes #{{.Issue}}{{end}}
'''

# Manager settings
[manager]
default_agent = "claude"
prompt = ""

[manager.agents.claude]
args = "--model opus"
```

### Template Variables

| Variable | Description |
|----------|-------------|
| `{{.ID}}` | Task ID |
| `{{.Title}}` | Title |
| `{{.Description}}` | Description |
| `{{.Branch}}` | Branch name |
| `{{.Worktree}}` | Worktree path |
| `{{.Status}}` | Status |
| `{{.Issue}}` | Issue number |
| `{{.PR}}` | PR number |
| `{{.Prompt}}` | Common prompt |
| `{{.Args}}` | Agent additional args |
| `{{.Model}}` | Model name (from `--model` flag or agent's default) |
| `{{.RepoRoot}}` | Repository root |
| `{{.GitDir}}` | .git directory |

### Built-in Agents

| Agent | Command Format |
|-------|----------------|
| `claude` | `claude [system-args] [args] "$PROMPT"` |
| `opencode` | `opencode [system-args] [args] -p "$PROMPT"` |
| `codex` | `codex [system-args] [args] "$PROMPT"` |

**Argument Types:**
- `system-args`: System-managed arguments (e.g., `--model`). Set by `--model` flag. Not user-configurable via config.toml.
- `args`: User-customizable arguments. Can be extended via `[agents.<name>].args` in config.toml.

---

## Naming Conventions

### Branch Names

| Pattern | Format |
|---------|--------|
| Without issue | `crew-<id>` |
| With issue | `crew-<id>-gh-<issue>` |

### Session Names

| Format |
|--------|
| `crew-<id>` |

### File Paths

| Resource | Path |
|----------|------|
| Crew directory | `.git/crew/` |
| Task store | `.git/crew/tasks.json` |
| tmux socket | `.git/crew/tmux.sock` |
| tmux config | `.git/crew/tmux.conf` |
| Repository config | `.git/crew/config.toml` |
| Scripts directory | `.git/crew/scripts/` |
| Task script | `.git/crew/scripts/task-<id>.sh` |
| Task prompt | `.git/crew/scripts/task-<id>-prompt.txt` |
| Review script | `.git/crew/scripts/review-<id>.sh` |
| Review prompt | `.git/crew/scripts/review-<id>-prompt.txt` |
| Logs directory | `.git/crew/logs/` |
| Global log | `.git/crew/logs/crew.log` |
| Task log | `.git/crew/logs/task-<id>.log` |

---

## Agent Contract

### Work Agent (Task Execution Agent)

Agent launched by `git crew start`.

**Available Commands**:
- `git crew show` - View task details
- `git crew complete` - Mark as complete
- `git crew review` - Run review
- `git crew comment` - Add comment
- `git crew diff` - View diff

**Required Behaviors**:
1. Run `git crew show` at start to check task details
2. Commit frequently (with clear messages)
3. Run `git crew complete` when done
4. Include "commit changes" and "run git crew complete" as final TODO list steps

**Recommended Behaviors**:
- Run tests after significant changes
- Make small, focused changes

**Prohibited Actions**:
- `git push` (reviewer handles merge and push)
- `git push --force` (dangerous, risk of data loss)

### Manager Agent (Orchestrator Agent)

Agent launched by `git crew manager`.

**Available Commands**:
- `git crew list` - Task list
- `git crew show <id>` - Task details
- `git crew new` - Create task
- `git crew edit` - Edit task
- `git crew start` - Start task
- `git crew peek` - Check session
- `git crew comment` - Add comment

**Constraints**:
- Read-only mode (no file editing)
- Does not write code directly

**Role**:
- Task creation and management
- Progress monitoring
- Review completed work
- Break down large tasks

---

## Dependencies

| Tool | Purpose | Required |
|------|---------|----------|
| git | Version control, worktree | Yes |
| tmux | Session management | Yes |
| gh | GitHub integration | No |
