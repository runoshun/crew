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

func (m *Model) dialogStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(Colors.Background).
		Padding(1, 2).
		Width(m.dialogWidth())
}

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
	case ModeNewTask:
		dialog = m.viewNewTaskDialog()
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

	// Find task title
	var taskTitle string
	for _, t := range m.tasks {
		if t.ID == m.confirmTaskID {
			taskTitle = t.Title
			break
		}
	}

	bg := Colors.Background
	width := m.dialogWidth() - 4
	lineStyle := lipgloss.NewStyle().Background(bg).Width(width)
	textStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.TitleNormal)
	keyStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.KeyText).Bold(true)
	mutedStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.Muted)

	title := lineStyle.Render(lipgloss.NewStyle().Background(bg).Foreground(color).Bold(true).
		Render(fmt.Sprintf("%s %s?", action, target)))
	taskTitleLine := lineStyle.Render(mutedStyle.Render(taskTitle))
	prompt := lineStyle.Render(textStyle.Render("This action cannot be undone."))
	emptyLine := lineStyle.Render("")
	buttons := lineStyle.Render(keyStyle.Render("[ y ]") + textStyle.Render(" Confirm  ") +
		mutedStyle.Render("[ n ]") + mutedStyle.Render(" Cancel"))

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		taskTitleLine,
		emptyLine,
		prompt,
		emptyLine,
		buttons,
	)

	return m.dialogStyle().Render(content)
}

func (m *Model) viewTitleInput() string {
	bg := Colors.Background
	width := m.dialogWidth() - 4
	lineStyle := lipgloss.NewStyle().Background(bg).Width(width)
	textStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.TitleNormal)
	keyStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.KeyText).Bold(true)
	mutedStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.Muted)
	labelStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.Primary).Bold(true)

	title := lineStyle.Render(labelStyle.Render("New Task"))
	stepInfo := lineStyle.Render(mutedStyle.Render("Step 1 of 2"))
	label := lineStyle.Render(labelStyle.Render("Title"))
	input := lineStyle.Render(m.titleInput.View())
	emptyLine := lineStyle.Render("")
	hint := lineStyle.Render(keyStyle.Render("enter") + textStyle.Render(" next  ") +
		keyStyle.Render("esc") + textStyle.Render(" cancel"))

	content := lipgloss.JoinVertical(lipgloss.Left, title, stepInfo, emptyLine, label, input, emptyLine, hint)
	return m.dialogStyle().Render(content)
}

func (m *Model) viewDescInput() string {
	bg := Colors.Background
	width := m.dialogWidth() - 4
	lineStyle := lipgloss.NewStyle().Background(bg).Width(width)
	textStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.TitleNormal)
	keyStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.KeyText).Bold(true)
	mutedStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.Muted)
	labelStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.Primary).Bold(true)

	title := lineStyle.Render(labelStyle.Render("New Task"))
	stepInfo := lineStyle.Render(mutedStyle.Render("Step 2 of 2"))
	titleLabel := lineStyle.Render(mutedStyle.Render("Title: ") + textStyle.Render(m.titleInput.Value()))
	label := lineStyle.Render(labelStyle.Render("Description (optional)"))
	input := lineStyle.Render(m.descInput.View())
	emptyLine := lineStyle.Render("")
	hint := lineStyle.Render(keyStyle.Render("enter") + textStyle.Render(" create  ") +
		keyStyle.Render("esc") + textStyle.Render(" back"))

	content := lipgloss.JoinVertical(lipgloss.Left, title, stepInfo, emptyLine, titleLabel, emptyLine, label, input, emptyLine, hint)
	return m.dialogStyle().Render(content)
}

func (m *Model) viewNewTaskDialog() string {
	bg := Colors.Background
	width := m.dialogWidth() - 4
	lineStyle := lipgloss.NewStyle().Background(bg).Width(width)
	textStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.TitleNormal)
	keyStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.KeyText).Bold(true)
	labelStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.Primary).Bold(true)
	labelMutedStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.Muted)

	title := lineStyle.Render(labelStyle.Render("New Task"))

	// Title field
	titleLabel := labelMutedStyle.Render("Title")
	if m.newTaskField == FieldTitle {
		titleLabel = labelStyle.Render("Title")
	}
	titleInput := lineStyle.Render(m.titleInput.View())

	// Description field
	descLabel := labelMutedStyle.Render("Description (optional)")
	if m.newTaskField == FieldDesc {
		descLabel = labelStyle.Render("Description (optional)")
	}
	descInput := lineStyle.Render(m.descInput.View())

	// Parent field
	parentLabel := labelMutedStyle.Render("Parent ID (optional)")
	if m.newTaskField == FieldParent {
		parentLabel = labelStyle.Render("Parent ID (optional)")
	}
	parentInput := lineStyle.Render(m.parentInput.View())

	emptyLine := lineStyle.Render("")
	hint := lineStyle.Render(
		keyStyle.Render("tab") + textStyle.Render(" next  ") +
			keyStyle.Render("shift+tab") + textStyle.Render(" prev  ") +
			keyStyle.Render("enter") + textStyle.Render(" create  ") +
			keyStyle.Render("esc") + textStyle.Render(" cancel"))

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		emptyLine,
		lineStyle.Render(titleLabel),
		titleInput,
		emptyLine,
		lineStyle.Render(descLabel),
		descInput,
		emptyLine,
		lineStyle.Render(parentLabel),
		parentInput,
		emptyLine,
		hint,
	)

	return m.dialogStyle().Render(content)
}

func (m *Model) viewAgentPicker() string {
	task := m.SelectedTask()
	if task == nil {
		return ""
	}

	bg := Colors.Background
	width := m.dialogWidth() - 4
	lineStyle := lipgloss.NewStyle().Background(bg).Width(width)
	textStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.TitleNormal)
	keyStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.KeyText).Bold(true)
	mutedStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.Muted)
	labelStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.Primary).Bold(true)

	title := lineStyle.Render(labelStyle.Render(fmt.Sprintf("Start Task #%d", task.ID)))
	taskTitle := lineStyle.Render(mutedStyle.Render(task.Title))
	selectLabel := lineStyle.Render(labelStyle.Render("Select agent"))
	emptyLine := lineStyle.Render("")

	// Build agent rows
	agentRows := make([]string, 0, len(m.builtinAgents)+len(m.customAgents)+1)
	cursor := 0

	for _, agent := range m.builtinAgents {
		row := m.renderAgentRow(agent, cursor == m.agentCursor && !m.startFocusCustom, width, bg)
		agentRows = append(agentRows, row)
		cursor++
	}

	if len(m.customAgents) > 0 {
		separator := lipgloss.NewStyle().Background(bg).Foreground(Colors.GroupLine).Width(width).
			Render("────────────────────────")
		agentRows = append(agentRows, separator)

		for _, agent := range m.customAgents {
			row := m.renderAgentRow(agent, cursor == m.agentCursor && !m.startFocusCustom, width, bg)
			agentRows = append(agentRows, row)
			cursor++
		}
	}

	customLabel := lineStyle.Render(mutedStyle.Render("Or enter custom command"))
	if m.startFocusCustom {
		customLabel = lineStyle.Render(labelStyle.Render("Or enter custom command"))
	}
	customInputView := lineStyle.Render(m.customInput.View())

	var hint string
	if m.startFocusCustom {
		hint = keyStyle.Render("tab") + textStyle.Render(" agents  ") +
			keyStyle.Render("enter") + textStyle.Render(" start  ") +
			keyStyle.Render("esc") + textStyle.Render(" cancel")
	} else {
		hint = keyStyle.Render("↑↓") + textStyle.Render(" select  ") +
			keyStyle.Render("tab") + textStyle.Render(" custom  ") +
			keyStyle.Render("enter") + textStyle.Render(" start  ") +
			keyStyle.Render("esc") + textStyle.Render(" cancel")
	}

	// Build content
	lines := []string{title, taskTitle, emptyLine, selectLabel}
	lines = append(lines, agentRows...)
	lines = append(lines, customLabel, customInputView, emptyLine, lineStyle.Render(hint))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return m.dialogStyle().Render(content)
}

// renderAgentRow renders a single agent row with cursor and command preview.
func (m *Model) renderAgentRow(agent string, selected bool, width int, bg lipgloss.Color) string {
	lineStyle := lipgloss.NewStyle().Background(bg).Width(width)
	nameStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.TitleNormal)
	previewStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.Muted)

	// Get command preview
	cmdPreview := m.agentCommands[agent]
	if cmdPreview == "" {
		cmdPreview = agent
	}

	// Format: "  ▸ agent_name    command_preview"
	name := fmt.Sprintf("%-12s", agent)

	if selected {
		cursorStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.Primary)
		selectedNameStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.TitleSelected).Bold(true)
		return lineStyle.Render("  " + cursorStyle.Render("▸") + " " + selectedNameStyle.Render(name) + "  " + previewStyle.Render(cmdPreview))
	}
	return lineStyle.Render("    " + nameStyle.Render(name) + "  " + previewStyle.Render(cmdPreview))
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
	case ModeConfirm, ModeInputTitle, ModeInputDesc, ModeNewTask, ModeStart, ModeHelp, ModeDetail:
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

func (m *Model) viewHelp() string {
	bg := Colors.Background
	width := m.dialogWidth() - 4
	lineStyle := lipgloss.NewStyle().Background(bg).Width(width)
	labelStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.Primary).Bold(true)
	keyStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.KeyText).Bold(true).Width(8)
	descStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.Muted)
	sectionStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.Muted).Bold(true)

	title := lineStyle.Render(labelStyle.Render("KEYBOARD SHORTCUTS"))
	emptyLine := lineStyle.Render("")

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
				{"o", "Sort"},
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

	var col1, col2 strings.Builder

	renderSection := func(b *strings.Builder, sectionIdx int) {
		section := sections[sectionIdx]
		b.WriteString(sectionStyle.Render(section.name))
		b.WriteString("\n")
		for _, bind := range section.binds {
			key := keyStyle.Render(bind.key)
			desc := descStyle.Render(bind.desc)
			fmt.Fprintf(b, "%s %s\n", key, desc)
		}
		b.WriteString("\n")
	}

	renderSection(&col1, 0)
	renderSection(&col2, 1)
	renderSection(&col2, 2)

	content := lipgloss.JoinHorizontal(lipgloss.Top,
		col1.String(),
		"    ",
		col2.String(),
	)

	hint := lineStyle.Render(keyStyle.Width(0).Render("esc") + descStyle.Render(" close"))

	return m.dialogStyle().Render(lipgloss.JoinVertical(lipgloss.Left, title, emptyLine, lineStyle.Render(content), hint))
}

func (m *Model) viewDetail() string {
	bg := Colors.Background
	width := m.dialogWidth() - 4
	lineStyle := lipgloss.NewStyle().Background(bg).Width(width)
	keyStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.KeyText).Bold(true)
	textStyle := lipgloss.NewStyle().Background(bg).Foreground(Colors.Muted)
	footer := lineStyle.Render(keyStyle.Render("↑↓") + textStyle.Render(" scroll  ") +
		keyStyle.Render("esc") + textStyle.Render(" back"))

	return m.dialogStyle().Render(m.detailViewport.View() + "\n\n" + footer)
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
