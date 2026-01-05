package domain

// Status represents the lifecycle state of a task.
type Status string

const (
	StatusTodo         Status = "todo"          // Created, awaiting start
	StatusInProgress   Status = "in_progress"   // Agent working
	StatusInReview     Status = "in_review"     // Work complete, awaiting review
	StatusNeedsInput   Status = "needs_input"   // Agent is waiting for user input
	StatusNeedsChanges Status = "needs_changes" // Reviewer requested changes
	StatusError        Status = "error"         // Session terminated abnormally
	StatusDone         Status = "done"          // Merge complete
	StatusClosed       Status = "closed"        // Discarded without merge
	StatusStopped      Status = "stopped"       // Manually stopped
)

// AllStatuses returns all valid status values.
func AllStatuses() []Status {
	return []Status{
		StatusTodo,
		StatusInProgress,
		StatusInReview,
		StatusNeedsInput,
		StatusNeedsChanges,
		StatusStopped,
		StatusError,
		StatusDone,
		StatusClosed,
	}
}

// transitions defines the allowed status transitions.
var transitions = map[Status][]Status{
	StatusTodo:         {StatusInProgress, StatusClosed},
	StatusInProgress:   {StatusInReview, StatusNeedsInput, StatusNeedsChanges, StatusStopped, StatusError, StatusClosed},
	StatusInReview:     {StatusInProgress, StatusDone, StatusClosed},
	StatusNeedsInput:   {StatusInProgress, StatusClosed},
	StatusNeedsChanges: {StatusInProgress, StatusClosed},
	StatusStopped:      {StatusInProgress, StatusClosed},
	StatusError:        {StatusInProgress, StatusClosed},
	StatusDone:         {StatusClosed},
	StatusClosed:       {},
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
	return s == StatusClosed
}

// CanStart returns true if a task in this status can be started.
func (s Status) CanStart() bool {
	return s == StatusTodo || s == StatusInReview || s == StatusStopped || s == StatusError
}

// Display returns a human-readable representation of the status.
func (s Status) Display() string {
	switch s {
	case StatusTodo:
		return "To Do"
	case StatusInProgress:
		return "In Progress"
	case StatusInReview:
		return "In Review"
	case StatusNeedsInput:
		return "Needs Input"
	case StatusNeedsChanges:
		return "Needs Changes"
	case StatusStopped:
		return "Stopped"
	case StatusError:
		return "Error"
	case StatusDone:
		return "Done"
	case StatusClosed:
		return "Closed"
	default:
		return string(s)
	}
}

// IsValid returns true if the status is a known valid value.
func (s Status) IsValid() bool {
	switch s {
	case StatusTodo, StatusInProgress, StatusInReview, StatusNeedsInput, StatusNeedsChanges, StatusStopped, StatusError, StatusDone, StatusClosed:
		return true
	default:
		return false
	}
}
