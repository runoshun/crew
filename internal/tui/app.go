package tui

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase"
)

// AutoRefreshInterval is the default refresh cadence for the TUI.
const AutoRefreshInterval = 5 * time.Second

// Model is the main bubbletea model for the TUI.
//
//nolint:govet // Field alignment not critical for UI menu items.
type actionMenuItem struct {
	ActionID    string
	Label       string
	Desc        string
	Key         string
	Action      func() (tea.Model, tea.Cmd)
	IsAvailable func() bool
	IsDefault   bool
}

func (m *Model) defaultActionID(task *domain.Task) string {
	if task == nil {
		return ""
	}
	switch task.Status {
	case domain.StatusTodo, domain.StatusError:
		return "start"
	case domain.StatusInProgress:
		return "attach"
	case domain.StatusDone:
		return "review_result"
	case domain.StatusMerged, domain.StatusClosed:
		return "detail"
	}
	return ""
}

//nolint:govet // Field alignment optimized for readability over memory layout.
type Model struct {
	// Dependencies (pointers first for alignment)
	container *app.Container
	config    *domain.Config
	warnings  []string
	err       error

	// State (slices - contain pointers)
	tasks            []*domain.Task
	comments         []domain.Comment
	commentCounts    map[int]int // taskID -> comment count
	builtinAgents    []string
	customAgents     []string
	managerAgents    []string
	agentCommands    map[string]string
	customKeybinds   map[string]domain.TUIKeybinding
	keybindWarnings  []string
	reviewResult     string // Review result text
	actionMenuLastID string
	actionMenuItems  []actionMenuItem

	// Components (structs with pointers)
	keys                KeyMap
	styles              Styles
	help                help.Model
	taskList            list.Model
	detailPanelViewport viewport.Model // For right pane detail panel

	// Input state (large structs)
	titleInput         textinput.Model
	descInput          textinput.Model
	parentInput        textinput.Model
	filterInput        textinput.Model
	customInput        textinput.Model
	execInput          textinput.Model
	reviewMessageInput textinput.Model

	// Review components
	reviewViewport   viewport.Model // For scrollable review result
	editCommentInput textinput.Model

	// Block dialog components
	blockInput textinput.Model

	// Panel content state (strings before smaller types)
	diffContent string // Cached diff content
	peekContent string // Cached peek content

	// Numeric state (smaller types last)
	mode                    Mode
	confirmAction           ConfirmAction
	sortMode                SortMode
	newTaskField            NewTaskField
	panelContent            PanelContent // Current content type displayed in right panel
	width                   int
	height                  int
	confirmTaskID           int
	agentCursor             int
	managerAgentCursor      int
	statusCursor            int
	actionMenuCursor        int
	actionMenuLastTaskID    int
	reviewTaskID            int // Task being reviewed
	reviewActionCursor      int // Cursor for action selection
	reviewMessageReturnMode Mode
	editCommentIndex        int // Index of comment being edited
	startFocusCustom        bool
	showAll                 bool
	detailFocused           bool // Right pane is focused for scrolling
	blockFocusUnblock       bool // True when Unblock button is focused in Block dialog
	autoRefresh             bool
	selectedTaskHasWorktree bool
	hideFooter              bool // Hide footer (used when embedded in workspace)
	hideDetailPanel         bool // Hide detail panel (used when embedded in workspace 1-pane mode)
	embedded                bool // Embedded mode (skip App padding, used in workspace)
	focused                 bool // Whether this TUI has focus (used when embedded in workspace)
	panelContentLoading     bool // Whether panel content is being loaded
}

// New creates a new TUI Model with the given container.
func New(c *app.Container) *Model {
	ti := textinput.New()
	ti.Placeholder = "Task title"
	ti.CharLimit = 200

	di := textinput.New()
	di.Placeholder = "Task description (optional)"
	di.CharLimit = 1000

	pi := textinput.New()
	pi.Placeholder = "Parent task ID (optional)"
	pi.CharLimit = 10

	fi := textinput.New()
	fi.Placeholder = "Filter tasks..."
	fi.CharLimit = 100

	ci := textinput.New()
	ci.Placeholder = "Enter custom command..."
	ci.CharLimit = 500

	ei := textinput.New()
	ei.Placeholder = "Enter command to execute..."
	ei.CharLimit = 500

	ri := textinput.New()
	ri.Placeholder = "Message to worker (optional, empty uses default)"
	ri.CharLimit = 500

	eci := textinput.New()
	eci.Placeholder = "Edit review comment..."
	eci.CharLimit = 2000

	bi := textinput.New()
	bi.Placeholder = "Enter block reason..."
	bi.CharLimit = 200

	styles := DefaultStyles()
	delegate := newTaskDelegate(styles)
	taskList := list.New([]list.Item{}, delegate, 0, 0)
	taskList.SetShowTitle(false)
	taskList.SetShowStatusBar(false)
	taskList.SetShowHelp(false)
	taskList.SetShowPagination(false)
	taskList.SetFilteringEnabled(true)
	taskList.DisableQuitKeybindings()

	// Initialize review viewport (size will be updated in WindowSizeMsg)
	reviewVp := viewport.New(0, 0)

	return &Model{
		container:          c,
		mode:               ModeNormal,
		tasks:              nil,
		keys:               DefaultKeyMap(),
		styles:             styles,
		help:               help.New(),
		taskList:           taskList,
		titleInput:         ti,
		descInput:          di,
		parentInput:        pi,
		filterInput:        fi,
		customInput:        ci,
		execInput:          ei,
		reviewMessageInput: ri,
		editCommentInput:   eci,
		blockInput:         bi,
		reviewViewport:     reviewVp,
		builtinAgents:      []string{"claude", "opencode", "codex"},
		customAgents:       nil,
		managerAgents:      nil,
		agentCommands:      make(map[string]string),
		customKeybinds:     make(map[string]domain.TUIKeybinding),
		keybindWarnings:    nil,
		commentCounts:      make(map[int]int),
		agentCursor:        0,
		startFocusCustom:   false,
		autoRefresh:        true,
	}
}

// Init initializes the model and returns the initial command.
func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.loadTasks(), m.loadConfig()}
	if m.autoRefresh {
		cmds = append(cmds, m.tick())
	}
	return tea.Batch(cmds...)
}

// DisableAutoRefresh disables periodic task refresh ticks.
func (m *Model) DisableAutoRefresh() {
	m.autoRefresh = false
}

// SetHideFooter sets whether to hide the footer (used when embedded in workspace).
func (m *Model) SetHideFooter(hide bool) {
	m.hideFooter = hide
}

// SetHideDetailPanel sets whether to hide the detail panel (used when embedded in workspace 1-pane mode).
func (m *Model) SetHideDetailPanel(hide bool) {
	m.hideDetailPanel = hide
}

// SetEmbedded sets whether the TUI is embedded in another view (skips App padding).
func (m *Model) SetEmbedded(embedded bool) {
	m.embedded = embedded
}

// SetFocused sets whether this TUI has focus (used when embedded in workspace).
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// UsesCursorKeys reports whether left/right should be forwarded to inputs.
func (m *Model) UsesCursorKeys() bool {
	if m.mode.IsInputMode() {
		return true
	}
	return m.mode == ModeStart && m.startFocusCustom
}

// UseHLPagingKeys limits page navigation to h/l.
func (m *Model) UseHLPagingKeys() {
	m.keys.PrevPage = key.NewBinding(
		key.WithKeys("h"),
		key.WithHelp("h", "prev page"),
	)
	m.keys.NextPage = key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "next page"),
	)
}

// tick returns a command that sends a tick message after the refresh interval.
func (m *Model) tick() tea.Cmd {
	return tea.Tick(AutoRefreshInterval, func(t time.Time) tea.Msg {
		return MsgTick{}
	})
}

// loadTasks returns a command that loads tasks from the repository.
func (m *Model) loadTasks() tea.Cmd {
	return func() tea.Msg {
		out, err := m.container.ListTasksUseCase().Execute(context.Background(), usecase.ListTasksInput{
			IncludeTerminal: m.showAll,
		})
		if err != nil {
			return MsgError{Err: err}
		}
		return MsgTasksLoaded{Tasks: out.Tasks}
	}
}

// loadConfig returns a command that loads the configuration.
func (m *Model) loadConfig() tea.Cmd {
	return func() tea.Msg {
		cfg, err := m.container.ConfigLoader.Load()
		if err != nil {
			// Config loading failure is not fatal; use defaults
			cfg = domain.NewDefaultConfig()
		}
		return MsgConfigLoaded{Config: cfg}
	}
}

// loadComments returns a command that loads comments for a task.
func (m *Model) loadComments(taskID int) tea.Cmd {
	return func() tea.Msg {
		out, err := m.container.ShowTaskUseCase().Execute(context.Background(), usecase.ShowTaskInput{TaskID: taskID})
		if err != nil {
			return MsgError{Err: err}
		}
		return MsgCommentsLoaded{TaskID: taskID, Comments: out.Comments}
	}
}

// loadCommentCounts returns a command that loads comment counts for all tasks.
func (m *Model) loadCommentCounts() tea.Cmd {
	return func() tea.Msg {
		counts := make(map[int]int)
		for _, task := range m.tasks {
			comments, err := m.container.Tasks.GetComments(task.ID)
			if err != nil {
				// Skip on error, just leave count as 0
				continue
			}
			counts[task.ID] = len(comments)
		}
		return MsgCommentCountsLoaded{CommentCounts: counts}
	}
}

// SelectedTask returns the currently selected task, or nil if none.
func (m *Model) SelectedTask() *domain.Task {
	if m.taskList.SelectedItem() == nil {
		return nil
	}
	if ti, ok := m.taskList.SelectedItem().(taskItem); ok {
		return ti.task
	}
	return nil
}

func (m *Model) canStopSelectedTask(task *domain.Task) bool {
	if task == nil {
		return false
	}
	if task.Status == domain.StatusInProgress {
		return true
	}
	return task.Session != ""
}

func (m *Model) hasWorktree(task *domain.Task) bool {
	if task == nil {
		return false
	}
	if m.container == nil || m.container.Worktrees == nil {
		m.err = fmt.Errorf("worktree manager not initialized")
		return false
	}
	branch := domain.BranchName(task.ID, task.Issue)
	exists, err := m.container.Worktrees.Exists(branch)
	if err != nil {
		m.err = fmt.Errorf("check worktree: %w", err)
		return false
	}
	if !exists {
		m.err = domain.ErrWorktreeNotFound
		return false
	}
	return true
}

func (m *Model) hasWorktreeQuiet(task *domain.Task) bool {
	if task == nil {
		return false
	}
	if m.container == nil || m.container.Worktrees == nil {
		return false
	}
	branch := domain.BranchName(task.ID, task.Issue)
	exists, err := m.container.Worktrees.Exists(branch)
	if err != nil {
		return false
	}
	return exists
}

func (m *Model) updateSelectedTaskWorktree() {
	task := m.SelectedTask()
	if task == nil {
		m.selectedTaskHasWorktree = false
		return
	}
	m.selectedTaskHasWorktree = m.hasWorktreeQuiet(task)
}

func sessionNameForLog(taskID int, isReview bool) string {
	if isReview {
		return domain.ReviewSessionName(taskID)
	}
	return domain.SessionName(taskID)
}

func (m *Model) sessionLogPath(taskID int, isReview bool) (string, error) {
	if m.container == nil {
		return "", fmt.Errorf("container not initialized")
	}
	crewDir := m.container.Config.CrewDir
	if crewDir == "" {
		return "", fmt.Errorf("crew dir not set")
	}
	sessionName := sessionNameForLog(taskID, isReview)
	return domain.SessionLogPath(crewDir, sessionName), nil
}

func (m *Model) hasSessionLog(taskID int, isReview bool) bool {
	logPath, err := m.sessionLogPath(taskID, isReview)
	if err != nil {
		return false
	}
	if _, err := os.Stat(logPath); err != nil {
		return false
	}
	return true
}

// updateTaskList updates the task list items from tasks.
func (m *Model) updateTaskList() {
	if m.filterInput.Value() != "" {
		m.applyFilter()
		return
	}
	m.setTaskItems(m.sortedTasks())
}

func (m *Model) setTaskItems(tasks []*domain.Task) {
	items := make([]list.Item, 0, len(tasks))
	for _, task := range tasks {
		count := m.commentCounts[task.ID]
		items = append(items, taskItem{task: task, commentCount: count})
	}
	m.taskList.SetItems(items)
	m.updateSelectedTaskWorktree()
}

// updateDetailPanelViewport updates the viewport size and content for the right pane.
func (m *Model) updateDetailPanelViewport() {
	if !m.showDetailPanel() {
		return
	}
	panelWidth := m.detailPanelWidth() - 4 // account for border and padding
	// Height calculation:
	// - panel height = m.height - 3 (see viewDetailPanel)
	// - minus 2 for panel header (Task #N + border line)
	panelHeight := m.height - 5
	if panelHeight < 5 {
		panelHeight = 5
	}
	m.detailPanelViewport.Width = panelWidth
	m.detailPanelViewport.Height = panelHeight
	m.detailPanelViewport.SetContent(m.panelContentString(panelWidth))
}

// updateLayoutSizes updates all layout-dependent sizes (taskList, viewport).
// Call this when detailFocused changes or window size changes.
func (m *Model) updateLayoutSizes() {
	// Set taskList width to headerFooterContentWidth for consistent alignment
	// Header/Footer use this same width with Padding(0, 1) which adds 2 to total width
	// but lipgloss Width() includes padding, so the content area matches
	listW := m.headerFooterContentWidth()
	// Height calculation: subtract header (2) and footer (3 if visible)
	// In embedded mode, footer is hidden so only subtract header
	listH := m.height - 2 // header
	if !m.hideFooter {
		listH -= 3 // footer (border + content + margin)
	}
	if listH < 5 {
		listH = 5
	}
	m.taskList.SetSize(listW, listH)
	m.updateDetailPanelViewport()
	m.updateReviewViewport()
}

// updateReviewViewport updates the review viewport size and content based on dialog dimensions.
func (m *Model) updateReviewViewport() {
	dialogW := m.dialogWidth() - 8 // Account for padding
	// Leave space for title, task line, hint, and padding (approximately 8 lines)
	dialogH := m.height - 16
	if dialogH < 5 {
		dialogH = 5
	}
	if dialogH > 20 {
		dialogH = 20
	}
	m.reviewViewport.Width = dialogW
	m.reviewViewport.Height = dialogH

	// Re-render content with word wrap and dialog background color when size changes.
	// Clear content when reviewResult is empty to avoid showing stale data.
	if m.reviewResult == "" {
		m.reviewViewport.SetContent("")
	} else {
		renderedContent := m.styles.RenderMarkdownWithBg(m.reviewResult, dialogW, Colors.Background)
		m.reviewViewport.SetContent(renderedContent)
	}
}

func (m *Model) dialogWidth() int {
	width := m.width - 16
	if width < 50 {
		width = 50
	}
	if width > 80 {
		width = 80
	}
	return width
}

var statusPriority = map[domain.Status]int{
	domain.StatusTodo:       0,
	domain.StatusInProgress: 1, // Work in progress
	domain.StatusDone:       2, // Complete, awaiting merge
	domain.StatusError:      3,
	domain.StatusMerged:     4,
	domain.StatusClosed:     5,
}

// getStatusPriority returns the sort priority for a status.
func getStatusPriority(status domain.Status) int {
	if p, ok := statusPriority[status]; ok {
		return p
	}
	// Unknown status goes to the end
	return 99
}

func (m *Model) sortedTasks() []*domain.Task {
	tasks := make([]*domain.Task, len(m.tasks))
	copy(tasks, m.tasks)

	switch m.sortMode {
	case SortByStatusAsc:
		sort.Slice(tasks, func(i, j int) bool {
			pi := getStatusPriority(tasks[i].Status)
			pj := getStatusPriority(tasks[j].Status)
			if pi != pj {
				return pi < pj
			}
			return tasks[i].ID < tasks[j].ID
		})
	case SortByStatusDesc:
		sort.Slice(tasks, func(i, j int) bool {
			pi := getStatusPriority(tasks[i].Status)
			pj := getStatusPriority(tasks[j].Status)
			if pi != pj {
				return pi > pj
			}
			return tasks[i].ID > tasks[j].ID
		})
	case SortByIDAsc:
		sort.Slice(tasks, func(i, j int) bool {
			return tasks[i].ID < tasks[j].ID
		})
	case SortByIDDesc:
		sort.Slice(tasks, func(i, j int) bool {
			return tasks[i].ID > tasks[j].ID
		})
	}

	return tasks
}

// startTask returns a command that starts a task with the given agent.
func (m *Model) startTask(taskID int, agent string) tea.Cmd {
	return func() tea.Msg {
		out, err := m.container.StartTaskUseCase().Execute(
			context.Background(),
			usecase.StartTaskInput{TaskID: taskID, Agent: agent},
		)
		if err != nil {
			return MsgError{Err: err}
		}
		return MsgTaskStarted{TaskID: taskID, SessionName: out.SessionName}
	}
}

// stopTask returns a command that stops a task session.
func (m *Model) stopTask(taskID int) tea.Cmd {
	return func() tea.Msg {
		out, err := m.container.StopTaskUseCase().Execute(
			context.Background(),
			usecase.StopTaskInput{TaskID: taskID},
		)
		if err != nil {
			return MsgError{Err: err}
		}
		return MsgTaskStopped{TaskID: taskID, SessionName: out.SessionName}
	}
}

// createTask returns a command that creates a new task.
func (m *Model) createTask(title, desc string) tea.Cmd {
	return func() tea.Msg {
		out, err := m.container.NewTaskUseCase().Execute(
			context.Background(),
			usecase.NewTaskInput{Title: title, Description: desc},
		)
		if err != nil {
			return MsgError{Err: err}
		}
		return MsgTaskCreated{TaskID: out.TaskID}
	}
}

// createTaskWithParent returns a command that creates a new task with optional parent.
func (m *Model) createTaskWithParent(title, desc, parentStr string) tea.Cmd {
	return func() tea.Msg {
		input := usecase.NewTaskInput{Title: title, Description: desc}

		// Parse parent ID if provided
		if parentStr != "" {
			var parentID int
			if _, err := fmt.Sscanf(parentStr, "%d", &parentID); err == nil {
				input.ParentID = &parentID
			}
		}

		out, err := m.container.NewTaskUseCase().Execute(
			context.Background(),
			input,
		)
		if err != nil {
			return MsgError{Err: err}
		}
		return MsgTaskCreated{TaskID: out.TaskID}
	}
}

// deleteTask returns a command that deletes a task.
func (m *Model) deleteTask(taskID int) tea.Cmd {
	return func() tea.Msg {
		_, err := m.container.DeleteTaskUseCase().Execute(
			context.Background(),
			usecase.DeleteTaskInput{TaskID: taskID},
		)
		if err != nil {
			return MsgError{Err: err}
		}
		return MsgTaskDeleted{TaskID: taskID}
	}
}

// closeTask returns a command that closes a task.
func (m *Model) closeTask(taskID int) tea.Cmd {
	return func() tea.Msg {
		_, err := m.container.CloseTaskUseCase().Execute(
			context.Background(),
			usecase.CloseTaskInput{TaskID: taskID},
		)
		if err != nil {
			return MsgError{Err: err}
		}
		return MsgTaskClosed{TaskID: taskID}
	}
}

// mergeTask returns a command that merges a task.
func (m *Model) mergeTask(taskID int) tea.Cmd {
	return func() tea.Msg {
		_, err := m.container.MergeTaskUseCase().Execute(
			context.Background(),
			usecase.MergeTaskInput{TaskID: taskID},
		)
		if err != nil {
			return MsgError{Err: err}
		}
		return MsgTaskMerged{TaskID: taskID}
	}
}

// copyTask returns a command that copies a task.
func (m *Model) copyTask(taskID int, copyAll bool) tea.Cmd {
	return func() tea.Msg {
		input := usecase.CopyTaskInput{SourceID: taskID}
		if copyAll {
			input.CopyAll = true
		}
		out, err := m.container.CopyTaskUseCase().Execute(
			context.Background(),
			input,
		)
		if err != nil {
			return MsgError{Err: err}
		}
		return MsgTaskCopied{OriginalID: taskID, NewID: out.TaskID}
	}
}

// updateStatus returns a command that updates the status of a task.
func (m *Model) updateStatus(taskID int, status domain.Status) tea.Cmd {
	return func() tea.Msg {
		_, err := m.container.EditTaskUseCase().Execute(
			context.Background(),
			usecase.EditTaskInput{
				TaskID: taskID,
				Status: &status,
			},
		)
		if err != nil {
			return MsgError{Err: err}
		}
		return MsgTaskStatusUpdated{TaskID: taskID, Status: status}
	}
}

// blockTask returns a command that blocks a task with a reason.
func (m *Model) blockTask(taskID int, reason string) tea.Cmd {
	return func() tea.Msg {
		_, err := m.container.EditTaskUseCase().Execute(
			context.Background(),
			usecase.EditTaskInput{
				TaskID:      taskID,
				BlockReason: &reason,
			},
		)
		if err != nil {
			return MsgError{Err: err}
		}
		return MsgReloadTasks{}
	}
}

// unblockTask returns a command that unblocks a task.
func (m *Model) unblockTask(taskID int) tea.Cmd {
	return func() tea.Msg {
		emptyReason := ""
		_, err := m.container.EditTaskUseCase().Execute(
			context.Background(),
			usecase.EditTaskInput{
				TaskID:      taskID,
				BlockReason: &emptyReason,
			},
		)
		if err != nil {
			return MsgError{Err: err}
		}
		return MsgReloadTasks{}
	}
}

func (m *Model) actionMenuItemsForTask(task *domain.Task) []actionMenuItem {
	if task == nil {
		return nil
	}

	hasWorktree := m.hasWorktreeQuiet(task)

	actions := []actionMenuItem{
		{
			ActionID: "start",
			Label:    "Start",
			Desc:     "Start with agent",
			Key:      "s",
			Action: func() (tea.Model, tea.Cmd) {
				m.mode = ModeStart
				return m, nil
			},
			IsAvailable: func() bool {
				return task.Status.CanStart() && !task.IsBlocked()
			},
		},
		{
			ActionID: "stop",
			Label:    "Stop",
			Desc:     "Stop running session",
			Key:      "S",
			Action: func() (tea.Model, tea.Cmd) {
				m.mode = ModeConfirm
				m.confirmAction = ConfirmStop
				m.confirmTaskID = task.ID
				return m, nil
			},
			IsAvailable: func() bool {
				return task.IsRunning() && task.Status == domain.StatusInProgress
			},
		},
		{
			ActionID: "attach",
			Label:    "Attach",
			Desc:     "Attach to session",
			Key:      "a",
			Action: func() (tea.Model, tea.Cmd) {
				return m, func() tea.Msg {
					return MsgAttachSession{TaskID: task.ID}
				}
			},
			IsAvailable: func() bool {
				return task.IsRunning()
			},
		},
		{
			ActionID: "show_diff",
			Label:    "Show Diff",
			Desc:     "Open diff for review",
			Key:      "D",
			Action: func() (tea.Model, tea.Cmd) {
				return m, func() tea.Msg {
					return MsgShowDiff{TaskID: task.ID}
				}
			},
			IsAvailable: func() bool {
				return task.Status == domain.StatusDone
			},
		},
		{
			ActionID: "show_worker_log",
			Label:    "Show Worker Log",
			Desc:     "Open worker session log",
			Action: func() (tea.Model, tea.Cmd) {
				return m, m.showLogInPager(task.ID, false)
			},
			IsAvailable: func() bool {
				return m.hasSessionLog(task.ID, false)
			},
		},
		{
			ActionID: "show_review_log",
			Label:    "Show Review Log",
			Desc:     "Open review session log",
			Action: func() (tea.Model, tea.Cmd) {
				return m, m.showLogInPager(task.ID, true)
			},
			IsAvailable: func() bool {
				return m.hasSessionLog(task.ID, true)
			},
		},
		{
			ActionID: "review_result",
			Label:    "Review Result",
			Desc:     "Open review summary",
			Key:      "r",
			Action: func() (tea.Model, tea.Cmd) {
				return m, m.loadReviewResult(task.ID)
			},
			IsAvailable: func() bool {
				return task.Status == domain.StatusDone
			},
		},
		{
			ActionID: "change_status",
			Label:    "Change Status",
			Desc:     "Pick new status",
			Key:      "e",
			Action: func() (tea.Model, tea.Cmd) {
				m.mode = ModeChangeStatus
				m.statusCursor = 0
				return m, nil
			},
			IsAvailable: func() bool {
				return !task.Status.IsTerminal()
			},
		},
		{
			ActionID: "exec",
			Label:    "Execute",
			Desc:     "Run command in worktree",
			Key:      "x",
			Action: func() (tea.Model, tea.Cmd) {
				m.mode = ModeExec
				m.execInput.Focus()
				return m, nil
			},
			IsAvailable: func() bool {
				return hasWorktree
			},
		},
		{
			ActionID: "request_changes",
			Label:    "Request Changes",
			Desc:     "Send request changes (back to in_progress, notify if available)",
			Key:      "R",
			Action: func() (tea.Model, tea.Cmd) {
				m.enterRequestChanges(task.ID, ModeActionMenu, true)
				return m, nil
			},
			IsAvailable: func() bool {
				return task.Status == domain.StatusDone && hasWorktree
			},
		},
		{
			ActionID: "copy",
			Label:    "Copy",
			Desc:     "Duplicate task",
			Key:      "y",
			Action: func() (tea.Model, tea.Cmd) {
				return m, m.copyTask(task.ID, false)
			},
			IsAvailable: func() bool {
				return true
			},
		},
		{
			ActionID: "copy_all",
			Label:    "Copy All",
			Desc:     "Copy task, comments, and code state",
			Key:      "Y",
			Action: func() (tea.Model, tea.Cmd) {
				return m, m.copyTask(task.ID, true)
			},
			IsAvailable: func() bool {
				return true
			},
		},
		{
			ActionID: "edit",
			Label:    "Edit",
			Desc:     "Edit task in editor",
			Key:      "E",
			Action: func() (tea.Model, tea.Cmd) {
				return m, m.editTaskInEditor(task.ID)
			},
			IsAvailable: func() bool {
				return true
			},
		},
		{
			ActionID: "delete",
			Label:    "Delete",
			Desc:     "Delete task",
			Key:      "d",
			Action: func() (tea.Model, tea.Cmd) {
				m.mode = ModeConfirm
				m.confirmAction = ConfirmDelete
				m.confirmTaskID = task.ID
				return m, nil
			},
			IsAvailable: func() bool {
				return true
			},
		},
		{
			ActionID: "close",
			Label:    "Close",
			Desc:     "Close task",
			Key:      "c",
			Action: func() (tea.Model, tea.Cmd) {
				m.mode = ModeConfirm
				m.confirmAction = ConfirmClose
				m.confirmTaskID = task.ID
				return m, nil
			},
			IsAvailable: func() bool {
				return !task.Status.IsTerminal()
			},
		},
		{
			ActionID: "merge",
			Label:    "Merge",
			Desc:     "Merge task",
			Key:      "m",
			Action: func() (tea.Model, tea.Cmd) {
				m.mode = ModeConfirm
				m.confirmAction = ConfirmMerge
				m.confirmTaskID = task.ID
				return m, nil
			},
			IsAvailable: func() bool {
				return task.Status == domain.StatusDone
			},
		},
		{
			ActionID: "detail",
			Label:    "Details",
			Desc:     "Open details panel",
			Key:      "v",
			Action: func() (tea.Model, tea.Cmd) {
				m.detailFocused = true
				m.updateLayoutSizes()
				return m, m.loadComments(task.ID)
			},
			IsAvailable: func() bool {
				return true
			},
		},
		{
			ActionID: "block",
			Label:    "Block",
			Desc:     "Block or unblock task",
			Key:      "b",
			Action: func() (tea.Model, tea.Cmd) {
				m.mode = ModeBlock
				m.blockFocusUnblock = false
				// Pre-fill with existing block reason if any
				if task.IsBlocked() {
					m.blockInput.SetValue(task.BlockReason)
				} else {
					m.blockInput.Reset()
				}
				m.blockInput.Focus()
				return m, nil
			},
			IsAvailable: func() bool {
				// Block is available for non-terminal tasks
				return !task.Status.IsTerminal()
			},
		},
	}

	var filtered []actionMenuItem
	for _, action := range actions {
		if action.IsAvailable == nil || action.IsAvailable() {
			filtered = append(filtered, action)
		}
	}

	defaultID := m.defaultActionID(task)

	if defaultID == "" {
		defaultID = "detail"
	}
	for i := range filtered {
		if filtered[i].ActionID == defaultID {
			filtered[i].IsDefault = true
			break
		}
	}

	return filtered
}

func (m *Model) updateAgents() {
	if m.config == nil {
		return
	}

	// Build agent command previews
	m.agentCommands = make(map[string]string)

	// Filter agents: only show worker-role agents that are not hidden
	// EnabledAgents() already filters out disabled agents
	m.builtinAgents = []string{}
	m.customAgents = []string{}
	m.managerAgents = []string{}
	for name, agentDef := range m.config.EnabledAgents() {
		// Skip hidden agents
		if agentDef.Hidden {
			continue
		}

		// Add to appropriate list based on role
		switch agentDef.Role {
		case domain.RoleWorker, "":
			m.builtinAgents = append(m.builtinAgents, name)
		case domain.RoleManager:
			m.managerAgents = append(m.managerAgents, name)
		case domain.RoleReviewer:
			continue
		default:
			continue
		}

		// Extract command from command template (simplified - first word)
		cmdTemplate := agentDef.CommandTemplate
		if cmdTemplate != "" {
			parts := strings.Fields(cmdTemplate)
			if len(parts) > 0 {
				m.agentCommands[name] = parts[0]
			}
		}
	}

	// Sort agent lists for stable alphabetical order
	sort.Strings(m.builtinAgents)
	sort.Strings(m.customAgents)
	sort.Strings(m.managerAgents)

	// Set cursor to default agent
	allAgents := m.allAgents()
	for i, a := range allAgents {
		if a == m.config.AgentsConfig.DefaultWorker {
			m.agentCursor = i
			break
		}
	}

	// Set manager cursor to default manager
	for i, a := range m.managerAgents {
		if a == m.config.AgentsConfig.DefaultManager {
			m.managerAgentCursor = i
			break
		}
	}
}

// allAgents returns all agents (built-in + custom).
func (m *Model) allAgents() []string {
	result := make([]string, 0, len(m.builtinAgents)+len(m.customAgents))
	result = append(result, m.builtinAgents...)
	result = append(result, m.customAgents...)
	return result
}

// loadCustomKeybindings loads custom keybindings from config and checks for conflicts.
func (m *Model) loadCustomKeybindings() {
	if m.config == nil {
		return
	}

	m.customKeybinds = make(map[string]domain.TUIKeybinding)
	m.keybindWarnings = nil

	// Get builtin keys
	builtinKeys := m.keys.GetBuiltinKeys()

	// Load custom keybindings
	for key, binding := range m.config.TUI.Keybindings {
		// Check for conflict with builtin keys
		if builtinKeys[key] && !binding.Override {
			warning := fmt.Sprintf("keybinding conflict: '%s' already exists (set override=true to override)", key)
			m.keybindWarnings = append(m.keybindWarnings, warning)
			continue
		}
		m.customKeybinds[key] = binding
	}
}

// domainExecCmd wraps domain.ExecCommand to implement tea.ExecCommand.
// This allows using the domain type for command construction while
// satisfying the tea.ExecCommand interface for TUI execution.
type domainExecCmd struct {
	cmd          *domain.ExecCommand
	stdin        io.Reader
	stdout       io.Writer
	stderr       io.Writer
	ignoreErrors bool // If true, Run() always returns nil
}

func (c *domainExecCmd) Run() error {
	// #nosec G204 - cmd.Program and cmd.Args come from trusted internal values
	execCmd := exec.Command(c.cmd.Program, c.cmd.Args...)
	if c.cmd.Dir != "" {
		execCmd.Dir = c.cmd.Dir
	}
	execCmd.Stdin = c.stdin
	execCmd.Stdout = c.stdout
	execCmd.Stderr = c.stderr
	if c.ignoreErrors {
		_ = execCmd.Run()
		return nil
	}
	return execCmd.Run()
}

func (c *domainExecCmd) SetStdin(r io.Reader)  { c.stdin = r }
func (c *domainExecCmd) SetStdout(w io.Writer) { c.stdout = w }
func (c *domainExecCmd) SetStderr(w io.Writer) { c.stderr = w }

// attachToSession returns a tea.Cmd that attaches to a tmux session.
// After the attach completes (user detaches), it triggers a task reload.
func (m *Model) attachToSession(taskID int) tea.Cmd {
	socketPath := m.container.Config.SocketPath
	sessionName := domain.SessionName(taskID)

	cmd := domain.NewCommand("tmux", []string{"-S", socketPath, "attach", "-t", sessionName}, "")
	return tea.Exec(&domainExecCmd{cmd: cmd}, func(err error) tea.Msg {
		// Reload tasks after detaching from the session
		return MsgReloadTasks{}
	})
}

// showDiff returns a tea.Cmd that shows the diff for a task.
// After the diff viewer closes, it triggers a task reload.
func (m *Model) showDiff(taskID int) tea.Cmd {
	return func() tea.Msg {
		uc := m.container.ShowDiffUseCaseForCommand()
		execCmd, err := uc.GetCommand(context.Background(), usecase.ShowDiffInput{
			TaskID: taskID,
		})
		if err != nil {
			return MsgError{Err: err}
		}

		return execDiffMsg{cmd: execCmd}
	}
}

func resolvePagerCommand() (string, []string, error) {
	pager := strings.TrimSpace(os.Getenv("PAGER"))
	if pager != "" {
		fields := strings.Fields(pager)
		if len(fields) == 0 {
			return "", nil, fmt.Errorf("PAGER is set but empty")
		}
		return fields[0], fields[1:], nil
	}

	candidates := []struct {
		program string
		args    []string
	}{
		{program: "less", args: []string{"-R"}},
		{program: "more"},
	}
	for _, candidate := range candidates {
		if _, err := exec.LookPath(candidate.program); err == nil {
			return candidate.program, candidate.args, nil
		}
	}

	return "", nil, fmt.Errorf("no pager found (set PAGER)")
}

// showLogInPager returns a tea.Cmd that opens a session log in a pager.
func (m *Model) showLogInPager(taskID int, isReview bool) tea.Cmd {
	return func() tea.Msg {
		logPath, err := m.sessionLogPath(taskID, isReview)
		if err != nil {
			return MsgError{Err: err}
		}
		if _, statErr := os.Stat(logPath); statErr != nil {
			if os.IsNotExist(statErr) {
				sessionName := sessionNameForLog(taskID, isReview)
				return MsgError{Err: fmt.Errorf("no log file found for session %s: %w", sessionName, domain.ErrNoSession)}
			}
			return MsgError{Err: fmt.Errorf("check log file: %w", statErr)}
		}
		program, args, err := resolvePagerCommand()
		if err != nil {
			return MsgError{Err: err}
		}
		cmd := domain.NewCommand(program, append(args, logPath), "")
		return execLogMsg{cmd: cmd}
	}
}

// execLogMsg is an internal message to trigger log pager execution.
type execLogMsg struct {
	cmd *domain.ExecCommand
}

// execDiffMsg is an internal message to trigger diff execution.
type execDiffMsg struct {
	cmd *domain.ExecCommand
}

// executeCommandInWorktree returns a tea.Cmd that executes a command in a task's worktree.
func (m *Model) executeCommandInWorktree(command string) tea.Cmd {
	task := m.SelectedTask()
	if task == nil {
		return func() tea.Msg {
			return MsgError{Err: domain.ErrTaskNotFound}
		}
	}

	branch := domain.BranchName(task.ID, task.Issue)
	wtPath, err := m.container.Worktrees.Resolve(branch)
	if err != nil {
		return func() tea.Msg {
			return MsgError{Err: fmt.Errorf("resolve worktree: %w", err)}
		}
	}

	cmd := domain.NewShellCommand(command, wtPath)
	return tea.Exec(&domainExecCmd{cmd: cmd}, func(err error) tea.Msg {
		return MsgReloadTasks{}
	})
}

// executeCommandInRepoRoot returns a tea.Cmd that executes a command in the repository root.
func (m *Model) executeCommandInRepoRoot(command string) tea.Cmd {
	cmd := domain.NewShellCommand(command, m.container.Config.RepoRoot)
	return tea.Exec(&domainExecCmd{cmd: cmd}, func(err error) tea.Msg {
		return MsgReloadTasks{}
	})
}

// handleCustomKeybinding handles a custom keybinding action.
func (m *Model) handleCustomKeybinding(binding domain.TUIKeybinding) (tea.Model, tea.Cmd) {
	task := m.SelectedTask()
	if task == nil {
		return m, nil
	}

	// Render template
	command, err := m.renderKeybindingTemplate(binding.Command, task)
	if err != nil {
		m.err = fmt.Errorf("render keybinding template: %w", err)
		return m, nil
	}

	// Execute the command in worktree or repository root
	if binding.Worktree {
		return m, m.executeCommandInWorktree(command)
	}
	return m, m.executeCommandInRepoRoot(command)
}

// KeybindingTemplateData contains data available for keybinding command templates.
type KeybindingTemplateData struct {
	// Task provides access to all task fields (e.g., {{.Task.BaseBranch}})
	Task *domain.Task

	// Convenience fields for backward compatibility
	TaskTitle    string // Same as .Task.Title
	TaskStatus   string // Same as string(.Task.Status)
	Branch       string // Branch name derived from task ID and issue
	WorktreePath string // Worktree path (empty if not created)
	TaskID       int    // Same as .Task.ID
}

// renderKeybindingTemplate renders a keybinding command template with task data.
func (m *Model) renderKeybindingTemplate(cmdTemplate string, task *domain.Task) (string, error) {
	branch := domain.BranchName(task.ID, task.Issue)
	wtPath, err := m.container.Worktrees.Resolve(branch)
	if err != nil {
		wtPath = "" // Worktree may not exist yet
	}

	data := KeybindingTemplateData{
		Task:         task,
		TaskID:       task.ID,
		TaskTitle:    task.Title,
		TaskStatus:   string(task.Status),
		Branch:       branch,
		WorktreePath: wtPath,
	}

	return renderTemplate(cmdTemplate, data)
}

// renderTemplate renders a template string with the given data.
func renderTemplate(tmpl string, data any) (string, error) {
	t, err := template.New("keybinding").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// notifyWorker returns a command that sends a review comment to the worker.
func (m *Model) notifyWorker(taskID int, message string, requestChanges bool) tea.Cmd {
	return func() tea.Msg {
		uc := m.container.AddCommentUseCase()
		_, err := uc.Execute(context.Background(), usecase.AddCommentInput{
			TaskID:         taskID,
			Message:        message,
			RequestChanges: requestChanges,
		})
		if err != nil {
			return MsgError{Err: err}
		}
		return MsgReviewActionCompleted{TaskID: taskID}
	}
}

// editComment returns a command that updates an existing comment.
func (m *Model) editComment(taskID int, index int, message string) tea.Cmd {
	return func() tea.Msg {
		uc := m.container.EditCommentUseCase()
		err := uc.Execute(context.Background(), usecase.EditCommentInput{
			TaskID:  taskID,
			Index:   index,
			Message: message,
		})
		if err != nil {
			return MsgError{Err: err}
		}
		return MsgReviewActionCompleted{TaskID: taskID}
	}
}

// prepareEditReviewComment loads the review comment and enters edit mode.
func (m *Model) prepareEditReviewComment() tea.Cmd {
	return func() tea.Msg {
		// Load comments for the task
		comments, err := m.container.Tasks.GetComments(m.reviewTaskID)
		if err != nil {
			return MsgError{Err: err}
		}

		// Find the last reviewer comment (most recent review result)
		var reviewComment *domain.Comment
		var reviewIndex int
		for i := len(comments) - 1; i >= 0; i-- {
			if comments[i].Author == "reviewer" {
				reviewComment = &comments[i]
				reviewIndex = i
				break
			}
		}

		if reviewComment == nil {
			return MsgError{Err: fmt.Errorf("no review comment found")}
		}

		return MsgPrepareEditComment{
			TaskID:  m.reviewTaskID,
			Index:   reviewIndex,
			Message: reviewComment.Text,
		}
	}
}

// getEditor returns the user's preferred editor from environment variables.
// Returns an error if neither EDITOR nor VISUAL is set.
func getEditor() (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		return "", domain.ErrEditorNotSet
	}
	return editor, nil
}

// editorExecCmd wraps editor execution with pre/post file operations.
// It implements tea.ExecCommand interface.
type editorExecCmd struct {
	container  *app.Container
	stdin      io.Reader
	stdout     io.Writer
	stderr     io.Writer
	tmpPath    string
	origMD     string
	commentMD  string // original comment markdown (only used when isComment is true)
	taskID     int
	commentIdx int  // comment index (only used when isComment is true)
	isComment  bool // true for comment edit, false for task edit
}

func (c *editorExecCmd) Run() error {
	editor, err := getEditor()
	if err != nil {
		return err
	}

	// Build editor command
	// #nosec G204 - editor comes from trusted environment variable
	cmd := exec.Command(editor, c.tmpPath)
	cmd.Stdin = c.stdin
	cmd.Stdout = c.stdout
	cmd.Stderr = c.stderr

	if runErr := cmd.Run(); runErr != nil {
		return fmt.Errorf("editor failed: %w", runErr)
	}

	// Read edited content
	editedContent, err := os.ReadFile(c.tmpPath)
	if err != nil {
		return fmt.Errorf("failed to read edited file: %w", err)
	}

	if c.isComment {
		// Comment edit mode
		if string(editedContent) == c.commentMD {
			// No changes
			return nil
		}

		// Update comment
		uc := c.container.EditCommentUseCase()
		editErr := uc.Execute(context.Background(), usecase.EditCommentInput{
			TaskID:  c.taskID,
			Index:   c.commentIdx,
			Message: strings.TrimSpace(string(editedContent)),
		})
		if editErr != nil {
			return fmt.Errorf("failed to update comment: %w", editErr)
		}
	} else {
		// Task edit mode
		if string(editedContent) == c.origMD {
			// No changes
			return nil
		}

		// Update task with edited content
		uc := c.container.EditTaskUseCase()
		_, editErr := uc.Execute(context.Background(), usecase.EditTaskInput{
			TaskID:     c.taskID,
			EditorEdit: true,
			EditorText: string(editedContent),
		})
		if editErr != nil {
			return fmt.Errorf("failed to update task: %w", editErr)
		}
	}

	return nil
}

func (c *editorExecCmd) SetStdin(r io.Reader)  { c.stdin = r }
func (c *editorExecCmd) SetStdout(w io.Writer) { c.stdout = w }
func (c *editorExecCmd) SetStderr(w io.Writer) { c.stderr = w }

// editTaskInEditor returns a tea.Cmd that opens the task in an editor.
// After the editor closes, the task is updated with the edited content.
func (m *Model) editTaskInEditor(taskID int) tea.Cmd {
	return func() tea.Msg {
		// Get current task with comments
		showUC := m.container.ShowTaskUseCase()
		showOut, err := showUC.Execute(context.Background(), usecase.ShowTaskInput{
			TaskID: taskID,
		})
		if err != nil {
			return MsgError{Err: err}
		}

		task := showOut.Task

		// Create temporary file with task content
		tmpFile, err := os.CreateTemp("", fmt.Sprintf("crew-task-%d-*.md", taskID))
		if err != nil {
			return MsgError{Err: fmt.Errorf("failed to create temp file: %w", err)}
		}
		tmpPath := tmpFile.Name()

		// Write task as markdown with comments
		markdown := task.ToMarkdownWithComments(showOut.Comments)
		if _, writeErr := tmpFile.WriteString(markdown); writeErr != nil {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
			return MsgError{Err: fmt.Errorf("failed to write temp file: %w", writeErr)}
		}
		if closeErr := tmpFile.Close(); closeErr != nil {
			_ = os.Remove(tmpPath)
			return MsgError{Err: fmt.Errorf("failed to close temp file: %w", closeErr)}
		}

		return execEditTaskMsg{
			taskID:  taskID,
			tmpPath: tmpPath,
			origMD:  markdown,
		}
	}
}

// execEditTaskMsg triggers editor execution for task editing.
type execEditTaskMsg struct {
	tmpPath string
	origMD  string
	taskID  int
}

// loadReviewResult loads the latest reviewer comment for display in ModeReviewResult.
func (m *Model) loadReviewResult(taskID int) tea.Cmd {
	return func() tea.Msg {
		comments, err := m.container.Tasks.GetComments(taskID)
		if err != nil {
			return MsgError{Err: err}
		}

		// Find the last reviewer comment
		var reviewText string
		for i := len(comments) - 1; i >= 0; i-- {
			if comments[i].Author == "reviewer" {
				reviewText = comments[i].Text
				break
			}
		}

		if reviewText == "" {
			return MsgError{Err: fmt.Errorf("no review result found for task #%d", taskID)}
		}

		return MsgReviewResultLoaded{
			TaskID: taskID,
			Review: reviewText,
		}
	}
}

// isManagerSessionRunning checks if the manager session is currently running.
// Returns (running, error). This is a shared helper to avoid duplicating
// session existence checks across multiple methods.
// Returns an error if container or Sessions is nil (defensive guard for partially initialized models).
func (m *Model) isManagerSessionRunning() (bool, error) {
	if m.container == nil || m.container.Sessions == nil {
		return false, fmt.Errorf("session manager not initialized")
	}
	sessionName := domain.ManagerSessionName()
	return m.container.Sessions.IsRunning(sessionName)
}

// attachToManagerSession returns a tea.Cmd that attaches to the manager session.
func (m *Model) attachToManagerSession() tea.Cmd {
	socketPath := m.container.Config.SocketPath
	sessionName := domain.ManagerSessionName()
	cmd := domain.NewCommand("tmux", []string{"-S", socketPath, "attach", "-t", sessionName}, "")
	return tea.Exec(&domainExecCmd{cmd: cmd}, func(err error) tea.Msg {
		return MsgReloadTasks{}
	})
}

// startOrAttachManagerSessionForTask returns a tea.Cmd that starts or attaches to manager session.
// If a manager session is already running, it attaches to the existing session to preserve the conversation.
// The selected agent is only used when starting a new session.
func (m *Model) startOrAttachManagerSessionForTask(managerAgent string) tea.Cmd {
	return func() tea.Msg {
		running, err := m.isManagerSessionRunning()
		if err != nil {
			return MsgError{Err: fmt.Errorf("check manager session: %w", err)}
		}

		// If running, attach to existing session to preserve conversation
		// User should stop the session first if they want to switch agents
		if running {
			return MsgAttachManagerSession{}
		}

		// Start new manager session
		uc := m.container.StartManagerUseCase()
		out, err := uc.Execute(context.Background(), usecase.StartManagerInput{
			Name:    managerAgent,
			Session: true,
		})
		if err != nil {
			return MsgError{Err: fmt.Errorf("start manager session: %w", err)}
		}

		return MsgManagerSessionStarted{SessionName: out.SessionName}
	}
}

// checkAndAttachOrSelectManager checks if manager session is running and returns
// appropriate message. If running, returns MsgAttachManagerSession for immediate attach.
// If not running, returns MsgShowManagerSelect to show agent selection UI.
func (m *Model) checkAndAttachOrSelectManager() tea.Cmd {
	return func() tea.Msg {
		running, err := m.isManagerSessionRunning()
		if err != nil {
			return MsgError{Err: fmt.Errorf("check manager session: %w", err)}
		}

		if running {
			// Attach to existing session immediately (skip agent selection)
			return MsgAttachManagerSession{}
		}

		// Session not running, show agent selection UI
		return MsgShowManagerSelect{}
	}
}

// loadDiffContent returns a command that loads diff content for the panel.
func (m *Model) loadDiffContent(taskID int) tea.Cmd {
	return func() tea.Msg {
		uc := m.container.ShowDiffUseCaseForCommand()
		execCmd, err := uc.GetCommand(context.Background(), usecase.ShowDiffInput{
			TaskID: taskID,
		})
		if err != nil {
			return MsgDiffLoaded{TaskID: taskID, Content: fmt.Sprintf("Error: %v", err)}
		}

		// Execute the command and capture output
		// #nosec G204 - execCmd comes from trusted internal use case
		cmd := exec.Command(execCmd.Program, execCmd.Args...)
		cmd.Dir = execCmd.Dir

		output, err := cmd.Output()
		if err != nil {
			// diff can return non-zero when there are differences, check if output exists
			if len(output) > 0 {
				return MsgDiffLoaded{TaskID: taskID, Content: string(output)}
			}
			return MsgDiffLoaded{TaskID: taskID, Content: fmt.Sprintf("Error: %v", err)}
		}

		content := string(output)
		if content == "" {
			content = "No changes"
		}

		return MsgDiffLoaded{TaskID: taskID, Content: content}
	}
}

// loadPeekContent returns a command that loads peek content for the panel.
func (m *Model) loadPeekContent(taskID int) tea.Cmd {
	return func() tea.Msg {
		uc := m.container.PeekSessionUseCase()
		out, err := uc.Execute(context.Background(), usecase.PeekSessionInput{
			TaskID: taskID,
			Escape: true, // Preserve ANSI sequences
		})
		if err != nil {
			return MsgPeekLoaded{TaskID: taskID, Content: fmt.Sprintf("No running session: %v", err)}
		}

		content := out.Output
		if content == "" {
			content = "No output"
		}

		return MsgPeekLoaded{TaskID: taskID, Content: content}
	}
}
