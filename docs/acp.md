# ACP Workflow

ACP is a separate execution path from the legacy tmux-based runner. Use the
`crew acp ...` command group for ACP sessions and do not mix it with legacy
commands (`crew start/send/attach/peek`).

Note: The subcommands below match `go run ./cmd/crew acp --help` in this repo.
If your installed `crew` binary does not list `start/attach/peek`, rebuild from
source or use `crew acp run` directly.

## Key differences from legacy

- ACP sessions are started and controlled via `crew acp ...` commands.
- tmux is display-only for ACP. Key presses in tmux do not reach the agent.
- Use `crew acp send` for prompts and `crew acp permission` for approvals.

## Quick start

```bash
crew acp start 1 --agent my-acp-agent --model gpt-4o
crew acp send 1 "Hello from ACP"
crew acp permission 1 #1
crew acp log 1
crew acp stop 1
```

## Command reference

### start

Start a tmux session that runs `crew acp run`.

```bash
crew acp start <task-id> --agent <agent> [--model <model>]
```

### run

Run ACP in the current terminal. Normally invoked by `crew acp start`.

```bash
crew acp run --task <task-id> --agent <agent> [--model <model>]
```

### send

Send a prompt to the ACP session.

```bash
crew acp send <task-id> "Prompt text"
crew acp send --task <task-id> --text "Prompt text"
```

### permission

Respond to a permission request. Use `#<index>` to select from the latest
permission request options (positional or `--option`).

```bash
crew acp permission <task-id> <option-id>
crew acp permission <task-id> #1
```

### cancel

Cancel the current ACP session request.

```bash
crew acp cancel <task-id>
```

### stop

Stop the ACP session cleanly.

```bash
crew acp stop <task-id>
```

### attach

Attach to the tmux session for viewing output (input is not sent to the agent).

```bash
crew acp attach <task-id>
```

### peek

Capture output from the ACP tmux session without attaching.

```bash
crew acp peek <task-id>
crew acp peek <task-id> --lines 50 --escape
```

### log

View the ACP event log.

```bash
crew acp log <task-id>
crew acp log <task-id> --raw
```

## Data locations

ACP persists its IPC, state, and event logs under:

```
.crew/acp/<namespace>/<task-id>/
```

Key files/directories:

- `commands/` - IPC commands written by `crew acp send/permission/cancel/stop`
- `state.json` - current execution substate
- `events.jsonl` - raw ACP event log (one JSON event per line)

## Testing notes

IPC, state storage, and event logging are port interfaces. Tests can swap in
in-memory implementations to verify permission waiting, end-turn handling, and
log generation without file I/O.
