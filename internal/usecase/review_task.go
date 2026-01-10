package usecase

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// ReviewTaskInput contains the parameters for reviewing a task.
// Fields are ordered to minimize memory padding.
type ReviewTaskInput struct {
	Agent   string // Agent name (optional, uses default reviewer if empty)
	Model   string // Model name override (optional, uses agent's default if empty)
	Message string // Additional instructions for the reviewer (optional)
	TaskID  int    // Task ID to review
	Verbose bool   // Show full output including intermediate steps (default: false)
}

// ReviewTaskOutput contains the result of reviewing a task.
// Fields are ordered to minimize memory padding.
type ReviewTaskOutput struct {
	Task   *domain.Task // The reviewed task
	Review string       // Review result from the agent
}

// ReviewTask is the use case for reviewing a task with an AI agent.
// Fields are ordered to minimize memory padding.
type ReviewTask struct {
	tasks        domain.TaskRepository
	worktrees    domain.WorktreeManager
	configLoader domain.ConfigLoader
	stdout       io.Writer
	stderr       io.Writer
	repoRoot     string
}

// NewReviewTask creates a new ReviewTask use case.
func NewReviewTask(
	tasks domain.TaskRepository,
	worktrees domain.WorktreeManager,
	configLoader domain.ConfigLoader,
	repoRoot string,
	stdout, stderr io.Writer,
) *ReviewTask {
	return &ReviewTask{
		tasks:        tasks,
		worktrees:    worktrees,
		configLoader: configLoader,
		repoRoot:     repoRoot,
		stdout:       stdout,
		stderr:       stderr,
	}
}

// Execute reviews a task using the specified AI agent.
func (uc *ReviewTask) Execute(ctx context.Context, in ReviewTaskInput) (*ReviewTaskOutput, error) {
	// Get task
	task, err := uc.tasks.Get(in.TaskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}

	// Load config for agent resolution
	cfg, err := uc.configLoader.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// Resolve agent from input or default
	agentName := in.Agent
	if agentName == "" {
		agentName = cfg.AgentsConfig.DefaultReviewer
	}

	// Get agent configuration
	agent, ok := cfg.Agents[agentName]
	if !ok {
		return nil, fmt.Errorf("agent %q: %w", agentName, domain.ErrAgentNotFound)
	}

	// Resolve model priority: CLI flag > agent config > builtin default
	model := in.Model
	if model == "" {
		model = agent.DefaultModel
	}

	// Get worktree path
	branch := domain.BranchName(task.ID, task.Issue)
	wtPath, err := uc.worktrees.Resolve(branch)
	if err != nil {
		return nil, fmt.Errorf("resolve worktree: %w", err)
	}

	// Build command data for template expansion
	cmdData := domain.CommandData{
		GitDir:      uc.repoRoot + "/.git",
		RepoRoot:    uc.repoRoot,
		Worktree:    wtPath,
		Title:       task.Title,
		Description: task.Description,
		Branch:      branch,
		Issue:       task.Issue,
		TaskID:      task.ID,
		Model:       model,
	}

	// Build user prompt
	userPrompt := in.Message
	if userPrompt == "" {
		userPrompt = cfg.AgentsConfig.ReviewerPrompt
	}
	if userPrompt == "" {
		userPrompt = "Please review this task."
	}

	// Render command and prompt
	defaultSystemPrompt := domain.DefaultReviewerSystemPrompt
	result, err := agent.RenderCommand(cmdData, `"$PROMPT"`, defaultSystemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("render agent command: %w", err)
	}

	// Build the script to run
	script := fmt.Sprintf(`#!/bin/bash
set -o pipefail

read -r -d '' PROMPT << 'END_OF_PROMPT'
%s
END_OF_PROMPT

%s
`, result.Prompt, result.Command)

	// Execute the command synchronously
	cmd := exec.CommandContext(ctx, "bash", "-c", script)
	cmd.Dir = wtPath

	// Capture output
	var stdout, stderr bytes.Buffer

	// In verbose mode, stream output in real-time
	// In normal mode, buffer everything and extract result after completion
	if in.Verbose && uc.stdout != nil {
		cmd.Stdout = io.MultiWriter(&stdout, uc.stdout)
	} else {
		cmd.Stdout = &stdout
	}
	if in.Verbose && uc.stderr != nil {
		cmd.Stderr = io.MultiWriter(&stderr, uc.stderr)
	} else {
		cmd.Stderr = &stderr
	}

	// Run the command
	if err := cmd.Run(); err != nil {
		// Include stderr in error message for debugging
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return nil, fmt.Errorf("run reviewer: %w: %s", err, errMsg)
		}
		return nil, fmt.Errorf("run reviewer: %w", err)
	}

	// Update task status to in_review if not already
	if task.Status != domain.StatusInReview {
		task.Status = domain.StatusInReview
		if err := uc.tasks.Save(task); err != nil {
			// Log but don't fail - the review was successful
			_, _ = fmt.Fprintf(os.Stderr, "warning: failed to update task status: %v\n", err)
		}
	}

	// Extract final review result if not verbose mode
	review := stdout.String()
	if !in.Verbose {
		review = extractReviewResult(review)
	}

	return &ReviewTaskOutput{
		Review: strings.TrimSpace(review),
		Task:   task,
	}, nil
}

// extractReviewResult extracts the final review result from the full output.
// It looks for domain.ReviewResultMarker and returns everything after it.
// If the marker is not found, it returns the full output as a fallback.
func extractReviewResult(output string) string {
	if idx := strings.Index(output, domain.ReviewResultMarker); idx != -1 {
		return output[idx+len(domain.ReviewResultMarker):]
	}
	return output // fallback: return full output
}
