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
	defaultBranch string
	err           error
}

func (m *MockGitForBaseBranch) GetDefaultBranch() (string, error) {
	return m.defaultBranch, m.err
}

func (m *MockGitForBaseBranch) GetNewTaskBaseBranch() (string, error) {
	return "", errors.New("not implemented")
}

func (m *MockGitForBaseBranch) CurrentBranch() (string, error) {
	return "", errors.New("not implemented")
}

func (m *MockGitForBaseBranch) BranchExists(branch string) (bool, error) {
	return false, errors.New("not implemented")
}

func (m *MockGitForBaseBranch) HasUncommittedChanges(worktreePath string) (bool, error) {
	return false, errors.New("not implemented")
}

func (m *MockGitForBaseBranch) HasMergeConflict(branch, target string) (bool, error) {
	return false, errors.New("not implemented")
}

func (m *MockGitForBaseBranch) Merge(branch string, noFF bool) error {
	return errors.New("not implemented")
}

func (m *MockGitForBaseBranch) DeleteBranch(branch string, force bool) error {
	return errors.New("not implemented")
}

func (m *MockGitForBaseBranch) ListBranches() ([]string, error) {
	return nil, errors.New("not implemented")
}

func TestResolveBaseBranch_Private(t *testing.T) {
	tests := []struct {
		name           string
		taskBaseBranch string
		defaultBranch  string
		gitErr         error
		want           string
		wantErr        bool
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
			name:           "task BaseBranch empty, GetDefaultBranch fails",
			taskBaseBranch: "",
			defaultBranch:  "",
			gitErr:         errors.New("git error"),
			want:           "",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &domain.Task{
				ID:         1,
				BaseBranch: tt.taskBaseBranch,
			}

			mockGit := &MockGitForBaseBranch{
				defaultBranch: tt.defaultBranch,
				err:           tt.gitErr,
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
	newTaskBaseBranch string
	err               error
}

func (m *MockGitForNewTaskBaseBranch) GetDefaultBranch() (string, error) {
	return "", errors.New("not implemented")
}

func (m *MockGitForNewTaskBaseBranch) GetNewTaskBaseBranch() (string, error) {
	return m.newTaskBaseBranch, m.err
}

func (m *MockGitForNewTaskBaseBranch) CurrentBranch() (string, error) {
	return "", errors.New("not implemented")
}

func (m *MockGitForNewTaskBaseBranch) BranchExists(branch string) (bool, error) {
	return false, errors.New("not implemented")
}

func (m *MockGitForNewTaskBaseBranch) HasUncommittedChanges(worktreePath string) (bool, error) {
	return false, errors.New("not implemented")
}

func (m *MockGitForNewTaskBaseBranch) HasMergeConflict(branch, target string) (bool, error) {
	return false, errors.New("not implemented")
}

func (m *MockGitForNewTaskBaseBranch) Merge(branch string, noFF bool) error {
	return errors.New("not implemented")
}

func (m *MockGitForNewTaskBaseBranch) DeleteBranch(branch string, force bool) error {
	return errors.New("not implemented")
}

func (m *MockGitForNewTaskBaseBranch) ListBranches() ([]string, error) {
	return nil, errors.New("not implemented")
}

func TestResolveNewTaskBaseBranch(t *testing.T) {
	tests := []struct {
		name              string
		baseBranch        string
		newTaskBaseBranch string
		gitErr            error
		want              string
		wantErr           bool
	}{
		{
			name:              "baseBranch specified",
			baseBranch:        "feature/test",
			newTaskBaseBranch: "main",
			want:              "feature/test",
			wantErr:           false,
		},
		{
			name:              "baseBranch empty, use GetNewTaskBaseBranch",
			baseBranch:        "",
			newTaskBaseBranch: "develop",
			want:              "develop",
			wantErr:           false,
		},
		{
			name:              "baseBranch empty, GetNewTaskBaseBranch fails",
			baseBranch:        "",
			newTaskBaseBranch: "",
			gitErr:            errors.New("git error"),
			want:              "",
			wantErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := &MockGitForNewTaskBaseBranch{
				newTaskBaseBranch: tt.newTaskBaseBranch,
				err:               tt.gitErr,
			}

			got, err := resolveNewTaskBaseBranch(tt.baseBranch, mockGit)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
