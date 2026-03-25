// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package fhirpath_test

import (
	"encoding/json"
	"testing"

	"github.com/caucehealth/gofhir/r4/fhirpath"
	"github.com/caucehealth/gofhir/r4/resources"
)

func patient() *resources.Patient {
	input := `{
		"resourceType":"Patient","id":"example","gender":"male","birthDate":"1980-01-01","active":true,
		"name":[
			{"use":"official","family":"Doe","given":["John","James"]},
			{"use":"nickname","family":"Doe","given":["Johnny"]}
		],
		"telecom":[{"system":"phone","value":"555-1234"}],
		"address":[{"city":"Springfield","state":"IL"}],
		"identifier":[{"system":"http://example.org","value":"12345"}]
	}`
	var p resources.Patient
	json.Unmarshal([]byte(input), &p)
	return &p
}

func observation() *resources.Observation {
	input := `{
		"resourceType":"Observation","id":"obs1","status":"final",
		"code":{"coding":[{"system":"http://loinc.org","code":"8867-4","display":"Heart rate"}],"text":"Heart rate"},
		"valueQuantity":{"value":72,"unit":"bpm","system":"http://unitsofmeasure.org","code":"/min"},
		"subject":{"reference":"Patient/example"},
		"component":[
			{"code":{"text":"systolic"},"valueQuantity":{"value":120,"unit":"mmHg"}},
			{"code":{"text":"diastolic"},"valueQuantity":{"value":80,"unit":"mmHg"}}
		]
	}`
	var obs resources.Observation
	json.Unmarshal([]byte(input), &obs)
	return &obs
}

// === Navigation ===

func TestNavigateSimpleField(t *testing.T) {
	result, err := fhirpath.Evaluate(patient(), "id")
	if err != nil {
		t.Fatal(err)
	}
	if result.String() != "example" {
		t.Errorf("got %q, want example", result.String())
	}
}

func TestNavigateNestedField(t *testing.T) {
	result, err := fhirpath.Evaluate(patient(), "name.family")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 families, got %d", len(result))
	}
}

func TestNavigateDeepField(t *testing.T) {
	result, err := fhirpath.Evaluate(observation(), "code.coding.code")
	if err != nil {
		t.Fatal(err)
	}
	if result.String() != "8867-4" {
		t.Errorf("got %q, want 8867-4", result.String())
	}
}

func TestNavigateArray(t *testing.T) {
	result, err := fhirpath.Evaluate(patient(), "name.given")
	if err != nil {
		t.Fatal(err)
	}
	// Should flatten: ["John", "James", "Johnny"]
	if len(result) != 3 {
		t.Errorf("expected 3 given names, got %d: %v", len(result), result)
	}
}

// === where() ===

func TestWhereFilter(t *testing.T) {
	result, err := fhirpath.Evaluate(patient(), "name.where(use='official').family")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result.String() != "Doe" {
		t.Errorf("got %v, want [Doe]", result)
	}
}

func TestWhereNoMatch(t *testing.T) {
	result, err := fhirpath.Evaluate(patient(), "name.where(use='temp').family")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty, got %v", result)
	}
}

// === exists(), empty(), count() ===

func TestExists(t *testing.T) {
	b, err := fhirpath.EvaluateBool(patient(), "name.exists()")
	if err != nil {
		t.Fatal(err)
	}
	if !b {
		t.Error("name should exist")
	}
}

func TestExistsWithCriteria(t *testing.T) {
	b, err := fhirpath.EvaluateBool(patient(), "name.exists(use='official')")
	if err != nil {
		t.Fatal(err)
	}
	if !b {
		t.Error("should have official name")
	}
}

func TestEmpty(t *testing.T) {
	b, err := fhirpath.EvaluateBool(patient(), "photo.empty()")
	if err != nil {
		t.Fatal(err)
	}
	if !b {
		t.Error("photo should be empty")
	}
}

func TestCount(t *testing.T) {
	result, err := fhirpath.Evaluate(patient(), "name.count()")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatal("count should return 1 element")
	}
	if result[0].(int64) != 2 {
		t.Errorf("name.count() = %v, want 2", result[0])
	}
}

// === Indexing ===

func TestIndex(t *testing.T) {
	result, err := fhirpath.Evaluate(patient(), "name[0].family")
	if err != nil {
		t.Fatal(err)
	}
	if result.String() != "Doe" {
		t.Errorf("got %q, want Doe", result.String())
	}
}

func TestFirstLast(t *testing.T) {
	result, err := fhirpath.Evaluate(patient(), "name.given.first()")
	if err != nil {
		t.Fatal(err)
	}
	if result.String() != "John" {
		t.Errorf("first() = %q, want John", result.String())
	}

	result, err = fhirpath.Evaluate(patient(), "name.given.last()")
	if err != nil {
		t.Fatal(err)
	}
	if result.String() != "Johnny" {
		t.Errorf("last() = %q, want Johnny", result.String())
	}
}

// === Comparison ===

func TestEquality(t *testing.T) {
	b, err := fhirpath.EvaluateBool(patient(), "gender = 'male'")
	if err != nil {
		t.Fatal(err)
	}
	if !b {
		t.Error("gender should equal male")
	}
}

func TestInequality(t *testing.T) {
	b, err := fhirpath.EvaluateBool(patient(), "gender != 'female'")
	if err != nil {
		t.Fatal(err)
	}
	if !b {
		t.Error("gender should not equal female")
	}
}

// === Boolean logic ===

func TestAnd(t *testing.T) {
	b, err := fhirpath.EvaluateBool(patient(), "active and (gender = 'male')")
	if err != nil {
		t.Fatal(err)
	}
	if !b {
		t.Error("active and male should be true")
	}
}

func TestOr(t *testing.T) {
	b, err := fhirpath.EvaluateBool(patient(), "gender = 'female' or active")
	if err != nil {
		t.Fatal(err)
	}
	if !b {
		t.Error("female or active should be true")
	}
}

func TestNot(t *testing.T) {
	b, err := fhirpath.EvaluateBool(patient(), "(gender = 'female').not()")
	if err != nil {
		t.Fatal(err)
	}
	if !b {
		t.Error("not female should be true")
	}
}

// === String functions ===

func TestStartsWith(t *testing.T) {
	b, err := fhirpath.EvaluateBool(patient(), "id.startsWith('ex')")
	if err != nil {
		t.Fatal(err)
	}
	if !b {
		t.Error("id should start with 'ex'")
	}
}

func TestContainsString(t *testing.T) {
	b, err := fhirpath.EvaluateBool(patient(), "id.contains('amp')")
	if err != nil {
		t.Fatal(err)
	}
	if !b {
		t.Error("id should contain 'amp'")
	}
}

func TestLength(t *testing.T) {
	result, err := fhirpath.Evaluate(patient(), "id.length()")
	if err != nil {
		t.Fatal(err)
	}
	if result[0].(int64) != 7 {
		t.Errorf("length = %v, want 7", result[0])
	}
}

// === Type checking ===

func TestIs(t *testing.T) {
	b, err := fhirpath.EvaluateBool(patient(), "active.is(boolean)")
	if err != nil {
		t.Fatal(err)
	}
	if !b {
		t.Error("active should be boolean")
	}
}

// === Arithmetic ===

func TestArithmetic(t *testing.T) {
	tests := []struct {
		expr string
		want float64
	}{
		{"2 + 3", 5},
		{"10 - 4", 6},
		{"3 * 7", 21},
		{"10 / 4", 2.5},
		{"10 mod 3", 1},
		{"10 div 3", 3},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := fhirpath.Evaluate(patient(), tt.expr)
			if err != nil {
				t.Fatal(err)
			}
			if len(result) != 1 {
				t.Fatalf("expected 1 result, got %d", len(result))
			}
			got := fhirpath.ToFloat(result[0])
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// === Compile once, evaluate many ===

func TestCompileAndReuse(t *testing.T) {
	expr, err := fhirpath.Compile("name.where(use='official').family")
	if err != nil {
		t.Fatal(err)
	}

	r1, _ := expr.Evaluate(patient())
	if r1.String() != "Doe" {
		t.Errorf("patient1: got %q", r1.String())
	}

	// Evaluate same expression on different resource
	input2 := `{"resourceType":"Patient","name":[{"use":"official","family":"Smith"}]}`
	var p2 resources.Patient
	json.Unmarshal([]byte(input2), &p2)

	r2, _ := expr.Evaluate(&p2)
	if r2.String() != "Smith" {
		t.Errorf("patient2: got %q", r2.String())
	}
}

// === Component navigation (real-world) ===

func TestObservationComponentValue(t *testing.T) {
	result, err := fhirpath.Evaluate(observation(), "component.code.text")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 component codes, got %d", len(result))
	}
}

// === all() ===

func TestAll(t *testing.T) {
	b, err := fhirpath.EvaluateBool(patient(), "name.all(family = 'Doe')")
	if err != nil {
		t.Fatal(err)
	}
	if !b {
		t.Error("all names should have family Doe")
	}

	b, err = fhirpath.EvaluateBool(patient(), "name.all(use = 'official')")
	if err != nil {
		t.Fatal(err)
	}
	if b {
		t.Error("not all names are official")
	}
}

// === select() ===

func TestSelect(t *testing.T) {
	result, err := fhirpath.Evaluate(patient(), "name.select(family)")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 families from select, got %d", len(result))
	}
}

// === Literals ===

func TestLiterals(t *testing.T) {
	result, err := fhirpath.Evaluate(patient(), "true")
	if err != nil {
		t.Fatal(err)
	}
	if result[0] != true {
		t.Error("true literal should be true")
	}

	result, err = fhirpath.Evaluate(patient(), "42")
	if err != nil {
		t.Fatal(err)
	}
	if result[0].(int64) != 42 {
		t.Errorf("42 literal = %v", result[0])
	}

	result, err = fhirpath.Evaluate(patient(), "'hello'")
	if err != nil {
		t.Fatal(err)
	}
	if result[0] != "hello" {
		t.Errorf("string literal = %v", result[0])
	}
}

// === Union ===

func TestUnion(t *testing.T) {
	result, err := fhirpath.Evaluate(patient(), "gender | birthDate")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Errorf("union should have 2 items, got %d", len(result))
	}
}

// === iif ===

func TestIif(t *testing.T) {
	result, err := fhirpath.Evaluate(patient(), "iif(active, 'yes', 'no')")
	if err != nil {
		t.Fatal(err)
	}
	if result.String() != "yes" {
		t.Errorf("iif = %q, want yes", result.String())
	}
}
