package workspace

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
	infraWorkspace "github.com/runoshun/git-crew/v2/internal/infra/workspace"
	"github.com/runoshun/git-crew/v2/internal/usecase"
)

// Mode represents the current UI mode.
type Mode int

const (
	ModeNormal Mode = iota
	ModeAddRepo
	ModeConfirmDelete
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
	help     help.Model
	addInput textinput.Model

	// Numeric state
	cursor      int
	width       int
	height      int
	deleteIndex int // Index of repo being deleted
	mode        Mode

	// Boolean state
	showHelp bool
	loading  bool
}

// New creates a new workspace TUI model.
func New() *Model {
	ai := textinput.New()
	ai.Placeholder = "Enter repository path..."
	ai.CharLimit = 500

	return &Model{
		store:     NewStoreFromDefault(),
		repos:     nil,
		repoInfos: make(map[string]domain.WorkspaceRepoInfo),
		cursor:    0,
		mode:      ModeNormal,
		keys:      DefaultKeyMap(),
		styles:    DefaultStyles(),
		help:      help.New(),
		addInput:  ai,
		loading:   true,
	}
}

// Init initializes the model.
func (m *Model) Init() tea.Cmd {
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

	// List tasks
	out, err := container.ListTasksUseCase().Execute(context.Background(), usecase.ListTasksInput{
		IncludeTerminal: true, // Include all tasks for summary
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
		m.help.Width = msg.Width
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
		// Returned from repo TUI, reload repos and summaries
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
	// Clear error on any key
	m.err = nil

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

	case msg.String() == "pgdown" || msg.String() == "ctrl+d":
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

	case msg.String() == "?":
		m.showHelp = !m.showHelp
		return m, nil
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
		// After the repo TUI exits, signal to reload
		return MsgRepoExited{}
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

// View renders the TUI.
func (m *Model) View() string {
	if m.width == 0 {
		return ""
	}

	var b strings.Builder

	// Title
	b.WriteString(m.styles.Title.Render("crew workspace"))
	b.WriteString("\n")
	b.WriteString(m.styles.Subtitle.Render(fmt.Sprintf("%d repositories", len(m.repos))))
	b.WriteString("\n\n")

	// Content based on mode
	switch m.mode { //nolint:exhaustive // ModeNormal handled in default
	case ModeAddRepo:
		b.WriteString(m.viewAddDialog())
	case ModeConfirmDelete:
		b.WriteString(m.viewDeleteDialog())
	default:
		b.WriteString(m.viewRepoList())
	}

	// Error message
	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(m.styles.Error.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	}

	// Help
	if m.showHelp {
		b.WriteString("\n")
		b.WriteString(m.help.View(m.keys))
	} else {
		b.WriteString("\n")
		b.WriteString(m.styles.Help.Render("Press ? for help"))
	}

	return b.String()
}

// viewRepoList renders the repo list.
func (m *Model) viewRepoList() string {
	if m.loading {
		return m.styles.Loading.Render("Loading repositories...")
	}

	if len(m.repos) == 0 {
		return m.styles.Subtitle.Render("No repositories. Press 'a' to add one.")
	}

	var b strings.Builder
	for i, repo := range m.repos {
		line := m.renderRepoLine(repo, i == m.cursor)
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

// renderRepoLine renders a single repo line.
func (m *Model) renderRepoLine(repo domain.WorkspaceRepo, selected bool) string {
	info, hasInfo := m.repoInfos[repo.Path]

	// Build the line content
	var parts []string

	// Name
	name := repo.DisplayName()
	if selected {
		parts = append(parts, m.styles.Selected.Render(name))
	} else {
		parts = append(parts, m.styles.RepoName.Render(name))
	}

	// Path (truncated)
	pathStr := truncatePath(repo.Path, 40)
	parts = append(parts, m.styles.RepoPath.Render(pathStr))

	// State and summary
	if hasInfo {
		if info.State == domain.RepoStateOK {
			parts = append(parts, m.styles.StateOK.Render("OK"))
			// Task summary
			summary := formatSummary(info.Summary)
			parts = append(parts, m.styles.Summary.Render(summary))
		} else {
			parts = append(parts, m.styles.StateError.Render(info.State.String()))
		}
	} else {
		parts = append(parts, m.styles.Loading.Render("loading..."))
	}

	return strings.Join(parts, "  ")
}

// viewAddDialog renders the add repo dialog.
func (m *Model) viewAddDialog() string {
	var b strings.Builder
	b.WriteString(m.styles.DialogTitle.Render("Add Repository"))
	b.WriteString("\n\n")
	b.WriteString("Enter the path to a git repository:\n\n")
	b.WriteString(m.styles.Input.Render(m.addInput.View()))
	b.WriteString("\n\n")
	b.WriteString(m.styles.Help.Render("Press Enter to add, Esc to cancel"))
	return m.styles.Dialog.Render(b.String())
}

// viewDeleteDialog renders the delete confirmation dialog.
func (m *Model) viewDeleteDialog() string {
	if m.deleteIndex >= len(m.repos) {
		return ""
	}
	repo := m.repos[m.deleteIndex]

	var b strings.Builder
	b.WriteString(m.styles.DialogTitle.Render("Remove Repository"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Remove %s from workspace?\n", m.styles.RepoName.Render(repo.DisplayName())))
	b.WriteString(m.styles.RepoPath.Render(repo.Path))
	b.WriteString("\n\n")
	b.WriteString(m.styles.Help.Render("Press Y to confirm, N or Esc to cancel"))
	return m.styles.Dialog.Render(b.String())
}

// truncatePath truncates a path, keeping the end.
func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	return "..." + path[len(path)-maxLen+3:]
}

// formatSummary formats a task summary for display.
func formatSummary(s domain.TaskSummary) string {
	if s.TotalActive == 0 {
		return "no active tasks"
	}

	var parts []string
	if s.InProgress > 0 {
		parts = append(parts, fmt.Sprintf("prog:%d", s.InProgress))
	}
	if s.NeedsInput > 0 {
		parts = append(parts, fmt.Sprintf("input:%d", s.NeedsInput))
	}
	if s.Todo > 0 {
		parts = append(parts, fmt.Sprintf("todo:%d", s.Todo))
	}
	if s.ForReview > 0 || s.Reviewing > 0 || s.Reviewed > 0 {
		parts = append(parts, fmt.Sprintf("review:%d", s.ForReview+s.Reviewing+s.Reviewed))
	}
	if s.Error > 0 || s.Stopped > 0 {
		parts = append(parts, fmt.Sprintf("err:%d", s.Error+s.Stopped))
	}

	return strings.Join(parts, " ")
}
