package shared

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// ReviewCommandDeps contains dependencies for preparing review commands.
// Fields are ordered to minimize memory padding.
type ReviewCommandDeps struct {
	ConfigLoader domain.ConfigLoader
	Worktrees    domain.WorktreeManager
	RepoRoot     string
}

// ReviewCommandInput contains parameters for building a review command.
// Fields are ordered to minimize memory padding.
type ReviewCommandInput struct {
	Task    *domain.Task
	Agent   string // Agent name (optional, uses default reviewer if empty)
	Model   string // Model name override (optional, uses agent default if empty)
	Message string // Additional instructions for the reviewer (optional)
}

// ReviewCommandOutput contains prepared review command data.
// Fields are ordered to minimize memory padding.
type ReviewCommandOutput struct {
	AgentName    string
	WorktreePath string
	Result       domain.RenderCommandResult
}

// PrepareReviewCommand resolves agent configuration and builds the review command.
func PrepareReviewCommand(deps ReviewCommandDeps, in ReviewCommandInput) (*ReviewCommandOutput, error) {
	cfg, err := deps.ConfigLoader.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	agentName := in.Agent
	if agentName == "" {
		agentName = cfg.AgentsConfig.DefaultReviewer
	}

	agent, ok := cfg.EnabledAgents()[agentName]
	if !ok {
		if _, exists := cfg.Agents[agentName]; exists {
			return nil, fmt.Errorf("agent %q is disabled: %w", agentName, domain.ErrAgentDisabled)
		}
		return nil, fmt.Errorf("agent %q: %w", agentName, domain.ErrAgentNotFound)
	}

	model := in.Model
	if model == "" {
		model = agent.DefaultModel
	}

	branch := domain.BranchName(in.Task.ID, in.Task.Issue)
	wtPath, err := deps.Worktrees.Resolve(branch)
	if err != nil {
		return nil, fmt.Errorf("resolve worktree: %w", err)
	}

	cmdData := domain.CommandData{
		GitDir:      deps.RepoRoot + "/.git",
		RepoRoot:    deps.RepoRoot,
		Worktree:    wtPath,
		Title:       in.Task.Title,
		Description: in.Task.Description,
		Branch:      branch,
		Issue:       in.Task.Issue,
		TaskID:      in.Task.ID,
		Model:       model,
	}

	userPrompt := in.Message
	if userPrompt == "" {
		userPrompt = cfg.AgentsConfig.ReviewerPrompt
	}
	if userPrompt == "" {
		userPrompt = "Please review this task."
	}

	defaultSystemPrompt := domain.DefaultReviewerSystemPrompt
	result, err := agent.RenderCommand(cmdData, `"$PROMPT"`, defaultSystemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("render agent command: %w", err)
	}

	return &ReviewCommandOutput{
		AgentName:    agentName,
		WorktreePath: wtPath,
		Result:       result,
	}, nil
}

// ReviewDeps contains dependencies for executing reviews.
// Fields are ordered to minimize memory padding.
type ReviewDeps struct {
	Tasks    domain.TaskRepository
	Executor domain.CommandExecutor
	Clock    domain.Clock
	Stderr   io.Writer
}

// ReviewInput contains parameters for executing a review.
// Fields are ordered to minimize memory padding.
type ReviewInput struct {
	Task            *domain.Task
	WorktreePath    string
	Result          domain.RenderCommandResult
	SkipStatusCheck bool
}

// ReviewOutput contains the review result.
// Fields are ordered to minimize memory padding.
type ReviewOutput struct {
	Review string
	IsLGTM bool
}

// ExecuteReview runs the review command and records its result.
func ExecuteReview(ctx context.Context, deps ReviewDeps, in ReviewInput) (*ReviewOutput, error) {
	if !in.SkipStatusCheck {
		// Review can only be executed on in_progress or done tasks
		if in.Task.Status != domain.StatusInProgress && in.Task.Status != domain.StatusDone {
			return nil, fmt.Errorf("cannot review task in %s status (must be in_progress or done): %w", in.Task.Status, domain.ErrInvalidTransition)
		}
	}

	script := fmt.Sprintf(`#!/bin/bash
set -o pipefail

read -r -d '' PROMPT << 'END_OF_PROMPT'
%s
END_OF_PROMPT

%s
`, in.Result.Prompt, in.Result.Command)

	execCmd := domain.NewBashCommand(script, in.WorktreePath)

	var stdoutBuf, stderrBuf bytes.Buffer
	if err := deps.Executor.ExecuteWithContext(ctx, execCmd, &stdoutBuf, &stderrBuf); err != nil {
		errMsg := strings.TrimSpace(stderrBuf.String())
		if errMsg != "" {
			return nil, fmt.Errorf("run reviewer: %w: %s", err, errMsg)
		}
		return nil, fmt.Errorf("run reviewer: %w", err)
	}

	review := strings.TrimSpace(extractReviewResult(stdoutBuf.String()))

	if review != "" {
		comment := domain.Comment{
			Text:   review,
			Time:   deps.Clock.Now(),
			Author: "reviewer",
		}
		if err := deps.Tasks.AddComment(in.Task.ID, comment); err != nil {
			if deps.Stderr != nil {
				_, _ = fmt.Fprintf(deps.Stderr, "warning: failed to add review comment: %v\n", err)
			}
		}
	}

	return &ReviewOutput{
		Review: review,
		IsLGTM: strings.HasPrefix(review, domain.ReviewLGTMPrefix),
	}, nil
}

// extractReviewResult extracts the final review result from the full output.
// It looks for domain.ReviewResultMarker and returns everything after it.
// If the marker is not found, it returns the full output as a fallback.
func extractReviewResult(output string) string {
	if idx := strings.Index(output, domain.ReviewResultMarker); idx != -1 {
		return output[idx+len(domain.ReviewResultMarker):]
	}
	return output
}
