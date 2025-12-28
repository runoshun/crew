// Package usecase contains application use cases.
package usecase

import (
	"context"
	"fmt"
	"slices"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// EditTaskInput contains the parameters for editing a task.
// All fields except TaskID are optional. Only non-nil/non-empty fields will be updated.
type EditTaskInput struct {
	Title        *string  // New title (nil = no change)
	Description  *string  // New description (nil = no change)
	AddLabels    []string // Labels to add
	RemoveLabels []string // Labels to remove
	TaskID       int      // Task ID to edit (required)
}

// EditTaskOutput contains the result of editing a task.
type EditTaskOutput struct {
	Task *domain.Task // The updated task
}

// EditTask is the use case for editing an existing task.
type EditTask struct {
	tasks domain.TaskRepository
}

// NewEditTask creates a new EditTask use case.
func NewEditTask(tasks domain.TaskRepository) *EditTask {
	return &EditTask{
		tasks: tasks,
	}
}

// Execute edits a task with the given input.
func (uc *EditTask) Execute(_ context.Context, in EditTaskInput) (*EditTaskOutput, error) {
	// Validate that at least one field is being updated
	if in.Title == nil && in.Description == nil && len(in.AddLabels) == 0 && len(in.RemoveLabels) == 0 {
		return nil, domain.ErrNoFieldsToUpdate
	}

	// Validate title is not empty if provided
	if in.Title != nil && *in.Title == "" {
		return nil, domain.ErrEmptyTitle
	}

	// Get existing task
	task, err := uc.tasks.Get(in.TaskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}

	// Update fields
	if in.Title != nil {
		task.Title = *in.Title
	}
	if in.Description != nil {
		task.Description = *in.Description
	}

	// Handle labels
	if len(in.AddLabels) > 0 || len(in.RemoveLabels) > 0 {
		task.Labels = updateLabels(task.Labels, in.AddLabels, in.RemoveLabels)
	}

	// Save updated task
	if err := uc.tasks.Save(task); err != nil {
		return nil, fmt.Errorf("save task: %w", err)
	}

	return &EditTaskOutput{Task: task}, nil
}

// updateLabels adds and removes labels from the current set.
// Returns a new slice with duplicates removed.
func updateLabels(current, add, remove []string) []string {
	// Create a set of labels to remove
	removeSet := make(map[string]bool, len(remove))
	for _, label := range remove {
		removeSet[label] = true
	}

	// Start with current labels (excluding ones to remove)
	labelSet := make(map[string]bool)
	for _, label := range current {
		if !removeSet[label] {
			labelSet[label] = true
		}
	}

	// Add new labels
	for _, label := range add {
		if !removeSet[label] { // Don't add if it's also being removed
			labelSet[label] = true
		}
	}

	// Convert back to slice
	if len(labelSet) == 0 {
		return nil
	}

	result := make([]string, 0, len(labelSet))
	for label := range labelSet {
		result = append(result, label)
	}

	slices.Sort(result)
	return result
}
