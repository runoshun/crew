package workspace

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	appPadding    = 4
	minSplitWidth = 120
	minPaneWidth  = 24
	leftPaneRatio = 0.3
)

// Model is the workspace TUI model.
// Fields are ordered to minimize memory padding.
type Model struct {
	// Dependencies
	store *infraWorkspace.Store

	// Repo models
	models     map[string]*tui.Model
	activeRepo string

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
	leftWidth   int
	rightWidth  int

	// Boolean state
	leftFocused bool
	loading     bool
}

// New creates a new workspace TUI model.
func New() *Model {
	ai := textinput.New()
	ai.Placeholder = "Enter repository path..."
	ai.CharLimit = 500

	store, storeErr := NewStoreFromDefault()

	return &Model{
		store:       store,
		models:      make(map[string]*tui.Model),
		activeRepo:  "",
		repos:       nil,
		repoInfos:   make(map[string]domain.WorkspaceRepoInfo),
		err:         storeErr, // Will be displayed on first render if store creation failed
		cursor:      0,
		mode:        ModeNormal,
		keys:        DefaultKeyMap(),
		styles:      DefaultStyles(),
		addInput:    ai,
		leftFocused: true,
		loading:     storeErr == nil, // Don't show loading if we already have an error
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
		m.tick(),
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

func (m *Model) tick() tea.Cmd {
	return tea.Tick(tui.AutoRefreshInterval, func(t time.Time) tea.Msg {
		return MsgTick{}
	})
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
	var warnBuf bytes.Buffer
	container, err := app.NewWithWarningWriter(repo.Path, &warnBuf)
	if err != nil {
		info.State = domain.RepoStateConfigError
		info.ErrorMsg = fmt.Sprintf("Config error: %v", err)
		return info
	}
	if warnBuf.Len() > 0 {
		info.WarningMsg = strings.TrimSpace(warnBuf.String())
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
		m.updateLayout()
		return m, m.propagateWindowSize()

	case MsgReposLoaded:
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.repos = msg.Repos
		m.pruneRepoState()
		m.clampCursor()
		m.syncActiveRepo()
		cmds := []tea.Cmd{m.loadAllSummaries()}
		if cmd := m.ensureActiveModel(); cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case MsgSummaryLoaded:
		m.repoInfos[msg.Path] = msg.Info
		if msg.Path == m.activeRepo {
			return m, m.ensureActiveModel()
		}
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
		} else {
			delete(m.models, msg.Path)
			delete(m.repoInfos, msg.Path)
			if msg.Path == m.activeRepo {
				m.activeRepo = ""
				m.leftFocused = true
			}
		}
		m.mode = ModeNormal
		if m.cursor >= len(m.repos)-1 && m.cursor > 0 {
			m.cursor--
		}
		return m, m.loadRepos()

	case MsgError:
		m.err = msg.Err
		return m, nil

	case tui.MsgFocusWorkspace:
		m.leftFocused = true
		return m, nil

	case RepoMsg:
		return m.routeRepoMsg(msg)

	case tui.MsgTick:
		return m.forwardToActiveModel(msg)

	case MsgTick:
		updated, cmd := m.forwardToActiveModel(tui.MsgTick{})
		return updated, tea.Batch(cmd, m.tick())
	}

	return m.forwardToActiveModel(msg)
}

// handleKey handles key events.
func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.mode == ModeNormal {
		if handled, cmd := m.handleFocusSwitch(msg); handled {
			return m, cmd
		}
	}
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
	if msg.String() == "q" || msg.String() == "ctrl+c" {
		return m, tea.Quit
	}
	if !m.leftFocused {
		return m.forwardToActiveModel(msg)
	}
	switch {
	case msg.String() == "up" || msg.String() == "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, m.setActiveRepoByCursor()

	case msg.String() == "down" || msg.String() == "j":
		if m.cursor < len(m.repos)-1 {
			m.cursor++
		}
		return m, m.setActiveRepoByCursor()

	case msg.String() == "pgup" || msg.String() == "ctrl+u":
		m.cursor -= 5
		if m.cursor < 0 {
			m.cursor = 0
		}
		return m, m.setActiveRepoByCursor()

	case msg.String() == "pgdown" || msg.String() == "ctrl+f":
		m.cursor += 5
		if m.cursor >= len(m.repos) {
			m.cursor = len(m.repos) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
		return m, m.setActiveRepoByCursor()

	case msg.String() == "enter":
		if len(m.repos) > 0 && m.cursor < len(m.repos) {
			repo := m.repos[m.cursor]
			// Check if repo is in a valid state
			if info, ok := m.repoInfos[repo.Path]; ok && info.State != domain.RepoStateOK {
				m.err = fmt.Errorf("cannot open: %s", info.State.String())
				return m, nil
			}
			// Update last opened and focus right pane
			_ = m.store.UpdateLastOpened(repo.Path)
			m.activeRepo = repo.Path
			m.leftFocused = false
			return m, m.ensureActiveModelAndSize()
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

func (m *Model) handleFocusSwitch(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "ctrl+left":
		m.leftFocused = true
		return true, nil
	case "ctrl+right":
		m.leftFocused = false
		return true, m.ensureActiveModelAndSize()
	case "tab":
		if !m.leftFocused {
			return false, nil
		}
		m.leftFocused = false
		return true, m.ensureActiveModelAndSize()
	case "left":
		if !m.leftFocused {
			if m.activeModelUsesCursorKeys() {
				return false, nil
			}
			m.leftFocused = true
			return true, nil
		}
		return false, nil
	case "right":
		if m.leftFocused {
			m.leftFocused = false
			return true, m.ensureActiveModelAndSize()
		}
		return false, nil
	default:
		return false, nil
	}
}

func (m *Model) ensureActiveModelAndSize() tea.Cmd {
	if m.activeRepo == "" {
		m.syncActiveRepo()
	}
	return m.ensureActiveModel()
}

func (m *Model) setActiveRepoByCursor() tea.Cmd {
	if len(m.repos) == 0 || m.cursor < 0 || m.cursor >= len(m.repos) {
		m.activeRepo = ""
		return nil
	}
	path := m.repos[m.cursor].Path
	if m.activeRepo == path {
		return nil
	}
	m.activeRepo = path
	return m.ensureActiveModel()
}

func (m *Model) clampCursor() {
	if len(m.repos) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.repos) {
		m.cursor = len(m.repos) - 1
	}
}

func (m *Model) syncActiveRepo() {
	if len(m.repos) == 0 {
		m.activeRepo = ""
		return
	}
	if m.activeRepo != "" {
		for i, repo := range m.repos {
			if repo.Path == m.activeRepo {
				m.cursor = i
				return
			}
		}
	}
	m.clampCursor()
	m.activeRepo = m.repos[m.cursor].Path
}

func (m *Model) pruneRepoState() {
	if len(m.repos) == 0 {
		m.repoInfos = make(map[string]domain.WorkspaceRepoInfo)
		m.models = make(map[string]*tui.Model)
		return
	}
	valid := make(map[string]struct{}, len(m.repos))
	for _, repo := range m.repos {
		valid[repo.Path] = struct{}{}
	}
	for path := range m.repoInfos {
		if _, ok := valid[path]; !ok {
			delete(m.repoInfos, path)
		}
	}
	for path := range m.models {
		if _, ok := valid[path]; !ok {
			delete(m.models, path)
		}
	}
}

func (m *Model) ensureActiveModel() tea.Cmd {
	if m.activeRepo == "" {
		return nil
	}
	info, ok := m.repoInfos[m.activeRepo]
	if !ok {
		return nil
	}
	if info.State != domain.RepoStateOK {
		return nil
	}
	if _, ok := m.models[m.activeRepo]; !ok {
		container, err := app.NewWithWarningWriter(m.activeRepo, nil)
		if err != nil {
			m.err = err
			return nil
		}
		model := tui.New(container)
		model.DisableAutoRefresh()
		model.UseHLPagingKeys()
		model.SetHideFooter(true)
		m.models[m.activeRepo] = model
		initCmd := m.wrapRepoCmd(m.activeRepo, model.Init())
		sizeCmd := m.updateModelSize(m.activeRepo)
		return tea.Batch(initCmd, sizeCmd)
	}
	return m.updateModelSize(m.activeRepo)
}

func (m *Model) updateModelSize(path string) tea.Cmd {
	if path == "" {
		return nil
	}
	msg := tea.WindowSizeMsg{Width: m.rightModelWidth(), Height: m.contentHeight()}
	return m.updateModel(path, msg)
}

func (m *Model) propagateWindowSize() tea.Cmd {
	if len(m.models) == 0 {
		return nil
	}
	msg := tea.WindowSizeMsg{Width: m.rightModelWidth(), Height: m.contentHeight()}
	cmds := make([]tea.Cmd, 0, len(m.models))
	for path := range m.models {
		if cmd := m.updateModel(path, msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (m *Model) updateModel(path string, msg tea.Msg) tea.Cmd {
	model, ok := m.models[path]
	if !ok || model == nil {
		return nil
	}
	updated, cmd := model.Update(msg)
	if updatedModel, ok := updated.(*tui.Model); ok {
		m.models[path] = updatedModel
	}
	return m.wrapRepoCmd(path, cmd)
}

func (m *Model) forwardToActiveModel(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.activeRepo == "" {
		return m, nil
	}
	model, ok := m.models[m.activeRepo]
	if !ok || model == nil {
		return m, nil
	}
	updated, cmd := model.Update(msg)
	if updatedModel, ok := updated.(*tui.Model); ok {
		m.models[m.activeRepo] = updatedModel
	}
	return m, m.wrapRepoCmd(m.activeRepo, cmd)
}

func (m *Model) activeModelUsesCursorKeys() bool {
	model, ok := m.models[m.activeRepo]
	if !ok || model == nil {
		return false
	}
	return model.UsesCursorKeys()
}

func (m *Model) routeRepoMsg(msg RepoMsg) (tea.Model, tea.Cmd) {
	if msg.Path == "" {
		return m, nil
	}
	model, ok := m.models[msg.Path]
	if !ok || model == nil {
		return m, nil
	}
	updated, cmd := model.Update(msg.Msg)
	if updatedModel, ok := updated.(*tui.Model); ok {
		m.models[msg.Path] = updatedModel
	}
	return m, m.wrapRepoCmd(msg.Path, cmd)
}

func (m *Model) wrapRepoCmd(path string, cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		msg := cmd()
		if msg == nil {
			return nil
		}
		switch typed := msg.(type) {
		case tea.BatchMsg:
			// BatchMsg is the only composite message we expect from tea.Batch.
			wrapped := make(tea.BatchMsg, 0, len(typed))
			for _, c := range typed {
				if wrappedCmd := m.wrapRepoCmd(path, c); wrappedCmd != nil {
					wrapped = append(wrapped, wrappedCmd)
				}
			}
			if len(wrapped) == 0 {
				return nil
			}
			return wrapped
		case tea.QuitMsg:
			return nil
		case RepoMsg:
			return typed
		default:
			// Route internal TUI messages through RepoMsg; program-level messages
			// (exec/alt-screen/etc.) must pass through unwrapped.
			// MsgFocusWorkspace is handled by workspace model, not wrapped.
			if _, ok := msg.(tui.MsgFocusWorkspace); ok {
				return msg
			}
			if _, ok := msg.(tui.Msg); ok {
				return RepoMsg{Path: path, Msg: msg}
			}
			return msg
		}
	}
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

func (m *Model) contentHeight() int {
	h := m.height - 2
	if h < 0 {
		h = 0
	}
	return h
}

func (m *Model) updateLayout() {
	contentWidth := m.contentWidth()
	if m.width < minSplitWidth {
		m.leftWidth = contentWidth
		m.rightWidth = contentWidth
		return
	}
	leftWidth := int(float64(contentWidth) * leftPaneRatio)
	if leftWidth < minPaneWidth {
		leftWidth = minPaneWidth
	}
	rightWidth := contentWidth - leftWidth
	if rightWidth < minPaneWidth {
		rightWidth = minPaneWidth
		leftWidth = contentWidth - rightWidth
		if leftWidth < minPaneWidth {
			leftWidth = minPaneWidth
		}
	}
	if leftWidth < 0 {
		leftWidth = 0
	}
	if rightWidth < 0 {
		rightWidth = 0
	}
	m.leftWidth = leftWidth
	m.rightWidth = rightWidth
}

func (m *Model) leftContentWidth() int {
	if m.width >= minSplitWidth && m.leftWidth > 0 {
		return m.leftWidth
	}
	return m.contentWidth()
}

func (m *Model) rightContentWidth() int {
	if m.width >= minSplitWidth && m.rightWidth > 0 {
		return m.rightWidth
	}
	return m.contentWidth()
}

func (m *Model) rightModelWidth() int {
	width := m.rightContentWidth()
	if m.isSplitView() && width > 0 {
		width--
	}
	if width < 0 {
		width = 0
	}
	return width
}

func (m *Model) isSplitView() bool {
	return m.width >= minSplitWidth
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

// viewMain renders the main view with header, list, and status line.
func (m *Model) viewMain() string {
	// Show empty state only if no error and no repos
	if len(m.repos) == 0 && !m.loading && m.err == nil {
		return m.viewEmptyState()
	}

	var panes string
	leftPane := m.viewLeftPane()
	if !m.isSplitView() {
		if m.leftFocused {
			panes = leftPane
		} else {
			panes = m.viewRightPane()
		}
	} else {
		rightPane := m.viewRightPane()
		panes = lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	}

	// Add unified status line at the bottom
	statusLine := m.viewStatusLine()
	return lipgloss.JoinVertical(lipgloss.Left, panes, statusLine)
}

func (m *Model) viewLeftPane() string {
	var b strings.Builder

	b.WriteString(m.viewHeader())
	b.WriteString("\n")

	if m.err != nil {
		b.WriteString(m.styles.Error.Render("Error: "+m.err.Error()) + "\n\n")
	}

	b.WriteString(m.viewRepoList())

	height := m.paneContentHeight()
	return lipgloss.NewStyle().Width(m.leftContentWidth()).Height(height).Render(b.String())
}

func (m *Model) viewRightPane() string {
	width := m.rightContentWidth()
	height := m.paneContentHeight()
	content := m.viewRightPaneContent()

	paneStyle := lipgloss.NewStyle().Width(width).Height(height)
	if m.isSplitView() {
		borderColor := tui.Colors.GroupLine
		if !m.leftFocused {
			borderColor = tui.Colors.Primary
		}
		paneStyle = paneStyle.
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(borderColor)
	}

	return paneStyle.Render(content)
}

func (m *Model) viewRightPaneContent() string {
	if m.activeRepo == "" {
		return m.withRightPaneHint(m.styles.Muted.Render("Select a repository to view tasks."))
	}
	info, ok := m.repoInfos[m.activeRepo]
	if !ok {
		return m.withRightPaneHint(m.styles.Loading.Render("Loading repository..."))
	}
	if info.State != domain.RepoStateOK {
		message := fmt.Sprintf("Cannot open: %s", info.State.String())
		if info.ErrorMsg != "" {
			message = message + "\n" + info.ErrorMsg
		}
		return m.withRightPaneHint(m.styles.Error.Render(message))
	}
	model, ok := m.models[m.activeRepo]
	if !ok || model == nil {
		return m.withRightPaneHint(m.styles.Loading.Render("Loading repository tasks..."))
	}
	content := model.View()
	if info.WarningMsg != "" {
		warning := lipgloss.NewStyle().Foreground(tui.Colors.Warning).Render("Warning: " + info.WarningMsg)
		return m.withRightPaneHint(warning + "\n" + content)
	}
	return m.withRightPaneHint(content)
}

func (m *Model) withRightPaneHint(content string) string {
	if !m.isSplitView() && !m.leftFocused {
		backKey := "left"
		if m.activeModelUsesCursorKeys() {
			backKey = "ctrl+left"
		}
		hint := m.styles.Muted.Render(backKey + ": back to list")
		return hint + "\n" + content
	}
	return content
}

// viewHeader renders the header with title left-aligned and count right-aligned.
func (m *Model) viewHeader() string {
	headerStyle := m.styles.Header
	textStyle := m.styles.HeaderText
	mutedStyle := lipgloss.NewStyle().Foreground(tui.Colors.Muted)

	titleText := "Workspace"
	title := textStyle.Render(titleText)

	contentWidth := m.leftContentWidth()
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

	contentWidth := m.leftContentWidth()
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

// viewStatusLine renders the unified status line at the bottom.
func (m *Model) viewStatusLine() string {
	tuiStyles := tui.DefaultStyles()
	statusLine := tui.NewStatusLine(m.contentWidth(), &tuiStyles)
	info := m.getStatusInfo()
	return statusLine.Render(info)
}

// getStatusInfo returns status line info based on current focus.
func (m *Model) getStatusInfo() tui.StatusLineInfo {
	if m.leftFocused {
		// Workspace pane focused
		return tui.StatusLineInfo{
			FocusPane: tui.FocusPaneWorkspace,
			KeyHints: []tui.KeyHint{
				{Key: "j/k", Desc: "nav"},
				{Key: "enter", Desc: "focus"},
				{Key: "tab", Desc: "next"},
				{Key: "a", Desc: "add"},
				{Key: "d", Desc: "remove"},
				{Key: "q", Desc: "quit"},
			},
		}
	}

	// TUI pane focused - get info from active model
	model, ok := m.models[m.activeRepo]
	if !ok || model == nil {
		return tui.StatusLineInfo{
			FocusPane: tui.FocusPaneTaskList,
		}
	}

	return model.GetStatusInfo()
}

// paneContentHeight returns the height available for pane content (excluding status line).
func (m *Model) paneContentHeight() int {
	// height - 1 for status line
	h := m.height - 1
	if h < 0 {
		h = 0
	}
	return h
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
