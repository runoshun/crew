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

	// Task list (grouped)
	b.WriteString(m.viewTaskList())

	// Pagination info
	// Pagination info is now shown in header, not at bottom

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

// viewHeader renders the header with "Tasks" and task count.
func (m *Model) viewHeader() string {
	// Left: "Tasks"
	title := m.styles.HeaderText.Render("Tasks")

	// Right: "showing X of Y tasks" (muted style)
	visibleCount := len(m.visibleTasks())
	totalCount := len(m.tasks)
	countText := fmt.Sprintf("showing %d of %d tasks", visibleCount, totalCount)
	// Use muted text style for task count (no border/padding)
	rightText := lipgloss.NewStyle().Foreground(Colors.Muted).Render(countText)

	// Calculate spacing to right-align
	headerWidth := m.width - 6 // padding
	if headerWidth < 40 {
		headerWidth = 40
	}
	leftLen := lipgloss.Width(title)
	rightLen := lipgloss.Width(rightText)
	spacing := headerWidth - leftLen - rightLen
	if spacing < 1 {
		spacing = 1
	}

	return m.styles.Header.Render(title + strings.Repeat(" ", spacing) + rightText)
}

// viewTaskList renders the flat task list (no group headers).
func (m *Model) viewTaskList() string {
	tasks := m.visibleTasks()
	if len(tasks) == 0 {
		return m.viewEmptyState()
	}

	var b strings.Builder

	// Calculate row width for full-width background highlight
	rowWidth := m.width - 6 // Account for app padding
	if rowWidth < 40 {
		rowWidth = 40
	}

	for i, task := range tasks {
		selected := i == m.cursor
		line := m.renderTaskItem(task, selected)

		if selected {
			// Apply subtle background for full row width
			b.WriteString(m.styles.TaskSelected.Width(rowWidth).Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")

		// Show description line for in_progress tasks
		if task.Status == domain.StatusInProgress && task.Description != "" {
			// Truncate description if too long
			desc := task.Description
			maxLen := 50
			if len(desc) > maxLen {
				desc = desc[:maxLen-3] + "..."
			}
			// Indent to align with title (indicator + id + status = ~15 chars)
			descLine := m.styles.TaskDesc.Render("  " + desc)
			b.WriteString(descLine)
			b.WriteString("\n")
		}
	}

	return m.styles.TaskList.Render(b.String())
}

// viewEmptyState renders a friendly empty state message.
func (m *Model) viewEmptyState() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(m.styles.Footer.Render("  No tasks yet\n\n"))
	b.WriteString(m.styles.Footer.Render("  Press "))
	b.WriteString(m.styles.FooterKey.Render("n"))
	b.WriteString(m.styles.Footer.Render(" to create your first task"))
	b.WriteString("\n")
	return b.String()
}

// renderTaskItem renders a single task item.
// Format: "> 17 ✔ Done  Overhaul TUI design" (indicator shown only when selected)
func (m *Model) renderTaskItem(task *domain.Task, selected bool) string {
	// Indicator: ">" for selected, space for normal
	var indicator string
	if selected {
		indicator = m.styles.CursorSelected.Render(">")
	} else {
		indicator = " "
	}

	// Task ID with fixed width
	idStr := fmt.Sprintf("%2d", task.ID)

	// Status: icon + text (e.g., "✔ Done", "➜ InPrg")
	statusIcon := StatusIcon(task.Status)
	statusText := StatusText(task.Status)
	statusFull := fmt.Sprintf("%s %-5s", statusIcon, statusText)

	var idPart, statusPart, titlePart string

	if selected {
		idPart = m.styles.TaskIDSelected.Render(idStr)
		statusPart = m.styles.StatusStyleSelected(task.Status).Render(statusFull)
		titlePart = m.styles.TaskTitleSelected.Render(task.Title)
	} else {
		idPart = m.styles.TaskID.Render(idStr)
		statusPart = m.styles.StatusStyle(task.Status).Render(statusFull)
		titlePart = m.styles.TaskTitle.Render(task.Title)
	}

	return fmt.Sprintf("%s %s %s  %s", indicator, idPart, statusPart, titlePart)
}

// viewConfirmDialog renders the confirmation dialog.
func (m *Model) viewConfirmDialog() string {
	var action, target string
	var color lipgloss.Color

	switch m.confirmAction {
	case ConfirmNone:
		return ""
	case ConfirmDelete:
		action = "Delete"
		target = fmt.Sprintf("task #%d", m.confirmTaskID)
		color = Colors.Error
	case ConfirmClose:
		action = "Close"
		target = fmt.Sprintf("task #%d", m.confirmTaskID)
		color = Colors.Closed
	case ConfirmStop:
		action = "Stop"
		target = fmt.Sprintf("session for task #%d", m.confirmTaskID)
		color = Colors.Error
	case ConfirmMerge:
		action = "Merge"
		target = fmt.Sprintf("task #%d", m.confirmTaskID)
		color = Colors.Done
	}

	// Styles for the dialog
	titleStyle := m.styles.DialogTitle.Foreground(color)
	promptStyle := m.styles.DialogPrompt
	keyStyle := m.styles.HelpKey

	title := titleStyle.Render(fmt.Sprintf("%s %s?", action, target))
	prompt := promptStyle.Render("This action cannot be undone.")

	// Buttons
	yesBtn := keyStyle.Render("[ y ] Confirm")
	noBtn := m.styles.Footer.Render("[ n ] Cancel")
	buttons := lipgloss.JoinHorizontal(lipgloss.Left, yesBtn, "  ", noBtn)

	// Box content
	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		prompt,
		"",
		buttons,
	)

	return m.styles.Dialog.BorderForeground(color).Render(content)
}

// viewTitleInput renders the title input dialog.
func (m *Model) viewTitleInput() string {
	title := m.styles.DialogTitle.Render("◆ New Task")
	stepInfo := m.styles.Footer.Render("Step 1 of 2")
	label := m.styles.InputPrompt.Render("Title")
	input := m.titleInput.View()
	hint := m.styles.FooterKey.Render("enter") + m.styles.Footer.Render(" next  ") +
		m.styles.FooterKey.Render("esc") + m.styles.Footer.Render(" cancel")

	content := lipgloss.JoinVertical(lipgloss.Left, title, stepInfo, "", label, input, "", hint)
	return m.styles.Dialog.Render(content)
}

// viewDescInput renders the description input dialog.
func (m *Model) viewDescInput() string {
	title := m.styles.DialogTitle.Render("◆ New Task")
	stepInfo := m.styles.Footer.Render("Step 2 of 2")
	titleLabel := m.styles.Footer.Render("Title: ") + m.styles.TaskTitle.Render(m.titleInput.Value())
	label := m.styles.InputPrompt.Render("Description (optional)")
	input := m.descInput.View()
	hint := m.styles.FooterKey.Render("enter") + m.styles.Footer.Render(" create  ") +
		m.styles.FooterKey.Render("esc") + m.styles.Footer.Render(" back")

	content := lipgloss.JoinVertical(lipgloss.Left, title, stepInfo, "", titleLabel, "", label, input, "", hint)
	return m.styles.Dialog.Render(content)
}

// viewAgentPicker renders the agent picker.
func (m *Model) viewAgentPicker() string {
	task := m.SelectedTask()
	if task == nil {
		return ""
	}

	title := m.styles.DialogTitle.Render(fmt.Sprintf("▶ Start Task #%d", task.ID))
	taskTitle := m.styles.Footer.Render(task.Title)
	selectLabel := m.styles.InputPrompt.Render("Select agent")

	var agentList strings.Builder
	allAgents := m.allAgents()
	cursor := 0

	// Render built-in agents
	for i, agent := range m.builtinAgents {
		agentList.WriteString(m.renderAgentRow(agent, cursor == m.agentCursor && !m.startFocusCustom))
		agentList.WriteString("\n")
		cursor++
		_ = i // suppress unused variable warning
	}

	// Separator between built-in and custom agents
	if len(m.customAgents) > 0 {
		separator := m.styles.GroupHeaderLine.Render("────────────────────────")
		agentList.WriteString(separator + "\n")

		// Render custom agents
		for _, agent := range m.customAgents {
			agentList.WriteString(m.renderAgentRow(agent, cursor == m.agentCursor && !m.startFocusCustom))
			agentList.WriteString("\n")
			cursor++
		}
	}

	// Custom command input
	customLabel := m.styles.Footer.Render("Or enter custom command")
	if m.startFocusCustom {
		customLabel = m.styles.InputPrompt.Render("Or enter custom command")
	}
	customInputView := m.customInput.View()

	// Hint based on focus
	var hint string
	if m.startFocusCustom {
		hint = m.styles.FooterKey.Render("tab") + m.styles.Footer.Render(" agents  ") +
			m.styles.FooterKey.Render("enter") + m.styles.Footer.Render(" start  ") +
			m.styles.FooterKey.Render("esc") + m.styles.Footer.Render(" cancel")
	} else {
		hint = m.styles.FooterKey.Render("↑↓") + m.styles.Footer.Render(" select  ") +
			m.styles.FooterKey.Render("tab") + m.styles.Footer.Render(" custom  ") +
			m.styles.FooterKey.Render("enter") + m.styles.Footer.Render(" start  ") +
			m.styles.FooterKey.Render("esc") + m.styles.Footer.Render(" cancel")
	}

	_ = allAgents // suppress unused variable warning

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		taskTitle,
		"",
		selectLabel,
		agentList.String(),
		customLabel,
		customInputView,
		"",
		hint,
	)
	return m.styles.Dialog.Render(content)
}

// renderAgentRow renders a single agent row with cursor and command preview.
func (m *Model) renderAgentRow(agent string, selected bool) string {
	// Get command preview
	cmdPreview := m.agentCommands[agent]
	if cmdPreview == "" {
		cmdPreview = agent
	}

	// Format: "  ▸ agent_name    command_preview"
	name := fmt.Sprintf("%-12s", agent)
	preview := m.styles.Footer.Render(cmdPreview)

	if selected {
		cursor := m.styles.CursorSelected.Render("▸")
		return "  " + cursor + " " + m.styles.TaskTitleSelected.Render(name) + "  " + preview
	}
	return "    " + name + "  " + preview
}

// viewFooter renders the footer with key hints.
func (m *Model) viewFooter() string {
	switch m.mode {
	case ModeNormal:
		// Match mock format: j/k nav  enter open  n new  ? help  q quit
		return m.styles.Footer.Render(
			m.styles.FooterKey.Render("j/k") + " nav  " +
				m.styles.FooterKey.Render("enter") + " open  " +
				m.styles.FooterKey.Render("n") + " new  " +
				m.styles.FooterKey.Render("?") + " help  " +
				m.styles.FooterKey.Render("q") + " quit",
		)
	case ModeFilter:
		return m.styles.Footer.Render("enter apply · esc cancel")
	case ModeConfirm, ModeInputTitle, ModeInputDesc, ModeStart, ModeHelp, ModeDetail:
		// Hints are shown in the dialogs/views themselves
		return ""
	}
	return ""
}

// viewHelp renders the help view.
func (m *Model) viewHelp() string {
	title := m.styles.HeaderText.Render("KEYBOARD SHORTCUTS")

	sections := []struct {
		name  string
		binds []struct {
			key  string
			desc string
		}
	}{
		{
			name: "NAVIGATION",
			binds: []struct {
				key  string
				desc string
			}{
				{"↑/k", "Move up"},
				{"↓/j", "Move down"},
				{"enter", "Open/Action"},
				{"/", "Filter"},
				{"v", "Details"},
			},
		},
		{
			name: "ACTIONS",
			binds: []struct {
				key  string
				desc string
			}{
				{"s", "Start"},
				{"S", "Stop"},
				{"a", "Attach"},
				{"n", "New Task"},
				{"d", "Delete"},
				{"c", "Close"},
			},
		},
		{
			name: "GENERAL",
			binds: []struct {
				key  string
				desc string
			}{
				{"r", "Refresh"},
				{"?", "Close Help"},
				{"q", "Quit"},
			},
		},
	}

	// Two-column layout for sections
	var col1, col2 strings.Builder

	renderSection := func(b *strings.Builder, sectionIdx int) {
		section := sections[sectionIdx]
		b.WriteString(m.styles.GroupHeaderLabel.Render(section.name))
		b.WriteString("\n")
		for _, bind := range section.binds {
			key := m.styles.HelpKey.Width(8).Render(bind.key)
			desc := m.styles.HelpDesc.Render(bind.desc)
			fmt.Fprintf(b, "%s %s\n", key, desc)
		}
		b.WriteString("\n")
	}

	// Navigation in col 1
	renderSection(&col1, 0)

	// Actions and General in col 2
	renderSection(&col2, 1)
	renderSection(&col2, 2)

	// Join columns
	content := lipgloss.JoinHorizontal(lipgloss.Top,
		col1.String(),
		"    ", // Gutter
		col2.String(),
	)

	// Box it
	return m.styles.Dialog.
		BorderForeground(Colors.Primary).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, "", content))
}

// viewDetail renders the task detail view.
func (m *Model) viewDetail() string {
	task := m.SelectedTask()
	if task == nil {
		return "No task selected"
	}

	var b strings.Builder

	// Styles
	labelStyle := m.styles.DetailLabel
	valueStyle := m.styles.DetailValue
	sectionStyle := m.styles.DetailTitle

	// Header
	b.WriteString(sectionStyle.Render(fmt.Sprintf("Task #%d", task.ID)))
	b.WriteString("\n")
	b.WriteString(m.styles.TaskTitleSelected.Render(task.Title))
	b.WriteString("\n\n")

	// Properties Grid
	// Status
	b.WriteString(labelStyle.Render("Status"))
	b.WriteString(m.styles.StatusStyle(task.Status).Render(string(task.Status)))
	b.WriteString("\n")

	// Agent
	if task.Agent != "" {
		b.WriteString(labelStyle.Render("Agent"))
		b.WriteString(valueStyle.Render(task.Agent))
		b.WriteString("\n")
	}

	// Session
	if task.Session != "" {
		b.WriteString(labelStyle.Render("Session"))
		b.WriteString(valueStyle.Render(task.Session))
		b.WriteString("\n")
	}

	// Created
	b.WriteString(labelStyle.Render("Created"))
	b.WriteString(valueStyle.Render(task.Created.Format("2006-01-02 15:04")))
	b.WriteString("\n")

	// Started
	if !task.Started.IsZero() {
		b.WriteString(labelStyle.Render("Started"))
		b.WriteString(valueStyle.Render(task.Started.Format("2006-01-02 15:04")))
		b.WriteString("\n")
	}

	// Description Section
	if task.Description != "" {
		b.WriteString("\n")
		b.WriteString(m.styles.DetailLabel.Render("Description"))
		b.WriteString("\n")
		b.WriteString(m.styles.DetailDesc.Render(task.Description))
		b.WriteString("\n")
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(m.styles.Footer.Render("[esc] back"))

	// Wrap in a nice box
	return m.styles.Dialog.
		Width(m.width - 4).
		BorderForeground(Colors.Muted).
		Render(b.String())
}
