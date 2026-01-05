package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/runoshun/git-crew/v2/internal/domain"
)

// Update handles messages and updates the model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		// Update all layout-dependent sizes
		m.updateLayoutSizes()
		return m, nil

	case MsgTasksLoaded:
		m.tasks = msg.Tasks
		m.updateTaskList()
		// Load comments for selected task if detail panel is visible
		task := m.SelectedTask()
		if task != nil && (m.showDetailPanel() || m.mode == ModeDetail) {
			return m, m.loadComments(task.ID)
		}
		return m, nil

	case MsgConfigLoaded:
		m.config = msg.Config
		m.warnings = msg.Config.Warnings
		m.updateAgents()
		return m, nil

	case MsgTaskStarted:
		return m, m.loadTasks()

	case MsgTaskStopped:
		m.mode = ModeNormal
		m.confirmAction = ConfirmNone
		return m, m.loadTasks()

	case MsgTaskCreated:
		m.mode = ModeNormal
		m.titleInput.Reset()
		m.descInput.Reset()
		m.parentInput.Reset()
		return m, m.loadTasks()

	case MsgTaskDeleted:
		m.mode = ModeNormal
		m.confirmAction = ConfirmNone
		return m, m.loadTasks()

	case MsgTaskClosed:
		m.mode = ModeNormal
		m.confirmAction = ConfirmNone
		return m, m.loadTasks()

	case MsgTaskMerged:
		m.mode = ModeNormal
		m.confirmAction = ConfirmNone
		return m, m.loadTasks()

	case MsgTaskCopied:
		return m, m.loadTasks()

	case MsgError:
		m.err = msg.Err
		m.mode = ModeNormal
		m.confirmAction = ConfirmNone
		return m, nil

	case MsgClearError:
		m.err = nil
		return m, nil

	case MsgAttachSession:
		// Use tea.Exec to attach to tmux session, returning to TUI after detach
		return m, m.attachToSession(msg.TaskID)

	case MsgReloadTasks:
		// Reload tasks after returning from external commands
		return m, m.loadTasks()

	case MsgShowDiff:
		// Trigger diff display for a task
		return m, m.showDiff(msg.TaskID)

	case execDiffMsg:
		// Execute the diff command using ExecCommand from domain
		return m, tea.Exec(&diffExecCmd{
			worktreePath: msg.cmd.Dir,
			diffCommand:  msg.cmd.Args[1], // Args is ["−c", "expanded_command"]
		}, func(err error) tea.Msg {
			return MsgReloadTasks{}
		})

	case MsgTick:
		// Auto-refresh: reload tasks and schedule next tick
		return m, tea.Batch(m.loadTasks(), m.tick())

	case MsgCommentsLoaded:
		m.comments = msg.Comments
		// Update viewport content if in detail mode (dialog)
		if m.mode == ModeDetail {
			width := m.dialogWidth() - 4
			m.detailViewport.SetContent(m.detailContent(width))
		}
		// Update detail panel viewport if showing
		if m.showDetailPanel() {
			m.updateDetailPanelViewport()
		}
		return m, nil
	}

	return m, nil
}

// handleKeyMsg handles keyboard input.
func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Clear error on any key press
	if m.err != nil {
		m.err = nil
	}

	switch m.mode {
	case ModeNormal:
		return m.handleNormalMode(msg)
	case ModeFilter:
		return m.handleFilterMode(msg)
	case ModeConfirm:
		return m.handleConfirmMode(msg)
	case ModeInputTitle:
		return m.handleInputTitleMode(msg)
	case ModeInputDesc:
		return m.handleInputDescMode(msg)
	case ModeNewTask:
		return m.handleNewTaskMode(msg)
	case ModeStart:
		return m.handleStartMode(msg)
	case ModeHelp:
		return m.handleHelpMode(msg)
	case ModeDetail:
		return m.handleDetailMode(msg)
	}

	return m, nil
}

// handleNormalMode handles keys in normal mode.
func (m *Model) handleNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// When detail panel is focused, handle scrolling keys
	if m.detailFocused {
		return m.handleDetailPanelFocused(msg)
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Up), key.Matches(msg, m.keys.Down):
		prevTask := m.SelectedTask()
		var cmd tea.Cmd
		m.taskList, cmd = m.taskList.Update(msg)
		newTask := m.SelectedTask()
		// If task changed and we're showing detail panel or detail view, load comments
		if prevTask != newTask && newTask != nil {
			if m.showDetailPanel() || m.mode == ModeDetail {
				// Update viewport content immediately, comments will update async
				m.updateDetailPanelViewport()
				return m, tea.Batch(cmd, m.loadComments(newTask.ID))
			}
		}
		return m, cmd

	case key.Matches(msg, m.keys.Enter):
		return m.handleSmartAction()

	case key.Matches(msg, m.keys.Start):
		task := m.SelectedTask()
		if task == nil || !task.Status.CanStart() {
			return m, nil
		}
		m.mode = ModeStart
		return m, nil

	case key.Matches(msg, m.keys.Stop):
		task := m.SelectedTask()
		if task == nil || task.Status != domain.StatusInProgress {
			return m, nil
		}
		m.mode = ModeConfirm
		m.confirmAction = ConfirmStop
		m.confirmTaskID = task.ID
		return m, nil

	case key.Matches(msg, m.keys.Attach):
		task := m.SelectedTask()
		if task == nil || !task.IsRunning() {
			return m, nil
		}
		return m, func() tea.Msg {
			return MsgAttachSession{TaskID: task.ID}
		}

	case key.Matches(msg, m.keys.New):
		m.mode = ModeNewTask
		m.newTaskField = FieldTitle
		m.titleInput.Focus()
		m.descInput.Blur()
		m.parentInput.Blur()
		return m, nil

	case key.Matches(msg, m.keys.Copy):
		task := m.SelectedTask()
		if task == nil {
			return m, nil
		}
		return m, m.copyTask(task.ID)

	case key.Matches(msg, m.keys.Delete):
		task := m.SelectedTask()
		if task == nil {
			return m, nil
		}
		m.mode = ModeConfirm
		m.confirmAction = ConfirmDelete
		m.confirmTaskID = task.ID
		return m, nil

	case key.Matches(msg, m.keys.Merge):
		task := m.SelectedTask()
		if task == nil || task.Status != domain.StatusInReview {
			return m, nil
		}
		m.mode = ModeConfirm
		m.confirmAction = ConfirmMerge
		m.confirmTaskID = task.ID
		return m, nil

	case key.Matches(msg, m.keys.Close):
		task := m.SelectedTask()
		if task == nil || task.Status.IsTerminal() {
			return m, nil
		}
		m.mode = ModeConfirm
		m.confirmAction = ConfirmClose
		m.confirmTaskID = task.ID
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		return m, m.loadTasks()

	case key.Matches(msg, m.keys.Filter):
		m.mode = ModeFilter
		m.filterInput.Focus()
		return m, nil

	case key.Matches(msg, m.keys.Sort):
		m.sortMode = m.sortMode.Next()
		m.updateTaskList()
		return m, nil

	case key.Matches(msg, m.keys.Help):
		m.mode = ModeHelp
		return m, nil

	case key.Matches(msg, m.keys.Detail):
		task := m.SelectedTask()
		if task == nil {
			return m, nil
		}
		// If detail panel is visible (wide screen), toggle focus instead of dialog
		if m.showDetailPanel() {
			m.detailFocused = true
			m.updateLayoutSizes() // Update both taskList and viewport sizes
			return m, m.loadComments(task.ID)
		}
		// Fall back to dialog mode for narrow screens
		m.initDetailViewport()
		m.mode = ModeDetail
		return m, m.loadComments(task.ID)

	case key.Matches(msg, m.keys.ToggleShowAll):
		m.showAll = !m.showAll
		return m, m.loadTasks()
	}

	return m, nil
}

// handleSmartAction performs context-aware action on Enter.
func (m *Model) handleSmartAction() (tea.Model, tea.Cmd) {
	task := m.SelectedTask()
	if task == nil {
		return m, nil
	}

	switch task.Status {
	case domain.StatusTodo, domain.StatusError, domain.StatusStopped:
		// Start the task
		m.mode = ModeStart
		return m, nil

	case domain.StatusInProgress, domain.StatusNeedsInput, domain.StatusNeedsChanges:
		// Attach to session (needs_input/needs_changes - task is running but waiting)
		return m, func() tea.Msg {
			return MsgAttachSession{TaskID: task.ID}
		}

	case domain.StatusInReview:
		// Show diff for review (attach is available via 'a' key)
		return m, func() tea.Msg {
			return MsgShowDiff{TaskID: task.ID}
		}

	case domain.StatusDone, domain.StatusClosed:
		// Show detail view for terminal states
		m.mode = ModeDetail
		return m, nil
	}

	return m, nil
}

// handleFilterMode handles keys in filter mode.
func (m *Model) handleFilterMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape):
		m.mode = ModeNormal
		m.filterInput.Reset()
		m.updateTaskList()
		return m, nil

	case msg.Type == tea.KeyEnter:
		m.mode = ModeNormal
		m.filterInput.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.applyFilter()
	return m, cmd
}

func (m *Model) applyFilter() {
	query := strings.ToLower(m.filterInput.Value())
	if query == "" {
		m.updateTaskList()
		return
	}

	var filtered []*domain.Task
	for _, t := range m.tasks {
		if strings.Contains(strings.ToLower(t.Title), query) ||
			strings.Contains(strings.ToLower(string(t.Status)), query) ||
			strings.Contains(strings.ToLower(t.Agent), query) {
			filtered = append(filtered, t)
		}
	}

	items := make([]list.Item, 0, len(filtered))
	for _, task := range filtered {
		items = append(items, taskItem{task: task})
	}
	m.taskList.SetItems(items)
}

// handleConfirmMode handles keys in confirm mode.
func (m *Model) handleConfirmMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape), msg.String() == "n", msg.String() == "N":
		m.mode = ModeNormal
		m.confirmAction = ConfirmNone
		return m, nil

	case key.Matches(msg, m.keys.Confirm):
		switch m.confirmAction {
		case ConfirmNone:
			// Nothing to confirm
		case ConfirmDelete:
			return m, m.deleteTask(m.confirmTaskID)
		case ConfirmClose:
			return m, m.closeTask(m.confirmTaskID)
		case ConfirmStop:
			return m, m.stopTask(m.confirmTaskID)
		case ConfirmMerge:
			return m, m.mergeTask(m.confirmTaskID)
		}
	}

	return m, nil
}

// handleInputTitleMode handles keys in title input mode.
func (m *Model) handleInputTitleMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape):
		m.mode = ModeNormal
		m.titleInput.Reset()
		return m, nil

	case msg.Type == tea.KeyEnter:
		if m.titleInput.Value() == "" {
			return m, nil
		}
		m.mode = ModeInputDesc
		m.titleInput.Blur()
		m.descInput.Focus()
		return m, nil
	}

	var cmd tea.Cmd
	m.titleInput, cmd = m.titleInput.Update(msg)
	return m, cmd
}

// handleInputDescMode handles keys in description input mode.
func (m *Model) handleInputDescMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape):
		m.mode = ModeInputTitle
		m.descInput.Reset()
		m.titleInput.Focus()
		return m, nil

	case msg.Type == tea.KeyEnter:
		title := m.titleInput.Value()
		desc := m.descInput.Value()
		return m, m.createTask(title, desc)
	}

	var cmd tea.Cmd
	m.descInput, cmd = m.descInput.Update(msg)
	return m, cmd
}

// handleNewTaskMode handles keys in new task form mode.
func (m *Model) handleNewTaskMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape):
		m.mode = ModeNormal
		m.titleInput.Reset()
		m.descInput.Reset()
		m.parentInput.Reset()
		return m, nil

	case msg.Type == tea.KeyTab:
		// Move to next field
		m.newTaskField = m.newTaskField.Next()
		m.focusNewTaskField()
		return m, nil

	case msg.Type == tea.KeyShiftTab:
		// Move to previous field
		m.newTaskField = m.newTaskField.Prev()
		m.focusNewTaskField()
		return m, nil

	case msg.Type == tea.KeyEnter:
		title := m.titleInput.Value()
		if title == "" {
			return m, nil
		}
		desc := m.descInput.Value()
		parent := m.parentInput.Value()
		return m, m.createTaskWithParent(title, desc, parent)
	}

	// Forward to current input field
	var cmd tea.Cmd
	switch m.newTaskField {
	case FieldTitle:
		m.titleInput, cmd = m.titleInput.Update(msg)
	case FieldDesc:
		m.descInput, cmd = m.descInput.Update(msg)
	case FieldParent:
		m.parentInput, cmd = m.parentInput.Update(msg)
	}
	return m, cmd
}

// focusNewTaskField focuses the current field in new task form.
func (m *Model) focusNewTaskField() {
	m.titleInput.Blur()
	m.descInput.Blur()
	m.parentInput.Blur()

	switch m.newTaskField {
	case FieldTitle:
		m.titleInput.Focus()
	case FieldDesc:
		m.descInput.Focus()
	case FieldParent:
		m.parentInput.Focus()
	}
}

// handleStartMode handles keys in agent picker mode.
func (m *Model) handleStartMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle Tab to switch focus between list and custom input
	if msg.Type == tea.KeyTab {
		m.startFocusCustom = !m.startFocusCustom
		if m.startFocusCustom {
			m.customInput.Focus()
		} else {
			m.customInput.Blur()
		}
		return m, nil
	}

	// If custom input is focused, handle text input
	if m.startFocusCustom {
		return m.handleStartModeCustomInput(msg)
	}

	// Handle agent list navigation
	return m.handleStartModeList(msg)
}

// handleStartModeList handles keys when agent list is focused.
func (m *Model) handleStartModeList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	allAgents := m.allAgents()

	switch {
	case key.Matches(msg, m.keys.Escape):
		m.mode = ModeNormal
		m.customInput.Reset()
		m.startFocusCustom = false
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.agentCursor > 0 {
			m.agentCursor--
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.agentCursor < len(allAgents)-1 {
			m.agentCursor++
		}
		return m, nil

	case msg.Type == tea.KeyEnter:
		task := m.SelectedTask()
		if task == nil {
			m.mode = ModeNormal
			return m, nil
		}
		if len(allAgents) == 0 {
			m.mode = ModeNormal
			return m, nil
		}
		agent := allAgents[m.agentCursor]
		m.mode = ModeNormal
		m.customInput.Reset()
		m.startFocusCustom = false
		return m, m.startTask(task.ID, agent)
	}

	return m, nil
}

// handleStartModeCustomInput handles keys when custom input is focused.
func (m *Model) handleStartModeCustomInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape):
		m.mode = ModeNormal
		m.customInput.Reset()
		m.startFocusCustom = false
		return m, nil

	case msg.Type == tea.KeyEnter:
		task := m.SelectedTask()
		if task == nil {
			m.mode = ModeNormal
			return m, nil
		}
		customCmd := m.customInput.Value()
		if customCmd == "" {
			return m, nil
		}
		m.mode = ModeNormal
		m.customInput.Reset()
		m.startFocusCustom = false
		return m, m.startTask(task.ID, customCmd)
	}

	// Forward to text input
	var cmd tea.Cmd
	m.customInput, cmd = m.customInput.Update(msg)
	return m, cmd
}

// handleHelpMode handles keys in help mode.
func (m *Model) handleHelpMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.Help), key.Matches(msg, m.keys.Quit):
		m.mode = ModeNormal
		return m, nil
	}

	return m, nil
}

// handleDetailMode handles keys in detail mode (dialog).
func (m *Model) handleDetailMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape), key.Matches(msg, m.keys.Detail):
		m.mode = ModeNormal
		return m, nil

	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.detailViewport, cmd = m.detailViewport.Update(msg)
	return m, cmd
}

// handleDetailPanelFocused handles keys when detail panel is focused (right pane).
func (m *Model) handleDetailPanelFocused(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	// Exit focus: v, Esc, h, left arrow
	case key.Matches(msg, m.keys.Detail), key.Matches(msg, m.keys.Escape):
		m.detailFocused = false
		m.updateLayoutSizes() // Restore taskList width
		return m, nil

	case msg.String() == "h", msg.String() == "left":
		m.detailFocused = false
		m.updateLayoutSizes() // Restore taskList width
		return m, nil

	// Scroll: j/k, up/down
	case key.Matches(msg, m.keys.Up), key.Matches(msg, m.keys.Down):
		var cmd tea.Cmd
		m.detailPanelViewport, cmd = m.detailPanelViewport.Update(msg)
		return m, cmd
	}

	// Forward other keys to viewport for page up/down etc.
	var cmd tea.Cmd
	m.detailPanelViewport, cmd = m.detailPanelViewport.Update(msg)
	return m, cmd
}
