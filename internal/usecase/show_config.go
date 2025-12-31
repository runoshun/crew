// Package usecase contains the application use cases.
package usecase

import (
	"context"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// ShowConfigInput contains the input for the ShowConfig use case.
type ShowConfigInput struct {
	IgnoreGlobal bool // Skip loading global config
	IgnoreRepo   bool // Skip loading repo config
}

// ShowConfigOutput contains the output of the ShowConfig use case.
type ShowConfigOutput struct {
	EffectiveConfig *domain.Config    // Merged effective configuration
	GlobalConfig    domain.ConfigInfo // Global config file info
	RepoConfig      domain.ConfigInfo // Repository config file info
}

// ShowConfig displays configuration file information.
type ShowConfig struct {
	configManager domain.ConfigManager
	configLoader  domain.ConfigLoader
}

// NewShowConfig creates a new ShowConfig use case.
func NewShowConfig(configManager domain.ConfigManager, configLoader domain.ConfigLoader) *ShowConfig {
	return &ShowConfig{
		configManager: configManager,
		configLoader:  configLoader,
	}
}

// Execute retrieves configuration file information.
func (uc *ShowConfig) Execute(_ context.Context, input ShowConfigInput) (*ShowConfigOutput, error) {
	output := &ShowConfigOutput{}

	// Get file info (unless ignored)
	if !input.IgnoreGlobal {
		output.GlobalConfig = uc.configManager.GetGlobalConfigInfo()
	}
	if !input.IgnoreRepo {
		output.RepoConfig = uc.configManager.GetRepoConfigInfo()
	}

	// Load effective config with options
	effectiveConfig, err := uc.configLoader.LoadWithOptions(domain.LoadConfigOptions{
		IgnoreGlobal: input.IgnoreGlobal,
		IgnoreRepo:   input.IgnoreRepo,
	})
	if err != nil {
		return nil, err
	}
	output.EffectiveConfig = effectiveConfig

	return output, nil
}
