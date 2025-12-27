# git-crew v2 Development Guide

## Read First

Before starting any work, read the project design documentation:

- [docs/README.md](./docs/README.md) - Project overview and documentation index

---

## Available Skills

| Skill | Trigger | Description |
|-------|---------|-------------|
| `dev-workflow` | `dev` | Development workflow with TODO tracking |
| `review-workflow` | `review` | Code review workflow with TODO tracking |
| `terminal` | - | Interactive terminal sessions (for TUI development) |
| `web-search` | - | Web search for research tasks |

Load a skill with the `skill` tool when needed.

---

## Common Rules

These rules apply to both development and review workflows.

### CI Requirements

Always run before committing:

```bash
mise run ci
```

### Branch Strategy

- **Never commit directly to main**
- Use feature branches: `feature/<description>`
- Create from main: `git checkout main && git checkout -b feature/<description>`

### Commit Message Format

```
<type>: <description>

[optional body]
```

Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`

### Documentation

- Keep docs/ in sync with code changes
- Update TASKS.md checkboxes when completing tasks

---

## Development Flow Overview

Development follows three phases. For detailed steps, use the `dev-workflow` skill.

### Phase 1: Setup

- Read docs/README.md and related docs (architecture.md, spec-*.md as needed)
- Check TASKS.md for current task
- Ensure you're on a feature branch (create if needed)

### Phase 2: Implementation

- Break down the task into TODOs
- Implement incrementally, running CI frequently
- Update TASKS.md checkboxes as you complete items

### Phase 3: Wrap-up

- Run final CI check
- Commit changes (when requested by user)
- **Retrospective**: Reflect on the session and propose guideline improvements if needed
- **Design Feedback**: Report any friction with docs/ to DESIGN_FEEDBACK.md

---

## Review Flow Overview

For detailed steps, use the `review-workflow` skill.

### What to Check

1. **Correctness** - Does the code do what it's supposed to do?
2. **Tests** - Are edge cases covered? Are tests readable?
3. **Architecture** - Does it follow the project structure?
4. **Error handling** - Are errors handled appropriately?
5. **Readability** - Will future developers understand this?
6. **Documentation** - Are docs/ updated if behavior changed?

### Review Process

1. Identify target (PR, branch, or files)
2. Run code locally: `mise run ci`
3. Check each item above
4. Provide specific, actionable feedback

---

## Developer Guidelines

### Rules (MUST follow)

1. **Always run `mise run ci` before committing**
2. **Write tests for all new code**
3. **Follow the 1 file = 1 UseCase pattern**
4. **Never commit directly to main**
5. **Handle all errors explicitly**
6. **Use domain errors, not raw errors**

### Best Practices (SHOULD follow)

1. Write code that reads well - clarity over cleverness
2. Keep functions small - if it's hard to test, it's too big
3. Use meaningful names - self-documenting code
4. Comment the "why", not the "what"
5. Fail fast - return errors early
6. Use ADT/sumtypes with `exhaustive` linter

### Things to Avoid

1. Don't use `panic()` for error handling
2. Don't ignore linter warnings
3. Don't add TODO comments without context
4. Don't over-engineer
5. Don't copy-paste code

---

## Reviewer Guidelines

### Rules (MUST follow)

1. **Run the code locally** - don't just read
2. **Check that tests exist and are meaningful**
3. **Verify CI passes**
4. **Be specific in feedback** - explain why and suggest alternatives

### Best Practices (SHOULD follow)

1. Start with the positive - acknowledge good work
2. Ask questions, don't demand - "Have you considered X?"
3. Distinguish blockers from suggestions - use [BLOCKING] or [NIT]
4. Review for correctness first, style second

### What NOT to Do

1. Don't bikeshed - focus on substance
2. Don't rewrite their PR - suggest improvements
3. Don't approve without reviewing
4. Don't block on minor issues

---

## Quick Reference

```bash
# Development
mise run build                  # Build binary
mise run test                   # Run tests
mise run lint                   # Check code style
mise run ci                     # Full CI check (required before commit)

# Coverage
mise run test:cover             # Run with coverage
go tool cover -func=coverage.out  # View summary
```
