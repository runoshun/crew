package usecase

import (
	"context"
	"fmt"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase/shared"
)

// ACPControlInput contains parameters for sending ACP control commands.
// Fields are ordered to minimize memory padding.
type ACPControlInput struct {
	CommandType domain.ACPCommandType
	Text        string
	OptionID    string
	TaskID      int
}

// ACPControlOutput contains the result of ACP control operation.
type ACPControlOutput struct{}

// ACPControl sends ACP control commands via IPC.
// Fields are ordered to minimize memory padding.
type ACPControl struct {
	tasks        domain.TaskRepository
	configLoader domain.ConfigLoader
	git          domain.Git
	ipcFactory   domain.ACPIPCFactory
	acpStates    domain.ACPStateStore
}

// NewACPControl creates a new ACPControl use case.
func NewACPControl(
	tasks domain.TaskRepository,
	configLoader domain.ConfigLoader,
	git domain.Git,
	ipcFactory domain.ACPIPCFactory,
	acpStates domain.ACPStateStore,
) *ACPControl {
	return &ACPControl{
		tasks:        tasks,
		configLoader: configLoader,
		git:          git,
		ipcFactory:   ipcFactory,
		acpStates:    acpStates,
	}
}

// Execute sends a command to the ACP runner via IPC.
func (uc *ACPControl) Execute(ctx context.Context, in ACPControlInput) (*ACPControlOutput, error) {
	if in.TaskID <= 0 {
		return nil, fmt.Errorf("invalid task ID: %d", in.TaskID)
	}

	_, err := shared.GetTask(uc.tasks, in.TaskID)
	if err != nil {
		return nil, err
	}

	cfg, err := uc.configLoader.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	namespace := shared.ResolveACPNamespace(cfg, uc.git)
	ipc := uc.ipcFactory.ForTask(namespace, in.TaskID)
	cmd := domain.ACPCommand{
		Type:     in.CommandType,
		Text:     in.Text,
		OptionID: in.OptionID,
	}
	if err := ipc.Send(ctx, cmd); err != nil {
		return nil, err
	}
	if uc.acpStates != nil {
		switch in.CommandType {
		case domain.ACPCommandPermission, domain.ACPCommandPrompt:
			if err := uc.acpStates.Save(ctx, namespace, in.TaskID, domain.ACPExecutionState{ExecutionSubstate: domain.ACPExecutionRunning}); err != nil {
				return nil, fmt.Errorf("save acp state: %w", err)
			}
		case domain.ACPCommandCancel, domain.ACPCommandStop:
		}
	}

	return &ACPControlOutput{}, nil
}
