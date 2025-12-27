---
name: review-workflow
description: Code review workflow with TODO tracking. Use when asked to review code, PRs, or changes.
---

# Review Workflow

Follow AGENTS.md Reviewer Guidelines, tracking progress with TodoWrite.

## TODO Management

At the start of each review, register steps as TODOs:

```
- [ ] Identify review target (PR, branch, or files)
- [ ] Run code locally: mise run ci
- [ ] Check: Correctness
- [ ] Check: Tests exist and are meaningful
- [ ] Check: Architecture follows project structure
- [ ] Check: Error handling
- [ ] Check: Readability
- [ ] Check: docs/ updated if needed
- [ ] Provide feedback
```

## Identifying Review Target

Ask or detect:
- PR number: `gh pr view <number>`
- Branch diff: `git diff main...<branch>`
- Specific files: read the files directly

## Rules

1. Register ALL check items as TODOs before starting
2. Update TODO status in real-time as you work
3. Only ONE item in progress at a time
