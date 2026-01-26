package domain

import (
	"reflect"
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

	// Check maps are initialized but empty (builtin agents are registered by infra/config)
	if cfg.Agents == nil {
		t.Error("Agents should not be nil")
	}
	if len(cfg.Agents) != 0 {
		t.Errorf("Agents should be empty, got %d entries", len(cfg.Agents))
	}

	// Check AgentsConfig defaults are empty
	if cfg.AgentsConfig.DefaultWorker != "" {
		t.Errorf("AgentsConfig.DefaultWorker = %q, want empty", cfg.AgentsConfig.DefaultWorker)
	}
	if cfg.AgentsConfig.DefaultManager != "" {
		t.Errorf("AgentsConfig.DefaultManager = %q, want empty", cfg.AgentsConfig.DefaultManager)
	}
}

func TestAgent_RenderCommand(t *testing.T) {
	tests := []struct {
		agent               Agent
		name                string
		promptOverride      string
		defaultSystemPrompt string
		defaultPrompt       string
		wantCommand         string
		wantPrompt          string
		data                CommandData
	}{
		{
			name: "basic command template with args",
			agent: Agent{
				CommandTemplate: "agent {{.Args}} {{.Prompt}}",
				Args:            "--model opus",
			},
			data:                CommandData{TaskID: 1, Title: "Fix bug"},
			promptOverride:      `"$PROMPT"`,
			defaultSystemPrompt: "System: {{.TaskID}}",
			defaultPrompt:       "Task: {{.Title}}",
			wantCommand:         `agent --model opus "$PROMPT"`,
			wantPrompt:          "System: 1\n\nTask: Fix bug",
		},
		{
			name: "opencode style with -m flag",
			agent: Agent{
				CommandTemplate: "opencode -m {{.Model}} {{.Args}} -p {{.Prompt}}",
				Args:            "",
			},
			data:                CommandData{TaskID: 2, Title: "Add feature", Model: "gpt-4"},
			promptOverride:      `"$PROMPT"`,
			defaultSystemPrompt: "Work on: {{.Title}}",
			defaultPrompt:       "",
			wantCommand:         `opencode -m gpt-4  -p "$PROMPT"`,
			wantPrompt:          "Work on: Add feature",
		},
		{
			name: "agent-specific prompt concatenates with default",
			agent: Agent{
				CommandTemplate: "agent {{.Prompt}}",
				SystemPrompt:    "Sys: {{.TaskID}}",
				Prompt:          "Custom: {{.Title}} (Task #{{.TaskID}})",
			},
			data: CommandData{
				TaskID: 42,
				Title:  "Fix the bug",
			},
			promptOverride:      `"$PROMPT"`,
			defaultSystemPrompt: "This should not be used",
			defaultPrompt:       "Default prompt",
			wantCommand:         `agent "$PROMPT"`,
			wantPrompt:          "Sys: 42\n\nDefault prompt\n\nCustom: Fix the bug (Task #42)",
		},
		{
			name: "agent prompt only (no default prompt)",
			agent: Agent{
				CommandTemplate: "agent {{.Prompt}}",
				Prompt:          "Agent specific prompt",
			},
			data:                CommandData{TaskID: 1},
			promptOverride:      `"$PROMPT"`,
			defaultSystemPrompt: "System",
			defaultPrompt:       "",
			wantCommand:         `agent "$PROMPT"`,
			wantPrompt:          "System\n\nAgent specific prompt",
		},
		{
			name: "default prompt only (no agent prompt)",
			agent: Agent{
				CommandTemplate: "agent {{.Prompt}}",
			},
			data:                CommandData{TaskID: 1},
			promptOverride:      `"$PROMPT"`,
			defaultSystemPrompt: "System",
			defaultPrompt:       "Default prompt",
			wantCommand:         `agent "$PROMPT"`,
			wantPrompt:          "System\n\nDefault prompt",
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

func TestAgent_RenderCommand_InvalidTemplate(t *testing.T) {
	agent := Agent{
		CommandTemplate: "{{.Invalid",
	}
	_, err := agent.RenderCommand(CommandData{}, `"$PROMPT"`, "sys", "default")
	if err == nil {
		t.Error("expected error for invalid template in CommandTemplate")
	}
}

func TestAgent_RenderCommand_InvalidArgsTemplate(t *testing.T) {
	agent := Agent{
		CommandTemplate: "agent {{.Args}}",
		Args:            "{{.Invalid",
	}
	_, err := agent.RenderCommand(CommandData{}, `"$PROMPT"`, "sys", "default")
	if err == nil {
		t.Error("expected error for invalid template in Args")
	}
}

func TestAgent_RenderCommand_InvalidPromptTemplate(t *testing.T) {
	agent := Agent{
		CommandTemplate: "agent {{.Prompt}}",
		Prompt:          "{{.Invalid",
	}
	_, err := agent.RenderCommand(CommandData{}, `"$PROMPT"`, "sys", "default")
	if err == nil {
		t.Error("expected error for invalid template in Prompt")
	}
}

func TestAgent_RenderCommand_WithModel(t *testing.T) {
	tests := []struct {
		agent       Agent
		name        string
		wantCommand string
		data        CommandData
	}{
		{
			name: "model in command template",
			agent: Agent{
				CommandTemplate: "agent --model {{.Model}} {{.Args}} {{.Prompt}}",
				Args:            "--verbose",
			},
			data:        CommandData{TaskID: 1, Model: "sonnet"},
			wantCommand: `agent --model sonnet --verbose "$PROMPT"`,
		},
		{
			name: "model with empty value",
			agent: Agent{
				CommandTemplate: "agent --model {{.Model}} {{.Prompt}}",
			},
			data:        CommandData{TaskID: 1, Model: ""},
			wantCommand: `agent --model  "$PROMPT"`,
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

func TestAgent_RenderCommand_WithContinue(t *testing.T) {
	tests := []struct {
		agent       Agent
		name        string
		wantCommand string
		data        CommandData
	}{
		{
			name: "continue flag enabled with -c",
			agent: Agent{
				CommandTemplate: "agent --model {{.Model}} {{.Args}}{{if .Continue}} -c{{end}} {{.Prompt}}",
				Args:            "--verbose",
			},
			data:        CommandData{TaskID: 1, Model: "sonnet", Continue: true},
			wantCommand: `agent --model sonnet --verbose -c "$PROMPT"`,
		},
		{
			name: "continue flag disabled",
			agent: Agent{
				CommandTemplate: "agent --model {{.Model}} {{.Args}}{{if .Continue}} -c{{end}} {{.Prompt}}",
				Args:            "--verbose",
			},
			data:        CommandData{TaskID: 1, Model: "sonnet", Continue: false},
			wantCommand: `agent --model sonnet --verbose "$PROMPT"`,
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
	// Create a config with some agents to test dynamic generation
	cfg := NewDefaultConfig()
	cfg.Agents["test-agent"] = Agent{
		Role:        RoleWorker,
		Args:        "--test-args",
		Description: "Test agent description",
	}

	content := RenderConfigTemplate(cfg)

	// Check that default values from constants are embedded
	if !strings.Contains(content, DefaultLogLevel) {
		t.Errorf("expected log level %q to be embedded in template", DefaultLogLevel)
	}

	// Check that Go template syntax for command_template is preserved
	if !strings.Contains(content, "{{.Model}}") {
		t.Error("expected {{.Model}} to be preserved in template")
	}

	// Check header is present
	if !strings.Contains(content, "# git-crew configuration") {
		t.Error("expected header to be present")
	}
}

func TestConfig_EnabledAgents(t *testing.T) {
	tests := []struct {
		config   *Config
		name     string
		wantKeys []string
		wantLen  int
	}{
		{
			name: "no disabled agents",
			config: &Config{
				Agents: map[string]Agent{
					"agent1": {CommandTemplate: "cmd1"},
					"agent2": {CommandTemplate: "cmd2"},
				},
				AgentsConfig: AgentsConfig{
					DisabledAgents: []string{},
				},
			},
			wantLen:  2,
			wantKeys: []string{"agent1", "agent2"},
		},
		{
			name: "some agents disabled",
			config: &Config{
				Agents: map[string]Agent{
					"agent1":   {CommandTemplate: "cmd1"},
					"agent2":   {CommandTemplate: "cmd2"},
					"disabled": {CommandTemplate: "cmd3"},
				},
				AgentsConfig: AgentsConfig{
					DisabledAgents: []string{"disabled"},
				},
			},
			wantLen:  2,
			wantKeys: []string{"agent1", "agent2"},
		},
		{
			name: "wildcard disabled pattern",
			config: &Config{
				Agents: map[string]Agent{
					"oc-small":  {CommandTemplate: "cmd1"},
					"oc-medium": {CommandTemplate: "cmd2"},
					"claude":    {CommandTemplate: "cmd3"},
				},
				AgentsConfig: AgentsConfig{
					DisabledAgents: []string{"oc-*"},
				},
			},
			wantLen:  1,
			wantKeys: []string{"claude"},
		},
		{
			name: "exclusion pattern re-enables agent",
			config: &Config{
				Agents: map[string]Agent{
					"oc-small":  {CommandTemplate: "cmd1"},
					"oc-medium": {CommandTemplate: "cmd2"},
					"claude":    {CommandTemplate: "cmd3"},
				},
				AgentsConfig: AgentsConfig{
					DisabledAgents: []string{"oc-*", "!oc-medium"},
				},
			},
			wantLen:  2,
			wantKeys: []string{"claude", "oc-medium"},
		},
		{
			name: "all agents disabled",
			config: &Config{
				Agents: map[string]Agent{
					"agent1": {CommandTemplate: "cmd1"},
					"agent2": {CommandTemplate: "cmd2"},
				},
				AgentsConfig: AgentsConfig{
					DisabledAgents: []string{"agent1", "agent2"},
				},
			},
			wantLen:  0,
			wantKeys: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.EnabledAgents()

			if len(got) != tt.wantLen {
				t.Errorf("EnabledAgents() returned %d agents, want %d", len(got), tt.wantLen)
			}

			for _, key := range tt.wantKeys {
				if _, ok := got[key]; !ok {
					t.Errorf("EnabledAgents() missing expected agent %q", key)
				}
			}

			// Verify disabled agents are not present
			for name := range tt.config.Agents {
				if IsAgentDisabled(name, tt.config.AgentsConfig.DisabledAgents) {
					if _, ok := got[name]; ok {
						t.Errorf("EnabledAgents() should not contain disabled agent %q", name)
					}
				}
			}
		})
	}
}

func TestConfig_GetReviewerAgents(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   []string
	}{
		{
			name: "returns only reviewer role agents",
			config: &Config{
				Agents: map[string]Agent{
					"worker1":   {Role: RoleWorker},
					"reviewer1": {Role: RoleReviewer},
					"reviewer2": {Role: RoleReviewer},
					"manager1":  {Role: RoleManager},
				},
			},
			want: []string{"reviewer1", "reviewer2"},
		},
		{
			name: "excludes hidden reviewer agents",
			config: &Config{
				Agents: map[string]Agent{
					"reviewer1":        {Role: RoleReviewer},
					"reviewer2":        {Role: RoleReviewer, Hidden: true},
					"reviewer3":        {Role: RoleReviewer},
					"hidden-reviewer4": {Role: RoleReviewer, Hidden: true},
				},
			},
			want: []string{"reviewer1", "reviewer3"},
		},
		{
			name: "excludes disabled agents",
			config: &Config{
				Agents: map[string]Agent{
					"reviewer1":  {Role: RoleReviewer},
					"reviewer2":  {Role: RoleReviewer},
					"oc-reviews": {Role: RoleReviewer},
				},
				AgentsConfig: AgentsConfig{
					DisabledAgents: []string{"oc-*"},
				},
			},
			want: []string{"reviewer1", "reviewer2"},
		},
		{
			name: "returns sorted list",
			config: &Config{
				Agents: map[string]Agent{
					"z-reviewer": {Role: RoleReviewer},
					"a-reviewer": {Role: RoleReviewer},
					"m-reviewer": {Role: RoleReviewer},
				},
			},
			want: []string{"a-reviewer", "m-reviewer", "z-reviewer"},
		},
		{
			name: "returns empty slice when no reviewers",
			config: &Config{
				Agents: map[string]Agent{
					"worker1":  {Role: RoleWorker},
					"manager1": {Role: RoleManager},
				},
			},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetReviewerAgents()

			if len(got) != len(tt.want) {
				t.Errorf("GetReviewerAgents() returned %d agents, want %d: got %v, want %v",
					len(got), len(tt.want), got, tt.want)
				return
			}

			for i, name := range tt.want {
				if got[i] != name {
					t.Errorf("GetReviewerAgents()[%d] = %q, want %q", i, got[i], name)
				}
			}
		})
	}
}

func TestConfig_ResolveInheritance(t *testing.T) {
	tests := []struct {
		wantErr error
		config  *Config
		want    map[string]Agent
		name    string
	}{
		{
			name: "simple inheritance",
			config: &Config{
				Agents: map[string]Agent{
					"base": {
						CommandTemplate: "agent {{.Args}}",
						Args:            "--base-arg",
					},
					"child": {
						Inherit: "base",
						Args:    "--child-arg",
					},
				},
			},
			want: map[string]Agent{
				"base": {
					CommandTemplate: "agent {{.Args}}",
					Args:            "--base-arg",
				},
				"child": {
					CommandTemplate: "agent {{.Args}}",
					Args:            "--child-arg",
				},
			},
			wantErr: nil,
		},
		{
			name: "partial override",
			config: &Config{
				Agents: map[string]Agent{
					"base": {
						CommandTemplate: "agent {{.Args}}",
						Args:            "--base-arg",
						Prompt:          "base prompt",
						DefaultModel:    "base-model",
						Env: map[string]string{
							"DEBUG": "0",
							"TOKEN": "base",
						},
					},
					"child": {
						Inherit:      "base",
						DefaultModel: "child-model",
						Env: map[string]string{
							"TOKEN": "child",
							"TRACE": "1",
						},
					},
				},
			},
			want: map[string]Agent{
				"base": {
					CommandTemplate: "agent {{.Args}}",
					Args:            "--base-arg",
					Prompt:          "base prompt",
					DefaultModel:    "base-model",
					Env: map[string]string{
						"DEBUG": "0",
						"TOKEN": "base",
					},
				},
				"child": {
					CommandTemplate: "agent {{.Args}}",
					Args:            "--base-arg",
					Prompt:          "base prompt",
					DefaultModel:    "child-model",
					Env: map[string]string{
						"DEBUG": "0",
						"TOKEN": "child",
						"TRACE": "1",
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "multi-level inheritance",
			config: &Config{
				Agents: map[string]Agent{
					"base": {
						CommandTemplate: "agent {{.Args}}",
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
			want: map[string]Agent{
				"base": {
					CommandTemplate: "agent {{.Args}}",
					Args:            "--base",
				},
				"middle": {
					CommandTemplate: "agent {{.Args}}",
					Args:            "--middle",
				},
				"child": {
					CommandTemplate: "agent {{.Args}}",
					Args:            "--child",
				},
			},
			wantErr: nil,
		},
		{
			name: "no inheritance",
			config: &Config{
				Agents: map[string]Agent{
					"agent1": {
						CommandTemplate: "cmd1",
					},
					"agent2": {
						CommandTemplate: "cmd2",
					},
				},
			},
			want: map[string]Agent{
				"agent1": {
					CommandTemplate: "cmd1",
				},
				"agent2": {
					CommandTemplate: "cmd2",
				},
			},
			wantErr: nil,
		},
		{
			name: "circular dependency - direct",
			config: &Config{
				Agents: map[string]Agent{
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
				Agents: map[string]Agent{
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
				Agents: map[string]Agent{
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
				got, ok := tt.config.Agents[name]
				if !ok {
					t.Errorf("agent %q not found in config", name)
					continue
				}

				if got.Inherit != "" {
					t.Errorf("agent %q: Inherit = %q, want empty (should be cleared after resolution)", name, got.Inherit)
				}
				if got.CommandTemplate != want.CommandTemplate {
					t.Errorf("agent %q: CommandTemplate = %q, want %q", name, got.CommandTemplate, want.CommandTemplate)
				}
				if got.Args != want.Args {
					t.Errorf("agent %q: Args = %q, want %q", name, got.Args, want.Args)
				}
				if got.Prompt != want.Prompt {
					t.Errorf("agent %q: Prompt = %q, want %q", name, got.Prompt, want.Prompt)
				}
				if got.DefaultModel != want.DefaultModel {
					t.Errorf("agent %q: DefaultModel = %q, want %q", name, got.DefaultModel, want.DefaultModel)
				}
				if !reflect.DeepEqual(got.Env, want.Env) {
					t.Errorf("agent %q: Env = %v, want %v", name, got.Env, want.Env)
				}
			}
		})
	}
}
