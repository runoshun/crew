---
name: dev-workflow
description: Task-driven development workflow with TODO tracking. Use when user says "dev", "dev-workflow", or asks to work on tasks, develop features, or proceed with implementation.
---

# Dev Workflow

Follow AGENTS.md Development Flow, tracking progress with TodoWrite.

## Auto-Start

When this skill is loaded, IMMEDIATELY start the workflow:

1. Read docs/README.md (if not already familiar with the project)
2. Check TASKS.md for current task
3. Register ALL implementation steps as TODOs
4. Begin working on the first TODO

Do NOT wait for user confirmation. Start working right away.

## TODO Management

Register these steps as TODOs (adapt based on actual task):

```
- [ ] Read docs/README.md
- [ ] Check TASKS.md for current task  
- [ ] Create feature branch
- [ ] <Implementation step 1>
- [ ] <Implementation step 2>
- [ ] ...
- [ ] Test: mise run ci
- [ ] Update TASKS.md checkboxes
- [ ] Commit (on user request)
```

## Rules

1. Start immediately upon skill load - no waiting
2. Register ALL steps as TODOs before implementation
3. Update TODO status in real-time as you work
4. Only ONE item in progress at a time
