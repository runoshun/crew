package tui

import (
	"context"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
	filteredTasks []*domain.Task
	builtinAgents []string          // Built-in agent names (claude, opencode, codex)
	customAgents  []string          // Custom agent names from config
	agentCommands map[string]string // Agent name -> command preview

	// Components (structs with pointers)
	keys   KeyMap
	styles Styles
	help   help.Model

	// Input state (large structs)
	titleInput  textinput.Model
	descInput   textinput.Model
	filterInput textinput.Model
	customInput textinput.Model // Custom agent command input

	// Numeric state (smaller types last)
	mode             Mode
	confirmAction    ConfirmAction
	cursor           int
	width            int
	height           int
	confirmTaskID    int
	agentCursor      int
	startFocusCustom bool // true = focus on custom input, false = focus on agent list
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

	return &Model{
		container:        c,
		mode:             ModeNormal,
		tasks:            nil,
		filteredTasks:    nil,
		cursor:           0,
		keys:             DefaultKeyMap(),
		styles:           DefaultStyles(),
		help:             help.New(),
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
	tasks := m.visibleTasks()
	if len(tasks) == 0 || m.cursor < 0 || m.cursor >= len(tasks) {
		return nil
	}
	return tasks[m.cursor]
}

// visibleTasks returns the tasks that should be displayed.
func (m *Model) visibleTasks() []*domain.Task {
	if m.filterInput.Value() != "" && m.filteredTasks != nil {
		return m.filteredTasks
	}
	return m.tasks
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
