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
- After completion: `in_progress` → `done`
- After merge: `done` → `merged`

**Steps:**

```bash
# 1. Create test task
crew new --title "E2E: Claude status transition test"

# 2. Start task
crew start <id> cc-medium

# 3. Check status (should be in_progress)
crew list | grep <id>

# 4. Run review(s) until requirement is satisfied
crew review <id>

# 5. Complete task (should move to done)
crew complete <id>

# 6. Merge task (should move to merged)
echo "y" | crew merge <id>

# 7. Check status (should be merged)
crew show <id>
```

**Verification Points:**
- [ ] Transitions to `in_progress` after start
- [ ] Transitions to `done` after complete
- [ ] Transitions to `merged` after merge

---

### 1.2 OpenCode (oc-medium-ag) Status Transitions

**Expected Behavior:** Same as Claude (todo → in_progress → done → merged)

**Steps:**

```bash
# 1. Create test task
crew new --title "E2E: OpenCode status transition test"

# 2. Start task
crew start <id> oc-medium-ag

# 3-7. Follow the same steps as Claude
```

**Verification Points:**
- [ ] TypeScript plugin loads correctly
- [ ] Status transitions on each step

---

### 1.3 Codex Status Transitions

**Expected Behavior:**
- On start: `todo` → `in_progress`
- After completion: `in_progress` → `done`
- After merge: `done` → `merged`

**Steps:**

```bash
# 1. Create test task
crew new --title "E2E: Codex status transition test"

# 2. Start task
crew start <id> codex

# 3. Check status (should be in_progress)
crew list | grep <id>

# 4. Run review(s) until requirement is satisfied
crew review <id>

# 5. Complete task (should move to done)
crew complete <id>

# 6. Merge task (should move to merged)
echo "y" | crew merge <id>

# 7. Check status (should be merged)
crew show <id>
```

**Verification Points:**
- [ ] notify configuration is parsed correctly
- [ ] Status transitions on completion and merge

**Known Limitations:**
- codex notify only supports `agent-turn-complete`
- Status does not change mid-turn; it remains `in_progress` until completion

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
# 1. Start task (in_progress)
crew start <id> cc-small

# 2. Wait for input request (status stays in_progress)
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

# 3. Set min_reviews=2 for this test (restore after test)
cp .crew/config.toml /tmp/crew-config.toml
cat >> .crew/config.toml << 'EOF'
[complete]
min_reviews = 2
EOF

# 4. Review (repeat until min_reviews is satisfied)
# Note: only successful (exit code 0) reviews increment the count
crew review <id>

# 5. Complete (should fail until min_reviews is satisfied)
crew complete <id>

# 6. Repeat review/complete until it succeeds
crew review <id>
crew complete <id>

# 7. Merge
echo "y" | crew merge <id>

# 8. Verify
crew show <id>  # status: merged

# 9. Restore config
mv /tmp/crew-config.toml .crew/config.toml
```

**Verification Points:**
- [ ] Transitions todo → in_progress → done → merged
- [ ] `crew complete` fails until `min_reviews` is satisfied
- [ ] worktree is deleted
- [ ] Merged to main

---

### 3.2 Review & Revision Flow

```bash
# 1. Create task and start
crew new --title "E2E: Review revision flow test"
crew start <id> cc-small

# 2. Review (request changes)
crew review <id>

# 3. Send comment
crew comment <id> -R "Please fix: XXX"

# 4. Check status (should remain in_progress)
crew list | grep <id>

# 5. Re-review after fixes
crew review <id>

# 6. Complete and merge
crew complete <id>
echo "y" | crew merge <id>
```

**Verification Points:**
- [ ] comment -R keeps status as in_progress
- [ ] Agent reads comment and makes fixes
- [ ] Review count increments and completion succeeds after requirements are met

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

# 2. poll in background (detects first change and exits)
# Use --expect (required) to specify current status
crew poll <id> --expect todo --command 'echo "Change detected!"' &

# 3. Verify status change detection
# Should output when status changes from todo (to in_progress) and process should exit

# 4. Verify timeout behavior (optional)
# Default timeout is 300s. Can be shortened for testing:
# crew poll <id> --expect todo --timeout 5
```

**Verification Points:**
- [ ] Exits immediately after first status change
- [ ] Executes command on change
- [ ] Exits on timeout (default: 300s)
- [ ] Exits if current status differs from --expect on startup

---

## Checklist

### Status Transitions
- [ ] Claude: todo → in_progress → done → merged
- [ ] OpenCode: Same as above
- [ ] Codex: Same as above

### TUI
- [ ] Basic operations (j/k/Enter/q)
- [ ] peek/attach/detach
- [ ] send

### Workflow
- [ ] Completion flow (todo → done → merged)
- [ ] Review revision flow
- [ ] `crew complete` blocked until `min_reviews` is satisfied

### Error Cases
- [ ] Access to non-existent resources
- [ ] Duplicate operations

### poll
- [ ] Status change detection
- [ ] Timeout and exit conditions
