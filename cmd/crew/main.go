// Package main is the entry point for the git-crew CLI.
package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/cli"
	"github.com/runoshun/git-crew/v2/internal/domain"
)

// version is set at build time using -ldflags.
var version = "dev"

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
		// Allow running without git repo for --version and --help
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
	rootCmd := cli.NewRootCommand(nil, version)

	// Commands that can run without a git repository
	if len(os.Args) == 1 {
		return rootCmd.Execute()
	}
	arg := os.Args[1]
	if arg == "--version" || arg == "-v" || arg == "version" ||
		arg == "--help" || arg == "-h" || arg == "help" ||
		strings.HasPrefix(arg, "--help-") ||
		arg == "workspace" || arg == "ws" {
		return rootCmd.Execute()
	}

	// For other commands, return the git error
	return gitErr
}
