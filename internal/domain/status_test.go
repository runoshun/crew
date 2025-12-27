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
		{"todo -> done", StatusTodo, StatusDone, false},
		{"todo -> in_review", StatusTodo, StatusInReview, false},
		{"todo -> error", StatusTodo, StatusError, false},

		// From in_progress
		{"in_progress -> in_review", StatusInProgress, StatusInReview, true},
		{"in_progress -> error", StatusInProgress, StatusError, true},
		{"in_progress -> closed", StatusInProgress, StatusClosed, true},
		{"in_progress -> done", StatusInProgress, StatusDone, false},
		{"in_progress -> todo", StatusInProgress, StatusTodo, false},

		// From in_review
		{"in_review -> in_progress", StatusInReview, StatusInProgress, true},
		{"in_review -> done", StatusInReview, StatusDone, true},
		{"in_review -> closed", StatusInReview, StatusClosed, true},
		{"in_review -> error", StatusInReview, StatusError, false},
		{"in_review -> todo", StatusInReview, StatusTodo, false},

		// From error
		{"error -> in_progress", StatusError, StatusInProgress, true},
		{"error -> closed", StatusError, StatusClosed, true},
		{"error -> todo", StatusError, StatusTodo, false},
		{"error -> in_review", StatusError, StatusInReview, false},
		{"error -> done", StatusError, StatusDone, false},

		// From done
		{"done -> closed", StatusDone, StatusClosed, true},
		{"done -> todo", StatusDone, StatusTodo, false},
		{"done -> in_progress", StatusDone, StatusInProgress, false},
		{"done -> in_review", StatusDone, StatusInReview, false},
		{"done -> error", StatusDone, StatusError, false},

		// From closed (terminal)
		{"closed -> todo", StatusClosed, StatusTodo, false},
		{"closed -> in_progress", StatusClosed, StatusInProgress, false},
		{"closed -> in_review", StatusClosed, StatusInReview, false},
		{"closed -> error", StatusClosed, StatusError, false},
		{"closed -> done", StatusClosed, StatusDone, false},
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
		{StatusInReview, false},
		{StatusError, false},
		{StatusDone, false},
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
		{StatusInReview, true},
		{StatusError, true},
		{StatusDone, false},
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
		{StatusInReview, "In Review"},
		{StatusError, "Error"},
		{StatusDone, "Done"},
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
		{StatusInReview, true},
		{StatusError, true},
		{StatusDone, true},
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
		StatusInReview,
		StatusError,
		StatusDone,
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
