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
		name  string
		agent WorkerAgent
		data  CommandData
		want  string
	}{
		{
			name: "claude style",
			agent: WorkerAgent{
				CommandTemplate: "{{.Command}} {{.SystemArgs}} {{.Args}} {{.Prompt}}",
				Command:         "claude",
				SystemArgs:      "--permission-mode acceptEdits",
				Args:            "--model opus",
			},
			data: CommandData{Prompt: "Fix the bug"},
			want: "claude --permission-mode acceptEdits --model opus Fix the bug",
		},
		{
			name: "opencode style with -p flag",
			agent: WorkerAgent{
				CommandTemplate: "{{.Command}} {{.Args}} -p {{.Prompt}}",
				Command:         "opencode",
				SystemArgs:      "",
				Args:            "-m gpt-4",
			},
			data: CommandData{Prompt: "Implement feature"},
			want: "opencode -m gpt-4 -p Implement feature",
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
				Prompt: "Fix bug",
			},
			want: "claude --add-dir /repo/.git --model opus Fix bug",
		},
		{
			name: "with RepoRoot in Args",
			agent: WorkerAgent{
				CommandTemplate: "{{.Command}} {{.Args}} {{.Prompt}}",
				Command:         "myagent",
				SystemArgs:      "",
				Args:            "--root {{.RepoRoot}}",
			},
			data: CommandData{
				RepoRoot: "/home/user/project",
				Prompt:   "Do something",
			},
			want: "myagent --root /home/user/project Do something",
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
				Prompt: "Work on it",
			},
			want: "agent --task 42 Work on it",
		},
		{
			name: "with template in Prompt",
			agent: WorkerAgent{
				CommandTemplate: "{{.Command}} {{.Prompt}}",
				Command:         "agent",
			},
			data: CommandData{
				TaskID: 42,
				Title:  "Fix the bug",
				Prompt: "Task #{{.TaskID}}: {{.Title}}",
			},
			want: "agent Task #42: Fix the bug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.agent.RenderCommand(tt.data)
			if err != nil {
				t.Fatalf("RenderCommand() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("RenderCommand() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWorkerAgent_RenderCommand_InvalidTemplate(t *testing.T) {
	agent := WorkerAgent{
		CommandTemplate: "{{.Invalid",
		Command:         "test",
	}
	_, err := agent.RenderCommand(CommandData{Prompt: "test"})
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
	_, err := agent.RenderCommand(CommandData{Prompt: "test"})
	if err == nil {
		t.Error("expected error for invalid template in SystemArgs")
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
	if !strings.Contains(content, DefaultWorkerPrompt) {
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
