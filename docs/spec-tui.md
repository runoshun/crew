# git-crew TUI Specification

## Overview

Interactive TUI using Bubbletea. Enables keyboard-based task listing, operations, and creation.

---

## Screen Modes

| Mode | Description |
|------|-------------|
| Normal | Task list display and operations |
| Filter | Task search |
| Confirm | Delete/merge confirmation dialog |
| InputTitle | New task title input |
| InputDesc | New task description input |
| Start | Select agent and launch |
| Help | Key bindings dialog |

---

## Normal Mode

### Screen Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│  git-crew                                                           │
│                                                                     │
│  3 items                                                            │
│                                                                     │
│  > #1 [in_progress] claude  Fix login bug                           │
│    issue:#42 | Fixing authentication issue                          │
│                                                                     │
│    #2 [todo] -  Add user settings page                              │
│    New feature request                                              │
│                                                                     │
│    #3 [in_review] -  Refactor database layer                        │
│    PR:#15 | Performance improvements                                │
│                                                                     │
│                                                                     │
│                                                                     │
│                                                                     │
│                                                                     │
│  Loaded 3 tasks                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Display Elements

| Element | Description |
|---------|-------------|
| Title | `git-crew` |
| Item count | `3 items` |
| Task row | `#<id> [<status>] <agent or -> <title>` |
| Description row | `issue:#N` `PR:#N` `<description>` separated by `\|` |
| Status bar | Last operation result (Loaded N tasks, Error: xxx, etc.) |
| Help | Normal mode hides help line; other modes show context help |

### Status Colors

| Status | Color |
|--------|-------|
| todo | Blue (#74B9FF) |
| in_progress | Yellow (#FDCB6E) |
| in_review | Purple (#A29BFE) |
| done | Green (#00B894) |
| closed | Gray (#636E72) |

### Key Operations

| Key | Operation | Condition | Result |
|-----|-----------|-----------|--------|
| `j` / `↓` | Next task | - | Cursor movement |
| `k` / `↑` | Previous task | - | Cursor movement |
| `Enter` | Smart action | - | Status-dependent operation (see below) |
| `s` | start | - | Go to Start mode |
| `S` | stop | in_progress | Stop session → in_review |
| `a` | attach | Session running | Attach to tmux |
| `n` | new | - | Go to InputTitle mode |
| `y` | copy | - | Copy task (appends "(copy)" to title) |
| `d` | delete | - | Go to Confirm mode |
| `m` | merge | - | Go to Confirm mode |
| `c` | close | - | Set task to closed state |
| `p` | Create PR | - | push + gh pr create |
| `r` | refresh | - | Reload task list |
| `/` | filter | - | Go to Filter mode |
| `?` | Help | - | Go to Help mode |
| `v` | Toggle detail | - | Show/hide detail panel |
| `q` | quit | - | Exit TUI |

### Enter Smart Action

| Condition | Executed Operation |
|-----------|--------------------|
| Session running | attach |
| status = `todo` | Go to Start mode |
| status = `in_progress` (no session) | Go to Start mode |
| status = `in_review` | review (show diff) |
| status = `done` / `closed` | Confirm mode (delete confirmation) |

### Copy (y) Behavior

Copies task to create a new task.

| Item | Behavior |
|------|----------|
| Title | Original title + " (copy)" |
| Description | Copied as-is |
| Status | `todo` (new) |
| Issue | Not inherited |
| PR | Not inherited |
| Worktree | Newly created |

```
Example: Copy #1 "Fix login bug"
     → #4 "Fix login bug (copy)" is created
```

---

## Start Mode

### Screen Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│  git-crew                                                           │
│                                                                     │
│  3 items                                                            │
│                                                                     │
│  > #1 [todo] -  Fix login bug                                       │
│    issue:#42 | Fixing authentication issue                          │
│                                                                     │
│    #2 [todo] -  Add user settings page                              │
│    ...                                                              │
│                                                                     │
│  ╭─ Start task #1 ──────────────────────────────────────────────╮   │
│  │                                                              │   │
│  │  Select agent:                                               │   │
│  │                                                              │   │
│  │  > claude      claude -p "# Task: ..."                       │   │
│  │    opencode    opencode -p "# Task: ..."                     │   │
│  │    codex       codex "{{.Title}}"                            │   │
│  │    ──────────────────────────────────────                    │   │
│  │    vim         vim (custom)                                  │   │
│  │                                                              │   │
│  │  Or type custom command: > _                                 │   │
│  │                                                              │   │
│  ╰──────────────────────────────────────────────────────────────╯   │
│                                                                     │
│  Select agent for task #1: Fix login bug                            │
│  j/k:select | Enter:start | Tab:custom input | Esc:cancel           │
└─────────────────────────────────────────────────────────────────────┘
```

### Display Elements

| Element | Description |
|---------|-------------|
| Dialog title | `Start task #<id>` |
| Agent list | Built-in + config-defined agents |
| Each row | `<name>    <command preview>` |
| Separator | Boundary between built-in and custom |
| Custom input field | Free-form command input |
| Status bar | Target task information |

### Agent List Structure

```
Built-in (BuiltinAgents)
├── claude
├── opencode
└── codex
──────────────────────
Custom (config.toml [agents.xxx])
├── vim
├── my-agent
└── ...
```

### Key Operations

| Key | Operation | Result |
|-----|-----------|--------|
| `j` / `↓` | Next option | Cursor movement |
| `k` / `↑` | Previous option | Cursor movement |
| `Enter` | Confirm selection | Launch with selected agent → attach |
| `Tab` | Focus custom input | Text input mode |
| `Esc` | Cancel | Return to Normal mode |

### Custom Input Key Operations

| Key | Operation | Result |
|-----|-----------|--------|
| (text input) | Command input | Text input |
| `Enter` | Confirm | Launch with entered command → attach |
| `Tab` | Return to selection list | Focus movement |
| `Esc` | Cancel | Return to Normal mode |

### Notes

- Template variables (`{{.Title}}`, etc.) can be used in custom input
- Empty Enter is invalid (nothing happens)
- Auto tmux attach after successful launch

---

## Help Mode

### Screen Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│                                                                     │
│                                                                     │
│                 ╭────────────────────────────────╮                  │
│                 │  Key bindings                 │                  │
│                 │                               │                  │
│                 │  j/down  next task            │                  │
│                 │  k/up    previous task        │                  │
│                 │  ...                          │                  │
│                 │                               │                  │
│                 │  ?:close | Esc:close | q:close │                  │
│                 ╰────────────────────────────────╯                  │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Key Operations

| Key | Operation | Result |
|-----|-----------|--------|
| `?` / `Esc` / `q` | Close | Return to Normal mode |

### Notes

- Open with `?` from Normal mode
- Dialog is centered using the standard dialog box style

---

## Filter Mode

### Screen Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│  git-crew                                                           │
│                                                                     │
│  Filter: login_                                                     │
│                                                                     │
│  > #1 [in_progress] claude  Fix login bug                           │
│    issue:#42 | Fixing authentication issue                          │
│                                                                     │
│                                                                     │
│                                                                     │
│                                                                     │
│                                                                     │
│                                                                     │
│                                                                     │
│                                                                     │
│                                                                     │
│                                                                     │
│  1 item matched                                                     │
│  Type to filter | Enter:select | Esc:clear                          │
└─────────────────────────────────────────────────────────────────────┘
```

### Display Elements

| Element | Description |
|---------|-------------|
| Filter input field | `Filter: <input text>` |
| Filter results | Only matched tasks displayed |
| Status bar | `N item(s) matched` |

### Filter Target

- Task title (partial match, case-insensitive)

### Key Operations

| Key | Operation | Result |
|-----|-----------|--------|
| (text input) | Filter string input | Real-time filtering |
| `Enter` | Confirm filter | To Normal mode (filter maintained) |
| `Esc` | Clear filter | To Normal mode (show all) |
| `↓` / `↑` | Cursor movement | Move within filter results |

### Notes

- Uses bubbles/list built-in filter functionality
- Can move with `j/k` after filter confirmation
- Press `/` again to re-edit filter

---

## Confirm Mode

### Screen Layout (Delete)

```
┌─────────────────────────────────────────────────────────────────────┐
│  git-crew                                                           │
│                                                                     │
│  3 items                                                            │
│                                                                     │
│  > #1 [done] -  Fix login bug                                       │
│    issue:#42 | Fixing authentication issue                          │
│                                                                     │
│    #2 [todo] -  Add user settings page                              │
│    ...                                                              │
│                                                                     │
│                                                                     │
│                                                                     │
│                                                                     │
│                                                                     │
│                                                                     │
│                                                                     │
│  Delete task #1: Fix login bug? [y/N]                               │
│  y:yes | n:no | Esc:cancel                                          │
└─────────────────────────────────────────────────────────────────────┘
```

### Screen Layout (Merge)

```
┌─────────────────────────────────────────────────────────────────────┐
│  git-crew                                                           │
│                                                                     │
│  3 items                                                            │
│                                                                     │
│  > #3 [in_review] -  Refactor database layer                        │
│    PR:#15 | Performance improvements                                │
│                                                                     │
│    ...                                                              │
│                                                                     │
│                                                                     │
│                                                                     │
│                                                                     │
│                                                                     │
│                                                                     │
│                                                                     │
│                                                                     │
│  Merge task #3 (crew-3) to main? [y/N]                              │
│  y:yes | n:no | Esc:cancel                                          │
└─────────────────────────────────────────────────────────────────────┘
```

### Confirmation Dialog Types

| Action | Message |
|--------|---------|
| Delete | `Delete task #<id>: <title>? [y/N]` |
| Merge | `Merge task #<id> (<branch>) to main? [y/N]` |

### Key Operations

| Key | Operation | Result |
|-----|-----------|--------|
| `y` / `Y` | Yes | Execute action → To Normal mode |
| `n` / `N` | No | Cancel → To Normal mode |
| `Esc` | Cancel | To Normal mode |

### Notes

- Delete: Stop session → Remove worktree → Remove git config
- Merge: Stop session → git merge → Remove worktree → status=done

---

## InputTitle Mode

### Screen Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│  git-crew                                                           │
│                                                                     │
│  3 items                                                            │
│                                                                     │
│  > #1 [in_progress] claude  Fix login bug                           │
│    ...                                                              │
│                                                                     │
│                                                                     │
│                                                                     │
│                                                                     │
│                                                                     │
│                                                                     │
│                                                                     │
│  ╭──────────────────────────────────────────────────────────────╮   │
│  │ Title: Add new feature_                                      │   │
│  ╰──────────────────────────────────────────────────────────────╯   │
│                                                                     │
│  Enter task title                                                   │
│  Enter:next | Esc:cancel                                            │
└─────────────────────────────────────────────────────────────────────┘
```

### Display Elements

| Element | Description |
|---------|-------------|
| Input field | `Title: <input text>` |
| Placeholder | `Task title...` (when empty) |
| Status bar | `Enter task title` |

### Key Operations

| Key | Operation | Result |
|-----|-----------|--------|
| (text input) | Title input | Text input |
| `Enter` | Confirm | To InputDesc mode (invalid if title empty) |
| `Esc` | Cancel | To Normal mode |

### Notes

- Title is required (cannot confirm with empty string)
- Max 200 characters

---

## InputDesc Mode

### Screen Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│  git-crew                                                           │
│                                                                     │
│  3 items                                                            │
│                                                                     │
│  > #1 [in_progress] claude  Fix login bug                           │
│    ...                                                              │
│                                                                     │
│  ╭──────────────────────────────────────────────────────────────╮   │
│  │                                                              │   │
│  │  Title: Add new feature                                      │   │
│  │                                                              │   │
│  │  Description:                                                │   │
│  │  ┃ This feature allows users to                              │   │
│  │  ┃ export their data in CSV format._                         │   │
│  │  ┃                                                           │   │
│  │  ┃                                                           │   │
│  │                                                              │   │
│  ╰──────────────────────────────────────────────────────────────╯   │
│                                                                     │
│  Enter description (optional)                                       │
│  Ctrl+D:create | Esc:skip                                           │
└─────────────────────────────────────────────────────────────────────┘
```

### Display Elements

| Element | Description |
|---------|-------------|
| Title display | Title entered in previous screen |
| Description input field | textarea (multi-line support) |
| Placeholder | `Description (optional)...` (when empty) |
| Status bar | `Enter description (optional)` |

### Key Operations

| Key | Operation | Result |
|-----|-----------|--------|
| (text input) | Description input | Text input |
| `Enter` | Newline | Newline within textarea |
| `Ctrl+D` | Confirm | Create task → To Normal mode |
| `Esc` | Skip | Create task without description → To Normal mode |

### Notes

- Description is optional (empty is OK)
- Multi-line input supported (uses textarea)
- Max 1000 characters
- Worktree automatically created after creation

---

## Detail Panel

### Screen Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│  git-crew                                                           │
│                                                                     │
│  3 items                                                            │
│                                                                     │
│  > #1 [in_progress] claude  Fix login bug                           │
│    issue:#42 | Fixing authentication issue                          │
│                                                                     │
│    #2 [todo] -  Add user settings page                              │
│    ...                                                              │
│                                                                     │
│  ╭──────────────────────────────────────────────────────────────╮   │
│  │                                                              │   │
│  │  ID: 1                                                       │   │
│  │  Title: Fix login bug                                        │   │
│  │  Status: in_progress                                         │   │
│  │  Branch: crew-1                                              │   │
│  │  Agent: claude                                               │   │
│  │  Issue: #42                                                  │   │
│  │  Description: Fixing authentication issue when user          │   │
│  │               tries to login with SSO                        │   │
│  │                                                              │   │
│  ╰──────────────────────────────────────────────────────────────╯   │
│  Loaded 3 tasks                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Display Elements

| Field | Display Condition |
|-------|-------------------|
| ID | Always displayed |
| Title | Always displayed |
| Status | Always displayed |
| Branch | Always displayed (`crew-<id>`) |
| Agent | Only when session running |
| Session | Only when session running (optional) |
| Issue | Only when issue linked |
| PR | Only when PR created |
| Description | Only when description exists |

### Key Operations

| Key | Operation | Result |
|-----|-----------|--------|
| `v` | Toggle | Show/hide detail panel |
| (others) | Normal operations | Normal mode key operations active |

### Notes

- Toggle display with `v` during Normal mode
- Normal key operations work while panel displayed
- Selected task details update on cursor movement

---

## Screen Transition Diagram

```
                    ┌─────────┐
                    │ Normal  │◄──────────────────────────┐
                    └────┬────┘                           │
                         │                                │
         ┌───────┬───────┼───────┬───────┬───────┬───────┐       │
         │       │       │       │       │       │       │       │
         ▼       ▼       ▼       ▼       ▼       ▼       ▼       │
      ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐   │
      │ / → │ │ s → │ │ n → │ │d/m→ │ │ v → │ │ ? → │ │Enter│   │
      │Filtr│ │Start│ │Title│ │Cnfrm│ │Detil│ │ Help│ │  │  │   │
      └──┬──┘ └──┬──┘ └──┬──┘ └──┬──┘ └──┬──┘ └──┬──┘ └──┼──┘   │
         │       │       │       │       │       │       │       │
         │       │       ▼       │       │       │       │       │
         │       │  ┌─────────┐  │       │       │       │       │
         │       │  │InputDesc│  │       │       │       │       │
         │       │  └────┬────┘  │       │       │       │
         │       │       │       │       │       │       │
         └───────┴───────┴───────┴───────┴───────┴───────┘
                              │
                              ▼
                        ┌──────────┐
                        │ tmux     │ (attach/review)
                        │ external │
                        └────┬─────┘
                             │
                             ▼
                        (TUI resumes)
```

---

## Error Display

### Status Bar on Error

```
  Error: task 1 not found
  s:start | S:stop | a:attach | ...
```

### Error Types

| Error | Example Message |
|-------|-----------------|
| Task not found | `Error: task <id> not found` |
| No session | `Error: no running session` |
| Worktree not found | `Error: worktree not found` |
| Merge failed | `Error: merge failed: <details>` |
| PR creation failed | `Error: failed to create PR: <details>` |
| gh not installed | `Error: gh command not available` |

### Notes

- Errors displayed in red in status bar
- Cleared on next operation
- Non-fatal errors allow TUI to continue
