package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkspaceRepo_DisplayName(t *testing.T) {
	tests := []struct {
		name     string
		repo     WorkspaceRepo
		expected string
	}{
		{
			name:     "uses Name if set",
			repo:     WorkspaceRepo{Path: "/path/to/repo", Name: "my-repo"},
			expected: "my-repo",
		},
		{
			name:     "uses basename if Name is empty",
			repo:     WorkspaceRepo{Path: "/path/to/my-project"},
			expected: "my-project",
		},
		{
			name:     "handles path without slashes",
			repo:     WorkspaceRepo{Path: "single-dir"},
			expected: "single-dir",
		},
		{
			name:     "handles trailing slash",
			repo:     WorkspaceRepo{Path: "/path/to/repo/"},
			expected: "repo",
		},
		{
			name:     "handles multiple trailing slashes",
			repo:     WorkspaceRepo{Path: "/path/to/repo///"},
			expected: "repo",
		},
		{
			name:     "handles root path with trailing slash",
			repo:     WorkspaceRepo{Path: "/"},
			expected: "/", // Return original for edge case
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.repo.DisplayName()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRepoState_String(t *testing.T) {
	tests := []struct {
		state    RepoState
		expected string
	}{
		{RepoStateOK, "OK"},
		{RepoStateNotGitRepo, "Not a git repo"},
		{RepoStateNotInitialized, "Not initialized"},
		{RepoStateConfigError, "Config error"},
		{RepoStateLoadError, "Load error"},
		{RepoStateNotFound, "Not found"},
		{RepoState(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

func TestNewTaskSummary(t *testing.T) {
	tasks := []*Task{
		{ID: 1, Status: StatusTodo},
		{ID: 2, Status: StatusTodo},
		{ID: 3, Status: StatusInProgress},
		{ID: 4, Status: StatusDone},
		{ID: 5, Status: StatusMerged},
		{ID: 6, Status: StatusError},
		{ID: 7, Status: StatusClosed},
		{ID: 8, Status: StatusClosed},
	}

	summary := NewTaskSummary(tasks)

	assert.Equal(t, 2, summary.Todo)
	assert.Equal(t, 1, summary.InProgress)
	assert.Equal(t, 1, summary.Done)
	assert.Equal(t, 1, summary.Merged)
	assert.Equal(t, 1, summary.Error)
	assert.Equal(t, 2, summary.Closed)
	assert.Equal(t, 5, summary.TotalActive) // Todo + InProgress + Done + Error
}

func TestNewTaskSummary_EmptyList(t *testing.T) {
	summary := NewTaskSummary(nil)

	assert.Equal(t, 0, summary.TotalActive)
}
