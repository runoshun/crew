// Package usecase contains application use cases.
package usecase

import (
	"context"
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// CreateTasksFromFileInput contains the parameters for creating tasks from a file.
type CreateTasksFromFileInput struct {
	Content    string // File content (Markdown with frontmatter)
	BaseBranch string // Base branch for all tasks (optional, empty = use default)
	DryRun     bool   // If true, parse and validate without creating tasks
}

// CreatedTask represents a task that was created from file input.
// Fields are ordered to minimize memory padding.
type CreatedTask struct {
	ParentID    *int
	Title       string
	Description string
	Labels      []string
	ID          int
}

// CreateTasksFromFileOutput contains the result of creating tasks from a file.
type CreateTasksFromFileOutput struct {
	Tasks []CreatedTask // Created tasks (or tasks that would be created in dry-run mode)
}

// CreateTasksFromFile is the use case for creating tasks from a file.
type CreateTasksFromFile struct {
	tasks        domain.TaskRepository
	git          domain.Git
	configLoader domain.ConfigLoader
	clock        domain.Clock
	logger       domain.Logger
}

// NewCreateTasksFromFile creates a new CreateTasksFromFile use case.
func NewCreateTasksFromFile(
	tasks domain.TaskRepository,
	git domain.Git,
	configLoader domain.ConfigLoader,
	clock domain.Clock,
	logger domain.Logger,
) *CreateTasksFromFile {
	return &CreateTasksFromFile{
		tasks:        tasks,
		git:          git,
		configLoader: configLoader,
		clock:        clock,
		logger:       logger,
	}
}

// Execute creates tasks from the given file content.
func (uc *CreateTasksFromFile) Execute(_ context.Context, in CreateTasksFromFileInput) (*CreateTasksFromFileOutput, error) {
	// Parse task drafts from content
	drafts, err := domain.ParseTaskDrafts(in.Content)
	if err != nil {
		return nil, err
	}

	// Load config for base branch resolution
	var config *domain.Config
	if uc.configLoader != nil {
		config, err = uc.configLoader.Load()
		if err != nil {
			return nil, fmt.Errorf("load config: %w", err)
		}
	}

	// Resolve base branch
	baseBranch, err := resolveNewTaskBaseBranch(in.BaseBranch, uc.git, config)
	if err != nil {
		return nil, err
	}

	// If dry-run, just return parsed drafts without creating tasks
	if in.DryRun {
		return uc.dryRun(drafts)
	}

	// Create tasks
	return uc.createTasks(drafts, baseBranch)
}

// dryRun validates and returns tasks that would be created.
func (uc *CreateTasksFromFile) dryRun(drafts []domain.TaskDraft) (*CreateTasksFromFileOutput, error) {
	result := &CreateTasksFromFileOutput{
		Tasks: make([]CreatedTask, 0, len(drafts)),
	}

	// Build a map of relative indices for parent resolution preview
	relativeIDs := make(map[int]int)
	for i := range drafts {
		relativeIDs[i+1] = i + 1 // In dry-run, use relative index as pseudo-ID
	}

	for i, draft := range drafts {
		var parentID *int

		if draft.ParentRef != "" {
			// Try to resolve parent reference
			resolved, err := domain.ResolveParentRef(draft.ParentRef, relativeIDs)
			if err != nil {
				return nil, fmt.Errorf("task %d: %w", i+1, err)
			}

			if resolved != nil {
				// Check if it's a relative reference within file
				if *resolved <= len(drafts) {
					// Relative reference - show as relative index
					parentID = resolved
				} else {
					// Absolute reference - verify task exists
					parent, err := uc.tasks.Get(*resolved)
					if err != nil {
						return nil, fmt.Errorf("task %d: get parent task: %w", i+1, err)
					}
					if parent == nil {
						return nil, fmt.Errorf("task %d: %w", i+1, domain.ErrParentNotFound)
					}
					parentID = resolved
				}
			}
		}

		result.Tasks = append(result.Tasks, CreatedTask{
			ID:          i + 1, // Use 1-based index as pseudo-ID in dry-run
			Title:       draft.Title,
			Description: draft.Description,
			Labels:      draft.Labels,
			ParentID:    parentID,
		})
	}

	return result, nil
}

// createTasks creates tasks from drafts.
func (uc *CreateTasksFromFile) createTasks(drafts []domain.TaskDraft, baseBranch string) (*CreateTasksFromFileOutput, error) {
	result := &CreateTasksFromFileOutput{
		Tasks: make([]CreatedTask, 0, len(drafts)),
	}

	// Map of relative index (1-based) to created task ID
	createdIDs := make(map[int]int)

	now := uc.clock.Now()

	for i, draft := range drafts {
		// Resolve parent reference
		var parentID *int
		if draft.ParentRef != "" {
			resolved, err := domain.ResolveParentRef(draft.ParentRef, createdIDs)
			if err != nil {
				return nil, fmt.Errorf("task %d: %w", i+1, err)
			}

			if resolved != nil {
				// Verify parent exists (either just created or existing)
				if _, ok := createdIDs[*resolved]; !ok {
					// Not a relative reference, check if it's an existing task
					parent, err := uc.tasks.Get(*resolved)
					if err != nil {
						return nil, fmt.Errorf("task %d: get parent task: %w", i+1, err)
					}
					if parent == nil {
						return nil, fmt.Errorf("task %d: %w", i+1, domain.ErrParentNotFound)
					}
				}
				parentID = resolved
			}
		}

		// Get next task ID
		id, err := uc.tasks.NextID()
		if err != nil {
			return nil, fmt.Errorf("task %d: generate task ID: %w", i+1, err)
		}

		// Create task
		task := &domain.Task{
			ID:          id,
			ParentID:    parentID,
			Title:       draft.Title,
			Description: draft.Description,
			Status:      domain.StatusTodo,
			Created:     now,
			Labels:      draft.Labels,
			BaseBranch:  baseBranch,
		}

		// Save task
		if err := uc.tasks.Save(task); err != nil {
			return nil, fmt.Errorf("task %d: save task: %w", i+1, err)
		}

		// Track created ID for relative reference resolution
		createdIDs[i+1] = id

		// Log task creation
		if uc.logger != nil {
			uc.logger.Info(id, "task", fmt.Sprintf("created from file: %q", draft.Title))
		}

		result.Tasks = append(result.Tasks, CreatedTask{
			ID:          id,
			Title:       draft.Title,
			Description: draft.Description,
			Labels:      draft.Labels,
			ParentID:    parentID,
		})
	}

	return result, nil
}
