# Auto Mode

Trigger: Say **"auto mode"** to activate this workflow for managing multiple in-progress tasks.

## Purpose

Advance all in-progress tasks automatically with minimal human intervention. Continue looping until:
- All tasks reach ✅ **LGTM** status, OR
- ⚠️ An undecidable problem occurs (stop and report to user)

## Important: No Interaction During Auto Mode

- **NO confirmations/questions until exit conditions are met** (y/n, selections, etc.)
- Manager makes autonomous decisions about:
  - When to execute `crew comment -R`
  - Whether to approve permission/input requests
- **Exception**: Only stop and report when encountering undecidable situations

## Task Flow (Expected)

```
in_progress → run `crew review --wait`
           → if LGTM: `crew complete` → done
done → ready to merge
```

Managers run `crew review --wait` and `crew complete` when a task is ready. Review does not change status.

## Target Statuses

| Status | In Scope | Notes |
|--------|----------|-------|
| `in_progress` | ✅ | Worker is working or waiting for input |
| `done` | ✅ | Ready to merge or close |
| `error` | ✅ | Problem occurred, report to user |
| `todo` | ❌ | Requires explicit start instruction |
| `merged` | ❌ | Terminal |
| `closed` | ❌ | Terminal |

---

## 1. Initialization

```bash
crew list
```

Filter for in-scope statuses. If no tasks in scope, exit with message: "No target tasks found."

---

## 2. Main Loop

**Re-fetch status with `crew list` at the start of each iteration.**

### Priority Order

| Priority | Status | Action |
|----------|--------|--------|
| 1 | `error` | Add to problem list |
| 2 | `done` | Add to completion list |
| 3 | `in_progress` | Evaluate and review |

---

## 3. Status Handlers

### 3.1 `error`

- Add to problem list immediately
- Include in final report for user attention

---

### 3.2 `done`

```bash
crew show <id> --last-review
```

- Add to completion list (ready to merge or close)
- If issues are found, send feedback and return the task to the loop:
  ```bash
  crew comment <id> -R "Please check the review comments and address them."
  ```

---

### 3.3 `in_progress`

```bash
crew peek <id> -n 50
```

**Input/permission request evaluation:**

| Approve (auto-continue) | Deny (flag as problem) |
|-------------------------|------------------------|
| Worktree read/build/test | Persistent host system changes |
| `mise run ci`, `go test`, `git diff`, `git status` | Out-of-scope worktree operations |
| Branch operations within task scope | Out-of-task file changes |
| | Destructive ops (`rm -rf`, `git reset --hard`, force push) |
| | Security risks (credentials, etc.) |

**If approved:**
```bash
crew send <id> "y"
crew send <id> Enter
```

**If undecidable:**
- Add to problem list and exit loop
- Delegate decision to user

**Review step (sync):**
```bash
crew review <id> --wait
```

**If LGTM:**
```bash
crew complete <id>
```

**If issues found:**
```bash
crew comment <id> -R "Please check the review comments and address them."
```

---

## 4. Wait Handling

When all tasks are only `in_progress`:

```bash
sleep 60
crew list
```

Return to loop start.

---

## 5. Loop Exit Conditions

Exit when any of the following:
- All target tasks reach ✅ `done` (ready to merge)
- ⚠️ Undecidable problem occurs
- Fatal error occurs

---

## 6. Final Report

```
## Auto Mode Complete

### ✅ Done (Ready to merge)
- #101: feat: add login button
- #103: fix: validation error

### ⚠️ Needs Attention
- #102: in_progress with undecidable request → manual review required
- #105: error status → session terminated unexpectedly

### 📊 Statistics
- Duration: 15 min
- Reviews evaluated: 4
- Feedback sent: 2

Merge? (y: all / specify IDs / n: skip)
```
