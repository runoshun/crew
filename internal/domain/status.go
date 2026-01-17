package domain

// Status represents the lifecycle state of a task.
type Status string

const (
	StatusTodo       Status = "todo"        // Created, awaiting start
	StatusInProgress Status = "in_progress" // Agent working
	StatusNeedsInput Status = "needs_input" // Agent is waiting for user input
	StatusForReview  Status = "for_review"  // Work complete, awaiting review
	StatusReviewing  Status = "reviewing"   // Review in progress
	StatusReviewed   Status = "reviewed"    // Review complete, results available
	StatusStopped    Status = "stopped"     // Manually stopped
	StatusError      Status = "error"       // Session terminated abnormally
	StatusClosed     Status = "closed"      // Closed (CloseReason specifies why)

	// Legacy status (for backward compatibility with old data)
	statusDoneLegacy Status = "done" // Legacy: renamed to closed
)

// AllStatuses returns all valid status values.
func AllStatuses() []Status {
	return []Status{
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
}

// transitions defines the allowed status transitions.
// Flow: in_progress → for_review → reviewing → reviewed → closed
//
//	↑               ↑                         ↓
//	└───────────────┴──────── (on changes) ───┘
var transitions = map[Status][]Status{
	StatusTodo:       {StatusInProgress, StatusClosed},
	StatusInProgress: {StatusForReview, StatusNeedsInput, StatusStopped, StatusError, StatusClosed},
	StatusNeedsInput: {StatusInProgress, StatusForReview, StatusClosed},
	StatusForReview:  {StatusReviewing, StatusInProgress, StatusClosed},
	StatusReviewing:  {StatusReviewed, StatusInProgress, StatusClosed},
	StatusReviewed:   {StatusInProgress, StatusClosed},
	StatusStopped:    {StatusInProgress, StatusClosed},
	StatusError:      {StatusInProgress, StatusClosed},
	StatusClosed:     {},
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
	return s == StatusClosed || s == statusDoneLegacy
}

// CanStart returns true if a task in this status can be started.
func (s Status) CanStart() bool {
	return s == StatusTodo || s == StatusForReview || s == StatusReviewed || s == StatusStopped || s == StatusError
}

// Display returns a human-readable representation of the status.
func (s Status) Display() string {
	switch s {
	case StatusTodo:
		return "To Do"
	case StatusInProgress:
		return "In Progress"
	case StatusNeedsInput:
		return "Needs Input"
	case StatusForReview:
		return "For Review"
	case StatusReviewing:
		return "Reviewing"
	case StatusReviewed:
		return "Reviewed"
	case StatusStopped:
		return "Stopped"
	case StatusError:
		return "Error"
	case StatusClosed, statusDoneLegacy:
		return "Closed"
	default:
		return string(s)
	}
}

// IsValid returns true if the status is a known valid value.
// Note: Legacy status "done" is not considered valid for new tasks.
func (s Status) IsValid() bool {
	switch s {
	case StatusTodo, StatusInProgress, StatusNeedsInput, StatusForReview, StatusReviewing, StatusReviewed, StatusStopped, StatusError, StatusClosed:
		return true
	case statusDoneLegacy:
		return false // Legacy status is not valid for new tasks
	default:
		return false
	}
}

// IsLegacyDone returns true if this is the legacy "done" status.
// Used for backward compatibility when displaying old tasks.
func (s Status) IsLegacyDone() bool {
	return s == statusDoneLegacy
}
