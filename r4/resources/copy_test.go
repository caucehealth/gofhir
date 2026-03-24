// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	"testing"

	"github.com/caucehealth/gofhir/r4/resources"
)

func TestDeepCopy(t *testing.T) {
	p, _ := resources.NewPatient().
		WithName("John", "Doe").
		WithGender(resources.AdministrativeGenderMale).
		WithBirthDate("1980-01-01").
		Build()

	cp, err := resources.DeepCopy(p)
	if err != nil {
		t.Fatal(err)
	}

	// Verify copy has same values
	if cp.GetGender() != resources.AdministrativeGenderMale {
		t.Error("copy gender should be male")
	}
	if cp.GetBirthDate() != "1980-01-01" {
		t.Error("copy birthDate should match")
	}
	if len(cp.Name) != 1 || *cp.Name[0].Family != "Doe" {
		t.Error("copy name should match")
	}

	// Verify independence — modifying copy doesn't affect original
	newGender := resources.AdministrativeGenderFemale
	cp.Gender = &newGender
	if *p.Gender == resources.AdministrativeGenderFemale {
		t.Error("modifying copy should not affect original")
	}

	// Modify copy's name
	newFamily := "Smith"
	cp.Name[0].Family = &newFamily
	if *p.Name[0].Family != "Doe" {
		t.Error("modifying copy name should not affect original")
	}
}
