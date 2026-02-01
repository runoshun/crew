package usecase

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase/shared"
)

const (
	reviewPeekLines       = 2000
	reviewLogTailLines    = 20
	reviewLogTailMaxBytes = 64 * 1024
)

// CompleteTaskInput contains the parameters for completing a task.
type CompleteTaskInput struct {
	Comment     string // Optional completion comment
	ReviewAgent string // Reviewer agent override (optional)
	TaskID      int    // Task ID to complete
	Verbose     bool   // Stream reviewer output while waiting
}

// CompleteTaskOutput contains the result of completing a task.
// Fields are ordered to minimize memory padding.
type CompleteTaskOutput struct {
	Task              *domain.Task      // The completed task
	ConflictMessage   string            // Conflict message to display (only set when ErrMergeConflict is returned)
	AutoFixReview     string            // Deprecated: auto_fix output (ignored)
	ReviewMode        domain.ReviewMode // Deprecated: review_mode (ignored)
	AutoFixMaxRetries int               // Deprecated: auto_fix setting (ignored)
	AutoFixRetryCount int               // Deprecated: auto_fix state (ignored)
	ShouldStartReview bool              // Deprecated: review auto-start (always false)
	AutoFixIsLGTM     bool              // Deprecated: auto_fix result (ignored)
}

// CompleteTask is the use case for marking a task as complete.
// Status transitions depend on configuration:
//   - skip_review: directly to done
//   - otherwise: require review count to meet min_reviews, then set done
//
// Fields are ordered to minimize memory padding.
type CompleteTask struct {
	tasks     domain.TaskRepository
	sessions  domain.SessionManager
	worktrees domain.WorktreeManager
	git       domain.Git
	config    domain.ConfigLoader
	clock     domain.Clock
	logger    domain.Logger
	executor  domain.CommandExecutor
	stderr    io.Writer
	crewDir   string
	repoRoot  string
}

// NewCompleteTask creates a new CompleteTask use case.
func NewCompleteTask(
	tasks domain.TaskRepository,
	sessions domain.SessionManager,
	worktrees domain.WorktreeManager,
	git domain.Git,
	config domain.ConfigLoader,
	clock domain.Clock,
	logger domain.Logger,
	executor domain.CommandExecutor,
	stderr io.Writer,
	crewDir string,
	repoRoot string,
) *CompleteTask {
	return &CompleteTask{
		tasks:     tasks,
		sessions:  sessions,
		worktrees: worktrees,
		git:       git,
		config:    config,
		clock:     clock,
		logger:    logger,
		executor:  executor,
		stderr:    stderr,
		crewDir:   crewDir,
		repoRoot:  repoRoot,
	}
}

// Execute marks a task as complete.
// Preconditions:
//   - Status is in_progress
//   - No uncommitted changes in worktree
//
// Processing:
//   - Validate review requirement (skip_review/min_reviews)
//   - Check for merge conflicts with base branch
//   - Run [complete].command if configured (abort on failure)
//   - Set status to done and save
func (uc *CompleteTask) Execute(ctx context.Context, in CompleteTaskInput) (*CompleteTaskOutput, error) {
	// Get the task
	task, err := shared.GetTask(uc.tasks, in.TaskID)
	if err != nil {
		return nil, err
	}

	// Validate status - must be in_progress
	if task.Status != domain.StatusInProgress {
		return nil, fmt.Errorf("cannot complete task in %s status (must be in_progress): %w", task.Status, domain.ErrInvalidTransition)
	}

	// Resolve worktree path
	branch := domain.BranchName(task.ID, task.Issue)
	worktreePath, err := uc.worktrees.Resolve(branch)
	if err != nil {
		return nil, fmt.Errorf("resolve worktree: %w", err)
	}

	// Check for uncommitted changes
	hasUncommitted, err := uc.git.HasUncommittedChanges(worktreePath)
	if err != nil {
		return nil, fmt.Errorf("check uncommitted changes: %w", err)
	}
	if hasUncommitted {
		return nil, domain.ErrUncommittedChanges
	}

	// Load config for completion checks
	cfg, err := uc.config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// Determine if we should skip review
	// Priority: task.SkipReview (if explicitly set) > config.Tasks.SkipReview > false
	var skipReview bool
	if task.SkipReview != nil {
		// Task has explicit setting, use it (respects --no-skip-review)
		skipReview = *task.SkipReview
	} else if cfg != nil {
		// Fall back to config setting
		skipReview = cfg.Tasks.SkipReview
	}
	// else: default false

	minReviews := domain.DefaultMinReviews
	if cfg != nil && cfg.Complete.MinReviews > 0 {
		minReviews = cfg.Complete.MinReviews
	}
	if cfg != nil && uc.logger != nil {
		if cfg.Complete.ReviewModeSet {
			uc.logger.Warn(task.ID, "task", "complete.review_mode is deprecated and ignored")
		}
		if cfg.Complete.AutoFixSet {
			uc.logger.Warn(task.ID, "task", "complete.auto_fix is deprecated and ignored")
		}
	}

	if !skipReview && task.ReviewCount < minReviews {
		reviewErr := uc.runReview(ctx, task, in.Verbose, in.ReviewAgent, cfg)
		if reviewErr != nil {
			return nil, reviewErr
		}
	}
	if !skipReview && task.ReviewCount < minReviews {
		return nil, fmt.Errorf("review required: have %d, need %d", task.ReviewCount, minReviews)
	}

	// Resolve base branch for conflict check
	baseBranch, err := resolveBaseBranch(task, uc.git)
	if err != nil {
		return nil, err
	}

	// Check for merge conflicts with base branch
	conflictHandler := shared.NewConflictHandler(uc.tasks, uc.sessions, uc.git, uc.clock)
	conflictOut, conflictErr := conflictHandler.CheckAndHandle(shared.ConflictCheckInput{
		TaskID:     task.ID,
		Branch:     branch,
		BaseBranch: baseBranch,
	})
	if conflictErr != nil {
		return &CompleteTaskOutput{ConflictMessage: conflictOut.Message}, conflictErr
	}

	if cfg != nil && cfg.Complete.Command != "" {
		// Execute the complete command using CommandExecutor
		cmd := domain.NewShellCommand(cfg.Complete.Command, worktreePath)
		output, execErr := uc.executor.Execute(cmd)
		if execErr != nil {
			return nil, fmt.Errorf("[complete].command failed: %s: %w", string(output), execErr)
		}
	}

	// Add comment if provided (only after completion conditions are met)
	if in.Comment != "" {
		comment := domain.Comment{
			Text: in.Comment,
			Time: uc.clock.Now(),
		}
		if commentErr := uc.tasks.AddComment(task.ID, comment); commentErr != nil {
			return nil, fmt.Errorf("add comment: %w", commentErr)
		}
	}

	task.Status = domain.StatusDone

	if saveErr := uc.tasks.Save(task); saveErr != nil {
		return nil, fmt.Errorf("save task: %w", saveErr)
	}

	if uc.logger != nil {
		if skipReview {
			uc.logger.Info(task.ID, "task", "completed (status: done, skip_review: true)")
		} else {
			uc.logger.Info(task.ID, "task", "completed (status: done, review requirement satisfied)")
		}
	}

	return &CompleteTaskOutput{
		Task:              task,
		ShouldStartReview: false,
		ReviewMode:        domain.ReviewModeAuto,
	}, nil
}

func (uc *CompleteTask) runReview(ctx context.Context, task *domain.Task, verbose bool, agent string, cfg *domain.Config) error {
	preComments, err := uc.tasks.GetComments(task.ID)
	if err != nil {
		return fmt.Errorf("get comments: %w", err)
	}
	preReviewerCount := countReviewerComments(preComments)

	reviewSessionName := domain.ReviewSessionName(task.ID)
	running, err := uc.sessions.IsRunning(reviewSessionName)
	if err != nil {
		return fmt.Errorf("check review session: %w", err)
	}

	var reviewScriptPath string
	if running {
		uc.writeReviewMessage(fmt.Sprintf("Review session already running for task #%d. Waiting...", task.ID))
	} else {
		uc.writeReviewMessage(fmt.Sprintf("Starting review for task #%d...", task.ID))
		reviewCmd, err := shared.PrepareReviewCommand(shared.ReviewCommandDeps{
			ConfigLoader: uc.config,
			Config:       cfg,
			Worktrees:    uc.worktrees,
			RepoRoot:     uc.repoRoot,
		}, shared.ReviewCommandInput{
			Task:  task,
			Agent: agent,
		})
		if err != nil {
			return err
		}

		reviewScriptPath, err = uc.writeReviewScript(reviewSessionName, task.ID, reviewCmd)
		if err != nil {
			return err
		}
		defer func() {
			_ = os.Remove(reviewScriptPath)
		}()

		if err := uc.sessions.Start(ctx, domain.StartSessionOptions{
			Name:      reviewSessionName,
			Dir:       reviewCmd.WorktreePath,
			Command:   reviewScriptPath,
			TaskID:    task.ID,
			TaskTitle: task.Title,
			TaskAgent: reviewCmd.AgentName,
			Type:      domain.SessionTypeReviewer,
		}); err != nil {
			return fmt.Errorf("start review session: %w", err)
		}
	}

	if err := uc.waitForReview(ctx, reviewSessionName, verbose); err != nil {
		return err
	}
	uc.writeReviewMessage(fmt.Sprintf("Review finished for task #%d.", task.ID))

	if err := uc.updateReviewMetadata(task, preReviewerCount, reviewSessionName); err != nil {
		return err
	}

	return nil
}

func (uc *CompleteTask) writeReviewScript(sessionName string, taskID int, reviewCmd *shared.ReviewCommandOutput) (string, error) {
	scriptsDir := filepath.Join(uc.crewDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0750); err != nil {
		return "", fmt.Errorf("create scripts directory: %w", err)
	}

	logPath := domain.SessionLogPath(uc.crewDir, sessionName)
	if err := os.MkdirAll(filepath.Dir(logPath), 0750); err != nil {
		return "", fmt.Errorf("create log directory: %w", err)
	}
	script := fmt.Sprintf(`#!/bin/bash
set -o pipefail

exec > >(tee -a %q) 2>&1

read -r -d '' PROMPT << 'END_OF_PROMPT'
%s
END_OF_PROMPT

%s
`, logPath, reviewCmd.Result.Prompt, reviewCmd.Result.Command)

	file, err := os.CreateTemp(scriptsDir, fmt.Sprintf("review-%d-*.sh", taskID))
	if err != nil {
		return "", fmt.Errorf("create review script: %w", err)
	}

	scriptPath := file.Name()
	if _, err := file.WriteString(script); err != nil {
		_ = file.Close()
		return "", fmt.Errorf("write review script: %w", err)
	}
	if err := file.Chmod(0700); err != nil {
		_ = file.Close()
		return "", fmt.Errorf("chmod review script: %w", err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("close review script: %w", err)
	}

	return scriptPath, nil
}

func (uc *CompleteTask) waitForReview(ctx context.Context, sessionName string, verbose bool) error {
	if !verbose || uc.stderr == nil {
		err := uc.sessions.Wait(ctx, sessionName)
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			_ = uc.sessions.Stop(sessionName)
		}
		return err
	}

	resultCh := make(chan error, 1)
	go func() {
		resultCh <- uc.sessions.Wait(ctx, sessionName)
	}()

	lastOutput := ""
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case err := <-resultCh:
			_ = uc.streamSessionOutput(sessionName, &lastOutput)
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				_ = uc.sessions.Stop(sessionName)
			}
			return err
		case <-ticker.C:
			_ = uc.streamSessionOutput(sessionName, &lastOutput)
		case <-ctx.Done():
			_ = uc.sessions.Stop(sessionName)
			return ctx.Err()
		}
	}
}

func (uc *CompleteTask) streamSessionOutput(sessionName string, lastOutput *string) error {
	output, err := uc.sessions.Peek(sessionName, reviewPeekLines, true)
	if err != nil {
		if errors.Is(err, domain.ErrNoSession) {
			return nil
		}
		return err
	}
	newOutput := diffOutput(*lastOutput, output)
	newOutput = strings.TrimPrefix(newOutput, "\n")
	if strings.TrimSpace(newOutput) != "" {
		_, _ = fmt.Fprintln(uc.stderr, newOutput)
	}

	*lastOutput = output
	return nil
}

func (uc *CompleteTask) updateReviewMetadata(task *domain.Task, preReviewerCount int, sessionName string) error {
	postComments, err := uc.tasks.GetComments(task.ID)
	if err != nil {
		return fmt.Errorf("get comments: %w", err)
	}
	postReviewerCount := countReviewerComments(postComments)
	if postReviewerCount <= preReviewerCount {
		return uc.noReviewCommentError(sessionName)
	}
	lastReviewerComment, ok := lastReviewerComment(postComments)
	if !ok {
		return uc.noReviewCommentError(sessionName)
	}

	shared.UpdateReviewMetadata(uc.clock, task, lastReviewerComment.Text)
	if err := uc.tasks.Save(task); err != nil {
		return fmt.Errorf("save review metadata: %w", err)
	}

	return nil
}

func (uc *CompleteTask) writeReviewMessage(message string) {
	if uc.stderr == nil {
		return
	}
	_, _ = fmt.Fprintln(uc.stderr, message)
}

func (uc *CompleteTask) noReviewCommentError(sessionName string) error {
	logPath := domain.SessionLogPath(uc.crewDir, sessionName)
	logTail := tailFileLines(logPath, reviewLogTailLines)
	if logTail == "" {
		return fmt.Errorf("reviewer did not add a comment: %w (log: %s)", domain.ErrNoReviewComment, logPath)
	}

	return fmt.Errorf("reviewer did not add a comment: %w\n\nreview log (tail from %s):\n%s", domain.ErrNoReviewComment, logPath, logTail)
}

func tailFileLines(path string, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}
	file, openErr := os.Open(path)
	if openErr != nil {
		return ""
	}
	defer func() {
		_ = file.Close()
	}()

	info, statErr := file.Stat()
	if statErr != nil {
		return ""
	}
	if info.Size() == 0 {
		return ""
	}

	offset := int64(0)
	if info.Size() > reviewLogTailMaxBytes {
		offset = info.Size() - reviewLogTailMaxBytes
	}
	if offset > 0 {
		if _, seekErr := file.Seek(offset, io.SeekStart); seekErr != nil {
			return ""
		}
	}
	data, readErr := io.ReadAll(file)
	if readErr != nil {
		return ""
	}
	text := string(data)
	if offset > 0 {
		if idx := strings.IndexByte(text, '\n'); idx >= 0 {
			text = text[idx+1:]
		}
	}
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return strings.Join(lines, "\n")
}

func diffOutput(prev, curr string) string {
	if curr == "" || curr == prev {
		return ""
	}
	if prev == "" {
		return curr
	}
	if strings.HasPrefix(curr, prev) {
		return strings.TrimPrefix(curr, prev)
	}
	for i := 0; i < len(prev); i++ {
		suffix := prev[i:]
		if strings.HasPrefix(curr, suffix) {
			return strings.TrimPrefix(curr, suffix)
		}
	}
	return curr
}

func countReviewerComments(comments []domain.Comment) int {
	count := 0
	for _, c := range comments {
		if c.Author == "reviewer" {
			count++
		}
	}
	return count
}

func lastReviewerComment(comments []domain.Comment) (domain.Comment, bool) {
	for i := len(comments) - 1; i >= 0; i-- {
		if comments[i].Author == "reviewer" {
			return comments[i], true
		}
	}
	return domain.Comment{}, false
}
