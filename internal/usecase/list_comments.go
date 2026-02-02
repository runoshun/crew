package usecase

import (
	"context"
	"slices"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// ListCommentsInput contains the parameters for listing comments.
type ListCommentsInput struct {
	Type          domain.CommentType // Filter by comment type (optional)
	Tags          []string           // Filter by tags (AND condition)
	AllNamespaces bool               // List comments across all namespaces when supported
}

// CommentWithTask contains a comment and its task metadata.
// Fields are ordered to minimize memory padding.
type CommentWithTask struct {
	Task    *domain.Task
	Comment domain.Comment
	Index   int
}

// ListCommentsOutput contains the result of listing comments.
type ListCommentsOutput struct {
	Comments []CommentWithTask
}

// ListComments is the use case for listing comments across tasks.
type ListComments struct {
	tasks domain.TaskRepository
}

// NewListComments creates a new ListComments use case.
func NewListComments(tasks domain.TaskRepository) *ListComments {
	return &ListComments{tasks: tasks}
}

// Execute lists comments matching the given input criteria.
func (uc *ListComments) Execute(_ context.Context, in ListCommentsInput) (*ListCommentsOutput, error) {
	if !in.Type.IsValid() {
		return nil, domain.ErrInvalidCommentType
	}
	filterTags := normalizeCommentTags(in.Tags)

	tasks, err := uc.listTasks(domain.TaskFilter{}, in.AllNamespaces)
	if err != nil {
		return nil, err
	}

	comments := make([]CommentWithTask, 0)
	for _, task := range tasks {
		if task == nil {
			continue
		}
		entries, err := uc.tasks.GetComments(task.ID)
		if err != nil {
			return nil, err
		}
		for idx, comment := range entries {
			if in.Type != domain.CommentTypeGeneral && comment.Type != in.Type {
				continue
			}
			if len(filterTags) > 0 && !commentHasTags(comment.Tags, filterTags) {
				continue
			}
			comments = append(comments, CommentWithTask{Task: task, Comment: comment, Index: idx})
		}
	}

	slices.SortFunc(comments, func(a, b CommentWithTask) int {
		if a.Comment.Time.Equal(b.Comment.Time) {
			return compareCommentTaskOrder(a, b)
		}
		if a.Comment.Time.After(b.Comment.Time) {
			return -1
		}
		return 1
	})

	return &ListCommentsOutput{Comments: comments}, nil
}

type commentNamespaceLister interface {
	ListAll(filter domain.TaskFilter) ([]*domain.Task, error)
}

func (uc *ListComments) listTasks(filter domain.TaskFilter, allNamespaces bool) ([]*domain.Task, error) {
	if allNamespaces {
		if lister, ok := uc.tasks.(commentNamespaceLister); ok {
			return lister.ListAll(filter)
		}
	}
	return uc.tasks.List(filter)
}

func commentHasTags(tags []string, filterTags []string) bool {
	if len(filterTags) == 0 {
		return true
	}
	normalized := normalizeCommentTags(tags)
	if len(normalized) == 0 {
		return false
	}
	set := make(map[string]bool, len(normalized))
	for _, tag := range normalized {
		set[tag] = true
	}
	for _, filter := range filterTags {
		if !set[filter] {
			return false
		}
	}
	return true
}

func compareCommentTaskOrder(a, b CommentWithTask) int {
	aNamespace := ""
	bNamespace := ""
	if a.Task != nil {
		aNamespace = a.Task.Namespace
	}
	if b.Task != nil {
		bNamespace = b.Task.Namespace
	}
	if aNamespace != bNamespace {
		if aNamespace < bNamespace {
			return -1
		}
		return 1
	}
	if a.Task != nil && b.Task != nil && a.Task.ID != b.Task.ID {
		return a.Task.ID - b.Task.ID
	}
	if a.Index != b.Index {
		return a.Index - b.Index
	}
	return 0
}
