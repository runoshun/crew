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
		name     string
		height   int
		task     *domain.Task
		comments []domain.Comment
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
			comments: nil,
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
			comments: nil,
		},
		{
			name:   "content fits exactly in height",
			height: 12,
			task: &domain.Task{
				ID:          1,
				Title:       "Task with exact content",
				Description: "Short description",
				Status:      domain.StatusForReview,
				Agent:       "claude",
				Created:     time.Now(),
				Started:     time.Now(),
			},
			comments: nil,
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
			comments: nil,
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
			// Initialize viewport for the test
			m.updateDetailPanelViewport()

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

			// Note: Since we now use viewport for scrolling, long content is handled
			// by the viewport's built-in scrolling rather than explicit truncation markers.
			// The content can be scrolled to view all of it.
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
	// Initialize viewport for the test
	m.updateDetailPanelViewport()

	result := m.viewDetailPanel()

	assert.NotEmpty(t, result)
	// Height should be respected even with wrapped title
	renderedHeight := lipgloss.Height(result)
	assert.LessOrEqual(t, renderedHeight, 13, "Height should not exceed panel height even with title wrapping")
}

func TestViewFooter_Truncation(t *testing.T) {
	tests := []struct {
		name        string
		width       int
		expectedHas string
		description string
	}{
		{
			name:        "Wide screen - full footer displayed",
			width:       120,
			expectedHas: "default",
			description: "Footer should display full content on wide screen",
		},
		{
			name:        "Narrow screen - footer truncated with ellipsis",
			width:       60,
			expectedHas: "...",
			description: "Footer should be truncated with ... when width is limited",
		},
		{
			name:        "Very narrow screen - minimal footer with ellipsis",
			width:       40,
			expectedHas: "...",
			description: "Footer should show ... when space is very limited",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			styles := DefaultStyles()
			delegate := newTaskDelegate(styles)
			taskList := list.New([]list.Item{}, delegate, tt.width, 20)

			// Create a simple task to populate the list
			task := &domain.Task{
				ID:      1,
				Title:   "Test task",
				Status:  domain.StatusTodo,
				Created: time.Now(),
			}
			taskList.SetItems([]list.Item{taskItem{task: task}})

			m := &Model{
				width:    tt.width,
				height:   20,
				tasks:    []*domain.Task{task},
				styles:   styles,
				taskList: taskList,
				mode:     ModeNormal,
			}

			result := m.viewFooter()

			// Verify the result is not empty
			assert.NotEmpty(t, result, "viewFooter should return non-empty string")

			// Check for expected content
			assert.Contains(t, result, tt.expectedHas, tt.description)
		})
	}
}

func TestFillViewportLines(t *testing.T) {
	bg := Colors.Background
	width := 20
	ds := dialogStyles{
		width: width,
		bg:    bg,
		line:  lipgloss.NewStyle().Background(bg).Width(width),
	}

	tests := []struct {
		name            string
		content         string
		height          int
		expectedLines   int
		hasEmptyLines   bool
		requirePadStyle bool
		description     string
	}{
		{
			name:          "content fits within height",
			content:       "line1\nline2\nline3",
			height:        5,
			expectedLines: 5,
			hasEmptyLines: true,
			description:   "Should pad remaining height with empty lines",
		},
		{
			name:          "content equals height",
			content:       "line1\nline2\nline3",
			height:        3,
			expectedLines: 3,
			hasEmptyLines: false,
			description:   "Should fit exactly without extra empty lines",
		},
		{
			name:          "content exceeds height",
			content:       "line1\nline2\nline3\nline4\nline5",
			height:        3,
			expectedLines: 3,
			hasEmptyLines: false,
			description:   "Should truncate to height",
		},
		{
			name:          "empty content",
			content:       "",
			height:        3,
			expectedLines: 3,
			hasEmptyLines: true,
			description:   "Empty content should fill with empty lines",
		},
		{
			name: "ANSI colored content",
			// Simulate diff-like colored output: green for additions, red for deletions
			content: lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1")).Render("+added") + "\n" +
				lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8")).Render("-removed") + "\n" +
				"plain",
			height:        5,
			expectedLines: 5,
			hasEmptyLines: true,
			description:   "ANSI colored lines should be padded correctly using lipgloss.Width",
		},
		{
			name: "ANSI reset within line",
			content: lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1")).Render("+added") +
				" " + "\x1b[0m" + "tail",
			height:          1,
			expectedLines:   1,
			hasEmptyLines:   false,
			requirePadStyle: true,
			description:     "ANSI reset should not strip background on pad",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ds.fillViewportLines(tt.content, tt.height)

			// Count resulting lines
			resultLines := strings.Split(result, "\n")
			assert.Equal(t, tt.expectedLines, len(resultLines), tt.description)

			// Verify each line has the expected width (background applied)
			for _, line := range resultLines {
				lineWidth := lipgloss.Width(line)
				assert.Equal(t, width, lineWidth, "Each line should have full width")
			}

			if tt.requirePadStyle {
				padSample := lipgloss.NewStyle().Background(bg).Render(" ")
				assert.Contains(t, resultLines[0], padSample, "Line should include background-filled pad even with ANSI reset")
			}
			if tt.hasEmptyLines {
				emptyLine := ds.emptyLine()
				assert.Contains(t, result, emptyLine, "Result should contain empty lines")
			}
		})
	}
}

func TestViewFooter_MinimalWidth(t *testing.T) {
	// Test the edge case where maxContentWidth <= 3
	// This should show only "..." for the content
	styles := DefaultStyles()
	delegate := newTaskDelegate(styles)

	// Use a very narrow width to force minimal content display
	width := 30
	taskList := list.New([]list.Item{}, delegate, width, 20)

	task := &domain.Task{
		ID:      1,
		Title:   "Test task",
		Status:  domain.StatusTodo,
		Created: time.Now(),
	}
	taskList.SetItems([]list.Item{taskItem{task: task}})

	m := &Model{
		width:    width,
		height:   20,
		tasks:    []*domain.Task{task},
		styles:   styles,
		taskList: taskList,
		mode:     ModeNormal,
	}

	result := m.viewFooter()

	// With very limited space, footer should still render properly
	assert.NotEmpty(t, result, "viewFooter should return non-empty string even with minimal width")
	// When space is very limited, the content should be truncated (possibly to just "...")
	assert.Contains(t, result, "...", "Footer should contain ellipsis when truncated")
}
