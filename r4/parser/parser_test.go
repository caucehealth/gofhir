// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package parser_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/caucehealth/gofhir/r4/parser"
	"github.com/caucehealth/gofhir/r4/resources"
)

func buildTestPatient(t *testing.T) *resources.Patient {
	t.Helper()
	p, err := resources.NewPatient().
		WithName("John", "Doe").
		WithGender(resources.AdministrativeGenderMale).
		WithBirthDate("1980-01-01").
		Build()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPrettyPrint(t *testing.T) {
	p := buildTestPatient(t)
	out, err := parser.Marshal(p, parser.Options{PrettyPrint: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "\n") {
		t.Error("pretty print should contain newlines")
	}
	if !strings.Contains(string(out), "  ") {
		t.Error("pretty print should contain indentation")
	}
}

func TestCompactPrint(t *testing.T) {
	p := buildTestPatient(t)
	out, err := parser.Marshal(p, parser.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "\n") {
		t.Error("compact should not contain newlines")
	}
}

func TestSuppressNarrative(t *testing.T) {
	// Build a patient with text
	input := `{"resourceType":"Patient","id":"1","text":{"status":"generated","div":"<div>test</div>"},"gender":"male"}`
	var p resources.Patient
	json.Unmarshal([]byte(input), &p)

	out, err := parser.Marshal(&p, parser.Options{SuppressNarrative: true})
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]json.RawMessage
	json.Unmarshal(out, &m)
	if _, ok := m["text"]; ok {
		t.Error("text should be suppressed")
	}
	if _, ok := m["gender"]; !ok {
		t.Error("gender should still be present")
	}
}

func TestSummaryMode(t *testing.T) {
	input := `{
		"resourceType":"Patient","id":"1",
		"meta":{"versionId":"1"},
		"text":{"status":"generated","div":"<div>long narrative</div>"},
		"contained":[{"resourceType":"Organization","id":"org1"}],
		"extension":[{"url":"http://example.org","valueString":"x"}],
		"gender":"male","birthDate":"1980-01-01"
	}`
	var p resources.Patient
	json.Unmarshal([]byte(input), &p)

	out, err := parser.Marshal(&p, parser.Options{SummaryMode: true})
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]json.RawMessage
	json.Unmarshal(out, &m)

	// Summary should exclude text, contained, extension
	for _, key := range []string{"text", "contained", "extension"} {
		if _, ok := m[key]; ok {
			t.Errorf("%s should be excluded in summary mode", key)
		}
	}
	// But keep id, meta, gender
	for _, key := range []string{"id", "meta", "gender"} {
		if _, ok := m[key]; !ok {
			t.Errorf("%s should be present in summary mode", key)
		}
	}
}

func TestIncludeElements(t *testing.T) {
	p := buildTestPatient(t)
	out, err := parser.Marshal(p, parser.Options{
		IncludeElements: []string{"id", "gender"},
	})
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]json.RawMessage
	json.Unmarshal(out, &m)

	if _, ok := m["resourceType"]; !ok {
		t.Error("resourceType should always be included")
	}
	if _, ok := m["name"]; ok {
		t.Error("name should be excluded")
	}
}

func TestExcludeElements(t *testing.T) {
	p := buildTestPatient(t)
	out, err := parser.Marshal(p, parser.Options{
		ExcludeElements: []string{"birthDate", "name"},
	})
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]json.RawMessage
	json.Unmarshal(out, &m)

	if _, ok := m["birthDate"]; ok {
		t.Error("birthDate should be excluded")
	}
	if _, ok := m["gender"]; !ok {
		t.Error("gender should still be present")
	}
}

func TestUnmarshal(t *testing.T) {
	input := `{"resourceType":"Patient","id":"1","gender":"male"}`
	var p resources.Patient
	if err := parser.Unmarshal([]byte(input), &p); err != nil {
		t.Fatal(err)
	}
	if p.GetGender() != resources.AdministrativeGenderMale {
		t.Error("gender should be male")
	}
}

func TestStrictModeRejectsUnknownFields(t *testing.T) {
	input := `{"resourceType":"Patient","id":"1","gender":"male","unknownField":"value","anotherBad":123}`
	var p resources.Patient
	err := parser.UnmarshalWithOptions([]byte(input), &p, parser.Options{
		ErrorMode: parser.Strict,
	})
	if err == nil {
		t.Fatal("strict mode should reject unknown fields")
	}

	perrs, ok := err.(parser.ParseErrors)
	if !ok {
		t.Fatalf("expected ParseErrors, got %T: %v", err, err)
	}
	if len(perrs) != 2 {
		t.Errorf("expected 2 errors, got %d: %v", len(perrs), perrs)
	}
	for _, pe := range perrs {
		if pe.Type != parser.ErrorUnknownField {
			t.Errorf("expected unknown_field error, got %s", pe.Type)
		}
	}
}

func TestStrictModeAcceptsValidFields(t *testing.T) {
	input := `{"resourceType":"Patient","id":"1","gender":"male","birthDate":"1980-01-01"}`
	var p resources.Patient
	err := parser.UnmarshalWithOptions([]byte(input), &p, parser.Options{
		ErrorMode: parser.Strict,
	})
	if err != nil {
		t.Fatalf("strict mode should accept valid fields: %v", err)
	}
	if p.GetGender() != resources.AdministrativeGenderMale {
		t.Error("gender should be male")
	}
}

func TestLenientModeAcceptsUnknownFields(t *testing.T) {
	input := `{"resourceType":"Patient","id":"1","unknownField":"value"}`
	var p resources.Patient
	err := parser.UnmarshalWithOptions([]byte(input), &p, parser.Options{
		ErrorMode: parser.Lenient,
	})
	if err != nil {
		t.Fatalf("lenient mode should accept unknown fields: %v", err)
	}
	// Unknown field should be captured in Extra
	if len(p.GetExtra()) != 1 {
		t.Errorf("expected 1 extra field, got %d", len(p.GetExtra()))
	}
}

func TestStripVersionsFromReferences(t *testing.T) {
	input := `{"resourceType":"Observation","id":"1","status":"final","code":{"text":"test"},"subject":{"reference":"Patient/123/_history/2"}}`
	var obs resources.Observation
	json.Unmarshal([]byte(input), &obs)

	out, err := parser.Marshal(&obs, parser.Options{StripVersionsFromReferences: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "/_history/") {
		t.Error("version should be stripped from references")
	}
	if !strings.Contains(string(out), `"Patient/123"`) {
		t.Error("reference base should be preserved")
	}
}

func TestParseErrorString(t *testing.T) {
	pe := parser.ParseError{
		Type:         parser.ErrorUnknownField,
		Field:        "badField",
		ResourceType: "Patient",
		Message:      `unrecognized field "badField"`,
	}
	s := pe.Error()
	if !strings.Contains(s, "Patient.badField") {
		t.Errorf("error string should contain resource.field: %s", s)
	}
}

func TestOmitResourceId(t *testing.T) {
	p := buildTestPatient(t)
	id := resources.AdministrativeGenderMale
	_ = id
	out, err := parser.Marshal(p, parser.Options{OmitResourceId: true})
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]json.RawMessage
	json.Unmarshal(out, &m)
	if _, ok := m["id"]; ok {
		t.Error("id should be omitted")
	}
	if _, ok := m["resourceType"]; !ok {
		t.Error("resourceType should still be present")
	}
	if _, ok := m["gender"]; !ok {
		t.Error("gender should still be present")
	}
}

func TestOmitDefaults(t *testing.T) {
	input := `{"resourceType":"Patient","id":"1","active":false,"gender":"male"}`
	var p resources.Patient
	json.Unmarshal([]byte(input), &p)

	out, err := parser.Marshal(&p, parser.Options{OmitDefaults: true})
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]json.RawMessage
	json.Unmarshal(out, &m)
	if _, ok := m["active"]; ok {
		t.Error("active=false should be omitted with OmitDefaults")
	}
	if _, ok := m["gender"]; !ok {
		t.Error("gender should still be present")
	}
	if _, ok := m["id"]; !ok {
		t.Error("id should still be present")
	}
}

func TestOptionComposition(t *testing.T) {
	input := `{
		"resourceType":"Patient","id":"1",
		"meta":{"versionId":"1"},
		"text":{"status":"generated","div":"<div>text</div>"},
		"gender":"male","active":false
	}`
	var p resources.Patient
	json.Unmarshal([]byte(input), &p)

	// SummaryMode + OmitDefaults + OmitResourceId
	out, err := parser.Marshal(&p, parser.Options{
		SummaryMode:    true,
		OmitDefaults:   true,
		OmitResourceId: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	json.Unmarshal(out, &m)

	if _, ok := m["text"]; ok {
		t.Error("text should be removed by SummaryMode")
	}
	if _, ok := m["id"]; ok {
		t.Error("id should be removed by OmitResourceId")
	}
	if _, ok := m["active"]; ok {
		t.Error("active=false should be removed by OmitDefaults")
	}
	if _, ok := m["resourceType"]; !ok {
		t.Error("resourceType should always be present")
	}
	if _, ok := m["gender"]; !ok {
		t.Error("gender should survive all filters")
	}
}

func TestSummaryModePlusIncludeElements(t *testing.T) {
	input := `{"resourceType":"Patient","id":"1","gender":"male","birthDate":"1980-01-01","text":{"status":"generated","div":"<div>t</div>"}}`
	var p resources.Patient
	json.Unmarshal([]byte(input), &p)

	// SummaryMode removes text; IncludeElements restricts to id+gender
	out, err := parser.Marshal(&p, parser.Options{
		SummaryMode:     true,
		IncludeElements: []string{"id", "gender"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	json.Unmarshal(out, &m)

	if _, ok := m["birthDate"]; ok {
		t.Error("birthDate should be excluded by IncludeElements")
	}
	if _, ok := m["text"]; ok {
		t.Error("text should be excluded by SummaryMode")
	}
	if _, ok := m["gender"]; !ok {
		t.Error("gender should be included")
	}
}
