package domain

// ExecCommand represents an external command to be executed.
// This type is used to pass command information between layers
// without exposing implementation details.
type ExecCommand struct {
	Program string
	Dir     string
	Args    []string
}
