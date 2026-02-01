# git-crew Reviewer Guide

## IMPORTANT: Follow This Workflow

1. **Identify target**: Decide what you are reviewing (task ID, PR, branch, or files)
2. **Inspect changes**: Use `crew show` / `crew diff` to understand the full context
3. **Run CI**: Verify tests/lint/build in the correct context
4. **Write review**: Write feedback with the required format after the marker line

---

## Basic Operations for understanding the context

```bash
# Show task details
crew show <id>

# Show task diff
crew diff <id>

# Add a review comment
crew comment <id> "<message>"
```

---

## Review Checklist

1. Correctness - Does the code work as intended?
2. Tests - Are edge cases covered?
3. Architecture - Does it follow project patterns?
4. Error handling - Are errors handled explicitly?
5. Readability - Will future developers understand this?
6. Documentation - Are docs updated if behavior changed?

---

## Output Format

IMPORTANT: Do NOT run `crew comment`. `crew complete` will record your review result.

Start with: `✅ LGTM`, `⚠️ Minor issues`, or `❌ Needs changes`.
Then list specific issues with file:line references.

IMPORTANT: your final review output must be after the marker line: `---REVIEW_RESULT---`, And DO NOT output other text after that.

```markdown
---REVIEW_RESULT---
## Summary
<1-2 sentence overview>

## Blocking Issues
- [BLOCKING] <issue description>
  - Why: <explanation>
  - Suggestion: <how to fix>

## Suggestions
- [NIT] <minor issue>
- [SUGGESTION] <improvement idea>
```
