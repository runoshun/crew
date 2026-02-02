package usecase

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase/shared"
)

const (
	reviewPeekLines       = 2000
	reviewLogTailLines    = 20
	reviewLogTailMaxBytes = 64 * 1024
	reviewLogReadMaxBytes = 256 * 1024
	reviewProgressEvery   = 2 * time.Minute
	reviewRunStartPrefix  = "---CREW_REVIEW_RUN_START--- "
)

// CompleteTaskInput contains the parameters for completing a task.
type CompleteTaskInput struct {
	Comment     string // Optional completion comment
	ReviewAgent string // Reviewer agent override (optional)
	TaskID      int    // Task ID to complete
	ForceReview bool   // Force running review even if not required
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
//   - otherwise: run review until success or max_reviews, then set done
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
//   - Validate review requirement (skip_review/max_reviews or forced review)
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

	maxReviews := domain.DefaultMaxReviews
	reviewSuccessRegex := domain.DefaultReviewSuccessRegex
	if cfg != nil {
		if cfg.Complete.MaxReviews > 0 {
			maxReviews = cfg.Complete.MaxReviews
		}
		if cfg.Complete.ReviewSuccessRegex != "" {
			reviewSuccessRegex = cfg.Complete.ReviewSuccessRegex
		}
	}
	if cfg != nil && uc.logger != nil {
		if cfg.Complete.ReviewModeSet {
			uc.logger.Warn(task.ID, "task", "complete.review_mode is deprecated and ignored")
		}
		if cfg.Complete.AutoFixSet {
			uc.logger.Warn(task.ID, "task", "complete.auto_fix is deprecated and ignored")
		}
	}

	// Run [complete].command before conflict/review (CI gate)
	if cfg != nil && cfg.Complete.Command != "" {
		cmd := domain.NewShellCommand(cfg.Complete.Command, worktreePath)
		output, execErr := uc.executor.Execute(cmd)
		if execErr != nil {
			return nil, fmt.Errorf("[complete].command failed: %s: %w", string(output), execErr)
		}
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

	shouldRunReview := in.ForceReview || !skipReview
	requireReviewSuccess := !skipReview
	if shouldRunReview {
		pattern := domain.AnchorReviewSuccessRegex(reviewSuccessRegex)
		reviewMatcher, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid review success regex: %w", err)
		}

		reviewSucceeded := false
		for attempt := 1; attempt <= maxReviews; attempt++ {
			reviewResult, reviewErr := uc.runReview(ctx, task, in.Verbose, in.ReviewAgent, cfg)
			if reviewErr != nil {
				return nil, reviewErr
			}
			if reviewMatcher.MatchString(reviewResult) {
				reviewSucceeded = true
				break
			}
		}
		if requireReviewSuccess && !reviewSucceeded {
			return nil, fmt.Errorf("review required: no review comment matched %q after %d attempt(s)", reviewSuccessRegex, maxReviews)
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

func (uc *CompleteTask) runReview(ctx context.Context, task *domain.Task, verbose bool, agent string, cfg *domain.Config) (string, error) {
	reviewSessionName := domain.ReviewSessionName(task.ID)
	running, err := uc.sessions.IsRunning(reviewSessionName)
	if err != nil {
		return "", fmt.Errorf("check review session: %w", err)
	}

	startedAt := uc.clock.Now()
	logOffset := int64(0)
	if running {
		// If the session is already running, try to avoid accidentally parsing an older review
		// result by only considering log content written after we start waiting.
		logPath := domain.SessionLogPath(uc.crewDir, reviewSessionName)
		if offset, t, ok := findLastReviewRunStart(logPath, int64(reviewLogReadMaxBytes)); ok {
			logOffset = offset
			if !t.IsZero() {
				startedAt = t
			}
		}
	}

	var reviewScriptPath string
	if running {
		uc.writeReviewMessage(fmt.Sprintf("Review session already running for task #%d. Waiting...", task.ID))
		uc.writeReviewMessage(fmt.Sprintf("Note: review may take a while. If it takes too long, re-run 'crew complete %d'.", task.ID))
	} else {
		uc.writeReviewMessage(fmt.Sprintf("Starting review for task #%d...", task.ID))
		uc.writeReviewMessage(fmt.Sprintf("Note: review may take a while. If it takes too long, re-run 'crew complete %d'.", task.ID))
		reviewCmd, prepareErr := shared.PrepareReviewCommand(shared.ReviewCommandDeps{
			ConfigLoader: uc.config,
			Config:       cfg,
			Worktrees:    uc.worktrees,
			RepoRoot:     uc.repoRoot,
		}, shared.ReviewCommandInput{
			Task:  task,
			Agent: agent,
		})
		if prepareErr != nil {
			return "", prepareErr
		}

		var scriptErr error
		reviewScriptPath, scriptErr = uc.writeReviewScript(reviewSessionName, task.ID, reviewCmd, startedAt)
		if scriptErr != nil {
			return "", scriptErr
		}
		defer func() {
			_ = os.Remove(reviewScriptPath)
		}()

		startErr := uc.sessions.Start(ctx, domain.StartSessionOptions{
			Name:      reviewSessionName,
			Dir:       reviewCmd.WorktreePath,
			Command:   reviewScriptPath,
			TaskID:    task.ID,
			TaskTitle: task.Title,
			TaskAgent: reviewCmd.AgentName,
			Type:      domain.SessionTypeReviewer,
		})
		if startErr != nil {
			return "", fmt.Errorf("start review session: %w", startErr)
		}
	}

	waitErr := uc.waitForReview(ctx, reviewSessionName, verbose, startedAt)
	if waitErr != nil {
		return "", waitErr
	}
	uc.writeReviewMessage(fmt.Sprintf("Review finished for task #%d.", task.ID))

	result, err := uc.updateReviewMetadata(task, reviewSessionName, logOffset)
	if err != nil {
		return "", err
	}

	return result, nil
}

func (uc *CompleteTask) writeReviewScript(sessionName string, taskID int, reviewCmd *shared.ReviewCommandOutput, startedAt time.Time) (string, error) {
	scriptsDir := filepath.Join(uc.crewDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0750); err != nil {
		return "", fmt.Errorf("create scripts directory: %w", err)
	}

	logPath := domain.SessionLogPath(uc.crewDir, sessionName)
	if err := os.MkdirAll(filepath.Dir(logPath), 0750); err != nil {
		return "", fmt.Errorf("create log directory: %w", err)
	}
	startStr := startedAt.UTC().Format(time.RFC3339)
	script := fmt.Sprintf(`#!/bin/bash
set -o pipefail

exec > >(tee %q) 2>&1

echo "---CREW_REVIEW_RUN_START--- %s"

read -r -d '' PROMPT << 'END_OF_PROMPT'
%s
END_OF_PROMPT

%s
`, logPath, startStr, reviewCmd.Result.Prompt, reviewCmd.Result.Command)

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

func (uc *CompleteTask) waitForReview(ctx context.Context, sessionName string, verbose bool, startedAt time.Time) error {
	resultCh := make(chan error, 1)
	go func() {
		resultCh <- uc.sessions.Wait(ctx, sessionName)
	}()

	lastOutput := ""
	var streamCh <-chan time.Time
	var progressCh <-chan time.Time
	if verbose && uc.stderr != nil {
		t := time.NewTicker(1 * time.Second)
		defer t.Stop()
		streamCh = t.C
	}
	if uc.stderr != nil {
		t := time.NewTicker(reviewProgressEvery)
		defer t.Stop()
		progressCh = t.C
	}

	if startedAt.IsZero() {
		startedAt = uc.clock.Now()
	}

	for {
		select {
		case err := <-resultCh:
			if streamCh != nil {
				_ = uc.streamSessionOutput(sessionName, &lastOutput)
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				_ = uc.sessions.Stop(sessionName)
			}
			return err

		case <-ctx.Done():
			_ = uc.sessions.Stop(sessionName)
			return ctx.Err()

		case <-streamCh:
			_ = uc.streamSessionOutput(sessionName, &lastOutput)

		case <-progressCh:
			elapsed := uc.clock.Now().Sub(startedAt)
			mins := int(elapsed / time.Minute)
			if mins < 0 {
				mins = 0
			}
			uc.writeReviewMessage(fmt.Sprintf("Review still running... elapsed %dm", mins))
		}
	}
}

func findLastReviewRunStart(logPath string, maxBytes int64) (int64, time.Time, bool) {
	if maxBytes <= 0 {
		return 0, time.Time{}, false
	}
	file, err := os.Open(logPath)
	if err != nil {
		return 0, time.Time{}, false
	}
	defer func() {
		_ = file.Close()
	}()

	info, err := file.Stat()
	if err != nil {
		return 0, time.Time{}, false
	}
	if info.Size() == 0 {
		return 0, time.Time{}, false
	}

	baseOffset := int64(0)
	if info.Size() > maxBytes {
		baseOffset = info.Size() - maxBytes
	}
	if baseOffset > 0 {
		if _, seekErr := file.Seek(baseOffset, io.SeekStart); seekErr != nil {
			return 0, time.Time{}, false
		}
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return 0, time.Time{}, false
	}
	text := string(data)
	if baseOffset > 0 {
		if idx := strings.IndexByte(text, '\n'); idx >= 0 {
			baseOffset += int64(idx + 1)
			text = text[idx+1:]
		} else {
			return 0, time.Time{}, false
		}
	}
	searchText := text
	idx := -1
	for {
		idx = strings.LastIndex(searchText, reviewRunStartPrefix)
		if idx < 0 {
			return 0, time.Time{}, false
		}
		if idx == 0 || searchText[idx-1] == '\n' {
			break
		}
		searchText = searchText[:idx]
	}
	line := searchText[idx:]
	if nl := strings.IndexByte(line, '\n'); nl >= 0 {
		line = line[:nl]
	}
	ts := strings.TrimSpace(strings.TrimPrefix(line, reviewRunStartPrefix))
	if ts == "" {
		return baseOffset + int64(idx), time.Time{}, true
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return baseOffset + int64(idx), time.Time{}, true
	}
	return baseOffset + int64(idx), t, true
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

func (uc *CompleteTask) updateReviewMetadata(task *domain.Task, sessionName string, logOffset int64) (string, error) {
	logPath := domain.SessionLogPath(uc.crewDir, sessionName)
	logTail := readFileTailBytes(logPath, logOffset, reviewLogReadMaxBytes)
	result, ok := extractReviewResult(logTail)
	if !ok {
		return "", uc.noReviewCommentError(sessionName)
	}
	result = strings.TrimSpace(result)
	if result == "" {
		return "", uc.noReviewCommentError(sessionName)
	}

	comment := domain.Comment{
		Author: "reviewer",
		Text:   result,
		Time:   uc.clock.Now(),
	}
	if err := uc.tasks.AddComment(task.ID, comment); err != nil {
		return "", fmt.Errorf("add review comment: %w", err)
	}

	shared.UpdateReviewMetadata(uc.clock, task, result)
	if err := uc.tasks.Save(task); err != nil {
		return "", fmt.Errorf("save review metadata: %w", err)
	}

	return result, nil
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
		return fmt.Errorf("reviewer did not output a review result: %w (log: %s)", domain.ErrNoReviewComment, logPath)
	}

	return fmt.Errorf("reviewer did not output a review result: %w\n\nreview log (tail from %s):\n%s", domain.ErrNoReviewComment, logPath, logTail)
}

func readFileTailBytes(path string, offset int64, maxBytes int64) string {
	if maxBytes <= 0 {
		return ""
	}
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() {
		_ = file.Close()
	}()

	info, err := file.Stat()
	if err != nil {
		return ""
	}
	if info.Size() == 0 {
		return ""
	}

	if offset < 0 {
		offset = 0
	}
	if offset > info.Size() {
		// File was rotated/truncated.
		offset = 0
	}

	// Cap reads to the last maxBytes.
	capOffset := int64(0)
	if info.Size() > maxBytes {
		capOffset = info.Size() - maxBytes
	}
	if offset < capOffset {
		offset = capOffset
	}

	if offset > 0 {
		if _, seekErr := file.Seek(offset, io.SeekStart); seekErr != nil {
			return ""
		}
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return ""
	}

	text := string(data)
	if offset > 0 {
		if idx := strings.IndexByte(text, '\n'); idx >= 0 {
			text = text[idx+1:]
		}
	}
	return text
}

func extractReviewResult(logText string) (string, bool) {
	if strings.TrimSpace(logText) == "" {
		return "", false
	}

	// If the log contains multiple review runs (e.g. file was appended), only consider
	// the latest run to avoid accidentally picking an old marker.
	searchText := logText
	idx := -1
	for {
		idx = strings.LastIndex(searchText, reviewRunStartPrefix)
		if idx < 0 {
			break
		}
		if idx == 0 || searchText[idx-1] == '\n' {
			break
		}
		searchText = searchText[:idx]
	}
	if idx >= 0 {
		logText = logText[idx+len(reviewRunStartPrefix):]
		if nl := strings.IndexByte(logText, '\n'); nl >= 0 {
			logText = logText[nl+1:]
		} else {
			logText = ""
		}
		logText = strings.TrimSpace(logText)
		if logText == "" {
			return "", false
		}
	}

	lines := strings.Split(logText, "\n")
	markerIdx := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) == domain.ReviewResultMarker {
			markerIdx = i
			break
		}
	}
	if markerIdx < 0 {
		return "", false
	}
	if markerIdx+1 >= len(lines) {
		return "", false
	}
	result := strings.Join(lines[markerIdx+1:], "\n")
	result = strings.TrimSpace(result)
	if result == "" {
		return "", false
	}
	return result, true
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
