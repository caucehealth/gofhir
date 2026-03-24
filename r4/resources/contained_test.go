// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	"encoding/json"
	"testing"

	"github.com/helixfhir/gofhir/r4/resources"
)

func TestContainedResourceType(t *testing.T) {
	raw := json.RawMessage(`{"resourceType":"Patient","id":"1"}`)
	rt, err := resources.ContainedResourceType(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt != "Patient" {
		t.Errorf("got %q, want Patient", rt)
	}
}

func TestParseContainedPatient(t *testing.T) {
	raw := json.RawMessage(`{"resourceType":"Patient","id":"1","gender":"male"}`)
	p, err := resources.ParseContainedPatient(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Gender == nil || string(*p.Gender) != "male" {
		t.Error("gender should be male")
	}
}

func TestParseContainedWrongType(t *testing.T) {
	raw := json.RawMessage(`{"resourceType":"Observation","id":"1"}`)
	_, err := resources.ParseContainedPatient(raw)
	if err == nil {
		t.Error("expected error for wrong resource type")
	}
}

func TestParseContained(t *testing.T) {
	raw := json.RawMessage(`{"resourceType":"Practitioner","id":"1","name":[{"family":"Smith"}]}`)
	result, err := resources.ParseContained(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	prac, ok := result.(*resources.Practitioner)
	if !ok {
		t.Fatalf("expected *Practitioner, got %T", result)
	}
	if len(prac.Name) != 1 || prac.Name[0].Family == nil || *prac.Name[0].Family != "Smith" {
		t.Error("name should be Smith")
	}
}

func TestContainedInResource(t *testing.T) {
	input := `{
		"resourceType": "Patient",
		"id": "parent",
		"contained": [
			{"resourceType": "Practitioner", "id": "prac1", "name": [{"family": "Jones"}]}
		],
		"generalPractitioner": [{"reference": "#prac1"}]
	}`

	var patient resources.Patient
	if err := json.Unmarshal([]byte(input), &patient); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(patient.Contained) != 1 {
		t.Fatal("should have one contained resource")
	}

	prac, err := resources.ParseContainedPractitioner(patient.Contained[0])
	if err != nil {
		t.Fatalf("parse contained: %v", err)
	}
	if len(prac.Name) != 1 || *prac.Name[0].Family != "Jones" {
		t.Error("contained practitioner name should be Jones")
	}
}
