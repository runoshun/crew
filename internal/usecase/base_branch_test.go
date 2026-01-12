package usecase

import (
	"errors"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockGitForBaseBranch is a minimal mock for testing ResolveBaseBranch.
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

func TestResolveBaseBranch(t *testing.T) {
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

			got, err := ResolveBaseBranch(task, mockGit)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
