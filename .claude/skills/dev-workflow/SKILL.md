---
name: dev-workflow
description: Task-driven development workflow with TODO tracking. Use when user says "dev", "dev-workflow", or asks to work on tasks, develop features, or proceed with implementation.
---

# Dev Workflow

Implements the Development Flow from AGENTS.md with detailed steps and TODO tracking.

## Auto-Start

When this skill is loaded, IMMEDIATELY begin Phase 1. Do NOT wait for user confirmation.

---

## Phase 1: Setup

**Goal**: Understand context and prepare workspace.

### Steps

1. **Read project docs** (skip if already familiar from this session)
   - Read docs/README.md for project overview
   - Read related docs based on the task:
     - docs/core-concepts.md - Design principles
     - docs/architecture.md - Code structure (always read for implementation tasks)
     - docs/spec-cli.md - Command specs (when implementing CLI)
     - docs/spec-tui.md - TUI specs (when implementing TUI)
   
2. **Check current task**
   - Read TASKS.md to identify the next uncompleted task
   - If unclear, ask the user which task to work on

3. **Ensure feature branch**
   - Check current branch: `git branch --show-current`
   - If on `main`: create feature branch `git checkout -b feature/<task-description>`
   - If already on `feature/*`: continue on current branch

### TODOs for Phase 1

```
- [ ] Read docs/README.md and related docs (architecture.md, spec-*.md as needed)
- [ ] Check TASKS.md for current task
- [ ] Ensure on feature branch
```

---

## Phase 2: Implementation

**Goal**: Complete the task with quality.

### Steps

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

### TODOs for Phase 2 (adapt to actual task)

```
- [ ] <Implementation step 1>
- [ ] <Implementation step 2>
- [ ] ...
- [ ] Run mise run ci
- [ ] Update TASKS.md checkboxes
```

---

## Phase 3: Wrap-up

**Goal**: Finalize changes and reflect on the session.

### Steps

1. **Final CI check**
   - Run `mise run ci` one last time
   - Ensure all tests pass

2. **Commit** (when user requests)
   - Stage relevant files
   - Write clear commit message following format in AGENTS.md

3. **Retrospective**
   - Review feedback received during the session:
     - Instructions repeated multiple times
     - Explicit requests like "please always do X"
     - Corrections from the user
   - If patterns suggest AGENTS.md or docs/ improvements:
     - Propose changes to the user
     - If approved, create a separate commit

4. **Design Feedback** (if applicable)
   - If you encountered friction with specs in docs/:
     - Specs difficult to implement as written
     - Inconsistencies between docs
     - Missing details requiring assumptions
   - Append findings to DESIGN_FEEDBACK.md (do NOT modify docs/ directly)

### TODOs for Phase 3

```
- [ ] Run final mise run ci
- [ ] Commit changes (on user request)
- [ ] Retrospective: review session for guideline improvements
- [ ] Design Feedback: document any friction with docs/
```

---

## Rules

1. **Start immediately** - Begin Phase 1 upon skill load
2. **Register ALL steps as TODOs** - Before starting each phase
3. **Update TODOs in real-time** - Mark in_progress/completed as you work
4. **One item in progress at a time** - Focus and complete before moving on
5. **Ask before proceeding to next task** - After Phase 3, confirm with user
