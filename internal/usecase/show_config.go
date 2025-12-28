// Package usecase contains the application use cases.
package usecase

import (
	"context"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// ShowConfigInput contains the input for the ShowConfig use case.
type ShowConfigInput struct{}

// ShowConfigOutput contains the output of the ShowConfig use case.
type ShowConfigOutput struct {
	GlobalConfig domain.ConfigInfo // Global config file info
	RepoConfig   domain.ConfigInfo // Repository config file info
}

// ShowConfig displays configuration file information.
type ShowConfig struct {
	configManager domain.ConfigManager
}

// NewShowConfig creates a new ShowConfig use case.
func NewShowConfig(configManager domain.ConfigManager) *ShowConfig {
	return &ShowConfig{
		configManager: configManager,
	}
}

// Execute retrieves configuration file information.
func (uc *ShowConfig) Execute(_ context.Context, _ ShowConfigInput) (*ShowConfigOutput, error) {
	return &ShowConfigOutput{
		GlobalConfig: uc.configManager.GetGlobalConfigInfo(),
		RepoConfig:   uc.configManager.GetRepoConfigInfo(),
	}, nil
}
