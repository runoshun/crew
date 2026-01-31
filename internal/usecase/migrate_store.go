// Package usecase contains the application use cases.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// MigrateStoreInput contains parameters for MigrateStore.
type MigrateStoreInput struct {
	// SkipComments migrates tasks without reading/writing comments.
	// This is useful for performance testing or recovering from legacy comment corruption.
	SkipComments bool

	// StrictComments fails the migration if comments cannot be read from the legacy store.
	// If false, comment read errors are tolerated and the task is migrated without comments.
	StrictComments bool
}

// MigrateStoreOutput contains migration results.
type MigrateStoreOutput struct {
	Total    int
	Migrated int
	Skipped  int
	// SkippedComments is the number of tasks whose comments could not be read
	// from the legacy store and were migrated without comments.
	SkippedComments int
}

// MigrateStore migrates tasks from a legacy store to the file store.
type MigrateStore struct {
	source   domain.TaskRepository
	dest     domain.TaskRepository
	destInit domain.StoreInitializer
}

// NewMigrateStore creates a new MigrateStore use case.
func NewMigrateStore(source, dest domain.TaskRepository, destInit domain.StoreInitializer) *MigrateStore {
	return &MigrateStore{source: source, dest: dest, destInit: destInit}
}

// Execute migrates all tasks and comments from the legacy store.
// Existing destination tasks are skipped if identical; otherwise it fails.
func (uc *MigrateStore) Execute(_ context.Context, in MigrateStoreInput) (*MigrateStoreOutput, error) {
	if uc.destInit == nil {
		return nil, errors.New("destination store initializer is nil")
	}
	if uc.source == nil || uc.dest == nil {
		return nil, errors.New("source or destination store is nil")
	}

	if _, err := uc.destInit.Initialize(); err != nil {
		return nil, fmt.Errorf("initialize destination store: %w", err)
	}

	tasks, err := uc.source.List(domain.TaskFilter{})
	if err != nil {
		return nil, fmt.Errorf("list source tasks: %w", err)
	}

	out := &MigrateStoreOutput{Total: len(tasks)}
	for _, task := range tasks {
		if task == nil {
			continue
		}

		var comments []domain.Comment
		commentsOK := true
		if !in.SkipComments {
			c, err := uc.source.GetComments(task.ID)
			if err != nil {
				if in.StrictComments {
					return nil, fmt.Errorf("get source comments for %d: %w", task.ID, err)
				}
				commentsOK = false
				out.SkippedComments++
				comments = nil
			} else {
				comments = normalizeComments(c)
			}
		}

		existing, err := uc.dest.Get(task.ID)
		if err != nil {
			return nil, fmt.Errorf("check destination task %d: %w", task.ID, err)
		}
		if existing != nil {
			sameTask := tasksEqual(normalizeTaskForCompare(task), normalizeTaskForCompare(existing))
			if in.SkipComments || !commentsOK {
				if sameTask {
					out.Skipped++
					continue
				}
				return nil, fmt.Errorf("%w: task %d", domain.ErrMigrationConflict, task.ID)
			}

			destComments, err := uc.dest.GetComments(task.ID)
			if err != nil {
				return nil, fmt.Errorf("get destination comments for %d: %w", task.ID, err)
			}
			if sameTask && commentsEqual(comments, normalizeComments(destComments)) {
				out.Skipped++
				continue
			}
			return nil, fmt.Errorf("%w: task %d", domain.ErrMigrationConflict, task.ID)
		}

		cloned := cloneTask(task)
		domain.NormalizeStatus(cloned)
		if err := uc.dest.SaveTaskWithComments(cloned, comments); err != nil {
			return nil, fmt.Errorf("save destination task %d: %w", task.ID, err)
		}
		out.Migrated++
	}

	if _, err := uc.destInit.Initialize(); err != nil {
		return nil, fmt.Errorf("repair destination store: %w", err)
	}

	return out, nil
}

func cloneTask(task *domain.Task) *domain.Task {
	if task == nil {
		return nil
	}
	cloned := *task
	if task.Labels != nil {
		cloned.Labels = append([]string{}, task.Labels...)
	}
	return &cloned
}

func normalizeTaskForCompare(task *domain.Task) *domain.Task {
	cloned := cloneTask(task)
	if cloned == nil {
		return nil
	}
	cloned.Namespace = ""
	if cloned.Labels != nil {
		slices.Sort(cloned.Labels)
		if len(cloned.Labels) == 0 {
			cloned.Labels = nil
		}
	}
	domain.NormalizeStatus(cloned)
	return cloned
}

func normalizeComments(comments []domain.Comment) []domain.Comment {
	if len(comments) == 0 {
		return nil
	}
	return comments
}

func tasksEqual(a, b *domain.Task) bool {
	if a == nil || b == nil {
		return a == b
	}
	return reflect.DeepEqual(a, b)
}

func commentsEqual(a, b []domain.Comment) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}
