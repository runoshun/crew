package tui

import (
	"bytes"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestDelegate_Render_ShowsLabels(t *testing.T) {
	task := &domain.Task{
		ID:      1,
		Title:   "Task with labels",
		Status:  domain.StatusTodo,
		Created: time.Now(),
		Labels:  []string{"bug", "ui"},
	}

	styles := DefaultStyles()
	delegate := newTaskDelegate(styles)

	// Create a dummy model
	l := list.New(nil, delegate, 100, 20)

	var buf bytes.Buffer
	delegate.Render(&buf, l, 0, taskItem{task: task})

	output := buf.String()
	assert.Contains(t, output, "[bug]")
	assert.Contains(t, output, "[ui]")
}
