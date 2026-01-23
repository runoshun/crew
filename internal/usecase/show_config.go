// Package usecase contains the application use cases.
package usecase

import (
	"context"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// ShowConfigInput contains the input for the ShowConfig use case.
type ShowConfigInput struct {
	IgnoreGlobal   bool // Skip loading global config
	IgnoreRepo     bool // Skip loading repo config (.git/crew/config.toml)
	IgnoreRootRepo bool // Skip loading root repo config (.crew.toml)
	IgnoreOverride bool // Skip loading override config (config.override.toml)
	IgnoreRuntime  bool // Skip loading runtime config (.git/crew/config.runtime.toml)
}

// ShowConfigOutput contains the output of the ShowConfig use case.
type ShowConfigOutput struct {
	EffectiveConfig *domain.Config    // Merged effective configuration
	GlobalConfig    domain.ConfigInfo // Global config file info
	OverrideConfig  domain.ConfigInfo // Override config file info (config.override.toml)
	RepoConfig      domain.ConfigInfo // Repository config file info (.git/crew/config.toml)
	RootRepoConfig  domain.ConfigInfo // Root repository config file info (.crew.toml)
	RuntimeConfig   domain.ConfigInfo // Runtime config file info (.git/crew/config.runtime.toml)
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
	if !input.IgnoreOverride {
		output.OverrideConfig = uc.configManager.GetOverrideConfigInfo()
	}
	if !input.IgnoreRepo {
		output.RepoConfig = uc.configManager.GetRepoConfigInfo()
	}
	if !input.IgnoreRootRepo {
		output.RootRepoConfig = uc.configManager.GetRootRepoConfigInfo()
	}
	if !input.IgnoreRuntime {
		output.RuntimeConfig = uc.configManager.GetRuntimeConfigInfo()
	}

	// Load effective config with options
	effectiveConfig, err := uc.configLoader.LoadWithOptions(domain.LoadConfigOptions{
		IgnoreGlobal:   input.IgnoreGlobal,
		IgnoreRepo:     input.IgnoreRepo,
		IgnoreRootRepo: input.IgnoreRootRepo,
		IgnoreOverride: input.IgnoreOverride,
		IgnoreRuntime:  input.IgnoreRuntime,
	})
	if err != nil {
		return nil, err
	}
	output.EffectiveConfig = effectiveConfig

	return output, nil
}
