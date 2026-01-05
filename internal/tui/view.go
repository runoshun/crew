package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	MinWidthForDetailPanel = 100
	MinDetailPanelWidth    = 40
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

	if len(m.warnings) > 0 {
		for _, w := range m.warnings {
			leftPane.WriteString(m.styles.ErrorMsg.Foreground(Colors.Warning).Render("Warning: "+w) + "\n")
		}
		leftPane.WriteString("\n")
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

	// Add filter indicator
	filterLabel := ""
	if m.showAll {
		filterLabel = "[all] · "
	} else {
		filterLabel = "[active] · "
	}

	sortLabel := "by " + m.sortMode.String()
	countText := fmt.Sprintf("%s%s · %d/%d", filterLabel, sortLabel, visibleCount, totalCount)
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
		color = Colors.Stopped
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
				{"a", "Toggle all"},
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
				{"A", "Attach"},
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

func (m *Model) detailPanelWidth() int {
	if !m.showDetailPanel() {
		return 0
	}
	// 40% of screen width, but at least MinDetailPanelWidth
	w := int(float64(m.width) * 0.4)
	if w < MinDetailPanelWidth {
		w = MinDetailPanelWidth
	}
	return w
}

func (m *Model) contentWidth() int {
	return m.width - appPadding
}

func (m *Model) listWidth() int {
	if m.showDetailPanel() {
		return m.contentWidth() - m.detailPanelWidth() - GutterWidth
	}
	return m.contentWidth()
}

func (m *Model) viewDetailPanel() string {
	panelWidth := m.detailPanelWidth()
	panelHeight := m.height - 2
	if panelHeight < 10 {
		panelHeight = 10
	}
	panelStyle := lipgloss.NewStyle().
		Width(panelWidth).
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

	// Track lines as we build content, stopping when we exceed panelHeight
	contentLines := make([]string, 0, panelHeight)
	totalHeight := 0

	// Helper function to add lines and track height
	addLines := func(lines ...string) bool {
		allAdded := true
		for _, line := range lines {
			h := lipgloss.Height(line)
			if h == 0 {
				h = 1
			}
			if totalHeight+h > panelHeight {
				allAdded = false
				break
			}
			contentLines = append(contentLines, line)
			totalHeight += h
		}
		// If we couldn't add all lines, add ellipsis
		if !allAdded {
			if totalHeight < panelHeight {
				// There's space for ellipsis
				contentLines = append(contentLines, "...")
				totalHeight++
			} else if len(contentLines) > 0 {
				// Panel is full, replace last line with ellipsis
				lastIdx := len(contentLines) - 1
				contentLines[lastIdx] = "..."
			}
		}
		return allAdded
	}

	// Header: "Task #N"
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(Colors.Primary).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(Colors.GroupLine).
		Width(panelWidth - 4)
	if !addLines(headerStyle.Render(fmt.Sprintf("Task #%d", task.ID)), "") {
		return panelStyle.Render(strings.Join(contentLines, "\n"))
	}

	// Title (may wrap to multiple lines)
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(Colors.TitleSelected).
		Width(panelWidth - 4)
	if !addLines(titleStyle.Render(task.Title), "") {
		return panelStyle.Render(strings.Join(contentLines, "\n"))
	}

	// Status
	labelStyle := lipgloss.NewStyle().
		Foreground(Colors.Muted).
		Width(10)
	valueStyle := lipgloss.NewStyle().
		Foreground(Colors.TitleNormal)

	statusIcon := StatusIcon(task.Status)
	statusText := StatusText(task.Status)
	statusLine := labelStyle.Render("Status") + m.styles.StatusStyle(task.Status).Render(statusIcon+" "+statusText)
	if !addLines(statusLine) {
		return panelStyle.Render(strings.Join(contentLines, "\n"))
	}

	// Labels (if present)
	if len(task.Labels) > 0 {
		var labelsBuilder strings.Builder
		for _, l := range task.Labels {
			labelsBuilder.WriteString("[" + l + "] ")
		}
		labelsLine := labelStyle.Render("Labels") + m.styles.TaskLabel.Render(labelsBuilder.String())
		if !addLines(labelsLine) {
			return panelStyle.Render(strings.Join(contentLines, "\n"))
		}
	}

	// Agent (if present)
	if task.Agent != "" {
		agentLine := labelStyle.Render("Agent") + valueStyle.Render(task.Agent)
		if !addLines(agentLine) {
			return panelStyle.Render(strings.Join(contentLines, "\n"))
		}
	}

	// Created
	createdLine := labelStyle.Render("Created") + valueStyle.Render(task.Created.Format("01/02 15:04"))
	if !addLines(createdLine) {
		return panelStyle.Render(strings.Join(contentLines, "\n"))
	}

	// Started (if present)
	if !task.Started.IsZero() {
		startedLine := labelStyle.Render("Started") + valueStyle.Render(task.Started.Format("01/02 15:04"))
		if !addLines(startedLine) {
			return panelStyle.Render(strings.Join(contentLines, "\n"))
		}
	}

	// Description (split into lines and add line-by-line)
	if task.Description != "" {
		descLabelStyle := lipgloss.NewStyle().
			Foreground(Colors.Muted).
			Bold(true)
		descStyle := lipgloss.NewStyle().
			Foreground(Colors.DescSelected).
			Width(panelWidth - 4)

		// Add label and empty line first
		if !addLines("", descLabelStyle.Render("Description")) {
			return panelStyle.Render(strings.Join(contentLines, "\n"))
		}

		// Render description with width constraint to get wrapped text
		renderedDesc := descStyle.Render(task.Description)
		// Split by newlines (lipgloss wraps text with \n)
		descLines := strings.Split(renderedDesc, "\n")

		// Add each line individually (will stop early if height limit reached)
		for _, line := range descLines {
			if !addLines(line) {
				return panelStyle.Render(strings.Join(contentLines, "\n"))
			}
		}
	}

	// Comments (may wrap and have multiple entries)
	if len(m.comments) > 0 {
		commentLabelStyle := lipgloss.NewStyle().
			Foreground(Colors.Muted).
			Bold(true)
		if !addLines("", commentLabelStyle.Render("Comments")) {
			return panelStyle.Render(strings.Join(contentLines, "\n"))
		}

		for _, comment := range m.comments {
			timeStr := comment.Time.Format("01/02 15:04")
			commentStyle := lipgloss.NewStyle().
				Foreground(Colors.DescSelected).
				Width(panelWidth - 4)
			commentLine := lipgloss.NewStyle().Foreground(Colors.Muted).Render("["+timeStr+"] ") + commentStyle.Render(comment.Text)
			if !addLines(commentLine) {
				return panelStyle.Render(strings.Join(contentLines, "\n"))
			}
		}
	}

	return panelStyle.Render(strings.Join(contentLines, "\n"))
}
