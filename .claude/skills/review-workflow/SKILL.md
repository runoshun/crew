---
name: review-workflow
description: Code review workflow with TODO tracking. Use when user says "review", "review-workflow", or asks to review code, PRs, or changes.
---

# Review Workflow

Follow AGENTS.md Reviewer Guidelines, tracking progress with TodoWrite.

## Auto-Start

When this skill is loaded, IMMEDIATELY start the workflow:

1. Identify review target (detect from context or ask if unclear)
2. Register ALL review steps as TODOs
3. Begin the review process

Do NOT wait for user confirmation. Start working right away.

## Identifying Review Target

Detect or ask:
- PR number: `gh pr view <number>`
- Branch diff: `git diff main...<branch>`
- Specific files: read the files directly

If target is ambiguous, ask the user briefly, then proceed.

## TODO Management

Register these steps as TODOs:

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

## Rules

1. Start immediately upon skill load - no waiting
2. Register ALL check items as TODOs before starting
3. Update TODO status in real-time as you work
4. Only ONE item in progress at a time
