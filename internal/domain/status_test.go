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
		{"todo -> error", StatusTodo, StatusError, false},
		{"todo -> merged", StatusTodo, StatusMerged, false},

		// From in_progress
		{"in_progress -> done", StatusInProgress, StatusDone, true},
		{"in_progress -> error", StatusInProgress, StatusError, true},
		{"in_progress -> closed", StatusInProgress, StatusClosed, true},
		{"in_progress -> merged", StatusInProgress, StatusMerged, false},
		{"in_progress -> todo", StatusInProgress, StatusTodo, false},

		// From done
		{"done -> merged", StatusDone, StatusMerged, true},
		{"done -> closed", StatusDone, StatusClosed, true},
		{"done -> in_progress", StatusDone, StatusInProgress, true},
		{"done -> todo", StatusDone, StatusTodo, false},
		{"done -> error", StatusDone, StatusError, false},

		// From error
		{"error -> in_progress", StatusError, StatusInProgress, true},
		{"error -> closed", StatusError, StatusClosed, true},
		{"error -> todo", StatusError, StatusTodo, false},
		{"error -> done", StatusError, StatusDone, false},
		{"error -> merged", StatusError, StatusMerged, false},

		// From merged (terminal)
		{"merged -> todo", StatusMerged, StatusTodo, false},
		{"merged -> in_progress", StatusMerged, StatusInProgress, false},
		{"merged -> done", StatusMerged, StatusDone, false},
		{"merged -> error", StatusMerged, StatusError, false},
		{"merged -> closed", StatusMerged, StatusClosed, false},

		// From closed (terminal)
		{"closed -> todo", StatusClosed, StatusTodo, false},
		{"closed -> in_progress", StatusClosed, StatusInProgress, false},
		{"closed -> done", StatusClosed, StatusDone, false},
		{"closed -> error", StatusClosed, StatusError, false},
		{"closed -> merged", StatusClosed, StatusMerged, false},
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
		{StatusDone, false},
		{StatusError, false},
		{StatusMerged, true},
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
		{StatusInProgress, true},
		{StatusDone, true},
		{StatusError, true},
		{StatusMerged, false},
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
		{StatusDone, "Done"},
		{StatusMerged, "Merged"},
		{StatusClosed, "Closed"},
		{StatusError, "Error"},
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
		{StatusDone, true},
		{StatusMerged, true},
		{StatusClosed, true},
		{StatusError, true},
		{Status("unknown"), false},
		{Status(""), false},
		// Legacy statuses are not valid for new tasks
		{Status("needs_input"), false},
		{Status("for_review"), false},
		{Status("reviewing"), false},
		{Status("reviewed"), false},
		{Status("stopped"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.valid {
				t.Errorf("IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestStatus_IsLegacy(t *testing.T) {
	tests := []struct {
		status   Status
		isLegacy bool
	}{
		{StatusTodo, false},
		{StatusInProgress, false},
		{StatusDone, false},
		{StatusMerged, false},
		{StatusClosed, false},
		{StatusError, false},
		{Status("needs_input"), true},
		{Status("for_review"), true},
		{Status("reviewing"), true},
		{Status("reviewed"), true},
		{Status("stopped"), true},
		{Status("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsLegacy(); got != tt.isLegacy {
				t.Errorf("IsLegacy() = %v, want %v", got, tt.isLegacy)
			}
		})
	}
}

func TestAllStatuses(t *testing.T) {
	statuses := AllStatuses()
	expected := []Status{
		StatusTodo,
		StatusInProgress,
		StatusDone,
		StatusMerged,
		StatusClosed,
		StatusError,
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

func TestNormalizeStatus(t *testing.T) {
	tests := []struct {
		name          string
		inputStatus   Status
		inputVersion  int
		inputReason   CloseReason
		expectStatus  Status
		expectReason  CloseReason
		expectVersion int
	}{
		// Already normalized (StatusVersion >= 2) - no change
		{
			name:          "already normalized todo",
			inputStatus:   StatusTodo,
			inputVersion:  2,
			expectStatus:  StatusTodo,
			expectVersion: 2,
		},
		{
			name:          "already normalized done",
			inputStatus:   StatusDone,
			inputVersion:  2,
			expectStatus:  StatusDone,
			expectVersion: 2,
		},

		// Legacy statuses (StatusVersion = 0) - normalize
		{
			name:          "legacy needs_input -> in_progress",
			inputStatus:   Status("needs_input"),
			inputVersion:  0,
			expectStatus:  StatusInProgress,
			expectVersion: StatusVersionCurrent,
		},
		{
			name:          "legacy stopped -> in_progress",
			inputStatus:   Status("stopped"),
			inputVersion:  0,
			expectStatus:  StatusInProgress,
			expectVersion: StatusVersionCurrent,
		},
		{
			name:          "legacy for_review -> in_progress",
			inputStatus:   Status("for_review"),
			inputVersion:  0,
			expectStatus:  StatusInProgress,
			expectVersion: StatusVersionCurrent,
		},
		{
			name:          "legacy reviewing -> in_progress",
			inputStatus:   Status("reviewing"),
			inputVersion:  0,
			expectStatus:  StatusInProgress,
			expectVersion: StatusVersionCurrent,
		},
		{
			name:          "legacy reviewed -> done",
			inputStatus:   Status("reviewed"),
			inputVersion:  0,
			expectStatus:  StatusDone,
			expectVersion: StatusVersionCurrent,
		},

		// Legacy done (StatusVersion = 0) -> closed
		{
			name:          "legacy done -> closed",
			inputStatus:   Status("done"),
			inputVersion:  0,
			expectStatus:  StatusClosed,
			expectReason:  CloseReasonAbandoned,
			expectVersion: StatusVersionCurrent,
		},
		{
			name:          "legacy done with existing reason -> closed",
			inputStatus:   Status("done"),
			inputVersion:  0,
			inputReason:   CloseReasonMerged,
			expectStatus:  StatusClosed,
			expectReason:  CloseReasonMerged, // Preserve existing reason
			expectVersion: StatusVersionCurrent,
		},

		// closed + CloseReasonMerged -> merged
		{
			name:          "closed with merged reason -> merged",
			inputStatus:   StatusClosed,
			inputVersion:  0,
			inputReason:   CloseReasonMerged,
			expectStatus:  StatusMerged,
			expectReason:  CloseReasonNone, // Reason is cleared after split
			expectVersion: StatusVersionCurrent,
		},

		// closed without merged reason -> closed
		{
			name:          "closed with abandoned reason -> closed",
			inputStatus:   StatusClosed,
			inputVersion:  0,
			inputReason:   CloseReasonAbandoned,
			expectStatus:  StatusClosed,
			expectReason:  CloseReasonAbandoned,
			expectVersion: StatusVersionCurrent,
		},
		{
			name:          "closed with no reason -> closed",
			inputStatus:   StatusClosed,
			inputVersion:  0,
			inputReason:   CloseReasonNone,
			expectStatus:  StatusClosed,
			expectReason:  CloseReasonNone,
			expectVersion: StatusVersionCurrent,
		},

		// Non-legacy statuses remain unchanged
		{
			name:          "todo unchanged",
			inputStatus:   StatusTodo,
			inputVersion:  0,
			expectStatus:  StatusTodo,
			expectVersion: StatusVersionCurrent,
		},
		{
			name:          "in_progress unchanged",
			inputStatus:   StatusInProgress,
			inputVersion:  0,
			expectStatus:  StatusInProgress,
			expectVersion: StatusVersionCurrent,
		},
		{
			name:          "error unchanged",
			inputStatus:   StatusError,
			inputVersion:  0,
			expectStatus:  StatusError,
			expectVersion: StatusVersionCurrent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &Task{
				Status:        tt.inputStatus,
				StatusVersion: tt.inputVersion,
				CloseReason:   tt.inputReason,
			}

			NormalizeStatus(task)

			if task.Status != tt.expectStatus {
				t.Errorf("Status = %v, want %v", task.Status, tt.expectStatus)
			}
			if task.StatusVersion != tt.expectVersion {
				t.Errorf("StatusVersion = %v, want %v", task.StatusVersion, tt.expectVersion)
			}
			if task.CloseReason != tt.expectReason {
				t.Errorf("CloseReason = %v, want %v", task.CloseReason, tt.expectReason)
			}
		})
	}
}

func TestNormalizeStatus_Idempotent(t *testing.T) {
	// Ensure normalizing twice produces the same result
	task := &Task{
		Status:        Status("needs_input"),
		StatusVersion: 0,
	}

	NormalizeStatus(task)
	firstStatus := task.Status
	firstVersion := task.StatusVersion

	NormalizeStatus(task)
	if task.Status != firstStatus {
		t.Errorf("Second normalization changed status: %v -> %v", firstStatus, task.Status)
	}
	if task.StatusVersion != firstVersion {
		t.Errorf("Second normalization changed version: %v -> %v", firstVersion, task.StatusVersion)
	}
}
