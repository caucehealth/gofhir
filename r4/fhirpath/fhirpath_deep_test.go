// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package fhirpath_test

import (
	"encoding/json"
	"testing"

	"github.com/caucehealth/gofhir/r4/fhirpath"
	"github.com/caucehealth/gofhir/r4/resources"
)

func condition() *resources.Condition {
	input := `{
		"resourceType":"Condition","id":"cond1",
		"code":{"coding":[{"system":"http://snomed.info/sct","code":"386661006","display":"Fever"}],"text":"Fever"},
		"subject":{"reference":"Patient/example"},
		"severity":{"coding":[{"code":"24484000","display":"Severe"}]},
		"onsetDateTime":"2024-01-15"
	}`
	var c resources.Condition
	json.Unmarshal([]byte(input), &c)
	return &c
}

func encounter() *resources.Encounter {
	input := `{
		"resourceType":"Encounter","id":"enc1","status":"finished",
		"class":{"system":"http://terminology.hl7.org/CodeSystem/v3-ActCode","code":"IMP"},
		"subject":{"reference":"Patient/example"},
		"period":{"start":"2024-01-01","end":"2024-01-05"},
		"participant":[
			{"individual":{"reference":"Practitioner/1"}},
			{"individual":{"reference":"Practitioner/2"}}
		]
	}`
	var e resources.Encounter
	json.Unmarshal([]byte(input), &e)
	return &e
}

func patientWithExtensions() *resources.Patient {
	input := `{
		"resourceType":"Patient","id":"ext-test","gender":"male",
		"extension":[
			{"url":"http://hl7.org/fhir/us/core/StructureDefinition/us-core-race","extension":[
				{"url":"ombCategory","valueCoding":{"system":"urn:oid:2.16.840.1.113883.6.238","code":"2106-3","display":"White"}},
				{"url":"text","valueString":"White"}
			]},
			{"url":"http://hl7.org/fhir/StructureDefinition/patient-birthTime","valueDateTime":"1980-01-01T10:30:00-05:00"}
		],
		"name":[{"family":"Test","given":["Extension"]}]
	}`
	var p resources.Patient
	json.Unmarshal([]byte(input), &p)
	return &p
}

// ============================================================================
// Null propagation — empty collections propagate correctly
// ============================================================================

func TestNullPropagation(t *testing.T) {
	p := &resources.Patient{ResourceType: "Patient"}

	tests := []struct {
		expr string
		want int // expected collection size
	}{
		{"name.family", 0},            // nil name → empty
		{"name.where(use='x')", 0},    // nil → empty
		{"gender", 0},                 // nil gender → empty
		{"id", 0},                     // nil id → empty
		{"name.family.length()", 0},   // empty chain → empty
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := fhirpath.Evaluate(p, tt.expr)
			if err != nil {
				t.Fatal(err)
			}
			if len(result) != tt.want {
				t.Errorf("got %d items, want %d: %v", len(result), tt.want, result)
			}
		})
	}
}

func TestNullBooleanSemantics(t *testing.T) {
	p := &resources.Patient{ResourceType: "Patient"}

	// empty.exists() → false
	b, _ := fhirpath.EvaluateBool(p, "name.exists()")
	if b {
		t.Error("empty.exists() should be false")
	}

	// empty.empty() → true
	b, _ = fhirpath.EvaluateBool(p, "name.empty()")
	if !b {
		t.Error("empty.empty() should be true")
	}

	// empty.count() = 0
	result, _ := fhirpath.Evaluate(p, "name.count()")
	if result[0].(int64) != 0 {
		t.Error("empty.count() should be 0")
	}
}

// ============================================================================
// Multiple resource types
// ============================================================================

func TestConditionNavigation(t *testing.T) {
	c := condition()

	result, _ := fhirpath.Evaluate(c, "code.text")
	if result.String() != "Fever" {
		t.Errorf("code.text = %q, want Fever", result.String())
	}

	result, _ = fhirpath.Evaluate(c, "code.coding.code")
	if result.String() != "386661006" {
		t.Errorf("coding.code = %q, want 386661006", result.String())
	}

	result, _ = fhirpath.Evaluate(c, "subject.reference")
	if result.String() != "Patient/example" {
		t.Errorf("subject.reference = %q", result.String())
	}
}

func TestEncounterNavigation(t *testing.T) {
	e := encounter()

	result, _ := fhirpath.Evaluate(e, "status")
	if result.String() != "finished" {
		t.Errorf("status = %q", result.String())
	}

	result, _ = fhirpath.Evaluate(e, "participant.count()")
	if result[0].(int64) != 2 {
		t.Error("should have 2 participants")
	}

	result, _ = fhirpath.Evaluate(e, "period.start")
	if result.String() != "2024-01-01" {
		t.Errorf("period.start = %q", result.String())
	}
}

// ============================================================================
// Real FHIR invariants (from StructureDefinitions)
// ============================================================================

func TestInvariantObsValueOrDataAbsent(t *testing.T) {
	// obs-6: dataAbsentReason SHALL only be present if value is not present
	// Simplified: value.exists() or dataAbsentReason.exists()
	obs := observation()
	b, err := fhirpath.EvaluateBool(obs, "value.exists()")
	if err != nil {
		t.Fatal(err)
	}
	if !b {
		t.Error("obs-6: observation with value should pass")
	}
}

func TestInvariantPatientContact(t *testing.T) {
	// pat-1: contact SHALL have a name or organization
	// name.exists() or telecom.exists() or address.exists() or organization.exists()
	p := patient()
	b, _ := fhirpath.EvaluateBool(p, "name.exists() or telecom.exists()")
	if !b {
		t.Error("pat-1: patient with name and telecom should pass")
	}
}

func TestInvariantDomainResource(t *testing.T) {
	// dom-2: If a resource is contained, it SHALL NOT contain nested Resources
	// contained.contained.empty()
	p := patient()
	b, _ := fhirpath.EvaluateBool(p, "contained.empty()")
	if !b {
		t.Error("patient with no contained should have empty contained")
	}
}

// ============================================================================
// Error cases — invalid expressions
// ============================================================================

func TestInvalidExpressions(t *testing.T) {
	tests := []string{
		"",
		".",
		"name.",
		"(((",
		"name[",
		"'unterminated",
	}
	for _, expr := range tests {
		t.Run(expr, func(t *testing.T) {
			_, err := fhirpath.Evaluate(patient(), expr)
			if err == nil {
				t.Errorf("expected error for %q", expr)
			}
		})
	}
}

func TestUnknownFunction(t *testing.T) {
	_, err := fhirpath.Evaluate(patient(), "name.nonExistentFunction()")
	if err == nil {
		t.Error("unknown function should error")
	}
}

func TestDivisionByZero(t *testing.T) {
	result, err := fhirpath.Evaluate(patient(), "10 / 0")
	if err != nil {
		t.Fatal(err)
	}
	// FHIRPath: division by zero → empty
	if len(result) != 0 {
		t.Errorf("10/0 should be empty, got %v", result)
	}
}

// ============================================================================
// Chained operations — complex real-world expressions
// ============================================================================

func TestChainedWhere(t *testing.T) {
	result, _ := fhirpath.Evaluate(patient(), "name.where(use='official').given.first()")
	if result.String() != "John" {
		t.Errorf("got %q, want John", result.String())
	}
}

func TestChainedStartsWith(t *testing.T) {
	b, _ := fhirpath.EvaluateBool(patient(), "name.where(use='official').given.first().startsWith('J')")
	if !b {
		t.Error("John should startsWith J")
	}
}

func TestChainedCount(t *testing.T) {
	b, _ := fhirpath.EvaluateBool(patient(), "name.where(use='official').given.count() = 2")
	if !b {
		t.Error("official name should have 2 given names")
	}
}

func TestNestedWhere(t *testing.T) {
	// Names where any given name starts with 'Jo'
	result, _ := fhirpath.Evaluate(patient(), "name.where(given.exists(startsWith('Jo')))")
	if len(result) != 2 {
		t.Errorf("both names have a 'Jo*' given, got %d", len(result))
	}
}

// ============================================================================
// Extension navigation
// ============================================================================

func TestExtensionFunction(t *testing.T) {
	p := patientWithExtensions()

	// Find race extension
	result, err := fhirpath.Evaluate(p, "extension('http://hl7.org/fhir/us/core/StructureDefinition/us-core-race')")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("should find 1 race extension, got %d", len(result))
	}

	// Find birthTime extension
	result, err = fhirpath.Evaluate(p, "extension('http://hl7.org/fhir/StructureDefinition/patient-birthTime')")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatal("should find birthTime extension")
	}

	// Missing extension
	result, _ = fhirpath.Evaluate(p, "extension('http://not-found')")
	if len(result) != 0 {
		t.Error("missing extension should return empty")
	}
}

// ============================================================================
// Value[x] polymorphic navigation
// ============================================================================

func TestValueXNavigation(t *testing.T) {
	obs := observation()

	// value is a union — accessing the Quantity variant
	result, _ := fhirpath.Evaluate(obs, "value")
	if len(result) == 0 {
		t.Fatal("value should not be empty")
	}

	// Navigate into the component's value
	result, _ = fhirpath.Evaluate(obs, "component.value")
	if len(result) != 2 {
		t.Errorf("should have 2 component values, got %d", len(result))
	}
}

// ============================================================================
// Boolean three-valued logic edge cases
// ============================================================================

func TestBooleanEdgeCases(t *testing.T) {
	p := patient()

	// true and true
	b, _ := fhirpath.EvaluateBool(p, "true and true")
	if !b {
		t.Error("true and true = true")
	}

	// true and false
	b, _ = fhirpath.EvaluateBool(p, "true and false")
	if b {
		t.Error("true and false = false")
	}

	// false or true
	b, _ = fhirpath.EvaluateBool(p, "false or true")
	if !b {
		t.Error("false or true = true")
	}

	// false implies anything = true
	b, _ = fhirpath.EvaluateBool(p, "false implies false")
	if !b {
		t.Error("false implies false = true")
	}

	// true xor false
	b, _ = fhirpath.EvaluateBool(p, "true xor false")
	if !b {
		t.Error("true xor false = true")
	}

	// true xor true
	b, _ = fhirpath.EvaluateBool(p, "true xor true")
	if b {
		t.Error("true xor true = false")
	}
}

// ============================================================================
// Comparison edge cases
// ============================================================================

func TestComparisonEdgeCases(t *testing.T) {
	tests := []struct {
		expr string
		want bool
	}{
		{"1 < 2", true},
		{"2 < 1", false},
		{"1 <= 1", true},
		{"2 >= 1", true},
		{"'a' < 'b'", true},
		{"'b' < 'a'", false},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			b, err := fhirpath.EvaluateBool(patient(), tt.expr)
			if err != nil {
				t.Fatal(err)
			}
			if b != tt.want {
				t.Errorf("got %v, want %v", b, tt.want)
			}
		})
	}
}

// ============================================================================
// String concatenation
// ============================================================================

func TestStringConcatenation(t *testing.T) {
	result, _ := fhirpath.Evaluate(patient(), "'Hello' + ' ' + 'World'")
	if result.String() != "Hello World" {
		t.Errorf("got %q, want 'Hello World'", result.String())
	}

	result, _ = fhirpath.Evaluate(patient(), "'a' & 'b'")
	if result.String() != "ab" {
		t.Errorf("& got %q, want 'ab'", result.String())
	}
}

// ============================================================================
// Multiple function calls on same compiled expression
// ============================================================================

func TestCompiledExpressionReuse(t *testing.T) {
	expr, _ := fhirpath.Compile("name.count()")

	patients := []string{
		`{"resourceType":"Patient","name":[{"family":"A"}]}`,
		`{"resourceType":"Patient","name":[{"family":"A"},{"family":"B"}]}`,
		`{"resourceType":"Patient","name":[{"family":"A"},{"family":"B"},{"family":"C"}]}`,
	}

	for i, pJSON := range patients {
		var p resources.Patient
		json.Unmarshal([]byte(pJSON), &p)
		result, _ := expr.Evaluate(&p)
		want := int64(i + 1)
		if result[0].(int64) != want {
			t.Errorf("patient %d: count = %v, want %d", i, result[0], want)
		}
	}
}

// ============================================================================
// in operator
// ============================================================================

func TestInOperator(t *testing.T) {
	b, _ := fhirpath.EvaluateBool(patient(), "gender in ('male' | 'female')")
	if !b {
		t.Error("male should be in (male | female)")
	}
}

// ============================================================================
// resolve() with resolver callback
// ============================================================================

func TestResolve(t *testing.T) {
	obs := observation()

	// Create a resolver that returns a Patient for "Patient/example"
	resolver := func(ref string) any {
		if ref == "Patient/example" {
			return patient()
		}
		return nil
	}

	result, err := fhirpath.EvaluateWithResolver(obs, "subject.resolve().name.family", resolver)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 || result.String() != "Doe" {
		t.Errorf("resolve().name.family = %v, want [Doe]", result)
	}
}

func TestResolveNoResolver(t *testing.T) {
	obs := observation()
	result, err := fhirpath.Evaluate(obs, "subject.resolve()")
	if err != nil {
		t.Fatal(err)
	}
	// No resolver → empty
	if len(result) != 0 {
		t.Error("resolve without resolver should return empty")
	}
}

// ============================================================================
// Date literals and comparison
// ============================================================================

func TestDateLiteralComparison(t *testing.T) {
	tests := []struct {
		expr string
		want bool
	}{
		{"@2024-01-01 < @2024-06-15", true},
		{"@2024-06-15 > @2024-01-01", true},
		{"@2024-01-01 = @2024-01-01", true},
		{"@2024-01-01 != @2024-06-15", true},
		{"@2023 < @2024", true},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			b, err := fhirpath.EvaluateBool(patient(), tt.expr)
			if err != nil {
				t.Fatal(err)
			}
			if b != tt.want {
				t.Errorf("got %v, want %v", b, tt.want)
			}
		})
	}
}

func TestDateFieldComparison(t *testing.T) {
	b, err := fhirpath.EvaluateBool(patient(), "birthDate < @2000-01-01")
	if err != nil {
		t.Fatal(err)
	}
	if !b {
		t.Error("1980-01-01 should be < 2000-01-01")
	}
}

// ============================================================================
// %resource and $this
// ============================================================================

func TestResourceEnvironment(t *testing.T) {
	p := patient()

	result, _ := fhirpath.Evaluate(p, "%resource.id")
	if result.String() != "example" {
		t.Errorf("%%resource.id = %q, want example", result.String())
	}
}

// ============================================================================
// Compiled expression with resolver
// ============================================================================

// ============================================================================
// Quantity literals and comparison
// ============================================================================

func TestQuantityLiteral(t *testing.T) {
	result, err := fhirpath.Evaluate(patient(), "5 'mg'")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result.String() != "5 'mg'" {
		t.Errorf("got %q", result.String())
	}
}

func TestQuantityComparison(t *testing.T) {
	tests := []struct {
		expr string
		want bool
	}{
		{"5 'mg' < 10 'mg'", true},
		{"10 'mg' > 5 'mg'", true},
		{"5 'mg' = 5 'mg'", true},
		{"5 'mg' != 10 'mg'", true},
		{"5 'mg' <= 5 'mg'", true},
		{"5 'mg' >= 5 'mg'", true},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			b, err := fhirpath.EvaluateBool(patient(), tt.expr)
			if err != nil {
				t.Fatal(err)
			}
			if b != tt.want {
				t.Errorf("got %v, want %v", b, tt.want)
			}
		})
	}
}

func TestQuantityComparisonDifferentUnits(t *testing.T) {
	// Different units — should not be equal
	b, _ := fhirpath.EvaluateBool(patient(), "5 'mg' = 5 'kg'")
	if b {
		t.Error("5 mg should not equal 5 kg")
	}
}

func TestQuantityFromResource(t *testing.T) {
	obs := observation()

	// Navigate into value[x] Quantity via the union's Quantity field
	result, err := fhirpath.Evaluate(obs, "value.Quantity.value")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Fatal("value.Quantity.value should not be empty")
	}

	// Component quantities
	result, err = fhirpath.Evaluate(obs, "component.value.Quantity.value")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 component values, got %d", len(result))
	}
}

// ============================================================================
// Date arithmetic
// ============================================================================

func TestDateArithmeticYears(t *testing.T) {
	tests := []struct {
		expr string
		want string
	}{
		{"@2024-01-15 + 1 'year'", "2025-01-15"},
		{"@2024-01-15 + 2 'years'", "2026-01-15"},
		{"@2024-01-15 - 1 'year'", "2023-01-15"},
		{"@2020-06-15 + 10 'years'", "2030-06-15"},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := fhirpath.Evaluate(patient(), tt.expr)
			if err != nil {
				t.Fatal(err)
			}
			if result.String() != tt.want {
				t.Errorf("got %q, want %q", result.String(), tt.want)
			}
		})
	}
}

func TestDateArithmeticMonths(t *testing.T) {
	tests := []struct {
		expr string
		want string
	}{
		{"@2024-01-15 + 1 'month'", "2024-02-15"},
		{"@2024-01-15 + 6 'months'", "2024-07-15"},
		{"@2024-03-31 + 1 'month'", "2024-05-01"}, // March 31 + 1 month → April 31 → May 1
		{"@2024-01-15 - 1 'month'", "2023-12-15"},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := fhirpath.Evaluate(patient(), tt.expr)
			if err != nil {
				t.Fatal(err)
			}
			if result.String() != tt.want {
				t.Errorf("got %q, want %q", result.String(), tt.want)
			}
		})
	}
}

func TestDateArithmeticDays(t *testing.T) {
	tests := []struct {
		expr string
		want string
	}{
		{"@2024-01-15 + 10 'days'", "2024-01-25"},
		{"@2024-01-31 + 1 'day'", "2024-02-01"},
		{"@2024-03-01 - 1 'day'", "2024-02-29"}, // leap year
		{"@2024-01-15 + 1 'week'", "2024-01-22"},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := fhirpath.Evaluate(patient(), tt.expr)
			if err != nil {
				t.Fatal(err)
			}
			if result.String() != tt.want {
				t.Errorf("got %q, want %q", result.String(), tt.want)
			}
		})
	}
}

func TestDateArithmeticLeapYear(t *testing.T) {
	// Feb 29 exists in 2024 (leap year) but not 2025
	result, _ := fhirpath.Evaluate(patient(), "@2024-02-29 + 1 'year'")
	got := result.String()
	// Go's AddDate: 2024-02-29 + 1 year = 2025-03-01
	if got != "2025-03-01" {
		t.Errorf("leap year: got %q, want 2025-03-01", got)
	}

	// Subtracting back: 2025-03-01 - 1 year = 2024-03-01 (not Feb 29)
	result, _ = fhirpath.Evaluate(patient(), "@2025-03-01 - 1 'year'")
	got = result.String()
	if got != "2024-03-01" {
		t.Errorf("reverse leap: got %q, want 2024-03-01", got)
	}
}

func TestDateArithmeticPrecisionPreserved(t *testing.T) {
	// Year-only: + 1 year stays year-only
	result, _ := fhirpath.Evaluate(patient(), "@2024 + 1 'year'")
	if result.String() != "2025" {
		t.Errorf("year precision: got %q, want 2025", result.String())
	}

	// Month-only
	result, _ = fhirpath.Evaluate(patient(), "@2024-06 + 3 'months'")
	if result.String() != "2024-09" {
		t.Errorf("month precision: got %q, want 2024-09", result.String())
	}
}

func TestDateArithmeticWithBirthDate(t *testing.T) {
	// Real-world: is patient older than 18?
	// birthDate is 1980-01-01, so birthDate + 18 years < today
	b, err := fhirpath.EvaluateBool(patient(), "birthDate + 18 'years' < @2024-01-01")
	if err != nil {
		// birthDate is a string field, arithmetic should work
		t.Fatal(err)
	}
	if !b {
		t.Error("1980-01-01 + 18 years = 1998-01-01 should be < 2024-01-01")
	}
}

func TestCompiledWithResolver(t *testing.T) {
	expr, _ := fhirpath.Compile("subject.resolve().name.family")
	expr.WithResolver(func(ref string) any {
		if ref == "Patient/example" {
			return patient()
		}
		return nil
	})

	result, _ := expr.Evaluate(observation())
	if result.String() != "Doe" {
		t.Errorf("got %q, want Doe", result.String())
	}
}
