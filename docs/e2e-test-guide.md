# E2E Test Guide

Manual E2E testing procedures for verifying functionality that automated tests cannot cover.

## Prerequisites

- Binary built with `mise run build`
- Each CLI (claude, opencode, codex) installed

---

## Environment Setup

### Automatic Setup

```bash
# Create test repository (created in .e2e-test/, gitignored)
./scripts/e2e-setup.sh

# Or create in a custom directory
./scripts/e2e-setup.sh /path/to/test-dir
```

### Manual Setup

```bash
# 1. Create test directory (inside repository, gitignored)
TEST_DIR=.e2e-test
mkdir -p $TEST_DIR && cd $TEST_DIR

# 2. Initialize git repository
git init
git config user.email "test@example.com"
git config user.name "E2E Test"

# 3. Create initial files
echo "# E2E Test" > README.md
git add . && git commit -m "Initial commit"

# 4. Initialize crew
crew init
```

### Cleanup

```bash
# After testing is complete
rm -rf .e2e-test
```

**Note:** `.e2e-test/` is already added to the main repository's `.gitignore`. Placing it inside the repository makes worktree access easier due to agent permission settings.

---

## 1. Status Transition Tests

Verify that each agent transitions status correctly.

### 1.1 Claude (cc-medium) Status Transitions

**Expected Behavior:**
- On start: `todo` → `in_progress`
- On idle/permission request: `in_progress` → `needs_input`
- After user input: `needs_input` → `in_progress`
- After `crew complete`: `in_progress` → `in_review`

**Steps:**

```bash
# 1. Create test task
crew new --title "E2E: Claude status transition test"

# 2. Start task
crew start <id> cc-medium

# 3. Check status (should be in_progress)
crew list | grep <id>

# 4. Wait for agent to become idle
sleep 30

# 5. Check status (should be needs_input)
crew list | grep <id>

# 6. Send some input
crew send <id> "echo test"
crew send <id> Enter

# 7. Check status (should return to in_progress)
sleep 5
crew list | grep <id>

# 8. Send completion instruction
crew send <id> "crew complete <id>"
crew send <id> Enter
# Allow permission
crew send <id> y
crew send <id> Enter

# 9. Check status (should be in_review)
sleep 10
crew list | grep <id>

# 10. Cleanup
crew close <id>
```

**Verification Points:**
- [ ] Transitions to `in_progress` after start
- [ ] Transitions to `needs_input` when idle
- [ ] Returns to `in_progress` after input
- [ ] Transitions to `in_review` after complete

---

### 1.2 OpenCode (oc-medium-ag) Status Transitions

**Expected Behavior:** Same as Claude

**Steps:**

```bash
# 1. Create test task
crew new --title "E2E: OpenCode status transition test"

# 2. Start task
crew start <id> oc-medium-ag

# 3-9. Follow the same steps as Claude

# 10. Cleanup
crew close <id>
```

**Verification Points:**
- [ ] TypeScript plugin loads correctly
- [ ] Status transitions on each event

---

### 1.3 Codex Status Transitions

**Expected Behavior:**
- On `agent-turn-complete` event: `in_progress` → `needs_input`

**Steps:**

```bash
# 1. Create test task
crew new --title "E2E: Codex status transition test"

# 2. Start task
crew start <id> codex

# 3. Check status
crew list | grep <id>

# 4. Wait for agent to complete turn (prompt display, etc.)
# codex only has notify for agent-turn-complete,
# so idle detection is limited

# 5. Check status
crew list | grep <id>

# 6. Cleanup
crew close <id>
```

**Verification Points:**
- [ ] notify configuration is parsed correctly
- [ ] Status transitions on turn completion

**Known Limitations:**
- codex notify only supports `agent-turn-complete`
- Immediate idle state detection is not possible

---

## 2. TUI Operation Tests

### 2.1 Basic Operations

**Steps:**

```bash
# 1. Launch TUI
crew

# 2. Test operations
# - j/k: Select task
# - Enter: Show details
# - s: Start task
# - p: peek (view session output)
# - a: attach (attach to session)
# - d: Show diff
# - ?: Show help
# - q: Quit
```

**Verification Points:**
- [ ] Key bindings work correctly
- [ ] Status display updates in real-time
- [ ] Error messages display appropriately

---

### 2.2 peek/attach

**Steps:**

```bash
# 1. Start task
crew start <id> cc-small

# 2. peek from TUI
crew
# Press p key to peek

# 3. Verify output is displayed

# 4. attach
# Press a key to attach
# Ctrl+b d to detach
```

**Verification Points:**
- [ ] peek displays latest output
- [ ] attach connects to session
- [ ] detach returns to TUI

---

### 2.3 send

**Steps:**

```bash
# 1. Start task (in needs_input state)
crew start <id> cc-small

# 2. Wait for needs_input
sleep 30

# 3. Send keys
crew send <id> "ls -la"
crew send <id> Enter

# 4. Verify with peek
crew peek <id>
```

**Verification Points:**
- [ ] String is sent correctly
- [ ] Enter is sent
- [ ] Special keys (Ctrl+C, etc.) are sent

---

## 3. Workflow Tests

### 3.1 Task Completion Flow

```bash
# 1. Create task
crew new --title "E2E: Completion flow test" --body "Please run echo hello"

# 2. Start
crew start <id> cc-small

# 3. Wait for completion (or manually run crew complete)

# 4. Review
crew review <id>

# 5. Merge
echo "y" | crew merge <id>

# 6. Verify
crew show <id>  # status: done
```

**Verification Points:**
- [ ] Transitions todo → in_progress → in_review → done
- [ ] worktree is deleted
- [ ] Merged to main

---

### 3.2 Review & Revision Flow

```bash
# 1. Create task, start, complete
crew new --title "E2E: Review revision flow test"
crew start <id> cc-small
# ... proceed to in_review

# 2. Review (request changes)
crew review <id>

# 3. Send comment
crew comment <id> -R "Please fix: XXX"

# 4. Check status (should return to in_progress)
crew list | grep <id>

# 5. Proceed to in_review again

# 6. Re-review and merge
crew review <id>
echo "y" | crew merge <id>
```

**Verification Points:**
- [ ] comment -R returns to in_progress
- [ ] Agent reads comment and makes fixes

---

## 4. Error Case Tests

### 4.1 Non-existent Task

```bash
crew show 99999
# Check if error message is appropriate
```

### 4.2 Non-existent Agent

```bash
crew start <id> nonexistent-agent
# Check if error message is appropriate
```

### 4.3 Already Started Task

```bash
crew start <id> cc-small
crew start <id> cc-small  # Second time
# Check for error or appropriate behavior
```

---

## 5. poll Command Tests

```bash
# 1. Create and start task
crew new --title "E2E: poll test"
crew start <id> cc-small

# 2. poll in background
crew poll <id> --timeout 120 &

# 3. Verify status change detection
# Should output when needs_input is reached

# 4. Stop poll after verification
kill %1
```

**Verification Points:**
- [ ] Outputs on status change
- [ ] Exits on timeout
- [ ] Exits on terminal state

---

## Checklist

### Status Transitions
- [ ] Claude: in_progress → needs_input → in_progress → in_review
- [ ] OpenCode: Same as above
- [ ] Codex: Status change via notify

### TUI
- [ ] Basic operations (j/k/Enter/q)
- [ ] peek/attach/detach
- [ ] send

### Workflow
- [ ] Completion flow (todo → done)
- [ ] Review revision flow

### Error Cases
- [ ] Access to non-existent resources
- [ ] Duplicate operations

### poll
- [ ] Status change detection
- [ ] Timeout and exit conditions
