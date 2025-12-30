package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/runoshun/git-crew/v2/internal/domain"
)

// View renders the TUI.
func (m *Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var content string
	switch m.mode {
	case ModeHelp:
		content = m.viewHelp()
	case ModeDetail:
		content = m.viewDetail()
	case ModeNormal, ModeFilter, ModeConfirm, ModeInputTitle, ModeInputDesc, ModeStart:
		content = m.viewMain()
	}

	return m.styles.App.Render(content)
}

// viewMain renders the main task list view.
func (m *Model) viewMain() string {
	var b strings.Builder

	// Header
	b.WriteString(m.viewHeader())
	b.WriteString("\n")

	// Error message (if any)
	if m.err != nil {
		b.WriteString(m.styles.ErrorMsg.Render("Error: "+m.err.Error()) + "\n\n")
	}

	// Filter input (if in filter mode)
	if m.mode == ModeFilter {
		b.WriteString(m.styles.InputPrompt.Render("Filter: "))
		b.WriteString(m.filterInput.View())
		b.WriteString("\n\n")
	} else if m.filterInput.Value() != "" {
		b.WriteString(m.styles.Footer.Render("Filtered: "+m.filterInput.Value()) + "\n\n")
	}

	// Task list
	b.WriteString(m.viewTaskList())

	// Dialogs/overlays
	switch m.mode {
	case ModeNormal, ModeFilter, ModeHelp, ModeDetail:
		// No overlay for these modes
	case ModeConfirm:
		b.WriteString("\n")
		b.WriteString(m.viewConfirmDialog())
	case ModeInputTitle:
		b.WriteString("\n")
		b.WriteString(m.viewTitleInput())
	case ModeInputDesc:
		b.WriteString("\n")
		b.WriteString(m.viewDescInput())
	case ModeStart:
		b.WriteString("\n")
		b.WriteString(m.viewAgentPicker())
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(m.viewFooter())

	return b.String()
}

// viewHeader renders the header.
func (m *Model) viewHeader() string {
	title := m.styles.HeaderText.Render("git-crew")
	taskCount := fmt.Sprintf(" (%d tasks)", len(m.visibleTasks()))
	return m.styles.Header.Render(title + m.styles.Footer.Render(taskCount))
}

// viewTaskList renders the task list.
func (m *Model) viewTaskList() string {
	tasks := m.visibleTasks()
	if len(tasks) == 0 {
		return m.styles.Footer.Render("No tasks. Press 'n' to create one.")
	}

	var b strings.Builder
	for i, task := range tasks {
		b.WriteString(m.renderTaskItem(task, i == m.cursor))
		b.WriteString("\n")
	}

	return m.styles.TaskList.Render(b.String())
}

// renderTaskItem renders a single task item.
func (m *Model) renderTaskItem(task *domain.Task, selected bool) string {
	// Cursor indicator (always same width, different style when selected)
	var cursor string
	if selected {
		cursor = m.styles.CursorSelected.Render(">")
	} else {
		cursor = m.styles.CursorNormal.Render(" ")
	}

	// Task ID
	id := fmt.Sprintf("#%-3d", task.ID)

	// Status with icon
	statusText := fmt.Sprintf("[%s %s]", StatusIcon(task.Status), task.Status)

	// Agent (if running)
	agent := ""
	if task.Agent != "" {
		agent = " @" + task.Agent
	}

	// Title
	title := task.Title

	// Apply styles based on selection
	if selected {
		id = m.styles.TaskIDSelected.Render(id)
		statusStyle := m.styles.StatusStyleSelected(task.Status)
		statusText = statusStyle.Render(statusText)
		title = m.styles.TaskTitleSelected.Render(title)
		if agent != "" {
			agent = m.styles.TaskAgentSelected.Render(agent)
		}
	} else {
		id = m.styles.TaskID.Render(id)
		statusStyle := m.styles.StatusStyle(task.Status)
		statusText = statusStyle.Render(statusText)
		title = m.styles.TaskTitle.Render(title)
		if agent != "" {
			agent = m.styles.TaskAgent.Render(agent)
		}
	}

	// Assemble (no indentation shift)
	return fmt.Sprintf("%s %s %s %s%s", cursor, id, statusText, title, agent)
}

// viewConfirmDialog renders the confirmation dialog.
func (m *Model) viewConfirmDialog() string {
	var action, target string
	switch m.confirmAction {
	case ConfirmNone:
		action = ""
		target = ""
	case ConfirmDelete:
		action = "Delete"
		target = fmt.Sprintf("task #%d", m.confirmTaskID)
	case ConfirmClose:
		action = "Close"
		target = fmt.Sprintf("task #%d", m.confirmTaskID)
	case ConfirmStop:
		action = "Stop"
		target = fmt.Sprintf("session for task #%d", m.confirmTaskID)
	case ConfirmMerge:
		action = "Merge"
		target = fmt.Sprintf("task #%d", m.confirmTaskID)
	}

	title := m.styles.DialogTitle.Render(action + "?")
	prompt := m.styles.DialogPrompt.Render(fmt.Sprintf("Are you sure you want to %s %s?", strings.ToLower(action), target))
	hint := m.styles.Footer.Render("[y] confirm  [n/esc] cancel")

	content := lipgloss.JoinVertical(lipgloss.Left, title, prompt, "", hint)
	return m.styles.Dialog.Render(content)
}

// viewTitleInput renders the title input dialog.
func (m *Model) viewTitleInput() string {
	title := m.styles.DialogTitle.Render("New Task")
	label := m.styles.InputPrompt.Render("Title: ")
	input := m.titleInput.View()
	hint := m.styles.Footer.Render("[enter] next  [esc] cancel")

	content := lipgloss.JoinVertical(lipgloss.Left, title, label+input, "", hint)
	return m.styles.Dialog.Render(content)
}

// viewDescInput renders the description input dialog.
func (m *Model) viewDescInput() string {
	title := m.styles.DialogTitle.Render("New Task")
	titleLabel := m.styles.Footer.Render("Title: " + m.titleInput.Value())
	label := m.styles.InputPrompt.Render("Description: ")
	input := m.descInput.View()
	hint := m.styles.Footer.Render("[enter] create  [esc] back")

	content := lipgloss.JoinVertical(lipgloss.Left, title, titleLabel, label+input, "", hint)
	return m.styles.Dialog.Render(content)
}

// viewAgentPicker renders the agent picker.
func (m *Model) viewAgentPicker() string {
	task := m.SelectedTask()
	if task == nil {
		return ""
	}

	title := m.styles.DialogTitle.Render(fmt.Sprintf("Start Task #%d", task.ID))

	var agentList strings.Builder
	for i, agent := range m.agents {
		cursor := "  "
		if i == m.agentCursor {
			cursor = "> "
		}

		line := fmt.Sprintf("%s%s", cursor, agent)
		if i == m.agentCursor {
			line = m.styles.TaskSelected.Render(line)
		}
		agentList.WriteString(line + "\n")
	}

	hint := m.styles.Footer.Render("[↑/↓] select  [enter] start  [esc] cancel")

	content := lipgloss.JoinVertical(lipgloss.Left, title, "", agentList.String(), hint)
	return m.styles.Dialog.Render(content)
}

// viewFooter renders the footer with key hints.
func (m *Model) viewFooter() string {
	var hints []string

	switch m.mode {
	case ModeNormal:
		hints = []string{
			m.styles.FooterKey.Render("↑↓") + " navigate",
			m.styles.FooterKey.Render("enter") + " action",
			m.styles.FooterKey.Render("s") + " start",
			m.styles.FooterKey.Render("n") + " new",
			m.styles.FooterKey.Render("?") + " help",
			m.styles.FooterKey.Render("q") + " quit",
		}
	case ModeFilter:
		hints = []string{
			m.styles.FooterKey.Render("enter") + " apply",
			m.styles.FooterKey.Render("esc") + " cancel",
		}
	case ModeConfirm, ModeInputTitle, ModeInputDesc, ModeStart, ModeHelp, ModeDetail:
		// Hints are shown in the dialogs/views themselves
	}

	return m.styles.Footer.Render(strings.Join(hints, "  "))
}

// viewHelp renders the help view.
func (m *Model) viewHelp() string {
	title := m.styles.Header.Render("Help")

	sections := []struct {
		name  string
		binds []struct {
			key  string
			desc string
		}
	}{
		{
			name: "Navigation",
			binds: []struct {
				key  string
				desc string
			}{
				{"↑/k", "Move up"},
				{"↓/j", "Move down"},
				{"enter", "Smart action (start/attach based on status)"},
				{"/", "Filter tasks"},
				{"v", "View task details"},
			},
		},
		{
			name: "Session Control",
			binds: []struct {
				key  string
				desc string
			}{
				{"s", "Start task with agent"},
				{"S", "Stop running session"},
				{"a", "Attach to session"},
			},
		},
		{
			name: "Task Management",
			binds: []struct {
				key  string
				desc string
			}{
				{"n", "Create new task"},
				{"y", "Copy task"},
				{"d", "Delete task"},
				{"m", "Merge task (in_review only)"},
				{"c", "Close task"},
			},
		},
		{
			name: "General",
			binds: []struct {
				key  string
				desc string
			}{
				{"r", "Refresh task list"},
				{"?", "Toggle help"},
				{"q", "Quit"},
				{"esc", "Cancel/back"},
			},
		},
	}

	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")

	for _, section := range sections {
		b.WriteString(m.styles.TaskTitle.Bold(true).Render(section.name))
		b.WriteString("\n")
		for _, bind := range section.binds {
			key := m.styles.HelpKey.Render(fmt.Sprintf("%-8s", bind.key))
			desc := m.styles.HelpDesc.Render(bind.desc)
			b.WriteString(fmt.Sprintf("  %s %s\n", key, desc))
		}
		b.WriteString("\n")
	}

	b.WriteString(m.styles.Footer.Render("Press ? or esc to close"))

	return b.String()
}

// viewDetail renders the task detail view.
func (m *Model) viewDetail() string {
	task := m.SelectedTask()
	if task == nil {
		return "No task selected"
	}

	var b strings.Builder

	// Title
	title := fmt.Sprintf("Task #%d: %s", task.ID, task.Title)
	b.WriteString(m.styles.DetailTitle.Render(title))
	b.WriteString("\n\n")

	// Status
	statusStyle := m.styles.StatusStyle(task.Status)
	b.WriteString(m.styles.DetailLabel.Render("Status:"))
	b.WriteString(statusStyle.Render(string(task.Status)))
	b.WriteString("\n")

	// Agent
	if task.Agent != "" {
		b.WriteString(m.styles.DetailLabel.Render("Agent:"))
		b.WriteString(m.styles.DetailValue.Render(task.Agent))
		b.WriteString("\n")
	}

	// Session
	if task.Session != "" {
		b.WriteString(m.styles.DetailLabel.Render("Session:"))
		b.WriteString(m.styles.DetailValue.Render(task.Session))
		b.WriteString("\n")
	}

	// Created
	b.WriteString(m.styles.DetailLabel.Render("Created:"))
	b.WriteString(m.styles.DetailValue.Render(task.Created.Format("2006-01-02 15:04")))
	b.WriteString("\n")

	// Started
	if !task.Started.IsZero() {
		b.WriteString(m.styles.DetailLabel.Render("Started:"))
		b.WriteString(m.styles.DetailValue.Render(task.Started.Format("2006-01-02 15:04")))
		b.WriteString("\n")
	}

	// Description
	if task.Description != "" {
		b.WriteString("\n")
		b.WriteString(m.styles.DetailLabel.Render("Description:"))
		b.WriteString("\n")
		b.WriteString(m.styles.DetailDesc.Render(task.Description))
		b.WriteString("\n")
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(m.styles.Footer.Render("[v/esc] back  [q] quit"))

	return b.String()
}
