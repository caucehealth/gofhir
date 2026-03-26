// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

// Example basic demonstrates creating, serializing, and parsing FHIR resources.
package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/caucehealth/gofhir/r4/bundle"
	dt "github.com/caucehealth/gofhir/r4/datatypes"
	"github.com/caucehealth/gofhir/r4/resources"
)

func main() {
	// Build a Patient using the fluent API
	patient, err := resources.NewPatient().
		WithName("John", "Doe").
		WithBirthDate("1980-03-15").
		WithGender(resources.AdministrativeGenderMale).
		Build()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Created patient:", *patient.Name[0].Family)

	// Serialize to JSON
	data, _ := json.MarshalIndent(patient, "", "  ")
	fmt.Println("\nPatient JSON:")
	fmt.Println(string(data))

	// Parse back from JSON
	var parsed resources.Patient
	json.Unmarshal(data, &parsed)
	fmt.Println("\nParsed back:", parsed.GetResourceType(), string(parsed.GetId()))

	// Build an Observation
	obs, _ := resources.NewObservation().
		WithStatus(resources.ObservationStatusFinal).
		WithCode("http://loinc.org", "8867-4", "Heart rate").
		WithSubject("Patient/123").
		Build()

	// Wrap in a Bundle
	b := bundle.New(bundle.TypeCollection).
		WithEntry(patient).
		WithEntry(obs).
		Build()
	bundleJSON, _ := json.MarshalIndent(b, "", "  ")
	fmt.Println("\nBundle with", len(b.Entry), "entries")

	// Use the Decimal type for precision
	d := dt.NewDecimal(3.14)
	fmt.Println("\nDecimal:", d.String())

	_ = bundleJSON
}
