package usecase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase/shared"
)

// ACPStartInput contains the parameters for starting an ACP session in tmux.
// Fields are ordered to minimize memory padding.
type ACPStartInput struct {
	Agent  string // Agent name (optional; uses default worker if empty)
	Model  string // Model override (optional)
	TaskID int    // Task ID
}

// ACPStartOutput contains the result of starting an ACP session.
type ACPStartOutput struct {
	SessionName  string // Name of the tmux session
	WorktreePath string // Path to the worktree
}

// ACPStart is the use case for starting ACP sessions in tmux.
// Fields are ordered to minimize memory padding.
type ACPStart struct {
	tasks        domain.TaskRepository
	sessions     domain.SessionManager
	worktrees    domain.WorktreeManager
	configLoader domain.ConfigLoader
	git          domain.Git
	runner       domain.ScriptRunner
	crewDir      string
	repoRoot     string
}

// NewACPStart creates a new ACPStart use case.
func NewACPStart(
	tasks domain.TaskRepository,
	sessions domain.SessionManager,
	worktrees domain.WorktreeManager,
	configLoader domain.ConfigLoader,
	git domain.Git,
	runner domain.ScriptRunner,
	crewDir string,
	repoRoot string,
) *ACPStart {
	return &ACPStart{
		tasks:        tasks,
		sessions:     sessions,
		worktrees:    worktrees,
		configLoader: configLoader,
		git:          git,
		runner:       runner,
		crewDir:      crewDir,
		repoRoot:     repoRoot,
	}
}

// Execute starts an ACP session in tmux for the given task.
func (uc *ACPStart) Execute(ctx context.Context, in ACPStartInput) (*ACPStartOutput, error) {
	task, err := shared.GetTask(uc.tasks, in.TaskID)
	if err != nil {
		return nil, err
	}

	if !task.Status.CanStart() {
		return nil, fmt.Errorf("cannot start task with status %q: %w", task.Status, domain.ErrInvalidTransition)
	}
	if task.IsBlocked() {
		return nil, fmt.Errorf("%w: %q", domain.ErrTaskBlocked, task.BlockReason)
	}

	cfg, err := uc.configLoader.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	agentName := in.Agent
	if agentName == "" {
		agentName = cfg.AgentsConfig.DefaultWorker
	}
	if agentName == "" {
		return nil, domain.ErrNoAgent
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

	wtPath, worktreeCreated, err := uc.ensureWorktree(task, cfg, agent)
	if err != nil {
		return nil, err
	}

	sessionName := domain.ACPSessionName(task.ID)
	if running, isRunningErr := uc.sessions.IsRunning(sessionName); isRunningErr != nil {
		return nil, fmt.Errorf("check session: %w", isRunningErr)
	} else if running {
		return nil, fmt.Errorf("acp session %q: %w", sessionName, domain.ErrSessionRunning)
	}

	command := uc.buildACPRunCommand(task.ID, agentName, model)

	scriptPath, err := uc.generateScript(task.ID, command, wtPath)
	if err != nil {
		if worktreeCreated {
			_ = uc.worktrees.Remove(domain.BranchName(task.ID, task.Issue))
		}
		return nil, err
	}

	if err := uc.sessions.Start(ctx, domain.StartSessionOptions{
		Name:      sessionName,
		Dir:       wtPath,
		Command:   scriptPath,
		TaskTitle: task.Title,
		TaskAgent: agentName,
		TaskID:    task.ID,
		Type:      domain.SessionTypeWorker,
	}); err != nil {
		_ = os.Remove(scriptPath)
		if worktreeCreated {
			_ = uc.worktrees.Remove(domain.BranchName(task.ID, task.Issue))
		}
		return nil, fmt.Errorf("start session: %w", err)
	}

	return &ACPStartOutput{
		SessionName:  sessionName,
		WorktreePath: wtPath,
	}, nil
}

func (uc *ACPStart) ensureWorktree(task *domain.Task, cfg *domain.Config, agent domain.Agent) (string, bool, error) {
	branch := domain.BranchName(task.ID, task.Issue)
	exists, err := uc.worktrees.Exists(branch)
	if err != nil {
		return "", false, fmt.Errorf("check worktree: %w", err)
	}
	if exists {
		wtPath, resolveErr := uc.worktrees.Resolve(branch)
		if resolveErr != nil {
			return "", false, fmt.Errorf("resolve worktree: %w", resolveErr)
		}
		return wtPath, false, nil
	}

	baseBranch, err := resolveBaseBranch(task, uc.git)
	if err != nil {
		return "", false, err
	}

	wtPath, err := uc.worktrees.Create(branch, baseBranch)
	if err != nil {
		return "", false, fmt.Errorf("create worktree: %w", err)
	}

	if setupErr := uc.worktrees.SetupWorktree(wtPath, &cfg.Worktree); setupErr != nil {
		_ = uc.worktrees.Remove(branch)
		return "", false, fmt.Errorf("setup worktree: %w", setupErr)
	}

	if setupErr := uc.setupAgent(task, wtPath, agent); setupErr != nil {
		_ = uc.worktrees.Remove(branch)
		return "", false, fmt.Errorf("setup agent: %w", setupErr)
	}

	return wtPath, true, nil
}

func (uc *ACPStart) setupAgent(task *domain.Task, wtPath string, agent domain.Agent) error {
	if agent.SetupScript == "" {
		return nil
	}

	gitDir := filepath.Join(uc.repoRoot, ".git")
	data := struct {
		GitDir   string
		RepoRoot string
		Worktree string
		TaskID   int
	}{
		GitDir:   gitDir,
		RepoRoot: uc.repoRoot,
		Worktree: wtPath,
		TaskID:   task.ID,
	}

	tmpl, err := template.New("acp-start-setup").Parse(agent.SetupScript)
	if err != nil {
		return fmt.Errorf("parse setup script template: %w", err)
	}

	var script strings.Builder
	if err := tmpl.Execute(&script, data); err != nil {
		return fmt.Errorf("expand setup script template: %w", err)
	}

	if err := uc.runner.Run(wtPath, script.String()); err != nil {
		return fmt.Errorf("run setup script: %w", err)
	}

	return nil
}

func (uc *ACPStart) buildACPRunCommand(taskID int, agentName, model string) string {
	crewBin, err := os.Executable()
	if err != nil {
		crewBin = "crew"
	}

	args := []string{
		crewBin,
		"acp",
		"run",
		"--task", strconv.Itoa(taskID),
		"--agent", agentName,
	}
	if model != "" {
		args = append(args, "--model", model)
	}

	return shellJoin(args)
}

type acpScriptData struct {
	Command string
	LogPath string
}

const acpScriptTemplate = `#!/bin/bash
set -o pipefail

# Redirect stderr to session log
exec 2>>"{{.LogPath}}"

# Run ACP session
{{.Command}}
`

func (uc *ACPStart) generateScript(taskID int, command string, worktreePath string) (string, error) {
	scriptsDir := filepath.Join(uc.crewDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0750); err != nil {
		return "", fmt.Errorf("create scripts directory: %w", err)
	}

	sessionName := domain.ACPSessionName(taskID)
	logPath := domain.SessionLogPath(uc.crewDir, sessionName)
	if err := writeACPSessionLogHeader(logPath, sessionName, worktreePath, command); err != nil {
		return "", fmt.Errorf("write session log header: %w", err)
	}

	tmpl := template.Must(template.New("acp-script").Parse(acpScriptTemplate))
	data := acpScriptData{Command: command, LogPath: logPath}
	var script strings.Builder
	if err := tmpl.Execute(&script, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	scriptPath := domain.ACPScriptPath(uc.crewDir, taskID)
	// G306: We intentionally use 0700 because this is an executable script
	if err := os.WriteFile(scriptPath, []byte(script.String()), 0700); err != nil { //nolint:gosec // executable script requires execute permission
		return "", fmt.Errorf("write script file: %w", err)
	}

	return scriptPath, nil
}

func writeACPSessionLogHeader(logPath, sessionName, workDir, command string) error {
	logsDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logsDir, 0750); err != nil {
		return fmt.Errorf("create logs directory: %w", err)
	}

	header := fmt.Sprintf(`================================================================================
Session: %s
Started: %s
Directory: %s
Command: %s
================================================================================

`, sessionName, time.Now().UTC().Format(time.RFC3339), workDir, command)

	if err := os.WriteFile(logPath, []byte(header), 0600); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	return nil
}

func shellJoin(parts []string) string {
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		quoted = append(quoted, shellQuote(part))
	}
	return strings.Join(quoted, " ")
}
