package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTasksFromFile_Execute_SingleTask(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	mockGit := &testutil.MockGit{CurrentBranchName: testutil.StringPtr("main")}
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCreateTasksFromFile(repo, mockGit, configLoader, clock, nil)

	content := `---
title: Test Task
labels: [backend, urgent]
---
Task description here.`

	// Execute
	out, err := uc.Execute(context.Background(), CreateTasksFromFileInput{
		Content: content,
	})

	// Assert
	require.NoError(t, err)
	require.Len(t, out.Tasks, 1)
	assert.Equal(t, 1, out.Tasks[0].ID)
	assert.Equal(t, "Test Task", out.Tasks[0].Title)
	assert.Equal(t, "Task description here.", out.Tasks[0].Description)
	assert.Equal(t, []string{"backend", "urgent"}, out.Tasks[0].Labels)
	assert.Nil(t, out.Tasks[0].ParentID)

	// Verify saved task
	task := repo.Tasks[1]
	require.NotNil(t, task)
	assert.Equal(t, "Test Task", task.Title)
	assert.Equal(t, "main", task.BaseBranch)
}

func TestCreateTasksFromFile_Execute_MultipleTasks(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	mockGit := &testutil.MockGit{CurrentBranchName: testutil.StringPtr("develop")}
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCreateTasksFromFile(repo, mockGit, configLoader, clock, nil)

	content := `---
title: Phase 1
labels: [backend]
---
Phase 1 description.

---
title: Phase 2
parent: 1
---
Phase 2 description.

---
title: Phase 3
parent: 1
---
Phase 3 description.`

	// Execute
	out, err := uc.Execute(context.Background(), CreateTasksFromFileInput{
		Content: content,
	})

	// Assert
	require.NoError(t, err)
	require.Len(t, out.Tasks, 3)

	// Task 1 (root)
	assert.Equal(t, 1, out.Tasks[0].ID)
	assert.Equal(t, "Phase 1", out.Tasks[0].Title)
	assert.Nil(t, out.Tasks[0].ParentID)

	// Task 2 (child of 1)
	assert.Equal(t, 2, out.Tasks[1].ID)
	assert.Equal(t, "Phase 2", out.Tasks[1].Title)
	require.NotNil(t, out.Tasks[1].ParentID)
	assert.Equal(t, 1, *out.Tasks[1].ParentID)

	// Task 3 (child of 1)
	assert.Equal(t, 3, out.Tasks[2].ID)
	assert.Equal(t, "Phase 3", out.Tasks[2].Title)
	require.NotNil(t, out.Tasks[2].ParentID)
	assert.Equal(t, 1, *out.Tasks[2].ParentID)

	// Verify all tasks saved
	assert.Len(t, repo.Tasks, 3)
}

func TestCreateTasksFromFile_Execute_AbsoluteParent(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	// Pre-create existing parent task
	repo.Tasks[100] = &domain.Task{
		ID:     100,
		Title:  "Existing Parent",
		Status: domain.StatusTodo,
	}
	repo.NextIDN = 1

	mockGit := &testutil.MockGit{CurrentBranchName: testutil.StringPtr("main")}
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCreateTasksFromFile(repo, mockGit, configLoader, clock, nil)

	content := `---
title: Child Task
parent: #100
---
Child of existing task.`

	// Execute
	out, err := uc.Execute(context.Background(), CreateTasksFromFileInput{
		Content: content,
	})

	// Assert
	require.NoError(t, err)
	require.Len(t, out.Tasks, 1)
	assert.Equal(t, 1, out.Tasks[0].ID)
	require.NotNil(t, out.Tasks[0].ParentID)
	assert.Equal(t, 100, *out.Tasks[0].ParentID)
}

func TestCreateTasksFromFile_Execute_AbsoluteParentWithHashPrefix(t *testing.T) {
	// Setup - test that #1 refers to existing task 1, not the first task in file
	repo := testutil.NewMockTaskRepository()
	// Pre-create existing parent task with ID 1
	repo.Tasks[1] = &domain.Task{
		ID:     1,
		Title:  "Existing Task 1",
		Status: domain.StatusTodo,
	}
	repo.NextIDN = 2

	mockGit := &testutil.MockGit{CurrentBranchName: testutil.StringPtr("main")}
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCreateTasksFromFile(repo, mockGit, configLoader, clock, nil)

	content := `---
title: First Task in File
---
This is the first task.

---
title: Second Task in File
parent: #1
---
This should reference existing task 1, not the first task in this file.`

	// Execute
	out, err := uc.Execute(context.Background(), CreateTasksFromFileInput{
		Content: content,
	})

	// Assert
	require.NoError(t, err)
	require.Len(t, out.Tasks, 2)

	// First task should have no parent
	assert.Equal(t, 2, out.Tasks[0].ID)
	assert.Nil(t, out.Tasks[0].ParentID)

	// Second task should reference existing task 1, not the first task in file (ID 2)
	assert.Equal(t, 3, out.Tasks[1].ID)
	require.NotNil(t, out.Tasks[1].ParentID)
	assert.Equal(t, 1, *out.Tasks[1].ParentID) // Points to existing task 1
}

func TestCreateTasksFromFile_Execute_DryRun(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	mockGit := &testutil.MockGit{CurrentBranchName: testutil.StringPtr("main")}
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCreateTasksFromFile(repo, mockGit, configLoader, clock, nil)

	content := `---
title: Task 1
---
Description 1.

---
title: Task 2
parent: 1
---
Description 2.`

	// Execute with dry-run
	out, err := uc.Execute(context.Background(), CreateTasksFromFileInput{
		Content: content,
		DryRun:  true,
	})

	// Assert
	require.NoError(t, err)
	require.Len(t, out.Tasks, 2)

	// Tasks should have pseudo-IDs
	assert.Equal(t, 1, out.Tasks[0].ID)
	assert.Equal(t, 2, out.Tasks[1].ID)

	// Verify NO tasks were actually saved
	assert.Empty(t, repo.Tasks)
}

func TestCreateTasksFromFile_Execute_WithBaseBranch(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	mockGit := &testutil.MockGit{CurrentBranchName: testutil.StringPtr("main")}
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCreateTasksFromFile(repo, mockGit, configLoader, clock, nil)

	content := `---
title: Test Task
---
Description.`

	// Execute with explicit base branch
	out, err := uc.Execute(context.Background(), CreateTasksFromFileInput{
		Content:    content,
		BaseBranch: "feature/base",
	})

	// Assert
	require.NoError(t, err)
	require.Len(t, out.Tasks, 1)

	// Verify base branch
	task := repo.Tasks[1]
	require.NotNil(t, task)
	assert.Equal(t, "feature/base", task.BaseBranch)
}

func TestCreateTasksFromFile_Execute_EmptyFile(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	mockGit := &testutil.MockGit{CurrentBranchName: testutil.StringPtr("main")}
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCreateTasksFromFile(repo, mockGit, configLoader, clock, nil)

	// Execute with empty content
	_, err := uc.Execute(context.Background(), CreateTasksFromFileInput{
		Content: "",
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrEmptyFile)
}

func TestCreateTasksFromFile_Execute_MissingTitle(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	mockGit := &testutil.MockGit{CurrentBranchName: testutil.StringPtr("main")}
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCreateTasksFromFile(repo, mockGit, configLoader, clock, nil)

	content := `---
labels: [bug]
---
No title here.`

	// Execute
	_, err := uc.Execute(context.Background(), CreateTasksFromFileInput{
		Content: content,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrEmptyTitle)
}

func TestCreateTasksFromFile_Execute_InvalidParentRef(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	mockGit := &testutil.MockGit{CurrentBranchName: testutil.StringPtr("main")}
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCreateTasksFromFile(repo, mockGit, configLoader, clock, nil)

	content := `---
title: Task
parent: invalid
---
Description.`

	// Execute
	_, err := uc.Execute(context.Background(), CreateTasksFromFileInput{
		Content: content,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrInvalidParentRef)
}

func TestCreateTasksFromFile_Execute_ParentNotFound(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	mockGit := &testutil.MockGit{CurrentBranchName: testutil.StringPtr("main")}
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCreateTasksFromFile(repo, mockGit, configLoader, clock, nil)

	content := `---
title: Task
parent: 999
---
Parent doesn't exist.`

	// Execute
	_, err := uc.Execute(context.Background(), CreateTasksFromFileInput{
		Content: content,
	})

	// Assert
	assert.ErrorIs(t, err, domain.ErrParentNotFound)
}

func TestCreateTasksFromFile_Execute_DryRun_AbsoluteParentNotVerified(t *testing.T) {
	// Setup - use nil dependencies to prove dry-run doesn't need them
	uc := NewCreateTasksFromFile(nil, nil, nil, nil, nil)

	content := `---
title: Task
parent: 999
---
Parent doesn't exist but dry-run doesn't verify.`

	// Execute with dry-run
	out, err := uc.Execute(context.Background(), CreateTasksFromFileInput{
		Content: content,
		DryRun:  true,
	})

	// Assert - dry-run does NOT verify absolute parent references (no I/O)
	// The parent reference is simply accepted as-is for preview purposes
	require.NoError(t, err)
	require.Len(t, out.Tasks, 1)
	require.NotNil(t, out.Tasks[0].ParentID)
	assert.Equal(t, 999, *out.Tasks[0].ParentID)
}

func TestCreateTasksFromFile_Execute_DryRun_NoIODependencies(t *testing.T) {
	// Setup - use nil for ALL dependencies to prove dry-run works without any I/O
	uc := NewCreateTasksFromFile(nil, nil, nil, nil, nil)

	content := `---
title: Root Task
labels: [backend, urgent]
---
Root description.

---
title: Child Task
parent: 1
---
Child description.`

	// Execute with dry-run - should succeed even with nil dependencies
	out, err := uc.Execute(context.Background(), CreateTasksFromFileInput{
		Content: content,
		DryRun:  true,
	})

	// Assert
	require.NoError(t, err)
	require.Len(t, out.Tasks, 2)

	// Verify task details are correctly parsed
	assert.Equal(t, 1, out.Tasks[0].ID)
	assert.Equal(t, "Root Task", out.Tasks[0].Title)
	assert.Equal(t, []string{"backend", "urgent"}, out.Tasks[0].Labels)
	assert.Nil(t, out.Tasks[0].ParentID)

	assert.Equal(t, 2, out.Tasks[1].ID)
	assert.Equal(t, "Child Task", out.Tasks[1].Title)
	require.NotNil(t, out.Tasks[1].ParentID)
	assert.Equal(t, 1, *out.Tasks[1].ParentID)
}

func TestCreateTasksFromFile_Execute_SaveError(t *testing.T) {
	// Setup
	repo := testutil.NewMockTaskRepository()
	repo.SaveErr = assert.AnError
	mockGit := &testutil.MockGit{CurrentBranchName: testutil.StringPtr("main")}
	configLoader := testutil.NewMockConfigLoader()
	clock := &testutil.MockClock{NowTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	uc := NewCreateTasksFromFile(repo, mockGit, configLoader, clock, nil)

	content := `---
title: Test Task
---
Description.`

	// Execute
	_, err := uc.Execute(context.Background(), CreateTasksFromFileInput{
		Content: content,
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save task")
}
