package domain

import "testing"

func TestIsAgentDisabled(t *testing.T) {
	tests := []struct {
		name     string
		agent    string
		patterns []string
		want     bool
	}{
		// Exact match
		{"exact match disabled", "agent1", []string{"agent1"}, true},
		{"exact match not disabled", "agent2", []string{"agent1"}, false},
		{"multiple exact patterns", "agent2", []string{"agent1", "agent2"}, true},
		{"empty patterns", "agent1", []string{}, false},
		{"empty pattern string", "agent1", []string{""}, false},

		// Prefix wildcard (oc-*)
		{"prefix wildcard match", "oc-small", []string{"oc-*"}, true},
		{"prefix wildcard match long", "oc-medium-large", []string{"oc-*"}, true},
		{"prefix wildcard no match", "cc-small", []string{"oc-*"}, false},
		{"prefix wildcard exact prefix", "oc-", []string{"oc-*"}, true},

		// Suffix wildcard (*-small)
		{"suffix wildcard match", "oc-small", []string{"*-small"}, true},
		{"suffix wildcard match different prefix", "cc-small", []string{"*-small"}, true},
		{"suffix wildcard no match", "oc-large", []string{"*-small"}, false},

		// Middle wildcard (oc-*-ag)
		{"middle wildcard match", "oc-medium-ag", []string{"oc-*-ag"}, true},
		{"middle wildcard match long", "oc-super-large-ag", []string{"oc-*-ag"}, true},
		{"middle wildcard no match prefix", "cc-medium-ag", []string{"oc-*-ag"}, false},
		{"middle wildcard no match suffix", "oc-medium-xx", []string{"oc-*-ag"}, false},

		// Multiple wildcards
		{"multiple wildcards", "a-b-c", []string{"*-*-*"}, true},
		{"multiple wildcards no match", "a-b", []string{"*-*-*"}, false},

		// Exclusion patterns
		{"exclusion re-enables", "oc-medium", []string{"oc-*", "!oc-medium"}, false},
		{"exclusion order independent", "oc-medium", []string{"!oc-medium", "oc-*"}, false},
		{"exclusion only affects matched", "oc-small", []string{"oc-*", "!oc-medium"}, true},
		{"exclusion with wildcard", "oc-medium", []string{"oc-*", "!oc-med*"}, false},
		{"exclusion without prior match", "agent1", []string{"!agent1"}, false},

		// Complex scenarios
		{"complex: multiple disable and exclude", "oc-medium", []string{"oc-*", "cc-*", "!oc-medium"}, false},
		{"complex: disabled by second pattern", "cc-large", []string{"oc-*", "cc-*", "!oc-medium"}, true},
		{"complex: not in any pattern", "xx-small", []string{"oc-*", "cc-*", "!oc-medium"}, false},

		// Edge cases
		{"single char wildcard", "a", []string{"*"}, true},
		{"empty agent name", "", []string{"*"}, true},
		{"asterisk only pattern", "anything", []string{"*"}, true},
		{"question mark wildcard", "agent1", []string{"agent?"}, true},
		{"question mark no match", "agent12", []string{"agent?"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAgentDisabled(tt.agent, tt.patterns)
			if got != tt.want {
				t.Errorf("IsAgentDisabled(%q, %v) = %v, want %v", tt.agent, tt.patterns, got, tt.want)
			}
		})
	}
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		input   string
		want    bool
	}{
		// Basic matching
		{"exact match", "foo", "foo", true},
		{"exact no match", "foo", "bar", false},

		// Wildcard *
		{"star prefix", "foo*", "foobar", true},
		{"star suffix", "*bar", "foobar", true},
		{"star middle", "foo*bar", "fooxyzbar", true},
		{"star only", "*", "anything", true},
		{"star empty match", "foo*", "foo", true},

		// Wildcard ?
		{"question mark", "fo?", "foo", true},
		{"question mark no match", "fo?", "fooo", false},
		{"multiple question marks", "f??", "foo", true},

		// Character ranges
		{"char range", "[abc]", "a", true},
		{"char range no match", "[abc]", "d", false},
		{"char range with wildcard", "[a-z]*", "hello", true},

		// Malformed patterns (should return false, not panic)
		{"malformed bracket", "[", "a", false},
		{"malformed range", "[z-a]", "b", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchPattern(tt.pattern, tt.input)
			if got != tt.want {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.pattern, tt.input, got, tt.want)
			}
		})
	}
}
