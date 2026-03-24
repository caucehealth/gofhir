// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	"strings"
	"testing"

	"github.com/caucehealth/gofhir/r4/resources"
)

func TestGenerateNarrativePatient(t *testing.T) {
	p, _ := resources.NewPatient().
		WithName("John", "Doe").
		WithGender(resources.AdministrativeGenderMale).
		WithBirthDate("1980-01-01").
		Build()

	narrative := resources.GenerateNarrative(p)
	if narrative == nil {
		t.Fatal("narrative should not be nil")
	}
	if narrative.Status == nil || *narrative.Status != "generated" {
		t.Error("status should be 'generated'")
	}
	if !strings.Contains(narrative.Div, "Doe") {
		t.Error("narrative should contain family name")
	}
	if !strings.Contains(narrative.Div, "male") {
		t.Error("narrative should contain gender")
	}
	if !strings.Contains(narrative.Div, "1980-01-01") {
		t.Error("narrative should contain birth date")
	}
	if !strings.Contains(narrative.Div, `xmlns="http://www.w3.org/1999/xhtml"`) {
		t.Error("narrative should have XHTML namespace")
	}
}

func TestGenerateNarrativeObservation(t *testing.T) {
	obs, _ := resources.NewObservation().
		WithStatus(resources.ObservationStatusFinal).
		WithCode("http://loinc.org", "85354-9", "Blood pressure").
		Build()

	narrative := resources.GenerateNarrative(obs)
	if narrative == nil {
		t.Fatal("narrative should not be nil")
	}
	if !strings.Contains(narrative.Div, "Blood pressure") {
		t.Error("narrative should contain code display")
	}
	if !strings.Contains(narrative.Div, "final") {
		t.Error("narrative should contain status")
	}
}

func TestGenerateNarrativeUnsupported(t *testing.T) {
	// Unsupported resource type returns nil
	type FakeResource struct{}
	narrative := resources.GenerateNarrative(&FakeResource{})
	if narrative != nil {
		t.Error("unsupported type should return nil narrative")
	}
}
