package shared

import "regexp"

var envNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// IsValidEnvVarName returns true if the name is a valid environment variable name.
func IsValidEnvVarName(name string) bool {
	return envNamePattern.MatchString(name)
}
