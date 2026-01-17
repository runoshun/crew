# git-crew Onboarding Guide

This guide helps you set up git-crew for your project. Follow the checklist below to configure crew for optimal use with AI agents.

---

## Prerequisites

Before starting, ensure:
- Git repository is initialized
- `crew init` has been run (`.git/crew/` directory exists)

---

## Onboarding Checklist

### 1. Basic Configuration

Set up the core crew configuration in `.git/crew/config.toml`:

```bash
# View current config
crew config

# Edit config manually
$EDITOR .git/crew/config.toml
```

**Settings to configure:**

| Setting | Description | Example |
|---------|-------------|---------|
| `agents.worker_default` | Default worker agent | `"cc"` or `"opencode"` |
| `agents.manager_default` | Default manager agent | `"cc-manager"` |
| `agents.reviewer_default` | Default reviewer agent | `"cc-reviewer"` |

---

### 2. Project Information for AI

Create or update project documentation that AI agents will use:

#### CLAUDE.md / AGENTS.md

Create a `CLAUDE.md` (or `AGENTS.md`) file in your repository root with:

```markdown
# Project Name

## Overview
Brief description of what this project does.

## Architecture
- Key directories and their purposes
- Important patterns and conventions

## Development

### Build
```bash
# How to build the project
```

### Test
```bash
# How to run tests
```

### CI
```bash
# CI command (used by workers before completing tasks)
```

## Coding Standards
- Language-specific conventions
- Error handling approach
- Testing requirements
```

**Tips:**
- Be specific about file locations and naming conventions
- Include examples of common patterns in your codebase
- Mention any tools or frameworks that require special handling

---

### 3. Development Workflow

Configure custom skills for your workflow:

#### Option A: Use built-in skills

git-crew includes `dev-workflow` and `review-workflow` skills that work with most projects.

#### Option B: Create custom skills

Create project-specific skills in `.claude/skills/`:

```bash
mkdir -p .claude/skills/my-workflow
```

Create `.claude/skills/my-workflow/SKILL.md`:

```markdown
# My Workflow

Custom workflow for this project.

## Steps
1. ...
2. ...
```

---

### 4. Permission Configuration (Optional)

For enhanced security, configure tool permissions:

#### Claude Code settings

Create `.claude/settings.json`:

```json
{
  "allowedTools": ["Read", "Write", "Edit", "Bash", "Glob", "Grep"],
  "blockedCommands": ["rm -rf /", "git push --force"]
}
```

#### Custom agent prompts

Add role-specific instructions in config:

```toml
[agents]
worker_prompt = """
Additional instructions for all workers.
"""

manager_prompt = """
Additional instructions for all managers.
"""
```

---

### 5. Worktree Setup (Optional)

Configure automatic worktree initialization:

```toml
[worktree]
# Command to run after worktree creation (e.g., install dependencies)
setup_command = "npm install"

# Files to copy from main repo (e.g., environment files)
copy = [".env.example", ".tool-versions"]
```

---

### 6. Complete Onboarding

After completing the checklist:

1. Add `onboarding_done = true` to your config:

```bash
# Edit config file directly:
$EDITOR .git/crew/config.toml

# Add the following line at the end:
# onboarding_done = true
```

2. Verify the configuration:

```bash
# Test worker help
crew --help-worker

# Test manager help (should no longer show onboarding reminder)
crew --help-manager

# List available agents
crew list-agents
```

---

## Quick Start Commands

| Task | Command |
|------|---------|
| Create a task | See below |
| Start working | `crew start <id>` |
| Check progress | `crew peek <id>` |
| Complete task | `crew complete` (run inside worktree) |

**Example: Create a task with HEREDOC**
```bash
crew new --title "..." --body "$(cat <<'EOF'
## Summary
- ...
EOF
)"
```

---

## Troubleshooting

### Agent not found

Ensure the agent is defined in config:

```bash
crew list-agents
```

### Worktree setup fails

Check the `[worktree]` section in config:

```bash
crew config
```

### Need more help

- Run `crew --help` for command reference
- Run `crew --help-manager` for manager-specific help
- Run `crew --help-worker` for worker-specific help
