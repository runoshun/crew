package domain

import "testing"

func TestStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name   string
		from   Status
		to     Status
		expect bool
	}{
		// From todo
		{"todo -> in_progress", StatusTodo, StatusInProgress, true},
		{"todo -> closed", StatusTodo, StatusClosed, true},
		{"todo -> for_review", StatusTodo, StatusForReview, false},
		{"todo -> error", StatusTodo, StatusError, false},

		// From in_progress
		{"in_progress -> for_review", StatusInProgress, StatusForReview, true},
		{"in_progress -> needs_input", StatusInProgress, StatusNeedsInput, true},
		{"in_progress -> stopped", StatusInProgress, StatusStopped, true},
		{"in_progress -> error", StatusInProgress, StatusError, true},
		{"in_progress -> closed", StatusInProgress, StatusClosed, true},
		{"in_progress -> reviewed", StatusInProgress, StatusReviewed, false},
		{"in_progress -> todo", StatusInProgress, StatusTodo, false},

		// From needs_input
		{"needs_input -> in_progress", StatusNeedsInput, StatusInProgress, true},
		{"needs_input -> for_review", StatusNeedsInput, StatusForReview, true},
		{"needs_input -> closed", StatusNeedsInput, StatusClosed, true},
		{"needs_input -> todo", StatusNeedsInput, StatusTodo, false},
		{"needs_input -> reviewed", StatusNeedsInput, StatusReviewed, false},

		// From for_review
		{"for_review -> reviewing", StatusForReview, StatusReviewing, true},
		{"for_review -> in_progress", StatusForReview, StatusInProgress, true},
		{"for_review -> closed", StatusForReview, StatusClosed, true},
		{"for_review -> todo", StatusForReview, StatusTodo, false},
		{"for_review -> reviewed", StatusForReview, StatusReviewed, false},

		// From reviewing
		{"reviewing -> reviewed", StatusReviewing, StatusReviewed, true},
		{"reviewing -> in_progress", StatusReviewing, StatusInProgress, true},
		{"reviewing -> closed", StatusReviewing, StatusClosed, true},
		{"reviewing -> for_review", StatusReviewing, StatusForReview, false},
		{"reviewing -> todo", StatusReviewing, StatusTodo, false},

		// From reviewed
		{"reviewed -> in_progress", StatusReviewed, StatusInProgress, true},
		{"reviewed -> closed", StatusReviewed, StatusClosed, true},
		{"reviewed -> for_review", StatusReviewed, StatusForReview, false},
		{"reviewed -> todo", StatusReviewed, StatusTodo, false},

		// From stopped
		{"stopped -> in_progress", StatusStopped, StatusInProgress, true},
		{"stopped -> closed", StatusStopped, StatusClosed, true},
		{"stopped -> todo", StatusStopped, StatusTodo, false},
		{"stopped -> reviewed", StatusStopped, StatusReviewed, false},

		// From error
		{"error -> in_progress", StatusError, StatusInProgress, true},
		{"error -> closed", StatusError, StatusClosed, true},
		{"error -> todo", StatusError, StatusTodo, false},
		{"error -> for_review", StatusError, StatusForReview, false},
		{"error -> reviewed", StatusError, StatusReviewed, false},

		// From closed (terminal)
		{"closed -> todo", StatusClosed, StatusTodo, false},
		{"closed -> in_progress", StatusClosed, StatusInProgress, false},
		{"closed -> for_review", StatusClosed, StatusForReview, false},
		{"closed -> error", StatusClosed, StatusError, false},
		{"closed -> closed", StatusClosed, StatusClosed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.from.CanTransitionTo(tt.to)
			if got != tt.expect {
				t.Errorf("CanTransitionTo(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.expect)
			}
		})
	}
}

func TestStatus_CanTransitionTo_UnknownStatus(t *testing.T) {
	unknown := Status("unknown")
	if unknown.CanTransitionTo(StatusTodo) {
		t.Error("unknown status should not transition to any status")
	}
}

func TestStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status   Status
		terminal bool
	}{
		{StatusTodo, false},
		{StatusInProgress, false},
		{StatusNeedsInput, false},
		{StatusForReview, false},
		{StatusReviewing, false},
		{StatusReviewed, false},
		{StatusStopped, false},
		{StatusError, false},
		{StatusClosed, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsTerminal(); got != tt.terminal {
				t.Errorf("IsTerminal() = %v, want %v", got, tt.terminal)
			}
		})
	}
}

func TestStatus_CanStart(t *testing.T) {
	tests := []struct {
		status   Status
		canStart bool
	}{
		{StatusTodo, true},
		{StatusInProgress, false},
		{StatusNeedsInput, false},
		{StatusForReview, true},
		{StatusReviewing, false},
		{StatusReviewed, true},
		{StatusStopped, true},
		{StatusError, true},
		{StatusClosed, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.CanStart(); got != tt.canStart {
				t.Errorf("CanStart() = %v, want %v", got, tt.canStart)
			}
		})
	}
}

func TestStatus_Display(t *testing.T) {
	tests := []struct {
		status  Status
		display string
	}{
		{StatusTodo, "To Do"},
		{StatusInProgress, "In Progress"},
		{StatusNeedsInput, "Needs Input"},
		{StatusForReview, "For Review"},
		{StatusReviewing, "Reviewing"},
		{StatusReviewed, "Reviewed"},
		{StatusStopped, "Stopped"},
		{StatusError, "Error"},
		{StatusClosed, "Closed"},
		{Status("unknown"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.Display(); got != tt.display {
				t.Errorf("Display() = %v, want %v", got, tt.display)
			}
		})
	}
}

func TestStatus_IsValid(t *testing.T) {
	tests := []struct {
		status Status
		valid  bool
	}{
		{StatusTodo, true},
		{StatusInProgress, true},
		{StatusNeedsInput, true},
		{StatusForReview, true},
		{StatusReviewing, true},
		{StatusReviewed, true},
		{StatusStopped, true},
		{StatusError, true},
		{StatusClosed, true},
		{Status("unknown"), false},
		{Status(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.valid {
				t.Errorf("IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestAllStatuses(t *testing.T) {
	statuses := AllStatuses()
	expected := []Status{
		StatusTodo,
		StatusInProgress,
		StatusNeedsInput,
		StatusForReview,
		StatusReviewing,
		StatusReviewed,
		StatusStopped,
		StatusError,
		StatusClosed,
	}

	if len(statuses) != len(expected) {
		t.Errorf("AllStatuses() returned %d statuses, want %d", len(statuses), len(expected))
	}

	for i, s := range expected {
		if statuses[i] != s {
			t.Errorf("AllStatuses()[%d] = %v, want %v", i, statuses[i], s)
		}
	}
}
