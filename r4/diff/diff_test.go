// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package diff_test

import (
	"encoding/json"
	"testing"

	"github.com/caucehealth/gofhir/r4/diff"
	"github.com/caucehealth/gofhir/r4/resources"
)

func TestCompareIdentical(t *testing.T) {
	p1 := &resources.Patient{ResourceType: "Patient"}
	p2 := &resources.Patient{ResourceType: "Patient"}

	result, err := diff.Compare(p1, p2)
	if err != nil {
		t.Fatal(err)
	}
	if result.HasChanges() {
		t.Errorf("identical resources should have no changes, got %d", len(result.Changes))
	}
}

func TestCompareAddedField(t *testing.T) {
	old := `{"resourceType":"Patient","id":"1"}`
	new := `{"resourceType":"Patient","id":"1","gender":"male"}`

	result, err := diff.CompareJSON(json.RawMessage(old), json.RawMessage(new))
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasChanges() {
		t.Fatal("should have changes")
	}
	additions := result.Additions()
	if len(additions) != 1 {
		t.Fatalf("expected 1 addition, got %d", len(additions))
	}
	if additions[0].Path != "gender" {
		t.Errorf("path = %q, want gender", additions[0].Path)
	}
}

func TestCompareRemovedField(t *testing.T) {
	old := `{"resourceType":"Patient","id":"1","gender":"male"}`
	new := `{"resourceType":"Patient","id":"1"}`

	result, err := diff.CompareJSON(json.RawMessage(old), json.RawMessage(new))
	if err != nil {
		t.Fatal(err)
	}
	removals := result.Removals()
	if len(removals) != 1 {
		t.Fatalf("expected 1 removal, got %d", len(removals))
	}
	if removals[0].Path != "gender" {
		t.Errorf("path = %q, want gender", removals[0].Path)
	}
}

func TestCompareModifiedField(t *testing.T) {
	old := `{"resourceType":"Patient","gender":"male"}`
	new := `{"resourceType":"Patient","gender":"female"}`

	result, err := diff.CompareJSON(json.RawMessage(old), json.RawMessage(new))
	if err != nil {
		t.Fatal(err)
	}
	mods := result.Modifications()
	if len(mods) != 1 {
		t.Fatalf("expected 1 modification, got %d", len(mods))
	}
	if mods[0].OldValue != "male" {
		t.Errorf("old = %v, want male", mods[0].OldValue)
	}
	if mods[0].NewValue != "female" {
		t.Errorf("new = %v, want female", mods[0].NewValue)
	}
}

func TestCompareNestedChanges(t *testing.T) {
	old := `{"resourceType":"Patient","name":[{"family":"Smith","given":["John"]}]}`
	new := `{"resourceType":"Patient","name":[{"family":"Jones","given":["John","Q"]}]}`

	result, err := diff.CompareJSON(json.RawMessage(old), json.RawMessage(new))
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasChanges() {
		t.Fatal("should have changes")
	}

	// family changed from Smith to Jones
	foundFamily := false
	foundGiven := false
	for _, c := range result.Changes {
		if c.Path == "name/0/family" && c.Type == diff.Modified {
			foundFamily = true
		}
		if c.Path == "name/0/given/1" && c.Type == diff.Added {
			foundGiven = true
		}
	}
	if !foundFamily {
		t.Error("should detect family name change")
	}
	if !foundGiven {
		t.Error("should detect added given name")
	}
}

func TestCompareArrayShrink(t *testing.T) {
	old := `{"resourceType":"Patient","name":[{"family":"A"},{"family":"B"}]}`
	new := `{"resourceType":"Patient","name":[{"family":"A"}]}`

	result, err := diff.CompareJSON(json.RawMessage(old), json.RawMessage(new))
	if err != nil {
		t.Fatal(err)
	}
	removals := result.Removals()
	if len(removals) == 0 {
		t.Error("should detect removed array element")
	}
}

func TestCompareStructs(t *testing.T) {
	p1, _ := resources.NewPatient().
		WithName("John", "Doe").
		WithGender(resources.AdministrativeGenderMale).
		Build()
	p2, _ := resources.NewPatient().
		WithName("Jane", "Doe").
		WithGender(resources.AdministrativeGenderFemale).
		Build()

	result, err := diff.Compare(p1, p2)
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasChanges() {
		t.Fatal("different patients should have changes")
	}

	// Should detect gender change and name change
	foundGender := false
	for _, c := range result.Changes {
		if c.Path == "gender" {
			foundGender = true
		}
	}
	if !foundGender {
		t.Error("should detect gender change")
	}
}

func TestCompareEmptyResult(t *testing.T) {
	result := &diff.Result{}
	if result.HasChanges() {
		t.Error("empty result should not have changes")
	}
	if len(result.Additions()) != 0 {
		t.Error("empty should have no additions")
	}
	if len(result.Removals()) != 0 {
		t.Error("empty should have no removals")
	}
	if len(result.Modifications()) != 0 {
		t.Error("empty should have no modifications")
	}
}
