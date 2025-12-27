# git-crew v2 Development Guide

## IMPORTANT: Read First

**Before starting any work, you MUST read the project design documentation:**

- [docs/README.md](./docs/README.md) - Project overview and documentation index

This document provides essential context about the architecture, design decisions, and specifications.

---

## Available Skills

The following skills are available for use during development:

| Skill | Usage |
|-------|-------|
| `terminal` | Interactive terminal sessions. Useful for TUI development and testing interactive features. Use the Bash tool for simple non-interactive commands; use this skill only when interactivity is needed. |
| `web-search` | Web search for research tasks. Use actively when the user requests investigation or needs up-to-date information. |

Load a skill with the `skill` tool when needed.

---

## Development Tasks (mise)

All development tasks are managed via mise. Run `mise tasks` to see available commands.

| Task | Description | Usage |
|------|-------------|-------|
| `build` | Build crew binary | `mise run build` |
| `test` | Run tests | `mise run test` |
| `test:race` | Run tests with race detector | `mise run test:race` |
| `test:cover` | Run tests with coverage | `mise run test:cover` |
| `lint` | Run golangci-lint | `mise run lint` |
| `ci` | Run CI checks (lint + test:race) | `mise run ci` |
| `ci:full` | Full CI (lint + test:cover + vuln) | `mise run ci:full` |
| `fmt` | Format code | `mise run fmt` |
| `tidy` | Tidy go.mod | `mise run tidy` |
| `vuln` | Check for vulnerabilities | `mise run vuln` |
| `install` | Install to ~/.local/bin | `mise run install` |
| `clean` | Clean build artifacts | `mise run clean` |

### CI Requirements

Before submitting any changes, ensure:

```bash
mise run ci
```

This runs both linting and tests with race detection.

---

## Development Flow

### 1. Starting Work

1. Read [docs/README.md](./docs/README.md) to understand the project
2. Check TASKS.md for the current task list
3. Create a feature branch from main: `git checkout main && git checkout -b feature/<description>`

### 2. During Development

1. Write code following the architecture in docs/architecture.md
2. Write tests alongside implementation
3. Commit in small, logical units with clear messages

### 3. Before Submitting

1. Ensure all tests pass: `mise run test:race`
2. Ensure linting passes: `mise run lint`
3. Run the full CI check: `mise run ci`
4. Check coverage with `mise run test:cover` to ensure critical logic is tested (100% coverage is not required, but core logic must be covered)
   - View coverage summary: `go tool cover -func=coverage.out`
   - Generate HTML report: `go tool cover -html=coverage.out -o coverage.html`
5. Review your own changes before requesting review

### 4. Code Review

1. Address all review comments
2. Re-run CI after making changes
3. Keep the PR focused on a single concern

### 5. Retrospective (Self-Improvement)

After completing the main task, reflect on the development process:

1. **Identify patterns** - Review feedback received during the session:
   - Instructions that were repeated multiple times
   - Explicit requests like "please always do X"
   - Corrections or clarifications from the user
2. **Propose improvements** - If any patterns suggest AGENTS.md or docs/ should be updated:
   - Create a separate commit for guideline changes
   - Keep it independent from the main task commits
   - Explain the reasoning in the commit message
3. **Scope check** - Ensure proposed changes are:
   - Generally applicable, not task-specific
   - Consistent with existing guidelines
   - Clear and actionable

**Reviewer responsibility**: When reviewing guideline changes, verify:
- The change is not overly specific to one task
- It doesn't conflict with existing guidelines
- It improves clarity for future work

### 6. Design Feedback (Optional)

If you encountered friction with the current design in docs/, document it:

1. **What to report**:
   - Specs that were difficult to implement as written
   - Inconsistencies between docs
   - Missing details that required assumptions
   - Suggestions for simplification or improvement
2. **How to report**:
   - Append to `DESIGN_FEEDBACK.md` in the repository root
   - Do NOT modify design docs directly without explicit approval
   - Include date, context, and specific file references
3. **Format**:
   ```markdown
   ## YYYY-MM-DD HH:MM: <Short Title>
   
   **Context**: <What task or implementation triggered this>
   **Issue**: <What was problematic>
   **Affected docs**: <File paths>
   **Suggestion**: <Proposed improvement, if any>
   ```

**Note**: Design changes require discussion. The reviewer or maintainer will process feedback separately from the development flow.

---

## Developer Guidelines

### Rules (MUST follow)

1. **Always run `mise run ci` before committing** - No exceptions
2. **Write tests for all new code** - Aim for meaningful coverage, not 100%
3. **Follow the 1 file = 1 UseCase pattern** - Keep use cases focused and testable
4. **Never commit directly to main** - Always use feature branches
5. **Keep dependencies minimal** - Only add libraries when truly necessary
6. **Handle all errors explicitly** - No ignored errors without clear justification
7. **Use domain errors** - Return domain-specific errors, not raw errors
8. **Keep docs/ in sync with code** - Update documentation when behavior or architecture changes

### Best Practices (SHOULD follow)

1. **Write code that reads well** - Clarity over cleverness
2. **Keep functions small** - If it's hard to test, it's too big
3. **Use meaningful names** - Variables, functions, and types should be self-documenting
4. **Comment the "why", not the "what"** - Code shows what, comments explain why
5. **Prefer composition over inheritance** - Go interfaces enable this naturally
6. **Make zero values useful** - Design types so zero values work correctly
7. **Fail fast** - Return errors early, don't bury them in nested conditions
8. **Use ADT/sumtypes actively** - Use `exhaustive` linter for enums, `go-sumtype` for sealed interfaces to ensure exhaustive pattern matching at compile time

### Things to Avoid

1. **Don't use `panic()` for error handling** - Reserve for truly unrecoverable situations
2. **Don't ignore linter warnings** - Fix them or add justified `//nolint` comments
3. **Don't add TODO comments without context** - Include why and when it should be done
4. **Don't over-engineer** - Solve today's problems, not imaginary future ones
5. **Don't copy-paste code** - Extract common functionality

---

## Reviewer Guidelines

### Rules (MUST follow)

1. **Review within 24 hours** - Don't block progress
2. **Run the code locally** - Don't just read, verify it works
3. **Check that tests exist and are meaningful** - Coverage alone is not enough
4. **Verify CI passes** - Don't approve if CI is failing
5. **Be specific in feedback** - "This is wrong" is not helpful; explain why and suggest alternatives

### Best Practices (SHOULD follow)

1. **Start with the positive** - Acknowledge good work before pointing out issues
2. **Ask questions, don't demand** - "Have you considered X?" works better than "Do X"
3. **Distinguish between blockers and suggestions** - Use clear labels like [BLOCKING] or [NIT]
4. **Review for correctness first, style second** - Functionality matters more than formatting
5. **Consider the author's context** - They may have constraints you're not aware of

### What to Look For

1. **Correctness** - Does the code do what it's supposed to do?
2. **Tests** - Are edge cases covered? Are tests readable?
3. **Architecture** - Does it follow the project structure?
4. **Error handling** - Are errors handled appropriately?
5. **Performance** - Any obvious issues? (Don't over-optimize)
6. **Security** - Any potential vulnerabilities?
7. **Readability** - Will future developers understand this?
8. **Documentation** - Are docs/ updated if behavior or architecture changed?

### What NOT to Do

1. **Don't bikeshed** - Focus on substance, not style preferences
2. **Don't rewrite their PR** - Suggest improvements, don't impose your style
3. **Don't approve without reviewing** - Take the responsibility seriously
4. **Don't block on minor issues** - Approve with suggestions for small fixes

---

## Commit Message Format

Use clear, descriptive commit messages:

```
<type>: <description>

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `refactor`: Code restructuring without behavior change
- `test`: Adding or updating tests
- `docs`: Documentation changes
- `chore`: Maintenance tasks

Examples:
```
feat: add task creation command

Implements `git crew new` command with --title, --desc, and --parent flags.
Follows the UseCase pattern defined in architecture.md.
```

```
fix: handle missing worktree gracefully

Return domain.ErrWorktreeNotFound instead of panicking when
worktree directory doesn't exist.
```

---

## Quick Reference

```bash
# Setup
mise install                    # Install Go and tools

# Development
mise run build                  # Build binary
mise run test                   # Run tests
mise run lint                   # Check code style
mise run ci                     # Full CI check

# Before committing
mise run fmt                    # Format code
mise run tidy                   # Clean up go.mod
mise run ci                     # Verify everything passes
```
