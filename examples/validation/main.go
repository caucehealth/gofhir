// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

// Example validation demonstrates resource validation with profiles and custom rules.
package main

import (
	"encoding/json"
	"fmt"

	"github.com/caucehealth/gofhir/r4/resources"
	"github.com/caucehealth/gofhir/r4/validate"
)

func main() {
	// --- Basic validation ---
	fmt.Println("=== Basic Validation ===")

	// An Observation missing required fields
	obs := &resources.Observation{ResourceType: "Observation"}
	v := validate.New()
	result := v.Validate(obs)

	if result.HasErrors() {
		fmt.Println("Observation validation errors:")
		for _, issue := range result.Errors() {
			fmt.Printf("  [%s] %s: %s\n", issue.Code, issue.Path, issue.Message)
		}
	}

	// --- Validate JSON directly ---
	fmt.Println("\n=== JSON Validation ===")

	validJSON := json.RawMessage(`{"resourceType":"Patient","id":"1","gender":"male"}`)
	result2, _ := validate.ValidateJSON(validJSON)
	if !result2.HasErrors() {
		fmt.Println("Patient JSON is valid")
	}

	// --- Custom rules ---
	fmt.Println("\n=== Custom Rule ===")

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
	v = validate.New(validate.WithRules(nameRule))
	result3 := v.Validate(p)
	for _, w := range result3.Warnings() {
		fmt.Printf("  Warning: %s\n", w.Message)
	}

	// --- OperationOutcome ---
	fmt.Println("\n=== OperationOutcome ===")

	oo := result.ToOperationOutcome()
	ooJSON, _ := json.MarshalIndent(oo, "", "  ")
	fmt.Println(string(ooJSON))
}
