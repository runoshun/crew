package workspace

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/runoshun/git-crew/v2/internal/tui"
)

// Styles holds the styles for the workspace TUI.
// Uses the same color palette as the task tree TUI for visual consistency.
type Styles struct {
	// App
	App lipgloss.Style

	// Header
	Header     lipgloss.Style
	HeaderText lipgloss.Style

	// List items
	ItemSelected lipgloss.Style

	// Cursor
	CursorNormal   lipgloss.Style
	CursorSelected lipgloss.Style

	// Repo display
	RepoName         lipgloss.Style
	RepoNameSelected lipgloss.Style
	RepoPath         lipgloss.Style
	RepoPathSelected lipgloss.Style

	// State indicators
	StateOK    lipgloss.Style
	StateError lipgloss.Style

	// Summary
	Summary       lipgloss.Style
	SummaryInProg lipgloss.Style
	SummaryTodo   lipgloss.Style
	SummaryDone   lipgloss.Style
	SummaryError  lipgloss.Style

	// UI elements
	Loading lipgloss.Style
	Error   lipgloss.Style
	Muted   lipgloss.Style

	// Footer
	Footer    lipgloss.Style
	FooterKey lipgloss.Style

	// Dialog
	Dialog      lipgloss.Style
	DialogTitle lipgloss.Style
	DialogText  lipgloss.Style
	DialogKey   lipgloss.Style
	DialogMuted lipgloss.Style
	Input       lipgloss.Style

	// Empty state
	EmptyTitle lipgloss.Style
	EmptyBody  lipgloss.Style
	EmptyKey   lipgloss.Style
	EmptyCmd   lipgloss.Style
}

// DefaultStyles returns the default styles using the unified color palette.
func DefaultStyles() Styles {
	colors := tui.Colors

	return Styles{
		App: lipgloss.NewStyle().
			Padding(1, 2),

		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(colors.Primary).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(colors.GroupLine).
			Padding(0, 1).
			MarginBottom(1),

		HeaderText: lipgloss.NewStyle().
			Bold(true).
			Foreground(colors.Primary),

		ItemSelected: lipgloss.NewStyle().
			Background(colors.SelectionBg),

		CursorNormal: lipgloss.NewStyle().
			Foreground(colors.Background).
			MarginRight(0),

		CursorSelected: lipgloss.NewStyle().
			Foreground(colors.Primary).
			Bold(true).
			MarginRight(0),

		RepoName: lipgloss.NewStyle().
			Foreground(colors.TitleNormal),

		RepoNameSelected: lipgloss.NewStyle().
			Foreground(colors.TitleSelected).
			Bold(true),

		RepoPath: lipgloss.NewStyle().
			Foreground(colors.DescNormal).
			Italic(true),

		RepoPathSelected: lipgloss.NewStyle().
			Foreground(colors.DescSelected).
			Italic(true),

		StateOK: lipgloss.NewStyle().
			Foreground(colors.Success),

		StateError: lipgloss.NewStyle().
			Foreground(colors.Error),

		Summary: lipgloss.NewStyle().
			Foreground(colors.Muted),

		SummaryInProg: lipgloss.NewStyle().
			Foreground(colors.InProgress),

		SummaryTodo: lipgloss.NewStyle().
			Foreground(colors.Todo),

		SummaryDone: lipgloss.NewStyle().
			Foreground(colors.Done),

		SummaryError: lipgloss.NewStyle().
			Foreground(colors.StatusError),

		Loading: lipgloss.NewStyle().
			Foreground(colors.Warning).
			Italic(true),

		Error: lipgloss.NewStyle().
			Foreground(colors.Error).
			Bold(true),

		Muted: lipgloss.NewStyle().
			Foreground(colors.Muted),

		Footer: lipgloss.NewStyle().
			Foreground(colors.DescNormal).
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(colors.GroupLine).
			Padding(0, 1).
			MarginTop(1),

		FooterKey: lipgloss.NewStyle().
			Foreground(colors.KeyText).
			Background(colors.GroupLine).
			Padding(0, 1),

		Dialog: lipgloss.NewStyle().
			Background(colors.Background).
			Padding(1, 2),

		DialogTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(colors.Primary).
			Background(colors.Background),

		DialogText: lipgloss.NewStyle().
			Foreground(colors.TitleNormal).
			Background(colors.Background),

		DialogKey: lipgloss.NewStyle().
			Foreground(colors.KeyText).
			Background(colors.Background).
			Bold(true),

		DialogMuted: lipgloss.NewStyle().
			Foreground(colors.Muted).
			Background(colors.Background),

		Input: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colors.Primary).
			Padding(0, 1),

		EmptyTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(colors.Primary),

		EmptyBody: lipgloss.NewStyle().
			Foreground(colors.Muted),

		EmptyKey: lipgloss.NewStyle().
			Foreground(colors.Maroon).
			Bold(true),

		EmptyCmd: lipgloss.NewStyle().
			Foreground(colors.Primary),
	}
}
