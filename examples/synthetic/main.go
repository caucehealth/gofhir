// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

// Example synthetic demonstrates generating random FHIR resources for testing.
package main

import (
	"encoding/json"
	"fmt"

	"github.com/caucehealth/gofhir/r4/bulk"
	"github.com/caucehealth/gofhir/r4/diff"
	"github.com/caucehealth/gofhir/r4/synthetic"
)

func main() {
	// Create a reproducible generator
	gen := synthetic.NewWithSeed(42)

	// Generate a single patient
	patient := gen.Patient()
	data, _ := json.MarshalIndent(patient, "", "  ")
	fmt.Println("=== Generated Patient ===")
	fmt.Println(string(data))

	// Generate a populated dataset
	patients, observations, conditions, encounters := gen.PopulatedBundle(5)
	fmt.Printf("\n=== Populated Bundle ===\n")
	fmt.Printf("  Patients:     %d\n", len(patients))
	fmt.Printf("  Observations: %d\n", len(observations))
	fmt.Printf("  Conditions:   %d\n", len(conditions))
	fmt.Printf("  Encounters:   %d\n", len(encounters))

	// Diff two patients
	p1 := gen.Patient()
	p2 := gen.Patient()
	result, _ := diff.Compare(p1, p2)
	fmt.Printf("\n=== Diff ===\n")
	fmt.Printf("  Changes: %d\n", len(result.Changes))
	for _, c := range result.Changes {
		fmt.Printf("  %s %s\n", c.Type, c.Path)
	}

	// Write as NDJSON
	fmt.Printf("\n=== NDJSON ===\n")
	w := bulk.NewNDJSONWriter(nil) // just showing the API
	_ = w
	fmt.Println("  bulk.NewNDJSONWriter(file) -> writer.Write(patient)")
}
