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
crew comment <id> "<message>"                    # Add a comment
crew comment <id> "<message>" --type report      # Work report
crew comment <id> "<message>" --type suggestion  # Improvement suggestion
crew comment <id> "<message>" --type friction    # Friction/blocker
```

Add comments in these situations:
- When you need user input or clarification on important decisions
- When you encounter blockers or unexpected issues
- When the task is complete (summary of what was done) → use `--type report`

**Friction reports** (`--type friction`):
Report friction when you experience obstacles that slow down work:
- Unclear or outdated documentation
- Missing test data or fixtures
- Confusing code structure or naming
- Tooling issues or environment problems

**Suggestions** (`--type suggestion`):
Propose improvements noticed during work:
- Code refactoring opportunities
- Architecture improvements
- Documentation updates needed
- Process improvements

### Complete Task
```bash
crew complete          # Mark task as complete
crew complete --force-review  # Run review even if not required
```

---

## Review Requirements

`[complete].max_reviews` controls how many review attempts are allowed before completion fails.
`[complete].review_success_regex` controls which review result is considered successful (matched from the start of the comment).

- Default `max_reviews`: 1
- Default `review_success_regex`: `✅ LGTM`
- `skip_review = true` bypasses the review requirement
- `crew complete --force-review` runs review even when not required
- Review count increments only when the review result is recorded
- Review runs synchronously inside `crew complete` and does not change task status unless completion succeeds

### Configuration

```toml
[complete]
command = "mise run ci"
max_reviews = 1
review_success_regex = "✅ LGTM"
```

### Deprecated Settings

- `[complete].min_reviews` is deprecated; use `max_reviews` instead.
- `[complete].review_mode` and `auto_fix` are deprecated and ignored.

---

## Task Status Reference

| Status | Meaning |
|--------|---------|
| `todo` | Created, awaiting start |
| `in_progress` | Work in progress (includes input waiting and paused states) |
| `done` | Implementation complete, awaiting merge or close |
| `merged` | Merged to base branch (terminal) |
| `closed` | Closed without merge (terminal) |
| `error` | Session terminated unexpectedly or manually stopped (restartable) |

Main flow:

```
todo -> in_progress -> done -> merged
                 \-> closed
error -> in_progress
```

`crew complete` transitions tasks to `done` after review requirements are met.

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

## Running Reviews

Reviews run synchronously inside `crew complete`. Use `--verbose` when you need to stream reviewer output.

---

## Template Variables (Agent Prompts)

Available in worker system_prompt/prompt templates:

- `{{.TaskID}}` - Task ID
- `{{.Title}}` - Task title
- `{{.Description}}` - Task description
- `{{.Branch}}` - Task branch name
- `{{.Issue}}` - GitHub issue number (0 if not linked)
- `{{.GitDir}}` - Path to .git directory
- `{{.RepoRoot}}` - Repository root path
- `{{.Worktree}}` - Worktree path
- `{{.Model}}` - Model name override
- `{{.ReviewAttempt}}` - Review attempt number (1 = first review)
- `{{.PreviousReview}}` - Previous review result (empty on first attempt)
- `{{.IsFollowUp}}` - true if review attempt > 1
- `{{.Continue}}` - true if `--continue` was specified

## Agent Configuration Tips

- Repository owners can set per-worker defaults in `.crew/config.toml` using the `model = "<provider/model>"` field.
- `crew start -m <model>` overrides the config value when a specific model is required.

---

## Prohibited Actions

| Action | Reason |
|--------|--------|
| `git push` | Reviewer pushes when merging |
| `git push --force` | Risk of data loss |
