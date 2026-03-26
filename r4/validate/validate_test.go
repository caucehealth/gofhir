// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	"encoding/json"
	"strings"
	"testing"

	dt "github.com/caucehealth/gofhir/r4/datatypes"
	"github.com/caucehealth/gofhir/r4/resources"
	"github.com/caucehealth/gofhir/r4/validate"
)

func TestValidateValidPatient(t *testing.T) {
	p, _ := resources.NewPatient().
		WithName("John", "Doe").
		WithGender(resources.AdministrativeGenderMale).
		Build()

	v := validate.New()
	result := v.Validate(p)
	if result.HasErrors() {
		for _, issue := range result.Errors() {
			t.Errorf("unexpected error: %s: %s", issue.Path, issue.Message)
		}
	}
}

func TestValidateRequiredFieldMissing(t *testing.T) {
	// Observation requires "code" per JSON schema
	obs := &resources.Observation{ResourceType: "Observation"}

	v := validate.New()
	result := v.Validate(obs)

	if !result.HasErrors() {
		t.Fatal("should have errors for missing required code")
	}

	found := false
	for _, issue := range result.Errors() {
		if strings.Contains(issue.Path, "code") && issue.Code == validate.CodeRequired {
			found = true
		}
	}
	if !found {
		t.Error("should have required-field error for Observation.code")
	}
}

func TestValidateRequiredFieldPresent(t *testing.T) {
	obs, _ := resources.NewObservation().
		WithStatus(resources.ObservationStatusFinal).
		WithCode("http://loinc.org", "1234", "Test").
		Build()

	v := validate.New()
	result := v.Validate(obs)

	for _, issue := range result.Errors() {
		if issue.Code == validate.CodeRequired {
			t.Errorf("unexpected required-field error: %s", issue.Message)
		}
	}
}

func TestValidateEnumBinding(t *testing.T) {
	// Set an invalid gender value
	badGender := resources.AdministrativeGender("invalid-gender")
	p := &resources.Patient{
		ResourceType: "Patient",
		Gender:       &badGender,
	}

	v := validate.New()
	result := v.Validate(p)

	found := false
	for _, issue := range result.Issues {
		if issue.Code == validate.CodeCodeInvalid && strings.Contains(issue.Path, "gender") {
			found = true
		}
	}
	if !found {
		t.Error("should reject invalid gender enum value")
	}
}

func TestValidateEnumBindingValid(t *testing.T) {
	gender := resources.AdministrativeGenderFemale
	p := &resources.Patient{
		ResourceType: "Patient",
		Gender:       &gender,
	}

	v := validate.New()
	result := v.Validate(p)

	for _, issue := range result.Issues {
		if issue.Code == validate.CodeCodeInvalid {
			t.Errorf("valid gender should not trigger enum error: %s", issue.Message)
		}
	}
}

func TestValidateIDLength(t *testing.T) {
	longID := dt.ID(strings.Repeat("x", 100))
	p := &resources.Patient{
		ResourceType: "Patient",
		Id:           &longID,
	}

	v := validate.New()
	result := v.Validate(p)

	found := false
	for _, issue := range result.Issues {
		if issue.Code == validate.CodeValue && strings.Contains(issue.Message, "64") {
			found = true
		}
	}
	if !found {
		t.Error("should reject ID longer than 64 characters")
	}
}

func TestValidateToOperationOutcome(t *testing.T) {
	obs := &resources.Observation{ResourceType: "Observation"}

	v := validate.New()
	result := v.Validate(obs)
	oo := result.ToOperationOutcome()

	if oo.ResourceType != "OperationOutcome" {
		t.Error("should produce OperationOutcome")
	}
	if len(oo.Issue) == 0 {
		t.Error("should have issues")
	}

	// Verify it's valid JSON
	data, err := json.Marshal(oo)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "required") {
		t.Error("should contain issue code")
	}
}

func TestValidateJSON(t *testing.T) {
	// Valid
	result, err := validate.ValidateJSON(json.RawMessage(`{"resourceType":"Patient","id":"1"}`))
	if err != nil {
		t.Fatal(err)
	}
	if result.HasErrors() {
		t.Error("valid patient should not have errors")
	}

	// Invalid — missing required field
	result2, err := validate.ValidateJSON(json.RawMessage(`{"resourceType":"Observation"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !result2.HasErrors() {
		t.Error("observation without code should have errors")
	}
}

func TestValidateCustomRule(t *testing.T) {
	// Custom rule: Patient must have at least one name
	nameRule := validate.RuleFunc(func(r resources.Resource) []validate.Issue {
		p, ok := r.(*resources.Patient)
		if !ok {
			return nil
		}
		if len(p.Name) == 0 {
			return []validate.Issue{{
				Severity: validate.SeverityWarning,
				Code:     validate.CodeInvariant,
				Path:     "Patient.name",
				Message:  "Patient should have at least one name",
			}}
		}
		return nil
	})

	p := &resources.Patient{ResourceType: "Patient"}
	v := validate.New(validate.WithRules(nameRule))
	result := v.Validate(p)

	found := false
	for _, issue := range result.Warnings() {
		if strings.Contains(issue.Path, "name") {
			found = true
		}
	}
	if !found {
		t.Error("custom rule should produce warning for missing name")
	}
}

func TestValidateEmptyResultNoErrors(t *testing.T) {
	result := &validate.Result{}
	if result.HasErrors() {
		t.Error("empty result should not have errors")
	}
	if len(result.Errors()) != 0 {
		t.Error("empty result should have no errors")
	}
	if len(result.Warnings()) != 0 {
		t.Error("empty result should have no warnings")
	}
}

func TestValidateMetadataLoaded(t *testing.T) {
	// Verify metadata is registered for key resources
	types := []string{"Patient", "Observation", "Condition", "Encounter", "Practitioner"}
	for _, rt := range types {
		meta := validate.GetResourceMeta(rt)
		if meta == nil {
			t.Errorf("no validation metadata for %s", rt)
			continue
		}
		if len(meta.Fields) == 0 {
			t.Errorf("%s should have fields", rt)
		}
	}

	// Verify total count
	total := 0
	for _, name := range []string{"Account", "Patient", "Observation", "Bundle"} {
		if validate.GetResourceMeta(name) != nil {
			total++
		}
	}
	// Bundle is skipped, so we should have 3 of 4
	if total != 3 {
		t.Errorf("expected 3 of 4 resources to have metadata, got %d", total)
	}
}

func TestValidateMultipleResources(t *testing.T) {
	v := validate.New()

	// Patient — no required fields from schema, should pass
	p := &resources.Patient{ResourceType: "Patient"}
	if v.Validate(p).HasErrors() {
		t.Error("empty Patient should pass (no required fields in schema)")
	}

	// Encounter — class is required
	e := &resources.Encounter{ResourceType: "Encounter"}
	result := v.Validate(e)
	found := false
	for _, issue := range result.Errors() {
		if strings.Contains(issue.Path, "class") {
			found = true
		}
	}
	if !found {
		t.Error("Encounter without class should fail validation")
	}
}

func TestValidateStructureDefCardinality(t *testing.T) {
	// Observation.status is required per StructureDefinition but NOT in JSON schema.
	// Our cardinality augmentation should catch this.
	obs := &resources.Observation{ResourceType: "Observation"}

	v := validate.New()
	result := v.Validate(obs)

	foundStatus := false
	foundCode := false
	for _, issue := range result.Errors() {
		if strings.Contains(issue.Path, "status") && issue.Code == validate.CodeRequired {
			foundStatus = true
		}
		if strings.Contains(issue.Path, "code") && issue.Code == validate.CodeRequired {
			foundCode = true
		}
	}
	if !foundStatus {
		t.Error("Observation.status should be required (from StructureDefinition)")
	}
	if !foundCode {
		t.Error("Observation.code should be required (from JSON schema)")
	}
}

func TestValidatePrimitiveFormats(t *testing.T) {
	// ID too long
	longID := dt.ID(strings.Repeat("a", 65))
	p := &resources.Patient{ResourceType: "Patient", Id: &longID}

	v := validate.New()
	result := v.Validate(p)

	found := false
	for _, issue := range result.Issues {
		if strings.Contains(issue.Message, "64") {
			found = true
		}
	}
	if !found {
		t.Error("ID > 64 chars should be flagged")
	}

	// Valid ID
	goodID := dt.ID("valid-id-123")
	p2 := &resources.Patient{ResourceType: "Patient", Id: &goodID}
	result2 := v.Validate(p2)
	for _, issue := range result2.Issues {
		if strings.Contains(issue.Path, "id") && issue.Code == validate.CodeValue {
			t.Errorf("valid ID should not be flagged: %s", issue.Message)
		}
	}
}

func TestValidateIDPattern(t *testing.T) {
	badID := dt.ID("has spaces!")
	p := &resources.Patient{ResourceType: "Patient", Id: &badID}

	v := validate.New()
	result := v.Validate(p)

	found := false
	for _, issue := range result.Issues {
		if issue.Code == validate.CodeValue && strings.Contains(issue.Message, "pattern") {
			found = true
		}
	}
	if !found {
		t.Error("ID with spaces should fail pattern validation")
	}
}

func TestValidateDateFormat(t *testing.T) {
	tests := []struct {
		date  string
		valid bool
	}{
		{"2024", true},
		{"2024-03", true},
		{"2024-03-15", true},
		{"2024-13-01", true}, // regex allows — semantic date validation is separate
		{"not-a-date", false},
		{"24-03-15", false},
	}

	v := validate.New()
	for _, tt := range tests {
		d := dt.Date(tt.date)
		p := &resources.Patient{ResourceType: "Patient", BirthDate: &d}
		result := v.Validate(p)

		hasDateError := false
		for _, issue := range result.Issues {
			if strings.Contains(issue.Path, "birthDate") && issue.Code == validate.CodeValue {
				hasDateError = true
			}
		}
		if tt.valid && hasDateError {
			t.Errorf("date %q should be valid", tt.date)
		}
		if !tt.valid && !hasDateError {
			t.Errorf("date %q should be invalid", tt.date)
		}
	}
}

func TestValidateCodeNoWhitespace(t *testing.T) {
	badCode := dt.Code("has space")
	p := &resources.Patient{ResourceType: "Patient", Language: &badCode}

	v := validate.New()
	result := v.Validate(p)

	found := false
	for _, issue := range result.Issues {
		if issue.Code == validate.CodeValue && strings.Contains(issue.Message, "whitespace") {
			found = true
		}
	}
	if !found {
		t.Error("code with spaces should fail validation")
	}
}

func TestProfileValidation(t *testing.T) {
	// Define a simple profile: Patient must have name (min=1) and gender (min=1)
	profileJSON := `{
		"resourceType": "StructureDefinition",
		"url": "http://example.org/StructureDefinition/required-patient",
		"name": "RequiredPatient",
		"type": "Patient",
		"snapshot": {
			"element": [
				{"path": "Patient", "min": 0, "max": "*"},
				{"path": "Patient.name", "min": 1, "max": "*", "type": [{"code": "HumanName"}]},
				{"path": "Patient.gender", "min": 1, "max": "1", "type": [{"code": "code"}],
				 "binding": {"strength": "required", "valueSet": "http://hl7.org/fhir/ValueSet/administrative-gender"}},
				{"path": "Patient.identifier", "min": 1, "max": "*", "type": [{"code": "Identifier"}]},
				{"path": "Patient.name.family", "min": 1, "max": "1", "type": [{"code": "string"}],
				 "constraint": [{"key": "custom-1", "expression": "length() > 0", "severity": "error", "human": "family name must not be empty"}]}
			]
		}
	}`

	registry := validate.NewProfileRegistry()
	if err := registry.Load(json.RawMessage(profileJSON)); err != nil {
		t.Fatal(err)
	}

	// Valid patient
	p, _ := resources.NewPatient().
		WithName("John", "Doe").
		WithGender(resources.AdministrativeGenderMale).
		Build()
	id := dt.Identifier{Value: strPtr("12345")}
	p.Identifier = append(p.Identifier, id)

	v := validate.New(validate.WithProfile(registry, "http://example.org/StructureDefinition/required-patient"))
	result := v.Validate(p)
	for _, issue := range result.Errors() {
		if issue.Code == validate.CodeRequired {
			t.Errorf("valid patient should not have required errors: %s", issue.Message)
		}
	}

	// Invalid patient — missing name, gender, identifier
	empty := &resources.Patient{ResourceType: "Patient"}
	result = v.Validate(empty)

	missingFields := map[string]bool{}
	for _, issue := range result.Errors() {
		if issue.Code == validate.CodeRequired {
			missingFields[issue.Path] = true
		}
	}
	if !missingFields["Patient.name"] {
		t.Error("should flag missing name")
	}
	if !missingFields["Patient.gender"] {
		t.Error("should flag missing gender")
	}
	if !missingFields["Patient.identifier"] {
		t.Error("should flag missing identifier")
	}
}

func TestProfileWrongResourceType(t *testing.T) {
	registry := validate.NewProfileRegistry()
	registry.Register(&validate.Profile{
		URL:  "http://example.org/obs-profile",
		Type: "Observation",
	})

	p := &resources.Patient{ResourceType: "Patient"}
	v := validate.New(validate.WithProfile(registry, "http://example.org/obs-profile"))
	result := v.Validate(p)

	found := false
	for _, issue := range result.Issues {
		if strings.Contains(issue.Message, "does not match") {
			found = true
		}
	}
	if !found {
		t.Error("should reject Patient against Observation profile")
	}
}

func strPtr(s string) *string { return &s }
func uriPtr(s string) *dt.URI { v := dt.URI(s); return &v }

func TestFHIRPathInvariants(t *testing.T) {
	// obs-6: value exists or dataAbsentReason exists
	v := validate.New(validate.WithInvariants(map[string]string{
		"obs-6": "value.exists() or dataAbsentReason.exists()",
	}))

	// Observation WITH value — should pass
	input := `{"resourceType":"Observation","status":"final","code":{"text":"test"},"valueQuantity":{"value":120}}`
	var obs resources.Observation
	json.Unmarshal([]byte(input), &obs)
	result := v.Validate(&obs)
	for _, issue := range result.Issues {
		if issue.Code == validate.CodeInvariant && strings.Contains(issue.Message, "obs-6 failed") {
			t.Error("obs with value should pass obs-6")
		}
	}

	// Empty Observation — should fail obs-6
	empty := &resources.Observation{ResourceType: "Observation"}
	result = v.Validate(empty)
	found := false
	for _, issue := range result.Issues {
		if issue.Code == validate.CodeInvariant && strings.Contains(issue.Message, "obs-6") {
			found = true
		}
	}
	if !found {
		t.Error("empty observation should fail obs-6 invariant")
	}
}

func TestSlicingClosedValidation(t *testing.T) {
	// Profile with sliced identifier: SSN (required) and MRN (optional)
	profileJSON := `{
		"resourceType": "StructureDefinition",
		"url": "http://example.org/StructureDefinition/sliced-patient",
		"name": "SlicedPatient",
		"type": "Patient",
		"snapshot": {
			"element": [
				{"path": "Patient", "min": 0, "max": "*"},
				{"path": "Patient.identifier", "min": 1, "max": "*",
				 "slicing": {
					"discriminator": [{"type": "value", "path": "system"}],
					"rules": "closed"
				 }
				},
				{"path": "Patient.identifier", "sliceName": "SSN", "min": 1, "max": "1",
				 "fixedValue": "http://hl7.org/fhir/sid/us-ssn",
				 "type": [{"code": "Identifier"}]
				},
				{"path": "Patient.identifier", "sliceName": "MRN", "min": 0, "max": "1",
				 "fixedValue": "http://hospital.example.org/mrn",
				 "type": [{"code": "Identifier"}]
				}
			]
		}
	}`

	registry := validate.NewProfileRegistry()
	if err := registry.Load(json.RawMessage(profileJSON)); err != nil {
		t.Fatal(err)
	}

	v := validate.New(validate.WithProfile(registry, "http://example.org/StructureDefinition/sliced-patient"))

	// Patient with SSN — should pass
	p := &resources.Patient{ResourceType: "Patient"}
	p.Identifier = append(p.Identifier, dt.Identifier{
		System: uriPtr("http://hl7.org/fhir/sid/us-ssn"),
		Value:  strPtr("123-45-6789"),
	})

	result := v.Validate(p)
	for _, issue := range result.Errors() {
		if strings.Contains(issue.Path, "SSN") && issue.Code == validate.CodeRequired {
			t.Errorf("patient with SSN should not have SSN required error: %s", issue.Message)
		}
	}

	// Patient without identifiers — should fail (SSN min=1)
	empty := &resources.Patient{ResourceType: "Patient"}
	result = v.Validate(empty)
	foundSSN := false
	for _, issue := range result.Errors() {
		if strings.Contains(issue.Message, "SSN") {
			foundSSN = true
		}
	}
	if !foundSSN {
		t.Error("patient without identifiers should fail SSN slice requirement")
	}
}

func TestSlicingOpenValidation(t *testing.T) {
	// Open slicing — unmatched elements are allowed
	profileJSON := `{
		"resourceType": "StructureDefinition",
		"url": "http://example.org/StructureDefinition/open-sliced",
		"name": "OpenSlicedPatient",
		"type": "Patient",
		"snapshot": {
			"element": [
				{"path": "Patient", "min": 0, "max": "*"},
				{"path": "Patient.identifier", "min": 0, "max": "*",
				 "slicing": {
					"discriminator": [{"type": "value", "path": "system"}],
					"rules": "open"
				 }
				},
				{"path": "Patient.identifier", "sliceName": "MRN", "min": 0, "max": "1",
				 "fixedValue": "http://hospital.example.org/mrn",
				 "type": [{"code": "Identifier"}]
				}
			]
		}
	}`

	registry := validate.NewProfileRegistry()
	if err := registry.Load(json.RawMessage(profileJSON)); err != nil {
		t.Fatal(err)
	}

	v := validate.New(validate.WithProfile(registry, "http://example.org/StructureDefinition/open-sliced"))

	// Patient with unmatched identifier — should pass (open slicing)
	p := &resources.Patient{ResourceType: "Patient"}
	p.Identifier = append(p.Identifier, dt.Identifier{
		System: uriPtr("http://other.org/id"),
		Value:  strPtr("XYZ"),
	})

	result := v.Validate(p)
	for _, issue := range result.Errors() {
		if strings.Contains(issue.Message, "does not match any defined slice") {
			t.Errorf("open slicing should allow unmatched elements: %s", issue.Message)
		}
	}
}

func TestSlicingMaxExceeded(t *testing.T) {
	profileJSON := `{
		"resourceType": "StructureDefinition",
		"url": "http://example.org/StructureDefinition/max-sliced",
		"name": "MaxSlicedPatient",
		"type": "Patient",
		"snapshot": {
			"element": [
				{"path": "Patient", "min": 0, "max": "*"},
				{"path": "Patient.identifier", "min": 0, "max": "*",
				 "slicing": {
					"discriminator": [{"type": "value", "path": "system"}],
					"rules": "open"
				 }
				},
				{"path": "Patient.identifier", "sliceName": "SSN", "min": 0, "max": "1",
				 "fixedValue": "http://hl7.org/fhir/sid/us-ssn",
				 "type": [{"code": "Identifier"}]
				}
			]
		}
	}`

	registry := validate.NewProfileRegistry()
	if err := registry.Load(json.RawMessage(profileJSON)); err != nil {
		t.Fatal(err)
	}

	v := validate.New(validate.WithProfile(registry, "http://example.org/StructureDefinition/max-sliced"))

	// Patient with 2 SSNs — should fail (max=1)
	p := &resources.Patient{ResourceType: "Patient"}
	p.Identifier = append(p.Identifier,
		dt.Identifier{System: uriPtr("http://hl7.org/fhir/sid/us-ssn"), Value: strPtr("111-11-1111")},
		dt.Identifier{System: uriPtr("http://hl7.org/fhir/sid/us-ssn"), Value: strPtr("222-22-2222")},
	)

	result := v.Validate(p)
	foundMax := false
	for _, issue := range result.Errors() {
		if strings.Contains(issue.Message, "max") && strings.Contains(issue.Message, "SSN") {
			foundMax = true
		}
	}
	if !foundMax {
		t.Error("two SSN identifiers should exceed max=1 for SSN slice")
	}
}
