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

	// Task list (v1-style grouped)
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

// viewHeader renders the header.
func (m *Model) viewHeader() string {
	title := m.styles.HeaderText.Render("git-crew")
	taskCount := fmt.Sprintf(" (%d tasks)", len(m.visibleTasks()))
	return m.styles.Header.Render(title + m.styles.Footer.Render(taskCount))
}

// viewTaskList renders the grouped task list (v1-style).
func (m *Model) viewTaskList() string {
	if len(m.groupedItems) == 0 {
		return m.styles.Footer.Render("No tasks. Press 'n' to create one.")
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
			b.WriteString("\n\n") // blank lines after header
		} else {
			selected := taskIdx == m.cursor
			// Render cursor + task item with proper indentation
			line := m.renderTaskItem(item.task, selected)
			cursor := "    "
			if selected {
				cursor = "  > "
			}
			b.WriteString(fmt.Sprintf("%s%s\n\n", cursor, line))
			taskIdx++
		}
	}

	return m.styles.TaskList.Render(b.String())
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

// renderGroupHeader renders a group header (v1-style).
func (m *Model) renderGroupHeader(status domain.Status, count int) string {
	// Format: "──────── in_progress (3) ────────"
	label := fmt.Sprintf(" %s %s (%d) ", StatusIcon(status), status, count)

	// Calculate line widths
	contentWidth := m.width - 4 // padding
	if contentWidth < 40 {
		contentWidth = 40
	}

	labelLen := len(label)
	lineLen := (contentWidth - labelLen) / 2
	if lineLen < 2 {
		lineLen = 2
	}

	leftLine := strings.Repeat("─", lineLen)
	rightLine := strings.Repeat("─", lineLen)

	// Apply status color to the label
	statusStyle := m.styles.StatusStyle(status)
	styledLabel := statusStyle.Render(label)

	return m.styles.GroupHeaderLine.Render(leftLine) + styledLabel + m.styles.GroupHeaderLine.Render(rightLine)
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

// renderTaskItem renders a single task item (v1-style single line).
// Format: "#1 [status] - title" (cursor is added separately in viewNormalList)
func (m *Model) renderTaskItem(task *domain.Task, selected bool) string {
	// Format: "#1 [status] - title" with colored status
	var idPart, statusPart, titlePart string

	if selected {
		idPart = m.styles.TaskSelected.Render(fmt.Sprintf("#%d", task.ID))
		statusPart = m.styles.StatusStyleSelected(task.Status).Render(fmt.Sprintf("[%s]", task.Status))
		titlePart = m.styles.TaskSelected.Render(fmt.Sprintf("- %s", task.Title))
	} else {
		idPart = m.styles.TaskNormal.Render(fmt.Sprintf("#%d", task.ID))
		statusPart = m.styles.StatusStyle(task.Status).Render(fmt.Sprintf("[%s]", task.Status))
		titlePart = m.styles.TaskNormal.Render(fmt.Sprintf("- %s", task.Title))
	}

	return fmt.Sprintf("%s %s %s", idPart, statusPart, titlePart)
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
	selectLabel := m.styles.InputPrompt.Render("Select agent:")

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
		separator := m.styles.Footer.Render("────────────────────────")
		agentList.WriteString(separator + "\n")

		// Render custom agents
		for _, agent := range m.customAgents {
			agentList.WriteString(m.renderAgentRow(agent, cursor == m.agentCursor && !m.startFocusCustom))
			agentList.WriteString("\n")
			cursor++
		}
	}

	// Custom command input
	customLabel := m.styles.Footer.Render("Or type custom command:")
	customInputView := m.customInput.View()
	if m.startFocusCustom {
		customLabel = m.styles.InputPrompt.Render("Or type custom command:")
	}

	// Hint based on focus
	var hint string
	if m.startFocusCustom {
		hint = m.styles.Footer.Render("[tab] agent list  [enter] start  [esc] cancel")
	} else {
		hint = m.styles.Footer.Render("[↑/↓] select  [tab] custom  [enter] start  [esc] cancel")
	}

	_ = allAgents // suppress unused variable warning

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
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
	cursorStr := "  "
	if selected {
		cursorStr = "> "
	}

	// Get command preview
	cmdPreview := m.agentCommands[agent]
	if cmdPreview == "" {
		cmdPreview = agent
	}

	// Format: "  agent_name    command_preview"
	name := fmt.Sprintf("%-12s", agent)
	preview := m.styles.Footer.Render(cmdPreview)

	if selected {
		return m.styles.TaskSelected.Render(cursorStr+name) + "  " + preview
	}
	return cursorStr + name + "  " + preview
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
