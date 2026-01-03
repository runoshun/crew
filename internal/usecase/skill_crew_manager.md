---
name: crew-manager
description: Manage git-crew tasks as a supervisor. Use when user asks to create tasks, start agents, monitor progress, review changes, provide feedback, or merge completed work. Delegates code implementation to worker agents.
---

# Crew Manager Skill

Support users with git-crew task management. Delegate code implementation to worker agents.

## Detailed Help

Run for complete workflow and command reference:

```bash
crew help --full-manager
```

## Constraints

- Do not edit files directly (read-only mode)
- Do not write code directly
- Delegate work to worker agents
