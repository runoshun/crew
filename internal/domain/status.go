package domain

// Status represents the lifecycle state of a task.
type Status string

const (
	StatusTodo       Status = "todo"        // Created, awaiting start
	StatusInProgress Status = "in_progress" // Working (includes input waiting, paused states)
	StatusDone       Status = "done"        // Complete, awaiting merge/close
	StatusMerged     Status = "merged"      // Merged (terminal)
	StatusClosed     Status = "closed"      // Closed without merge (terminal)
	StatusError      Status = "error"       // Session abnormally terminated or manually stopped (restartable)

	// Legacy statuses (for backward compatibility with old data)
	// These are mapped to new statuses during normalization
	statusNeedsInputLegacy Status = "needs_input" // -> in_progress
	statusForReviewLegacy  Status = "for_review"  // -> in_progress
	statusReviewingLegacy  Status = "reviewing"   // -> in_progress
	statusReviewedLegacy   Status = "reviewed"    // -> done
	statusStoppedLegacy    Status = "stopped"     // -> in_progress
)

// AllStatuses returns all valid status values.
func AllStatuses() []Status {
	return []Status{
		StatusTodo,
		StatusInProgress,
		StatusDone,
		StatusMerged,
		StatusClosed,
		StatusError,
	}
}

// transitions defines the allowed status transitions.
//
// Main flow:
//
//	todo → in_progress → done → merged
//	                 ↓       ↓
//	              error   closed
//
// Error recovery: error → in_progress
// Rework: done → in_progress
var transitions = map[Status][]Status{
	StatusTodo:       {StatusInProgress, StatusClosed},
	StatusInProgress: {StatusDone, StatusError, StatusClosed},
	StatusDone:       {StatusMerged, StatusClosed, StatusInProgress},
	StatusError:      {StatusInProgress, StatusClosed},
	StatusMerged:     {}, // Terminal
	StatusClosed:     {}, // Terminal
}

// CanTransitionTo returns true if the status can transition to the target status.
func (s Status) CanTransitionTo(target Status) bool {
	allowed, ok := transitions[s]
	if !ok {
		return false
	}
	for _, t := range allowed {
		if t == target {
			return true
		}
	}
	return false
}

// IsTerminal returns true if the status is a terminal state.
func (s Status) IsTerminal() bool {
	return s == StatusMerged || s == StatusClosed
}

// CanStart returns true if a task in this status can be started.
// Startable: todo, in_progress, done, error
// Non-startable: merged, closed (terminal states)
func (s Status) CanStart() bool {
	return s == StatusTodo || s == StatusInProgress || s == StatusDone || s == StatusError
}

// Display returns a human-readable representation of the status.
func (s Status) Display() string {
	switch s {
	case StatusTodo:
		return "To Do"
	case StatusInProgress, statusNeedsInputLegacy, statusForReviewLegacy, statusReviewingLegacy, statusStoppedLegacy:
		return "In Progress"
	case StatusDone, statusReviewedLegacy:
		return "Done"
	case StatusMerged:
		return "Merged"
	case StatusClosed:
		return "Closed"
	case StatusError:
		return "Error"
	default:
		return string(s)
	}
}

// IsValid returns true if the status is a known valid value.
// Legacy statuses are NOT valid (they should be normalized).
func (s Status) IsValid() bool {
	switch s {
	case StatusTodo, StatusInProgress, StatusDone, StatusMerged, StatusClosed, StatusError:
		return true
	case statusNeedsInputLegacy, statusForReviewLegacy, statusReviewingLegacy, statusReviewedLegacy, statusStoppedLegacy:
		return false
	default:
		return false
	}
}

// IsLegacy returns true if this is a legacy status that needs normalization.
func (s Status) IsLegacy() bool {
	switch s {
	case statusNeedsInputLegacy, statusForReviewLegacy, statusReviewingLegacy, statusReviewedLegacy, statusStoppedLegacy:
		return true
	case StatusTodo, StatusInProgress, StatusDone, StatusMerged, StatusClosed, StatusError:
		return false
	default:
		return false
	}
}

// NormalizeStatus normalizes legacy status values in a task to the new status model.
// It updates the task in-place and sets StatusVersion to the current version.
//
// Legacy status mapping:
//   - todo -> todo
//   - in_progress -> in_progress
//   - needs_input -> in_progress
//   - stopped -> in_progress
//   - for_review/reviewing -> in_progress
//   - reviewed -> done
//   - closed + closeReason=merged -> merged
//   - closed + closeReason=abandoned/empty -> closed
//   - error -> error
//   - done (legacy, StatusVersion=0) -> closed
//   - done (new, StatusVersion>=2) -> done (no change)
func NormalizeStatus(task *Task) {
	// Already normalized
	if task.StatusVersion >= StatusVersionCurrent {
		return
	}

	// Map legacy statuses to new statuses
	switch task.Status {
	case statusNeedsInputLegacy, statusStoppedLegacy, statusForReviewLegacy, statusReviewingLegacy:
		task.Status = StatusInProgress
	case statusReviewedLegacy:
		task.Status = StatusDone
	case StatusClosed:
		// Split closed status based on CloseReason
		if task.CloseReason == CloseReasonMerged {
			task.Status = StatusMerged
			task.CloseReason = CloseReasonNone // No longer needed
		}
		// Otherwise keep as StatusClosed
	case StatusDone:
		// StatusDone = "done" has different meanings:
		// - Legacy (StatusVersion=0): meant "closed/finished"
		// - New (StatusVersion>=2): means "complete, awaiting merge/close"
		// Since we already checked StatusVersion at top, this is legacy "done"
		if task.StatusVersion == 0 {
			task.Status = StatusClosed
			if task.CloseReason == CloseReasonNone {
				task.CloseReason = CloseReasonAbandoned
			}
		}
		// StatusVersion=1 would keep as StatusDone (hypothetical intermediate version)
	case StatusTodo, StatusInProgress, StatusMerged, StatusError:
		// These statuses remain unchanged
	}

	// Mark as normalized
	task.StatusVersion = StatusVersionCurrent
}
