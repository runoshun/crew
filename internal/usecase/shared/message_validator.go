package shared

import (
	"strings"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// ValidateMessage trims whitespace from the message and validates it is not empty.
// Returns the trimmed message if valid, otherwise returns domain.ErrEmptyMessage.
// This centralizes the common pattern of:
//
//	message := strings.TrimSpace(in.Message)
//	if message == "" {
//	    return nil, domain.ErrEmptyMessage
//	}
func ValidateMessage(message string) (string, error) {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return "", domain.ErrEmptyMessage
	}
	return trimmed, nil
}
