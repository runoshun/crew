package shared

import "github.com/runoshun/git-crew/v2/internal/domain"

// ResolveACPNamespace resolves namespace for ACP IPC paths.
func ResolveACPNamespace(cfg *domain.Config, git domain.Git) string {
	if cfg != nil && cfg.Tasks.Namespace != "" {
		namespace := domain.SanitizeNamespace(cfg.Tasks.Namespace)
		if namespace != "" {
			return namespace
		}
	}
	if git != nil {
		email, err := git.UserEmail()
		if err == nil {
			namespace := domain.NamespaceFromEmail(email)
			if namespace != "" {
				return namespace
			}
		}
	}
	return "default"
}
