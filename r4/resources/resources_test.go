// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	"encoding/json"
	"testing"

	dt "github.com/helixfhir/gofhir/r4/datatypes"
	"github.com/helixfhir/gofhir/r4/resources"
)

// --- Round-trip tests ---

func TestPatientRoundTrip(t *testing.T) {
	input := `{
		"resourceType": "Patient",
		"id": "example",
		"active": true,
		"name": [{"family": "Doe", "given": ["John"]}],
		"gender": "male",
		"birthDate": "1980-03-15",
		"address": [{"city": "Anytown", "state": "CA"}],
		"telecom": [{"system": "phone", "value": "555-1234"}]
	}`

	var patient resources.Patient
	if err := json.Unmarshal([]byte(input), &patient); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if patient.ResourceType != "Patient" {
		t.Errorf("resourceType = %q, want Patient", patient.ResourceType)
	}
	if patient.Id == nil || string(*patient.Id) != "example" {
		t.Errorf("id = %v, want example", patient.Id)
	}
	if patient.Active == nil || !*patient.Active {
		t.Error("active should be true")
	}
	if len(patient.Name) != 1 || patient.Name[0].Family == nil || *patient.Name[0].Family != "Doe" {
		t.Error("name.family should be Doe")
	}
	if patient.Gender == nil || *patient.Gender != "male" {
		t.Error("gender should be male")
	}

	out, err := json.Marshal(&patient)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var reparsed resources.Patient
	if err := json.Unmarshal(out, &reparsed); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if reparsed.Id == nil || string(*reparsed.Id) != "example" {
		t.Error("round-trip: id mismatch")
	}
	if reparsed.Gender == nil || *reparsed.Gender != "male" {
		t.Error("round-trip: gender mismatch")
	}
}

func TestPatientWithDeceasedRoundTrip(t *testing.T) {
	input := `{
		"resourceType": "Patient",
		"id": "deceased-example",
		"deceasedBoolean": true
	}`

	var patient resources.Patient
	if err := json.Unmarshal([]byte(input), &patient); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if patient.Deceased == nil {
		t.Fatal("deceased should not be nil")
	}
	if patient.Deceased.Boolean == nil || !*patient.Deceased.Boolean {
		t.Error("deceased.boolean should be true")
	}

	out, err := json.Marshal(&patient)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal map: %v", err)
	}
	if _, ok := m["deceasedBoolean"]; !ok {
		t.Error("round-trip: deceasedBoolean should be present in JSON")
	}
}

func TestObservationRoundTrip(t *testing.T) {
	input := `{
		"resourceType": "Observation",
		"id": "blood-pressure",
		"status": "final",
		"code": {
			"coding": [{"system": "http://loinc.org", "code": "85354-9", "display": "Blood pressure"}]
		},
		"subject": {"reference": "Patient/example"},
		"valueQuantity": {
			"value": 120,
			"unit": "mmHg",
			"system": "http://unitsofmeasure.org",
			"code": "mm[Hg]"
		}
	}`

	var obs resources.Observation
	if err := json.Unmarshal([]byte(input), &obs); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if obs.Status == nil || *obs.Status != "final" {
		t.Error("status should be final")
	}
	if obs.Value == nil || obs.Value.Quantity == nil {
		t.Fatal("value.quantity should not be nil")
	}
	if obs.Value.Quantity.Value == nil || *obs.Value.Quantity.Value != 120 {
		t.Error("value.quantity.value should be 120")
	}

	out, err := json.Marshal(&obs)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal map: %v", err)
	}
	if _, ok := m["valueQuantity"]; !ok {
		t.Error("round-trip: valueQuantity should be present in JSON")
	}
}

func TestEncounterRoundTrip(t *testing.T) {
	input := `{
		"resourceType": "Encounter",
		"id": "example",
		"status": "finished",
		"class": {"system": "http://terminology.hl7.org/CodeSystem/v3-ActCode", "code": "IMP"},
		"subject": {"reference": "Patient/example"}
	}`

	var enc resources.Encounter
	if err := json.Unmarshal([]byte(input), &enc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if enc.Status == nil || *enc.Status != "finished" {
		t.Error("status should be finished")
	}

	out, err := json.Marshal(&enc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var reparsed resources.Encounter
	if err := json.Unmarshal(out, &reparsed); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if reparsed.Status == nil || *reparsed.Status != "finished" {
		t.Error("round-trip: status mismatch")
	}
}

func TestConditionRoundTrip(t *testing.T) {
	input := `{
		"resourceType": "Condition",
		"id": "example",
		"code": {
			"coding": [{"system": "http://snomed.info/sct", "code": "386661006", "display": "Fever"}]
		},
		"subject": {"reference": "Patient/example"}
	}`

	var cond resources.Condition
	if err := json.Unmarshal([]byte(input), &cond); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cond.Code == nil || len(cond.Code.Coding) == 0 {
		t.Fatal("code.coding should be present")
	}

	out, err := json.Marshal(&cond)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var reparsed resources.Condition
	if err := json.Unmarshal(out, &reparsed); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
}

func TestPractitionerRoundTrip(t *testing.T) {
	input := `{
		"resourceType": "Practitioner",
		"id": "example",
		"name": [{"family": "Smith", "given": ["Jane"]}]
	}`

	var prac resources.Practitioner
	if err := json.Unmarshal([]byte(input), &prac); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	out, err := json.Marshal(&prac)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var reparsed resources.Practitioner
	if err := json.Unmarshal(out, &reparsed); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if len(reparsed.Name) != 1 || reparsed.Name[0].Family == nil || *reparsed.Name[0].Family != "Smith" {
		t.Error("round-trip: name mismatch")
	}
}

// --- Builder tests ---

func TestPatientBuilder(t *testing.T) {
	p, err := resources.NewPatient().
		WithName("John", "Doe").
		WithBirthDate("1980-03-15").
		WithGender(resources.AdministrativeGenderMale).
		Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if p.ResourceType != "Patient" {
		t.Errorf("resourceType = %q, want Patient", p.ResourceType)
	}
	if len(p.Name) != 1 {
		t.Fatal("should have one name")
	}
	if p.Name[0].Family == nil || *p.Name[0].Family != "Doe" {
		t.Error("family should be Doe")
	}
	if len(p.Name[0].Given) != 1 || p.Name[0].Given[0] != "John" {
		t.Error("given should be [John]")
	}
	if p.BirthDate == nil || string(*p.BirthDate) != "1980-03-15" {
		t.Error("birthDate should be 1980-03-15")
	}
	if p.Gender == nil || *p.Gender != "male" {
		t.Error("gender should be male")
	}

	// Verify JSON round-trip
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var reparsed resources.Patient
	if err := json.Unmarshal(data, &reparsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if reparsed.Gender == nil || *reparsed.Gender != "male" {
		t.Error("round-trip: gender mismatch")
	}
}

func TestObservationBuilder(t *testing.T) {
	obs, err := resources.NewObservation().
		WithStatus("final").
		WithCode("http://loinc.org", "85354-9", "Blood pressure").
		WithSubject("Patient/example").
		Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if obs.Status == nil || *obs.Status != "final" {
		t.Error("status should be final")
	}
	if len(obs.Code.Coding) != 1 {
		t.Fatal("should have one coding")
	}
}

func TestEncounterBuilder(t *testing.T) {
	enc, err := resources.NewEncounter().
		WithStatus("finished").
		WithClass("http://terminology.hl7.org/CodeSystem/v3-ActCode", "IMP").
		WithSubject("Patient/example").
		Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if enc.Status == nil || *enc.Status != "finished" {
		t.Error("status should be finished")
	}
}

func TestConditionBuilder(t *testing.T) {
	cond, err := resources.NewCondition().
		WithCode("http://snomed.info/sct", "386661006", "Fever").
		WithSubject("Patient/example").
		Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if cond.Code == nil || len(cond.Code.Coding) != 1 {
		t.Fatal("should have one coding")
	}
}

func TestDiagnosticReportRoundTrip(t *testing.T) {
	input := `{
		"resourceType": "DiagnosticReport",
		"id": "dr-example",
		"status": "final",
		"code": {
			"coding": [{"system": "http://loinc.org", "code": "58410-2", "display": "CBC panel"}]
		},
		"subject": {"reference": "Patient/example"},
		"conclusion": "Normal blood count"
	}`

	var dr resources.DiagnosticReport
	if err := json.Unmarshal([]byte(input), &dr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if dr.Status == nil || *dr.Status != "final" {
		t.Error("status should be final")
	}
	if len(dr.Code.Coding) != 1 {
		t.Fatal("should have one coding")
	}
	if dr.Conclusion == nil || *dr.Conclusion != "Normal blood count" {
		t.Error("conclusion mismatch")
	}

	out, err := json.Marshal(&dr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var reparsed resources.DiagnosticReport
	if err := json.Unmarshal(out, &reparsed); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if reparsed.Status == nil || *reparsed.Status != "final" {
		t.Error("round-trip: status mismatch")
	}
	if reparsed.Conclusion == nil || *reparsed.Conclusion != "Normal blood count" {
		t.Error("round-trip: conclusion mismatch")
	}
}

func TestMedicationRequestRoundTrip(t *testing.T) {
	input := `{
		"resourceType": "MedicationRequest",
		"id": "medrx-example",
		"status": "active",
		"intent": "order",
		"subject": {"reference": "Patient/example"},
		"medicationCodeableConcept": {
			"coding": [{"system": "http://www.nlm.nih.gov/research/umls/rxnorm", "code": "1049502", "display": "Acetaminophen 325 MG"}]
		},
		"authoredOn": "2023-01-15"
	}`

	var mr resources.MedicationRequest
	if err := json.Unmarshal([]byte(input), &mr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if mr.Status == nil || string(*mr.Status) != "active" {
		t.Error("status should be active")
	}
	if mr.Intent == nil || string(*mr.Intent) != "order" {
		t.Error("intent should be order")
	}
	if mr.AuthoredOn == nil || string(*mr.AuthoredOn) != "2023-01-15" {
		t.Error("authoredOn should be 2023-01-15")
	}
	// Check polymorphic medication field
	if mr.Medication == nil || mr.Medication.CodeableConcept == nil {
		t.Fatal("medication.codeableConcept should not be nil")
	}

	out, err := json.Marshal(&mr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var reparsed resources.MedicationRequest
	if err := json.Unmarshal(out, &reparsed); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if reparsed.Status == nil || string(*reparsed.Status) != "active" {
		t.Error("round-trip: status mismatch")
	}
	if reparsed.Medication == nil || reparsed.Medication.CodeableConcept == nil {
		t.Error("round-trip: medication lost")
	}

	// Verify the polymorphic field appears correctly in JSON
	var m map[string]json.RawMessage
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal map: %v", err)
	}
	if _, ok := m["medicationCodeableConcept"]; !ok {
		t.Error("round-trip: medicationCodeableConcept should be present in JSON")
	}
}

func TestDiagnosticReportBuilder(t *testing.T) {
	dr, err := resources.NewDiagnosticReport().
		WithStatus("final").
		WithCode("http://loinc.org", "58410-2", "CBC panel").
		WithSubject("Patient/example").
		Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if dr.Status == nil || *dr.Status != "final" {
		t.Error("status should be final")
	}
	if len(dr.Code.Coding) != 1 {
		t.Fatal("should have one coding")
	}
	if dr.Subject == nil || dr.Subject.Reference == nil || *dr.Subject.Reference != "Patient/example" {
		t.Error("subject should be Patient/example")
	}
}

func TestMedicationRequestBuilder(t *testing.T) {
	mr, err := resources.NewMedicationRequest().
		WithStatus("active").
		WithIntent("order").
		WithSubject("Patient/example").
		WithMedicationCodeableConcept("http://www.nlm.nih.gov/research/umls/rxnorm", "1049502", "Acetaminophen 325 MG").
		Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if mr.Status == nil || string(*mr.Status) != "active" {
		t.Error("status should be active")
	}
	if mr.Intent == nil || string(*mr.Intent) != "order" {
		t.Error("intent should be order")
	}
	if mr.Medication == nil || mr.Medication.CodeableConcept == nil {
		t.Fatal("medication should be set")
	}
	if len(mr.Medication.CodeableConcept.Coding) != 1 {
		t.Fatal("should have one medication coding")
	}
}

func TestPractitionerBuilder(t *testing.T) {
	prac, err := resources.NewPractitioner().
		WithName("Jane", "Smith").
		Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if len(prac.Name) != 1 || prac.Name[0].Family == nil || *prac.Name[0].Family != "Smith" {
		t.Error("name should be Smith")
	}
}

// --- Edge case tests ---

func TestEmptyOptionalFieldsOmitted(t *testing.T) {
	p, err := resources.NewPatient().Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Optional fields should not be present
	for _, field := range []string{"name", "gender", "birthDate", "address"} {
		if _, ok := m[field]; ok {
			t.Errorf("optional field %q should be omitted when empty", field)
		}
	}

	// resourceType should always be present
	if _, ok := m["resourceType"]; !ok {
		t.Error("resourceType should always be present")
	}
}

func TestPolymorphicValueRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(t *testing.T, obs resources.Observation)
	}{
		{
			name:  "valueString",
			input: `{"resourceType":"Observation","status":"final","code":{"text":"test"},"valueString":"hello"}`,
			check: func(t *testing.T, obs resources.Observation) {
				if obs.Value == nil || obs.Value.String == nil || *obs.Value.String != "hello" {
					t.Error("value.string should be hello")
				}
			},
		},
		{
			name:  "valueBoolean",
			input: `{"resourceType":"Observation","status":"final","code":{"text":"test"},"valueBoolean":true}`,
			check: func(t *testing.T, obs resources.Observation) {
				if obs.Value == nil || obs.Value.Boolean == nil || !*obs.Value.Boolean {
					t.Error("value.boolean should be true")
				}
			},
		},
		{
			name:  "valueInteger",
			input: `{"resourceType":"Observation","status":"final","code":{"text":"test"},"valueInteger":42}`,
			check: func(t *testing.T, obs resources.Observation) {
				if obs.Value == nil || obs.Value.Integer == nil || *obs.Value.Integer != 42 {
					t.Error("value.integer should be 42")
				}
			},
		},
		{
			name:  "valueCodeableConcept",
			input: `{"resourceType":"Observation","status":"final","code":{"text":"test"},"valueCodeableConcept":{"text":"positive"}}`,
			check: func(t *testing.T, obs resources.Observation) {
				if obs.Value == nil || obs.Value.CodeableConcept == nil || obs.Value.CodeableConcept.Text == nil {
					t.Error("value.codeableConcept.text should be present")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var obs resources.Observation
			if err := json.Unmarshal([]byte(tt.input), &obs); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			tt.check(t, obs)

			// Round-trip
			out, err := json.Marshal(&obs)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var obs2 resources.Observation
			if err := json.Unmarshal(out, &obs2); err != nil {
				t.Fatalf("re-unmarshal: %v", err)
			}
			tt.check(t, obs2)
		})
	}
}

func TestNestedExtensionRoundTrip(t *testing.T) {
	input := `{
		"resourceType": "Patient",
		"id": "ext-example",
		"extension": [{
			"url": "http://example.org/fhir/StructureDefinition/patient-importance",
			"extension": [{
				"url": "level",
				"valueCode": "VIP"
			}, {
				"url": "reason",
				"valueString": "Donor"
			}]
		}]
	}`

	var patient resources.Patient
	if err := json.Unmarshal([]byte(input), &patient); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(patient.Extension) != 1 {
		t.Fatal("should have one extension")
	}
	ext := patient.Extension[0]
	if string(ext.Url) != "http://example.org/fhir/StructureDefinition/patient-importance" {
		t.Error("extension URL mismatch")
	}
	if len(ext.Extension) != 2 {
		t.Fatal("should have two nested extensions")
	}
	if ext.Extension[0].ValueCode == nil || string(*ext.Extension[0].ValueCode) != "VIP" {
		t.Error("nested ext[0].valueCode should be VIP")
	}

	out, err := json.Marshal(&patient)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var reparsed resources.Patient
	if err := json.Unmarshal(out, &reparsed); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if len(reparsed.Extension) != 1 || len(reparsed.Extension[0].Extension) != 2 {
		t.Error("round-trip: nested extensions lost")
	}
}

func TestReferenceRoundTrip(t *testing.T) {
	ref := dt.Reference{
		Reference: strPtr("Patient/123"),
		Display:   strPtr("John Doe"),
	}

	data, err := json.Marshal(ref)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var reparsed dt.Reference
	if err := json.Unmarshal(data, &reparsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if reparsed.Reference == nil || *reparsed.Reference != "Patient/123" {
		t.Error("reference mismatch")
	}
	if reparsed.Display == nil || *reparsed.Display != "John Doe" {
		t.Error("display mismatch")
	}
}

func TestBuildRequiredFieldsMissing(t *testing.T) {
	// Observation.code is required (1..1) — but it's a struct type, so our generator
	// doesn't validate it. Observation.status is *string (optional in schema but
	// typically required). This test verifies Build() succeeds with minimal fields
	// since most FHIR required fields are struct types.
	_, err := resources.NewObservation().Build()
	if err != nil {
		t.Fatalf("Build should succeed for Observation with no required string fields: %v", err)
	}
}

func TestParseInstant(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"2023-01-15T10:30:00Z", false},
		{"2023-01-15T10:30:00+05:00", false},
		{"2023-01-15T10:30:00.123Z", false},
		{"2023-01-15T10:30:00.123456789Z", false},
		{"not-a-date", true},
		{"2023-01-15", true},
		{"2023-01-15T10:30:00", true}, // missing timezone
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := dt.ParseInstant(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseInstant(%q) err = %v, wantErr = %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}
