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

	// Check Log level
	if cfg.Log.Level != DefaultLogLevel {
		t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, DefaultLogLevel)
	}

	// Check maps are initialized but empty (builtin agents are registered by infra/builtin)
	if cfg.Agents == nil {
		t.Error("Agents should not be nil")
	}
	if len(cfg.Agents) != 0 {
		t.Errorf("Agents should be empty, got %d entries", len(cfg.Agents))
	}
	if cfg.Workers == nil {
		t.Error("Workers should not be nil")
	}
	if len(cfg.Workers) != 0 {
		t.Errorf("Workers should be empty, got %d entries", len(cfg.Workers))
	}
	if cfg.Managers == nil {
		t.Error("Managers should not be nil")
	}
	if len(cfg.Managers) != 0 {
		t.Errorf("Managers should be empty, got %d entries", len(cfg.Managers))
	}

	// Check WorkersConfig has default system prompt
	if cfg.WorkersConfig.SystemPrompt != DefaultSystemPrompt {
		t.Errorf("WorkersConfig.SystemPrompt = %q, want %q", cfg.WorkersConfig.SystemPrompt, DefaultSystemPrompt)
	}
	if cfg.WorkersConfig.Prompt != "" {
		t.Errorf("WorkersConfig.Prompt = %q, want empty", cfg.WorkersConfig.Prompt)
	}

	// Check ManagersConfig has default system prompt
	if cfg.ManagersConfig.SystemPrompt != DefaultManagerSystemPrompt {
		t.Errorf("ManagersConfig.SystemPrompt = %q, want %q", cfg.ManagersConfig.SystemPrompt, DefaultManagerSystemPrompt)
	}
	if cfg.ManagersConfig.Prompt != "" {
		t.Errorf("ManagersConfig.Prompt = %q, want empty", cfg.ManagersConfig.Prompt)
	}
}

func TestAgent_RenderCommand(t *testing.T) {
	tests := []struct {
		name                string
		agent               Worker
		data                CommandData
		promptOverride      string
		defaultSystemPrompt string
		defaultPrompt       string
		wantCommand         string
		wantPrompt          string
	}{
		{
			name: "claude style with shell variable",
			agent: Worker{
				CommandTemplate: "{{.Command}} {{.SystemArgs}} {{.Args}} {{.Prompt}}",
				Command:         "claude",
				SystemArgs:      "--permission-mode acceptEdits",
				Args:            "--model opus",
			},
			data:                CommandData{TaskID: 1, Title: "Fix bug"},
			promptOverride:      `"$PROMPT"`,
			defaultSystemPrompt: "System: {{.TaskID}}",
			defaultPrompt:       "Task: {{.Title}}",
			wantCommand:         `claude --permission-mode acceptEdits --model opus "$PROMPT"`,
			wantPrompt:          "System: 1\n\nTask: Fix bug",
		},
		{
			name: "opencode style with -p flag",
			agent: Worker{
				CommandTemplate: "{{.Command}} {{.Args}} -p {{.Prompt}}",
				Command:         "opencode",
				Args:            "-m gpt-4",
			},
			data:                CommandData{TaskID: 2, Title: "Add feature"},
			promptOverride:      `"$PROMPT"`,
			defaultSystemPrompt: "Work on: {{.Title}}",
			defaultPrompt:       "",
			wantCommand:         `opencode -m gpt-4 -p "$PROMPT"`,
			wantPrompt:          "Work on: Add feature",
		},
		{
			name: "with GitDir in SystemArgs",
			agent: Worker{
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
			promptOverride:      `"$PROMPT"`,
			defaultSystemPrompt: "Do it",
			defaultPrompt:       "",
			wantCommand:         `claude --add-dir /repo/.git --model opus "$PROMPT"`,
			wantPrompt:          "Do it",
		},
		{
			name: "worker-specific prompt overrides default",
			agent: Worker{
				CommandTemplate: "{{.Command}} {{.Prompt}}",
				Command:         "agent",
				SystemPrompt:    "Sys: {{.TaskID}}",
				Prompt:          "Custom: {{.Title}} (Task #{{.TaskID}})",
			},
			data: CommandData{
				TaskID: 42,
				Title:  "Fix the bug",
			},
			promptOverride:      `"$PROMPT"`,
			defaultSystemPrompt: "This should not be used",
			defaultPrompt:       "Nor this",
			wantCommand:         `agent "$PROMPT"`,
			wantPrompt:          "Sys: 42\n\nCustom: Fix the bug (Task #42)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.agent.RenderCommand(tt.data, tt.promptOverride, tt.defaultSystemPrompt, tt.defaultPrompt)
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

func TestWorker_RenderCommand_InvalidTemplate(t *testing.T) {
	agent := Worker{
		CommandTemplate: "{{.Invalid",
		Command:         "test",
	}
	_, err := agent.RenderCommand(CommandData{}, `"$PROMPT"`, "sys", "default")
	if err == nil {
		t.Error("expected error for invalid template in CommandTemplate")
	}
}

func TestWorker_RenderCommand_InvalidSystemArgsTemplate(t *testing.T) {
	agent := Worker{
		CommandTemplate: "{{.Command}} {{.SystemArgs}}",
		Command:         "test",
		SystemArgs:      "{{.Invalid",
	}
	_, err := agent.RenderCommand(CommandData{}, `"$PROMPT"`, "sys", "default")
	if err == nil {
		t.Error("expected error for invalid template in SystemArgs")
	}
}

func TestWorker_RenderCommand_InvalidPromptTemplate(t *testing.T) {
	agent := Worker{
		CommandTemplate: "{{.Command}} {{.Prompt}}",
		Command:         "test",
		Prompt:          "{{.Invalid",
	}
	_, err := agent.RenderCommand(CommandData{}, `"$PROMPT"`, "sys", "default")
	if err == nil {
		t.Error("expected error for invalid template in Prompt")
	}
}

func TestAgent_RenderCommand_WithModel(t *testing.T) {
	tests := []struct {
		name        string
		agent       Worker
		data        CommandData
		wantCommand string
	}{
		{
			name: "claude with model in SystemArgs",
			agent: Worker{
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
			agent: Worker{
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
			agent: Worker{
				CommandTemplate: "{{.Command}} {{.SystemArgs}} {{.Prompt}}",
				Command:         "agent",
				SystemArgs:      "--model {{.Model}}",
			},
			data:        CommandData{TaskID: 1, Model: ""},
			wantCommand: `agent --model  "$PROMPT"`,
		},
		{
			name: "user Args not affected by model",
			agent: Worker{
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
			result, err := tt.agent.RenderCommand(tt.data, `"$PROMPT"`, "sys", "default prompt")
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
	// Create a config with some workers to test dynamic generation
	cfg := NewDefaultConfig()
	cfg.Workers["test-worker"] = Worker{
		Agent:       "test-agent",
		Args:        "--test-args",
		Description: "Test worker description",
	}

	content := RenderConfigTemplate(cfg)

	// Check that default values from constants are embedded
	if !strings.Contains(content, DefaultLogLevel) {
		t.Errorf("expected log level %q to be embedded in template", DefaultLogLevel)
	}
	if !strings.Contains(content, DefaultWorkerName) {
		t.Errorf("expected default_worker %q to be embedded in template", DefaultWorkerName)
	}
	formattedSysPrompt := formatPromptForTemplate(DefaultSystemPrompt)
	if !strings.Contains(content, "# system_prompt = "+formattedSysPrompt) {
		t.Errorf("expected system prompt to be embedded in template")
	}

	// Check that Go template syntax for command_template is preserved
	if !strings.Contains(content, "{{.Command}}") {
		t.Error("expected {{.Command}} to be preserved in template")
	}

	// Check header is present
	if !strings.Contains(content, "# git-crew configuration") {
		t.Error("expected header to be present")
	}

	// Check manager section is present
	if !strings.Contains(content, "## Manager configuration") {
		t.Error("expected manager configuration section to be present")
	}
	if !strings.Contains(content, "[managers.default]") {
		t.Error("expected [managers.default] to be present in template")
	}

	// Check that dynamically registered worker is included
	if !strings.Contains(content, "[workers.test-worker]") {
		t.Error("expected [workers.test-worker] to be present in template")
	}
	if !strings.Contains(content, `agent = "test-agent"`) {
		t.Error("expected test worker agent to be present in template")
	}
}

func TestConfig_ResolveInheritance(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		want    map[string]Worker
		wantErr error
	}{
		{
			name: "simple inheritance",
			config: &Config{
				Workers: map[string]Worker{
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
			want: map[string]Worker{
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
				Workers: map[string]Worker{
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
			want: map[string]Worker{
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
				Workers: map[string]Worker{
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
			want: map[string]Worker{
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
				Workers: map[string]Worker{
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
			want: map[string]Worker{
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
				Workers: map[string]Worker{
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
			want: map[string]Worker{
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
				Workers: map[string]Worker{
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
				Workers: map[string]Worker{
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
				Workers: map[string]Worker{
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
