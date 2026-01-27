# git-crew Worker Guide

## IMPORTANT: Follow This Workflow

1. **Read task**: `crew show` - Understand the requirements
2. **Implement**: Make changes following the task description
3. **Test**: Run CI - Ensure all tests pass
4. **Commit**: `git add && git commit`
5. **Complete**: `crew complete` - Mark task as done

DO NOT skip any steps. Run CI before completing.

---

## Basic Operations

### Check Task Information
```bash
crew show              # Show current task details
```

### Report Progress
```bash
crew comment <id> "<message>"   # Add a comment
```

Add comments in these situations:
- When you need user input or clarification on important decisions
- When you encounter blockers or unexpected issues
- When the task is complete (summary of what was done)

### Complete Task
```bash
crew complete          # Mark task as complete
```

---

## Review Mode

`[complete].review_mode` controls how reviews run on completion:

- **auto** (default): Start review asynchronously in the background
- **manual**: Set status to `for_review` without starting review automatically
- **auto_fix**: Run review synchronously and output the result

### Workflow with auto_fix

```bash
# 1. Make changes and commit
git add . && git commit -m "feat: ..."

# 2. Run crew complete
crew complete
# → If LGTM: Done!
# → If not LGTM: Review feedback is shown

# 3. If feedback received, fix issues and repeat
git add . && git commit -m "fix: address review feedback"
crew complete
# → Loop until LGTM or max retries reached
```

### Configuration

```toml
[complete]
command = "mise run ci"
review_mode = "auto_fix"  # auto, manual, or auto_fix
auto_fix_max_retries = 3  # Maximum retry attempts (default: 3)
```

---

## Resolving Conflicts

When conflicts occur with main branch:

```bash
# Fetch and merge main (use main, NOT origin/main)
git fetch origin main:main
git merge main

# After resolving conflicts
git add <files>
git commit
```

**Important**: Use `main`, not `origin/main`.

---

## CI Tests

Run the project's CI checks after making changes. The specific CI command depends on your project configuration.

Check your project's **CLAUDE.md** or **AGENTS.md** for the exact CI command to use.

---

## Typical Workflow

```bash
# 1. Check task information
crew show

# 2. Implement changes

# 3. Run CI (see "CI Tests" section for your project's specific command)

# 4. Commit changes: DO NOT USE git -C <dir> <cmd> form, USE git <cmd> simply.
git add <files>
git commit -m "feat: ..."

# 5. Complete task
crew complete
```

---

## Available Commands

| Command | Description |
|---------|-------------|
| `crew show` | Show task details |
| `crew complete` | Mark task as complete |
| `crew comment` | Add a comment |
| `crew diff` | Show diff |

---

## Agent Configuration Tips

- Repository owners can set per-worker defaults in `.crew/config.toml` using the `model = "<provider/model>"` field.
- `crew start -m <model>` overrides the config value when a specific model is required.

---

## Prohibited Actions

| Action | Reason |
|--------|--------|
| `git push` | Reviewer pushes when merging |
| `git push --force` | Risk of data loss |
