package tui

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"
	"github.com/runoshun/git-crew/v2/internal/domain"
)

// Colors defines the color palette for the TUI.
// Designed with a "Modern Dark" aesthetic for a commercial-grade look.
var Colors = struct {
	// Base colors
	Primary    lipgloss.Color
	Secondary  lipgloss.Color
	Muted      lipgloss.Color
	Subtle     lipgloss.Color // New: Even more muted for unobtrusive elements
	Error      lipgloss.Color
	Success    lipgloss.Color
	Warning    lipgloss.Color
	Background lipgloss.Color

	// Title/text colors
	TitleNormal   lipgloss.Color
	TitleSelected lipgloss.Color
	DescNormal    lipgloss.Color
	DescSelected  lipgloss.Color
	KeyText       lipgloss.Color

	// Status colors
	Todo        lipgloss.Color
	InProgress  lipgloss.Color
	NeedsInput  lipgloss.Color
	ForReview   lipgloss.Color
	Reviewing   lipgloss.Color
	Reviewed    lipgloss.Color
	Stopped     lipgloss.Color
	StatusError lipgloss.Color
	Closed      lipgloss.Color

	// Group header
	GroupLine lipgloss.Color
}{
	// Modern Dark Palette (Catppuccin-inspired - Modern Soft Variant)
	Primary:    lipgloss.Color("#89B4FA"), // Blue
	Secondary:  lipgloss.Color("#CBA6F7"), // Mauve
	Muted:      lipgloss.Color("#A6ADC8"), // Subtext1
	Subtle:     lipgloss.Color("#585B70"), // Surface2
	Error:      lipgloss.Color("#F38BA8"), // Red
	Success:    lipgloss.Color("#A6E3A1"), // Green
	Warning:    lipgloss.Color("#F9E2AF"), // Yellow
	Background: lipgloss.Color("#1E1E2E"), // Base

	// Text colors
	TitleNormal:   lipgloss.Color("#CDD6F4"), // Text
	TitleSelected: lipgloss.Color("#CDD6F4"), // Text (keep white on selection for contrast)
	DescNormal:    lipgloss.Color("#6C7086"), // Overlay0
	DescSelected:  lipgloss.Color("#A6ADC8"), // Subtext1 (brighter than normal)
	KeyText:       lipgloss.Color("#B4BEFE"), // Lavender (for keys)

	// Status colors
	Todo:        lipgloss.Color("#89B4FA"), // Blue (pending)
	InProgress:  lipgloss.Color("#F9E2AF"), // Yellow (active work)
	NeedsInput:  lipgloss.Color("#94E2D5"), // Teal (waiting for input)
	ForReview:   lipgloss.Color("#CBA6F7"), // Mauve (awaiting review)
	Reviewing:   lipgloss.Color("#F9E2AF"), // Yellow (review in progress)
	Reviewed:    lipgloss.Color("#A6E3A1"), // Green (review complete)
	Stopped:     lipgloss.Color("#FAB387"), // Peach/Orange
	StatusError: lipgloss.Color("#F38BA8"), // Red
	Closed:      lipgloss.Color("#6C7086"), // Overlay0

	// Group header / UI Elements
	GroupLine: lipgloss.Color("#313244"), // Surface0
}

// Styles contains all the lipgloss styles for the TUI.
type Styles struct {
	// App
	App lipgloss.Style

	// Header
	Header     lipgloss.Style
	HeaderText lipgloss.Style

	// Task list
	TaskList           lipgloss.Style
	TaskItem           lipgloss.Style
	TaskNormal         lipgloss.Style
	TaskSelected       lipgloss.Style
	SelectionIndicator lipgloss.Style
	TaskID             lipgloss.Style
	TaskIDSelected     lipgloss.Style
	TaskTitle          lipgloss.Style
	TaskTitleSelected  lipgloss.Style
	TaskDesc           lipgloss.Style
	TaskDescSelected   lipgloss.Style
	TaskAgent          lipgloss.Style
	TaskAgentSelected  lipgloss.Style
	TaskLabel          lipgloss.Style
	TaskLabelSelected  lipgloss.Style
	CursorNormal       lipgloss.Style
	CursorSelected     lipgloss.Style

	// Group header
	GroupHeaderLine  lipgloss.Style
	GroupHeaderLabel lipgloss.Style

	// Status badges (normal)
	StatusTodo       lipgloss.Style
	StatusInProgress lipgloss.Style
	StatusNeedsInput lipgloss.Style
	StatusForReview  lipgloss.Style
	StatusReviewing  lipgloss.Style
	StatusReviewed   lipgloss.Style
	StatusStopped    lipgloss.Style
	StatusError      lipgloss.Style
	StatusClosed     lipgloss.Style

	// Status badges (selected - brighter)
	StatusTodoSelected       lipgloss.Style
	StatusInProgressSelected lipgloss.Style
	StatusNeedsInputSelected lipgloss.Style
	StatusForReviewSelected  lipgloss.Style
	StatusReviewingSelected  lipgloss.Style
	StatusReviewedSelected   lipgloss.Style
	StatusStoppedSelected    lipgloss.Style
	StatusErrorSelected      lipgloss.Style
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
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(Colors.GroupLine).
			Padding(0, 1).
			MarginBottom(1),

		HeaderText: lipgloss.NewStyle().
			Bold(true).
			Foreground(Colors.Primary),

		TaskList: lipgloss.NewStyle().
			MarginBottom(1),

		TaskItem: lipgloss.NewStyle().
			PaddingLeft(1).
			PaddingRight(1),

		TaskNormal: lipgloss.NewStyle().
			Foreground(Colors.TitleNormal),

		TaskSelected: lipgloss.NewStyle().
			Background(lipgloss.Color("#262637")), // Very subtle highlight, no foreground override

		SelectionIndicator: lipgloss.NewStyle().
			Foreground(Colors.Primary),

		TaskID: lipgloss.NewStyle().
			Foreground(Colors.Muted).
			Width(3).
			MarginRight(1),

		TaskIDSelected: lipgloss.NewStyle().
			Foreground(Colors.Primary).
			Bold(true).
			Width(3).
			MarginRight(1),

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
			Italic(true).
			MarginLeft(1),

		TaskAgentSelected: lipgloss.NewStyle().
			Foreground(Colors.Secondary).
			Italic(true).
			MarginLeft(1),

		TaskLabel: lipgloss.NewStyle().
			Foreground(Colors.Secondary).
			MarginRight(1),

		TaskLabelSelected: lipgloss.NewStyle().
			Foreground(Colors.Secondary).
			Bold(true).
			MarginRight(1),

		CursorNormal: lipgloss.NewStyle().
			Foreground(Colors.Background). // Hide cursor in normal mode (matches bg)
			MarginRight(0),

		CursorSelected: lipgloss.NewStyle().
			Foreground(Colors.Primary).
			Bold(true).
			MarginRight(0),

		// Group header styles
		GroupHeaderLine: lipgloss.NewStyle().
			Foreground(Colors.GroupLine),

		GroupHeaderLabel: lipgloss.NewStyle().
			Foreground(Colors.Muted).
			Bold(true),

		StatusTodo: lipgloss.NewStyle().
			Foreground(Colors.Todo),

		StatusInProgress: lipgloss.NewStyle().
			Foreground(Colors.InProgress),

		StatusNeedsInput: lipgloss.NewStyle().
			Foreground(Colors.NeedsInput),

		StatusForReview: lipgloss.NewStyle().
			Foreground(Colors.ForReview),

		StatusReviewing: lipgloss.NewStyle().
			Foreground(Colors.Reviewing),

		StatusReviewed: lipgloss.NewStyle().
			Foreground(Colors.Reviewed),

		StatusStopped: lipgloss.NewStyle().
			Foreground(Colors.Stopped),

		StatusError: lipgloss.NewStyle().
			Foreground(Colors.StatusError),

		StatusClosed: lipgloss.NewStyle().
			Foreground(Colors.Closed),

		// Selected status badges (brighter/bold)
		StatusTodoSelected: lipgloss.NewStyle().
			Foreground(Colors.Todo).
			Bold(true),

		StatusInProgressSelected: lipgloss.NewStyle().
			Foreground(Colors.InProgress).
			Bold(true),

		StatusNeedsInputSelected: lipgloss.NewStyle().
			Foreground(Colors.NeedsInput).
			Bold(true),

		StatusForReviewSelected: lipgloss.NewStyle().
			Foreground(Colors.ForReview).
			Bold(true),

		StatusReviewingSelected: lipgloss.NewStyle().
			Foreground(Colors.Reviewing).
			Bold(true),

		StatusReviewedSelected: lipgloss.NewStyle().
			Foreground(Colors.Reviewed).
			Bold(true),

		StatusStoppedSelected: lipgloss.NewStyle().
			Foreground(Colors.Stopped).
			Bold(true),

		StatusErrorSelected: lipgloss.NewStyle().
			Foreground(Colors.StatusError).
			Bold(true),

		StatusClosedSelected: lipgloss.NewStyle().
			Foreground(Colors.Closed).
			Bold(true),

		Help: lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Colors.Muted),

		HelpKey: lipgloss.NewStyle().
			Foreground(Colors.Muted).
			Bold(true),

		HelpDesc: lipgloss.NewStyle().
			Foreground(Colors.Subtle),

		Footer: lipgloss.NewStyle().
			Foreground(Colors.DescNormal).
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(Colors.GroupLine).
			Padding(0, 1).
			MarginTop(1),

		FooterKey: lipgloss.NewStyle().
			Foreground(Colors.KeyText).
			Background(Colors.GroupLine).
			Padding(0, 1),

		// Pagination dots
		PaginationDot: lipgloss.NewStyle().
			Foreground(Colors.GroupLine),

		PaginationDotActive: lipgloss.NewStyle().
			Foreground(Colors.Primary).
			Bold(true),

		Dialog: lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Colors.Muted).
			MarginTop(1),

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
	case domain.StatusNeedsInput:
		return s.StatusNeedsInput
	case domain.StatusForReview:
		return s.StatusForReview
	case domain.StatusReviewing:
		return s.StatusReviewing
	case domain.StatusReviewed:
		return s.StatusReviewed
	case domain.StatusStopped:
		return s.StatusStopped
	case domain.StatusError:
		return s.StatusError
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
	case domain.StatusNeedsInput:
		return s.StatusNeedsInputSelected
	case domain.StatusForReview:
		return s.StatusForReviewSelected
	case domain.StatusReviewing:
		return s.StatusReviewingSelected
	case domain.StatusReviewed:
		return s.StatusReviewedSelected
	case domain.StatusStopped:
		return s.StatusStoppedSelected
	case domain.StatusError:
		return s.StatusErrorSelected
	case domain.StatusClosed:
		return s.StatusClosedSelected
	default:
		return s.StatusTodoSelected
	}
}

// StatusText returns a short text label for a given status.
func StatusText(status domain.Status) string {
	switch status {
	case domain.StatusTodo:
		return "Todo"
	case domain.StatusInProgress:
		return "InPrg"
	case domain.StatusNeedsInput:
		return "Input"
	case domain.StatusForReview:
		return "ForRev"
	case domain.StatusReviewing:
		return "Revwg"
	case domain.StatusReviewed:
		return "Revwd"
	case domain.StatusStopped:
		return "Stop"
	case domain.StatusError:
		return "Err"
	case domain.StatusClosed:
		return "Clsd"
	default:
		return "?"
	}
}

// StatusIcon returns an icon for a given status.
func StatusIcon(status domain.Status) string {
	switch status {
	case domain.StatusTodo:
		return "●"
	case domain.StatusInProgress:
		return "➜"
	case domain.StatusNeedsInput:
		return "?"
	case domain.StatusForReview:
		return "◎"
	case domain.StatusReviewing:
		return "⟳"
	case domain.StatusReviewed:
		return "✔"
	case domain.StatusStopped:
		return "⏸"
	case domain.StatusError:
		return "✕"
	case domain.StatusClosed:
		return "−"
	default:
		return "?"
	}
}

// RenderMarkdown renders markdown text with the given width.
func (s Styles) RenderMarkdown(text string, width int) string {
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(s.markdownStyle()),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return text
	}

	out, err := r.Render(text)
	if err != nil {
		return text
	}

	return strings.TrimSpace(out)
}

func (s Styles) markdownStyle() ansi.StyleConfig {
	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: stringPtr(string(Colors.TitleNormal)),
			},
		},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: stringPtr(string(Colors.Primary)),
				Bold:  boolPtr(true),
			},
		},
		Code: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: stringPtr(string(Colors.TitleSelected)),
			},
		},
		CodeBlock: ansi.StyleCodeBlock{
			Theme: "catppuccin-mocha",
		},
		Table: ansi.StyleTable{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					Color: stringPtr(string(Colors.TitleNormal)),
				},
			},
		},
		Link: ansi.StylePrimitive{
			Color:     stringPtr(string(Colors.Primary)),
			Underline: boolPtr(true),
		},
		LinkText: ansi.StylePrimitive{
			Color: stringPtr(string(Colors.Primary)),
			Bold:  boolPtr(true),
		},
		List: ansi.StyleList{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					Color: stringPtr(string(Colors.TitleNormal)),
				},
			},
		},
		Item: ansi.StylePrimitive{
			Color: stringPtr(string(Colors.TitleNormal)),
		},
		BlockQuote: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: stringPtr(string(Colors.Muted)),
			},
		},
		HorizontalRule: ansi.StylePrimitive{
			Color: stringPtr(string(Colors.GroupLine)),
		},
		Strong: ansi.StylePrimitive{
			Bold: boolPtr(true),
		},
		Emph: ansi.StylePrimitive{
			Italic: boolPtr(true),
		},
	}
}

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
