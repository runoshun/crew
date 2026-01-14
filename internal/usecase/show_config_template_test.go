package usecase

import (
	"context"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShowConfigTemplate_Execute(t *testing.T) {
	tests := []struct {
		name           string
		input          ShowConfigTemplateInput
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "template with default config structure",
			input: ShowConfigTemplateInput{
				Config: domain.NewDefaultConfig(),
			},
			wantContains: []string{
				"[agents]",
				"worker_default",
				"manager_default",
			},
		},
		{
			name: "template with empty agents map",
			input: ShowConfigTemplateInput{
				Config: &domain.Config{
					Agents: make(map[string]domain.Agent),
				},
			},
			wantContains: []string{
				"[agents]",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewShowConfigTemplate()
			out, err := uc.Execute(context.Background(), tt.input)

			require.NoError(t, err)
			require.NotNil(t, out)

			for _, want := range tt.wantContains {
				assert.Contains(t, out.Template, want, "template should contain %q", want)
			}

			for _, notWant := range tt.wantNotContain {
				assert.NotContains(t, out.Template, notWant, "template should not contain %q", notWant)
			}
		})
	}
}

func TestShowConfigTemplate_Execute_NilConfig(t *testing.T) {
	uc := NewShowConfigTemplate()
	out, err := uc.Execute(context.Background(), ShowConfigTemplateInput{
		Config: nil,
	})

	assert.Nil(t, out)
	assert.ErrorIs(t, err, domain.ErrConfigNil)
}
