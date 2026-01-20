package usecase

import (
	"errors"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockGitForBaseBranch is a minimal mock for testing resolveBaseBranch.
type MockGitForBaseBranch struct {
	defaultBranch    string
	defaultBranchErr error
}

func (m *MockGitForBaseBranch) GetDefaultBranch() (string, error) {
	return m.defaultBranch, m.defaultBranchErr
}

func (m *MockGitForBaseBranch) CurrentBranch() (string, error) {
	return "", errors.New("not implemented")
}

func (m *MockGitForBaseBranch) BranchExists(_ string) (bool, error) {
	return false, errors.New("not implemented")
}

func (m *MockGitForBaseBranch) HasUncommittedChanges(_ string) (bool, error) {
	return false, errors.New("not implemented")
}

func (m *MockGitForBaseBranch) HasMergeConflict(_, _ string) (bool, error) {
	return false, errors.New("not implemented")
}

func (m *MockGitForBaseBranch) Merge(_ string, _ bool) error {
	return errors.New("not implemented")
}

func (m *MockGitForBaseBranch) DeleteBranch(_ string, _ bool) error {
	return errors.New("not implemented")
}

func (m *MockGitForBaseBranch) ListBranches() ([]string, error) {
	return nil, errors.New("not implemented")
}

func TestResolveBaseBranch_Private(t *testing.T) {
	tests := []struct {
		name             string
		taskBaseBranch   string
		defaultBranch    string
		defaultBranchErr error
		want             string
		wantErr          bool
	}{
		{
			name:           "task has BaseBranch set",
			taskBaseBranch: "develop",
			defaultBranch:  "main",
			want:           "develop",
			wantErr:        false,
		},
		{
			name:           "task BaseBranch empty, use GetDefaultBranch",
			taskBaseBranch: "",
			defaultBranch:  "main",
			want:           "main",
			wantErr:        false,
		},
		{
			name:             "task BaseBranch empty, GetDefaultBranch fails",
			taskBaseBranch:   "",
			defaultBranch:    "",
			defaultBranchErr: errors.New("git error"),
			want:             "",
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &domain.Task{
				ID:         1,
				BaseBranch: tt.taskBaseBranch,
			}

			mockGit := &MockGitForBaseBranch{
				defaultBranch:    tt.defaultBranch,
				defaultBranchErr: tt.defaultBranchErr,
			}

			got, err := resolveBaseBranch(task, mockGit)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// MockGitForNewTaskBaseBranch is a minimal mock for testing resolveNewTaskBaseBranch.
type MockGitForNewTaskBaseBranch struct {
	currentBranch    string
	currentBranchErr error
	defaultBranch    string
	defaultBranchErr error
}

func (m *MockGitForNewTaskBaseBranch) GetDefaultBranch() (string, error) {
	return m.defaultBranch, m.defaultBranchErr
}

func (m *MockGitForNewTaskBaseBranch) CurrentBranch() (string, error) {
	return m.currentBranch, m.currentBranchErr
}

func (m *MockGitForNewTaskBaseBranch) BranchExists(_ string) (bool, error) {
	return false, errors.New("not implemented")
}

func (m *MockGitForNewTaskBaseBranch) HasUncommittedChanges(_ string) (bool, error) {
	return false, errors.New("not implemented")
}

func (m *MockGitForNewTaskBaseBranch) HasMergeConflict(_, _ string) (bool, error) {
	return false, errors.New("not implemented")
}

func (m *MockGitForNewTaskBaseBranch) Merge(_ string, _ bool) error {
	return errors.New("not implemented")
}

func (m *MockGitForNewTaskBaseBranch) DeleteBranch(_ string, _ bool) error {
	return errors.New("not implemented")
}

func (m *MockGitForNewTaskBaseBranch) ListBranches() ([]string, error) {
	return nil, errors.New("not implemented")
}

func TestResolveNewTaskBaseBranch(t *testing.T) {
	tests := []struct {
		name             string
		baseBranch       string
		newTaskBase      string // config.Tasks.NewTaskBase
		configNil        bool   // set config to nil
		currentBranch    string
		currentBranchErr error
		defaultBranch    string
		defaultBranchErr error
		want             string
		wantErr          bool
	}{
		{
			name:          "baseBranch specified",
			baseBranch:    "feature/test",
			currentBranch: "main",
			want:          "feature/test",
			wantErr:       false,
		},
		{
			name:          "baseBranch empty, config nil, use CurrentBranch",
			baseBranch:    "",
			configNil:     true,
			currentBranch: "feature/current",
			want:          "feature/current",
			wantErr:       false,
		},
		{
			name:          "baseBranch empty, newTaskBase empty, use CurrentBranch",
			baseBranch:    "",
			newTaskBase:   "",
			currentBranch: "feature/current",
			want:          "feature/current",
			wantErr:       false,
		},
		{
			name:          "baseBranch empty, newTaskBase current, use CurrentBranch",
			baseBranch:    "",
			newTaskBase:   "current",
			currentBranch: "feature/current",
			want:          "feature/current",
			wantErr:       false,
		},
		{
			name:          "baseBranch empty, newTaskBase default, use GetDefaultBranch",
			baseBranch:    "",
			newTaskBase:   "default",
			defaultBranch: "main",
			want:          "main",
			wantErr:       false,
		},
		{
			name:             "baseBranch empty, newTaskBase default, GetDefaultBranch fails",
			baseBranch:       "",
			newTaskBase:      "default",
			defaultBranchErr: errors.New("git error"),
			want:             "",
			wantErr:          true,
		},
		{
			name:             "baseBranch empty, newTaskBase empty, CurrentBranch fails",
			baseBranch:       "",
			newTaskBase:      "",
			currentBranchErr: errors.New("git error"),
			want:             "",
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := &MockGitForNewTaskBaseBranch{
				currentBranch:    tt.currentBranch,
				currentBranchErr: tt.currentBranchErr,
				defaultBranch:    tt.defaultBranch,
				defaultBranchErr: tt.defaultBranchErr,
			}

			var config *domain.Config
			if !tt.configNil {
				config = &domain.Config{
					Tasks: domain.TasksConfig{
						NewTaskBase: tt.newTaskBase,
					},
				}
			}

			got, err := resolveNewTaskBaseBranch(tt.baseBranch, mockGit, config)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
