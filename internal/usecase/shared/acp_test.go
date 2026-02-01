package shared

import (
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/require"
)

type stubGit struct {
	email string
}

func (s stubGit) CurrentBranch() (string, error)                { return "", nil }
func (s stubGit) UserEmail() (string, error)                    { return s.email, nil }
func (s stubGit) BranchExists(string) (bool, error)             { return false, nil }
func (s stubGit) HasUncommittedChanges(string) (bool, error)    { return false, nil }
func (s stubGit) HasMergeConflict(string, string) (bool, error) { return false, nil }
func (s stubGit) GetMergeConflictFiles(string, string) ([]string, error) {
	return nil, nil
}
func (s stubGit) Merge(string, bool) error          { return nil }
func (s stubGit) DeleteBranch(string, bool) error   { return nil }
func (s stubGit) ListBranches() ([]string, error)   { return nil, nil }
func (s stubGit) GetDefaultBranch() (string, error) { return "", nil }

func TestResolveACPNamespace_ConfigNamespace(t *testing.T) {
	cfg := &domain.Config{Tasks: domain.TasksConfig{Namespace: "Team Alpha"}}

	got := ResolveACPNamespace(cfg, stubGit{email: "user@example.com"})

	require.Equal(t, "team-alpha", got)
}

func TestResolveACPNamespace_EmailFallback(t *testing.T) {
	cfg := &domain.Config{}

	got := ResolveACPNamespace(cfg, stubGit{email: "Dev.User+1@example.com"})

	require.Equal(t, "dev-user-1", got)
}
