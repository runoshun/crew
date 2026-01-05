package cli

import (
	"bytes"
	_ "embed"
	"io"
	"sort"
	"text/template"

	"github.com/runoshun/git-crew/v2/internal/domain"
)

// Embedded help templates.
//
//go:embed help_worker.md.tmpl
var helpWorkerTmpl string

//go:embed help_manager.md.tmpl
var helpManagerTmpl string

// WorkerInfo holds information about a worker for help display.
type WorkerInfo struct {
	Name        string
	Model       string
	Description string
}

// HelpTemplateData holds data for rendering help templates.
type HelpTemplateData struct {
	Workers []WorkerInfo
}

// NewHelpTemplateData creates HelpTemplateData from config.
// Only shows agents with role=worker that are not hidden.
func NewHelpTemplateData(cfg *domain.Config) HelpTemplateData {
	if cfg == nil {
		return HelpTemplateData{}
	}

	workers := make([]WorkerInfo, 0, len(cfg.Agents))
	for name, agent := range cfg.Agents {
		// Skip hidden agents and non-worker roles
		if agent.Hidden || (agent.Role != "" && agent.Role != domain.RoleWorker) {
			continue
		}
		workers = append(workers, WorkerInfo{
			Name:        name,
			Model:       agent.DefaultModel,
			Description: agent.Description,
		})
	}

	sort.Slice(workers, func(i, j int) bool {
		return workers[i].Name < workers[j].Name
	})

	return HelpTemplateData{
		Workers: workers,
	}
}

// RenderWorkerHelp renders the worker help template.
func RenderWorkerHelp(w io.Writer, data HelpTemplateData) error {
	return renderHelpTemplate(w, helpWorkerTmpl, data)
}

// RenderManagerHelp renders the manager help template.
func RenderManagerHelp(w io.Writer, data HelpTemplateData) error {
	return renderHelpTemplate(w, helpManagerTmpl, data)
}

// renderHelpTemplate renders a help template with the given data.
func renderHelpTemplate(w io.Writer, tmplStr string, data HelpTemplateData) error {
	tmpl, err := template.New("help").Parse(tmplStr)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if execErr := tmpl.Execute(&buf, data); execErr != nil {
		return execErr
	}

	_, err = w.Write(buf.Bytes())
	return err
}
