# git-crew

> **1 User, Infinite AI Team.**

**git-crew** is a task management CLI that combines **Git Worktree** and **Tmux** to enable a fully parallelized AI development workflow.

Unlike traditional "1-on-1" AI coding where you wait for the AI to finish typing, git-crew allows you to orchestrate a team of AI agents. While you plan the next feature with the **Manager**, multiple **Workers** are implementing, testing, and fixing bugs in the background—each in their own isolated environment.

## 🚀 The Crew Workflow

### 1. The Manager: Your Development Partner

The **Manager** is not just a chatbot; it's an autonomous agent that understands your project and orchestrates work.

```bash
# Initialize the repository
crew init

# Start a Manager session
crew manager
```

**What the Manager does:**
- 🗣️ **Requirements Analysis**: Discusses vague ideas and turns them into concrete specs.
- 📋 **Task Planning**: Breaks down features into smaller, manageable tasks.
- 🚀 **Orchestration**: Creates tasks (`crew new`) and assigns Worker agents (`crew start`) to execute them in parallel.
- 👀 **Monitoring**: Tracks progress across all active tasks.

### 2. Parallel Execution

While the Manager coordinates, **Worker** agents execute tasks in completely isolated **Git Worktrees**.

- **Isolated Workspaces**: Each task runs in its own directory (`<repo>-worktrees/task-123/`), keeping your main working tree clean.
- **Parallel Productivity**: Run multiple tasks simultaneously. While one agent fixes a bug, another writes docs, and you plan the next feature.
- **Background Processing**: Agents run in Tmux sessions. Detach and let them work; attach when they need you.

### 3. Review & Merge

When a Worker finishes, you review their work just like a human PR.

```bash
# Check status of all tasks
crew list

# Review changes (git diff wrapper)
crew diff 1

# Merge the feature branch into main
crew merge 1
```

---

## ⚙️ Configuration: Specialized Agents

You can define specialized agents in `~/.config/crew/config.toml`.

**Smart Dispatching:**
The **Manager** reads the `description` fields to automatically assign the most suitable agent for each task. It routes simple fixes to faster, cheaper models and complex architecture tasks to smarter models, optimizing both cost and speed without you having to choose manually.

**Example Configuration:**

```toml
[agents]
worker_default = "claude-dev"

# Cheap & Fast: Manager assigns this for typos and simple docs
[agents.claude-fast]
inherit = "claude"
default_model = "haiku"
description = "Trivial tasks: typo fixes, simple doc edits, small bug fixes."

# Standard: Daily driver
[agents.claude-dev]
inherit = "claude"
default_model = "sonnet"
description = "Standard tasks: feature implementation, refactoring, tests."

# Smart & Slow: Manager assigns this for complex logic
[agents.claude-architect]
inherit = "claude"
default_model = "opus"
description = "Complex tasks: system design, difficult debugging, migration planning."

# Specialized: Frontend
[agents.frontend-specialist]
inherit = "opencode"
default_model = "google/gemini-3-pro-preview"
description = "Frontend Specialist: UI/UX tasks, styling, responsiveness."
```

### Custom Agents

You are not limited to pre-built integrations. You can define fully custom agents to use **any CLI tool** (e.g., a custom python script, a different AI wrapper).

For details on custom agents and full configuration options, see [Core Concepts: Configuration](./docs/core-concepts.md#4-configuration-system) or run `crew config`.

---

## ⚡ Quick Start (Manual Mode)

You can also use git-crew manually without the Manager.

```bash
# 1. Create a task
crew new "Add dark mode support"

# 2. Start an agent (defaults to worker_default)
crew start 1

# 3. Attach to the session to interact
crew attach 1

# 4. (Optional) Run a specialized agent
crew start 2 --worker claude-fast
```

---

## 📦 Installation

### Binary (Recommended)

Download the latest binary from [GitHub Releases](https://github.com/runoshun/git-crew/releases).

### From Source

```bash
git clone https://github.com/runoshun/git-crew.git
cd git-crew
go build -o crew ./cmd/crew
cp crew ~/.local/bin/
```

### Requirements

**Core Dependencies:**
- git
- tmux

**AI Agent CLI (One or more required):**
- [Claude Code](https://docs.anthropic.com/en/docs/agents-and-tools/claude-code/overview) (`claude`)
- [OpenCode](https://github.com/runoshun/opencode) (`opencode`)
- Or any other terminal-based AI agent.

---

## License

MIT
