// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package bundle_test

import (
	"encoding/json"
	"testing"

	"github.com/helixfhir/gofhir/r4/bundle"
	"github.com/helixfhir/gofhir/r4/resources"
)

func TestBundleBuilder(t *testing.T) {
	p, err := resources.NewPatient().
		WithName("John", "Doe").
		WithGender(resources.AdministrativeGenderMale).
		Build()
	if err != nil {
		t.Fatalf("build patient: %v", err)
	}

	b := bundle.New(bundle.TypeSearchset).
		WithID("test-bundle").
		WithTotal(1).
		WithEntry(p).
		Build()

	if b.ResourceType != "Bundle" {
		t.Errorf("resourceType = %q, want Bundle", b.ResourceType)
	}
	if string(b.Type) != "searchset" {
		t.Errorf("type = %q, want searchset", b.Type)
	}
	if b.Total == nil || *b.Total != 1 {
		t.Error("total should be 1")
	}
	if len(b.Entry) != 1 {
		t.Fatal("should have one entry")
	}
	if b.Entry[0].Resource == nil {
		t.Fatal("entry resource should not be nil")
	}
}

func TestBundleRoundTrip(t *testing.T) {
	input := `{
		"resourceType": "Bundle",
		"type": "searchset",
		"total": 1,
		"entry": [{
			"fullUrl": "http://example.com/Patient/1",
			"resource": {
				"resourceType": "Patient",
				"id": "1",
				"name": [{"family": "Doe"}]
			}
		}]
	}`

	var b bundle.Bundle
	if err := json.Unmarshal([]byte(input), &b); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if b.Type != bundle.TypeSearchset {
		t.Errorf("type = %q, want searchset", b.Type)
	}
	if b.Total == nil || *b.Total != 1 {
		t.Error("total should be 1")
	}
	if len(b.Entry) != 1 {
		t.Fatal("should have one entry")
	}

	// Verify we can parse the contained resource
	var patient resources.Patient
	if err := json.Unmarshal(b.Entry[0].Resource, &patient); err != nil {
		t.Fatalf("unmarshal patient from bundle: %v", err)
	}
	if patient.Id == nil || string(*patient.Id) != "1" {
		t.Error("patient id should be 1")
	}

	// Round-trip
	out, err := json.Marshal(&b)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var b2 bundle.Bundle
	if err := json.Unmarshal(out, &b2); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if b2.Type != bundle.TypeSearchset {
		t.Error("round-trip: type mismatch")
	}
	if len(b2.Entry) != 1 {
		t.Error("round-trip: entry count mismatch")
	}
}

func TestBundleMarshalJSON(t *testing.T) {
	p, err := resources.NewPatient().
		WithName("Jane", "Smith").
		Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	b := bundle.New(bundle.TypeCollection).
		WithEntry(p).
		Build()

	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal map: %v", err)
	}

	if _, ok := m["resourceType"]; !ok {
		t.Error("resourceType should be present")
	}
	if _, ok := m["type"]; !ok {
		t.Error("type should be present")
	}
	if _, ok := m["entry"]; !ok {
		t.Error("entry should be present")
	}
}

func TestBundleEmptyEntries(t *testing.T) {
	b := bundle.New(bundle.TypeTransaction).Build()

	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Empty entries should be omitted
	if _, ok := m["entry"]; ok {
		t.Error("empty entry array should be omitted")
	}
}
