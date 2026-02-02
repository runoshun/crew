# git-crew Reviewer Guide

## IMPORTANT: Follow This Workflow

NOTE: Default flow is to use `crew complete`. Use `crew comment` only for manual reviews.

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

```

---

## Manual Review Only

Use this only when you are not running `crew complete`.

```bash
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

{{if .IsFollowUp}}
## Follow-up Review Mode

When reviewing a follow-up attempt, focus on:

1. Verify previous review issues have been addressed
2. Check ONLY changes made since last review
3. Report ONLY blocking issues - skip new minor issues

If all issues are addressed, respond with "✅ LGTM".

---
{{end}}

## Output Format

IMPORTANT: Do NOT run `crew comment` when using `crew complete`. It records your review result.

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
