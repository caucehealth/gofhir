// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"encoding/json"
	"fmt"
)

// ContainedResourceType extracts the resourceType field from a contained
// resource's raw JSON without fully unmarshaling it.
func ContainedResourceType(raw json.RawMessage) (string, error) {
	var header struct {
		ResourceType string `json:"resourceType"`
	}
	if err := json.Unmarshal(raw, &header); err != nil {
		return "", fmt.Errorf("reading resourceType: %w", err)
	}
	if header.ResourceType == "" {
		return "", fmt.Errorf("contained resource has no resourceType")
	}
	return header.ResourceType, nil
}

// ParseContained unmarshals a contained resource as the appropriate type
// based on its resourceType field. Returns the typed resource as an any value.
// The caller should type-assert to the expected resource type.
// Supports all 145 FHIR R4 resource types via the generated registry.
func ParseContained(raw json.RawMessage) (any, error) {
	return ParseResource(raw)
}

// ParseContainedAs unmarshals a contained resource and verifies its type matches
// the expected resourceType. Returns an error if the types don't match.
func ParseContainedAs[T any](raw json.RawMessage, expected string) (*T, error) {
	rt, err := ContainedResourceType(raw)
	if err != nil {
		return nil, err
	}
	if rt != expected {
		return nil, fmt.Errorf("expected %s, got %s", expected, rt)
	}
	var r T
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// ParseContainedPatient unmarshals a contained resource as a Patient.
func ParseContainedPatient(raw json.RawMessage) (*Patient, error) {
	return ParseContainedAs[Patient](raw, "Patient")
}

// ParseContainedObservation unmarshals a contained resource as an Observation.
func ParseContainedObservation(raw json.RawMessage) (*Observation, error) {
	return ParseContainedAs[Observation](raw, "Observation")
}

// ParseContainedPractitioner unmarshals a contained resource as a Practitioner.
func ParseContainedPractitioner(raw json.RawMessage) (*Practitioner, error) {
	return ParseContainedAs[Practitioner](raw, "Practitioner")
}
