package cli

import (
	"fmt"
	"io"
	"sort"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

func buildHelpData(cfg *domain.Config) domain.HelpData {
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
	return domain.HelpData{Workers: workers}
}

func showManagerHelp(w io.Writer, cfg *domain.Config) error {
	help, err := domain.RenderManagerHelp(buildHelpData(cfg))
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, help)
	return err
}

func showWorkerHelp(w io.Writer) error {
	help, err := domain.RenderWorkerHelp()
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, help)
	return err
}
