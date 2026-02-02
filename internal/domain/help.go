package domain

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"text/template"
)

//go:embed workflow_manager.md
var workflowManagerTmpl string

//go:embed workflow_worker.md
var workflowWorkerTmpl string

//go:embed workflow_reviewer.md
var workflowReviewerTmpl string

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
func RenderManagerHelp(cfg *Config, data HelpData) (string, []string, error) {
	if cfg == nil {
		cfg = &Config{}
	}
	return renderHelpTemplate(helpTemplateConfig{
		inline:  cfg.Help.Manager,
		file:    cfg.Help.ManagerFile,
		name:    "help.manager",
		builtin: workflowManagerTmpl,
	}, data)
}

// RenderWorkerHelp renders the worker help.
func RenderWorkerHelp(cfg *Config) (string, []string, error) {
	if cfg == nil {
		cfg = &Config{}
	}
	return renderHelpTemplate(helpTemplateConfig{
		inline:  cfg.Help.Worker,
		file:    cfg.Help.WorkerFile,
		name:    "help.worker",
		builtin: workflowWorkerTmpl,
	}, nil)
}

// RenderReviewerHelp renders the reviewer help.
func RenderReviewerHelp(cfg *Config, followUp bool) (string, []string, error) {
	return renderHelpTemplate(helpTemplateConfig{
		name:    "help.reviewer",
		builtin: workflowReviewerTmpl,
	}, CommandData{IsFollowUp: followUp})
}

// RenderManagerOnboardingHelp renders the manager onboarding help.
func RenderManagerOnboardingHelp(cfg *Config) (string, []string, error) {
	if cfg == nil {
		cfg = &Config{}
	}
	return renderHelpTemplate(helpTemplateConfig{
		inline:  cfg.Help.ManagerOnboarding,
		file:    cfg.Help.ManagerOnboardingFile,
		name:    "help.manager_onboarding",
		builtin: workflowManagerOnboardingTmpl,
	}, nil)
}

// RenderManagerAutoHelp renders the manager auto mode help.
func RenderManagerAutoHelp(cfg *Config) (string, []string, error) {
	if cfg == nil {
		cfg = &Config{}
	}
	return renderHelpTemplate(helpTemplateConfig{
		inline:  cfg.Help.ManagerAuto,
		file:    cfg.Help.ManagerAutoFile,
		name:    "help.manager_auto",
		builtin: workflowManagerAutoTmpl,
	}, nil)
}

type helpTemplateConfig struct {
	inline  string
	file    string
	name    string
	builtin string
}

func renderHelpTemplate(cfgTemplate helpTemplateConfig, data any) (string, []string, error) {
	warnings := []string{}
	selectedTemplate := cfgTemplate.builtin
	sourceName := "builtin"

	if cfgTemplate.file != "" {
		content, err := os.ReadFile(cfgTemplate.file)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("failed to read %s_file: %v; falling back to builtin", cfgTemplate.name, err))
		} else {
			selectedTemplate = string(content)
			sourceName = cfgTemplate.name + "_file"
		}
	} else if cfgTemplate.inline != "" {
		selectedTemplate = cfgTemplate.inline
		sourceName = cfgTemplate.name
	}

	content, err := renderTemplate(selectedTemplate, data)
	if err == nil {
		return content, warnings, nil
	}

	if sourceName == "builtin" {
		return "", warnings, err
	}

	warnings = append(warnings, fmt.Sprintf("failed to render %s: %v; falling back to builtin", sourceName, err))
	content, err = renderTemplate(cfgTemplate.builtin, data)
	return content, warnings, err
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
