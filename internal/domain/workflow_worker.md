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

# 4. Commit changes
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

- Repository owners can set per-worker defaults in `.git/crew/config.toml` using the `model = "<provider/model>"` field.
- `crew start -m <model>` overrides the config value when a specific model is required.

---

## Prohibited Actions

| Action | Reason |
|--------|--------|
| `git push` | Reviewer pushes when merging |
| `git push --force` | Risk of data loss |
