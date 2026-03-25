// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"fmt"
	"html"
	"strings"

	dt "github.com/caucehealth/gofhir/r4/datatypes"
)

// GenerateNarrative creates a simple HTML narrative (text.div) for a resource
// based on its key fields. The narrative status is set to "generated".
// Supports Patient, Observation, Condition, Encounter, Practitioner,
// DiagnosticReport, and MedicationRequest.
func GenerateNarrative(resource Resource) *dt.Narrative {
	var rows []string

	switch r := resource.(type) {
	case *Patient:
		rows = patientNarrative(r)
	case *Observation:
		rows = observationNarrative(r)
	case *Condition:
		rows = conditionNarrative(r)
	case *Encounter:
		rows = encounterNarrative(r)
	case *Practitioner:
		rows = practitionerNarrative(r)
	case *DiagnosticReport:
		rows = diagnosticReportNarrative(r)
	case *MedicationRequest:
		rows = medicationRequestNarrative(r)
	default:
		return nil
	}

	if len(rows) == 0 {
		return nil
	}

	status := "generated"
	div := buildDiv(rows)
	return &dt.Narrative{Status: &status, Div: div}
}

func patientNarrative(p *Patient) []string {
	var rows []string
	if len(p.Name) > 0 {
		name := formatHumanName(p.Name[0])
		if name != "" {
			rows = append(rows, row("Name", name))
		}
	}
	if p.Gender != nil {
		rows = append(rows, row("Gender", string(*p.Gender)))
	}
	if p.BirthDate != nil {
		rows = append(rows, row("Birth Date", string(*p.BirthDate)))
	}
	if len(p.Identifier) > 0 {
		rows = append(rows, row("Identifier", formatIdentifier(p.Identifier[0])))
	}
	return rows
}

func observationNarrative(o *Observation) []string {
	var rows []string
	rows = append(rows, row("Code", formatCodeableConcept(o.Code)))
	if o.Status != nil {
		rows = append(rows, row("Status", string(*o.Status)))
	}
	if o.Value != nil {
		if o.Value.Quantity != nil {
			rows = append(rows, row("Value", formatQuantity(*o.Value.Quantity)))
		} else if o.Value.String != nil {
			rows = append(rows, row("Value", *o.Value.String))
		} else if o.Value.CodeableConcept != nil {
			rows = append(rows, row("Value", formatCodeableConcept(*o.Value.CodeableConcept)))
		}
	}
	if o.Subject != nil && o.Subject.Reference != nil {
		rows = append(rows, row("Subject", *o.Subject.Reference))
	}
	return rows
}

func conditionNarrative(c *Condition) []string {
	var rows []string
	if c.Code != nil {
		rows = append(rows, row("Code", formatCodeableConcept(*c.Code)))
	}
	if c.Subject.Reference != nil {
		rows = append(rows, row("Subject", *c.Subject.Reference))
	}
	if c.Severity != nil {
		rows = append(rows, row("Severity", formatCodeableConcept(*c.Severity)))
	}
	return rows
}

func encounterNarrative(e *Encounter) []string {
	var rows []string
	if e.Status != nil {
		rows = append(rows, row("Status", string(*e.Status)))
	}
	if e.Subject != nil && e.Subject.Reference != nil {
		rows = append(rows, row("Subject", *e.Subject.Reference))
	}
	return rows
}

func practitionerNarrative(p *Practitioner) []string {
	var rows []string
	if len(p.Name) > 0 {
		name := formatHumanName(p.Name[0])
		if name != "" {
			rows = append(rows, row("Name", name))
		}
	}
	return rows
}

func diagnosticReportNarrative(d *DiagnosticReport) []string {
	var rows []string
	rows = append(rows, row("Code", formatCodeableConcept(d.Code)))
	if d.Status != nil {
		rows = append(rows, row("Status", string(*d.Status)))
	}
	if d.Subject != nil && d.Subject.Reference != nil {
		rows = append(rows, row("Subject", *d.Subject.Reference))
	}
	return rows
}

func medicationRequestNarrative(m *MedicationRequest) []string {
	var rows []string
	if m.Status != nil {
		rows = append(rows, row("Status", string(*m.Status)))
	}
	if m.Intent != nil {
		rows = append(rows, row("Intent", string(*m.Intent)))
	}
	if m.Medication != nil && m.Medication.CodeableConcept != nil {
		rows = append(rows, row("Medication", formatCodeableConcept(*m.Medication.CodeableConcept)))
	}
	return rows
}

func formatHumanName(n dt.HumanName) string {
	var parts []string
	if n.Family != nil {
		parts = append(parts, *n.Family)
	}
	parts = append(parts, n.Given...)
	return strings.Join(parts, ", ")
}

func formatCodeableConcept(cc dt.CodeableConcept) string {
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
}

func formatQuantity(q dt.Quantity) string {
	if q.Value != nil && q.Unit != nil {
		return fmt.Sprintf("%v %s", *q.Value, *q.Unit)
	}
	if q.Value != nil {
		return fmt.Sprintf("%v", *q.Value)
	}
	return ""
}

func formatIdentifier(id dt.Identifier) string {
	if id.Value != nil {
		return *id.Value
	}
	return ""
}

func row(label, value string) string {
	return fmt.Sprintf("<tr><td>%s</td><td>%s</td></tr>",
		html.EscapeString(label), html.EscapeString(value))
}

func buildDiv(rows []string) string {
	var b strings.Builder
	b.WriteString(`<div xmlns="http://www.w3.org/1999/xhtml">`)
	b.WriteString("<table><tbody>")
	for _, r := range rows {
		b.WriteString(r)
	}
	b.WriteString("</tbody></table>")
	b.WriteString("</div>")
	return b.String()
}
