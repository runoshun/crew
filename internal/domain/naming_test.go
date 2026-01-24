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

func TestPathFunctions(t *testing.T) {
	crewDir := "/repo/.git/crew"

	t.Run("ScriptPath", func(t *testing.T) {
		got := ScriptPath(crewDir, 1)
		want := "/repo/.git/crew/scripts/task-1.sh"
		if got != want {
			t.Errorf("ScriptPath(%q, 1) = %q, want %q", crewDir, got, want)
		}
	})

	t.Run("PromptPath", func(t *testing.T) {
		got := PromptPath(crewDir, 42)
		want := "/repo/.git/crew/scripts/task-42-prompt.txt"
		if got != want {
			t.Errorf("PromptPath(%q, 42) = %q, want %q", crewDir, got, want)
		}
	})

	t.Run("ReviewScriptPath", func(t *testing.T) {
		got := ReviewScriptPath(crewDir, 1)
		want := "/repo/.git/crew/scripts/review-1.sh"
		if got != want {
			t.Errorf("ReviewScriptPath(%q, 1) = %q, want %q", crewDir, got, want)
		}
	})

	t.Run("ReviewPromptPath", func(t *testing.T) {
		got := ReviewPromptPath(crewDir, 1)
		want := "/repo/.git/crew/scripts/review-1-prompt.txt"
		if got != want {
			t.Errorf("ReviewPromptPath(%q, 1) = %q, want %q", crewDir, got, want)
		}
	})

	t.Run("TaskLogPath", func(t *testing.T) {
		got := TaskLogPath(crewDir, 1)
		want := "/repo/.git/crew/logs/task-1.log"
		if got != want {
			t.Errorf("TaskLogPath(%q, 1) = %q, want %q", crewDir, got, want)
		}
	})

	t.Run("GlobalLogPath", func(t *testing.T) {
		got := GlobalLogPath(crewDir)
		want := "/repo/.git/crew/logs/crew.log"
		if got != want {
			t.Errorf("GlobalLogPath(%q) = %q, want %q", crewDir, got, want)
		}
	})

	t.Run("SessionLogPath", func(t *testing.T) {
		got := SessionLogPath(crewDir, "crew-1")
		want := "/repo/.git/crew/logs/crew-1.log"
		if got != want {
			t.Errorf("SessionLogPath(%q, %q) = %q, want %q", crewDir, "crew-1", got, want)
		}
	})

	t.Run("SessionLogPath_review", func(t *testing.T) {
		got := SessionLogPath(crewDir, "crew-1-review")
		want := "/repo/.git/crew/logs/crew-1-review.log"
		if got != want {
			t.Errorf("SessionLogPath(%q, %q) = %q, want %q", crewDir, "crew-1-review", got, want)
		}
	})

	t.Run("TasksStorePath", func(t *testing.T) {
		got := TasksStorePath(crewDir)
		want := "/repo/.git/crew/tasks.json"
		if got != want {
			t.Errorf("TasksStorePath(%q) = %q, want %q", crewDir, got, want)
		}
	})

	t.Run("TmuxSocketPath", func(t *testing.T) {
		got := TmuxSocketPath(crewDir)
		want := "/repo/.git/crew/tmux.sock"
		if got != want {
			t.Errorf("TmuxSocketPath(%q) = %q, want %q", crewDir, got, want)
		}
	})

	t.Run("TmuxConfigPath", func(t *testing.T) {
		got := TmuxConfigPath(crewDir)
		want := "/repo/.git/crew/tmux.conf"
		if got != want {
			t.Errorf("TmuxConfigPath(%q) = %q, want %q", crewDir, got, want)
		}
	})

	t.Run("WorktreePath", func(t *testing.T) {
		worktreeDir := "/repo/.git/crew/worktrees"
		got := WorktreePath(worktreeDir, 1)
		want := "/repo/.git/crew/worktrees/1"
		if got != want {
			t.Errorf("WorktreePath(%q, 1) = %q, want %q", worktreeDir, got, want)
		}
	})
}
