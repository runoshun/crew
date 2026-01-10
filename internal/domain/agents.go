package domain

import "path/filepath"

// IsAgentDisabled checks if an agent name matches any of the disabled patterns.
// Patterns support:
//   - Exact match: "agent1" matches only "agent1"
//   - Wildcard (*): "oc-*" matches "oc-small", "oc-medium", etc.
//   - Exclusion (!): "!oc-medium" re-enables an agent that would otherwise be disabled
//
// Exclusion patterns are always evaluated last, regardless of their position in the list.
// This means ["oc-*", "!oc-medium"] and ["!oc-medium", "oc-*"] produce the same result.
func IsAgentDisabled(name string, patterns []string) bool {
	disabled := false

	// First pass: check all non-exclusion patterns
	for _, pattern := range patterns {
		if len(pattern) == 0 {
			continue
		}
		if pattern[0] == '!' {
			continue // Skip exclusion patterns in first pass
		}
		if matchPattern(pattern, name) {
			disabled = true
		}
	}

	// Second pass: check exclusion patterns (re-enable if matched)
	for _, pattern := range patterns {
		if len(pattern) == 0 {
			continue
		}
		if pattern[0] != '!' {
			continue // Skip non-exclusion patterns in second pass
		}
		exclusionPattern := pattern[1:]
		if matchPattern(exclusionPattern, name) {
			disabled = false
		}
	}

	return disabled
}

// matchPattern checks if a name matches a pattern using filepath.Match semantics.
// Supports wildcards: * matches any sequence of non-separator characters.
func matchPattern(pattern, name string) bool {
	// filepath.Match returns error only for malformed patterns
	matched, err := filepath.Match(pattern, name)
	if err != nil {
		// Treat malformed patterns as non-matching
		return false
	}
	return matched
}
