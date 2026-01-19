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
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase"
)

const autoRefreshInterval = 5 * time.Second

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
	case domain.StatusTodo, domain.StatusError, domain.StatusStopped:
		return "start"
	case domain.StatusInProgress, domain.StatusNeedsInput:
		return "attach"
	case domain.StatusReviewing:
		return "attach_review"
	case domain.StatusForReview:
		return "show_diff"
	case domain.StatusReviewed:
		return "review_result"
	case domain.StatusClosed:
		return "detail"
	default:
		if task.Status.IsLegacyDone() {
			return "detail"
		}
	}
	return ""
}

type Model struct {
	// Dependencies (pointers first for alignment)
	container *app.Container
	config    *domain.Config
	warnings  []string
	err       error

	// State (slices - contain pointers)
	tasks           []*domain.Task
	comments        []domain.Comment
	commentCounts   map[int]int // taskID -> comment count
	builtinAgents   []string
	customAgents    []string
	agentCommands   map[string]string
	customKeybinds  map[string]domain.TUIKeybinding
	keybindWarnings []string
	reviewResult    string // Review result text
	actionMenuItems []actionMenuItem

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

	// Numeric state (smaller types last)
	mode                 Mode
	confirmAction        ConfirmAction
	sortMode             SortMode
	newTaskField         NewTaskField
	width                int
	height               int
	confirmTaskID        int
	agentCursor          int
	statusCursor         int
	actionMenuCursor     int
	reviewTaskID         int // Task being reviewed
	reviewActionCursor   int // Cursor for action selection
	editCommentIndex     int // Index of comment being edited
	startFocusCustom     bool
	showAll              bool
	detailFocused        bool // Right pane is focused for scrolling
	confirmReviewSession bool
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
	ri.Placeholder = "Message to worker (optional, Enter to skip)"
	ri.CharLimit = 500

	eci := textinput.New()
	eci.Placeholder = "Edit review comment..."
	eci.CharLimit = 2000

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
		reviewViewport:     reviewVp,
		builtinAgents:      []string{"claude", "opencode", "codex"},
		customAgents:       nil,
		agentCommands:      make(map[string]string),
		customKeybinds:     make(map[string]domain.TUIKeybinding),
		keybindWarnings:    nil,
		commentCounts:      make(map[int]int),
		agentCursor:        0,
		startFocusCustom:   false,
	}
}

// Init initializes the model and returns the initial command.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadTasks(),
		m.loadConfig(),
		m.tick(),
	)
}

// tick returns a command that sends a tick message after the refresh interval.
func (m *Model) tick() tea.Cmd {
	return tea.Tick(autoRefreshInterval, func(t time.Time) tea.Msg {
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
	if task.Status == domain.StatusInProgress || task.Status == domain.StatusNeedsInput || task.Status == domain.StatusReviewing {
		return true
	}
	return task.Session != ""
}

func (m *Model) hasWorktree(task *domain.Task) bool {
	if task == nil {
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

// updateTaskList updates the task list items from tasks.
func (m *Model) updateTaskList() {
	sorted := m.sortedTasks()
	items := make([]list.Item, 0, len(sorted))
	for _, task := range sorted {
		count := m.commentCounts[task.ID]
		items = append(items, taskItem{task: task, commentCount: count})
	}
	m.taskList.SetItems(items)
}

// updateDetailPanelViewport updates the viewport size and content for the right pane.
func (m *Model) updateDetailPanelViewport() {
	if !m.showDetailPanel() {
		return
	}
	panelWidth := m.detailPanelWidth() - 4 // account for border and padding
	panelHeight := m.height - 6            // account for header/footer and hint
	if panelHeight < 10 {
		panelHeight = 10
	}
	m.detailPanelViewport.Width = panelWidth
	m.detailPanelViewport.Height = panelHeight
	m.detailPanelViewport.SetContent(m.detailPanelContent(panelWidth))
}

// updateLayoutSizes updates all layout-dependent sizes (taskList, viewport).
// Call this when detailFocused changes or window size changes.
func (m *Model) updateLayoutSizes() {
	// Set taskList width to headerFooterContentWidth for consistent alignment
	// Header/Footer use this same width with Padding(0, 1) which adds 2 to total width
	// but lipgloss Width() includes padding, so the content area matches
	listW := m.headerFooterContentWidth()
	m.taskList.SetSize(listW, m.height-8)
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

	// Re-render content with word wrap and dialog background color when size changes
	if m.reviewResult != "" {
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
	domain.StatusReviewing:  0, // Review in progress
	domain.StatusInProgress: 1, // Work in progress
	domain.StatusNeedsInput: 1, // Waiting for input
	domain.StatusForReview:  2, // Awaiting review
	domain.StatusReviewed:   2, // Review complete
	domain.StatusError:      3,
	domain.StatusStopped:    3,
	domain.StatusTodo:       4,
	domain.StatusClosed:     5,
}

// getStatusPriority returns the sort priority for a status.
// Handles legacy "done" status by treating it as closed.
func getStatusPriority(status domain.Status) int {
	if p, ok := statusPriority[status]; ok {
		return p
	}
	// Handle legacy "done" status as closed (priority 5)
	if status.IsLegacyDone() {
		return statusPriority[domain.StatusClosed]
	}
	// Unknown status goes to the end
	return 99
}

func (m *Model) sortedTasks() []*domain.Task {
	tasks := make([]*domain.Task, len(m.tasks))
	copy(tasks, m.tasks)

	switch m.sortMode {
	case SortByStatus:
		sort.Slice(tasks, func(i, j int) bool {
			pi := getStatusPriority(tasks[i].Status)
			pj := getStatusPriority(tasks[j].Status)
			if pi != pj {
				return pi < pj
			}
			return tasks[i].ID < tasks[j].ID
		})
	case SortByID:
		sort.Slice(tasks, func(i, j int) bool {
			return tasks[i].ID < tasks[j].ID
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
func (m *Model) stopTask(taskID int, review bool) tea.Cmd {
	return func() tea.Msg {
		out, err := m.container.StopTaskUseCase().Execute(
			context.Background(),
			usecase.StopTaskInput{TaskID: taskID, Review: review},
		)
		if err != nil {
			return MsgError{Err: err}
		}
		return MsgTaskStopped{TaskID: taskID, Review: review, SessionName: out.SessionName}
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
func (m *Model) copyTask(taskID int) tea.Cmd {
	return func() tea.Msg {
		out, err := m.container.CopyTaskUseCase().Execute(
			context.Background(),
			usecase.CopyTaskInput{SourceID: taskID},
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

// updateAgents updates the agent lists from config.
// Agents with role=worker and hidden=false are shown in the TUI.
func (m *Model) actionMenuItemsForTask(task *domain.Task) []actionMenuItem {
	if task == nil {
		return nil
	}

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
				return task.Status.CanStart()
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
			ActionID: "attach_review",
			Label:    "Attach Review",
			Desc:     "Open review session",
			Key:      "A",
			Action: func() (tea.Model, tea.Cmd) {
				return m, func() tea.Msg {
					return MsgAttachSession{TaskID: task.ID, Review: true}
				}
			},
			IsAvailable: func() bool {
				return task.Status == domain.StatusReviewing
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
				return task.Status == domain.StatusForReview
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
				return task.Status == domain.StatusReviewed
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
				return m.hasWorktree(task)
			},
		},
		{
			ActionID: "review",
			Label:    "Review",
			Desc:     "Run reviewer session",
			Key:      "R",
			Action: func() (tea.Model, tea.Cmd) {
				return m, m.reviewTask(task.ID)
			},
			IsAvailable: func() bool {
				return m.hasWorktree(task)
			},
		},
		{
			ActionID: "copy",
			Label:    "Copy",
			Desc:     "Duplicate task",
			Key:      "y",
			Action: func() (tea.Model, tea.Cmd) {
				return m, m.copyTask(task.ID)
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
				return task.Status == domain.StatusForReview || task.Status == domain.StatusReviewed
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
	for name, agentDef := range m.config.EnabledAgents() {
		// Skip hidden agents and non-worker roles
		if agentDef.Hidden || (agentDef.Role != "" && agentDef.Role != domain.RoleWorker) {
			continue
		}
		m.builtinAgents = append(m.builtinAgents, name)
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

	// Set cursor to default agent
	allAgents := m.allAgents()
	for i, a := range allAgents {
		if a == m.config.AgentsConfig.DefaultWorker {
			m.agentCursor = i
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
// If review is true, attaches to the review session instead of the work session.
func (m *Model) attachToSession(taskID int, review bool) tea.Cmd {
	socketPath := m.container.Config.SocketPath
	var sessionName string
	if review {
		sessionName = domain.ReviewSessionName(taskID)
	} else {
		sessionName = domain.SessionName(taskID)
	}

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
			TaskID:        taskID,
			UseTUICommand: true, // Use tui_command for TUI context
		})
		if err != nil {
			return MsgError{Err: err}
		}

		return execDiffMsg{cmd: execCmd}
	}
}

// execDiffMsg is an internal message to trigger diff execution.
type execDiffMsg struct {
	cmd *domain.ExecCommand
}

// executeCommand returns a tea.Cmd that executes a command in a task's worktree.
func (m *Model) executeCommand(command string) tea.Cmd {
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

	// Execute the command
	return m, m.executeCommand(command)
}

// renderKeybindingTemplate renders a keybinding command template with task data.
func (m *Model) renderKeybindingTemplate(cmdTemplate string, task *domain.Task) (string, error) {
	branch := domain.BranchName(task.ID, task.Issue)
	wtPath, err := m.container.Worktrees.Resolve(branch)
	if err != nil {
		wtPath = "" // Worktree may not exist yet
	}

	data := map[string]interface{}{
		"TaskID":       task.ID,
		"TaskTitle":    task.Title,
		"TaskStatus":   string(task.Status),
		"Branch":       branch,
		"WorktreePath": wtPath,
	}

	return renderTemplate(cmdTemplate, data)
}

// renderTemplate renders a template string with the given data.
func renderTemplate(tmpl string, data map[string]interface{}) (string, error) {
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

// reviewTask returns a command that reviews a task using the AI reviewer.
func (m *Model) reviewTask(taskID int) tea.Cmd {
	return func() tea.Msg {
		uc := m.container.ReviewTaskUseCase(io.Discard, io.Discard)
		_, err := uc.Execute(context.Background(), usecase.ReviewTaskInput{
			TaskID: taskID,
			Wait:   false, // TUI uses background execution (tmux review session)
		})
		if err != nil {
			return MsgError{Err: err}
		}
		return MsgReloadTasks{}
	}
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
		return MsgReviewActionCompleted{TaskID: taskID, Action: ReviewActionNotifyWorker}
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
		return MsgReviewActionCompleted{TaskID: taskID, Action: ReviewActionEditComment}
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
