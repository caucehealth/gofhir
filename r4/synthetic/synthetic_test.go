// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package synthetic_test

import (
	"encoding/json"
	"testing"

	"github.com/caucehealth/gofhir/r4/resources"
	"github.com/caucehealth/gofhir/r4/synthetic"
)

func TestPatientGeneration(t *testing.T) {
	gen := synthetic.NewWithSeed(42)
	p := gen.Patient()

	if p.ResourceType != "Patient" {
		t.Errorf("type = %q", p.ResourceType)
	}
	if p.Id == nil {
		t.Error("should have ID")
	}
	if len(p.Name) == 0 {
		t.Error("should have name")
	}
	if p.Name[0].Family == nil || *p.Name[0].Family == "" {
		t.Error("should have family name")
	}
	if p.Gender == nil {
		t.Error("should have gender")
	}
	if p.BirthDate == nil {
		t.Error("should have birth date")
	}

	// Should be valid JSON
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var roundtrip resources.Patient
	if err := json.Unmarshal(data, &roundtrip); err != nil {
		t.Fatal(err)
	}
}

func TestObservationGeneration(t *testing.T) {
	gen := synthetic.NewWithSeed(42)
	obs := gen.Observation("patient-1")

	if obs.ResourceType != "Observation" {
		t.Errorf("type = %q", obs.ResourceType)
	}
	if len(obs.Code.Coding) == 0 {
		t.Error("should have code")
	}
	if obs.Value == nil || obs.Value.Quantity == nil {
		t.Error("should have value quantity")
	}
	if obs.Subject == nil || obs.Subject.Reference == nil {
		t.Error("should reference patient")
	}
	if *obs.Subject.Reference != "Patient/patient-1" {
		t.Errorf("subject = %q", *obs.Subject.Reference)
	}

	data, err := json.Marshal(obs)
	if err != nil {
		t.Fatal(err)
	}
	var roundtrip resources.Observation
	if err := json.Unmarshal(data, &roundtrip); err != nil {
		t.Fatal(err)
	}
}

func TestConditionGeneration(t *testing.T) {
	gen := synthetic.NewWithSeed(42)
	c := gen.Condition("patient-1")

	if c.ResourceType != "Condition" {
		t.Errorf("type = %q", c.ResourceType)
	}
	if c.Code == nil {
		t.Error("should have code")
	}
	if c.ClinicalStatus == nil {
		t.Error("should have clinical status")
	}
	if c.Subject.Reference == nil {
		t.Error("should reference patient")
	}
}

func TestEncounterGeneration(t *testing.T) {
	gen := synthetic.NewWithSeed(42)
	e := gen.Encounter("patient-1")

	if e.ResourceType != "Encounter" {
		t.Errorf("type = %q", e.ResourceType)
	}
	if e.Class.Code == nil {
		t.Error("should have class code")
	}
	if e.Period == nil {
		t.Error("should have period")
	}
	if e.Subject == nil {
		t.Error("should reference patient")
	}
}

func TestReproducibility(t *testing.T) {
	gen1 := synthetic.NewWithSeed(123)
	gen2 := synthetic.NewWithSeed(123)

	p1 := gen1.Patient()
	p2 := gen2.Patient()

	data1, _ := json.Marshal(p1)
	data2, _ := json.Marshal(p2)

	if string(data1) != string(data2) {
		t.Error("same seed should produce identical output")
	}
}

func TestDifferentSeeds(t *testing.T) {
	gen1 := synthetic.NewWithSeed(1)
	gen2 := synthetic.NewWithSeed(2)

	p1 := gen1.Patient()
	p2 := gen2.Patient()

	data1, _ := json.Marshal(p1)
	data2, _ := json.Marshal(p2)

	if string(data1) == string(data2) {
		t.Error("different seeds should produce different output")
	}
}

func TestPatientBundle(t *testing.T) {
	gen := synthetic.NewWithSeed(42)
	patients := gen.PatientBundle(10)

	if len(patients) != 10 {
		t.Errorf("expected 10 patients, got %d", len(patients))
	}

	// All should have unique IDs
	ids := map[string]bool{}
	for _, p := range patients {
		id := string(*p.Id)
		if ids[id] {
			t.Errorf("duplicate ID: %s", id)
		}
		ids[id] = true
	}
}

func TestPopulatedBundle(t *testing.T) {
	gen := synthetic.NewWithSeed(42)
	patients, obs, conds, encs := gen.PopulatedBundle(5)

	if len(patients) != 5 {
		t.Errorf("expected 5 patients, got %d", len(patients))
	}
	if len(obs) == 0 {
		t.Error("should have observations")
	}
	// At least some conditions expected (probabilistic, but with 5 patients very likely)
	if len(encs) == 0 {
		t.Error("should have encounters")
	}

	// All observations should reference a known patient
	patientIDs := map[string]bool{}
	for _, p := range patients {
		patientIDs["Patient/"+string(*p.Id)] = true
	}
	for _, o := range obs {
		if o.Subject == nil || o.Subject.Reference == nil {
			t.Error("observation should reference patient")
			continue
		}
		if !patientIDs[*o.Subject.Reference] {
			t.Errorf("observation references unknown patient: %s", *o.Subject.Reference)
		}
	}

	_ = conds // conditions are probabilistic
}

func TestJSONRoundTrip(t *testing.T) {
	gen := synthetic.NewWithSeed(42)
	patients, obs, conds, encs := gen.PopulatedBundle(3)

	// Verify all generated resources round-trip through JSON
	for _, p := range patients {
		data, _ := json.Marshal(p)
		var rt resources.Patient
		if err := json.Unmarshal(data, &rt); err != nil {
			t.Errorf("patient round-trip: %v", err)
		}
	}
	for _, o := range obs {
		data, _ := json.Marshal(o)
		var rt resources.Observation
		if err := json.Unmarshal(data, &rt); err != nil {
			t.Errorf("observation round-trip: %v", err)
		}
	}
	for _, c := range conds {
		data, _ := json.Marshal(c)
		var rt resources.Condition
		if err := json.Unmarshal(data, &rt); err != nil {
			t.Errorf("condition round-trip: %v", err)
		}
	}
	for _, e := range encs {
		data, _ := json.Marshal(e)
		var rt resources.Encounter
		if err := json.Unmarshal(data, &rt); err != nil {
			t.Errorf("encounter round-trip: %v", err)
		}
	}
}
