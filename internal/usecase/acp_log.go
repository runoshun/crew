package usecase

import (
	"context"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase/shared"
)

// ACPLogInput contains parameters for reading ACP event logs.
type ACPLogInput struct {
	TaskID int // Task ID
}

// ACPLogOutput contains the result of reading ACP event logs.
type ACPLogOutput struct {
	Events []domain.ACPEvent
}

// ACPLog is the use case for reading ACP event logs.
type ACPLog struct {
	tasks        domain.TaskRepository
	configLoader domain.ConfigLoader
	git          domain.Git
	eventReader  func(namespace string, taskID int) domain.ACPEventReader
}

// NewACPLog creates a new ACPLog use case.
func NewACPLog(
	tasks domain.TaskRepository,
	configLoader domain.ConfigLoader,
	git domain.Git,
	eventReader func(namespace string, taskID int) domain.ACPEventReader,
) *ACPLog {
	return &ACPLog{
		tasks:        tasks,
		configLoader: configLoader,
		git:          git,
		eventReader:  eventReader,
	}
}

// Execute reads ACP event logs for a task.
func (uc *ACPLog) Execute(ctx context.Context, in ACPLogInput) (*ACPLogOutput, error) {
	_, err := shared.GetTask(uc.tasks, in.TaskID)
	if err != nil {
		return nil, err
	}

	cfg, err := uc.configLoader.Load()
	if err != nil {
		return nil, err
	}

	namespace := shared.ResolveACPNamespace(cfg, uc.git)
	reader := uc.eventReader(namespace, in.TaskID)

	events, err := reader.ReadAll(ctx)
	if err != nil {
		return nil, err
	}

	return &ACPLogOutput{Events: events}, nil
}
