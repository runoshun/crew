package usecase

import (
	"context"
	"os"
	"strings"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// ReviewSessionEndedInput contains the parameters for handling review session termination.
type ReviewSessionEndedInput struct {
	Output   string // Output from the reviewer (to save as comment)
	TaskID   int    // Task ID
	ExitCode int    // Exit code of the reviewer process
}

// ReviewSessionEndedOutput contains the result of handling review session termination.
type ReviewSessionEndedOutput struct {
	Ignored bool // True if the callback was ignored (already cleaned up)
}

// ReviewSessionEnded is the use case for handling review session termination.
// This is called by the review script's EXIT trap.
type ReviewSessionEnded struct {
	tasks   domain.TaskRepository
	clock   domain.Clock
	crewDir string
}

// NewReviewSessionEnded creates a new ReviewSessionEnded use case.
func NewReviewSessionEnded(tasks domain.TaskRepository, clock domain.Clock, crewDir string) *ReviewSessionEnded {
	return &ReviewSessionEnded{
		tasks:   tasks,
		clock:   clock,
		crewDir: crewDir,
	}
}

// Execute handles review session termination.
// It saves the review output as a comment, updates status, and deletes script files.
func (uc *ReviewSessionEnded) Execute(_ context.Context, in ReviewSessionEndedInput) (*ReviewSessionEndedOutput, error) {
	// Get task
	task, err := uc.tasks.Get(in.TaskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		// Task doesn't exist, nothing to do
		return &ReviewSessionEndedOutput{Ignored: true}, nil
	}

	// Check if task is in reviewing status (expected state)
	if task.Status != domain.StatusReviewing {
		// Already processed or unexpected state
		return &ReviewSessionEndedOutput{Ignored: true}, nil
	}

	// Extract the review result
	reviewResult := in.Output
	if idx := strings.Index(in.Output, domain.ReviewResultMarker); idx != -1 {
		reviewResult = in.Output[idx+len(domain.ReviewResultMarker):]
	}
	reviewResult = strings.TrimSpace(reviewResult)

	// Save review result as comment if there's content
	if reviewResult != "" {
		comment := domain.Comment{
			Text:   reviewResult,
			Time:   uc.clock.Now(),
			Author: "reviewer",
		}
		if err := uc.tasks.AddComment(task.ID, comment); err != nil {
			// Log but don't fail
			_ = err
		}
	}

	// Update status based on exit code
	if in.ExitCode == 0 {
		task.Status = domain.StatusReviewed
	} else {
		// Keep as reviewing or set error? Let's keep as for_review to allow retry
		task.Status = domain.StatusForReview
	}

	// Save task
	if err := uc.tasks.Save(task); err != nil {
		return nil, err
	}

	// Cleanup script files (ignore errors)
	uc.cleanupScriptFiles(in.TaskID)

	return &ReviewSessionEndedOutput{Ignored: false}, nil
}

// cleanupScriptFiles removes the generated review script files.
func (uc *ReviewSessionEnded) cleanupScriptFiles(taskID int) {
	scriptPath := domain.ReviewScriptPath(uc.crewDir, taskID)
	_ = os.Remove(scriptPath)
	promptPath := domain.ReviewPromptPath(uc.crewDir, taskID)
	_ = os.Remove(promptPath)
}
