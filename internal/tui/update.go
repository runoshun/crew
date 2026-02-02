package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
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
		// Load comment counts for all tasks and comments for selected task if detail panel is visible
		task := m.SelectedTask()
		cmds := []tea.Cmd{m.loadCommentCounts()}
		if task != nil && m.showDetailPanel() {
			cmds = append(cmds, m.loadComments(task.ID))
		}
		return m, tea.Batch(cmds...)

	case MsgConfigLoaded:
		m.config = msg.Config
		m.warnings = msg.Config.Warnings
		m.updateAgents()
		m.loadCustomKeybindings()
		return m, nil

	case MsgTaskStarted:
		return m, m.loadTasks()

	case MsgTaskStopped:
		m.mode = ModeNormal
		m.confirmAction = ConfirmNone
		if msg.SessionName == "" {
			m.err = fmt.Errorf("no running session for task #%d", msg.TaskID)
			return m, m.loadTasks()
		}
		m.err = fmt.Errorf("session stopped for task #%d", msg.TaskID)
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

	case MsgTaskStatusUpdated:
		m.mode = ModeNormal
		return m, m.loadTasks()

	case MsgTaskCopied:
		return m, m.loadTasks()

	case MsgError:
		m.err = msg.Err
		m.mode = ModeNormal
		m.confirmAction = ConfirmNone
		m.resetReviewState()
		return m, nil

	case MsgClearError:
		m.err = nil
		return m, nil

	case MsgAttachSession:
		// Use tea.Exec to attach to tmux session, returning to TUI after detach
		return m, m.attachToSession(msg.TaskID)

	case MsgReloadTasks:
		// Reload tasks after returning from external commands
		// Also reload comments if detail panel is showing
		cmds := []tea.Cmd{m.loadTasks()}
		if m.showDetailPanel() {
			if task := m.SelectedTask(); task != nil {
				cmds = append(cmds, m.loadComments(task.ID))
			}
		}
		return m, tea.Batch(cmds...)

	case MsgShowDiff:
		// Trigger diff display for a task
		return m, m.showDiff(msg.TaskID)

	case execLogMsg:
		// Execute the log pager command
		return m, tea.Exec(&domainExecCmd{cmd: msg.cmd}, func(err error) tea.Msg {
			if err != nil {
				return MsgError{Err: err}
			}
			return MsgReloadTasks{}
		})

	case execDiffMsg:
		// Execute the diff command using domainExecCmd wrapper
		// diff can return non-zero when there are differences, so ignore errors
		return m, tea.Exec(&domainExecCmd{cmd: msg.cmd, ignoreErrors: true}, func(err error) tea.Msg {
			return MsgReloadTasks{}
		})

	case execEditTaskMsg:
		// Execute the editor for task editing
		// Clean up temp file after editor closes
		return m, tea.Exec(&editorExecCmd{
			container: m.container,
			taskID:    msg.taskID,
			tmpPath:   msg.tmpPath,
			origMD:    msg.origMD,
			isComment: false,
		}, func(err error) tea.Msg {
			_ = os.Remove(msg.tmpPath)
			if err != nil {
				return MsgError{Err: err}
			}
			return MsgReloadTasks{}
		})

	case MsgTick:
		// Auto-refresh: reload tasks and schedule next tick
		if m.autoRefresh {
			return m, tea.Batch(m.loadTasks(), m.tick())
		}
		return m, m.loadTasks()

	case MsgCommentsLoaded:
		m.comments = msg.Comments
		// Update detail panel viewport if showing
		if m.showDetailPanel() {
			m.updateDetailPanelViewport()
		}
		return m, nil

	case MsgCommentCountsLoaded:
		m.commentCounts = msg.CommentCounts
		m.updateTaskList()
		return m, nil

	case MsgReviewActionCompleted:
		m.mode = ModeNormal
		m.resetReviewState()
		return m, m.loadTasks()

	case MsgPrepareEditComment:
		m.mode = ModeEditReviewComment
		m.editCommentIndex = msg.Index
		m.editCommentInput.SetValue(msg.Message)
		m.editCommentInput.Focus()
		return m, nil

	case MsgReviewResultLoaded:
		m.reviewTaskID = msg.TaskID
		m.reviewResult = msg.Review
		m.mode = ModeReviewResult
		m.updateReviewViewport() // This renders content with word wrap
		return m, nil

	case MsgReviewModeChanged:
		// Update the config in memory
		if m.config != nil {
			m.config.Complete.ReviewMode = msg.Mode
			m.config.Complete.ReviewModeSet = true
		}
		return m, nil

	case MsgManagerSessionStarted:
		// Manager session started, now attach to it
		return m, m.attachToManagerSession()

	case MsgAttachManagerSession:
		// Attach to manager session
		return m, m.attachToManagerSession()

	case MsgShowManagerSelect:
		// Show manager agent selection UI
		m.mode = ModeSelectManager
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
	case ModeSelectManager:
		return m.handleSelectManagerMode(msg)
	case ModeHelp:
		return m.handleHelpMode(msg)
	case ModeChangeStatus:
		return m.handleEditStatusMode(msg)
	case ModeExec:
		return m.handleExecMode(msg)
	case ModeReviewResult:
		return m.handleReviewResultMode(msg)
	case ModeActionMenu:
		return m.handleActionMenuMode(msg)
	case ModeReviewAction:
		return m.handleReviewActionMode(msg)
	case ModeReviewMessage:
		return m.handleReviewMessageMode(msg)
	case ModeEditReviewComment:
		return m.handleEditReviewCommentMode(msg)
	case ModeBlock:
		return m.handleBlockMode(msg)
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
		// If task changed and we're showing detail panel, load comments
		if prevTask != newTask && newTask != nil {
			m.updateSelectedTaskWorktree()
			if m.showDetailPanel() {
				// Update viewport content immediately, comments will update async
				m.updateDetailPanelViewport()
				return m, tea.Batch(cmd, m.loadComments(newTask.ID))
			}
		}
		return m, cmd

	case key.Matches(msg, m.keys.PrevPage):
		m.taskList.Paginator.PrevPage()
		m.updateSelectedTaskWorktree()
		return m, nil

	case key.Matches(msg, m.keys.NextPage):
		m.taskList.Paginator.NextPage()
		m.updateSelectedTaskWorktree()
		return m, nil

	case key.Matches(msg, m.keys.Enter):
		return m.handleDefaultAction()

	case key.Matches(msg, m.keys.Default):
		return m.openActionMenu()

	case key.Matches(msg, m.keys.Start):
		task := m.SelectedTask()
		if task == nil {
			return m, nil
		}
		m.mode = ModeStart
		return m, nil

	case key.Matches(msg, m.keys.Stop):
		task := m.SelectedTask()
		if task == nil {
			return m, nil
		}
		if !m.canStopSelectedTask(task) {
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

	case key.Matches(msg, m.keys.Exec):
		task := m.SelectedTask()
		if !m.hasWorktree(task) {
			return m, nil
		}
		m.mode = ModeExec
		m.execInput.Focus()
		return m, nil

	case key.Matches(msg, m.keys.Review):
		task := m.SelectedTask()
		if task == nil || task.Status != domain.StatusDone {
			return m, nil
		}
		if !m.hasWorktree(task) {
			return m, nil
		}
		m.enterRequestChanges(task.ID, ModeNormal, true)
		return m, nil

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
		return m, m.copyTask(task.ID, false)

	case key.Matches(msg, m.keys.CopyAll):
		task := m.SelectedTask()
		if task == nil {
			return m, nil
		}
		return m, m.copyTask(task.ID, true)

	case key.Matches(msg, m.keys.Delete):
		task := m.SelectedTask()
		if task == nil {
			return m, nil
		}
		m.mode = ModeConfirm
		m.confirmAction = ConfirmDelete
		m.confirmTaskID = task.ID
		return m, nil

	case key.Matches(msg, m.keys.Edit):
		task := m.SelectedTask()
		if task == nil {
			return m, nil
		}
		return m, m.editTaskInEditor(task.ID)

	case key.Matches(msg, m.keys.EditStatus):
		task := m.SelectedTask()
		if task == nil {
			return m, nil
		}
		m.mode = ModeChangeStatus
		m.statusCursor = 0
		return m, nil

	case key.Matches(msg, m.keys.Merge):
		task := m.SelectedTask()
		if task == nil || task.Status != domain.StatusDone {
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
		// Always use right pane focus mode (works on both wide and narrow screens)
		m.detailFocused = true
		m.updateLayoutSizes() // Update both taskList and viewport sizes
		return m, m.loadComments(task.ID)

	// Tab: focus detail panel (in workspace mode, cycles through panes)
	case msg.Type == tea.KeyTab:
		task := m.SelectedTask()
		if task == nil {
			return m, nil
		}
		m.detailFocused = true
		m.updateLayoutSizes()
		return m, m.loadComments(task.ID)

	case key.Matches(msg, m.keys.ToggleShowAll):
		m.showAll = !m.showAll
		return m, m.loadTasks()

	case key.Matches(msg, m.keys.ToggleReviewMode):
		return m, m.cycleReviewMode()

	case key.Matches(msg, m.keys.Block):
		task := m.SelectedTask()
		if task == nil || task.Status.IsTerminal() {
			return m, nil
		}
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

	case key.Matches(msg, m.keys.Manager):
		// Check if manager session is already running
		// If running, attach immediately; otherwise show agent selection
		return m, m.checkAndAttachOrSelectManager()
	}

	// Check custom keybindings
	keyStr := msg.String()
	if binding, exists := m.customKeybinds[keyStr]; exists {
		return m.handleCustomKeybinding(binding)
	}

	return m, nil
}

// handleDefaultAction performs context-aware default action.
func (m *Model) handleDefaultAction() (tea.Model, tea.Cmd) {
	return m.performDefaultAction(m.SelectedTask())
}

func (m *Model) openActionMenu() (tea.Model, tea.Cmd) {
	task := m.SelectedTask()
	items := m.actionMenuItemsForTask(task)
	if len(items) == 0 {
		return m, nil
	}
	m.actionMenuItems = items
	m.actionMenuCursor = 0
	for i, item := range items {
		if item.IsDefault {
			m.actionMenuCursor = i
			break
		}
	}
	if m.actionMenuLastID != "" && task != nil && task.ID == m.actionMenuLastTaskID {
		for i, item := range items {
			if item.ActionID == m.actionMenuLastID {
				m.actionMenuCursor = i
				break
			}
		}
	}
	m.mode = ModeActionMenu
	return m, nil
}

func (m *Model) handleActionMenuMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape):
		if len(m.actionMenuItems) > 0 && m.actionMenuCursor < len(m.actionMenuItems) {
			m.actionMenuLastID = m.actionMenuItems[m.actionMenuCursor].ActionID
			if task := m.SelectedTask(); task != nil {
				m.actionMenuLastTaskID = task.ID
			}
		}
		m.mode = ModeNormal
		m.actionMenuItems = nil
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.actionMenuCursor > 0 {
			m.actionMenuCursor--
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.actionMenuCursor < len(m.actionMenuItems)-1 {
			m.actionMenuCursor++
		}
		return m, nil

	case msg.Type == tea.KeyEnter:
		if len(m.actionMenuItems) == 0 {
			m.mode = ModeNormal
			return m, nil
		}
		selected := m.actionMenuItems[m.actionMenuCursor]
		m.actionMenuLastID = selected.ActionID
		if task := m.SelectedTask(); task != nil {
			m.actionMenuLastTaskID = task.ID
		}
		m.mode = ModeNormal
		m.actionMenuItems = nil
		if selected.Action == nil {
			return m, nil
		}
		return selected.Action()

	case msg.Type == tea.KeyRunes:
		if len(m.actionMenuItems) == 0 || len(msg.Runes) == 0 {
			return m, nil
		}
		runeKey := string(msg.Runes)
		for _, action := range m.actionMenuItems {
			if action.Key == runeKey {
				m.actionMenuLastID = action.ActionID
				if task := m.SelectedTask(); task != nil {
					m.actionMenuLastTaskID = task.ID
				}
				m.mode = ModeNormal
				m.actionMenuItems = nil
				if action.Action == nil {
					return m, nil
				}
				return action.Action()
			}
		}
		return m, nil

	case msg.Type == tea.KeySpace:
		return m, nil
	}

	return m, nil
}

func (m *Model) performDefaultAction(task *domain.Task) (tea.Model, tea.Cmd) {
	if task == nil {
		return m, nil
	}

	switch task.Status {
	case domain.StatusTodo, domain.StatusError:
		// Start the task
		m.mode = ModeStart
		return m, nil

	case domain.StatusInProgress:
		// Attach to session
		return m, func() tea.Msg {
			return MsgAttachSession{TaskID: task.ID}
		}

	case domain.StatusDone:
		// Load and show review result, then allow actions (merge, request changes, etc.)
		return m, m.loadReviewResult(task.ID)

	case domain.StatusMerged, domain.StatusClosed:
		// Show detail view for terminal states
		m.detailFocused = true
		m.updateLayoutSizes()
		return m, m.loadComments(task.ID)
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
	tasks := m.sortedTasks()
	if query == "" {
		m.setTaskItems(tasks)
		return
	}

	filtered := make([]*domain.Task, 0, len(tasks))
	for _, t := range tasks {
		if strings.Contains(strings.ToLower(t.Title), query) ||
			strings.Contains(strings.ToLower(string(t.Status)), query) ||
			strings.Contains(strings.ToLower(t.Agent), query) {
			filtered = append(filtered, t)
		}
	}
	m.setTaskItems(filtered)
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

	case msg.Type == tea.KeySpace:
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

	case msg.Type == tea.KeySpace:
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

// handleSelectManagerMode handles keys in manager selection mode.
func (m *Model) handleSelectManagerMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape):
		m.mode = ModeNormal
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.managerAgentCursor > 0 {
			m.managerAgentCursor--
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.managerAgentCursor < len(m.managerAgents)-1 {
			m.managerAgentCursor++
		}
		return m, nil

	case msg.Type == tea.KeyEnter:
		if len(m.managerAgents) == 0 {
			m.mode = ModeNormal
			return m, nil
		}
		agent := m.managerAgents[m.managerAgentCursor]
		m.mode = ModeNormal

		// Start or attach to manager session.
		// The selected manager agent is used to start the session.
		return m, m.startOrAttachManagerSessionForTask(agent)
	}

	return m, nil
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

	// Tab: return focus to workspace (in workspace mode)
	case msg.Type == tea.KeyTab:
		m.detailFocused = false
		m.updateLayoutSizes()
		return m, func() tea.Msg { return MsgFocusWorkspace{} }

	// Arrow keys: 1 line scroll
	case msg.String() == "up":
		m.detailPanelViewport.ScrollUp(1)
		return m, nil

	case msg.String() == "down":
		m.detailPanelViewport.ScrollDown(1)
		return m, nil

	// j/k: half page scroll
	case msg.String() == "j":
		m.detailPanelViewport.HalfPageDown()
		return m, nil

	case msg.String() == "k":
		m.detailPanelViewport.HalfPageUp()
		return m, nil

	// g/G: jump to top/bottom
	case msg.String() == "g":
		m.detailPanelViewport.GotoTop()
		return m, nil

	case msg.String() == "G":
		m.detailPanelViewport.GotoBottom()
		return m, nil
	}

	// Forward other keys to viewport for page up/down, mouse scroll, etc.
	var cmd tea.Cmd
	m.detailPanelViewport, cmd = m.detailPanelViewport.Update(msg)
	return m, cmd
}

// handleEditStatusMode handles keys in status change mode.
func (m *Model) handleEditStatusMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	task := m.SelectedTask()
	if task == nil {
		m.mode = ModeNormal
		return m, nil
	}

	transitions := m.getStatusTransitions(task.Status)

	switch {
	case key.Matches(msg, m.keys.Escape):
		m.mode = ModeNormal
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.statusCursor > 0 {
			m.statusCursor--
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.statusCursor < len(transitions)-1 {
			m.statusCursor++
		}
		return m, nil

	case msg.Type == tea.KeyEnter:
		if len(transitions) == 0 {
			m.mode = ModeNormal
			return m, nil
		}
		newStatus := transitions[m.statusCursor]
		return m, m.updateStatus(task.ID, newStatus)
	}

	return m, nil
}

func (m *Model) getStatusTransitions(current domain.Status) []domain.Status {
	all := domain.AllStatuses()
	var normal []domain.Status
	var forced []domain.Status
	for _, s := range all {
		if s == current {
			continue
		}
		if current.CanTransitionTo(s) {
			normal = append(normal, s)
		} else {
			forced = append(forced, s)
		}
	}
	return append(normal, forced...)
}

// handleExecMode handles keys in exec mode.
func (m *Model) handleExecMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape):
		m.mode = ModeNormal
		m.execInput.Reset()
		return m, nil

	case msg.Type == tea.KeyEnter:
		task := m.SelectedTask()
		if task == nil {
			m.mode = ModeNormal
			return m, nil
		}
		cmdStr := m.execInput.Value()
		if cmdStr == "" {
			return m, nil
		}
		m.mode = ModeNormal
		m.execInput.Reset()
		return m, m.executeCommandInWorktree(cmdStr)
	}

	var cmd tea.Cmd
	m.execInput, cmd = m.execInput.Update(msg)
	return m, cmd
}

// handleReviewResultMode handles keys when viewing review results.
func (m *Model) handleReviewResultMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape):
		m.mode = ModeNormal
		m.resetReviewState()
		return m, nil

	case msg.Type == tea.KeyEnter:
		// Move to action selection
		m.mode = ModeReviewAction
		m.reviewActionCursor = 0
		return m, nil

	case key.Matches(msg, m.keys.Up), key.Matches(msg, m.keys.Down):
		// Scroll the review viewport
		var cmd tea.Cmd
		m.reviewViewport, cmd = m.reviewViewport.Update(msg)
		return m, cmd
	}

	// Forward other keys to viewport for page up/down
	var cmd tea.Cmd
	m.reviewViewport, cmd = m.reviewViewport.Update(msg)
	return m, cmd
}

// handleReviewActionMode handles keys when selecting a review action.
func (m *Model) handleReviewActionMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	const numActions = 5 // Request Changes, Comment Only, Merge, Close, Edit Comment

	switch {
	case key.Matches(msg, m.keys.Escape):
		// Go back to review result
		m.mode = ModeReviewResult
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.reviewActionCursor > 0 {
			m.reviewActionCursor--
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.reviewActionCursor < numActions-1 {
			m.reviewActionCursor++
		}
		return m, nil

	case msg.Type == tea.KeyEnter:
		// Execute selected action
		switch m.reviewActionCursor {
		case 0: // Request Changes - enter message input mode
			m.enterRequestChanges(m.reviewTaskID, ModeReviewAction, false)
			return m, nil
		case 1: // NotifyWorker without restart (just send comment)
			return m, m.notifyWorker(m.reviewTaskID, m.reviewResult, false)
		case 2: // Merge - use confirm dialog
			m.mode = ModeConfirm
			m.confirmAction = ConfirmMerge
			m.confirmTaskID = m.reviewTaskID
			return m, nil
		case 3: // Close - use confirm dialog
			m.mode = ModeConfirm
			m.confirmAction = ConfirmClose
			m.confirmTaskID = m.reviewTaskID
			return m, nil
		case 4: // Edit Comment - enter edit mode
			return m, m.prepareEditReviewComment()
		}
	}

	return m, nil
}

func (m *Model) enterRequestChanges(taskID int, returnMode Mode, resetReviewContext bool) {
	m.reviewTaskID = taskID
	if resetReviewContext {
		m.reviewResult = ""
		m.reviewActionCursor = 0
	}
	m.reviewMessageReturnMode = returnMode
	m.mode = ModeReviewMessage
	m.reviewMessageInput.Reset()
	m.reviewMessageInput.Focus()
}

func (m *Model) resetReviewState() {
	m.reviewTaskID = 0
	m.reviewResult = ""
	m.reviewActionCursor = 0
	m.reviewMessageReturnMode = ModeNormal
}

// defaultReviewMessage is the default message when the user leaves the input empty.
const defaultReviewMessage = "Please address the review comments above."

// handleReviewMessageMode handles keys when entering a message for Request Changes.
func (m *Model) handleReviewMessageMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape):
		m.reviewMessageInput.Blur()
		returnMode := m.reviewMessageReturnMode
		m.reviewMessageReturnMode = ModeNormal
		//nolint:exhaustive
		switch returnMode {
		case ModeReviewAction:
			m.mode = ModeReviewAction
			return m, nil
		case ModeActionMenu:
			m.resetReviewState()
			m.mode = ModeNormal
			return m.openActionMenu()
		default:
			m.resetReviewState()
			m.mode = ModeNormal
			return m, nil
		}

	case msg.Type == tea.KeyEnter:
		// Submit the message (or use default if empty)
		message := m.reviewMessageInput.Value()
		if message == "" {
			message = defaultReviewMessage
		}
		m.reviewMessageInput.Blur()
		return m, m.notifyWorker(m.reviewTaskID, message, true)
	}

	// Update text input
	var cmd tea.Cmd
	m.reviewMessageInput, cmd = m.reviewMessageInput.Update(msg)
	return m, cmd
}

// handleEditReviewCommentMode handles keys when editing a review comment.
func (m *Model) handleEditReviewCommentMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape):
		// Cancel and go back to review action selection
		m.mode = ModeReviewAction
		m.editCommentInput.Blur()
		return m, nil

	case msg.Type == tea.KeyEnter:
		// Save the edited comment
		message := m.editCommentInput.Value()
		if message == "" {
			// Don't allow empty comments
			return m, nil
		}
		m.editCommentInput.Blur()
		return m, m.editComment(m.reviewTaskID, m.editCommentIndex, message)
	}

	// Update text input
	var cmd tea.Cmd
	m.editCommentInput, cmd = m.editCommentInput.Update(msg)
	return m, cmd
}

// handleBlockMode handles keys in block dialog mode.
func (m *Model) handleBlockMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape):
		m.mode = ModeNormal
		m.blockInput.Blur()
		m.blockInput.Reset()
		m.blockFocusUnblock = false
		return m, nil

	case msg.Type == tea.KeyTab:
		// Toggle between reason input and unblock button
		m.blockFocusUnblock = !m.blockFocusUnblock
		if m.blockFocusUnblock {
			m.blockInput.Blur()
		} else {
			m.blockInput.Focus()
		}
		return m, nil

	case msg.Type == tea.KeyEnter:
		task := m.SelectedTask()
		if task == nil {
			m.mode = ModeNormal
			return m, nil
		}

		if m.blockFocusUnblock {
			// Unblock the task
			m.mode = ModeNormal
			m.blockInput.Blur()
			m.blockInput.Reset()
			m.blockFocusUnblock = false
			return m, m.unblockTask(task.ID)
		}

		// Block the task with the entered reason
		reason := m.blockInput.Value()
		if reason == "" {
			// Need a reason to block
			return m, nil
		}
		m.mode = ModeNormal
		m.blockInput.Blur()
		m.blockInput.Reset()
		m.blockFocusUnblock = false
		return m, m.blockTask(task.ID, reason)
	}

	// Forward to text input if reason input is focused
	if !m.blockFocusUnblock {
		var cmd tea.Cmd
		m.blockInput, cmd = m.blockInput.Update(msg)
		return m, cmd
	}

	return m, nil
}
