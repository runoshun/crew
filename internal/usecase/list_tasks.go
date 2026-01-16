package usecase

import (
	"context"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// ListTasksInput contains the parameters for listing tasks.
type ListTasksInput struct {
	ParentID         *int     // Filter by parent task ID (nil = all tasks)
	Labels           []string // Filter by labels (AND condition)
	IncludeTerminal  bool     // Include terminal status tasks (closed)
	IncludeSessions  bool     // Include session information
	IncludeProcesses bool     // Include process information (implies IncludeSessions)
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
	tasks    domain.TaskRepository
	sessions domain.SessionManager
}

// NewListTasks creates a new ListTasks use case.
func NewListTasks(tasks domain.TaskRepository, sessions domain.SessionManager) *ListTasks {
	return &ListTasks{
		tasks:    tasks,
		sessions: sessions,
	}
}

// Execute lists tasks matching the given input criteria.
func (uc *ListTasks) Execute(_ context.Context, in ListTasksInput) (*ListTasksOutput, error) {
	filter := domain.TaskFilter{
		ParentID: in.ParentID,
		Labels:   in.Labels,
	}

	tasks, err := uc.tasks.List(filter)
	if err != nil {
		return nil, err
	}

	// Filter out terminal status tasks if not requested
	if !in.IncludeTerminal {
		tasks = filterActiveOnly(tasks)
	}

	// If sessions or processes not requested, return simple task list
	if !in.IncludeSessions && !in.IncludeProcesses {
		return &ListTasksOutput{Tasks: tasks}, nil
	}

	// Build task list with session/process information
	tasksWithInfo := make([]TaskWithSession, 0, len(tasks))
	for _, task := range tasks {
		sessionName := domain.SessionName(task.ID)

		// Check if session is running
		isRunning := false
		if uc.sessions != nil {
			running, err := uc.sessions.IsRunning(sessionName)
			if err == nil {
				isRunning = running
			}
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

// filterActiveOnly removes tasks with terminal status (closed).
func filterActiveOnly(tasks []*domain.Task) []*domain.Task {
	var result []*domain.Task
	for _, t := range tasks {
		if t.Status != domain.StatusClosed {
			result = append(result, t)
		}
	}
	return result
}
