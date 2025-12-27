package cli

import (
	"bytes"
	"testing"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
)

// mockClock is a test double for domain.Clock.
type mockClock struct {
	now time.Time
}

func (m *mockClock) Now() time.Time {
	return m.now
}

func TestPrintTaskList_Empty(t *testing.T) {
	var buf bytes.Buffer
	clock := &mockClock{now: time.Now()}

	printTaskList(&buf, []*domain.Task{}, clock)

	// Should only have header
	expected := "ID   PARENT   STATUS   AGENT   LABELS   TITLE\n"
	assert.Equal(t, expected, buf.String())
}

func TestPrintTaskList_SingleTask(t *testing.T) {
	var buf bytes.Buffer
	clock := &mockClock{now: time.Now()}

	tasks := []*domain.Task{
		{
			ID:     1,
			Title:  "Test task",
			Status: domain.StatusTodo,
		},
	}

	printTaskList(&buf, tasks, clock)

	output := buf.String()
	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "PARENT")
	assert.Contains(t, output, "STATUS")
	assert.Contains(t, output, "AGENT")
	assert.Contains(t, output, "LABELS")
	assert.Contains(t, output, "TITLE")
	assert.Contains(t, output, "1")
	assert.Contains(t, output, "-") // PARENT is nil
	assert.Contains(t, output, "todo")
	assert.Contains(t, output, "Test task")
}

func TestPrintTaskList_WithParent(t *testing.T) {
	var buf bytes.Buffer
	clock := &mockClock{now: time.Now()}

	parentID := 1
	tasks := []*domain.Task{
		{
			ID:       2,
			ParentID: &parentID,
			Title:    "Child task",
			Status:   domain.StatusTodo,
		},
	}

	printTaskList(&buf, tasks, clock)

	output := buf.String()
	assert.Contains(t, output, "2")
	assert.Contains(t, output, "1") // Parent ID
	assert.Contains(t, output, "Child task")
}

func TestPrintTaskList_WithAgent(t *testing.T) {
	var buf bytes.Buffer
	clock := &mockClock{now: time.Now()}

	tasks := []*domain.Task{
		{
			ID:     1,
			Title:  "Task with agent",
			Status: domain.StatusInProgress,
			Agent:  "claude",
		},
	}

	printTaskList(&buf, tasks, clock)

	output := buf.String()
	assert.Contains(t, output, "claude")
}

func TestPrintTaskList_WithLabels(t *testing.T) {
	var buf bytes.Buffer
	clock := &mockClock{now: time.Now()}

	tasks := []*domain.Task{
		{
			ID:     1,
			Title:  "Task with labels",
			Status: domain.StatusTodo,
			Labels: []string{"bug", "urgent"},
		},
	}

	printTaskList(&buf, tasks, clock)

	output := buf.String()
	assert.Contains(t, output, "[bug,urgent]")
}

func TestPrintTaskList_InProgressWithElapsed(t *testing.T) {
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	started := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC) // 1 hour ago

	var buf bytes.Buffer
	clock := &mockClock{now: now}

	tasks := []*domain.Task{
		{
			ID:      1,
			Title:   "In progress task",
			Status:  domain.StatusInProgress,
			Started: started,
		},
	}

	printTaskList(&buf, tasks, clock)

	output := buf.String()
	assert.Contains(t, output, "in_progress (1h)")
}

func TestPrintTaskList_MultipleTasks(t *testing.T) {
	var buf bytes.Buffer
	clock := &mockClock{now: time.Now()}

	parentID := 1
	tasks := []*domain.Task{
		{
			ID:     1,
			Title:  "Parent task",
			Status: domain.StatusInProgress,
			Agent:  "claude",
			Labels: []string{"feature"},
		},
		{
			ID:       2,
			ParentID: &parentID,
			Title:    "Child task",
			Status:   domain.StatusTodo,
		},
		{
			ID:     3,
			Title:  "Done task",
			Status: domain.StatusDone,
		},
	}

	printTaskList(&buf, tasks, clock)

	output := buf.String()
	// Verify all tasks are present
	assert.Contains(t, output, "Parent task")
	assert.Contains(t, output, "Child task")
	assert.Contains(t, output, "Done task")
	// Verify statuses
	assert.Contains(t, output, "in_progress")
	assert.Contains(t, output, "todo")
	assert.Contains(t, output, "done")
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"zero", 0, "0s"},
		{"seconds", 30 * time.Second, "30s"},
		{"just under minute", 59 * time.Second, "59s"},
		{"one minute", 1 * time.Minute, "1m"},
		{"minutes", 5 * time.Minute, "5m"},
		{"just under hour", 59 * time.Minute, "59m"},
		{"one hour", 1 * time.Hour, "1h"},
		{"hours", 5 * time.Hour, "5h"},
		{"just under day", 23 * time.Hour, "23h"},
		{"one day", 24 * time.Hour, "1d"},
		{"days", 3 * 24 * time.Hour, "3d"},
		{"week", 7 * 24 * time.Hour, "7d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}
