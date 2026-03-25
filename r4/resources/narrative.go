// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"sync"

	dt "github.com/caucehealth/gofhir/r4/datatypes"
)

// NarrativeGenerator creates XHTML narratives for FHIR resources using
// Go html/template. It ships with default templates for common resources
// and supports registering custom templates per resource type.
type NarrativeGenerator struct {
	mu        sync.RWMutex
	templates map[string]*template.Template
}

// DefaultNarrativeGenerator is the package-level generator with built-in templates.
var DefaultNarrativeGenerator = newDefaultGenerator()

// GenerateNarrative creates a simple HTML narrative for a resource using
// the default generator. Returns nil for unsupported resource types.
func GenerateNarrative(resource Resource) *dt.Narrative {
	return DefaultNarrativeGenerator.Generate(resource)
}

// NewNarrativeGenerator creates a generator with the default templates.
func NewNarrativeGenerator() *NarrativeGenerator {
	return newDefaultGenerator()
}

// RegisterTemplate adds or replaces a template for a resource type.
// The template receives the resource struct as its data context.
// It must produce table rows (<tr>...</tr>) for the narrative table.
func (g *NarrativeGenerator) RegisterTemplate(resourceType string, tmpl string) error {
	t, err := template.New(resourceType).Funcs(narrativeFuncs).Parse(tmpl)
	if err != nil {
		return fmt.Errorf("narrative template %s: %w", resourceType, err)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.templates[resourceType] = t
	return nil
}

// Generate creates an XHTML narrative for a resource.
func (g *NarrativeGenerator) Generate(resource Resource) *dt.Narrative {
	rt := resource.GetResourceType()

	g.mu.RLock()
	tmpl, ok := g.templates[rt]
	g.mu.RUnlock()

	if !ok {
		return nil
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, resource); err != nil {
		return nil
	}

	rows := buf.String()
	if strings.TrimSpace(rows) == "" {
		return nil
	}

	status := "generated"
	div := `<div xmlns="http://www.w3.org/1999/xhtml"><table><tbody>` + rows + `</tbody></table></div>`
	return &dt.Narrative{Status: &status, Div: div}
}

// Template helper functions available to all narrative templates.
var narrativeFuncs = template.FuncMap{
	"humanName": func(n dt.HumanName) string {
		var parts []string
		if n.Family != nil {
			parts = append(parts, *n.Family)
		}
		parts = append(parts, n.Given...)
		return strings.Join(parts, ", ")
	},
	"codeableConcept": func(cc dt.CodeableConcept) string {
		if cc.Text != nil {
			return *cc.Text
		}
		if len(cc.Coding) > 0 && cc.Coding[0].Display != nil {
			return *cc.Coding[0].Display
		}
		if len(cc.Coding) > 0 && cc.Coding[0].Code != nil {
			return string(*cc.Coding[0].Code)
		}
		return ""
	},
	"quantity": func(q dt.Quantity) string {
		if q.Value != nil && q.Unit != nil {
			return fmt.Sprintf("%s %s", q.Value.String(), *q.Unit)
		}
		if q.Value != nil {
			return q.Value.String()
		}
		return ""
	},
	"identifier": func(id dt.Identifier) string {
		if id.Value != nil {
			return *id.Value
		}
		return ""
	},
	"deref": func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	},
}

func newDefaultGenerator() *NarrativeGenerator {
	g := &NarrativeGenerator{templates: make(map[string]*template.Template)}

	// Register default templates for common resource types
	defaults := map[string]string{
		"Patient": `
{{- range .Name}}<tr><td>Name</td><td>{{humanName .}}</td></tr>{{end -}}
{{- if .Gender}}<tr><td>Gender</td><td>{{.GetGender}}</td></tr>{{end -}}
{{- if .BirthDate}}<tr><td>Birth Date</td><td>{{.GetBirthDate}}</td></tr>{{end -}}
{{- range .Identifier}}<tr><td>Identifier</td><td>{{identifier .}}</td></tr>{{end -}}
`,
		"Observation": `
{{- if .Code}}<tr><td>Code</td><td>{{codeableConcept .Code}}</td></tr>{{end -}}
{{- if .Status}}<tr><td>Status</td><td>{{.GetStatus}}</td></tr>{{end -}}
{{- if .Value}}{{if .Value.Quantity}}<tr><td>Value</td><td>{{quantity .Value.Quantity}}</td></tr>{{end -}}
{{- if .Value.String}}<tr><td>Value</td><td>{{deref .Value.String}}</td></tr>{{end -}}
{{- if .Value.CodeableConcept}}<tr><td>Value</td><td>{{codeableConcept .Value.CodeableConcept}}</td></tr>{{end}}{{end -}}
{{- if .Subject}}{{if .Subject.Reference}}<tr><td>Subject</td><td>{{deref .Subject.Reference}}</td></tr>{{end}}{{end -}}
`,
		"Condition": `
{{- if .Code}}<tr><td>Code</td><td>{{codeableConcept .Code}}</td></tr>{{end -}}
{{- if .Subject}}{{if .Subject.Reference}}<tr><td>Subject</td><td>{{deref .Subject.Reference}}</td></tr>{{end}}{{end -}}
{{- if .Severity}}<tr><td>Severity</td><td>{{codeableConcept .Severity}}</td></tr>{{end -}}
`,
		"Encounter": `
{{- if .Status}}<tr><td>Status</td><td>{{.GetStatus}}</td></tr>{{end -}}
{{- if .Subject}}{{if .Subject.Reference}}<tr><td>Subject</td><td>{{deref .Subject.Reference}}</td></tr>{{end}}{{end -}}
`,
		"Practitioner": `
{{- range .Name}}<tr><td>Name</td><td>{{humanName .}}</td></tr>{{end -}}
`,
		"DiagnosticReport": `
{{- if .Code}}<tr><td>Code</td><td>{{codeableConcept .Code}}</td></tr>{{end -}}
{{- if .Status}}<tr><td>Status</td><td>{{.GetStatus}}</td></tr>{{end -}}
{{- if .Subject}}{{if .Subject.Reference}}<tr><td>Subject</td><td>{{deref .Subject.Reference}}</td></tr>{{end}}{{end -}}
`,
		"MedicationRequest": `
{{- if .Status}}<tr><td>Status</td><td>{{.GetStatus}}</td></tr>{{end -}}
{{- if .Intent}}<tr><td>Intent</td><td>{{.GetIntent}}</td></tr>{{end -}}
{{- if .Medication}}{{if .Medication.CodeableConcept}}<tr><td>Medication</td><td>{{codeableConcept .Medication.CodeableConcept}}</td></tr>{{end}}{{end -}}
`,
		"Organization": `
{{- if .Name}}<tr><td>Name</td><td>{{deref .Name}}</td></tr>{{end -}}
{{- range .Identifier}}<tr><td>Identifier</td><td>{{identifier .}}</td></tr>{{end -}}
`,
		"Location": `
{{- if .Name}}<tr><td>Name</td><td>{{deref .Name}}</td></tr>{{end -}}
{{- if .Status}}<tr><td>Status</td><td>{{.GetStatus}}</td></tr>{{end -}}
{{- if .Address}}<tr><td>City</td><td>{{deref .Address.City}}</td></tr>{{end -}}
`,
		"Procedure": `
{{- if .Code}}<tr><td>Code</td><td>{{codeableConcept .Code}}</td></tr>{{end -}}
{{- if .Status}}<tr><td>Status</td><td>{{.GetStatus}}</td></tr>{{end -}}
{{- if .Subject}}{{if .Subject.Reference}}<tr><td>Subject</td><td>{{deref .Subject.Reference}}</td></tr>{{end}}{{end -}}
`,
		"Immunization": `
{{- if .VaccineCode}}<tr><td>Vaccine</td><td>{{codeableConcept .VaccineCode}}</td></tr>{{end -}}
{{- if .Status}}<tr><td>Status</td><td>{{.GetStatus}}</td></tr>{{end -}}
{{- if .Patient}}{{if .Patient.Reference}}<tr><td>Patient</td><td>{{deref .Patient.Reference}}</td></tr>{{end}}{{end -}}
`,
		"AllergyIntolerance": `
{{- if .Code}}<tr><td>Code</td><td>{{codeableConcept .Code}}</td></tr>{{end -}}
{{- if .Patient}}{{if .Patient.Reference}}<tr><td>Patient</td><td>{{deref .Patient.Reference}}</td></tr>{{end}}{{end -}}
`,
		"CarePlan": `
{{- if .Status}}<tr><td>Status</td><td>{{.GetStatus}}</td></tr>{{end -}}
{{- if .Intent}}<tr><td>Intent</td><td>{{.GetIntent}}</td></tr>{{end -}}
{{- if .Subject}}{{if .Subject.Reference}}<tr><td>Subject</td><td>{{deref .Subject.Reference}}</td></tr>{{end}}{{end -}}
`,
	}

	for rt, tmpl := range defaults {
		g.RegisterTemplate(rt, tmpl)
	}
	return g
}
