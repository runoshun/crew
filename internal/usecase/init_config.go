// Package usecase contains the application use cases.
package usecase

import (
	"context"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// InitConfigInput contains the input for the InitConfig use case.
type InitConfigInput struct {
	Config *domain.Config // Config with builtin agents registered (for template generation)
	Global bool           // If true, initialize global config; otherwise repository config
}

// InitConfigOutput contains the output of the InitConfig use case.
type InitConfigOutput struct {
	Path string // Path to the created config file
}

// InitConfig generates a configuration file template.
type InitConfig struct {
	configManager domain.ConfigManager
}

// NewInitConfig creates a new InitConfig use case.
func NewInitConfig(configManager domain.ConfigManager) *InitConfig {
	return &InitConfig{
		configManager: configManager,
	}
}

// Execute creates a configuration file with default template.
func (uc *InitConfig) Execute(_ context.Context, in InitConfigInput) (*InitConfigOutput, error) {
	var err error
	var path string

	if in.Global {
		info := uc.configManager.GetGlobalConfigInfo()
		path = info.Path
		err = uc.configManager.InitGlobalConfig(in.Config)
	} else {
		info := uc.configManager.GetRepoConfigInfo()
		path = info.Path
		err = uc.configManager.InitRepoConfig(in.Config)
	}

	if err != nil {
		return nil, err
	}

	return &InitConfigOutput{Path: path}, nil
}
