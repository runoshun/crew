package domain

// ExecCommand represents an external command to be executed.
// This type is used to pass command information between layers
// without exposing implementation details.
// Fields are ordered to minimize memory padding.
type ExecCommand struct {
	Program string   // The command to execute (e.g., "sh", "bash", "git")
	Dir     string   // Working directory (empty means current directory)
	Args    []string // Command arguments
}

// NewShellCommand creates an ExecCommand that runs a shell command.
// It uses "sh -c" to execute the command string.
func NewShellCommand(cmd, dir string) *ExecCommand {
	return &ExecCommand{
		Program: "sh",
		Args:    []string{"-c", cmd},
		Dir:     dir,
	}
}

// NewBashCommand creates an ExecCommand that runs a bash script.
func NewBashCommand(script, dir string) *ExecCommand {
	return &ExecCommand{
		Program: "bash",
		Args:    []string{"-c", script},
		Dir:     dir,
	}
}

// NewCommand creates an ExecCommand with explicit program and arguments.
func NewCommand(program string, args []string, dir string) *ExecCommand {
	return &ExecCommand{
		Program: program,
		Args:    args,
		Dir:     dir,
	}
}
