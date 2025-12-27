# git-crew Core Concepts

## 1. Fundamental Principle

### 1 Task = 1 Worktree = 1 Session

The core design principle of git-crew:

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

#### Overview

Git worktree allows creating multiple working directories from a single repository.
git-crew uses this to provide isolated working environments for each task.

#### Directory Structure

```
/path/to/repo/                    # Main repository
├── .git/
│   ├── config                    # Git configuration
│   ├── worktrees/                # Worktree references (managed by git)
│   │   ├── crew-1/
│   │   └── crew-2/
│   └── crew/                     # git-crew dedicated directory
│       ├── tasks.json            # Task store
│       ├── config.toml           # Configuration
│       ├── tmux.sock             # tmux socket
│       ├── tmux.conf             # tmux configuration
│       ├── scripts/              # Task scripts
│       └── logs/                 # Log files
└── src/                          # Working files

/path/to/repo-worktrees/          # Worktree directory (sibling of main repo)
├── crew-1/                       # Worktree for task 1
│   ├── .git -> ../../repo/.git/worktrees/crew-1
│   └── src/
├── crew-2/                       # Worktree for task 2
│   └── ...
└── crew-3-gh-123/                # Task linked to GitHub issue #123
    └── ...
```

#### Worktree Management

git-crew manages worktrees using native `git worktree` commands.

**Main Operations**:

| Operation | Git Command | Description |
|-----------|-------------|-------------|
| Create | `git worktree add <path> <branch>` | Create a worktree for the branch |
| Remove | `git worktree remove <path>` | Remove the worktree |
| List | `git worktree list --porcelain` | List all worktrees |

**Worktree Placement**:
- Location: `<repo>-worktrees/<branch>/`
- Worktrees are created in a sibling directory of the main repository

#### Branch Management

**Branch Naming Convention**:
- Basic format: `crew-<taskID>`
- With issue: `crew-<taskID>-gh-<issueNumber>`

**Branch Lifecycle**:

```
1. Task Creation
   - Branch is NOT created (lazy creation)

2. First Start
   - Create new branch from base branch (default: main)
   - Create worktree

3. Subsequent Starts
   - Use existing branch/worktree

4. Successful Merge
   - Merge into main
   - Delete worktree
   - Delete branch

5. Close
   - Delete worktree
   - Branch is NOT deleted (cleaned up by prune)
```

**Base Branch**:
- Default: `main`
- When copied with `git crew cp`: source task's branch
- Stored in `task.BaseBranch`, used at start time

---

### 2.2 tmux Session Management

#### Overview

tmux is a terminal multiplexer. git-crew uses it to:
- Run AI agents in the background
- Manage multiple sessions simultaneously
- Provide flexible attach/detach operations

#### Socket Isolation

git-crew uses a dedicated socket to isolate from system tmux:

```
Socket path: .git/crew/tmux.sock
```

**Benefits of Isolation**:
- No interference with existing tmux sessions
- Can list/operate only git-crew sessions
- Independent session space per repository

#### tmux Configuration

`.git/crew/tmux.conf`:

```
unbind-key -a              # Unbind all keys
bind-key -n C-g detach-client  # Ctrl+G to detach
set -g status off          # Hide status bar
set -g escape-time 0       # No escape delay
```

**Design Intent**:
- Minimal keybindings (detach only)
- Don't interfere with AI agent key inputs
- Simple UI (no status bar)

#### Session Operations

| Operation | tmux Command |
|-----------|--------------|
| List | `tmux -S <socket> list-sessions` |
| Check exists | `tmux -S <socket> has-session -t <name>` |
| Create | `tmux -S <socket> new-session -d -s <name> -c <dir> <cmd>` |
| Kill | `tmux -S <socket> kill-session -t <name>` |
| Attach | `tmux -S <socket> attach -t <name>` |
| Capture | `tmux -S <socket> capture-pane -t <name> -p` |
| Send keys | `tmux -S <socket> send-keys -t <name> <keys>` |

---

### 2.3 Process Management and Session Termination Detection

#### Task Script

A wrapper script generated when each session starts:

**Path**: `.git/crew/scripts/task-<id>.sh`

**Structure**:

```bash
#!/bin/bash
set -o pipefail

# Read prompt from separate file (avoids escaping issues)
PROMPT=$(cat ".git/crew/scripts/task-<id>-prompt.txt")

# Callback on session termination
SESSION_ENDED() {
  local code=$?
  git crew _session-ended <task-id> "$code" || true
}

# Signal handling
trap SESSION_ENDED EXIT    # Both normal and abnormal exit
trap 'exit 130' INT        # Ctrl+C → exit code 130
trap 'exit 143' TERM       # kill → exit code 143
trap 'exit 129' HUP        # hangup → exit code 129

# Run agent
<agent-command> "$PROMPT"
```

#### Session Termination Detection Mechanism

```
┌─────────────────┐
│ tmux session    │
│ ┌─────────────┐ │
│ │ task-N.sh   │ │ ─── trap EXIT ───┐
│ │ └─ agent    │ │                  │
│ └─────────────┘ │                  ▼
└─────────────────┘     git crew _session-ended <id> <code>
                                     │
                                     ▼
                        ┌────────────────────────┐
                        │ Termination handling:  │
                        │ - Clear agent info     │
                        │ - Delete script        │
                        │ - Update status        │
                        └────────────────────────┘
```

**Processing by Exit Code**:

| Exit Code | Meaning | Status Transition |
|-----------|---------|-------------------|
| 0 | Normal exit | Maintain (keep `in_review` if already) |
| Non-zero | Abnormal exit | `error` |
| 130 | Ctrl+C | `error` |
| 143 | SIGTERM | `error` |

#### Race Condition Prevention

`_session-ended` is ignored when:
- Task's agent info is already cleared

**Scenario**:
1. User executes `git crew stop`
2. `stop` clears agent info
3. tmux session terminates
4. `_session-ended` is called but ignored (no agent info)

This prevents conflicts between stop and EXIT trap.

---

### 2.4 Data Store

#### JSON Store

**File**: `.git/crew/tasks.json`

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
      "started": "2024-01-01T01:00:00Z",
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
      "status": "in_progress",
      "created": "2024-01-01T01:00:00Z",
      "started": "2024-01-01T01:30:00Z",
      "agent": "claude",
      "session": "crew-2",
      "issue": 101,
      "pr": 0,
      "baseBranch": "main",
      "labels": []
    }
  },
  "comments": {
    "1": [
      {
        "text": "Started planning",
        "time": "2024-01-01T01:30:00Z"
      }
    ]
  }
}
```

**Design**:
- Task ID is a string key (JSON constraint)
- `meta.nextTaskID` is a monotonic counter (no reuse)
- `parentID` is null for root tasks, integer for sub-tasks
- `comments` is an array keyed by task ID

#### File Locking

Concurrent access control:
- Read: shared lock
- Write: exclusive lock
- Lock file: `.git/crew/tasks.json.lock`

---

## 3. Task Hierarchy

### Parent-Child Relationship

Tasks can have parent-child relationships to organize large work into smaller pieces while maintaining visibility of the overall goal.

```
#1 Auth refactoring (parent)
├── #2 OAuth2.0 implementation
│   ├── #5 Google OAuth
│   └── #6 GitHub OAuth
├── #3 Session management
└── #4 Token refresh

#7 Fix typo (standalone)
```

### Rules

| Rule | Description |
|------|-------------|
| Unlimited nesting | Sub-tasks can have their own sub-tasks |
| Both can have worktrees | Parent and child tasks can both have worktrees |
| Manual status | Parent status is managed manually (not auto-calculated) |
| GitHub compatible | Aligns with GitHub Sub-issues feature |

### Use Cases

| Scenario | Structure |
|----------|-----------|
| Large feature | Parent task for overview, sub-tasks for implementation |
| Bug fix | Single standalone task |
| Refactoring | Parent task for goal, sub-tasks for each component |

### Commands

```bash
# Create parent task
git crew new --title "Auth refactoring" --desc "## Goal\n..."

# Create sub-task
git crew new --parent 1 --title "OAuth2.0 implementation"

# Create nested sub-task
git crew new --parent 2 --title "Google OAuth"

# List all tasks (flat view with PARENT column)
git crew list

# Show task with sub-tasks
git crew show 1

# Import GitHub issue as sub-task
git crew import 123 --parent 1
```

---

## 4. Task Lifecycle

### Status Definitions

| Status | Description | Session |
|--------|-------------|---------|
| `todo` | Created, not started | None |
| `in_progress` | Work in progress | Yes (normally) |
| `in_review` | Work complete, awaiting review | None |
| `error` | Session terminated abnormally | None |
| `done` | Merge complete | None |
| `closed` | Discarded without merge | None |


### Transition Rules

| Current | → | Target | Trigger |
|---------|---|--------|---------|
| - | → | `todo` | `new` |
| `todo` | → | `in_progress` | `start` |
| `todo` | → | `closed` | `close` |
| `in_progress` | → | `in_review` | `complete`, `stop` |
| `in_progress` | → | `error` | Abnormal session exit |
| `in_progress` | → | `closed` | `close` |
| `in_review` | → | `in_progress` | `start` (resume) |
| `in_review` | → | `done` | `merge` |
| `in_review` | → | `closed` | `close` |
| `error` | → | `in_progress` | `start` (retry) |
| `error` | → | `closed` | `close` |
| `done` | → | `closed` | `close` |

---

## 5. Agent Contract

### Work Agent

Task execution agent launched by `git crew start`.

#### Available Commands

| Command | Purpose |
|---------|---------|
| `git crew show` | View task details |
| `git crew complete` | Mark as complete |
| `git crew comment` | Add comment |
| `git crew diff` | View diff |
| `git crew review` | Run review |

#### Expected Behaviors

1. **On start**: Run `git crew show` to check task details
2. **During work**: Commit frequently (with clear messages)
3. **On completion**: Run `git crew complete`
4. **TODO list**: Include "commit" and "complete" as final steps

#### Prohibited Actions

| Action | Reason |
|--------|--------|
| `git push` | Reviewer handles push |
| `git push --force` | Risk of data loss |

### Manager Agent

Read-only agent launched by `git crew manager`.

#### Available Commands

| Command | Purpose |
|---------|---------|
| `git crew list` | Task list |
| `git crew show <id>` | Task details |
| `git crew new` | Create task |
| `git crew edit` | Edit task |
| `git crew start` | Start task |
| `git crew peek` | Check session |
| `git crew comment` | Add comment |

#### Constraints

- No file editing (read-only mode)
- Does not write code directly

---

## 6. Configuration System

### Priority Order

1. Repository config: `.git/crew/config.toml`
2. User config: `~/.config/git-crew/config.toml`
3. Built-in defaults

Later settings override (merge with) earlier settings.

### Main Settings

```toml
# Default agent
default_agent = "claude"

# Common prompt (appended to all agents)
[agent]
prompt = "When complete, run 'git crew complete'."

# Log level
[log]
level = "info"

# Agent settings
[agents.claude]
args = "--model claude-sonnet-4-20250514"

[agents.opencode]
args = "-m gpt-4"

# Custom agent
[agents.my-agent]
command = 'my-agent "{{.Title}}"'

# CI gate on completion
[complete]
command = "mise run ci"

# Diff display
[diff]
command = "git diff main...HEAD | delta"
tui_command = "git diff --color main...HEAD | less -R"

# Review settings
[review]
agent = "claude"
prompt = "Review for bugs and quality"

# GitHub integration
[github]
pr_body = "## {{.Title}}\n\n{{.Description}}"

# Manager settings
[manager]
default_agent = "claude"
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
| `{{.Args}}` | Agent additional args |
| `{{.RepoRoot}}` | Repository root |
| `{{.GitDir}}` | .git directory |

---

## 7. Naming Conventions

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

## 8. Dependencies

| Tool | Purpose | Required |
|------|---------|----------|
| git | Version control, worktree | Yes |
| tmux | Session management | Yes |
| gh | GitHub integration | No |

Worktree management is implemented natively using `git worktree` commands (no external tools required).

---

## 9. Security Considerations

### File Permissions

| File | Permission | Reason |
|------|------------|--------|
| `tasks.json` | 0644 | Data file |
| `config.toml` | 0644 | Config file |
| `tmux.sock` | 0600 | Socket (user only) |
| `scripts/*.sh` | 0755 | Executable scripts |
| `logs/*.log` | 0644 | Log files |

### Prompt Handling

Prompts are saved to a separate file and read via shell:

```bash
PROMPT=$(cat "task-1-prompt.txt")
agent "$PROMPT"
```

**Reasons**:
- Avoid shell escaping issues
- Handle arbitrary prompt content
- Prevent injection attacks
