package workspace

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"
	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
	infraWorkspace "github.com/runoshun/git-crew/v2/internal/infra/workspace"
	"github.com/runoshun/git-crew/v2/internal/tui"
	"github.com/runoshun/git-crew/v2/internal/usecase"
)

// Mode represents the current UI mode.
type Mode int

const (
	ModeNormal Mode = iota
	ModeAddRepo
	ModeConfirmDelete
)

const (
	appPadding = 4
)

// Model is the workspace TUI model.
// Fields are ordered to minimize memory padding.
type Model struct {
	// Dependencies
	store *infraWorkspace.Store

	// State
	repos     []domain.WorkspaceRepo
	repoInfos map[string]domain.WorkspaceRepoInfo
	err       error

	// Components
	keys     KeyMap
	styles   Styles
	addInput textinput.Model

	// Numeric state
	cursor      int
	width       int
	height      int
	deleteIndex int // Index of repo being deleted
	mode        Mode

	// Boolean state
	loading bool
}

// New creates a new workspace TUI model.
func New() *Model {
	ai := textinput.New()
	ai.Placeholder = "Enter repository path..."
	ai.CharLimit = 500

	store, storeErr := NewStoreFromDefault()

	return &Model{
		store:     store,
		repos:     nil,
		repoInfos: make(map[string]domain.WorkspaceRepoInfo),
		err:       storeErr, // Will be displayed on first render if store creation failed
		cursor:    0,
		mode:      ModeNormal,
		keys:      DefaultKeyMap(),
		styles:    DefaultStyles(),
		addInput:  ai,
		loading:   storeErr == nil, // Don't show loading if we already have an error
	}
}

// Init initializes the model.
func (m *Model) Init() tea.Cmd {
	// Don't try to load if store creation failed
	if m.store == nil {
		return nil
	}
	return tea.Batch(
		m.loadRepos(),
	)
}

// loadRepos loads repos from the workspace file.
func (m *Model) loadRepos() tea.Cmd {
	return func() tea.Msg {
		file, err := m.store.Load()
		if err != nil {
			return MsgReposLoaded{Err: err}
		}
		return MsgReposLoaded{Repos: file.Repos}
	}
}

// loadSummary loads the task summary for a single repo.
func loadSummary(repo domain.WorkspaceRepo) tea.Cmd {
	return func() tea.Msg {
		info := loadRepoInfo(repo)
		return MsgSummaryLoaded{
			Path: repo.Path,
			Info: info,
		}
	}
}

// loadRepoInfo loads the full info for a repo.
func loadRepoInfo(repo domain.WorkspaceRepo) domain.WorkspaceRepoInfo {
	info := domain.WorkspaceRepoInfo{
		Repo: repo,
	}

	// Check if path exists
	if _, err := os.Stat(repo.Path); os.IsNotExist(err) {
		info.State = domain.RepoStateNotFound
		info.ErrorMsg = "Path does not exist"
		return info
	}

	// Check if it's a git repo by checking for .git
	gitPath := filepath.Join(repo.Path, ".git")
	if _, err := os.Stat(gitPath); os.IsNotExist(err) {
		info.State = domain.RepoStateNotGitRepo
		info.ErrorMsg = "Not a git repository"
		return info
	}

	// Check if crew is initialized
	crewPath := filepath.Join(repo.Path, ".crew")
	if _, err := os.Stat(crewPath); os.IsNotExist(err) {
		info.State = domain.RepoStateNotInitialized
		info.ErrorMsg = "crew not initialized"
		return info
	}

	// Try to load tasks
	container, err := app.New(repo.Path)
	if err != nil {
		info.State = domain.RepoStateConfigError
		info.ErrorMsg = fmt.Sprintf("Config error: %v", err)
		return info
	}

	// List tasks (exclude terminal statuses for faster loading per spec)
	out, err := container.ListTasksUseCase().Execute(context.Background(), usecase.ListTasksInput{
		IncludeTerminal: false,
	})
	if err != nil {
		info.State = domain.RepoStateLoadError
		info.ErrorMsg = fmt.Sprintf("Load error: %v", err)
		return info
	}

	info.State = domain.RepoStateOK
	info.Summary = domain.NewTaskSummary(out.Tasks)
	return info
}

// Update handles messages.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case MsgReposLoaded:
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.repos = msg.Repos
		// Start loading summaries for all repos
		return m, m.loadAllSummaries()

	case MsgSummaryLoaded:
		m.repoInfos[msg.Path] = msg.Info
		return m, nil

	case MsgRepoAdded:
		if msg.Err != nil {
			m.err = msg.Err
		}
		m.mode = ModeNormal
		m.addInput.Reset()
		return m, m.loadRepos()

	case MsgRepoRemoved:
		if msg.Err != nil {
			m.err = msg.Err
		}
		m.mode = ModeNormal
		if m.cursor >= len(m.repos)-1 && m.cursor > 0 {
			m.cursor--
		}
		return m, m.loadRepos()

	case MsgError:
		m.err = msg.Err
		return m, nil

	case MsgRepoExited:
		// Returned from repo TUI, show error if any and reload repos
		if msg.Err != nil {
			m.err = fmt.Errorf("crew tui failed: %w", msg.Err)
		}
		m.loading = true
		return m, m.loadRepos()

	case MsgTick:
		// Periodic refresh (could add auto-refresh here)
		return m, nil
	}

	return m, nil
}

// handleKey handles key events.
func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode { //nolint:exhaustive // ModeNormal handled in default
	case ModeAddRepo:
		return m.handleAddMode(msg)
	case ModeConfirmDelete:
		return m.handleDeleteMode(msg)
	default:
		return m.handleNormalMode(msg)
	}
}

// handleNormalMode handles keys in normal mode.
func (m *Model) handleNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "q" || msg.String() == "ctrl+c":
		return m, tea.Quit

	case msg.String() == "up" || msg.String() == "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case msg.String() == "down" || msg.String() == "j":
		if m.cursor < len(m.repos)-1 {
			m.cursor++
		}
		return m, nil

	case msg.String() == "pgup" || msg.String() == "ctrl+u":
		m.cursor -= 5
		if m.cursor < 0 {
			m.cursor = 0
		}
		return m, nil

	case msg.String() == "pgdown" || msg.String() == "ctrl+f":
		m.cursor += 5
		if m.cursor >= len(m.repos) {
			m.cursor = len(m.repos) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
		return m, nil

	case msg.String() == "enter":
		if len(m.repos) > 0 && m.cursor < len(m.repos) {
			repo := m.repos[m.cursor]
			// Check if repo is in a valid state
			if info, ok := m.repoInfos[repo.Path]; ok && info.State != domain.RepoStateOK {
				m.err = fmt.Errorf("cannot open: %s", info.State.String())
				return m, nil
			}
			// Update last opened and launch repo TUI
			_ = m.store.UpdateLastOpened(repo.Path)
			return m, m.openRepo(repo.Path)
		}
		return m, nil

	case msg.String() == "a":
		m.mode = ModeAddRepo
		m.addInput.Focus()
		return m, textinput.Blink

	case msg.String() == "d":
		if len(m.repos) > 0 && m.cursor < len(m.repos) {
			m.mode = ModeConfirmDelete
			m.deleteIndex = m.cursor
		}
		return m, nil

	case msg.String() == "r":
		m.loading = true
		return m, m.loadRepos()
	}

	return m, nil
}

// handleAddMode handles keys in add repo mode.
func (m *Model) handleAddMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		path := m.addInput.Value()
		if path == "" {
			m.mode = ModeNormal
			return m, nil
		}
		return m, m.addRepo(path)

	case "esc":
		m.mode = ModeNormal
		m.addInput.Reset()
		return m, nil
	}

	// Handle input
	var cmd tea.Cmd
	m.addInput, cmd = m.addInput.Update(msg)
	return m, cmd
}

// handleDeleteMode handles keys in delete confirmation mode.
func (m *Model) handleDeleteMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		if m.deleteIndex < len(m.repos) {
			path := m.repos[m.deleteIndex].Path
			return m, m.removeRepo(path)
		}
		m.mode = ModeNormal
		return m, nil

	case "n", "N", "esc", "q":
		m.mode = ModeNormal
		return m, nil
	}

	return m, nil
}

// addRepo returns a command that adds a repo.
func (m *Model) addRepo(path string) tea.Cmd {
	return func() tea.Msg {
		err := m.store.AddRepo(path)
		return MsgRepoAdded{Path: path, Err: err}
	}
}

// removeRepo returns a command that removes a repo.
func (m *Model) removeRepo(path string) tea.Cmd {
	return func() tea.Msg {
		err := m.store.RemoveRepo(path)
		return MsgRepoRemoved{Path: path, Err: err}
	}
}

// openRepo returns a command that opens a repo TUI.
func (m *Model) openRepo(path string) tea.Cmd {
	// Find the crew binary
	crewPath, err := os.Executable()
	if err != nil {
		crewPath = "crew"
	}

	cmd := exec.Command(crewPath, "tui")
	cmd.Dir = path
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		// After the repo TUI exits, signal to reload (pass error if any)
		return MsgRepoExited{Err: err}
	})
}

// loadAllSummaries returns commands to load all repo summaries.
func (m *Model) loadAllSummaries() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(m.repos))
	for _, repo := range m.repos {
		r := repo // capture
		cmds = append(cmds, loadSummary(r))
	}
	return tea.Batch(cmds...)
}

// contentWidth returns the available content width.
func (m *Model) contentWidth() int {
	w := m.width - appPadding
	if w < 0 {
		w = 0
	}
	return w
}

// View renders the TUI.
func (m *Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var base string
	switch m.mode { //nolint:exhaustive // ModeNormal handled in default
	case ModeAddRepo:
		base = m.viewAddDialog()
	case ModeConfirmDelete:
		base = m.viewDeleteDialog()
	default:
		base = m.viewMain()
	}

	return m.styles.App.Render(base)
}

// viewMain renders the main view with header, list, and footer.
func (m *Model) viewMain() string {
	// Show empty state only if no error and no repos
	if len(m.repos) == 0 && !m.loading && m.err == nil {
		return m.viewEmptyState()
	}

	var b strings.Builder

	b.WriteString(m.viewHeader())
	b.WriteString("\n")

	if m.err != nil {
		b.WriteString(m.styles.Error.Render("Error: "+m.err.Error()) + "\n\n")
	}

	b.WriteString(m.viewRepoList())
	b.WriteString("\n")
	b.WriteString(m.viewFooter())

	return b.String()
}

// viewHeader renders the header with title left-aligned and count right-aligned.
func (m *Model) viewHeader() string {
	headerStyle := m.styles.Header
	textStyle := m.styles.HeaderText
	mutedStyle := lipgloss.NewStyle().Foreground(tui.Colors.Muted)

	titleText := "Workspace"
	title := textStyle.Render(titleText)

	contentWidth := m.contentWidth()
	countText := fmt.Sprintf("%d repos", len(m.repos))

	leftLen := lipgloss.Width(title)
	rightLen := len(countText) // Plain text length before rendering
	// Header style has Padding(0, 1), so inner content width is contentWidth - 2
	innerWidth := contentWidth - 2
	spacing := innerWidth - leftLen - rightLen

	// Truncate right text if too wide (truncate plain text before rendering)
	if spacing < 1 {
		maxRightWidth := innerWidth - leftLen - 1 // 1 for minimum spacing
		if maxRightWidth <= 3 {
			countText = ""
		} else if len(countText) > maxRightWidth-3 {
			countText = countText[:maxRightWidth-3] + "..."
		}
		rightLen = len(countText)
		spacing = innerWidth - leftLen - rightLen
		if spacing < 1 {
			spacing = 1
		}
	}

	rightText := mutedStyle.Render(countText)
	content := title + strings.Repeat(" ", spacing) + rightText
	return headerStyle.Width(contentWidth).Render(content)
}

// viewRepoList renders the repo list.
func (m *Model) viewRepoList() string {
	if m.loading {
		return m.styles.Loading.Render("Loading repositories...")
	}

	// Don't show "No repositories" message if there's an error
	// (error is already displayed above)
	if len(m.repos) == 0 {
		if m.err != nil {
			return ""
		}
		return m.styles.Muted.Render("No repositories. Press 'a' to add one.")
	}

	var b strings.Builder
	for i, repo := range m.repos {
		line := m.renderRepoLine(repo, i == m.cursor)
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

// renderRepoLine renders a single repo line matching the task tree TUI style.
func (m *Model) renderRepoLine(repo domain.WorkspaceRepo, selected bool) string {
	info, hasInfo := m.repoInfos[repo.Path]

	// Cursor indicator (▸ for selected, space for normal)
	var cursor string
	if selected {
		cursor = m.styles.CursorSelected.Render("▸")
	} else {
		cursor = m.styles.CursorNormal.Render(" ")
	}

	// Name
	name := repo.DisplayName()
	var nameStr string
	if selected {
		nameStr = m.styles.RepoNameSelected.Render(name)
	} else {
		nameStr = m.styles.RepoName.Render(name)
	}

	// Path (truncated)
	pathStr := truncatePath(repo.Path, 40)
	var pathRendered string
	if selected {
		pathRendered = m.styles.RepoPathSelected.Render(pathStr)
	} else {
		pathRendered = m.styles.RepoPath.Render(pathStr)
	}

	// State and summary
	var statusStr string
	if hasInfo {
		if info.State == domain.RepoStateOK {
			statusStr = m.formatSummaryColored(info.Summary)
		} else {
			statusStr = m.styles.StateError.Render(info.State.String())
		}
	} else {
		statusStr = m.styles.Loading.Render("loading...")
	}

	// Build line with consistent spacing: "▸ name    path    status"
	line := fmt.Sprintf("%s %s  %s  %s", cursor, nameStr, pathRendered, statusStr)

	contentWidth := m.contentWidth()
	lineWidth := lipgloss.Width(line)

	// Truncate line if it exceeds content width (ANSI-aware truncation)
	if lineWidth > contentWidth && contentWidth > 0 {
		line = truncate.StringWithTail(line, uint(contentWidth), "")
		lineWidth = lipgloss.Width(line)
	}

	// Apply row background for selected items
	if selected {
		if lineWidth < contentWidth {
			// Pad to full width with background
			padding := strings.Repeat(" ", contentWidth-lineWidth)
			line = m.styles.ItemSelected.Render(line + padding)
		} else {
			line = m.styles.ItemSelected.Render(line)
		}
	}

	return line
}

// formatSummaryColored formats task summary with status-specific colors.
func (m *Model) formatSummaryColored(s domain.TaskSummary) string {
	if s.TotalActive == 0 {
		return m.styles.Summary.Render("no active tasks")
	}

	var parts []string
	if s.InProgress > 0 {
		parts = append(parts, m.styles.SummaryInProg.Render(fmt.Sprintf("prog:%d", s.InProgress)))
	}
	if s.Todo > 0 {
		parts = append(parts, m.styles.SummaryTodo.Render(fmt.Sprintf("todo:%d", s.Todo)))
	}
	if s.Done > 0 {
		parts = append(parts, m.styles.SummaryDone.Render(fmt.Sprintf("done:%d", s.Done)))
	}
	if s.Error > 0 {
		parts = append(parts, m.styles.SummaryError.Render(fmt.Sprintf("err:%d", s.Error)))
	}

	return strings.Join(parts, " ")
}

// viewFooter renders the footer with key hints.
func (m *Model) viewFooter() string {
	keyStyle := m.styles.FooterKey
	contentWidth := m.contentWidth()

	content := keyStyle.Render("j/k") + " nav  " +
		keyStyle.Render("enter") + " open  " +
		keyStyle.Render("a") + " add  " +
		keyStyle.Render("d") + " remove  " +
		keyStyle.Render("r") + " refresh  " +
		keyStyle.Render("q") + " quit"

	return m.styles.Footer.Width(contentWidth).Render(content)
}

// viewEmptyState renders a friendly empty state message.
func (m *Model) viewEmptyState() string {
	contentWidth := m.contentWidth()
	contentHeight := m.height - 2
	if contentWidth < 0 {
		contentWidth = 0
	}
	if contentHeight < 0 {
		contentHeight = 0
	}

	titleStyle := m.styles.EmptyTitle
	bodyStyle := m.styles.EmptyBody
	keyStyle := m.styles.EmptyKey
	cmdStyle := m.styles.EmptyCmd

	title := titleStyle.Render("No repositories")
	subtitle := bodyStyle.Render("Add a repository to get started")

	hint1 := lipgloss.JoinHorizontal(lipgloss.Left,
		bodyStyle.Render("Press "),
		keyStyle.Render("a"),
		bodyStyle.Render(" to add a repository"),
	)
	hint2 := lipgloss.JoinHorizontal(lipgloss.Left,
		bodyStyle.Render("Or run: "),
		cmdStyle.Render("crew workspace add /path/to/repo"),
	)

	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		subtitle,
		"",
		hint1,
		hint2,
	)

	return lipgloss.Place(contentWidth, contentHeight, lipgloss.Center, lipgloss.Center, content)
}

// viewAddDialog renders the add repo dialog.
func (m *Model) viewAddDialog() string {
	dialogWidth := m.dialogWidth()
	bg := tui.Colors.Background
	lineWidth := dialogWidth - 4
	if lineWidth < 0 {
		lineWidth = 0
	}
	lineStyle := lipgloss.NewStyle().Background(bg).Width(lineWidth)

	title := lineStyle.Render(m.styles.DialogTitle.Render("Add Repository"))
	emptyLine := lineStyle.Render("")
	label := lineStyle.Render(m.styles.DialogText.Render("Enter the path to a git repository:"))
	input := lineStyle.Render(m.styles.Input.Render(m.addInput.View()))
	hint := lineStyle.Render(
		m.styles.DialogKey.Render("enter") + m.styles.DialogText.Render(" add  ") +
			m.styles.DialogKey.Render("esc") + m.styles.DialogText.Render(" cancel"))

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		emptyLine,
		label,
		input,
		emptyLine,
		hint,
	)

	return m.styles.Dialog.Width(dialogWidth).Render(content)
}

// dialogWidth returns the width for dialogs.
func (m *Model) dialogWidth() int {
	contentWidth := m.contentWidth()
	width := contentWidth - 10
	if width > 80 {
		width = 80
	}
	// Minimum width for usability, but never exceed content width
	minWidth := 20
	if minWidth > contentWidth {
		minWidth = contentWidth
	}
	if width < minWidth {
		width = minWidth
	}
	// Final clamp to content width
	if width > contentWidth {
		width = contentWidth
	}
	return width
}

// viewDeleteDialog renders the delete confirmation dialog.
func (m *Model) viewDeleteDialog() string {
	if m.deleteIndex >= len(m.repos) {
		return ""
	}
	repo := m.repos[m.deleteIndex]

	dialogWidth := m.dialogWidth()
	bg := tui.Colors.Background
	lineWidth := dialogWidth - 4
	if lineWidth < 0 {
		lineWidth = 0
	}
	lineStyle := lipgloss.NewStyle().Background(bg).Width(lineWidth)
	titleColor := lipgloss.NewStyle().Background(bg).Foreground(tui.Colors.Error).Bold(true)

	title := lineStyle.Render(titleColor.Render("Remove Repository?"))
	emptyLine := lineStyle.Render("")
	repoLine := lineStyle.Render(m.styles.DialogText.Render(repo.DisplayName()))
	pathLine := lineStyle.Render(m.styles.DialogMuted.Render(repo.Path))
	warnLine := lineStyle.Render(m.styles.DialogText.Render("This will remove the repository from the workspace list."))
	hint := lineStyle.Render(
		m.styles.DialogKey.Render("[ y ]") + m.styles.DialogText.Render(" Confirm  ") +
			m.styles.DialogMuted.Render("[ n ]") + m.styles.DialogMuted.Render(" Cancel"))

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		repoLine,
		pathLine,
		emptyLine,
		warnLine,
		emptyLine,
		hint,
	)

	return m.styles.Dialog.Width(dialogWidth).Render(content)
}

// truncatePath truncates a path, keeping the end.
func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	return "..." + path[len(path)-maxLen+3:]
}
