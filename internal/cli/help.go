package cli

import (
	"fmt"
	"io"
	"sort"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

func buildHelpData(cfg *domain.Config) domain.HelpData {
	if cfg == nil {
		cfg = &domain.Config{}
	}
	enabledAgents := cfg.EnabledAgents()
	workers := make([]domain.WorkerInfo, 0, len(enabledAgents))
	for name, agent := range enabledAgents {
		if agent.Hidden || (agent.Role != "" && agent.Role != domain.RoleWorker) {
			continue
		}
		workers = append(workers, domain.WorkerInfo{
			Name:        name,
			Model:       agent.DefaultModel,
			Description: agent.Description,
		})
	}
	sort.Slice(workers, func(i, j int) bool {
		return workers[i].Name < workers[j].Name
	})
	return domain.HelpData{
		Workers:        workers,
		OnboardingDone: cfg.OnboardingDone,
	}
}

func showManagerHelp(w io.Writer, errW io.Writer, cfg *domain.Config) error {
	help, warnings, err := domain.RenderManagerHelp(cfg, buildHelpData(cfg))
	if err != nil {
		return err
	}
	writeWarnings(errW, warnings)
	_, err = fmt.Fprint(w, help)
	return err
}

func showWorkerHelp(w io.Writer, errW io.Writer, cfg *domain.Config) error {
	help, warnings, err := domain.RenderWorkerHelp(cfg)
	if err != nil {
		return err
	}
	writeWarnings(errW, warnings)
	_, err = fmt.Fprint(w, help)
	return err
}

func showReviewerHelp(w io.Writer, errW io.Writer, cfg *domain.Config, followUp bool) error {
	help, warnings, err := domain.RenderReviewerHelp(cfg, followUp)
	if err != nil {
		return err
	}
	writeWarnings(errW, warnings)
	_, err = fmt.Fprint(w, help)
	return err
}

func showManagerOnboardingHelp(w io.Writer, errW io.Writer, cfg *domain.Config) error {
	help, warnings, err := domain.RenderManagerOnboardingHelp(cfg)
	if err != nil {
		return err
	}
	writeWarnings(errW, warnings)
	_, err = fmt.Fprint(w, help)
	return err
}

func showManagerAutoHelp(w io.Writer, errW io.Writer, cfg *domain.Config) error {
	help, warnings, err := domain.RenderManagerAutoHelp(cfg)
	if err != nil {
		return err
	}
	writeWarnings(errW, warnings)
	_, err = fmt.Fprint(w, help)
	return err
}

func writeWarnings(w io.Writer, warnings []string) {
	for _, warning := range warnings {
		_, _ = fmt.Fprintf(w, "Warning: %s\n", warning)
	}
}
