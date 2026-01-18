# Auto Mode

Trigger: Say **"auto mode"** to activate this workflow for managing multiple in-progress tasks.

## Purpose

Advance all in-progress tasks automatically with minimal human intervention. Continue looping until:
- All tasks reach ✅ **LGTM** status, OR
- ⚠️ An undecidable problem occurs (stop and report to user)

## Important: No Interaction During Auto Mode

- **NO confirmations/questions until exit conditions are met** (y/n, selections, etc.)
- Manager makes autonomous decisions about:
  - When to run `crew review <id>`
  - When to execute `crew comment -R`
  - Whether to approve `needs_input` requests
- **Exception**: Only stop and report when encountering undecidable situations

## Target Statuses

- **In scope**: `in_progress`, `needs_input`, `for_review`, `reviewing`, `reviewed`
- **Out of scope**: `todo` (requires explicit start instruction)

---

## 1. Initialization

```bash
crew list --status in_progress,needs_input,for_review,reviewing,reviewed
```

If no tasks in scope, exit with message: "No target tasks found."

---

## 2. Main Loop

**Re-fetch status with `crew list` at the start of each iteration.**

### Priority Order

| Priority | Status | Action |
|----------|--------|--------|
| 1 | `for_review` | Start review |
| 2 | `reviewed` | Process review result |
| 3 | `needs_input` | Evaluate input request |
| 4 | `reviewing` | Skip (wait) |
| 5 | `in_progress` | Check progress only |

---

## 3. Status Handlers

### 3.1 `for_review`

```bash
crew review <id>
```

Status changes to `reviewing` → then `reviewed` when complete → handled in next loop by 3.2.

---

### 3.2 `reviewed`

```bash
crew show <id> --last-review
```

**Evaluation:** Manager reads the review content and judges whether it's LGTM with no issues.

**If LGTM:**
- Add to completion list (exclude from loop)

**If issues found:**
```bash
crew comment <id> -R "Please check the review comments and address them."
```

---

### 3.3 `needs_input`

```bash
crew peek <id> -n 50
```

**Evaluation criteria:**

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

---

### 3.4 `reviewing`

- No action (prioritize other tasks)
- Add to warning list if exceeds 10 minutes

---

### 3.5 `in_progress`

```bash
crew peek <id> -n 30
```

**Stall detection:**
- Same output 3 times in a row → add to warning list
- Error message detected → add to problem list

---

## 4. Wait Handling

When all tasks are only `in_progress` or `reviewing`:

```bash
sleep 60
crew list --status in_progress,needs_input,for_review,reviewing,reviewed
```

Return to loop start.

---

## 5. Loop Exit Conditions

Exit when any of the following:
- All target tasks reach ✅ LGTM
- ⚠️ Undecidable problem occurs
- Fatal error occurs

---

## 6. Final Report

```
## Auto Mode Complete

### ✅ LGTM (Ready to merge)
- #101: feat: add login button
- #103: fix: validation error

### ⚠️ Needs Attention
- #102: needs_input with undecidable request → manual review required

### 📊 Statistics
- Duration: 15 min
- Reviews executed: 4
- Feedback sent: 2

Merge? (y: all / specify IDs / n: skip)
```
