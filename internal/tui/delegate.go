package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"
	"github.com/runoshun/git-crew/v2/internal/domain"
)

type taskItem struct {
	task *domain.Task
}

func (t taskItem) FilterValue() string {
	return t.task.Title
}

type taskDelegate struct {
	styles Styles
}

func newTaskDelegate(styles Styles) taskDelegate {
	return taskDelegate{styles: styles}
}

func (d taskDelegate) Height() int {
	return 2
}

func (d taskDelegate) Spacing() int {
	return 1
}

func (d taskDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d taskDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ti, ok := item.(taskItem)
	if !ok {
		return
	}
	task := ti.task
	selected := index == m.Index()

	indicatorChar := " "
	if selected {
		indicatorChar = ">"
	}

	idStr := fmt.Sprintf("%3d", task.ID)
	statusIcon := StatusIcon(task.Status)
	statusText := fmt.Sprintf("%-5s", StatusText(task.Status))

	prefixWidth := 19
	listWidth := m.Width()
	maxTitleLen := listWidth - prefixWidth - 2
	if maxTitleLen < 10 {
		maxTitleLen = 10
	}

	title := task.Title
	if runewidth.StringWidth(title) > maxTitleLen {
		title = runewidth.Truncate(title, maxTitleLen-3, "...")
	}

	var line string
	if selected {
		indicator := d.styles.SelectionIndicator.Bold(true).Render(indicatorChar)
		idPart := d.styles.TaskID.Bold(true).Render(idStr)
		iconPart := d.styles.StatusStyle(task.Status).Bold(true).Render(statusIcon)
		textPart := d.styles.StatusStyle(task.Status).Bold(true).Render(statusText)
		titlePart := d.styles.TaskTitle.Bold(true).Render(title)
		line = "  " + indicator + " " + idPart + "  " + iconPart + " " + textPart + "  " + titlePart
	} else {
		indicator := d.styles.SelectionIndicator.Render(indicatorChar)
		idPart := d.styles.TaskID.Render(idStr)
		iconPart := d.styles.StatusStyle(task.Status).Render(statusIcon)
		textPart := d.styles.StatusStyle(task.Status).Render(statusText)
		titlePart := d.styles.TaskTitle.Render(title)
		line = "  " + indicator + " " + idPart + "  " + iconPart + " " + textPart + "  " + titlePart
	}
	lineWidth := runewidth.StringWidth(line)
	if lineWidth < listWidth {
		line += fmt.Sprintf("%*s", listWidth-lineWidth, "")
	}
	_, _ = fmt.Fprintln(w, line)

	descLine := "                   "
	if task.Description != "" {
		desc := task.Description
		maxDescLen := listWidth - prefixWidth - 2
		if maxDescLen < 10 {
			maxDescLen = 10
		}
		if runewidth.StringWidth(desc) > maxDescLen {
			desc = runewidth.Truncate(desc, maxDescLen-3, "...")
		}
		descLine += desc
	}
	descLineWidth := runewidth.StringWidth(descLine)
	if descLineWidth < listWidth {
		descLine += fmt.Sprintf("%*s", listWidth-descLineWidth, "")
	}
	_, _ = fmt.Fprint(w, d.styles.TaskDesc.Render(descLine))
}
