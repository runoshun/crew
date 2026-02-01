package domain

import "testing"

func TestParseBranchTaskID(t *testing.T) {
	tests := []struct {
		name   string
		branch string
		wantID int
		wantOK bool
	}{
		// Valid crew branches
		{"crew-1", "crew-1", 1, true},
		{"crew-42", "crew-42", 42, true},
		{"crew-999", "crew-999", 999, true},
		{"with issue crew-1-gh-123", "crew-1-gh-123", 1, true},
		{"with issue crew-42-gh-456", "crew-42-gh-456", 42, true},

		// Invalid branches
		{"main branch", "main", 0, false},
		{"feature branch", "feature/foo", 0, false},
		{"empty string", "", 0, false},
		{"crew- without ID", "crew-", 0, false},
		{"crew without dash", "crew1", 0, false},
		{"similar but wrong prefix", "my-crew-1", 0, false},
		{"crew-gh- without ID", "crew-gh-123", 0, false},
		{"crew-1-gh- without issue", "crew-1-gh-", 0, false},
		{"non-numeric ID", "crew-abc", 0, false},
		{"non-numeric issue", "crew-1-gh-abc", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := ParseBranchTaskID(tt.branch)
			if gotID != tt.wantID {
				t.Errorf("ParseBranchTaskID(%q) ID = %d, want %d", tt.branch, gotID, tt.wantID)
			}
			if gotOK != tt.wantOK {
				t.Errorf("ParseBranchTaskID(%q) OK = %v, want %v", tt.branch, gotOK, tt.wantOK)
			}
		})
	}
}

func TestNamespaceFromEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  string
	}{
		{name: "empty", email: "", want: ""},
		{name: "missing at", email: "invalid", want: ""},
		{name: "simple", email: "user@example.com", want: "user"},
		{name: "sanitize", email: "User.Name+tag@example.com", want: "user-name-tag"},
		{name: "leading symbols", email: ".user@example.com", want: "user"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NamespaceFromEmail(tt.email)
			if got != tt.want {
				t.Errorf("NamespaceFromEmail(%q) = %q, want %q", tt.email, got, tt.want)
			}
		})
	}
}

func TestSanitizeNamespace(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "simple", input: "crew", want: "crew"},
		{name: "uppercase", input: "Crew", want: "crew"},
		{name: "spaces", input: "crew tasks", want: "crew-tasks"},
		{name: "symbols", input: "user.name+tag", want: "user-name-tag"},
		{name: "trim hyphens", input: "-name-", want: "name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeNamespace(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeNamespace(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBranchName(t *testing.T) {
	tests := []struct {
		name   string
		want   string
		taskID int
		issue  int
	}{
		{name: "without issue", taskID: 1, issue: 0, want: "crew-1"},
		{name: "without issue larger ID", taskID: 42, issue: 0, want: "crew-42"},
		{name: "with issue", taskID: 1, issue: 123, want: "crew-1-gh-123"},
		{name: "with issue larger numbers", taskID: 42, issue: 456, want: "crew-42-gh-456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BranchName(tt.taskID, tt.issue)
			if got != tt.want {
				t.Errorf("BranchName(%d, %d) = %q, want %q", tt.taskID, tt.issue, got, tt.want)
			}
		})
	}
}

func TestSessionName(t *testing.T) {
	tests := []struct {
		want   string
		taskID int
	}{
		{taskID: 1, want: "crew-1"},
		{taskID: 42, want: "crew-42"},
		{taskID: 999, want: "crew-999"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := SessionName(tt.taskID)
			if got != tt.want {
				t.Errorf("SessionName(%d) = %q, want %q", tt.taskID, got, tt.want)
			}
		})
	}
}

func TestACPSessionName(t *testing.T) {
	tests := []struct {
		want   string
		taskID int
	}{
		{taskID: 1, want: "crew-acp-1"},
		{taskID: 42, want: "crew-acp-42"},
		{taskID: 999, want: "crew-acp-999"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := ACPSessionName(tt.taskID)
			if got != tt.want {
				t.Errorf("ACPSessionName(%d) = %q, want %q", tt.taskID, got, tt.want)
			}
		})
	}
}

func TestPathFunctions(t *testing.T) {
	crewDir := "/repo/.crew"

	t.Run("ScriptPath", func(t *testing.T) {
		got := ScriptPath(crewDir, 1)
		want := "/repo/.crew/scripts/task-1.sh"
		if got != want {
			t.Errorf("ScriptPath(%q, 1) = %q, want %q", crewDir, got, want)
		}
	})

	t.Run("ACPScriptPath", func(t *testing.T) {
		got := ACPScriptPath(crewDir, 2)
		want := "/repo/.crew/scripts/acp-task-2.sh"
		if got != want {
			t.Errorf("ACPScriptPath(%q, 2) = %q, want %q", crewDir, got, want)
		}
	})

	t.Run("PromptPath", func(t *testing.T) {
		got := PromptPath(crewDir, 42)
		want := "/repo/.crew/scripts/task-42-prompt.txt"
		if got != want {
			t.Errorf("PromptPath(%q, 42) = %q, want %q", crewDir, got, want)
		}
	})

	t.Run("TaskLogPath", func(t *testing.T) {
		got := TaskLogPath(crewDir, 1)
		want := "/repo/.crew/logs/task-1.log"
		if got != want {
			t.Errorf("TaskLogPath(%q, 1) = %q, want %q", crewDir, got, want)
		}
	})

	t.Run("GlobalLogPath", func(t *testing.T) {
		got := GlobalLogPath(crewDir)
		want := "/repo/.crew/logs/crew.log"
		if got != want {
			t.Errorf("GlobalLogPath(%q) = %q, want %q", crewDir, got, want)
		}
	})

	t.Run("SessionLogPath", func(t *testing.T) {
		got := SessionLogPath(crewDir, "crew-1")
		want := "/repo/.crew/logs/crew-1.log"
		if got != want {
			t.Errorf("SessionLogPath(%q, %q) = %q, want %q", crewDir, "crew-1", got, want)
		}
	})

	t.Run("TasksStorePath", func(t *testing.T) {
		got := TasksStorePath(crewDir)
		want := "/repo/.crew/tasks"
		if got != want {
			t.Errorf("TasksStorePath(%q) = %q, want %q", crewDir, got, want)
		}
	})

	t.Run("TmuxSocketPath", func(t *testing.T) {
		got := TmuxSocketPath(crewDir)
		want := "/repo/.crew/tmux.sock"
		if got != want {
			t.Errorf("TmuxSocketPath(%q) = %q, want %q", crewDir, got, want)
		}
	})

	t.Run("TmuxConfigPath", func(t *testing.T) {
		got := TmuxConfigPath(crewDir)
		want := "/repo/.crew/tmux.conf"
		if got != want {
			t.Errorf("TmuxConfigPath(%q) = %q, want %q", crewDir, got, want)
		}
	})

	t.Run("WorktreePath", func(t *testing.T) {
		worktreeDir := "/repo/.crew/worktrees"
		got := WorktreePath(worktreeDir, 1)
		want := "/repo/.crew/worktrees/1"
		if got != want {
			t.Errorf("WorktreePath(%q, 1) = %q, want %q", worktreeDir, got, want)
		}
	})
}
