package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase/shared"
)

// ListTasksInput contains the parameters for listing tasks.
type ListTasksInput struct {
	ParentID         *int     // Filter by parent task ID (nil = all tasks)
	Labels           []string // Filter by labels (AND condition)
	IncludeTerminal  bool     // Include terminal status tasks (merged/closed)
	IncludeSessions  bool     // Include session information
	IncludeProcesses bool     // Include process information (implies IncludeSessions)
	AllNamespaces    bool     // List tasks across all namespaces when supported
}

type taskNamespaceLister interface {
	ListAll(filter domain.TaskFilter) ([]*domain.Task, error)
}

// TaskWithSession contains a task with session and process information.
// Fields are ordered to minimize memory padding.
type TaskWithSession struct {
	Task        *domain.Task
	SessionName string
	Processes   []domain.ProcessInfo
	IsRunning   bool
}

// ListTasksOutput contains the result of listing tasks.
type ListTasksOutput struct {
	Tasks         []*domain.Task    // List of tasks matching the filter (if sessions not requested)
	TasksWithInfo []TaskWithSession // List of tasks with session info (if sessions requested)
}

// ListTasks is the use case for listing tasks.
type ListTasks struct {
	tasks     domain.TaskRepository
	sessions  domain.SessionManager
	acpStates domain.ACPStateStore
}

// NewListTasks creates a new ListTasks use case.
func NewListTasks(tasks domain.TaskRepository, sessions domain.SessionManager, acpStates domain.ACPStateStore) *ListTasks {
	return &ListTasks{
		tasks:     tasks,
		sessions:  sessions,
		acpStates: acpStates,
	}
}

// Execute lists tasks matching the given input criteria.
func (uc *ListTasks) Execute(ctx context.Context, in ListTasksInput) (*ListTasksOutput, error) {
	filter := domain.TaskFilter{
		ParentID: in.ParentID,
		Labels:   in.Labels,
	}

	tasks, err := uc.listTasks(filter, in.AllNamespaces)
	if err != nil {
		return nil, err
	}

	// Filter out terminal status tasks if not requested
	if !in.IncludeTerminal {
		tasks = filterActiveOnly(tasks)
	}

	if err := uc.attachExecutionSubstates(ctx, tasks); err != nil {
		return nil, err
	}

	// If sessions or processes not requested, return simple task list
	if !in.IncludeSessions && !in.IncludeProcesses {
		return &ListTasksOutput{Tasks: tasks}, nil
	}

	// Build task list with session/process information
	tasksWithInfo := make([]TaskWithSession, 0, len(tasks))
	for _, task := range tasks {
		sessionName := domain.SessionName(task.ID)

		// Check if session is running (ignore errors for list display)
		isRunning := false
		if uc.sessions != nil {
			isRunning, _ = shared.CheckSessionRunning(uc.sessions, task.ID)
		}

		info := TaskWithSession{
			Task:        task,
			SessionName: sessionName,
			IsRunning:   isRunning,
		}

		// Get process information if requested
		if in.IncludeProcesses && isRunning {
			processes, err := uc.sessions.GetPaneProcesses(sessionName)
			if err == nil {
				info.Processes = processes
			}
		}

		tasksWithInfo = append(tasksWithInfo, info)
	}

	return &ListTasksOutput{TasksWithInfo: tasksWithInfo}, nil
}

func (uc *ListTasks) attachExecutionSubstates(ctx context.Context, tasks []*domain.Task) error {
	if uc.acpStates == nil {
		return nil
	}
	for _, task := range tasks {
		if task == nil {
			continue
		}
		namespace := task.Namespace
		if namespace == "" {
			namespace = domain.DefaultNamespace
		}
		state, err := uc.acpStates.Load(ctx, namespace, task.ID)
		if err != nil {
			if errors.Is(err, domain.ErrACPStateNotFound) {
				continue
			}
			return fmt.Errorf("load acp state for %s#%d: %w", namespace, task.ID, err)
		}
		task.ExecutionSubstate = state.ExecutionSubstate
	}
	return nil
}

func (uc *ListTasks) listTasks(filter domain.TaskFilter, allNamespaces bool) ([]*domain.Task, error) {
	if allNamespaces {
		if lister, ok := uc.tasks.(taskNamespaceLister); ok {
			return lister.ListAll(filter)
		}
	}
	return uc.tasks.List(filter)
}

// filterActiveOnly removes tasks with terminal status (merged/closed).
func filterActiveOnly(tasks []*domain.Task) []*domain.Task {
	var result []*domain.Task
	for _, t := range tasks {
		if !t.Status.IsTerminal() {
			result = append(result, t)
		}
	}
	return result
}
