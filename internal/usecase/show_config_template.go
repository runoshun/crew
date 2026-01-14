// Package usecase contains the application use cases.
package usecase

import (
	"context"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// ShowConfigTemplateInput contains the input for the ShowConfigTemplate use case.
type ShowConfigTemplateInput struct {
	Config *domain.Config // Config with builtin agents registered (for template generation)
}

// ShowConfigTemplateOutput contains the output of the ShowConfigTemplate use case.
type ShowConfigTemplateOutput struct {
	Template string // Configuration template content
}

// ShowConfigTemplate generates a configuration template and returns it as a string.
type ShowConfigTemplate struct {
	// No dependencies needed - template is generated from input Config
}

// NewShowConfigTemplate creates a new ShowConfigTemplate use case.
func NewShowConfigTemplate() *ShowConfigTemplate {
	return &ShowConfigTemplate{}
}

// Execute generates and returns a configuration template.
func (uc *ShowConfigTemplate) Execute(_ context.Context, in ShowConfigTemplateInput) (*ShowConfigTemplateOutput, error) {
	if in.Config == nil {
		return nil, domain.ErrConfigNil
	}

	template := domain.RenderConfigTemplate(in.Config)

	return &ShowConfigTemplateOutput{Template: template}, nil
}
