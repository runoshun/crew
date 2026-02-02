# Reviewer Template Variables

Available in reviewer system_prompt/prompt templates:

- `{{.TaskID}}` - Task ID
- `{{.Title}}` - Task title
- `{{.Description}}` - Task description
- `{{.Branch}}` - Task branch name
- `{{.Issue}}` - GitHub issue number (0 if not linked)
- `{{.GitDir}}` - Path to .git directory
- `{{.RepoRoot}}` - Repository root path
- `{{.Worktree}}` - Worktree path
- `{{.Model}}` - Model name override
- `{{.ReviewAttempt}}` - Review attempt number (1 = first review)
- `{{.PreviousReview}}` - Previous review result (empty on first attempt)
- `{{.IsFollowUp}}` - true if review attempt > 1
- `{{.Continue}}` - true if `--continue` was specified
