package tui

import (
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestViewDetailPanel_ShowsLabels(t *testing.T) {
	task := &domain.Task{
		ID:          1,
		Title:       "Task with labels",
		Description: "Description",
		Status:      domain.StatusTodo,
		Created:     time.Now(),
		Labels:      []string{"bug", "ui"},
	}

	styles := DefaultStyles()
	delegate := newTaskDelegate(styles)
	taskList := list.New([]list.Item{}, delegate, 0, 0)
	taskList.SetItems([]list.Item{taskItem{task: task}})

	m := &Model{
		width:    120,
		height:   20,
		tasks:    []*domain.Task{task},
		styles:   styles,
		taskList: taskList,
	}
	// Initialize viewport for the test
	m.updateDetailPanelViewport()

	result := m.viewDetailPanel()

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "[bug]")
	assert.Contains(t, result, "[ui]")
	assert.Contains(t, result, "Labels")
}
