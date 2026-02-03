// Package main is the entry point for the git-crew CLI.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/cli"
	"github.com/runoshun/git-crew/v2/internal/domain"
)

// version is set at build time using -ldflags.
var version = "dev"

var newRootCommand = cli.NewRootCommand

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Create dependency injection container
	container, err := app.New(cwd)
	if err != nil {
		// Allow running without git repo for no-args/help/version/workspace
		if errors.Is(err, domain.ErrNotGitRepository) {
			return runWithoutContainer(err)
		}
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// Create and execute root command
	rootCmd := cli.NewRootCommand(container, version)
	return rootCmd.Execute()
}

// runWithoutContainer handles cases where git repo is not found.
// This allows no-args, help, version, and workspace commands to work without a git repository.
func runWithoutContainer(gitErr error) error {
	rootCmd := newRootCommand(nil, version)

	// Commands that can run without a git repository
	if canRunWithoutGit(os.Args[1:]) {
		return rootCmd.Execute()
	}
	// For other commands, return the git error
	return gitErr
}

func canRunWithoutGit(args []string) bool {
	if len(args) == 0 {
		return true
	}
	switch args[0] {
	case "help", "workspace", "ws":
		return true
	}
	for _, arg := range args {
		if isAllowedWithoutGitFlag(arg) {
			return true
		}
	}
	return false
}

func isAllowedWithoutGitFlag(arg string) bool {
	switch arg {
	case "--version", "--help", "-h", "--follow-up",
		"--help-worker", "--help-reviewer", "--help-reviwer",
		"--help-manager", "--help-manager-onboarding", "--help-manager-auto":
		return true
	default:
		return false
	}
}
