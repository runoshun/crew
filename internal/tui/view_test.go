package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestViewDetailPanel_RespectsPanelHeight(t *testing.T) {
	tests := []struct {
		name        string
		height      int
		task        *domain.Task
		comments    []domain.Comment
		expectTrunc bool
	}{
		{
			name:   "short content fits within height",
			height: 20,
			task: &domain.Task{
				ID:          1,
				Title:       "Short task",
				Description: "Brief description",
				Status:      domain.StatusTodo,
				Created:     time.Now(),
			},
			comments:    nil,
			expectTrunc: false,
		},
		{
			name:   "long title and description exceeds height",
			height: 11,
			task: &domain.Task{
				ID:          1,
				Title:       "This is a very long task title that will definitely wrap to multiple lines when rendered in the detail panel because it exceeds the width",
				Description: "This is a very long description that also wraps to multiple lines. " + strings.Repeat("Lorem ipsum dolor sit amet. ", 10),
				Status:      domain.StatusInProgress,
				Agent:       "claude",
				Created:     time.Now(),
				Started:     time.Now(),
			},
			comments:    nil,
			expectTrunc: true,
		},
		{
			name:   "content fits exactly in height",
			height: 12,
			task: &domain.Task{
				ID:          1,
				Title:       "Task with exact content",
				Description: "Short description",
				Status:      domain.StatusInReview,
				Agent:       "claude",
				Created:     time.Now(),
				Started:     time.Now(),
			},
			comments:    nil,
			expectTrunc: false,
		},
		{
			name:   "minimum height handles truncation gracefully",
			height: 10,
			task: &domain.Task{
				ID:          1,
				Title:       "Task title",
				Description: "Description that may not fully fit",
				Status:      domain.StatusTodo,
				Created:     time.Now(),
			},
			comments:    nil,
			expectTrunc: false, // With minimum height, basic info should fit
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			styles := DefaultStyles()
			delegate := newTaskDelegate(styles)
			taskList := list.New([]list.Item{}, delegate, 0, 0)
			taskList.SetItems([]list.Item{taskItem{task: tt.task}})

			m := &Model{
				width:    120,
				height:   tt.height + 2, // viewDetailPanel subtracts 2
				tasks:    []*domain.Task{tt.task},
				comments: tt.comments,
				styles:   styles,
				taskList: taskList,
			}

			result := m.viewDetailPanel()

			// Verify the result is not empty
			assert.NotEmpty(t, result, "viewDetailPanel should return non-empty string")

			// Count actual rendered lines
			// The panelStyle.Render wraps with Height(), so we need to check the output
			// doesn't exceed the specified height
			renderedHeight := lipgloss.Height(result)

			// The panel height should match what we specified
			expectedHeight := tt.height
			if expectedHeight < 10 {
				expectedHeight = 10
			}

			assert.LessOrEqual(t, renderedHeight, expectedHeight,
				"Rendered panel height should not exceed specified panel height")

			// Check for truncation indicator if expected
			if tt.expectTrunc {
				assert.Contains(t, result, "...", "Long content should be truncated with ellipsis")
			}
		})
	}
}

func TestViewDetailPanel_EmptyTask(t *testing.T) {
	styles := DefaultStyles()
	delegate := newTaskDelegate(styles)
	taskList := list.New([]list.Item{}, delegate, 0, 0)

	m := &Model{
		width:    120,
		height:   20,
		tasks:    []*domain.Task{},
		styles:   styles,
		taskList: taskList,
	}

	result := m.viewDetailPanel()

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Select a task")
}

func TestDetailPanelWidth(t *testing.T) {
	tests := []struct {
		name     string
		width    int
		expected int
	}{
		{
			name:     "Narrow screen, no panel",
			width:    80,
			expected: 0,
		},
		{
			name:     "Threshold width, minimum width panel",
			width:    100, // 40% is 40, which is MinDetailPanelWidth
			expected: 40,
		},
		{
			name:     "Slightly wider screen, 40% panel",
			width:    110, // 40% is 44
			expected: 44,
		},
		{
			name:     "Wide screen, 40% panel",
			width:    200,
			expected: 80,
		},
		{
			name:     "Very wide screen, 40% panel",
			width:    300,
			expected: 120,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Model{width: tt.width}
			assert.Equal(t, tt.expected, m.detailPanelWidth())
		})
	}
}

func TestViewDetailPanel_TitleWrapping(t *testing.T) {
	// Test that long titles are properly handled
	longTitle := strings.Repeat("Very long task title ", 10)
	task := &domain.Task{
		ID:      1,
		Title:   longTitle,
		Status:  domain.StatusTodo,
		Created: time.Now(),
	}

	styles := DefaultStyles()
	delegate := newTaskDelegate(styles)
	taskList := list.New([]list.Item{}, delegate, 0, 0)
	taskList.SetItems([]list.Item{taskItem{task: task}})

	m := &Model{
		width:    120,
		height:   15,
		tasks:    []*domain.Task{task},
		styles:   styles,
		taskList: taskList,
	}

	result := m.viewDetailPanel()

	assert.NotEmpty(t, result)
	// Height should be respected even with wrapped title
	renderedHeight := lipgloss.Height(result)
	assert.LessOrEqual(t, renderedHeight, 13, "Height should not exceed panel height even with title wrapping")
}
