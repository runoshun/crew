package main

import "testing"

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canRunWithoutGit(tt.args); got != tt.want {
				t.Fatalf("canRunWithoutGit(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}
