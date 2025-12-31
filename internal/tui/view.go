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
	if paginationInfo := m.viewPagination(); paginationInfo != "" {
		b.WriteString("\n")
		b.WriteString(paginationInfo)
	}

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

// viewHeader renders the header with branding and task count.
func (m *Model) viewHeader() string {
	// Brand logo with accent
	logo := m.styles.HeaderText.Render("◆ git-crew")

	// Task count badge
	taskCount := len(m.visibleTasks())
	countText := fmt.Sprintf("%d tasks", taskCount)
	if taskCount == 1 {
		countText = "1 task"
	}
	// Subtle badge
	countBadge := m.styles.Footer.Render(" · " + countText)

	return m.styles.Header.Render(logo + countBadge)
}

// viewTaskList renders the grouped task list.
func (m *Model) viewTaskList() string {
	if len(m.groupedItems) == 0 {
		return m.viewEmptyState()
	}

	// Calculate visible range for pagination
	listHeight := m.listHeight()
	startIdx, endIdx := m.visibleRange(m.groupedItems, listHeight)

	// Count task index before startIdx
	taskIdxOffset := 0
	for i := 0; i < startIdx; i++ {
		if !m.groupedItems[i].isHeader {
			taskIdxOffset++
		}
	}

	var b strings.Builder
	taskIdx := taskIdxOffset // Track task index for selection
	for i := startIdx; i < endIdx; i++ {
		item := m.groupedItems[i]

		if item.isHeader {
			// Add blank line before header (except at the very start)
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(m.renderGroupHeader(item.status, item.count))
			b.WriteString("\n")
		} else {
			selected := taskIdx == m.cursor
			// Render cursor + task item with proper indentation
			line := m.renderTaskItem(item.task, selected)

			// Use the background color for the whole line if selected
			if selected {
				cursor := m.styles.CursorSelected.Render("▸")
				b.WriteString(m.styles.TaskSelected.Render(fmt.Sprintf("  %s %s", cursor, line)))
				b.WriteString("\n")
			} else {
				b.WriteString(fmt.Sprintf("    %s\n", line))
			}
			taskIdx++
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

// listHeight returns the available height for the list.
func (m *Model) listHeight() int {
	// Reserve space for header, footer, padding, etc.
	reserved := 8 // header + footer + margins
	if m.err != nil {
		reserved += 2
	}
	if m.mode == ModeFilter || m.filterInput.Value() != "" {
		reserved += 2
	}

	height := m.height - reserved
	if height < 5 {
		height = 5
	}
	return height
}

// visibleRange calculates the start and end indices for visible items.
func (m *Model) visibleRange(items []listItem, listHeight int) (int, int) {
	if len(items) == 0 {
		return 0, 0
	}

	// Find the current cursor position in items
	cursorItemIdx := m.findCursorItemIndex(items)

	// Calculate actual line count (headers take 3 lines: blank + header + blank, tasks take 2: task + blank)
	calcLines := func(start, end int) int {
		lines := 0
		for i := start; i < end && i < len(items); i++ {
			if items[i].isHeader {
				lines += 3 // blank before + header + blank after
			} else {
				lines += 2 // task + blank after
			}
		}
		return lines
	}

	// Check if all items fit
	totalLines := calcLines(0, len(items))
	if totalLines <= listHeight {
		return 0, len(items)
	}

	// Need to scroll - center the cursor in the visible window
	// Start from the beginning and find a window that includes the cursor
	startIdx := 0
	endIdx := 0

	// Find the starting position that includes the cursor
	// We want the cursor to be visible, preferably with some context before it
	for endIdx < len(items) {
		linesUsed := calcLines(startIdx, endIdx+1)
		if linesUsed <= listHeight {
			endIdx++
		} else {
			// Can't fit more, check if cursor is visible
			if endIdx > cursorItemIdx {
				break // cursor is visible
			}
			// Need to scroll - move start forward
			startIdx++
			endIdx++
		}
	}

	// Ensure cursor is in the visible range
	if cursorItemIdx < startIdx {
		startIdx = cursorItemIdx
		// Recalculate endIdx
		endIdx = startIdx
		for endIdx < len(items) && calcLines(startIdx, endIdx+1) <= listHeight {
			endIdx++
		}
	} else if cursorItemIdx >= endIdx {
		endIdx = cursorItemIdx + 1
		// Recalculate startIdx
		for startIdx < cursorItemIdx && calcLines(startIdx, endIdx) > listHeight {
			startIdx++
		}
	}

	return startIdx, endIdx
}

// findCursorItemIndex finds the index in items that corresponds to the current cursor.
func (m *Model) findCursorItemIndex(items []listItem) int {
	taskIdx := 0
	for i, item := range items {
		if item.isHeader {
			continue
		}
		if taskIdx == m.cursor {
			return i
		}
		taskIdx++
	}
	return 0
}

// renderGroupHeader renders a group header with status icon and count.
func (m *Model) renderGroupHeader(status domain.Status, count int) string {
	// Format: "─── ● in_progress (3) ───────────────────"
	icon := StatusIcon(status)
	label := fmt.Sprintf(" %s %s ", icon, status)
	countStr := fmt.Sprintf("(%d)", count)

	// Calculate line widths for balanced appearance
	contentWidth := m.width - 6 // padding
	if contentWidth < 40 {
		contentWidth = 40
	}

	// Left line is short, right line fills the rest
	leftLineLen := 3
	labelLen := len(label) + len(countStr)
	rightLineLen := contentWidth - leftLineLen - labelLen - 2
	if rightLineLen < 3 {
		rightLineLen = 3
	}

	leftLine := strings.Repeat("─", leftLineLen)
	rightLine := strings.Repeat("─", rightLineLen)

	// Apply status color to the icon and label
	statusStyle := m.styles.StatusStyle(status)
	styledLabel := statusStyle.Render(label)
	styledCount := m.styles.Footer.Render(countStr)

	return m.styles.GroupHeaderLine.Render(leftLine) + styledLabel + styledCount + " " + m.styles.GroupHeaderLine.Render(rightLine)
}

// viewPagination renders pagination info if needed.
func (m *Model) viewPagination() string {
	items := m.groupedItems
	if len(items) == 0 {
		return ""
	}

	listHeight := m.listHeight()
	startIdx, endIdx := m.visibleRange(items, listHeight)

	// Count total tasks (not headers)
	totalTasks := 0
	for _, item := range items {
		if !item.isHeader {
			totalTasks++
		}
	}

	// Count visible tasks
	visibleTasks := 0
	for i := startIdx; i < endIdx; i++ {
		if !items[i].isHeader {
			visibleTasks++
		}
	}

	if visibleTasks >= totalTasks {
		return "" // All visible, no pagination needed
	}

	// Show pagination hint
	return m.styles.Footer.Render(fmt.Sprintf("Showing %d of %d tasks (↑↓ to scroll)", visibleTasks, totalTasks))
}

// renderTaskItem renders a single task item.
// Format: "#1  title  [agent]" (cursor is added separately in viewTaskList)
func (m *Model) renderTaskItem(task *domain.Task, selected bool) string {
	var idPart, titlePart, agentPart string

	// Task ID with fixed width for alignment
	idStr := fmt.Sprintf("#%-3d", task.ID)

	if selected {
		// When selected, the container has a background, so we just bold the text
		// and use the primary/selected colors
		idPart = m.styles.TaskIDSelected.Render(idStr)
		titlePart = m.styles.TaskTitleSelected.Render(task.Title)
		if task.Agent != "" {
			agentPart = m.styles.TaskAgentSelected.Render(fmt.Sprintf(" [%s]", task.Agent))
		}
	} else {
		idPart = m.styles.TaskID.Render(idStr)
		titlePart = m.styles.TaskTitle.Render(task.Title)
		if task.Agent != "" {
			agentPart = m.styles.TaskAgent.Render(fmt.Sprintf(" [%s]", task.Agent))
		}
	}

	return fmt.Sprintf("%s %s%s", idPart, titlePart, agentPart)
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
		// Main mode hints - organized by importance
		return m.styles.FooterKey.Render("↑↓") + m.styles.Footer.Render(" navigate  ") +
			m.styles.FooterKey.Render("enter") + m.styles.Footer.Render(" action  ") +
			m.styles.FooterKey.Render("s") + m.styles.Footer.Render(" start  ") +
			m.styles.FooterKey.Render("n") + m.styles.Footer.Render(" new  ") +
			m.styles.FooterKey.Render("?") + m.styles.Footer.Render(" help  ") +
			m.styles.FooterKey.Render("q") + m.styles.Footer.Render(" quit")
	case ModeFilter:
		return m.styles.FooterKey.Render("enter") + m.styles.Footer.Render(" apply  ") +
			m.styles.FooterKey.Render("esc") + m.styles.Footer.Render(" cancel")
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
