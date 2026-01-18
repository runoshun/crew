package domain

import (
	"bytes"
	_ "embed"
	"text/template"
)

//go:embed workflow_manager.md
var workflowManagerTmpl string

//go:embed workflow_worker.md
var workflowWorkerTmpl string

//go:embed workflow_manager_onboarding.md
var workflowManagerOnboardingTmpl string

//go:embed workflow_manager_auto.md
var workflowManagerAutoTmpl string

// WorkerInfo holds information about a worker for help display.
type WorkerInfo struct {
	Name        string
	Model       string
	Description string
}

// HelpData holds data for rendering help templates.
type HelpData struct {
	Workers        []WorkerInfo
	OnboardingDone bool
}

// RenderManagerHelp renders the manager help with worker information.
func RenderManagerHelp(data HelpData) (string, error) {
	return renderTemplate(workflowManagerTmpl, data)
}

// RenderWorkerHelp renders the worker help.
func RenderWorkerHelp() (string, error) {
	return renderTemplate(workflowWorkerTmpl, nil)
}

// RenderManagerOnboardingHelp renders the manager onboarding help.
func RenderManagerOnboardingHelp() (string, error) {
	return renderTemplate(workflowManagerOnboardingTmpl, nil)
}

// RenderManagerAutoHelp renders the manager auto mode help.
func RenderManagerAutoHelp() (string, error) {
	return renderTemplate(workflowManagerAutoTmpl, nil)
}

func renderTemplate(tmplStr string, data any) (string, error) {
	tmpl, err := template.New("help").Parse(tmplStr)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
