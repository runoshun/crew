package main

import (
	"errors"
	"os"
	"testing"

	"github.com/runoshun/git-crew/v2/internal/cli"
)

func TestCanRunWithoutGit(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{
			name: "no args",
			args: nil,
			want: true,
		},
		{
			name: "help flag",
			args: []string{"--help"},
			want: true,
		},
		{
			name: "help follow-up order",
			args: []string{"--follow-up", "--help-reviewer"},
			want: true,
		},
		{
			name: "follow-up only",
			args: []string{"--follow-up"},
			want: false,
		},
		{
			name: "version flag",
			args: []string{"--version"},
			want: true,
		},
		{
			name: "help subcommand",
			args: []string{"help", "new"},
			want: true,
		},
		{
			name: "workspace list",
			args: []string{"workspace", "list"},
			want: true,
		},
		{
			name: "ws alias",
			args: []string{"ws"},
			want: true,
		},
		{
			name: "non-allowed command",
			args: []string{"new", "--title", "test"},
			want: false,
		},
		{
			name: "unknown help flag",
			args: []string{"--help-foo"},
			want: false,
		},
		{
			name: "version shorthand",
			args: []string{"-v"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canRunWithoutGit(tt.args); got != tt.want {
				t.Fatalf("canRunWithoutGit(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestRunWithoutContainer_NoArgsExecutes(t *testing.T) {
	originalArgs := os.Args
	originalRoot := newRootCommand
	called := false
	restore := cli.SetLaunchUnifiedTUIFunc(func(string) error {
		called = true
		return nil
	})
	t.Cleanup(func() {
		os.Args = originalArgs
		newRootCommand = originalRoot
		restore()
	})

	os.Args = []string{"crew"}
	newRootCommand = cli.NewRootCommand

	if err := runWithoutContainer(errors.New("git")); err != nil {
		t.Fatalf("runWithoutContainer returned error: %v", err)
	}
	if !called {
		t.Fatalf("expected root command to execute")
	}
}
