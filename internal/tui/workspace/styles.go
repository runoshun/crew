package workspace

import "github.com/charmbracelet/lipgloss"

// Colors used in the workspace TUI.
var (
	ColorPrimary    = lipgloss.Color("#7C3AED") // Purple
	ColorSecondary  = lipgloss.Color("#6B7280") // Gray
	ColorSuccess    = lipgloss.Color("#10B981") // Green
	ColorWarning    = lipgloss.Color("#F59E0B") // Amber
	ColorError      = lipgloss.Color("#EF4444") // Red
	ColorBackground = lipgloss.Color("#1F2937") // Dark gray
	ColorMuted      = lipgloss.Color("#9CA3AF") // Light gray
)

// Styles holds the styles for the workspace TUI.
type Styles struct {
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	Selected    lipgloss.Style
	Normal      lipgloss.Style
	RepoName    lipgloss.Style
	RepoPath    lipgloss.Style
	StateOK     lipgloss.Style
	StateError  lipgloss.Style
	Summary     lipgloss.Style
	SummaryItem lipgloss.Style
	Loading     lipgloss.Style
	Help        lipgloss.Style
	Error       lipgloss.Style
	Dialog      lipgloss.Style
	DialogTitle lipgloss.Style
	Input       lipgloss.Style
}

// DefaultStyles returns the default styles.
func DefaultStyles() Styles {
	return Styles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1),
		Subtitle: lipgloss.NewStyle().
			Foreground(ColorMuted).
			MarginBottom(1),
		Selected: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorPrimary).
			Padding(0, 1),
		Normal: lipgloss.NewStyle().
			Padding(0, 1),
		RepoName: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")),
		RepoPath: lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true),
		StateOK: lipgloss.NewStyle().
			Foreground(ColorSuccess),
		StateError: lipgloss.NewStyle().
			Foreground(ColorError),
		Summary: lipgloss.NewStyle().
			Foreground(ColorSecondary),
		SummaryItem: lipgloss.NewStyle().
			Foreground(ColorMuted),
		Loading: lipgloss.NewStyle().
			Foreground(ColorWarning).
			Italic(true),
		Help: lipgloss.NewStyle().
			Foreground(ColorMuted).
			MarginTop(1),
		Error: lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true),
		Dialog: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2),
		DialogTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1),
		Input: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorSecondary).
			Padding(0, 1),
	}
}
