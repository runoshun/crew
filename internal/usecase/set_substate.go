// Package usecase contains application use cases.
package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase/shared"
)

// SetSubstateInput contains parameters for updating ACP execution substate.
type SetSubstateInput struct {
	Substate domain.ACPExecutionSubstate
	TaskID   int
}

// SetSubstateOutput contains the result of SetSubstate.
type SetSubstateOutput struct{}

// SetSubstate updates ACP execution substate for a task.
type SetSubstate struct {
	tasks     domain.TaskRepository
	acpStates domain.ACPStateStore
}

// NewSetSubstate creates a new SetSubstate use case.
func NewSetSubstate(tasks domain.TaskRepository, acpStates domain.ACPStateStore) *SetSubstate {
	return &SetSubstate{
		tasks:     tasks,
		acpStates: acpStates,
	}
}

// Execute updates the execution substate for a task.
func (uc *SetSubstate) Execute(ctx context.Context, in SetSubstateInput) (*SetSubstateOutput, error) {
	if in.TaskID <= 0 {
		return nil, fmt.Errorf("invalid task ID: %d", in.TaskID)
	}
	if !in.Substate.IsValid() {
		return nil, domain.ErrInvalidACPExecutionSubstate
	}
	if uc.acpStates == nil {
		return nil, fmt.Errorf("acp state store is not configured")
	}

	task, err := shared.GetTask(uc.tasks, in.TaskID)
	if err != nil {
		return nil, err
	}

	namespace := task.Namespace
	if namespace == "" {
		namespace = "default"
	}

	state, err := uc.acpStates.Load(ctx, namespace, in.TaskID)
	if err != nil && !errors.Is(err, domain.ErrACPStateNotFound) {
		return nil, fmt.Errorf("load acp state: %w", err)
	}

	state.ExecutionSubstate = in.Substate
	if err := uc.acpStates.Save(ctx, namespace, in.TaskID, state); err != nil {
		return nil, fmt.Errorf("save acp state: %w", err)
	}

	return &SetSubstateOutput{}, nil
}
