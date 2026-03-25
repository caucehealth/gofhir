// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	dt "github.com/caucehealth/gofhir/r4/datatypes"
	"github.com/caucehealth/gofhir/r4/resources"
)

// --- Helpers ---

// loadTestData reads a JSON file from testdata/.
func loadTestData(t *testing.T, filename string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + filename)
	if err != nil {
		t.Fatalf("loading testdata/%s: %v", filename, err)
	}
	return data
}

// normalizeJSON re-marshals JSON to canonical form for comparison.
func normalizeJSON(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("normalizeJSON: %v", err)
	}
	return m
}

// assertJSONRoundTrip verifies unmarshal→marshal→unmarshal produces equivalent JSON.
// It compares all keys present in the output (not the input, since we may drop _ keys).
func assertJSONRoundTrip[T any](t *testing.T, input []byte, zero T) {
	t.Helper()

	// Parse
	if err := json.Unmarshal(input, &zero); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	// Marshal
	output, err := json.Marshal(&zero)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	// Re-parse to verify
	var reparsed T
	if err := json.Unmarshal(output, &reparsed); err != nil {
		t.Fatalf("re-unmarshal failed: %v", err)
	}

	// Compare normalized JSON (strips _ keys from input)
	inputNorm := normalizeJSON(t, input)
	outputNorm := normalizeJSON(t, output)

	// Check every key in the output matches the input
	for key, outVal := range outputNorm {
		inVal, ok := inputNorm[key]
		if !ok {
			// Key in output but not in input (e.g. resourceType added by marshal)
			continue
		}
		outJSON, _ := json.Marshal(outVal)
		inJSON, _ := json.Marshal(inVal)
		if string(outJSON) != string(inJSON) {
			t.Errorf("round-trip mismatch on key %q:\n  input:  %s\n  output: %s", key, inJSON, outJSON)
		}
	}

	// Check for keys in input (after stripping _) that are missing from output
	for key := range inputNorm {
		if _, ok := outputNorm[key]; !ok {
			t.Errorf("round-trip: key %q present in input but missing from output", key)
		}
	}
}

// ============================================================================
// Round-trip tests against OFFICIAL HL7 FHIR R4 examples
// ============================================================================

func TestPatientHL7Example(t *testing.T) {
	data := loadTestData(t, "patient-example.json")
	var patient resources.Patient
	assertJSONRoundTrip(t, data, patient)

	// Verify specific fields from the real example
	if err := json.Unmarshal(data, &patient); err != nil {
		t.Fatal(err)
	}
	if patient.Id == nil || string(*patient.Id) != "example" {
		t.Error("id should be 'example'")
	}
	if patient.Gender == nil || string(*patient.Gender) != "male" {
		t.Error("gender should be 'male'")
	}
	if patient.Active == nil || !*patient.Active {
		t.Error("active should be true")
	}
	if patient.BirthDate == nil || string(*patient.BirthDate) != "1974-12-25" {
		t.Error("birthDate should be '1974-12-25'")
	}
	if patient.Deceased == nil || patient.Deceased.Boolean == nil || *patient.Deceased.Boolean {
		t.Error("deceasedBoolean should be false")
	}
	// Real example has 3 identifiers
	if len(patient.Identifier) < 1 {
		t.Error("should have at least one identifier")
	}
	// Real example has name, telecom, address, contact
	if len(patient.Name) == 0 {
		t.Error("should have at least one name")
	}
	if len(patient.Telecom) == 0 {
		t.Error("should have at least one telecom")
	}
	if len(patient.Address) == 0 {
		t.Error("should have at least one address")
	}
	if patient.ManagingOrganization == nil {
		t.Error("managingOrganization should be present")
	}
}

func TestObservationHL7Example(t *testing.T) {
	data := loadTestData(t, "observation-example.json")
	var obs resources.Observation
	assertJSONRoundTrip(t, data, obs)

	if err := json.Unmarshal(data, &obs); err != nil {
		t.Fatal(err)
	}
	if obs.Status == nil || string(*obs.Status) != "final" {
		t.Error("status should be 'final'")
	}
	// Real example has valueQuantity
	if obs.Value == nil {
		t.Fatal("value should not be nil")
	}
	if obs.Value.Quantity == nil {
		t.Fatal("value should be a Quantity")
	}
	if obs.Value.Quantity.Value == nil {
		t.Error("quantity.value should be set")
	}
	if obs.Value.Quantity.Unit == nil {
		t.Error("quantity.unit should be set")
	}
	// Real example has effectiveDateTime (polymorphic)
	if obs.Effective == nil || obs.Effective.DateTime == nil {
		t.Error("effectiveDateTime should be present")
	}
	// Has code with coding
	if len(obs.Code.Coding) == 0 {
		t.Error("code.coding should not be empty")
	}
	// Has subject reference
	if obs.Subject == nil || obs.Subject.Reference == nil {
		t.Error("subject.reference should be set")
	}
}

func TestEncounterHL7Example(t *testing.T) {
	data := loadTestData(t, "encounter-example.json")
	var enc resources.Encounter
	assertJSONRoundTrip(t, data, enc)

	if err := json.Unmarshal(data, &enc); err != nil {
		t.Fatal(err)
	}
	if enc.Status == nil || string(*enc.Status) != "in-progress" {
		t.Errorf("status should be 'in-progress', got %v", enc.Status)
	}
}

func TestPractitionerHL7Example(t *testing.T) {
	data := loadTestData(t, "practitioner-example.json")
	var prac resources.Practitioner
	assertJSONRoundTrip(t, data, prac)

	if err := json.Unmarshal(data, &prac); err != nil {
		t.Fatal(err)
	}
	if prac.Active == nil || !*prac.Active {
		t.Error("active should be true")
	}
	if len(prac.Name) == 0 {
		t.Error("should have at least one name")
	}
	if len(prac.Identifier) == 0 {
		t.Error("should have identifiers")
	}
	if len(prac.Qualification) == 0 {
		t.Error("should have qualifications")
	}
}

func TestConditionHL7Example(t *testing.T) {
	data := loadTestData(t, "condition-example.json")
	var cond resources.Condition
	assertJSONRoundTrip(t, data, cond)

	if err := json.Unmarshal(data, &cond); err != nil {
		t.Fatal(err)
	}
	// Real condition has onsetDateTime (polymorphic)
	if cond.Onset == nil || cond.Onset.DateTime == nil {
		t.Error("onsetDateTime should be present")
	}
	// Has bodySite
	if len(cond.BodySite) == 0 {
		t.Error("bodySite should be present")
	}
	// Has severity
	if cond.Severity == nil {
		t.Error("severity should be present")
	}
}

func TestDiagnosticReportHL7Example(t *testing.T) {
	data := loadTestData(t, "diagnosticreport-single.json")
	var dr resources.DiagnosticReport
	assertJSONRoundTrip(t, data, dr)

	if err := json.Unmarshal(data, &dr); err != nil {
		t.Fatal(err)
	}
	if dr.Status == nil || string(*dr.Status) != "final" {
		t.Error("status should be 'final'")
	}
	if len(dr.Result) == 0 {
		t.Error("should have results")
	}
	if dr.Issued == nil {
		t.Error("issued should be present")
	}
}

func TestMedicationRequestHL7Example(t *testing.T) {
	data := loadTestData(t, "medicationrequest0301.json")
	var mr resources.MedicationRequest

	assertJSONRoundTrip(t, data, mr)

	if err := json.Unmarshal(data, &mr); err != nil {
		t.Fatal(err)
	}
	if mr.Status == nil || string(*mr.Status) != "completed" {
		t.Errorf("status should be 'completed', got %v", mr.Status)
	}
	if mr.Intent == nil || string(*mr.Intent) != "order" {
		t.Errorf("intent should be 'order', got %v", mr.Intent)
	}
	// This example uses contained resources
	if len(mr.Contained) == 0 {
		t.Error("should have contained resources")
	}
	// Has medicationReference (polymorphic)
	if mr.Medication == nil {
		t.Error("medication should be present")
	}
	// Has dosageInstruction
	if len(mr.DosageInstruction) == 0 {
		t.Error("should have dosage instructions")
	}
	// Has dispenseRequest
	if mr.DispenseRequest == nil {
		t.Error("dispenseRequest should be present")
	}
}

// ============================================================================
// Polymorphic value[x] — comprehensive tests
// ============================================================================

func TestPolymorphicValueTypes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(t *testing.T, obs resources.Observation)
	}{
		{
			name:  "valueQuantity",
			input: `{"resourceType":"Observation","status":"final","code":{"text":"test"},"valueQuantity":{"value":120,"unit":"mmHg","system":"http://unitsofmeasure.org","code":"mm[Hg]"}}`,
			check: func(t *testing.T, obs resources.Observation) {
				if obs.Value == nil || obs.Value.Quantity == nil {
					t.Fatal("expected Quantity")
				}
				if obs.Value.Quantity.Value == nil || obs.Value.Quantity.Value.Float64() != 120 {
					t.Error("quantity value should be 120")
				}
				if obs.Value.Quantity.Unit == nil || *obs.Value.Quantity.Unit != "mmHg" {
					t.Error("quantity unit should be mmHg")
				}
			},
		},
		{
			name:  "valueString",
			input: `{"resourceType":"Observation","status":"final","code":{"text":"test"},"valueString":"positive"}`,
			check: func(t *testing.T, obs resources.Observation) {
				if obs.Value == nil || obs.Value.String == nil || *obs.Value.String != "positive" {
					t.Error("expected valueString 'positive'")
				}
			},
		},
		{
			name:  "valueBoolean_true",
			input: `{"resourceType":"Observation","status":"final","code":{"text":"test"},"valueBoolean":true}`,
			check: func(t *testing.T, obs resources.Observation) {
				if obs.Value == nil || obs.Value.Boolean == nil || !*obs.Value.Boolean {
					t.Error("expected true")
				}
			},
		},
		{
			name:  "valueBoolean_false",
			input: `{"resourceType":"Observation","status":"final","code":{"text":"test"},"valueBoolean":false}`,
			check: func(t *testing.T, obs resources.Observation) {
				if obs.Value == nil || obs.Value.Boolean == nil || *obs.Value.Boolean {
					t.Error("expected false")
				}
			},
		},
		{
			name:  "valueInteger_zero",
			input: `{"resourceType":"Observation","status":"final","code":{"text":"test"},"valueInteger":0}`,
			check: func(t *testing.T, obs resources.Observation) {
				if obs.Value == nil || obs.Value.Integer == nil || *obs.Value.Integer != 0 {
					t.Error("expected integer 0")
				}
			},
		},
		{
			name:  "valueInteger_negative",
			input: `{"resourceType":"Observation","status":"final","code":{"text":"test"},"valueInteger":-5}`,
			check: func(t *testing.T, obs resources.Observation) {
				if obs.Value == nil || obs.Value.Integer == nil || *obs.Value.Integer != -5 {
					t.Error("expected integer -5")
				}
			},
		},
		{
			name:  "valueCodeableConcept",
			input: `{"resourceType":"Observation","status":"final","code":{"text":"test"},"valueCodeableConcept":{"coding":[{"system":"http://snomed.info/sct","code":"260385009","display":"Negative"}],"text":"Negative"}}`,
			check: func(t *testing.T, obs resources.Observation) {
				if obs.Value == nil || obs.Value.CodeableConcept == nil {
					t.Fatal("expected CodeableConcept")
				}
				if len(obs.Value.CodeableConcept.Coding) != 1 {
					t.Fatal("expected one coding")
				}
				if obs.Value.CodeableConcept.Text == nil || *obs.Value.CodeableConcept.Text != "Negative" {
					t.Error("text should be 'Negative'")
				}
			},
		},
		{
			name:  "valueDateTime",
			input: `{"resourceType":"Observation","status":"final","code":{"text":"test"},"valueDateTime":"2023-01-15T10:30:00Z"}`,
			check: func(t *testing.T, obs resources.Observation) {
				if obs.Value == nil || obs.Value.DateTime == nil || *obs.Value.DateTime != "2023-01-15T10:30:00Z" {
					t.Error("expected dateTime")
				}
			},
		},
		{
			name:  "valuePeriod",
			input: `{"resourceType":"Observation","status":"final","code":{"text":"test"},"valuePeriod":{"start":"2023-01-01","end":"2023-12-31"}}`,
			check: func(t *testing.T, obs resources.Observation) {
				if obs.Value == nil || obs.Value.Period == nil {
					t.Fatal("expected Period")
				}
				if obs.Value.Period.Start == nil {
					t.Error("period.start should be set")
				}
				if obs.Value.Period.End == nil {
					t.Error("period.end should be set")
				}
			},
		},
		{
			name:  "valueRange",
			input: `{"resourceType":"Observation","status":"final","code":{"text":"test"},"valueRange":{"low":{"value":3.5},"high":{"value":5.5}}}`,
			check: func(t *testing.T, obs resources.Observation) {
				if obs.Value == nil || obs.Value.Range == nil {
					t.Fatal("expected Range")
				}
				if obs.Value.Range.Low == nil || obs.Value.Range.Low.Value == nil {
					t.Error("range.low should be set")
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

			// Round-trip marshal→unmarshal
			out, err := json.Marshal(&obs)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var obs2 resources.Observation
			if err := json.Unmarshal(out, &obs2); err != nil {
				t.Fatalf("re-unmarshal: %v", err)
			}
			tt.check(t, obs2)

			// Verify the correct JSON key is present
			var m map[string]json.RawMessage
			json.Unmarshal(out, &m)
			var keyName string
			if strings.Contains(tt.name, "valueBoolean") {
				keyName = "valueBoolean"
			} else if strings.Contains(tt.name, "valueInteger") {
				keyName = "valueInteger"
			} else {
				keyName = tt.name
			}
			if _, ok := m[keyName]; !ok {
				t.Errorf("expected key %q in marshaled JSON", keyName)
			}
		})
	}
}

// Test Patient.deceased[x] with DateTime variant
func TestPatientDeceasedDateTime(t *testing.T) {
	input := `{"resourceType":"Patient","deceasedDateTime":"2023-06-15"}`
	var p resources.Patient
	if err := json.Unmarshal([]byte(input), &p); err != nil {
		t.Fatal(err)
	}
	if p.Deceased == nil || p.Deceased.DateTime == nil {
		t.Fatal("deceasedDateTime should be parsed")
	}
	if *p.Deceased.DateTime != "2023-06-15" {
		t.Errorf("got %q, want 2023-06-15", *p.Deceased.DateTime)
	}

	out, _ := json.Marshal(&p)
	var m map[string]json.RawMessage
	json.Unmarshal(out, &m)
	if _, ok := m["deceasedDateTime"]; !ok {
		t.Error("deceasedDateTime should survive round-trip")
	}
	if _, ok := m["deceasedBoolean"]; ok {
		t.Error("deceasedBoolean should NOT be present")
	}
}

// Test Patient.multipleBirth[x]
func TestPatientMultipleBirth(t *testing.T) {
	input := `{"resourceType":"Patient","multipleBirthInteger":3}`
	var p resources.Patient
	if err := json.Unmarshal([]byte(input), &p); err != nil {
		t.Fatal(err)
	}
	if p.MultipleBirth == nil || p.MultipleBirth.Integer == nil {
		t.Fatal("multipleBirthInteger should be parsed")
	}
	if *p.MultipleBirth.Integer != 3 {
		t.Errorf("got %v, want 3", *p.MultipleBirth.Integer)
	}
}

// Test Condition.onset[x] with Age variant
func TestConditionOnsetAge(t *testing.T) {
	input := `{"resourceType":"Condition","subject":{"reference":"Patient/1"},"onsetAge":{"value":52,"unit":"years","system":"http://unitsofmeasure.org","code":"a"}}`
	var c resources.Condition
	if err := json.Unmarshal([]byte(input), &c); err != nil {
		t.Fatal(err)
	}
	if c.Onset == nil {
		t.Fatal("onset should not be nil")
	}

	out, _ := json.Marshal(&c)
	var m map[string]json.RawMessage
	json.Unmarshal(out, &m)
	if _, ok := m["onsetAge"]; !ok {
		t.Error("onsetAge should survive round-trip")
	}
}

// ============================================================================
// Extension edge cases
// ============================================================================

func TestNestedExtensionRoundTrip(t *testing.T) {
	input := `{
		"resourceType": "Patient",
		"extension": [{
			"url": "http://example.org/fhir/StructureDefinition/complex-ext",
			"extension": [{
				"url": "level",
				"valueCode": "VIP"
			}, {
				"url": "reason",
				"valueString": "Donor"
			}, {
				"url": "score",
				"valueInteger": 42
			}, {
				"url": "active",
				"valueBoolean": true
			}]
		}]
	}`

	var patient resources.Patient
	if err := json.Unmarshal([]byte(input), &patient); err != nil {
		t.Fatal(err)
	}

	if len(patient.Extension) != 1 {
		t.Fatalf("expected 1 extension, got %d", len(patient.Extension))
	}
	ext := patient.Extension[0]
	if len(ext.Extension) != 4 {
		t.Fatalf("expected 4 nested extensions, got %d", len(ext.Extension))
	}

	// Verify each nested extension value type
	if ext.Extension[0].ValueCode == nil || string(*ext.Extension[0].ValueCode) != "VIP" {
		t.Error("ext[0] valueCode should be VIP")
	}
	if ext.Extension[1].ValueString == nil || *ext.Extension[1].ValueString != "Donor" {
		t.Error("ext[1] valueString should be Donor")
	}
	if ext.Extension[2].ValueInteger == nil || *ext.Extension[2].ValueInteger != 42 {
		t.Error("ext[2] valueInteger should be 42")
	}
	if ext.Extension[3].ValueBoolean == nil || !*ext.Extension[3].ValueBoolean {
		t.Error("ext[3] valueBoolean should be true")
	}

	// Full round-trip
	out, err := json.Marshal(&patient)
	if err != nil {
		t.Fatal(err)
	}
	var reparsed resources.Patient
	if err := json.Unmarshal(out, &reparsed); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(patient.Extension, reparsed.Extension) {
		t.Error("extensions should be identical after round-trip")
	}
}

func TestExtensionOnDatatypes(t *testing.T) {
	// Extensions can appear on complex types, not just resources
	input := `{
		"resourceType": "Patient",
		"name": [{
			"family": "Smith",
			"extension": [{
				"url": "http://example.org/maiden-name",
				"valueString": "Johnson"
			}]
		}]
	}`

	var p resources.Patient
	if err := json.Unmarshal([]byte(input), &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Name) != 1 || len(p.Name[0].Extension) != 1 {
		t.Fatal("name should have one extension")
	}
	if p.Name[0].Extension[0].ValueString == nil || *p.Name[0].Extension[0].ValueString != "Johnson" {
		t.Error("extension value should be Johnson")
	}
}

// ============================================================================
// Serialization edge cases
// ============================================================================

func TestEmptyResourceMarshal(t *testing.T) {
	p, _ := resources.NewPatient().Build()
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]json.RawMessage
	json.Unmarshal(data, &m)

	// Only resourceType should be present
	if _, ok := m["resourceType"]; !ok {
		t.Error("resourceType must always be present")
	}
	for _, field := range []string{"name", "gender", "birthDate", "address", "telecom", "identifier"} {
		if _, ok := m[field]; ok {
			t.Errorf("empty optional field %q should be omitted", field)
		}
	}
}

func TestBooleanFalseNotOmitted(t *testing.T) {
	// deceasedBoolean: false must NOT be omitted (it's meaningful)
	input := `{"resourceType":"Patient","deceasedBoolean":false}`
	var p resources.Patient
	if err := json.Unmarshal([]byte(input), &p); err != nil {
		t.Fatal(err)
	}
	if p.Deceased == nil || p.Deceased.Boolean == nil {
		t.Fatal("deceasedBoolean:false should be parsed")
	}
	if *p.Deceased.Boolean != false {
		t.Error("should be false")
	}

	out, _ := json.Marshal(&p)
	var m map[string]json.RawMessage
	json.Unmarshal(out, &m)
	if _, ok := m["deceasedBoolean"]; !ok {
		t.Error("deceasedBoolean:false must NOT be omitted from JSON")
	}
}

func TestZeroIntegerNotOmitted(t *testing.T) {
	input := `{"resourceType":"Observation","status":"final","code":{"text":"t"},"valueInteger":0}`
	var obs resources.Observation
	if err := json.Unmarshal([]byte(input), &obs); err != nil {
		t.Fatal(err)
	}

	out, _ := json.Marshal(&obs)
	var m map[string]json.RawMessage
	json.Unmarshal(out, &m)
	if _, ok := m["valueInteger"]; !ok {
		t.Error("valueInteger:0 must NOT be omitted from JSON")
	}
}

func TestEmptyStringFieldOmitted(t *testing.T) {
	// A Patient with no fields set should not produce empty string fields
	p := resources.Patient{ResourceType: "Patient"}
	data, _ := json.Marshal(&p)
	if strings.Contains(string(data), `"gender":""`) {
		t.Error("empty gender should not appear in JSON")
	}
}

// ============================================================================
// Negative / malformed input tests
// ============================================================================

func TestUnknownFieldsPreserved(t *testing.T) {
	input := `{
		"resourceType": "Patient",
		"id": "test",
		"gender": "male",
		"customField": "hello",
		"anotherUnknown": {"nested": true}
	}`

	var patient resources.Patient
	if err := json.Unmarshal([]byte(input), &patient); err != nil {
		t.Fatal(err)
	}

	// Known fields parsed normally
	if patient.Gender == nil || string(*patient.Gender) != "male" {
		t.Error("gender should be male")
	}

	// Unknown fields captured in Extra
	if len(patient.Extra) != 2 {
		t.Fatalf("expected 2 extra fields, got %d: %v", len(patient.Extra), patient.Extra)
	}
	if string(patient.Extra["customField"]) != `"hello"` {
		t.Errorf("customField should be \"hello\", got %s", patient.Extra["customField"])
	}

	// Round-trip preserves unknown fields
	out, err := json.Marshal(&patient)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	json.Unmarshal(out, &m)
	if _, ok := m["customField"]; !ok {
		t.Error("customField should survive round-trip")
	}
	if _, ok := m["anotherUnknown"]; !ok {
		t.Error("anotherUnknown should survive round-trip")
	}
}

func TestInvalidJSON(t *testing.T) {
	var p resources.Patient
	err := json.Unmarshal([]byte(`{not valid json}`), &p)
	if err == nil {
		t.Error("should fail on invalid JSON")
	}
}

func TestEmptyJSON(t *testing.T) {
	var p resources.Patient
	err := json.Unmarshal([]byte(`{}`), &p)
	if err != nil {
		t.Fatalf("empty object should parse: %v", err)
	}
	if p.ResourceType != "" {
		t.Error("resourceType should be empty for empty input")
	}
}

func TestNullFields(t *testing.T) {
	input := `{"resourceType":"Patient","gender":null,"birthDate":null,"name":null}`
	var p resources.Patient
	if err := json.Unmarshal([]byte(input), &p); err != nil {
		t.Fatalf("null fields should parse: %v", err)
	}
	if p.Gender != nil {
		t.Error("null gender should be nil")
	}
	if p.BirthDate != nil {
		t.Error("null birthDate should be nil")
	}
	if p.Name != nil {
		t.Error("null name should be nil")
	}
}

// ============================================================================
// Reference handling
// ============================================================================

func TestReferenceWithIdentifier(t *testing.T) {
	input := `{
		"resourceType": "Observation",
		"status": "final",
		"code": {"text": "test"},
		"subject": {
			"reference": "Patient/123",
			"type": "Patient",
			"display": "John Doe",
			"identifier": {
				"system": "http://example.org/mrn",
				"value": "MRN-123"
			}
		}
	}`

	var obs resources.Observation
	if err := json.Unmarshal([]byte(input), &obs); err != nil {
		t.Fatal(err)
	}
	if obs.Subject == nil {
		t.Fatal("subject should be set")
	}
	if obs.Subject.Reference == nil || *obs.Subject.Reference != "Patient/123" {
		t.Error("reference mismatch")
	}
	if obs.Subject.Display == nil || *obs.Subject.Display != "John Doe" {
		t.Error("display mismatch")
	}
	if obs.Subject.Identifier == nil || obs.Subject.Identifier.Value == nil || *obs.Subject.Identifier.Value != "MRN-123" {
		t.Error("identifier.value mismatch")
	}
	if obs.Subject.Type == nil || string(*obs.Subject.Type) != "Patient" {
		t.Error("type mismatch")
	}

	// Round-trip
	out, _ := json.Marshal(&obs)
	var obs2 resources.Observation
	json.Unmarshal(out, &obs2)
	if obs2.Subject.Identifier == nil || *obs2.Subject.Identifier.Value != "MRN-123" {
		t.Error("identifier should survive round-trip")
	}
}

// ============================================================================
// Builder tests
// ============================================================================

func TestPatientBuilder(t *testing.T) {
	p, err := resources.NewPatient().
		WithName("John", "Doe").
		WithBirthDate("1980-03-15").
		WithGender(resources.AdministrativeGenderMale).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	// Verify AND round-trip
	data, _ := json.Marshal(p)
	var reparsed resources.Patient
	json.Unmarshal(data, &reparsed)

	if reparsed.Gender == nil || string(*reparsed.Gender) != "male" {
		t.Error("gender should survive builder→marshal→unmarshal")
	}
	if reparsed.BirthDate == nil || string(*reparsed.BirthDate) != "1980-03-15" {
		t.Error("birthDate should survive round-trip")
	}
	if len(reparsed.Name) != 1 || *reparsed.Name[0].Family != "Doe" {
		t.Error("name should survive round-trip")
	}
}

func TestObservationBuilderWithEnum(t *testing.T) {
	obs, err := resources.NewObservation().
		WithStatus(resources.ObservationStatusFinal).
		WithCode("http://loinc.org", "85354-9", "Blood pressure").
		WithSubject("Patient/example").
		Build()
	if err != nil {
		t.Fatal(err)
	}
	if *obs.Status != resources.ObservationStatusFinal {
		t.Error("status should be ObservationStatusFinal")
	}
}

func TestBuildFailsOnMissingRequired(t *testing.T) {
	// Observation requires "code" (1..1)
	_, err := resources.NewObservation().
		WithStatus(resources.ObservationStatusFinal).
		Build()
	if err == nil {
		t.Error("Observation.Build() should fail when 'code' is not set")
	}
	if err != nil && !strings.Contains(err.Error(), "code") {
		t.Errorf("error should mention 'code', got: %v", err)
	}

	// Setting code should make it pass
	_, err = resources.NewObservation().
		WithStatus(resources.ObservationStatusFinal).
		WithCode("http://loinc.org", "1234-5", "Test").
		Build()
	if err != nil {
		t.Errorf("should succeed with code set: %v", err)
	}

	// Condition requires "subject" (1..1)
	_, err = resources.NewCondition().
		WithCode("http://snomed.info/sct", "386661006", "Fever").
		Build()
	if err == nil {
		t.Error("Condition.Build() should fail when 'subject' is not set")
	}

	// Setting subject should make it pass
	_, err = resources.NewCondition().
		WithCode("http://snomed.info/sct", "386661006", "Fever").
		WithSubject("Patient/1").
		Build()
	if err != nil {
		t.Errorf("should succeed with subject set: %v", err)
	}
}

func TestMedicationRequestBuilderWithCode(t *testing.T) {
	mr, err := resources.NewMedicationRequest().
		WithStatus(dt.Code("active")).
		WithIntent(dt.Code("order")).
		WithSubject("Patient/example").
		WithMedicationCodeableConcept("http://www.nlm.nih.gov/research/umls/rxnorm", "1049502", "Acetaminophen").
		Build()
	if err != nil {
		t.Fatal(err)
	}

	// Round-trip and verify medication survived
	data, _ := json.Marshal(mr)
	var m map[string]json.RawMessage
	json.Unmarshal(data, &m)
	if _, ok := m["medicationCodeableConcept"]; !ok {
		t.Error("medicationCodeableConcept should be in JSON")
	}
}

// ============================================================================
// ParseInstant tests
// ============================================================================

func TestParseInstant(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"2023-01-15T10:30:00Z", false},
		{"2023-01-15T10:30:00+05:00", false},
		{"2023-01-15T10:30:00-08:00", false},
		{"2023-01-15T10:30:00.123Z", false},
		{"2023-01-15T10:30:00.123456789Z", false},
		{"not-a-date", true},
		{"2023-01-15", true},
		{"2023-01-15T10:30:00", true}, // missing timezone
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := dt.ParseInstant(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseInstant(%q) err=%v, wantErr=%v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && result.IsZero() {
				t.Error("successful parse should not return zero time")
			}
		})
	}
}

// ============================================================================
// Deep backbone element test
// ============================================================================

func TestMedicationRequestBackboneElements(t *testing.T) {
	// Test that deeply nested backbone elements (dispenseRequest, dosageInstruction) parse
	data := loadTestData(t, "medicationrequest0301.json")
	var mr resources.MedicationRequest
	if err := json.Unmarshal(data, &mr); err != nil {
		t.Fatal(err)
	}

	if mr.DispenseRequest == nil {
		t.Fatal("dispenseRequest should be present")
	}
	if len(mr.DosageInstruction) == 0 {
		t.Fatal("dosageInstruction should be present")
	}

	// Verify dosage has nested structure
	dosage := mr.DosageInstruction[0]
	if dosage.Text == nil {
		t.Error("dosage.text should be present")
	}
}

// ============================================================================
// Element extension tests (_field companions)
// ============================================================================

func TestElementExtensionRoundTrip(t *testing.T) {
	// _birthDate carries an extension on the birthDate primitive
	input := `{
		"resourceType": "Patient",
		"id": "elem-ext-test",
		"birthDate": "1974-12-25",
		"_birthDate": {
			"extension": [{
				"url": "http://hl7.org/fhir/StructureDefinition/patient-birthTime",
				"valueDateTime": "1974-12-25T14:35:45-05:00"
			}]
		}
	}`

	var patient resources.Patient
	if err := json.Unmarshal([]byte(input), &patient); err != nil {
		t.Fatal(err)
	}

	// Verify the value parsed
	if patient.BirthDate == nil || string(*patient.BirthDate) != "1974-12-25" {
		t.Error("birthDate value should be 1974-12-25")
	}

	// Verify the element extension parsed
	if patient.BirthDateElement == nil {
		t.Fatal("_birthDate element should be present")
	}
	if len(patient.BirthDateElement.Extension) != 1 {
		t.Fatalf("_birthDate should have 1 extension, got %d", len(patient.BirthDateElement.Extension))
	}
	ext := patient.BirthDateElement.Extension[0]
	if string(ext.Url) != "http://hl7.org/fhir/StructureDefinition/patient-birthTime" {
		t.Error("extension URL mismatch")
	}
	if ext.ValueDateTime == nil || string(*ext.ValueDateTime) != "1974-12-25T14:35:45-05:00" {
		t.Error("extension valueDateTime mismatch")
	}

	// Round-trip
	out, err := json.Marshal(&patient)
	if err != nil {
		t.Fatal(err)
	}

	// Verify _birthDate is in the output JSON
	var m map[string]json.RawMessage
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["_birthDate"]; !ok {
		t.Error("_birthDate should be present in marshaled JSON")
	}
	if _, ok := m["birthDate"]; !ok {
		t.Error("birthDate should be present in marshaled JSON")
	}

	// Re-parse and verify extension survived
	var reparsed resources.Patient
	if err := json.Unmarshal(out, &reparsed); err != nil {
		t.Fatal(err)
	}
	if reparsed.BirthDateElement == nil || len(reparsed.BirthDateElement.Extension) != 1 {
		t.Error("_birthDate extension should survive round-trip")
	}
}

func TestElementExtensionWithId(t *testing.T) {
	input := `{
		"resourceType": "Patient",
		"gender": "male",
		"_gender": {
			"id": "gender-element-1",
			"extension": [{
				"url": "http://example.org/original-coding",
				"valueCode": "M"
			}]
		}
	}`

	var patient resources.Patient
	if err := json.Unmarshal([]byte(input), &patient); err != nil {
		t.Fatal(err)
	}

	if patient.GenderElement == nil {
		t.Fatal("_gender element should be present")
	}
	if patient.GenderElement.Id == nil || *patient.GenderElement.Id != "gender-element-1" {
		t.Error("element id should be gender-element-1")
	}
	if len(patient.GenderElement.Extension) != 1 {
		t.Fatal("should have 1 extension")
	}

	// Round-trip
	out, _ := json.Marshal(&patient)
	var reparsed resources.Patient
	json.Unmarshal(out, &reparsed)
	if reparsed.GenderElement == nil || reparsed.GenderElement.Id == nil || *reparsed.GenderElement.Id != "gender-element-1" {
		t.Error("element id should survive round-trip")
	}
}

func TestHL7PatientElementExtensions(t *testing.T) {
	// The official HL7 patient example has _birthDate — verify it round-trips
	// WITHOUT stripping underscore keys
	data := loadTestData(t, "patient-example.json")

	var patient resources.Patient
	if err := json.Unmarshal(data, &patient); err != nil {
		t.Fatal(err)
	}

	// Verify _birthDate was parsed
	if patient.BirthDateElement == nil {
		t.Fatal("_birthDate from HL7 example should be parsed")
	}
	if len(patient.BirthDateElement.Extension) == 0 {
		t.Fatal("_birthDate should have extensions")
	}

	// Marshal and verify _birthDate is in output
	out, err := json.Marshal(&patient)
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]json.RawMessage
	json.Unmarshal(out, &m)
	if _, ok := m["_birthDate"]; !ok {
		t.Error("_birthDate should be present in marshaled output of HL7 example")
	}
}

// ============================================================================
// Multiple value[x] variants — verify both are preserved
// ============================================================================

func TestMultipleValueXVariantsPreserved(t *testing.T) {
	// When input has both valueQuantity and valueString, both should be parsed
	input := `{"resourceType":"Observation","status":"final","code":{"text":"test"},"valueQuantity":{"value":120,"unit":"mmHg"},"valueString":"also present"}`
	var obs resources.Observation
	if err := json.Unmarshal([]byte(input), &obs); err != nil {
		t.Fatal(err)
	}
	if obs.Value == nil {
		t.Fatal("value should not be nil")
	}
	if obs.Value.Quantity == nil {
		t.Error("valueQuantity should be parsed")
	}
	if obs.Value.String == nil || *obs.Value.String != "also present" {
		t.Error("valueString should also be parsed")
	}
}

// ============================================================================
// Round-trip identity — marshal→unmarshal produces equivalent resource
// ============================================================================

func TestRoundTripIdentity(t *testing.T) {
	tests := []string{
		`{"resourceType":"Patient","id":"1","gender":"male","birthDate":"1980-01-01","name":[{"family":"Doe","given":["John","James"]}],"active":true}`,
		`{"resourceType":"Observation","id":"2","status":"final","code":{"coding":[{"system":"http://loinc.org","code":"8867-4","display":"Heart rate"}],"text":"Heart rate"},"valueQuantity":{"value":72,"unit":"bpm"}}`,
		`{"resourceType":"Condition","id":"3","code":{"text":"Diabetes"},"subject":{"reference":"Patient/1"}}`,
	}
	for _, input := range tests {
		// Parse
		var m1 map[string]json.RawMessage
		json.Unmarshal([]byte(input), &m1)
		rt := string(m1["resourceType"])

		res, err := resources.ParseResource(json.RawMessage(input))
		if err != nil {
			t.Fatalf("parse %s: %v", rt, err)
		}

		// Marshal back
		out, err := json.Marshal(res)
		if err != nil {
			t.Fatalf("marshal %s: %v", rt, err)
		}

		// Compare key-by-key
		var m2 map[string]json.RawMessage
		json.Unmarshal(out, &m2)
		for k := range m1 {
			if _, ok := m2[k]; !ok {
				t.Errorf("%s: key %q lost in round-trip", rt, k)
			}
		}
	}
}

// ============================================================================
// Negative tests — malformed and edge-case inputs
// ============================================================================

func TestUnmarshalInvalidJSON(t *testing.T) {
	var p resources.Patient
	if err := json.Unmarshal([]byte(`{not valid json}`), &p); err == nil {
		t.Error("should fail on invalid JSON")
	}
}

func TestUnmarshalWrongTypeForField(t *testing.T) {
	// gender should be string, not number
	input := `{"resourceType":"Patient","gender":123}`
	var p resources.Patient
	err := json.Unmarshal([]byte(input), &p)
	if err == nil {
		// Go's json.Unmarshal is lenient about this — it silently fails
		// Verify the field is nil (not corrupted)
		if p.Gender != nil {
			t.Errorf("gender should be nil when given wrong type, got %v", *p.Gender)
		}
	}
}

func TestUnmarshalNullFields(t *testing.T) {
	input := `{"resourceType":"Patient","id":null,"gender":null,"name":null}`
	var p resources.Patient
	if err := json.Unmarshal([]byte(input), &p); err != nil {
		t.Fatalf("null fields should be accepted: %v", err)
	}
	if p.Id != nil {
		t.Error("null id should result in nil")
	}
	if p.Gender != nil {
		t.Error("null gender should result in nil")
	}
	if p.Name != nil {
		t.Error("null name should result in nil slice")
	}
}

func TestEmptyArrayVsNilArray(t *testing.T) {
	// Empty array should marshal differently from absent array
	input1 := `{"resourceType":"Patient"}`
	input2 := `{"resourceType":"Patient","name":[]}`

	var p1, p2 resources.Patient
	json.Unmarshal([]byte(input1), &p1)
	json.Unmarshal([]byte(input2), &p2)

	out1, _ := json.Marshal(&p1)
	out2, _ := json.Marshal(&p2)

	// Both should omit empty name (omitempty on slices)
	var m1, m2 map[string]json.RawMessage
	json.Unmarshal(out1, &m1)
	json.Unmarshal(out2, &m2)

	if _, ok := m1["name"]; ok {
		t.Error("absent name should not appear in output")
	}
	if _, ok := m2["name"]; ok {
		t.Error("empty name array should be omitted (omitempty)")
	}
}

func TestBooleanFalsePreserved(t *testing.T) {
	// false booleans should survive round-trip (not be omitted as zero value)
	input := `{"resourceType":"Patient","id":"1","active":false}`
	var p resources.Patient
	json.Unmarshal([]byte(input), &p)

	if p.Active == nil {
		t.Fatal("active:false should not be nil")
	}
	if *p.Active != false {
		t.Error("active should be false")
	}

	out, _ := json.Marshal(&p)
	var m map[string]json.RawMessage
	json.Unmarshal(out, &m)
	if _, ok := m["active"]; !ok {
		t.Error("active:false should be preserved in output")
	}
}

func TestZeroIntegerPreserved(t *testing.T) {
	input := `{"resourceType":"Observation","status":"final","code":{"text":"t"},"valueInteger":0}`
	var obs resources.Observation
	json.Unmarshal([]byte(input), &obs)

	if obs.Value == nil || obs.Value.Integer == nil {
		t.Fatal("valueInteger:0 should be parsed")
	}
	if *obs.Value.Integer != 0 {
		t.Error("valueInteger should be 0")
	}

	// Round-trip
	out, _ := json.Marshal(&obs)
	if !strings.Contains(string(out), `"valueInteger":0`) {
		t.Error("valueInteger:0 should survive round-trip")
	}
}
