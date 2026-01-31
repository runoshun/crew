// Package usecase contains application use cases.
package usecase

import (
	"context"
	"fmt"
	"slices"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase/shared"
)

// EditTaskInput contains the parameters for editing a task.
// All fields except TaskID are optional. Only non-nil/non-empty fields will be updated.
type EditTaskInput struct {
	Title        *string         // New title (nil = no change)
	Description  *string         // New description (nil = no change)
	Status       *domain.Status  // New status (nil = no change)
	SkipReview   *bool           // New skip_review setting (nil = no change)
	ParentID     *int            // New parent ID (nil = no change, 0 = remove parent)
	BlockReason  *string         // New block reason (nil = no change, "" = unblock)
	EditorText   string          // Markdown text from editor (only used when EditorEdit is true)
	Labels       []string        // Labels to set (replaces all existing labels, nil = no change)
	AddLabels    []string        // Labels to add
	RemoveLabels []string        // Labels to remove
	IfStatus     []domain.Status // Conditional status update: only update if current status is in this list
	TaskID       int             // Task ID to edit (required)
	LabelsSet    bool            // True if Labels was explicitly set (to distinguish nil from empty)
	EditorEdit   bool            // True if editing via editor (title/description from markdown)
	RemoveParent bool            // True to remove parent (set ParentID to nil)
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
	// Handle editor mode
	if in.EditorEdit {
		return uc.executeEditorMode(in)
	}

	// Validate that at least one field is being updated
	if in.Title == nil && in.Description == nil && in.Status == nil && in.SkipReview == nil && in.ParentID == nil && in.BlockReason == nil && !in.RemoveParent && !in.LabelsSet && len(in.AddLabels) == 0 && len(in.RemoveLabels) == 0 {
		return nil, domain.ErrNoFieldsToUpdate
	}

	// Validate title is not empty if provided
	if in.Title != nil && *in.Title == "" {
		return nil, domain.ErrEmptyTitle
	}

	// Validate status is valid if provided
	if in.Status != nil && !in.Status.IsValid() {
		return nil, domain.ErrInvalidStatus
	}

	// Validate IfStatus values are valid if provided
	for _, status := range in.IfStatus {
		if !status.IsValid() {
			return nil, domain.ErrInvalidStatus
		}
	}

	// Get existing task
	task, err := shared.GetTask(uc.tasks, in.TaskID)
	if err != nil {
		return nil, err
	}

	// Update fields
	if in.Title != nil {
		task.Title = *in.Title
	}
	if in.Description != nil {
		task.Description = *in.Description
	}
	if in.Status != nil {
		// Check if conditional status update is requested
		if len(in.IfStatus) > 0 {
			// Only update status if current status matches one of the conditions
			matched := false
			for _, allowedStatus := range in.IfStatus {
				if task.Status == allowedStatus {
					matched = true
					break
				}
			}
			if matched {
				// Manual status change via edit bypasses transition rules
				task.Status = *in.Status
			}
			// If condition not met, skip status update but continue with other fields
		} else {
			// No condition specified, update status unconditionally
			// Manual status change via edit bypasses transition rules
			task.Status = *in.Status
		}
	}

	// Handle skip_review
	if in.SkipReview != nil {
		task.SkipReview = in.SkipReview
	}

	// Handle block reason
	if in.BlockReason != nil {
		task.BlockReason = *in.BlockReason
	}

	// Handle parent change
	if in.RemoveParent {
		// Remove parent (make this a root task)
		task.ParentID = nil
	} else if in.ParentID != nil {
		newParentID := *in.ParentID
		if newParentID == 0 {
			// --parent 0 means remove parent
			task.ParentID = nil
		} else if newParentID < 0 {
			// Negative IDs are invalid
			return nil, domain.ErrInvalidParentID
		} else {
			// Validate parent exists
			parent, err := uc.tasks.Get(newParentID)
			if err != nil {
				return nil, fmt.Errorf("get parent task: %w", err)
			}
			if parent == nil {
				return nil, domain.ErrParentNotFound
			}

			// Check for circular reference: ensure we're not setting a descendant as parent
			if err := uc.checkCircularReference(in.TaskID, newParentID); err != nil {
				return nil, err
			}

			task.ParentID = &newParentID
		}
	}

	// Handle labels
	if in.LabelsSet {
		// Replace all labels with the new set (deduplicated)
		if len(in.Labels) == 0 {
			task.Labels = nil
		} else {
			// Deduplicate using a set
			labelSet := make(map[string]bool, len(in.Labels))
			for _, label := range in.Labels {
				labelSet[label] = true
			}
			task.Labels = make([]string, 0, len(labelSet))
			for label := range labelSet {
				task.Labels = append(task.Labels, label)
			}
			slices.Sort(task.Labels)
		}
	} else if len(in.AddLabels) > 0 || len(in.RemoveLabels) > 0 {
		task.Labels = updateLabels(task.Labels, in.AddLabels, in.RemoveLabels)
	}

	// Save updated task
	if err := uc.tasks.Save(task); err != nil {
		return nil, fmt.Errorf("save task: %w", err)
	}

	return &EditTaskOutput{Task: task}, nil
}

// executeEditorMode handles editing a task via editor (markdown format).
func (uc *EditTask) executeEditorMode(in EditTaskInput) (*EditTaskOutput, error) {
	// Get existing task
	task, err := shared.GetTask(uc.tasks, in.TaskID)
	if err != nil {
		return nil, err
	}

	// Parse editor content (includes comments)
	content, err := domain.ParseEditorContent(in.EditorText)
	if err != nil {
		return nil, fmt.Errorf("failed to parse editor content: %w", err)
	}

	// Get original comments to validate and prepare updates
	originalComments, err := uc.tasks.GetComments(in.TaskID)
	if err != nil {
		return nil, fmt.Errorf("get comments: %w", err)
	}

	// Validate comment count matches (no add/delete allowed)
	if len(content.Comments) != len(originalComments) {
		return nil, fmt.Errorf("comment count mismatch: expected %d, got %d (adding/deleting comments is not allowed)",
			len(originalComments), len(content.Comments))
	}

	// Prepare updated comments list (copy of original, with edits applied)
	updatedComments := make([]domain.Comment, len(originalComments))
	copy(updatedComments, originalComments)

	// Validate all comment indices are unique and form a complete sequence 0..N-1
	if len(originalComments) > 0 {
		// Track which indices are present
		seenIndices := make(map[int]bool, len(content.Comments))

		for _, parsed := range content.Comments {
			// Validate index is within bounds
			if parsed.Index < 0 || parsed.Index >= len(originalComments) {
				return nil, fmt.Errorf("invalid comment index: %d", parsed.Index)
			}

			// Check for duplicate indices
			if seenIndices[parsed.Index] {
				return nil, fmt.Errorf("duplicate comment index: %d", parsed.Index)
			}
			seenIndices[parsed.Index] = true

			// Apply text update (preserve original author and time)
			original := originalComments[parsed.Index]
			updatedComments[parsed.Index] = domain.Comment{
				Text:   parsed.Text,
				Time:   original.Time,
				Author: original.Author,
			}
		}

		// Verify all indices are present (0 to N-1)
		for i := 0; i < len(originalComments); i++ {
			if !seenIndices[i] {
				return nil, fmt.Errorf("missing comment index: %d", i)
			}
		}
	}

	// Update task fields
	task.Title = content.Title
	task.Description = content.Description
	if content.LabelsFound {
		task.Labels = content.Labels
	}

	// Handle parent change from editor
	if content.ParentFound {
		// Check if parent actually changed
		parentChanged := false
		if content.ParentID == nil && task.ParentID != nil {
			parentChanged = true
		} else if content.ParentID != nil && task.ParentID == nil {
			parentChanged = true
		} else if content.ParentID != nil && task.ParentID != nil && *content.ParentID != *task.ParentID {
			parentChanged = true
		}

		if parentChanged {
			if content.ParentID == nil {
				// Remove parent
				task.ParentID = nil
			} else {
				newParentID := *content.ParentID
				// Validate parent exists
				parent, err := uc.tasks.Get(newParentID)
				if err != nil {
					return nil, fmt.Errorf("get parent task: %w", err)
				}
				if parent == nil {
					return nil, domain.ErrParentNotFound
				}

				// Check for circular reference
				if err := uc.checkCircularReference(in.TaskID, newParentID); err != nil {
					return nil, err
				}

				task.ParentID = &newParentID
			}
		}
	}

	// Handle skip_review change from editor
	if content.SkipReviewFound {
		task.SkipReview = content.SkipReview
	}

	// Atomically save task and comments together
	if err := uc.tasks.SaveTaskWithComments(task, updatedComments); err != nil {
		return nil, fmt.Errorf("save task with comments: %w", err)
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

// checkCircularReference checks if setting newParentID as parent of taskID would create a cycle.
// Returns ErrCircularReference if a cycle would be created.
func (uc *EditTask) checkCircularReference(taskID, newParentID int) error {
	// Self-reference is a cycle
	if taskID == newParentID {
		return domain.ErrCircularReference
	}

	// Check if newParentID is a descendant of taskID by traversing descendants
	return uc.checkIsDescendant(taskID, newParentID)
}

// checkIsDescendant checks if targetID is a descendant of taskID.
// Uses BFS to traverse the task tree.
func (uc *EditTask) checkIsDescendant(taskID, targetID int) error {
	// BFS queue of task IDs to check
	queue := []int{taskID}
	visited := make(map[int]bool)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true

		// Get children of current task
		children, err := uc.tasks.GetChildren(current)
		if err != nil {
			return fmt.Errorf("get children: %w", err)
		}

		for _, child := range children {
			if child.ID == targetID {
				// Found: targetID is a descendant of taskID
				return domain.ErrCircularReference
			}
			queue = append(queue, child.ID)
		}
	}

	return nil
}
