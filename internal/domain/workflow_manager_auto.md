# Auto Mode

Trigger: Say **"auto mode"** to activate this workflow for managing multiple in-progress tasks.

## Purpose

Advance all in-progress tasks automatically with minimal human intervention. Continue looping until each task reaches one of these states:
- ✅ **LGTM** - Reviewer approval complete (task is `reviewed` with LGTM comment)
- ⚠️ **Problematic** - Task flagged for issues (manager decision to stop and report to user)

## Important: No Interaction During Auto Mode

- **NO confirmations/questions until exit conditions are met** (y/n, selections, etc.)
- Manager makes autonomous decisions about:
  - When to run `crew review <id>`
  - When to execute `crew comment -R`
  - Whether to approve `needs_input` requests
- **Exception**: Only in final summary, list items that cannot be decided (do NOT ask mid-session)

## Target Statuses

- **In scope**: `in_progress`, `needs_input`, `for_review`, `reviewing`, `reviewed`
- **Out of scope**: `todo` (requires explicit start instruction)

## Review Result Handling

- ✅ **LGTM only** = task complete (marks endpoint)
- ⚠️ **Minor issues** = NOT LGTM (continue work loop)
- ❌ **Needs changes** = NOT LGTM (continue work loop)

## Auto Mode Loop Specification (Priority Order)

1. **`crew list`** - Enumerate in-scope tasks
2. **`for_review` status**:
   - Execute `crew review <id>`
   - When status becomes `reviewed`, check reviewer comment
   - ✅ If LGTM: Mark task as complete, move to next
   - ⚠️/❌ If issues: Send `crew comment <id> -R "..."` **without asking** (notify worker, reset to `in_progress`)
3. **`reviewed` status**:
   - Check reviewer comment with `crew show <id>`
   - Same handling as above
   - Note: `reviewed` state cannot be completed with `crew complete` (no merge in auto mode)
4. **`needs_input` status**:
   - Check requirement with `crew peek <id>`
   - Manager autonomously approves safe requests
   - Flag problematic ones (destructive, security, host impact, out-of-scope, etc.)
   - Continue other tasks first, report problems in final summary
5. **`reviewing` status**:
   - Wait (other tasks have priority)
6. **`in_progress` status**:
   - Wait, but use `crew peek <id>` to detect stalls/errors
7. **Loop exit**: All in-scope tasks reach ✅ LGTM or ⚠️ problematic state

## Safety Guidelines

**Approve** (examples):
- Worktree read/build/test/format: `mise run ci`, `go test`, `git diff`, `git status`
- Branch operations within task scope

**Deny** (examples):
- Host system impact: persistent changes, permissions, service control
- Out-of-scope worktree operations
- Out-of-task file changes (e.g., doc-fix task modifying test files)
- Destructive: broad `rm`, `git reset --hard` without justification, force push
- Security risk: credential handling, etc.

## Comment Template for Addressing Issues

When reviewer finds ⚠️/❌ issues:

```bash
crew comment <id> -R "$(cat <<'MSG'
Please address the reviewer's feedback (see task comments, author=reviewer).

Priority issues:
1) [Brief description]
2) [Brief description]
3) [Brief description]

After fixing, set status to for_review for re-review.
MSG
)"
```

**Best practices**:
- Reference instead of transcribing full findings
- Highlight top 3 priorities
- Keep concise and actionable
- Avoid duplicating reviewer comments

## Example Workflow

```bash
# Start auto mode (monitor multiple tasks)
crew list  # See in_progress and for_review tasks

# Task #100 is for_review
crew review 100
# After review: found issues
crew comment 100 -R "..."  # Send feedback, reset to in_progress

# Task #101 is for_review
crew review 101
# After review: LGTM
# ✅ Task #101 complete

# Task #100 now in_progress (was notified of feedback)
# (Continue monitoring until it returns to for_review)
crew poll 100 --expect for_review  # Wait for next review submission

# Task #102 is needs_input (asking for permission)
crew peek 102  # Check what permission is needed
# If safe: approve with crew comment -R "..."
# If risky: flag for manual decision in summary

# Loop continues until all tasks are ✅ LGTM or ⚠️ flagged
```
