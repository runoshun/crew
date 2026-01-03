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

// dialogStyles holds common styles for dialog rendering.
type dialogStyles struct {
	line       lipgloss.Style // Full-width line with background
	text       lipgloss.Style // Normal text
	key        lipgloss.Style // Key hints (bold)
	muted      lipgloss.Style // Muted/secondary text
	label      lipgloss.Style // Active label (primary color, bold)
	labelMuted lipgloss.Style // Inactive label (muted)
	bg         lipgloss.Color
	width      int
}

// newDialogStyles creates common styles for dialog rendering.
func (m *Model) newDialogStyles() dialogStyles {
	bg := Colors.Background
	width := m.dialogWidth() - 4
	return dialogStyles{
		width:      width,
		bg:         bg,
		line:       lipgloss.NewStyle().Background(bg).Width(width),
		text:       lipgloss.NewStyle().Background(bg).Foreground(Colors.TitleNormal),
		key:        lipgloss.NewStyle().Background(bg).Foreground(Colors.KeyText).Bold(true),
		muted:      lipgloss.NewStyle().Background(bg).Foreground(Colors.Muted),
		label:      lipgloss.NewStyle().Background(bg).Foreground(Colors.Primary).Bold(true),
		labelMuted: lipgloss.NewStyle().Background(bg).Foreground(Colors.Muted),
	}
}

// emptyLine returns an empty line with background.
func (ds dialogStyles) emptyLine() string {
	return ds.line.Render("")
}

// renderLine renders text with full width and background.
func (ds dialogStyles) renderLine(s string) string {
	return ds.line.Render(s)
}

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

	ds := m.newDialogStyles()
	titleStyle := lipgloss.NewStyle().Background(ds.bg).Foreground(color).Bold(true)

	title := ds.renderLine(titleStyle.Render(fmt.Sprintf("%s %s?", action, target)))
	taskTitleLine := ds.renderLine(ds.muted.Render(taskTitle))
	prompt := ds.renderLine(ds.text.Render("This action cannot be undone."))
	buttons := ds.renderLine(ds.key.Render("[ y ]") + ds.text.Render(" Confirm  ") +
		ds.muted.Render("[ n ]") + ds.muted.Render(" Cancel"))

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		taskTitleLine,
		ds.emptyLine(),
		prompt,
		ds.emptyLine(),
		buttons,
	)

	return m.dialogStyle().Render(content)
}

func (m *Model) viewTitleInput() string {
	ds := m.newDialogStyles()

	title := ds.renderLine(ds.label.Render("New Task"))
	stepInfo := ds.renderLine(ds.muted.Render("Step 1 of 2"))
	label := ds.renderLine(ds.label.Render("Title"))
	input := ds.renderLine(m.titleInput.View())
	hint := ds.renderLine(ds.key.Render("enter") + ds.text.Render(" next  ") +
		ds.key.Render("esc") + ds.text.Render(" cancel"))

	content := lipgloss.JoinVertical(lipgloss.Left, title, stepInfo, ds.emptyLine(), label, input, ds.emptyLine(), hint)
	return m.dialogStyle().Render(content)
}

func (m *Model) viewDescInput() string {
	ds := m.newDialogStyles()

	title := ds.renderLine(ds.label.Render("New Task"))
	stepInfo := ds.renderLine(ds.muted.Render("Step 2 of 2"))
	titleLabel := ds.renderLine(ds.muted.Render("Title: ") + ds.text.Render(m.titleInput.Value()))
	label := ds.renderLine(ds.label.Render("Description (optional)"))
	input := ds.renderLine(m.descInput.View())
	hint := ds.renderLine(ds.key.Render("enter") + ds.text.Render(" create  ") +
		ds.key.Render("esc") + ds.text.Render(" back"))

	content := lipgloss.JoinVertical(lipgloss.Left, title, stepInfo, ds.emptyLine(), titleLabel, ds.emptyLine(), label, input, ds.emptyLine(), hint)
	return m.dialogStyle().Render(content)
}

func (m *Model) viewNewTaskDialog() string {
	ds := m.newDialogStyles()

	title := ds.renderLine(ds.label.Render("New Task"))

	// Title field
	titleLabel := ds.labelMuted.Render("Title")
	if m.newTaskField == FieldTitle {
		titleLabel = ds.label.Render("Title")
	}
	titleInput := ds.renderLine(m.titleInput.View())

	// Description field
	descLabel := ds.labelMuted.Render("Description (optional)")
	if m.newTaskField == FieldDesc {
		descLabel = ds.label.Render("Description (optional)")
	}
	descInput := ds.renderLine(m.descInput.View())

	// Parent field
	parentLabel := ds.labelMuted.Render("Parent ID (optional)")
	if m.newTaskField == FieldParent {
		parentLabel = ds.label.Render("Parent ID (optional)")
	}
	parentInput := ds.renderLine(m.parentInput.View())

	hint := ds.renderLine(
		ds.key.Render("tab") + ds.text.Render(" next  ") +
			ds.key.Render("shift+tab") + ds.text.Render(" prev  ") +
			ds.key.Render("enter") + ds.text.Render(" create  ") +
			ds.key.Render("esc") + ds.text.Render(" cancel"))

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		ds.emptyLine(),
		ds.renderLine(titleLabel),
		titleInput,
		ds.emptyLine(),
		ds.renderLine(descLabel),
		descInput,
		ds.emptyLine(),
		ds.renderLine(parentLabel),
		parentInput,
		ds.emptyLine(),
		hint,
	)

	return m.dialogStyle().Render(content)
}

func (m *Model) viewAgentPicker() string {
	task := m.SelectedTask()
	if task == nil {
		return ""
	}

	ds := m.newDialogStyles()

	title := ds.renderLine(ds.label.Render(fmt.Sprintf("Start Task #%d", task.ID)))
	taskTitle := ds.renderLine(ds.muted.Render(task.Title))
	selectLabel := ds.renderLine(ds.label.Render("Select agent"))

	// Build agent rows
	agentRows := make([]string, 0, len(m.builtinAgents)+len(m.customAgents)+1)
	cursor := 0

	for _, agent := range m.builtinAgents {
		row := m.renderAgentRow(agent, cursor == m.agentCursor && !m.startFocusCustom, ds)
		agentRows = append(agentRows, row)
		cursor++
	}

	if len(m.customAgents) > 0 {
		separator := lipgloss.NewStyle().Background(ds.bg).Foreground(Colors.GroupLine).Width(ds.width).
			Render("────────────────────────")
		agentRows = append(agentRows, separator)

		for _, agent := range m.customAgents {
			row := m.renderAgentRow(agent, cursor == m.agentCursor && !m.startFocusCustom, ds)
			agentRows = append(agentRows, row)
			cursor++
		}
	}

	customLabel := ds.renderLine(ds.muted.Render("Or enter custom command"))
	if m.startFocusCustom {
		customLabel = ds.renderLine(ds.label.Render("Or enter custom command"))
	}
	customInputView := ds.renderLine(m.customInput.View())

	var hint string
	if m.startFocusCustom {
		hint = ds.key.Render("tab") + ds.text.Render(" agents  ") +
			ds.key.Render("enter") + ds.text.Render(" start  ") +
			ds.key.Render("esc") + ds.text.Render(" cancel")
	} else {
		hint = ds.key.Render("↑↓") + ds.text.Render(" select  ") +
			ds.key.Render("tab") + ds.text.Render(" custom  ") +
			ds.key.Render("enter") + ds.text.Render(" start  ") +
			ds.key.Render("esc") + ds.text.Render(" cancel")
	}

	// Build content
	lines := []string{title, taskTitle, ds.emptyLine(), selectLabel}
	lines = append(lines, agentRows...)
	lines = append(lines, customLabel, customInputView, ds.emptyLine(), ds.renderLine(hint))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return m.dialogStyle().Render(content)
}

// renderAgentRow renders a single agent row with cursor and command preview.
func (m *Model) renderAgentRow(agent string, selected bool, ds dialogStyles) string {
	// Get command preview
	cmdPreview := m.agentCommands[agent]
	if cmdPreview == "" {
		cmdPreview = agent
	}

	// Use a base style with background for all parts
	baseStyle := lipgloss.NewStyle().Background(ds.bg)
	space := baseStyle.Render(" ")
	doubleSpace := baseStyle.Render("  ")

	// Format: "  ▸ agent_name    command_preview"
	name := fmt.Sprintf("%-12s", agent)

	if selected {
		cursorStyle := baseStyle.Foreground(Colors.Primary)
		selectedNameStyle := baseStyle.Foreground(Colors.TitleSelected).Bold(true)
		return ds.renderLine(doubleSpace + cursorStyle.Render("▸") + space + selectedNameStyle.Render(name) + doubleSpace + ds.muted.Render(cmdPreview))
	}
	return ds.renderLine(baseStyle.Render("    ") + ds.text.Render(name) + doubleSpace + ds.muted.Render(cmdPreview))
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
	ds := m.newDialogStyles()
	keyStyleWide := ds.key.Width(8)
	sectionStyle := lipgloss.NewStyle().Background(ds.bg).Foreground(Colors.Muted).Bold(true)

	title := ds.renderLine(ds.label.Render("KEYBOARD SHORTCUTS"))

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
			key := keyStyleWide.Render(bind.key)
			desc := ds.muted.Render(bind.desc)
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

	hint := ds.renderLine(ds.key.Render("esc") + ds.muted.Render(" close"))

	return m.dialogStyle().Render(lipgloss.JoinVertical(lipgloss.Left, title, ds.emptyLine(), ds.renderLine(content), hint))
}

func (m *Model) viewDetail() string {
	ds := m.newDialogStyles()
	footer := ds.renderLine(ds.key.Render("↑↓") + ds.muted.Render(" scroll  ") +
		ds.key.Render("esc") + ds.muted.Render(" back"))

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

	if len(m.comments) > 0 {
		b.WriteString("\n\n")
		commentLabelStyle := lipgloss.NewStyle().
			Foreground(Colors.Muted).
			Bold(true)
		b.WriteString(commentLabelStyle.Render("Comments"))
		b.WriteString("\n")
		for _, comment := range m.comments {
			timeStr := comment.Time.Format("01/02 15:04")
			commentStyle := lipgloss.NewStyle().
				Foreground(Colors.DescSelected).
				Width(DetailPanelWidth - 4)
			b.WriteString(lipgloss.NewStyle().Foreground(Colors.Muted).Render("[" + timeStr + "] "))
			b.WriteString(commentStyle.Render(comment.Text))
			b.WriteString("\n")
		}
	}

	content := b.String()

	// Truncate content if it exceeds panel height
	// Note: lipgloss renders with word wrapping, so we need to count actual display lines
	renderedLines := strings.Split(content, "\n")
	totalLines := 0
	truncatedLines := make([]string, 0, len(renderedLines))

	for _, line := range renderedLines {
		// Calculate how many display lines this logical line will take
		lineHeight := lipgloss.Height(line)
		if lineHeight == 0 {
			lineHeight = 1 // Empty lines still take 1 line
		}

		if totalLines+lineHeight > panelHeight {
			// Truncate and add ellipsis indicator
			if totalLines < panelHeight {
				truncatedLines = append(truncatedLines, "...")
			}
			break
		}

		truncatedLines = append(truncatedLines, line)
		totalLines += lineHeight
	}

	content = strings.Join(truncatedLines, "\n")
	return panelStyle.Render(content)
}
