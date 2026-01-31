package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/runoshun/git-crew/v2/internal/app"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/usecase"
	"github.com/spf13/cobra"
)

const (
	legacyStoreAuto = "auto"
	legacyStoreGit  = "git"
	legacyStoreJSON = "json"
)

// newMigrateCommand creates the migrate command.
func newMigrateCommand(c *app.Container) *cobra.Command {
	var opts struct {
		From            string
		SourceNamespace string
		Namespace       string
		JSONPath        string
		SkipComments    bool
		StrictComments  bool
	}

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate legacy task data to file store",
		Long: `Migrate legacy task data (git/json store) into the file store.

The legacy store is detected automatically unless --from is specified.
The destination namespace follows the standard resolution rules unless
--namespace is provided.

Examples:
  # Auto-detect legacy store and migrate
  crew migrate

  # Migrate from legacy JSON store
  crew migrate --from json

  # Migrate from legacy git store into a specific namespace
  crew migrate --from git --namespace work

  # Migrate from a specific tasks.json path
  crew migrate --from json --json-path ./tasks.json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			from := strings.ToLower(strings.TrimSpace(opts.From))
			if from == "" {
				from = legacyStoreAuto
			}

			namespace, err := resolveMigrationNamespace(c, opts.Namespace)
			if err != nil {
				return err
			}

			sourceNamespace := ""
			if from == legacyStoreAuto || from == legacyStoreGit {
				sourceNamespace, err = resolveSourceNamespace(c, opts.SourceNamespace)
				if err != nil {
					return err
				}
			}

			source, sourceKind, err := resolveLegacyStore(c, opts.From, opts.JSONPath, sourceNamespace)
			if err != nil {
				return err
			}

			dest, destInit := c.FileStore(namespace)
			uc := c.MigrateStoreUseCase(source, dest, destInit)
			out, err := uc.Execute(cmd.Context(), usecase.MigrateStoreInput{SkipComments: opts.SkipComments, StrictComments: opts.StrictComments})
			if err != nil {
				return err
			}

			if out.Total == 0 {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No legacy tasks found in %s store\n", sourceKind)
				return nil
			}

			summary := fmt.Sprintf("Migrated %d task(s) from %s store to .crew/tasks/%s", out.Migrated, sourceKind, namespace)
			if out.Skipped > 0 {
				summary += fmt.Sprintf(" (skipped %d existing)", out.Skipped)
			}
			if out.SkippedComments > 0 {
				summary += fmt.Sprintf(" (skipped comments for %d task(s))", out.SkippedComments)
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), summary)
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.From, "from", legacyStoreAuto, "Legacy store type: auto, git, json")
	cmd.Flags().StringVar(&opts.SourceNamespace, "source-namespace", "", "Legacy gitstore namespace (default: auto-detect)")
	cmd.Flags().StringVar(&opts.Namespace, "namespace", "", "Destination namespace (default: resolved from config/email)")
	cmd.Flags().StringVar(&opts.JSONPath, "json-path", "", "Path to legacy tasks.json (for json store)")
	cmd.Flags().BoolVar(&opts.SkipComments, "skip-comments", false, "Migrate tasks without comments")
	cmd.Flags().BoolVar(&opts.StrictComments, "strict-comments", false, "Fail migration if legacy comments cannot be read")

	return cmd
}

func resolveMigrationNamespace(c *app.Container, override string) (string, error) {
	if override != "" {
		namespace := domain.SanitizeNamespace(override)
		if namespace == "" {
			return "", domain.ErrInvalidNamespace
		}
		return namespace, nil
	}

	if c == nil || c.ConfigLoader == nil {
		return "", domain.ErrConfigNil
	}
	cfg, err := c.ConfigLoader.Load()
	if err != nil {
		return "", err
	}
	if cfg != nil && cfg.Tasks.Namespace != "" {
		namespace := domain.SanitizeNamespace(cfg.Tasks.Namespace)
		if namespace != "" {
			return namespace, nil
		}
	}

	if c.Git != nil {
		email, err := c.Git.UserEmail()
		if err == nil {
			namespace := domain.NamespaceFromEmail(email)
			if namespace != "" {
				return namespace, nil
			}
		}
	}

	return "default", nil
}

func resolveSourceNamespace(c *app.Container, override string) (string, error) {
	if override != "" {
		ns := domain.SanitizeNamespace(override)
		if ns == "" {
			return "", domain.ErrInvalidNamespace
		}
		return ns, nil
	}

	// Auto-detect legacy gitstore namespace.
	// Historical default was "crew"; also try current namespace resolution.
	candidates := []string{"crew"}
	if c != nil && c.ConfigLoader != nil {
		cfg, err := c.ConfigLoader.Load()
		if err == nil && cfg != nil {
			ns := domain.SanitizeNamespace(cfg.Tasks.Namespace)
			if ns != "" && ns != "crew" {
				candidates = append(candidates, ns)
			}
		}
	}
	if c != nil && c.Git != nil {
		email, err := c.Git.UserEmail()
		if err == nil {
			ns := domain.NamespaceFromEmail(email)
			if ns != "" && ns != "crew" {
				candidates = append(candidates, ns)
			}
		}
	}

	var found []string
	for _, ns := range candidates {
		_, ok, err := loadGitStore(c, ns)
		if err != nil {
			return "", err
		}
		if ok {
			found = append(found, ns)
		}
	}
	if len(found) == 0 {
		return "", nil
	}
	if len(found) > 1 {
		return "", fmt.Errorf("multiple legacy gitstore namespaces found: %s (use --source-namespace)", strings.Join(found, ", "))
	}
	return found[0], nil
}

func resolveLegacyStore(c *app.Container, from, jsonPath, namespace string) (domain.TaskRepository, string, error) {
	if c == nil {
		return nil, "", errors.New("container is nil")
	}

	normalized := strings.ToLower(strings.TrimSpace(from))
	if normalized == "" {
		normalized = legacyStoreAuto
	}

	switch normalized {
	case legacyStoreAuto:
		jsonStore, jsonFound, err := detectJSONStore(c, jsonPath)
		if err != nil {
			return nil, "", err
		}
		gitStore, gitFound, err := detectGitStore(c, namespace)
		if err != nil {
			return nil, "", err
		}
		if jsonFound && gitFound {
			return nil, "", fmt.Errorf("both legacy stores found; specify --from")
		}
		if jsonFound {
			return jsonStore, legacyStoreJSON, nil
		}
		if gitFound {
			return gitStore, legacyStoreGit, nil
		}
		return nil, "", domain.ErrLegacyStoreNotFound
	case legacyStoreJSON:
		store, found, err := loadJSONStore(c, jsonPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, "", domain.ErrLegacyStoreNotFound
			}
			return nil, "", err
		}
		if !found {
			return nil, "", domain.ErrLegacyStoreNotFound
		}
		return store, legacyStoreJSON, nil
	case legacyStoreGit:
		store, found, err := loadGitStore(c, namespace)
		if err != nil {
			return nil, "", err
		}
		if !found {
			return nil, "", domain.ErrLegacyStoreNotFound
		}
		return store, legacyStoreGit, nil
	default:
		return nil, "", fmt.Errorf("invalid --from: %s (expected auto, git, json)", normalized)
	}
}

func detectJSONStore(c *app.Container, jsonPath string) (domain.TaskRepository, bool, error) {
	store, found, err := loadJSONStore(c, jsonPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return store, found, nil
}

func loadJSONStore(c *app.Container, jsonPath string) (domain.TaskRepository, bool, error) {
	path, err := resolveJSONStorePath(c, jsonPath)
	if err != nil {
		return nil, false, err
	}
	if path == "" {
		return nil, false, os.ErrNotExist
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, os.ErrNotExist
		}
		return nil, false, fmt.Errorf("stat legacy json store: %w", err)
	}
	store, storeInit := c.JSONStore(path)
	if !storeInit.IsInitialized() {
		return store, false, nil
	}
	return store, true, nil
}

func resolveJSONStorePath(c *app.Container, override string) (string, error) {
	if override != "" {
		return override, nil
	}
	if c == nil {
		return "", errors.New("container is nil")
	}
	candidates := []string{
		filepath.Join(c.Config.CrewDir, "tasks.json"),
		filepath.Join(c.Config.GitDir, "crew", "tasks.json"),
	}
	found := ""
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return "", fmt.Errorf("stat legacy json store: %w", err)
		}
		if found != "" {
			return "", fmt.Errorf("multiple legacy json stores found; use --json-path")
		}
		found = candidate
	}
	return found, nil
}

func detectGitStore(c *app.Container, namespace string) (domain.TaskRepository, bool, error) {
	store, found, err := loadGitStore(c, namespace)
	if err != nil {
		return nil, false, err
	}
	return store, found, nil
}

func loadGitStore(c *app.Container, namespace string) (domain.TaskRepository, bool, error) {
	if c == nil {
		return nil, false, errors.New("container is nil")
	}
	store, storeInit, err := c.GitStore(namespace)
	if err != nil {
		return nil, false, err
	}
	if storeInit.IsInitialized() {
		return store, true, nil
	}
	tasks, err := store.List(domain.TaskFilter{})
	if err != nil {
		return nil, false, err
	}
	return store, len(tasks) > 0, nil
}
