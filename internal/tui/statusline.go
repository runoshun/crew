package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// FocusPane represents which pane is currently focused.
type FocusPane int

const (
	FocusPaneWorkspace FocusPane = iota
	FocusPaneTaskList
	FocusPaneTaskDetail
)

func (f FocusPane) String() string {
	switch f {
	case FocusPaneWorkspace:
		return "workspace"
	case FocusPaneTaskList:
		return "tasks"
	case FocusPaneTaskDetail:
		return "detail"
	default:
		return "unknown"
	}
}

// StatusLineInfo contains information for rendering the status line.
// Fields are ordered to minimize memory padding.
type StatusLineInfo struct {
	Pagination string // Optional pagination info (e.g., "1/3")
	KeyHints   []KeyHint
	FocusPane  FocusPane
}

// KeyHint represents a key and its description.
type KeyHint struct {
	Key  string
	Desc string
}

// StatusLine renders a unified status line at the bottom of the screen.
// Fields are ordered to minimize memory padding.
type StatusLine struct {
	styles *Styles
	width  int
}

// NewStatusLine creates a new StatusLine with the given width and styles.
func NewStatusLine(width int, styles *Styles) *StatusLine {
	return &StatusLine{
		width:  width,
		styles: styles,
	}
}

// SetWidth updates the status line width.
func (s *StatusLine) SetWidth(width int) {
	s.width = width
}

// Render renders the status line with the given info.
func (s *StatusLine) Render(info StatusLineInfo) string {
	keyStyle := s.styles.FooterKey
	mutedStyle := lipgloss.NewStyle().Foreground(Colors.Muted)

	// Build key hints
	hints := make([]string, 0, len(info.KeyHints))
	for _, h := range info.KeyHints {
		hints = append(hints, keyStyle.Render(h.Key)+" "+h.Desc)
	}
	content := strings.Join(hints, "  ")

	// Add focus indicator
	focusIndicator := mutedStyle.Render("focus:" + info.FocusPane.String())

	// Calculate spacing
	contentWidth := s.width - 2 // Account for padding

	rightContent := focusIndicator
	if info.Pagination != "" {
		rightContent = info.Pagination + "  " + focusIndicator
	}
	rightLen := lipgloss.Width(rightContent)
	contentLen := lipgloss.Width(content)

	// Truncate content if needed
	maxContentWidth := contentWidth - rightLen - 2
	if contentLen > maxContentWidth {
		if maxContentWidth <= 3 {
			content = "..."
		} else {
			truncateStyle := lipgloss.NewStyle().MaxWidth(maxContentWidth - 3)
			content = truncateStyle.Render(content) + "..."
		}
		contentLen = lipgloss.Width(content)
	}

	spacing := contentWidth - contentLen - rightLen
	if spacing < 1 {
		spacing = 1
	}

	fullContent := content + strings.Repeat(" ", spacing) + rightContent
	return s.styles.Footer.Width(s.width).Render(fullContent)
}

// GetStatusInfo returns status line info for the TUI model.
func (m *Model) GetStatusInfo() StatusLineInfo {
	info := StatusLineInfo{
		FocusPane:  FocusPaneTaskList,
		Pagination: m.taskList.Paginator.View(),
	}

	if m.detailFocused {
		info.FocusPane = FocusPaneTaskDetail
		info.KeyHints = []KeyHint{
			{Key: "j/k", Desc: "scroll"},
			{Key: "tab", Desc: "next"},
			{Key: "h/←", Desc: "back"},
			{Key: "q", Desc: "quit"},
		}
		return info
	}

	// Normal mode - task list focused
	switch m.mode { //nolint:exhaustive // Dialog modes handled by default
	case ModeNormal:
		info.KeyHints = []KeyHint{
			{Key: "j/k", Desc: "nav"},
			{Key: "enter", Desc: "default"},
			{Key: "space", Desc: "actions"},
			{Key: "tab", Desc: "next"},
			{Key: "?", Desc: "help"},
			{Key: "q", Desc: "quit"},
		}
	case ModeFilter:
		info.KeyHints = []KeyHint{
			{Key: "enter", Desc: "apply"},
			{Key: "esc", Desc: "cancel"},
		}
	case ModeChangeStatus:
		info.KeyHints = []KeyHint{
			{Key: "enter", Desc: "select"},
			{Key: "esc", Desc: "cancel"},
		}
	case ModeExec:
		info.KeyHints = []KeyHint{
			{Key: "enter", Desc: "execute"},
			{Key: "esc", Desc: "cancel"},
		}
	default:
		// Dialog modes - no hints in status line
		info.KeyHints = nil
	}

	return info
}
