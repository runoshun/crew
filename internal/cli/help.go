package cli

import (
	"bytes"
	_ "embed"
	"io"
	"text/template"
)

// Embedded help templates.
//
//go:embed help_worker.md.tmpl
var helpWorkerTmpl string

//go:embed help_manager.md.tmpl
var helpManagerTmpl string

// HelpTemplateData holds data for rendering help templates.
// Currently empty as templates don't use template variables,
// but kept for future extensibility.
type HelpTemplateData struct{}

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
