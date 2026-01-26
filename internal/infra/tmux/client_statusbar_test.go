package tmux

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_ConfigureStatusBar_Colors(t *testing.T) {
	socketPath, crewDir, cleanup := setupTestEnv(t)
	defer cleanup()

	client := NewClient(socketPath, crewDir)

	tests := []struct {
		name           string
		sessionType    domain.SessionType
		expectedMainBg string
	}{
		{
			name:           "Worker Session",
			sessionType:    domain.SessionTypeWorker,
			expectedMainBg: "bg=#1e66f5", // Blue
		},
		{
			name:           "Reviewer Session",
			sessionType:    domain.SessionTypeReviewer,
			expectedMainBg: "bg=#8839ef", // Purple
		},
		{
			name:           "Manager Session",
			sessionType:    domain.SessionTypeManager,
			expectedMainBg: "bg=#40a02b", // Green
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionName := strings.ReplaceAll(strings.ToLower(tt.name), " ", "-")

			err := client.Start(context.Background(), domain.StartSessionOptions{
				Name:      sessionName,
				Dir:       crewDir,
				Command:   "sleep 60",
				TaskID:    1,
				TaskTitle: "Test Task",
				TaskAgent: "test-agent",
				Type:      tt.sessionType,
			})
			require.NoError(t, err)

			// Check status-style
			// tmux -S socket show-options -t session -v status-style
			cmd := exec.Command("tmux", "-S", socketPath, "show-options", "-t", sessionName, "-v", "status-style")
			out, err := cmd.Output()
			require.NoError(t, err)

			style := strings.TrimSpace(string(out))
			assert.Contains(t, style, tt.expectedMainBg, "status-style should contain expected background color")
		})
	}
}
