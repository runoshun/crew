package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/runoshun/git-crew/v2/internal/domain"
)

// Colors defines the color palette for the TUI.
var Colors = struct {
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Muted     lipgloss.Color
	Error     lipgloss.Color
	Success   lipgloss.Color
	Warning   lipgloss.Color

	// Status colors
	Todo        lipgloss.Color
	InProgress  lipgloss.Color
	InReview    lipgloss.Color
	StatusError lipgloss.Color
	Done        lipgloss.Color
	Closed      lipgloss.Color
}{
	Primary:   lipgloss.Color("12"), // Blue
	Secondary: lipgloss.Color("5"),  // Purple
	Muted:     lipgloss.Color("8"),  // Gray
	Error:     lipgloss.Color("9"),  // Red
	Success:   lipgloss.Color("10"), // Green
	Warning:   lipgloss.Color("11"), // Yellow

	// Status colors matching spec
	Todo:        lipgloss.Color("12"), // Blue
	InProgress:  lipgloss.Color("11"), // Yellow
	InReview:    lipgloss.Color("5"),  // Purple
	StatusError: lipgloss.Color("9"),  // Red
	Done:        lipgloss.Color("10"), // Green
	Closed:      lipgloss.Color("8"),  // Gray
}

// Styles contains all the lipgloss styles for the TUI.
type Styles struct {
	// App
	App lipgloss.Style

	// Header
	Header     lipgloss.Style
	HeaderText lipgloss.Style

	// Task list
	TaskList          lipgloss.Style
	TaskItem          lipgloss.Style
	TaskSelected      lipgloss.Style
	TaskID            lipgloss.Style
	TaskIDSelected    lipgloss.Style
	TaskTitle         lipgloss.Style
	TaskTitleSelected lipgloss.Style
	TaskAgent         lipgloss.Style
	TaskAgentSelected lipgloss.Style
	CursorNormal      lipgloss.Style
	CursorSelected    lipgloss.Style

	// Status badges (normal)
	StatusTodo       lipgloss.Style
	StatusInProgress lipgloss.Style
	StatusInReview   lipgloss.Style
	StatusError      lipgloss.Style
	StatusDone       lipgloss.Style
	StatusClosed     lipgloss.Style

	// Status badges (selected - brighter)
	StatusTodoSelected       lipgloss.Style
	StatusInProgressSelected lipgloss.Style
	StatusInReviewSelected   lipgloss.Style
	StatusErrorSelected      lipgloss.Style
	StatusDoneSelected       lipgloss.Style
	StatusClosedSelected     lipgloss.Style

	// Help
	Help     lipgloss.Style
	HelpKey  lipgloss.Style
	HelpDesc lipgloss.Style

	// Footer
	Footer    lipgloss.Style
	FooterKey lipgloss.Style

	// Dialog
	Dialog       lipgloss.Style
	DialogTitle  lipgloss.Style
	DialogPrompt lipgloss.Style

	// Input
	Input       lipgloss.Style
	InputPrompt lipgloss.Style

	// Error
	ErrorMsg lipgloss.Style

	// Detail view
	DetailTitle lipgloss.Style
	DetailLabel lipgloss.Style
	DetailValue lipgloss.Style
	DetailDesc  lipgloss.Style
}

// DefaultStyles returns the default styles for the TUI.
func DefaultStyles() Styles {
	return Styles{
		App: lipgloss.NewStyle().
			Padding(1, 2),

		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(Colors.Primary).
			MarginBottom(1),

		HeaderText: lipgloss.NewStyle().
			Bold(true),

		TaskList: lipgloss.NewStyle().
			MarginBottom(1),

		TaskItem: lipgloss.NewStyle().
			PaddingLeft(2),

		TaskSelected: lipgloss.NewStyle().
			Bold(true).
			Foreground(Colors.Primary),

		TaskID: lipgloss.NewStyle().
			Foreground(Colors.Muted).
			Width(5),

		TaskIDSelected: lipgloss.NewStyle().
			Foreground(Colors.Primary).
			Bold(true).
			Width(5),

		TaskTitle: lipgloss.NewStyle(),

		TaskTitleSelected: lipgloss.NewStyle().
			Foreground(Colors.Primary).
			Bold(true),

		TaskAgent: lipgloss.NewStyle().
			Foreground(Colors.Secondary).
			Italic(true),

		TaskAgentSelected: lipgloss.NewStyle().
			Foreground(Colors.Primary).
			Italic(true),

		CursorNormal: lipgloss.NewStyle().
			Foreground(Colors.Muted),

		CursorSelected: lipgloss.NewStyle().
			Foreground(Colors.Primary).
			Bold(true),

		StatusTodo: lipgloss.NewStyle().
			Foreground(Colors.Todo).
			Width(12),

		StatusInProgress: lipgloss.NewStyle().
			Foreground(Colors.InProgress).
			Width(12),

		StatusInReview: lipgloss.NewStyle().
			Foreground(Colors.InReview).
			Width(12),

		StatusError: lipgloss.NewStyle().
			Foreground(Colors.StatusError).
			Width(12),

		StatusDone: lipgloss.NewStyle().
			Foreground(Colors.Done).
			Width(12),

		StatusClosed: lipgloss.NewStyle().
			Foreground(Colors.Closed).
			Width(12),

		// Selected status badges (brighter/bold)
		StatusTodoSelected: lipgloss.NewStyle().
			Foreground(Colors.Todo).
			Bold(true).
			Width(12),

		StatusInProgressSelected: lipgloss.NewStyle().
			Foreground(Colors.InProgress).
			Bold(true).
			Width(12),

		StatusInReviewSelected: lipgloss.NewStyle().
			Foreground(Colors.InReview).
			Bold(true).
			Width(12),

		StatusErrorSelected: lipgloss.NewStyle().
			Foreground(Colors.StatusError).
			Bold(true).
			Width(12),

		StatusDoneSelected: lipgloss.NewStyle().
			Foreground(Colors.Done).
			Bold(true).
			Width(12),

		StatusClosedSelected: lipgloss.NewStyle().
			Foreground(Colors.Closed).
			Bold(true).
			Width(12),

		Help: lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Colors.Muted),

		HelpKey: lipgloss.NewStyle().
			Foreground(Colors.Primary).
			Bold(true),

		HelpDesc: lipgloss.NewStyle().
			Foreground(Colors.Muted),

		Footer: lipgloss.NewStyle().
			Foreground(Colors.Muted),

		FooterKey: lipgloss.NewStyle().
			Foreground(Colors.Primary).
			Bold(true),

		Dialog: lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Colors.Warning),

		DialogTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(Colors.Warning),

		DialogPrompt: lipgloss.NewStyle(),

		Input: lipgloss.NewStyle().
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Colors.Primary),

		InputPrompt: lipgloss.NewStyle().
			Foreground(Colors.Primary).
			Bold(true),

		ErrorMsg: lipgloss.NewStyle().
			Foreground(Colors.Error).
			Bold(true),

		DetailTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(Colors.Primary).
			MarginBottom(1),

		DetailLabel: lipgloss.NewStyle().
			Foreground(Colors.Muted).
			Width(12),

		DetailValue: lipgloss.NewStyle(),

		DetailDesc: lipgloss.NewStyle().
			Foreground(Colors.Muted).
			MarginTop(1),
	}
}

// StatusStyle returns the style for a given status.
func (s Styles) StatusStyle(status domain.Status) lipgloss.Style {
	switch status {
	case domain.StatusTodo:
		return s.StatusTodo
	case domain.StatusInProgress:
		return s.StatusInProgress
	case domain.StatusInReview:
		return s.StatusInReview
	case domain.StatusError:
		return s.StatusError
	case domain.StatusDone:
		return s.StatusDone
	case domain.StatusClosed:
		return s.StatusClosed
	default:
		return s.StatusTodo
	}
}

// StatusStyleSelected returns the selected style for a given status.
func (s Styles) StatusStyleSelected(status domain.Status) lipgloss.Style {
	switch status {
	case domain.StatusTodo:
		return s.StatusTodoSelected
	case domain.StatusInProgress:
		return s.StatusInProgressSelected
	case domain.StatusInReview:
		return s.StatusInReviewSelected
	case domain.StatusError:
		return s.StatusErrorSelected
	case domain.StatusDone:
		return s.StatusDoneSelected
	case domain.StatusClosed:
		return s.StatusClosedSelected
	default:
		return s.StatusTodoSelected
	}
}

// StatusIcon returns an icon for a given status.
func StatusIcon(status domain.Status) string {
	switch status {
	case domain.StatusTodo:
		return "○"
	case domain.StatusInProgress:
		return "●"
	case domain.StatusInReview:
		return "◉"
	case domain.StatusError:
		return "✗"
	case domain.StatusDone:
		return "✓"
	case domain.StatusClosed:
		return "−"
	default:
		return "?"
	}
}
