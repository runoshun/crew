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
		taskID int
		issue  int
		want   string
	}{
		{"without issue", 1, 0, "crew-1"},
		{"without issue larger ID", 42, 0, "crew-42"},
		{"with issue", 1, 123, "crew-1-gh-123"},
		{"with issue larger numbers", 42, 456, "crew-42-gh-456"},
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
		taskID int
		want   string
	}{
		{1, "crew-1"},
		{42, "crew-42"},
		{999, "crew-999"},
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
