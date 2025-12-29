package usecase

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"text/template"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// DefaultDiffCommand is the default diff command when not configured.
const DefaultDiffCommand = "git diff {{.BaseBranch}}...HEAD{{if .Args}} {{.Args}}{{end}}"

// ShowDiffInput contains the parameters for showing task diff.
// Fields are ordered to minimize memory padding.
type ShowDiffInput struct {
	Args   []string // Additional diff arguments
	TaskID int      // Task ID (required)
}

// ShowDiffOutput contains the result of showing task diff.
type ShowDiffOutput struct {
	WorktreePath string // Path to the worktree where diff was executed
}

// ShowDiff is the use case for displaying task change diff.
type ShowDiff struct {
	tasks     domain.TaskRepository
	worktrees domain.WorktreeManager
	config    domain.ConfigLoader
	execCmd   func(name string, args ...string) *exec.Cmd
	stdout    io.Writer
	stderr    io.Writer
}

// NewShowDiff creates a new ShowDiff use case.
func NewShowDiff(
	tasks domain.TaskRepository,
	worktrees domain.WorktreeManager,
	config domain.ConfigLoader,
	stdout io.Writer,
	stderr io.Writer,
) *ShowDiff {
	return &ShowDiff{
		tasks:     tasks,
		worktrees: worktrees,
		config:    config,
		execCmd:   exec.Command,
		stdout:    stdout,
		stderr:    stderr,
	}
}

// SetExecCmd sets a custom exec.Cmd factory for testing.
func (uc *ShowDiff) SetExecCmd(fn func(name string, args ...string) *exec.Cmd) {
	uc.execCmd = fn
}

// DiffTemplateData contains the data for expanding diff command template.
type DiffTemplateData struct {
	Args       string // Additional arguments (space-separated)
	BaseBranch string // Base branch for the task (e.g., "main")
}

// Execute displays the diff for a task.
// Preconditions:
//   - Task exists
//   - Worktree exists
//
// Processing:
//   - Resolve task's worktree path
//   - Execute diff using diff.command from config
//   - Expand {{.Args}} with additional arguments
func (uc *ShowDiff) Execute(_ context.Context, in ShowDiffInput) (*ShowDiffOutput, error) {
	// Get the task
	task, err := uc.tasks.Get(in.TaskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}

	// Resolve worktree path
	branch := domain.BranchName(task.ID, task.Issue)
	worktreePath, err := uc.worktrees.Resolve(branch)
	if err != nil {
		return nil, fmt.Errorf("resolve worktree: %w", err)
	}

	// Load config
	cfg, err := uc.config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// Get diff command from config or use default
	diffCommand := DefaultDiffCommand
	if cfg != nil && cfg.Diff.Command != "" {
		diffCommand = cfg.Diff.Command
	}

	// Expand template with args and task info
	baseBranch := task.BaseBranch
	if baseBranch == "" {
		baseBranch = "main" // Default base branch
	}
	data := DiffTemplateData{
		Args:       strings.Join(in.Args, " "),
		BaseBranch: baseBranch,
	}

	tmpl, err := template.New("diff").Parse(diffCommand)
	if err != nil {
		return nil, fmt.Errorf("parse diff command template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("expand diff command template: %w", err)
	}

	expandedCommand := buf.String()

	// Execute the diff command
	cmd := uc.execCmd("sh", "-c", expandedCommand)
	cmd.Dir = worktreePath
	cmd.Stdout = uc.stdout
	cmd.Stderr = uc.stderr

	// Run the command - ignore exit code as diff can return non-zero when there are differences
	_ = cmd.Run()

	return &ShowDiffOutput{
		WorktreePath: worktreePath,
	}, nil
}
