---
name: dev-workflow
description: Task-driven development workflow with TODO tracking. Use when user says "dev", "dev-workflow", or asks to work on tasks, develop features, or proceed with implementation.
---

# Dev Workflow

Implements the Development Flow from AGENTS.md with detailed steps and TODO tracking.

## Auto-Start

When this skill is loaded, IMMEDIATELY begin Phase 1. Do NOT wait for user confirmation.

---

## TODOs

Register ALL TODOs at once when starting Phase 1:

```
# Phase 1: Setup
- [ ] Read docs/README.md, core-concepts.md, architecture.md, spec-cli.md, spec-tui.md
- [ ] Check TASKS.md for current task
- [ ] Ensure on feature branch
# Phase 2: Implementation
- [ ] Plan implementation and update TODOs (add specific steps here)
# Phase 3: Wrap-up
- [ ] Run final CI check
- [ ] Check test coverage for new code
- [ ] Commit and merge (ask user - single confirmation for both)
- [ ] Review session feedback → propose guideline updates
- [ ] Document any docs/ friction in DESIGN_FEEDBACK.md
- [ ] Ask user before proceeding to next task
```

---

## Rules

1. **Start immediately** - Begin Phase 1 upon skill load
2. **Register ALL TODOs at once** - Include all phases when starting Phase 1
3. **Update TODOs in real-time** - Mark in_progress/completed as you work
4. **One item in progress at a time** - Focus and complete before moving on
5. **Add implementation steps** - When reaching "Plan implementation and update TODOs", add specific steps

---

## Reference: Phase Details

### Phase 1: Setup

**Goal**: Understand context and prepare workspace.

1. **Read project docs** (skip if already familiar from this session)
   - **MUST read ALL docs**:
     - docs/README.md - Project overview
     - docs/core-concepts.md - Design principles
     - docs/architecture.md - Code structure
     - docs/spec-cli.md - CLI command specs
     - docs/spec-tui.md - TUI screen specs

2. **Check current task**
   - Read TASKS.md to identify the next uncompleted task
   - If unclear, ask the user which task to work on

3. **Ensure feature branch**
   - Check current branch: `git branch --show-current`
   - If on `main`: create feature branch `git checkout -b feature/<task-description>`
   - If already on `feature/*`: continue on current branch

### Phase 2: Implementation

**Goal**: Complete the task with quality.

1. **Plan implementation**
   - Break down the task into specific TODOs
   - Each TODO should be a single, verifiable step

2. **Implement incrementally**
   - Work on ONE TODO at a time
   - Mark TODO as `in_progress` when starting
   - Mark TODO as `completed` immediately when done

3. **Run CI frequently**
   - Run `mise run ci` after significant changes
   - Fix any issues before proceeding

4. **Update TASKS.md**
   - Check off completed items in TASKS.md as you go

### Phase 3: Wrap-up

**Goal**: Finalize changes and reflect on the session.

1. **Final CI check**
   - Run `mise run ci` one last time
   - Ensure all tests pass

2. **Check test coverage**
   - Run `mise run test:cover`
   - Review coverage: `go tool cover -func=coverage.out | grep -E "(total|<new-package>)"`
   - Add tests if coverage is insufficient

3. **Commit and Merge**
   - When task is complete, ask user once: "Commit and merge to main?"
   - If yes:
     - Stage and commit with clear message
     - Switch to main: `git checkout main`
     - Merge with no-ff: `git merge --no-ff feature/<branch>`
     - Delete feature branch: `git branch -d feature/<branch>`
   - If continuing on same branch (e.g., next task is closely related), skip merge

4. **Retrospective**
   - Review feedback received during the session:
     - Instructions repeated multiple times
     - Explicit requests like "please always do X"
     - Corrections from the user
   - If patterns suggest AGENTS.md or docs/ improvements:
     - Propose changes to the user
     - If approved, create a separate commit

5. **Design Feedback** (if applicable)
   - If you encountered friction with specs in docs/:
     - Specs difficult to implement as written
     - Inconsistencies between docs
     - Missing details requiring assumptions
   - Append findings to DESIGN_FEEDBACK.md (do NOT modify docs/ directly)
