// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/caucehealth/gofhir/r4/resources"
)

func TestFHIRErrorString(t *testing.T) {
	err := resources.ErrNotFound("Patient", "123")
	if !strings.Contains(err.Error(), "Patient/123 not found") {
		t.Errorf("error = %q, want 'Patient/123 not found'", err.Error())
	}
	if err.StatusCode != 404 {
		t.Errorf("status = %d, want 404", err.StatusCode)
	}
}

func TestFHIRErrorOutcomeJSON(t *testing.T) {
	err := resources.ErrValidation("Patient.name is required")
	data, jsonErr := json.Marshal(err.Outcome)
	if jsonErr != nil {
		t.Fatal(jsonErr)
	}
	s := string(data)
	if !strings.Contains(s, "OperationOutcome") {
		t.Error("should contain resourceType")
	}
	if !strings.Contains(s, "Patient.name is required") {
		t.Error("should contain diagnostics")
	}
	if !strings.Contains(s, "error") {
		t.Error("should contain severity")
	}
}

func TestOutcomeBuilder(t *testing.T) {
	oo := resources.NewOutcome().
		WithIssue(resources.SeverityError, resources.IssueCodeRequired, "name is required").
		WithIssueAt(resources.SeverityWarning, resources.IssueCodeValue, "birthDate precision low", "Patient.birthDate").
		Build()

	if len(oo.Issue) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(oo.Issue))
	}
	if *oo.Issue[0].Severity != resources.SeverityError {
		t.Error("first issue should be error")
	}
	if *oo.Issue[1].Severity != resources.SeverityWarning {
		t.Error("second issue should be warning")
	}
	if len(oo.Issue[1].Expression) != 1 || oo.Issue[1].Expression[0] != "Patient.birthDate" {
		t.Error("second issue should have expression")
	}
}

func TestOutcomeHasErrors(t *testing.T) {
	oo := resources.NewOutcome().
		WithIssue(resources.SeverityWarning, resources.IssueCodeValue, "just a warning").
		Build()
	if oo.HasErrors() {
		t.Error("warnings-only outcome should not have errors")
	}

	oo2 := resources.NewOutcome().
		WithIssue(resources.SeverityError, resources.IssueCodeRequired, "missing field").
		Build()
	if !oo2.HasErrors() {
		t.Error("error outcome should have errors")
	}
}

func TestErrInvalidResource(t *testing.T) {
	err := resources.ErrInvalidResource("malformed JSON")
	if err.StatusCode != 400 {
		t.Errorf("status = %d, want 400", err.StatusCode)
	}
	if !strings.Contains(err.Error(), "malformed JSON") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestFHIRErrorWithNoOutcome(t *testing.T) {
	err := &resources.FHIRError{StatusCode: 500}
	if !strings.Contains(err.Error(), "500") {
		t.Error("should mention status code")
	}
}
