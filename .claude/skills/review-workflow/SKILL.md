---
name: review-workflow
description: Code review workflow with TODO tracking. Use when user says "review", "review-workflow", or asks to review code, PRs, or changes.
---

# Review Workflow

Implements the Review Flow from AGENTS.md with detailed steps and TODO tracking.

## Prerequisites

Before reviewing, understand the development context:

- Read docs/README.md and related docs (architecture.md, spec-*.md as needed)
- Understand the Development Flow in AGENTS.md (what developers are expected to do)

## Auto-Start

When this skill is loaded, IMMEDIATELY begin the review. Do NOT wait for user confirmation.

---

## Phase 1: Identify Target

**Goal**: Determine what to review.

### Detection Order

1. Check if user specified a target (PR number, branch, files)
2. Check current branch - if on `feature/*`, review that branch vs main
3. Ask user if still unclear

### Commands

```bash
# For PR
gh pr view <number>
gh pr diff <number>

# For branch
git diff main...HEAD
git log main..HEAD --oneline

# For uncommitted changes
git diff
git status
```

---

## Phase 2: Run Locally

**Goal**: Verify the code works.

### Steps

1. **Run CI**
   ```bash
   mise run ci
   ```

2. **Note any failures** - These are blocking issues

---

## Phase 3: Check Quality

**Goal**: Evaluate code against project standards.

### Checklist

| Check | What to Look For |
|-------|------------------|
| **Correctness** | Does it do what it's supposed to? Edge cases handled? |
| **Tests** | Do tests exist? Are they meaningful? Cover edge cases? |
| **Architecture** | Follows 1 file = 1 UseCase? Proper layer separation? |
| **Error Handling** | All errors handled? Domain errors used? |
| **Readability** | Clear names? Appropriate comments? Not too clever? |
| **Documentation** | docs/ updated if behavior changed? TASKS.md updated? |

### How to Check

- Read the diff carefully
- Cross-reference with docs/architecture.md for patterns
- Check that tests actually test the right things

---

## Phase 4: Provide Feedback

**Goal**: Give actionable feedback.

### Feedback Format

```markdown
## Summary
<1-2 sentence overview>

## Blocking Issues
- [BLOCKING] <issue description>
  - Why: <explanation>
  - Suggestion: <how to fix>

## Suggestions
- [NIT] <minor issue>
- [SUGGESTION] <improvement idea>

## Positive Notes
- <what was done well>
```

### Rules for Feedback

1. **Be specific** - Point to exact lines/files
2. **Explain why** - Not just "this is wrong"
3. **Suggest alternatives** - Don't just criticize
4. **Distinguish severity** - [BLOCKING] vs [NIT] vs [SUGGESTION]
5. **Acknowledge good work** - Start with positives when possible

---

## TODOs

Register these when starting a review:

```
- [ ] Read docs/README.md and related docs (architecture.md, spec-*.md as needed)
- [ ] Identify review target
- [ ] Run mise run ci
- [ ] Check: Correctness
- [ ] Check: Tests
- [ ] Check: Architecture
- [ ] Check: Error handling
- [ ] Check: Readability
- [ ] Check: Documentation
- [ ] Provide feedback
```

---

## Rules

1. **Start immediately** - Begin identifying target upon skill load
2. **Run code locally** - Never review without running CI
3. **Be constructive** - Goal is to improve, not criticize
4. **Focus on substance** - Don't bikeshed on style preferences
5. **Complete all checks** - Don't skip items in the checklist
