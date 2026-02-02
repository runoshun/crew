// Package usecase contains application use cases.
package usecase

import (
	"context"
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase/shared"
)

// SetSubstateInput contains parameters for updating execution substate.
type SetSubstateInput struct {
	Substate domain.ExecutionSubstate
	TaskID   int
}

// SetSubstateOutput contains the result of SetSubstate.
type SetSubstateOutput struct{}

// SetSubstate updates execution substate for a task.
type SetSubstate struct {
	tasks domain.TaskRepository
}

// NewSetSubstate creates a new SetSubstate use case.
func NewSetSubstate(tasks domain.TaskRepository) *SetSubstate {
	return &SetSubstate{tasks: tasks}
}

// Execute updates the execution substate for a task.
func (uc *SetSubstate) Execute(_ context.Context, in SetSubstateInput) (*SetSubstateOutput, error) {
	if in.TaskID <= 0 {
		return nil, fmt.Errorf("invalid task ID: %d", in.TaskID)
	}
	if !in.Substate.IsValid() {
		return nil, domain.ErrInvalidExecutionSubstate
	}

	task, err := shared.GetTask(uc.tasks, in.TaskID)
	if err != nil {
		return nil, err
	}

	task.ExecutionSubstate = in.Substate
	if err := uc.tasks.Save(task); err != nil {
		return nil, fmt.Errorf("save task: %w", err)
	}

	return &SetSubstateOutput{}, nil
}
