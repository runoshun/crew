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
		if worker.Model != "" {
			t.Errorf("%s.Model = %q, want empty (falls back to builtin default)", name, worker.Model)
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

func TestAgent_RenderCommand_WithModel(t *testing.T) {
	tests := []struct {
		name        string
		agent       WorkerAgent
		data        CommandData
		wantCommand string
	}{
		{
			name: "claude with model in SystemArgs",
			agent: WorkerAgent{
				CommandTemplate: "{{.Command}} {{.SystemArgs}} {{.Args}} {{.Prompt}}",
				Command:         "claude",
				SystemArgs:      "--model {{.Model}}",
				Args:            "--verbose",
			},
			data:        CommandData{TaskID: 1, Model: "sonnet"},
			wantCommand: `claude --model sonnet --verbose "$PROMPT"`,
		},
		{
			name: "opencode with model in SystemArgs",
			agent: WorkerAgent{
				CommandTemplate: "{{.Command}} {{.SystemArgs}} {{.Args}} --prompt {{.Prompt}}",
				Command:         "opencode",
				SystemArgs:      "-m {{.Model}}",
				Args:            "",
			},
			data:        CommandData{TaskID: 1, Model: "gpt-4o"},
			wantCommand: `opencode -m gpt-4o  --prompt "$PROMPT"`,
		},
		{
			name: "model with empty value in SystemArgs",
			agent: WorkerAgent{
				CommandTemplate: "{{.Command}} {{.SystemArgs}} {{.Prompt}}",
				Command:         "agent",
				SystemArgs:      "--model {{.Model}}",
			},
			data:        CommandData{TaskID: 1, Model: ""},
			wantCommand: `agent --model  "$PROMPT"`,
		},
		{
			name: "user Args not affected by model",
			agent: WorkerAgent{
				CommandTemplate: "{{.Command}} {{.SystemArgs}} {{.Args}} {{.Prompt}}",
				Command:         "claude",
				SystemArgs:      "--model {{.Model}}",
				Args:            "--custom-flag",
			},
			data:        CommandData{TaskID: 1, Model: "opus"},
			wantCommand: `claude --model opus --custom-flag "$PROMPT"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.agent.RenderCommand(tt.data, `"$PROMPT"`, "default prompt")
			if err != nil {
				t.Fatalf("RenderCommand() error = %v", err)
			}
			if result.Command != tt.wantCommand {
				t.Errorf("RenderCommand().Command = %q, want %q", result.Command, tt.wantCommand)
			}
		})
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

func TestConfig_ResolveInheritance(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		want    map[string]WorkerAgent
		wantErr error
	}{
		{
			name: "simple inheritance",
			config: &Config{
				Workers: map[string]WorkerAgent{
					"base": {
						CommandTemplate: "{{.Command}} {{.Args}}",
						Command:         "base-cmd",
						Args:            "--base-arg",
					},
					"child": {
						Inherit: "base",
						Args:    "--child-arg",
					},
				},
			},
			want: map[string]WorkerAgent{
				"base": {
					CommandTemplate: "{{.Command}} {{.Args}}",
					Command:         "base-cmd",
					Args:            "--base-arg",
				},
				"child": {
					CommandTemplate: "{{.Command}} {{.Args}}",
					Command:         "base-cmd",
					Args:            "--child-arg",
				},
			},
			wantErr: nil,
		},
		{
			name: "partial override",
			config: &Config{
				Workers: map[string]WorkerAgent{
					"base": {
						CommandTemplate: "{{.Command}} {{.SystemArgs}} {{.Args}}",
						Command:         "base-cmd",
						SystemArgs:      "--system",
						Args:            "--base-arg",
						Prompt:          "base prompt",
						Model:           "base-model",
					},
					"child": {
						Inherit:    "base",
						SystemArgs: "--child-system",
						Model:      "child-model",
					},
				},
			},
			want: map[string]WorkerAgent{
				"base": {
					CommandTemplate: "{{.Command}} {{.SystemArgs}} {{.Args}}",
					Command:         "base-cmd",
					SystemArgs:      "--system",
					Args:            "--base-arg",
					Prompt:          "base prompt",
					Model:           "base-model",
				},
				"child": {
					CommandTemplate: "{{.Command}} {{.SystemArgs}} {{.Args}}",
					Command:         "base-cmd",
					SystemArgs:      "--child-system",
					Args:            "--base-arg",
					Prompt:          "base prompt",
					Model:           "child-model",
				},
			},
			wantErr: nil,
		},
		{
			name: "model inheritance fallback",
			config: &Config{
				Workers: map[string]WorkerAgent{
					"base": {
						CommandTemplate: "{{.Command}}",
						Command:         "base",
						Model:           "base-model",
					},
					"child": {
						Inherit: "base",
					},
				},
			},
			want: map[string]WorkerAgent{
				"base": {
					CommandTemplate: "{{.Command}}",
					Command:         "base",
					Model:           "base-model",
				},
				"child": {
					CommandTemplate: "{{.Command}}",
					Command:         "base",
					Model:           "base-model",
				},
			},
			wantErr: nil,
		},
		{
			name: "multi-level inheritance",
			config: &Config{
				Workers: map[string]WorkerAgent{
					"base": {
						CommandTemplate: "{{.Command}} {{.Args}}",
						Command:         "base-cmd",
						Args:            "--base",
					},
					"middle": {
						Inherit: "base",
						Args:    "--middle",
					},
					"child": {
						Inherit: "middle",
						Args:    "--child",
					},
				},
			},
			want: map[string]WorkerAgent{
				"base": {
					CommandTemplate: "{{.Command}} {{.Args}}",
					Command:         "base-cmd",
					Args:            "--base",
				},
				"middle": {
					CommandTemplate: "{{.Command}} {{.Args}}",
					Command:         "base-cmd",
					Args:            "--middle",
				},
				"child": {
					CommandTemplate: "{{.Command}} {{.Args}}",
					Command:         "base-cmd",
					Args:            "--child",
				},
			},
			wantErr: nil,
		},
		{
			name: "no inheritance",
			config: &Config{
				Workers: map[string]WorkerAgent{
					"worker1": {
						CommandTemplate: "{{.Command}}",
						Command:         "cmd1",
					},
					"worker2": {
						CommandTemplate: "{{.Command}}",
						Command:         "cmd2",
					},
				},
			},
			want: map[string]WorkerAgent{
				"worker1": {
					CommandTemplate: "{{.Command}}",
					Command:         "cmd1",
				},
				"worker2": {
					CommandTemplate: "{{.Command}}",
					Command:         "cmd2",
				},
			},
			wantErr: nil,
		},
		{
			name: "circular dependency - direct",
			config: &Config{
				Workers: map[string]WorkerAgent{
					"a": {
						Inherit: "b",
					},
					"b": {
						Inherit: "a",
					},
				},
			},
			wantErr: ErrCircularInheritance,
		},
		{
			name: "circular dependency - indirect",
			config: &Config{
				Workers: map[string]WorkerAgent{
					"a": {
						Inherit: "b",
					},
					"b": {
						Inherit: "c",
					},
					"c": {
						Inherit: "a",
					},
				},
			},
			wantErr: ErrCircularInheritance,
		},
		{
			name: "parent not found",
			config: &Config{
				Workers: map[string]WorkerAgent{
					"child": {
						Inherit: "nonexistent",
					},
				},
			},
			wantErr: ErrInheritParentNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ResolveInheritance()

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("ResolveInheritance() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("ResolveInheritance() unexpected error = %v", err)
			}

			for name, want := range tt.want {
				got, ok := tt.config.Workers[name]
				if !ok {
					t.Errorf("worker %q not found in config", name)
					continue
				}

				if got.Inherit != "" {
					t.Errorf("worker %q: Inherit = %q, want empty (should be cleared after resolution)", name, got.Inherit)
				}
				if got.CommandTemplate != want.CommandTemplate {
					t.Errorf("worker %q: CommandTemplate = %q, want %q", name, got.CommandTemplate, want.CommandTemplate)
				}
				if got.Command != want.Command {
					t.Errorf("worker %q: Command = %q, want %q", name, got.Command, want.Command)
				}
				if got.SystemArgs != want.SystemArgs {
					t.Errorf("worker %q: SystemArgs = %q, want %q", name, got.SystemArgs, want.SystemArgs)
				}
				if got.Args != want.Args {
					t.Errorf("worker %q: Args = %q, want %q", name, got.Args, want.Args)
				}
				if got.Prompt != want.Prompt {
					t.Errorf("worker %q: Prompt = %q, want %q", name, got.Prompt, want.Prompt)
				}
				if got.Model != want.Model {
					t.Errorf("worker %q: Model = %q, want %q", name, got.Model, want.Model)
				}
			}
		})
	}
}
