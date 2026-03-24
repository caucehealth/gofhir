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
