package usecase

import (
	"context"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
)

func TestPruneTasks_Execute(t *testing.T) {
	// This test is superseded by TestPruneTasks_Execute_Real which uses
	// specific mock implementations for this use case.
}

// We need a better way to test this with the existing mocks or extend them.
// Let's rewrite the test to use a custom mock struct locally to override behaviors easily.

type mockGitForPrune struct {
	branches []string
	deleted  []string
}

func (m *mockGitForPrune) ListBranches() ([]string, error) {
	return m.branches, nil
}

func (m *mockGitForPrune) DeleteBranch(branch string, force bool) error {
	m.deleted = append(m.deleted, branch)
	return nil
}

// Implement other methods required by interface but not used
func (m *mockGitForPrune) CurrentBranch() (string, error)                { return "", nil }
func (m *mockGitForPrune) BranchExists(string) (bool, error)             { return true, nil }
func (m *mockGitForPrune) HasUncommittedChanges(string) (bool, error)    { return false, nil }
func (m *mockGitForPrune) HasMergeConflict(string, string) (bool, error) { return false, nil }
func (m *mockGitForPrune) Merge(string, bool) error                      { return nil }

type mockWorktreeForPrune struct {
	worktrees []domain.WorktreeInfo
	removed   []string
}

func (m *mockWorktreeForPrune) List() ([]domain.WorktreeInfo, error) {
	return m.worktrees, nil
}

func (m *mockWorktreeForPrune) Remove(branch string) error {
	m.removed = append(m.removed, branch)
	return nil
}

// Other methods
func (m *mockWorktreeForPrune) Create(string, string) (string, error)              { return "", nil }
func (m *mockWorktreeForPrune) SetupWorktree(string, *domain.WorktreeConfig) error { return nil }
func (m *mockWorktreeForPrune) Resolve(string) (string, error)                     { return "", nil }
func (m *mockWorktreeForPrune) Exists(string) (bool, error)                        { return false, nil }

func TestPruneTasks_Execute_Real(t *testing.T) {
	tests := []struct {
		name              string
		input             PruneTasksInput
		tasks             []*domain.Task
		branches          []string
		worktrees         []domain.WorktreeInfo
		expectedTasks     int
		expectedBranches  int
		expectedWorktrees int
	}{
		{
			name: "prune closed tasks",
			input: PruneTasksInput{
				All:    false,
				DryRun: false,
			},
			tasks: []*domain.Task{
				{ID: 1, Status: domain.StatusClosed},
				{ID: 2, Status: domain.StatusTodo},
			},
			branches:          []string{"crew-1", "other"},
			worktrees:         nil,
			expectedTasks:     1, // Task 1
			expectedBranches:  1, // crew-1
			expectedWorktrees: 0,
		},
		{
			name: "prune all (closed and done)",
			input: PruneTasksInput{
				All:    true,
				DryRun: false,
			},
			tasks: []*domain.Task{
				{ID: 1, Status: domain.StatusClosed},
				{ID: 2, Status: domain.StatusDone},
				{ID: 3, Status: domain.StatusInProgress},
			},
			branches:          []string{"crew-1", "crew-2", "crew-3"},
			worktrees:         nil,
			expectedTasks:     2, // 1 and 2
			expectedBranches:  2, // crew-1 and crew-2
			expectedWorktrees: 0,
		},
		{
			name: "orphan branches",
			input: PruneTasksInput{
				All:    false,
				DryRun: false,
			},
			tasks: []*domain.Task{
				{ID: 2, Status: domain.StatusTodo},
			},
			branches:          []string{"crew-1", "crew-2"}, // crew-1 is orphan (task 1 not in DB)
			worktrees:         nil,
			expectedTasks:     0,
			expectedBranches:  1, // crew-1
			expectedWorktrees: 0,
		},
		{
			name: "dry run",
			input: PruneTasksInput{
				All:    true,
				DryRun: true,
			},
			tasks: []*domain.Task{
				{ID: 1, Status: domain.StatusClosed},
			},
			branches:          []string{"crew-1"},
			worktrees:         nil,
			expectedTasks:     1,
			expectedBranches:  1,
			expectedWorktrees: 0,
		},
		{
			name: "prune worktrees",
			input: PruneTasksInput{
				All: false,
			},
			tasks: []*domain.Task{
				{ID: 1, Status: domain.StatusClosed},
			},
			branches: []string{"crew-1"},
			worktrees: []domain.WorktreeInfo{
				{Branch: "crew-1", Path: "/tmp/1"},
			},
			expectedTasks:     1,
			expectedBranches:  1,
			expectedWorktrees: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := testutil.NewMockTaskRepository()
			for _, task := range tt.tasks {
				_ = repo.Save(task)
			}

			git := &mockGitForPrune{branches: tt.branches}
			wt := &mockWorktreeForPrune{worktrees: tt.worktrees}

			uc := NewPruneTasks(repo, wt, git)
			out, err := uc.Execute(context.Background(), tt.input)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(out.DeletedTasks) != tt.expectedTasks {
				t.Errorf("expected %d deleted tasks, got %d", tt.expectedTasks, len(out.DeletedTasks))
			}

			if len(out.DeletedBranches) != tt.expectedBranches {
				t.Errorf("expected %d deleted branches, got %d", tt.expectedBranches, len(out.DeletedBranches))
			}

			if len(out.DeletedWorktrees) != tt.expectedWorktrees {
				t.Errorf("expected %d deleted worktrees, got %d", tt.expectedWorktrees, len(out.DeletedWorktrees))
			}

			// Verify actual deletion for non-dry-run
			if !tt.input.DryRun {
				// Check DB
				for _, dt := range out.DeletedTasks {
					got, _ := repo.Get(dt.ID)
					if got != nil {
						t.Errorf("task %d should have been deleted from DB", dt.ID)
					}
				}
				// Check Git
				if len(git.deleted) != tt.expectedBranches {
					t.Errorf("expected %d git delete calls, got %d", tt.expectedBranches, len(git.deleted))
				}
				// Check Worktrees
				if len(wt.removed) != tt.expectedWorktrees {
					t.Errorf("expected %d worktree remove calls, got %d", tt.expectedWorktrees, len(wt.removed))
				}
			} else {
				// Dry run verification
				if len(git.deleted) > 0 {
					t.Error("git delete should not be called in dry run")
				}
			}
		})
	}
}
