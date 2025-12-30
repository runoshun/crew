package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/runoshun/git-crew/v2/internal/domain"
)

// Colors defines the color palette for the TUI (v1-style Hex colors).
var Colors = struct {
	// Base colors
	Primary    lipgloss.Color
	Secondary  lipgloss.Color
	Muted      lipgloss.Color
	Error      lipgloss.Color
	Success    lipgloss.Color
	Warning    lipgloss.Color
	Background lipgloss.Color

	// Title/text colors
	TitleNormal   lipgloss.Color
	TitleSelected lipgloss.Color
	DescNormal    lipgloss.Color
	DescSelected  lipgloss.Color

	// Status colors
	Todo        lipgloss.Color
	InProgress  lipgloss.Color
	InReview    lipgloss.Color
	StatusError lipgloss.Color
	Done        lipgloss.Color
	Closed      lipgloss.Color

	// Group header
	GroupLine lipgloss.Color
}{
	// v1-style Hex color palette
	Primary:    lipgloss.Color("#6C5CE7"), // Purple
	Secondary:  lipgloss.Color("#A29BFE"), // Lavender
	Muted:      lipgloss.Color("#636E72"), // Gray
	Error:      lipgloss.Color("#D63031"), // Red
	Success:    lipgloss.Color("#00B894"), // Green
	Warning:    lipgloss.Color("#FDCB6E"), // Yellow
	Background: lipgloss.Color("#2D3436"), // Dark gray

	// v1-style title colors
	TitleNormal:   lipgloss.Color("#DFE6E9"), // Light gray
	TitleSelected: lipgloss.Color("#FFEAA7"), // Yellow (selected)
	DescNormal:    lipgloss.Color("#636E72"), // Gray
	DescSelected:  lipgloss.Color("#B2BEC3"), // Light gray

	// v1-style status colors
	Todo:        lipgloss.Color("#74B9FF"), // Light blue
	InProgress:  lipgloss.Color("#FDCB6E"), // Yellow
	InReview:    lipgloss.Color("#A29BFE"), // Lavender
	StatusError: lipgloss.Color("#D63031"), // Red
	Done:        lipgloss.Color("#00B894"), // Green
	Closed:      lipgloss.Color("#636E72"), // Gray

	// Group header line color
	GroupLine: lipgloss.Color("#636E72"),
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
	TaskNormal        lipgloss.Style
	TaskSelected      lipgloss.Style
	TaskID            lipgloss.Style
	TaskIDSelected    lipgloss.Style
	TaskTitle         lipgloss.Style
	TaskTitleSelected lipgloss.Style
	TaskDesc          lipgloss.Style
	TaskDescSelected  lipgloss.Style
	TaskAgent         lipgloss.Style
	TaskAgentSelected lipgloss.Style
	CursorNormal      lipgloss.Style
	CursorSelected    lipgloss.Style

	// Group header
	GroupHeaderLine  lipgloss.Style
	GroupHeaderLabel lipgloss.Style

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

	// Pagination
	PaginationDot       lipgloss.Style
	PaginationDotActive lipgloss.Style

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

		TaskNormal: lipgloss.NewStyle().
			Foreground(Colors.TitleNormal),

		TaskSelected: lipgloss.NewStyle().
			Bold(true).
			Foreground(Colors.TitleSelected),

		TaskID: lipgloss.NewStyle().
			Foreground(Colors.Muted).
			Width(5),

		TaskIDSelected: lipgloss.NewStyle().
			Foreground(Colors.TitleSelected).
			Bold(true).
			Width(5),

		TaskTitle: lipgloss.NewStyle().
			Foreground(Colors.TitleNormal),

		TaskTitleSelected: lipgloss.NewStyle().
			Foreground(Colors.TitleSelected).
			Bold(true),

		TaskDesc: lipgloss.NewStyle().
			Foreground(Colors.DescNormal),

		TaskDescSelected: lipgloss.NewStyle().
			Foreground(Colors.DescSelected),

		TaskAgent: lipgloss.NewStyle().
			Foreground(Colors.Secondary).
			Italic(true),

		TaskAgentSelected: lipgloss.NewStyle().
			Foreground(Colors.TitleSelected).
			Italic(true),

		CursorNormal: lipgloss.NewStyle().
			Foreground(Colors.Muted),

		CursorSelected: lipgloss.NewStyle().
			Foreground(Colors.TitleSelected).
			Bold(true),

		// Group header styles (v1-style)
		GroupHeaderLine: lipgloss.NewStyle().
			Foreground(Colors.GroupLine),

		GroupHeaderLabel: lipgloss.NewStyle().
			Foreground(Colors.Muted),

		StatusTodo: lipgloss.NewStyle().
			Foreground(Colors.Todo),

		StatusInProgress: lipgloss.NewStyle().
			Foreground(Colors.InProgress),

		StatusInReview: lipgloss.NewStyle().
			Foreground(Colors.InReview),

		StatusError: lipgloss.NewStyle().
			Foreground(Colors.StatusError),

		StatusDone: lipgloss.NewStyle().
			Foreground(Colors.Done),

		StatusClosed: lipgloss.NewStyle().
			Foreground(Colors.Closed),

		// Selected status badges (brighter/bold)
		StatusTodoSelected: lipgloss.NewStyle().
			Foreground(Colors.Todo).
			Bold(true),

		StatusInProgressSelected: lipgloss.NewStyle().
			Foreground(Colors.InProgress).
			Bold(true),

		StatusInReviewSelected: lipgloss.NewStyle().
			Foreground(Colors.InReview).
			Bold(true),

		StatusErrorSelected: lipgloss.NewStyle().
			Foreground(Colors.StatusError).
			Bold(true),

		StatusDoneSelected: lipgloss.NewStyle().
			Foreground(Colors.Done).
			Bold(true),

		StatusClosedSelected: lipgloss.NewStyle().
			Foreground(Colors.Closed).
			Bold(true),

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

		// Pagination dots (v1-style)
		PaginationDot: lipgloss.NewStyle().
			Foreground(Colors.Muted),

		PaginationDotActive: lipgloss.NewStyle().
			Foreground(Colors.TitleSelected).
			Bold(true),

		Dialog: lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Colors.Primary),

		DialogTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(Colors.Primary),

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
