package tui

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"sort"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase"
)

// Model is the main bubbletea model for the TUI.
type Model struct {
	// Dependencies (pointers first for alignment)
	container *app.Container
	config    *domain.Config
	err       error

	// State (slices - contain pointers)
	tasks         []*domain.Task
	builtinAgents []string
	customAgents  []string
	agentCommands map[string]string

	// Components (structs with pointers)
	keys           KeyMap
	styles         Styles
	help           help.Model
	taskList       list.Model
	detailViewport viewport.Model

	// Input state (large structs)
	titleInput  textinput.Model
	descInput   textinput.Model
	filterInput textinput.Model
	customInput textinput.Model

	// Numeric state (smaller types last)
	mode             Mode
	confirmAction    ConfirmAction
	sortMode         SortMode
	width            int
	height           int
	confirmTaskID    int
	agentCursor      int
	startFocusCustom bool
}

// New creates a new TUI Model with the given container.
func New(c *app.Container) *Model {
	ti := textinput.New()
	ti.Placeholder = "Task title"
	ti.CharLimit = 200

	di := textinput.New()
	di.Placeholder = "Task description (optional)"
	di.CharLimit = 1000

	fi := textinput.New()
	fi.Placeholder = "Filter tasks..."
	fi.CharLimit = 100

	ci := textinput.New()
	ci.Placeholder = "Enter custom command..."
	ci.CharLimit = 500

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
		filterInput:      fi,
		customInput:      ci,
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
	)
}

// loadTasks returns a command that loads tasks from the repository.
func (m *Model) loadTasks() tea.Cmd {
	return func() tea.Msg {
		out, err := m.container.ListTasksUseCase().Execute(context.Background(), usecase.ListTasksInput{})
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

func (m *Model) initDetailViewport() {
	width := m.width - 12
	height := m.height - 10
	if width < 40 {
		width = 40
	}
	if height < 10 {
		height = 10
	}
	m.detailViewport = viewport.New(width, height)
	m.detailViewport.Style = lipgloss.NewStyle().Background(Colors.Background)
	m.detailViewport.SetContent(m.detailContent(width))
}

func (m *Model) detailContent(width int) string {
	task := m.SelectedTask()
	if task == nil {
		return "No task selected"
	}

	lineStyle := lipgloss.NewStyle().
		Width(width).
		Background(Colors.Background)

	wrapStyle := lipgloss.NewStyle().
		Width(width).
		Background(Colors.Background)

	var lines []string

	lines = append(lines, lineStyle.Render(m.styles.DetailTitle.Background(Colors.Background).Render(fmt.Sprintf("Task #%d", task.ID))))
	lines = append(lines, wrapStyle.Render(m.styles.TaskTitleSelected.Background(Colors.Background).Render(task.Title)))
	lines = append(lines, lineStyle.Render(""))

	lines = append(lines, lineStyle.Render(
		m.styles.DetailLabel.Background(Colors.Background).Render("Status")+
			m.styles.StatusStyle(task.Status).Background(Colors.Background).Render(string(task.Status))))

	if task.Agent != "" {
		lines = append(lines, lineStyle.Render(
			m.styles.DetailLabel.Background(Colors.Background).Render("Agent")+
				m.styles.DetailValue.Background(Colors.Background).Render(task.Agent)))
	}

	if task.Session != "" {
		lines = append(lines, lineStyle.Render(
			m.styles.DetailLabel.Background(Colors.Background).Render("Session")+
				m.styles.DetailValue.Background(Colors.Background).Render(task.Session)))
	}

	lines = append(lines, lineStyle.Render(
		m.styles.DetailLabel.Background(Colors.Background).Render("Created")+
			m.styles.DetailValue.Background(Colors.Background).Render(task.Created.Format("2006-01-02 15:04"))))

	if !task.Started.IsZero() {
		lines = append(lines, lineStyle.Render(
			m.styles.DetailLabel.Background(Colors.Background).Render("Started")+
				m.styles.DetailValue.Background(Colors.Background).Render(task.Started.Format("2006-01-02 15:04"))))
	}

	if task.Description != "" {
		lines = append(lines, lineStyle.Render(""))
		lines = append(lines, lineStyle.Render(m.styles.DetailLabel.Background(Colors.Background).Render("Description")))
		lines = append(lines, wrapStyle.Render(m.styles.DetailDesc.Background(Colors.Background).Width(width).Render(task.Description)))
	}

	result := ""
	for i, line := range lines {
		result += line
		if i < len(lines)-1 {
			result += "\n"
		}
	}
	return result
}

var statusPriority = map[domain.Status]int{
	domain.StatusInReview:   0,
	domain.StatusInProgress: 1,
	domain.StatusError:      2,
	domain.StatusTodo:       3,
	domain.StatusDone:       4,
	domain.StatusClosed:     5,
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

// updateAgents updates the agent lists from config.
func (m *Model) updateAgents() {
	if m.config == nil {
		return
	}

	// Build agent command previews
	m.agentCommands = make(map[string]string)

	// Built-in agents with their commands
	m.builtinAgents = []string{}
	for name, builtin := range domain.BuiltinWorkers {
		m.builtinAgents = append(m.builtinAgents, name)
		m.agentCommands[name] = builtin.Command
	}

	// Custom agents from config (exclude built-ins)
	m.customAgents = []string{}
	for name, worker := range m.config.Workers {
		if _, isBuiltin := domain.BuiltinWorkers[name]; !isBuiltin {
			m.customAgents = append(m.customAgents, name)
			m.agentCommands[name] = worker.Command
		}
	}

	// Set cursor to default agent
	if m.config.WorkersConfig.Default != "" {
		allAgents := m.allAgents()
		for i, a := range allAgents {
			if a == m.config.WorkersConfig.Default {
				m.agentCursor = i
				break
			}
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
			TaskID: taskID,
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
