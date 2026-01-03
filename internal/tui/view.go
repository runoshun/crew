package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	MinWidthForDetailPanel = 100
	DetailPanelWidth       = 40
	GutterWidth            = 1
	appPadding             = 4
)

// View renders the TUI.
func (m *Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	base := m.viewMain()

	var dialog string
	switch m.mode {
	case ModeNormal, ModeFilter:
	case ModeConfirm:
		dialog = m.viewConfirmDialog()
	case ModeInputTitle:
		dialog = m.viewTitleInput()
	case ModeInputDesc:
		dialog = m.viewDescInput()
	case ModeStart:
		dialog = m.viewAgentPicker()
	case ModeHelp:
		dialog = m.viewHelp()
	case ModeDetail:
		dialog = m.viewDetail()
	}

	if dialog != "" {
		return m.styles.App.Render(m.overlayDialog(base, dialog))
	}

	return m.styles.App.Render(base)
}

func (m *Model) overlayDialog(_, dialog string) string {
	return lipgloss.Place(
		m.width-appPadding,
		m.height-2,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}

func (m *Model) viewMain() string {
	var leftPane strings.Builder

	leftPane.WriteString(m.viewHeader())
	leftPane.WriteString("\n")

	if m.err != nil {
		leftPane.WriteString(m.styles.ErrorMsg.Render("Error: "+m.err.Error()) + "\n\n")
	}

	if m.mode == ModeFilter {
		leftPane.WriteString(m.styles.InputPrompt.Render("Filter: "))
		leftPane.WriteString(m.filterInput.View())
		leftPane.WriteString("\n\n")
	} else if m.filterInput.Value() != "" {
		leftPane.WriteString(m.styles.Footer.Render("Filtered: "+m.filterInput.Value()) + "\n\n")
	}

	leftPane.WriteString(m.viewTaskList())
	leftPane.WriteString("\n")
	leftPane.WriteString(m.viewFooter())

	if m.showDetailPanel() {
		rightContent := m.viewDetailPanel()
		return lipgloss.JoinHorizontal(lipgloss.Top, leftPane.String(), rightContent)
	}

	return leftPane.String()
}

func (m *Model) headerFooterContentWidth() int {
	width := m.listWidth() - 6
	if m.showDetailPanel() {
		width -= 1
	}
	if width < 40 {
		width = 40
	}
	return width
}

func (m *Model) viewHeader() string {
	title := m.styles.HeaderText.Render("Tasks")

	contentWidth := m.headerFooterContentWidth()
	visibleCount := len(m.taskList.Items())
	totalCount := len(m.tasks)

	sortLabel := "by " + m.sortMode.String()
	countText := fmt.Sprintf("%s · %d/%d", sortLabel, visibleCount, totalCount)
	rightText := lipgloss.NewStyle().Foreground(Colors.Muted).Render(countText)

	leftLen := lipgloss.Width(title)
	rightLen := lipgloss.Width(rightText)
	spacing := contentWidth - leftLen - rightLen
	if spacing < 1 {
		spacing = 1
	}

	content := title + strings.Repeat(" ", spacing) + rightText
	return m.styles.Header.Render(content)
}

func (m *Model) viewTaskList() string {
	if len(m.taskList.Items()) == 0 {
		return m.viewEmptyState()
	}
	return m.taskList.View()
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

func (m *Model) viewFooter() string {
	var content string
	switch m.mode {
	case ModeNormal:
		content = m.styles.FooterKey.Render("j/k") + " nav  " +
			m.styles.FooterKey.Render("enter") + " open  " +
			m.styles.FooterKey.Render("n") + " new  " +
			m.styles.FooterKey.Render("?") + " help  " +
			m.styles.FooterKey.Render("q") + " quit"
	case ModeFilter:
		content = "enter apply · esc cancel"
	case ModeConfirm, ModeInputTitle, ModeInputDesc, ModeStart, ModeHelp, ModeDetail:
		return ""
	default:
		return ""
	}

	pagination := m.taskList.Paginator.View()

	contentWidth := m.headerFooterContentWidth()
	contentLen := lipgloss.Width(content)
	paginationLen := lipgloss.Width(pagination)
	spacing := contentWidth - contentLen - paginationLen
	if spacing < 1 {
		spacing = 1
	}

	fullContent := content + strings.Repeat(" ", spacing) + pagination
	return m.styles.Footer.Render(fullContent)
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

func (m *Model) viewDetail() string {
	width := m.width - 8
	if width < 40 {
		width = 40
	}

	footer := m.styles.Footer.Render("[↑↓] scroll  [esc] back")

	dialogStyle := lipgloss.NewStyle().
		Background(Colors.Background).
		Padding(1, 2).
		Width(width + 4)

	return dialogStyle.Render(m.detailViewport.View() + "\n\n" + footer)
}

func (m *Model) showDetailPanel() bool {
	return m.width >= MinWidthForDetailPanel
}

func (m *Model) contentWidth() int {
	return m.width - appPadding
}

func (m *Model) listWidth() int {
	if m.showDetailPanel() {
		return m.contentWidth() - DetailPanelWidth - GutterWidth
	}
	return m.contentWidth()
}

func (m *Model) viewDetailPanel() string {
	panelHeight := m.height - 2
	if panelHeight < 10 {
		panelHeight = 10
	}
	panelStyle := lipgloss.NewStyle().
		Width(DetailPanelWidth).
		Height(panelHeight).
		PaddingLeft(GutterWidth).
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(Colors.GroupLine)

	task := m.SelectedTask()
	if task == nil {
		emptyStyle := lipgloss.NewStyle().
			Foreground(Colors.Muted).
			Padding(2, 1)
		return panelStyle.Render(emptyStyle.Render("Select a task\nto view details"))
	}

	var b strings.Builder

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(Colors.Primary).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(Colors.GroupLine).
		Width(DetailPanelWidth - 4)
	b.WriteString(headerStyle.Render(fmt.Sprintf("Task #%d", task.ID)))
	b.WriteString("\n\n")

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(Colors.TitleSelected).
		Width(DetailPanelWidth - 4)
	b.WriteString(titleStyle.Render(task.Title))
	b.WriteString("\n\n")

	labelStyle := lipgloss.NewStyle().
		Foreground(Colors.Muted).
		Width(10)
	valueStyle := lipgloss.NewStyle().
		Foreground(Colors.TitleNormal)

	b.WriteString(labelStyle.Render("Status"))
	statusIcon := StatusIcon(task.Status)
	statusText := StatusText(task.Status)
	b.WriteString(m.styles.StatusStyle(task.Status).Render(statusIcon + " " + statusText))
	b.WriteString("\n")

	if task.Agent != "" {
		b.WriteString(labelStyle.Render("Agent"))
		b.WriteString(valueStyle.Render(task.Agent))
		b.WriteString("\n")
	}

	b.WriteString(labelStyle.Render("Created"))
	b.WriteString(valueStyle.Render(task.Created.Format("01/02 15:04")))
	b.WriteString("\n")

	if !task.Started.IsZero() {
		b.WriteString(labelStyle.Render("Started"))
		b.WriteString(valueStyle.Render(task.Started.Format("01/02 15:04")))
		b.WriteString("\n")
	}

	if task.Description != "" {
		b.WriteString("\n")
		descLabelStyle := lipgloss.NewStyle().
			Foreground(Colors.Muted).
			Bold(true)
		b.WriteString(descLabelStyle.Render("Description"))
		b.WriteString("\n")
		descStyle := lipgloss.NewStyle().
			Foreground(Colors.DescSelected).
			Width(DetailPanelWidth - 4)
		b.WriteString(descStyle.Render(task.Description))
	}

	return panelStyle.Render(b.String())
}
