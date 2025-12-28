package domain

// Issue represents a GitHub issue.
// Fields are ordered to minimize memory padding.
type Issue struct {
	Title  string
	Body   string
	Labels []string
	Number int
}
