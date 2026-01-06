package tui

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"
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
type Model struct {
	// Dependencies (pointers first for alignment)
	container *app.Container
	config    *domain.Config
	warnings  []string
	err       error

	// State (slices - contain pointers)
	tasks         []*domain.Task
	comments      []domain.Comment
	builtinAgents []string
	customAgents  []string
	agentCommands map[string]string

	// Components (structs with pointers)
	keys                KeyMap
	styles              Styles
	help                help.Model
	taskList            list.Model
	detailPanelViewport viewport.Model // For right pane detail panel

	// Input state (large structs)
	titleInput  textinput.Model
	descInput   textinput.Model
	parentInput textinput.Model
	filterInput textinput.Model
	customInput textinput.Model
	execInput   textinput.Model

	// Numeric state (smaller types last)
	mode             Mode
	confirmAction    ConfirmAction
	sortMode         SortMode
	newTaskField     NewTaskField
	width            int
	height           int
	confirmTaskID    int
	agentCursor      int
	statusCursor     int
	startFocusCustom bool
	showAll          bool
	detailFocused    bool // Right pane is focused for scrolling
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

	styles := DefaultStyles()
	delegate := newTaskDelegate(styles)
	taskList := list.New([]list.Item{}, delegate, 0, 0)
	taskList.SetShowTitle(false)
	taskList.SetShowStatusBar(false)
	taskList.SetShowHelp(false)
	taskList.SetShowPagination(false)
	taskList.SetFilteringEnabled(true)
	taskList.DisableQuitKeybindings()

	return &Model{
		container:        c,
		mode:             ModeNormal,
		tasks:            nil,
		keys:             DefaultKeyMap(),
		styles:           styles,
		help:             help.New(),
		taskList:         taskList,
		titleInput:       ti,
		descInput:        di,
		parentInput:      pi,
		filterInput:      fi,
		customInput:      ci,
		execInput:        ei,
		builtinAgents:    []string{"claude", "opencode", "codex"},
		customAgents:     nil,
		agentCommands:    make(map[string]string),
		agentCursor:      0,
		startFocusCustom: false,
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

// updateTaskList updates the task list items from tasks.
func (m *Model) updateTaskList() {
	sorted := m.sortedTasks()
	items := make([]list.Item, 0, len(sorted))
	for _, task := range sorted {
		items = append(items, taskItem{task: task})
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
	domain.StatusInReview:     0,
	domain.StatusInProgress:   1,
	domain.StatusNeedsInput:   1, // 追加
	domain.StatusNeedsChanges: 1, // 追加
	domain.StatusError:        2,
	domain.StatusStopped:      2, // 追加
	domain.StatusTodo:         3,
	domain.StatusDone:         4,
	domain.StatusClosed:       5,
}

func (m *Model) sortedTasks() []*domain.Task {
	tasks := make([]*domain.Task, len(m.tasks))
	copy(tasks, m.tasks)

	switch m.sortMode {
	case SortByStatus:
		sort.Slice(tasks, func(i, j int) bool {
			pi := statusPriority[tasks[i].Status]
			pj := statusPriority[tasks[j].Status]
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
func (m *Model) stopTask(taskID int) tea.Cmd {
	return func() tea.Msg {
		_, err := m.container.StopTaskUseCase().Execute(
			context.Background(),
			usecase.StopTaskInput{TaskID: taskID},
		)
		if err != nil {
			return MsgError{Err: err}
		}
		return MsgTaskStopped{TaskID: taskID}
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
func (m *Model) updateAgents() {
	if m.config == nil {
		return
	}

	// Build agent command previews
	m.agentCommands = make(map[string]string)

	// Filter agents: only show worker-role agents that are not hidden
	m.builtinAgents = []string{}
	m.customAgents = []string{}
	for name, agentDef := range m.config.Agents {
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

// tmuxAttachCmd implements tea.ExecCommand for attaching to a tmux session.
type tmuxAttachCmd struct {
	stdin       io.Reader
	stdout      io.Writer
	stderr      io.Writer
	socketPath  string
	sessionName string
}

func (c *tmuxAttachCmd) Run() error {
	// #nosec G204 - socketPath and sessionName are trusted internal values
	cmd := exec.Command("tmux", "-S", c.socketPath, "attach", "-t", c.sessionName)
	cmd.Stdin = c.stdin
	cmd.Stdout = c.stdout
	cmd.Stderr = c.stderr
	return cmd.Run()
}

func (c *tmuxAttachCmd) SetStdin(r io.Reader)  { c.stdin = r }
func (c *tmuxAttachCmd) SetStdout(w io.Writer) { c.stdout = w }
func (c *tmuxAttachCmd) SetStderr(w io.Writer) { c.stderr = w }

// attachToSession returns a tea.Cmd that attaches to a tmux session.
// After the attach completes (user detaches), it triggers a task reload.
func (m *Model) attachToSession(taskID int) tea.Cmd {
	socketPath := m.container.Config.SocketPath
	sessionName := domain.SessionName(taskID)

	return tea.Exec(&tmuxAttachCmd{
		socketPath:  socketPath,
		sessionName: sessionName,
	}, func(err error) tea.Msg {
		// Reload tasks after detaching from the session
		return MsgReloadTasks{}
	})
}

// diffExecCmd implements tea.ExecCommand for showing diff.
type diffExecCmd struct {
	stdin        io.Reader
	stdout       io.Writer
	stderr       io.Writer
	worktreePath string
	diffCommand  string
}

func (c *diffExecCmd) Run() error {
	// #nosec G204 - diffCommand is from config, trusted
	cmd := exec.Command("sh", "-c", c.diffCommand)
	cmd.Dir = c.worktreePath
	cmd.Stdin = c.stdin
	cmd.Stdout = c.stdout
	cmd.Stderr = c.stderr
	// Ignore exit code as diff can return non-zero when there are differences
	_ = cmd.Run()
	return nil
}

func (c *diffExecCmd) SetStdin(r io.Reader)  { c.stdin = r }
func (c *diffExecCmd) SetStdout(w io.Writer) { c.stdout = w }
func (c *diffExecCmd) SetStderr(w io.Writer) { c.stderr = w }

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

// execCmd implements tea.ExecCommand for executing a command in a worktree.
type execCmd struct {
	stdin        io.Reader
	stdout       io.Writer
	stderr       io.Writer
	worktreePath string
	command      string
}

func (c *execCmd) Run() error {
	// #nosec G204 - command is user-provided, trusted in this context
	cmd := exec.Command("sh", "-c", c.command)
	cmd.Dir = c.worktreePath
	cmd.Stdin = c.stdin
	cmd.Stdout = c.stdout
	cmd.Stderr = c.stderr
	return cmd.Run()
}

func (c *execCmd) SetStdin(r io.Reader)  { c.stdin = r }
func (c *execCmd) SetStdout(w io.Writer) { c.stdout = w }
func (c *execCmd) SetStderr(w io.Writer) { c.stderr = w }

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

	return tea.Exec(&execCmd{
		worktreePath: wtPath,
		command:      command,
	}, func(err error) tea.Msg {
		return MsgReloadTasks{}
	})
}
