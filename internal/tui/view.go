package tui

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	overlay "github.com/rmhubbert/bubbletea-overlay"
	"github.com/runoshun/git-crew/v2/internal/domain"
)

// ansiResetPattern matches ANSI reset sequences that affect background:
// - \x1b[0m or \x1b[m (full reset)
// - \x1b[49m (background reset only)
var ansiResetPattern = regexp.MustCompile(`\x1b\[(0?m|49m)`)

// ansiTrailingResetPattern matches trailing ANSI reset at end of string
var ansiTrailingResetPattern = regexp.MustCompile(`\x1b\[0?m$`)

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
		key:        lipgloss.NewStyle().Background(bg).Foreground(Colors.Maroon).Bold(true),
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

// fillViewportLines pads each line of the viewport content to the dialog width
// and fills remaining height with empty lines, all with the dialog background color.
// This ensures the entire viewport area has consistent background color.
//
// The function handles ANSI escape sequences correctly by reinserting the background
// color after any ANSI reset sequences (\x1b[0m) within the line content.
func (ds dialogStyles) fillViewportLines(viewportContent string, height int) string {
	lines := strings.Split(viewportContent, "\n")
	result := make([]string, 0, height)
	padStyle := lipgloss.NewStyle().Background(ds.bg)

	// Extract the background color ANSI sequence.
	// lipgloss.Render("") returns "\x1b[48;...m\x1b[0m" (bg + reset),
	// so we strip the trailing reset to get just the background sequence.
	bgSeq := ansiTrailingResetPattern.ReplaceAllString(padStyle.Render(""), "")

	// Pad each content line to the full width
	for _, line := range lines {
		if len(result) >= height {
			break
		}
		if line == "" {
			result = append(result, ds.emptyLine())
			continue
		}

		renderedLine := line
		if lipgloss.Width(renderedLine) > ds.width {
			renderedLine = lipgloss.NewStyle().MaxWidth(ds.width).Render(renderedLine)
		}
		lineWidth := lipgloss.Width(renderedLine)
		padWidth := ds.width - lineWidth
		if padWidth < 0 {
			padWidth = 0
		}

		// Reinsert background color after any ANSI reset sequences within the line.
		// This ensures the background color is maintained even when glamour/markdown
		// renderers emit reset codes (e.g., after bold text).
		fixedLine := bgSeq + ansiResetPattern.ReplaceAllString(renderedLine, "$0"+bgSeq)

		// Add padding with background color
		paddedLine := fixedLine
		if padWidth > 0 {
			paddedLine = fixedLine + padStyle.Width(padWidth).Render("")
		}
		result = append(result, paddedLine)
	}

	// Fill remaining height with empty lines
	for len(result) < height {
		result = append(result, ds.emptyLine())
	}

	return strings.Join(result, "\n")
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
	case ModeChangeStatus:
		dialog = m.viewStatusPicker()
	case ModeExec:
		dialog = m.viewExecDialog()
	case ModeActionMenu:
		dialog = m.viewActionMenuDialog()
	case ModeReviewResult:
		dialog = m.viewReviewResultDialog()
	case ModeReviewAction:
		dialog = m.viewReviewActionDialog()
	case ModeReviewMessage:
		dialog = m.viewReviewMessageDialog()
	case ModeEditReviewComment:
		dialog = m.viewEditReviewCommentDialog()
	case ModeBlock:
		dialog = m.viewBlockDialog()
	}

	if dialog != "" {
		return m.styles.App.Render(m.overlayDialog(base, dialog))
	}

	return m.styles.App.Render(base)
}

func (m *Model) overlayDialog(base, dialog string) string {
	return overlay.Composite(
		dialog,
		base,
		overlay.Center,
		overlay.Center,
		0, 0,
	)
}

func (m *Model) viewMain() string {
	if len(m.tasks) == 0 {
		return m.viewEmptyState()
	}

	// On narrow screen with detail focused, show only the detail panel
	if !m.showListPane() {
		return m.viewDetailPanel()
	}

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
	// Return the same width as taskList for consistent alignment
	// taskList width is set in updateLayoutSizes() based on listWidth()
	// We subtract 4 from listWidth to account for the left margin (4 spaces) used in delegate rendering
	width := m.listWidth() - 4
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

	// Add review mode indicator (show current mode)
	reviewModeLabel := ""
	if m.config != nil {
		mode := domain.ReviewModeAuto
		if m.config.Complete.ReviewModeSet {
			mode = m.config.Complete.ReviewMode
		} else if m.config.Complete.AutoFixSet && m.config.Complete.AutoFix { //nolint:staticcheck // Legacy compatibility
			mode = domain.ReviewModeAutoFix
		}
		switch mode {
		case domain.ReviewModeAutoFix:
			reviewModeLabel = lipgloss.NewStyle().Foreground(Colors.Primary).Render("[R:fix]") + " · "
		case domain.ReviewModeManual:
			reviewModeLabel = lipgloss.NewStyle().Foreground(Colors.Warning).Render("[R:man]") + " · "
		case domain.ReviewModeAuto:
			reviewModeLabel = lipgloss.NewStyle().Foreground(Colors.Success).Render("[R:auto]") + " · "
		}
	} else {
		reviewModeLabel = lipgloss.NewStyle().Foreground(Colors.Success).Render("[R:auto]") + " · "
	}

	// Add filter indicator
	filterLabel := ""
	if m.showAll {
		filterLabel = "[all] · "
	} else {
		filterLabel = "[active] · "
	}

	sortLabel := "by " + m.sortMode.String()
	countText := fmt.Sprintf("%s%s%s · %d/%d", reviewModeLabel, filterLabel, sortLabel, visibleCount, totalCount)
	rightText := lipgloss.NewStyle().Foreground(Colors.Muted).Render(countText)

	leftLen := lipgloss.Width(title)
	rightLen := lipgloss.Width(rightText)
	// Header style has Padding(0, 1), so inner content width is contentWidth - 2
	innerWidth := contentWidth - 2
	spacing := innerWidth - leftLen - rightLen
	if spacing < 1 {
		spacing = 1
	}

	content := title + strings.Repeat(" ", spacing) + rightText
	// Set width dynamically to match list width
	return m.styles.Header.Width(contentWidth).Render(content)
}

func (m *Model) viewTaskList() string {
	if len(m.taskList.Items()) == 0 {
		return m.viewFilteredEmptyState()
	}
	return m.taskList.View()
}

// viewEmptyState renders a friendly empty state message centered on screen.
func (m *Model) viewEmptyState() string {
	contentWidth := m.contentWidth()
	contentHeight := m.height - 2
	if contentWidth < 0 {
		contentWidth = 0
	}
	if contentHeight < 0 {
		contentHeight = 0
	}

	titleStyle := m.styles.HeaderText
	bodyStyle := lipgloss.NewStyle().Foreground(Colors.Muted)
	keyStyle := lipgloss.NewStyle().Foreground(Colors.Maroon).Bold(true)
	cmdStyle := lipgloss.NewStyle().Foreground(Colors.Primary)

	// ASCII logo - compact to fit narrow terminals
	logo := lipgloss.NewStyle().Foreground(Colors.Primary).Render(`
      ___           ___           ___           ___     
     /  /\         /  /\         /  /\         /__/\    
    /  /:/        /  /::\       /  /:/_       _\_ \:\   
   /  /:/        /  /:/\:\     /  /:/ /\     /__/\ \:\  
  /  /:/  ___   /  /:/~/:/    /  /:/ /:/_   _\_ \:\ \:\ 
 /__/:/  /  /\ /__/:/ /:/___ /__/:/ /:/ /\ /__/\ \:\ \:\
 \  \:\ /  /:/ \  \:\/:::::/ \  \:\/:/ /:/ \  \:\ \:\/:/
  \  \:\  /:/   \  \::/~~~~   \  \::/ /:/   \  \:\ \::/ 
   \  \:\/:/     \  \:\        \  \:\/:/     \  \:\/:/  
    \  \::/       \  \:\        \  \::/       \  \::/   
     \__\/         \__\/         \__\/         \__\/`)

	title := titleStyle.Render("No tasks yet")
	subtitle := bodyStyle.Render("Create your first task to get started")

	// Hints section
	hint1 := lipgloss.JoinHorizontal(lipgloss.Left,
		bodyStyle.Render("Press "),
		keyStyle.Render("n"),
		bodyStyle.Render(" to create a new task"),
	)
	hint2 := lipgloss.JoinHorizontal(lipgloss.Left,
		bodyStyle.Render("Or run: "),
		cmdStyle.Render("crew new --title \"Your task\""),
	)
	hint3 := lipgloss.JoinHorizontal(lipgloss.Left,
		bodyStyle.Render("Need help? "),
		cmdStyle.Render("crew manager"),
		bodyStyle.Render(" for AI assistance"),
	)

	// Build content based on available height
	var content string
	if contentHeight >= 14 && contentWidth >= 40 {
		// Full layout with logo
		content = lipgloss.JoinVertical(lipgloss.Center,
			logo,
			"",
			title,
			subtitle,
			"",
			hint1,
			hint2,
			hint3,
		)
	} else {
		// Compact layout without logo
		content = lipgloss.JoinVertical(lipgloss.Center,
			title,
			subtitle,
			"",
			hint1,
			hint2,
			hint3,
		)
	}

	return lipgloss.Place(contentWidth, contentHeight, lipgloss.Center, lipgloss.Center, content)
}

// viewFilteredEmptyState renders empty state when filter hides tasks.
func (m *Model) viewFilteredEmptyState() string {
	contentWidth := m.contentWidth()
	contentHeight := m.height - 2
	if contentWidth < 0 {
		contentWidth = 0
	}
	if contentHeight < 0 {
		contentHeight = 0
	}

	titleStyle := m.styles.HeaderText
	bodyStyle := lipgloss.NewStyle().Foreground(Colors.Muted)
	hintStyle := lipgloss.NewStyle().Foreground(Colors.Maroon).Bold(true)

	title := titleStyle.Render("No matching tasks")
	primary := bodyStyle.Render("Clear the filter to see all tasks")
	hint := lipgloss.JoinHorizontal(
		lipgloss.Center,
		bodyStyle.Render("Hint: "),
		hintStyle.Render("press / then backspace"),
	)

	content := lipgloss.JoinVertical(lipgloss.Center, title, "", primary, hint)
	return lipgloss.Place(contentWidth, contentHeight, lipgloss.Center, lipgloss.Center, content)
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
		if m.confirmReviewSession {
			target = fmt.Sprintf("review session for task #%d", m.confirmTaskID)
		} else {
			target = fmt.Sprintf("work session for task #%d", m.confirmTaskID)
		}
		color = Colors.Stopped
	case ConfirmMerge:
		action = "Merge"
		target = fmt.Sprintf("task #%d", m.confirmTaskID)
		color = Colors.Reviewed
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
	lines := make([]string, 0, 4+len(agentRows)+4)
	lines = append(lines, title, taskTitle, ds.emptyLine(), selectLabel)
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
			m.styles.FooterKey.Render("enter") + " default  " +
			m.styles.FooterKey.Render("space") + " actions  " +
			m.styles.FooterKey.Render("n") + " new  " +
			m.styles.FooterKey.Render("?") + " help  " +
			m.styles.FooterKey.Render("q") + " quit"
		if m.canStopSelectedTask(m.SelectedTask()) {
			content = content + "  " + m.styles.FooterKey.Render("S") + " stop"
		}
	case ModeFilter:
		content = "enter apply · esc cancel"
	case ModeChangeStatus:
		content = "enter select · esc cancel"
	case ModeExec:
		content = "enter execute · esc cancel"
	case ModeConfirm, ModeInputTitle, ModeInputDesc, ModeNewTask, ModeStart, ModeHelp, ModeActionMenu, ModeReviewResult, ModeReviewAction, ModeReviewMessage, ModeEditReviewComment, ModeBlock:
		return ""
	default:
		return ""
	}

	pagination := m.taskList.Paginator.View()

	contentWidth := m.headerFooterContentWidth()
	// Footer style has Padding(0, 1), so inner content width is contentWidth - 2
	innerWidth := contentWidth - 2
	contentLen := lipgloss.Width(content)
	paginationLen := lipgloss.Width(pagination)

	// Truncate content if too wide
	maxContentWidth := innerWidth - paginationLen - 1 // 1 for spacing
	if contentLen > maxContentWidth {
		if maxContentWidth <= 3 {
			// When space is very limited, show only "..."
			content = "..."
		} else {
			// Truncate content and append "..."
			truncateStyle := lipgloss.NewStyle().MaxWidth(maxContentWidth - 3)
			content = truncateStyle.Render(content) + "..."
		}
		contentLen = lipgloss.Width(content)
	}

	spacing := innerWidth - contentLen - paginationLen
	if spacing < 1 {
		spacing = 1
	}

	fullContent := content + strings.Repeat(" ", spacing) + pagination
	// Set width dynamically to match list width
	return m.styles.Footer.Width(contentWidth).Render(fullContent)
}

func (m *Model) viewHelp() string {
	ds := m.newDialogStyles()
	keyStyleWide := ds.key.Width(8)
	sectionStyle := lipgloss.NewStyle().Background(ds.bg).Foreground(Colors.Muted).Bold(true)
	baseStyle := lipgloss.NewStyle().Background(ds.bg)

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
				{"enter", "Default"},
				{"space", "Actions"},
				{"/", "Filter"},
				{"o", "Sort"},
				{"v", "Details"},
				{"A", "Toggle all"},
				{"t", "Toggle review mode"},
			},
		},
		{
			name: "ACTIONS",
			binds: []struct {
				key  string
				desc string
			}{
				{"space", "Actions"},
				{"s", "Start"},
				{"S", "Stop (work/review)"},
				{"a", "Attach"},
				{"x", "Execute"},
				{"R", "Review"},
				{"n", "New Task"},
				{"e", "Change Status"},
				{"E", "Edit Task"},
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

	// Build section columns as arrays of lines
	renderSection := func(sectionIdx int) []string {
		section := sections[sectionIdx]
		lines := make([]string, 0, len(section.binds)+2)
		lines = append(lines, sectionStyle.Render(section.name))
		for _, bind := range section.binds {
			key := keyStyleWide.Render(bind.key)
			desc := ds.muted.Render(bind.desc)
			lines = append(lines, key+baseStyle.Render(" ")+desc)
		}
		return lines
	}

	col1Lines := renderSection(0)
	col2Lines := renderSection(1)
	col2Lines = append(col2Lines, "") // Empty line between ACTIONS and GENERAL
	col2Lines = append(col2Lines, renderSection(2)...)

	// Pad columns to same length
	maxLines := len(col1Lines)
	if len(col2Lines) > maxLines {
		maxLines = len(col2Lines)
	}
	for len(col1Lines) < maxLines {
		col1Lines = append(col1Lines, "")
	}
	for len(col2Lines) < maxLines {
		col2Lines = append(col2Lines, "")
	}

	// Calculate column widths
	col1Width := 0
	for _, line := range col1Lines {
		if w := lipgloss.Width(line); w > col1Width {
			col1Width = w
		}
	}
	gapWidth := 4
	col2Width := ds.width - col1Width - gapWidth
	if col2Width < 20 {
		col2Width = 20
	}

	// Build merged lines with background
	col1Style := baseStyle.Width(col1Width)
	gapStyle := baseStyle.Width(gapWidth)
	col2Style := baseStyle.Width(col2Width)

	var contentLines []string
	for i := 0; i < maxLines; i++ {
		line := col1Style.Render(col1Lines[i]) + gapStyle.Render("") + col2Style.Render(col2Lines[i])
		contentLines = append(contentLines, ds.renderLine(line))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, contentLines...)

	hint := ds.renderLine(ds.key.Render("esc") + ds.muted.Render(" close"))

	return m.dialogStyle().Render(lipgloss.JoinVertical(lipgloss.Left, title, ds.emptyLine(), content, hint))
}

func (m *Model) showDetailPanel() bool {
	// Always show detail panel when focused, even on narrow screens
	if m.detailFocused {
		return true
	}
	return m.width >= MinWidthForDetailPanel
}

// showListPane returns whether the list pane should be shown.
// On narrow screens with detail focused, hide the list entirely.
func (m *Model) showListPane() bool {
	if m.detailFocused && m.width < MinWidthForDetailPanel {
		return false
	}
	return true
}

func (m *Model) viewStatusPicker() string {
	task := m.SelectedTask()
	if task == nil {
		return ""
	}

	ds := m.newDialogStyles()
	baseStyle := lipgloss.NewStyle().Background(ds.bg)

	title := ds.renderLine(ds.label.Render("Change Status"))
	taskLine := ds.renderLine(ds.text.Render(fmt.Sprintf("Task #%d: %s", task.ID, task.Title)))
	currentLine := ds.renderLine(ds.muted.Render("Current: ") + m.styles.StatusStyle(task.Status).Background(ds.bg).Render(string(task.Status)))
	selectLabel := ds.renderLine(ds.label.Render("Select new status:"))

	transitions := m.getStatusTransitions(task.Status)
	var statusRows []string

	if len(transitions) == 0 {
		statusRows = append(statusRows, ds.renderLine(ds.muted.Render("  No transitions available")))
	} else {
		hasForced := false
		for i, status := range transitions {
			isNormal := task.Status.CanTransitionTo(status)
			if !isNormal && !hasForced {
				// Add separator before forced transitions with centered (force)
				label := " (force) "
				totalWidth := ds.width - 4
				sideWidth := (totalWidth - len(label)) / 2
				if sideWidth < 0 {
					sideWidth = 0
				}
				sep := strings.Repeat("─", sideWidth)
				fill := ""
				if sideWidth*2+len(label) < totalWidth {
					fill = " "
				}
				separatorLine := ds.muted.Render("  " + sep + label + sep + fill)
				statusRows = append(statusRows, ds.renderLine(separatorLine))
				hasForced = true
			}

			selected := i == m.statusCursor
			cursor := " "
			cursorStyle := ds.label.Foreground(Colors.Primary)
			style := ds.text
			if selected {
				cursor = "▸"
				style = ds.label
			}

			displayText := string(status)
			row := ds.renderLine(baseStyle.Render("  ") + cursorStyle.Render(cursor) + baseStyle.Render(" ") + style.Render(displayText))
			statusRows = append(statusRows, row)
		}
	}

	hint := ds.renderLine(ds.key.Render("enter") + ds.text.Render(" select · ") +
		ds.key.Render("esc") + ds.text.Render(" cancel"))

	lines := make([]string, 0, 6+len(statusRows)+2)
	lines = append(lines, title, ds.emptyLine(), taskLine, currentLine, ds.emptyLine(), selectLabel)
	lines = append(lines, statusRows...)
	lines = append(lines, ds.emptyLine(), hint)

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return m.dialogStyle().Render(content)
}

func (m *Model) viewExecDialog() string {
	task := m.SelectedTask()
	if task == nil {
		return ""
	}

	ds := m.newDialogStyles()
	title := ds.renderLine(ds.label.Render("Execute Command"))
	taskLine := ds.renderLine(ds.muted.Render(fmt.Sprintf("Task #%d: %s", task.ID, task.Title)))

	branch := domain.BranchName(task.ID, task.Issue)
	wtPath, _ := m.container.Worktrees.Resolve(branch)
	// Make path relative to repo root if possible for better display
	displayPath := wtPath
	if rel, err := filepath.Rel(m.container.Config.RepoRoot, wtPath); err == nil {
		displayPath = rel
	}
	wtLine := ds.renderLine(ds.muted.Render("Worktree: " + displayPath))

	label := ds.renderLine(ds.label.Render("Command"))
	input := ds.renderLine(m.execInput.View())
	hint := ds.renderLine(ds.key.Render("enter") + ds.text.Render(" execute  ") +
		ds.key.Render("esc") + ds.text.Render(" cancel"))

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		ds.emptyLine(),
		taskLine,
		wtLine,
		ds.emptyLine(),
		label,
		input,
		ds.emptyLine(),
		hint,
	)

	return m.dialogStyle().Render(content)
}

func (m *Model) detailPanelWidth() int {
	if !m.showDetailPanel() {
		return 0
	}

	// On narrow screen + focused: use full width (list is hidden)
	if m.detailFocused && m.width < MinWidthForDetailPanel {
		return m.contentWidth()
	}

	// Determine ratio based on focus state
	var ratio float64
	if m.detailFocused {
		// Wide screen + focused: 70%
		ratio = 0.7
	} else {
		// Not focused (only possible on wide screens): 40%
		ratio = 0.4
	}

	w := int(float64(m.width) * ratio)
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

// detailPanelContent builds the content string for the detail panel viewport.
func (m *Model) detailPanelContent(contentWidth int) string {
	task := m.SelectedTask()
	if task == nil {
		return "Select a task\nto view details"
	}

	var lines []string

	// Header: "Task #N"
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(Colors.Primary).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(Colors.GroupLine).
		Width(contentWidth)
	lines = append(lines, headerStyle.Render(fmt.Sprintf("Task #%d", task.ID)), "")

	// Title (may wrap to multiple lines)
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(Colors.TitleSelected).
		Width(contentWidth)
	lines = append(lines, titleStyle.Render(task.Title), "")

	// Status
	labelStyle := lipgloss.NewStyle().
		Foreground(Colors.Muted).
		Width(10)
	valueStyle := lipgloss.NewStyle().
		Foreground(Colors.TitleNormal)

	statusIcon := StatusIcon(task.Status)
	statusText := StatusText(task.Status)
	statusLine := labelStyle.Render("Status") + m.styles.StatusStyle(task.Status).Render(statusIcon+" "+statusText)
	lines = append(lines, statusLine)

	// Labels (if present)
	if len(task.Labels) > 0 {
		var labelsBuilder strings.Builder
		for _, l := range task.Labels {
			labelsBuilder.WriteString("[" + l + "] ")
		}
		labelsLine := labelStyle.Render("Labels") + m.styles.TaskLabel.Render(labelsBuilder.String())
		lines = append(lines, labelsLine)
	}

	// Agent (if present)
	if task.Agent != "" {
		agentLine := labelStyle.Render("Agent") + valueStyle.Render(task.Agent)
		lines = append(lines, agentLine)
	}

	// Created
	createdLine := labelStyle.Render("Created") + valueStyle.Render(task.Created.Format("01/02 15:04"))
	lines = append(lines, createdLine)

	// Started (if present)
	if !task.Started.IsZero() {
		startedLine := labelStyle.Render("Started") + valueStyle.Render(task.Started.Format("01/02 15:04"))
		lines = append(lines, startedLine)
	}

	// Description
	if task.Description != "" {
		descLabelStyle := lipgloss.NewStyle().
			Foreground(Colors.Muted).
			Bold(true)
		lines = append(lines, "", descLabelStyle.Render("Description"))

		// Render description with markdown
		renderedDesc := m.styles.RenderMarkdown(task.Description, contentWidth)
		lines = append(lines, renderedDesc)
	}

	// Comments
	if len(m.comments) > 0 {
		commentLabelStyle := lipgloss.NewStyle().
			Foreground(Colors.Muted).
			Bold(true)
		lines = append(lines, "", commentLabelStyle.Render("Comments"))

		separator := lipgloss.NewStyle().
			Foreground(Colors.GroupLine).
			Render("─────────────────")
		lines = append(lines, separator)

		for i, comment := range m.comments {
			if i > 0 {
				lines = append(lines, "")
			}
			timeStr := comment.Time.Format("01/02 15:04")
			authorPart := ""
			if comment.Author != "" {
				authorPart = " · " + comment.Author
			}
			headerLine := lipgloss.NewStyle().Foreground(Colors.Muted).Render(timeStr + authorPart)
			lines = append(lines, headerLine)

			commentStyle := lipgloss.NewStyle().
				Foreground(Colors.TitleNormal).
				Width(contentWidth)
			lines = append(lines, commentStyle.Render(comment.Text))
		}
	}

	return strings.Join(lines, "\n")
}

func (m *Model) viewDetailPanel() string {
	panelWidth := m.detailPanelWidth()
	panelHeight := m.height - 2
	if panelHeight < 10 {
		panelHeight = 10
	}

	// On narrow screen (full width mode), no left border needed
	fullWidthMode := !m.showListPane()

	panelStyle := lipgloss.NewStyle().
		Width(panelWidth).
		Height(panelHeight)

	if !fullWidthMode {
		// Change border color based on focus state
		borderColor := Colors.GroupLine
		if m.detailFocused {
			borderColor = Colors.Primary
		}
		panelStyle = panelStyle.
			PaddingLeft(GutterWidth).
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(borderColor)
	}

	task := m.SelectedTask()
	if task == nil {
		emptyStyle := lipgloss.NewStyle().
			Foreground(Colors.Muted).
			Padding(2, 1)
		return panelStyle.Render(emptyStyle.Render("Select a task\nto view details"))
	}

	// Use viewport for scrollable content
	content := m.detailPanelViewport.View()

	// Add footer hint when focused
	if m.detailFocused {
		footerStyle := lipgloss.NewStyle().
			Foreground(Colors.Muted).
			Width(panelWidth - 4)
		scrollInfo := fmt.Sprintf(" %d%%", int(m.detailPanelViewport.ScrollPercent()*100))
		footer := footerStyle.Render("↑↓ scroll · j/k half · g/G jump · v/esc back" + scrollInfo)
		content = content + "\n" + footer
	}

	return panelStyle.Render(content)
}

func (m *Model) viewActionMenuDialog() string {
	if len(m.actionMenuItems) == 0 {
		return ""
	}
	// Find task title
	var taskTitle string
	taskID := 0
	if task := m.SelectedTask(); task != nil {
		taskTitle = task.Title
		taskID = task.ID
	}

	ds := m.newDialogStyles()
	baseStyle := lipgloss.NewStyle().Background(ds.bg)
	labelStyle := baseStyle.Foreground(Colors.TitleNormal)
	mutedStyle := baseStyle.Foreground(Colors.Muted)
	cursorStyle := baseStyle.Foreground(Colors.Primary)

	title := ds.renderLine(ds.label.Render("Task Actions"))
	taskLine := ""
	if taskID > 0 {
		taskLine = ds.renderLine(ds.muted.Render(fmt.Sprintf("Task #%d: %s", taskID, taskTitle)))
	}

	rows := make([]string, 0, len(m.actionMenuItems))
	for i, action := range m.actionMenuItems {
		cursor := " "
		rowStyle := labelStyle
		if i == m.actionMenuCursor {
			cursor = "▸"
			rowStyle = ds.label
		}
		defaultTag := ""
		if action.IsDefault {
			defaultTag = ds.muted.Render(" (default)")
		}
		keyLabel := ""
		if action.Key != "" {
			keyLabel = ds.key.Render(action.Key) + baseStyle.Render(" ")
		}
		row := ds.renderLine(
			baseStyle.Render("  ") +
				cursorStyle.Render(cursor) +
				baseStyle.Render(" ") +
				keyLabel +
				rowStyle.Render(fmt.Sprintf("%-14s", action.Label)) +
				baseStyle.Render(" ") +
				mutedStyle.Render(action.Desc) +
				defaultTag,
		)
		rows = append(rows, row)
	}

	hint := ds.renderLine(
		ds.key.Render("↑↓") + ds.text.Render(" select  ") +
			ds.key.Render("enter") + ds.text.Render(" execute  ") +
			ds.key.Render("esc") + ds.text.Render(" close"))

	lines := []string{title}
	if taskLine != "" {
		lines = append(lines, ds.emptyLine(), taskLine)
	}
	lines = append(lines, ds.emptyLine())
	lines = append(lines, rows...)
	lines = append(lines, ds.emptyLine(), hint)

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return m.dialogStyle().Render(content)
}

func (m *Model) viewReviewResultDialog() string {
	ds := m.newDialogStyles()

	title := ds.renderLine(ds.label.Render("Review Complete"))

	// Find task title
	var taskTitle string
	for _, t := range m.tasks {
		if t.ID == m.reviewTaskID {
			taskTitle = t.Title
			break
		}
	}

	taskLine := ds.renderLine(ds.muted.Render(fmt.Sprintf("Task #%d: %s", m.reviewTaskID, taskTitle)))

	// Use viewport for scrollable review content
	// Fill viewport lines with background color to ensure consistent appearance
	viewportContent := ds.fillViewportLines(m.reviewViewport.View(), m.reviewViewport.Height)

	// Add scroll indicator
	scrollInfo := ""
	if m.reviewViewport.TotalLineCount() > m.reviewViewport.VisibleLineCount() {
		scrollInfo = fmt.Sprintf(" %d%%", int(m.reviewViewport.ScrollPercent()*100))
	}

	hint := ds.renderLine(
		ds.key.Render("j/k") + ds.text.Render(" scroll  ") +
			ds.key.Render("enter") + ds.text.Render(" actions  ") +
			ds.key.Render("esc") + ds.text.Render(" close") +
			ds.muted.Render(scrollInfo))

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		ds.emptyLine(),
		taskLine,
		ds.emptyLine(),
		viewportContent,
		ds.emptyLine(),
		hint,
	)

	return m.dialogStyle().Render(content)
}

func (m *Model) viewReviewActionDialog() string {
	ds := m.newDialogStyles()
	baseStyle := lipgloss.NewStyle().Background(ds.bg)

	title := ds.renderLine(ds.label.Render("Select Action"))

	// Find task
	var task *domain.Task
	for _, t := range m.tasks {
		if t.ID == m.reviewTaskID {
			task = t
			break
		}
	}
	if task == nil {
		return ""
	}

	taskLine := ds.renderLine(ds.muted.Render(fmt.Sprintf("Task #%d: %s", task.ID, task.Title)))

	// Action options (5 options)
	type actionOption struct {
		label string
		desc  string
	}

	options := []actionOption{
		{"Request Changes", "Send message and restart task (back to in_progress)"},
		{"Comment Only", "Send review as comment without restarting"},
		{"Merge", "Approve and merge the task (LGTM)"},
		{"Close", "Close the task without merging"},
		{"Edit Comment", "Edit the saved review comment"},
	}

	actionRows := make([]string, 0, len(options))
	for i, opt := range options {
		selected := i == m.reviewActionCursor
		cursor := " "
		cursorStyle := ds.label.Foreground(Colors.Primary)
		labelStyle := ds.text
		descStyle := ds.muted
		if selected {
			cursor = "▸"
			labelStyle = ds.label
		}

		row := ds.renderLine(
			baseStyle.Render("  ") +
				cursorStyle.Render(cursor) +
				baseStyle.Render(" ") +
				labelStyle.Render(fmt.Sprintf("%-16s", opt.label)) +
				baseStyle.Render(" ") +
				descStyle.Render(opt.desc))
		actionRows = append(actionRows, row)
	}

	hint := ds.renderLine(
		ds.key.Render("↑↓") + ds.text.Render(" select  ") +
			ds.key.Render("enter") + ds.text.Render(" confirm  ") +
			ds.key.Render("esc") + ds.text.Render(" back"))

	lines := make([]string, 0, 4+len(actionRows)+2)
	lines = append(lines, title, ds.emptyLine(), taskLine, ds.emptyLine())
	lines = append(lines, actionRows...)
	lines = append(lines, ds.emptyLine(), hint)

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return m.dialogStyle().Render(content)
}

func (m *Model) viewReviewMessageDialog() string {
	ds := m.newDialogStyles()

	title := ds.renderLine(ds.label.Render("Request Changes"))

	// Find task
	var task *domain.Task
	for _, t := range m.tasks {
		if t.ID == m.reviewTaskID {
			task = t
			break
		}
	}
	if task == nil {
		return ""
	}

	taskLine := ds.renderLine(ds.muted.Render(fmt.Sprintf("Task #%d: %s", task.ID, task.Title)))

	description := ds.renderLine(ds.text.Render("Enter a message for the worker (optional):"))

	inputLine := ds.renderLine(m.reviewMessageInput.View())

	defaultHint := ds.renderLine(ds.muted.Render("Leave empty to use: \"Please address the review comments above.\""))

	hint := ds.renderLine(
		ds.key.Render("enter") + ds.text.Render(" send  ") +
			ds.key.Render("esc") + ds.text.Render(" back"))

	content := lipgloss.JoinVertical(lipgloss.Left,
		title, ds.emptyLine(),
		taskLine, ds.emptyLine(),
		description,
		inputLine,
		defaultHint, ds.emptyLine(),
		hint,
	)

	return m.dialogStyle().Render(content)
}

func (m *Model) viewEditReviewCommentDialog() string {
	ds := m.newDialogStyles()

	title := ds.renderLine(ds.label.Render("Edit Review Comment"))

	// Find task
	var task *domain.Task
	for _, t := range m.tasks {
		if t.ID == m.reviewTaskID {
			task = t
			break
		}
	}
	if task == nil {
		return ""
	}

	taskLine := ds.renderLine(ds.muted.Render(fmt.Sprintf("Task #%d: %s", task.ID, task.Title)))

	description := ds.renderLine(ds.text.Render("Edit the review comment:"))

	inputLine := ds.renderLine(m.editCommentInput.View())

	hint := ds.renderLine(
		ds.key.Render("enter") + ds.text.Render(" save  ") +
			ds.key.Render("esc") + ds.text.Render(" cancel"))

	content := lipgloss.JoinVertical(lipgloss.Left,
		title, ds.emptyLine(),
		taskLine, ds.emptyLine(),
		description,
		inputLine, ds.emptyLine(),
		hint,
	)

	return m.dialogStyle().Render(content)
}

func (m *Model) viewBlockDialog() string {
	task := m.SelectedTask()
	if task == nil {
		return ""
	}

	ds := m.newDialogStyles()
	baseStyle := lipgloss.NewStyle().Background(ds.bg)

	title := ds.renderLine(ds.label.Render("Block Task"))
	taskLine := ds.renderLine(ds.muted.Render(fmt.Sprintf("Task #%d: %s", task.ID, task.Title)))

	// Show current status if blocked
	var statusLine string
	if task.IsBlocked() {
		statusLine = ds.renderLine(ds.muted.Render("Currently blocked: ") + ds.text.Render(task.BlockReason))
	} else {
		statusLine = ds.renderLine(ds.muted.Render("Not currently blocked"))
	}

	// Reason input
	reasonLabel := ds.labelMuted.Render("Reason:")
	if !m.blockFocusUnblock {
		reasonLabel = ds.label.Render("Reason:")
	}
	inputLine := ds.renderLine(m.blockInput.View())

	// Unblock button
	unblockLabel := ds.muted.Render("[ Unblock ]")
	if m.blockFocusUnblock {
		unblockLabel = ds.label.Render("[ Unblock ]")
	}
	unblockLine := ds.renderLine(baseStyle.Render("  ") + unblockLabel)

	// Hint
	var hint string
	if m.blockFocusUnblock {
		hint = ds.renderLine(
			ds.key.Render("tab") + ds.text.Render(" reason  ") +
				ds.key.Render("enter") + ds.text.Render(" unblock  ") +
				ds.key.Render("esc") + ds.text.Render(" cancel"))
	} else {
		hint = ds.renderLine(
			ds.key.Render("tab") + ds.text.Render(" unblock  ") +
				ds.key.Render("enter") + ds.text.Render(" block  ") +
				ds.key.Render("esc") + ds.text.Render(" cancel"))
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		title, ds.emptyLine(),
		taskLine,
		statusLine, ds.emptyLine(),
		ds.renderLine(reasonLabel),
		inputLine, ds.emptyLine(),
		unblockLine, ds.emptyLine(),
		hint,
	)

	return m.dialogStyle().Render(content)
}
