package domain

import (
	"strings"
	"testing"
)

func TestRepoCrewDir(t *testing.T) {
	got := RepoCrewDir("/home/user/project")
	want := "/home/user/project/.git/crew"
	if got != want {
		t.Errorf("RepoCrewDir() = %q, want %q", got, want)
	}
}

func TestRepoConfigPath(t *testing.T) {
	got := RepoConfigPath("/home/user/project")
	want := "/home/user/project/.git/crew/config.toml"
	if got != want {
		t.Errorf("RepoConfigPath() = %q, want %q", got, want)
	}
}

func TestGlobalCrewDir(t *testing.T) {
	got := GlobalCrewDir("/home/user/.config")
	want := "/home/user/.config/crew"
	if got != want {
		t.Errorf("GlobalCrewDir() = %q, want %q", got, want)
	}
}

func TestGlobalConfigPath(t *testing.T) {
	got := GlobalConfigPath("/home/user/.config")
	want := "/home/user/.config/crew/config.toml"
	if got != want {
		t.Errorf("GlobalConfigPath() = %q, want %q", got, want)
	}
}

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	if cfg.Log.Level != DefaultLogLevel {
		t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, DefaultLogLevel)
	}
	if cfg.WorkersConfig.Default != DefaultWorkerName {
		t.Errorf("WorkersConfig.Default = %q, want %q", cfg.WorkersConfig.Default, DefaultWorkerName)
	}
	if cfg.Workers == nil {
		t.Error("Workers should not be nil")
	}
	// Check WorkersConfig has default prompt
	if cfg.WorkersConfig.Prompt != DefaultWorkerPrompt {
		t.Errorf("WorkersConfig.Prompt = %q, want %q", cfg.WorkersConfig.Prompt, DefaultWorkerPrompt)
	}
	// Check builtin workers are configured from BuiltinWorkers
	for name, builtin := range BuiltinWorkers {
		worker, ok := cfg.Workers[name]
		if !ok {
			t.Errorf("expected %s worker to be configured", name)
			continue
		}
		if worker.CommandTemplate != builtin.CommandTemplate {
			t.Errorf("%s.CommandTemplate = %q, want %q", name, worker.CommandTemplate, builtin.CommandTemplate)
		}
		if worker.Command != builtin.Command {
			t.Errorf("%s.Command = %q, want %q", name, worker.Command, builtin.Command)
		}
		if worker.SystemArgs != builtin.SystemArgs {
			t.Errorf("%s.SystemArgs = %q, want %q", name, worker.SystemArgs, builtin.SystemArgs)
		}
		if worker.Args != builtin.DefaultArgs {
			t.Errorf("%s.Args = %q, want %q", name, worker.Args, builtin.DefaultArgs)
		}
		// Worker.Prompt is empty; defaults come from WorkersConfig.Prompt
		if worker.Prompt != "" {
			t.Errorf("%s.Prompt = %q, want empty (falls back to WorkersConfig.Prompt)", name, worker.Prompt)
		}
	}
}

func TestAgent_RenderCommand(t *testing.T) {
	tests := []struct {
		name           string
		agent          WorkerAgent
		data           CommandData
		promptOverride string
		defaultPrompt  string
		wantCommand    string
		wantPrompt     string
	}{
		{
			name: "claude style with shell variable",
			agent: WorkerAgent{
				CommandTemplate: "{{.Command}} {{.SystemArgs}} {{.Args}} {{.Prompt}}",
				Command:         "claude",
				SystemArgs:      "--permission-mode acceptEdits",
				Args:            "--model opus",
			},
			data:           CommandData{TaskID: 1, Title: "Fix bug"},
			promptOverride: `"$PROMPT"`,
			defaultPrompt:  "Task: {{.Title}}",
			wantCommand:    `claude --permission-mode acceptEdits --model opus "$PROMPT"`,
			wantPrompt:     "Task: Fix bug",
		},
		{
			name: "opencode style with -p flag",
			agent: WorkerAgent{
				CommandTemplate: "{{.Command}} {{.Args}} -p {{.Prompt}}",
				Command:         "opencode",
				Args:            "-m gpt-4",
			},
			data:           CommandData{TaskID: 2, Title: "Add feature"},
			promptOverride: `"$PROMPT"`,
			defaultPrompt:  "Work on: {{.Title}}",
			wantCommand:    `opencode -m gpt-4 -p "$PROMPT"`,
			wantPrompt:     "Work on: Add feature",
		},
		{
			name: "with GitDir in SystemArgs",
			agent: WorkerAgent{
				CommandTemplate: "{{.Command}} {{.SystemArgs}} {{.Args}} {{.Prompt}}",
				Command:         "claude",
				SystemArgs:      "--add-dir {{.GitDir}}",
				Args:            "--model opus",
			},
			data: CommandData{
				GitDir: "/repo/.git",
				TaskID: 3,
				Title:  "Fix",
			},
			promptOverride: `"$PROMPT"`,
			defaultPrompt:  "Do it",
			wantCommand:    `claude --add-dir /repo/.git --model opus "$PROMPT"`,
			wantPrompt:     "Do it",
		},
		{
			name: "worker-specific prompt overrides default",
			agent: WorkerAgent{
				CommandTemplate: "{{.Command}} {{.Prompt}}",
				Command:         "agent",
				Prompt:          "Custom: {{.Title}} (Task #{{.TaskID}})",
			},
			data: CommandData{
				TaskID: 42,
				Title:  "Fix the bug",
			},
			promptOverride: `"$PROMPT"`,
			defaultPrompt:  "This should not be used",
			wantCommand:    `agent "$PROMPT"`,
			wantPrompt:     "Custom: Fix the bug (Task #42)",
		},
		{
			name: "with task info in Args",
			agent: WorkerAgent{
				CommandTemplate: "{{.Command}} {{.Args}} {{.Prompt}}",
				Command:         "agent",
				Args:            "--task {{.TaskID}}",
			},
			data: CommandData{
				TaskID: 42,
				Title:  "Fix bug",
			},
			promptOverride: `"$PROMPT"`,
			defaultPrompt:  "Work on it",
			wantCommand:    `agent --task 42 "$PROMPT"`,
			wantPrompt:     "Work on it",
		},
		{
			name: "prompt template with all fields",
			agent: WorkerAgent{
				CommandTemplate: "{{.Command}} {{.Prompt}}",
				Command:         "agent",
				Prompt:          "Task #{{.TaskID}}: {{.Title}}\nBranch: {{.Branch}}\nWorktree: {{.Worktree}}",
			},
			data: CommandData{
				TaskID:   1,
				Title:    "Fix bug",
				Branch:   "crew-1",
				Worktree: "/path/to/worktree",
			},
			promptOverride: `"$PROMPT"`,
			defaultPrompt:  "",
			wantCommand:    `agent "$PROMPT"`,
			wantPrompt:     "Task #1: Fix bug\nBranch: crew-1\nWorktree: /path/to/worktree",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.agent.RenderCommand(tt.data, tt.promptOverride, tt.defaultPrompt)
			if err != nil {
				t.Fatalf("RenderCommand() error = %v", err)
			}
			if result.Command != tt.wantCommand {
				t.Errorf("RenderCommand().Command = %q, want %q", result.Command, tt.wantCommand)
			}
			if result.Prompt != tt.wantPrompt {
				t.Errorf("RenderCommand().Prompt = %q, want %q", result.Prompt, tt.wantPrompt)
			}
		})
	}
}

func TestWorkerAgent_RenderCommand_InvalidTemplate(t *testing.T) {
	agent := WorkerAgent{
		CommandTemplate: "{{.Invalid",
		Command:         "test",
	}
	_, err := agent.RenderCommand(CommandData{}, `"$PROMPT"`, "default")
	if err == nil {
		t.Error("expected error for invalid template in CommandTemplate")
	}
}

func TestWorkerAgent_RenderCommand_InvalidSystemArgsTemplate(t *testing.T) {
	agent := WorkerAgent{
		CommandTemplate: "{{.Command}} {{.SystemArgs}}",
		Command:         "test",
		SystemArgs:      "{{.Invalid",
	}
	_, err := agent.RenderCommand(CommandData{}, `"$PROMPT"`, "default")
	if err == nil {
		t.Error("expected error for invalid template in SystemArgs")
	}
}

func TestWorkerAgent_RenderCommand_InvalidPromptTemplate(t *testing.T) {
	agent := WorkerAgent{
		CommandTemplate: "{{.Command}} {{.Prompt}}",
		Command:         "test",
		Prompt:          "{{.Invalid",
	}
	_, err := agent.RenderCommand(CommandData{}, `"$PROMPT"`, "default")
	if err == nil {
		t.Error("expected error for invalid template in Prompt")
	}
}

func TestRenderConfigTemplate(t *testing.T) {
	content, err := RenderConfigTemplate()
	if err != nil {
		t.Fatalf("RenderConfigTemplate() error = %v", err)
	}

	// Check that default values from constants are embedded
	if !strings.Contains(content, DefaultLogLevel) {
		t.Errorf("expected log level %q to be embedded in template", DefaultLogLevel)
	}
	if !strings.Contains(content, DefaultWorkerName) {
		t.Errorf("expected default_worker %q to be embedded in template", DefaultWorkerName)
	}
	formattedPrompt := formatPromptForTemplate(DefaultWorkerPrompt)
	if !strings.Contains(content, "# prompt = "+formattedPrompt) {
		t.Errorf("expected worker prompt to be embedded in template")
	}
	for name, builtin := range BuiltinWorkers {
		if builtin.DefaultArgs != "" && !strings.Contains(content, builtin.DefaultArgs) {
			t.Errorf("expected %s args %q to be embedded in template", name, builtin.DefaultArgs)
		}
	}

	// Check that Go template syntax for command_template is preserved
	if !strings.Contains(content, "{{.Command}}") {
		t.Error("expected {{.Command}} to be preserved in template")
	}

	// Check header is present
	if !strings.Contains(content, "# git-crew configuration") {
		t.Error("expected header to be present")
	}
}
