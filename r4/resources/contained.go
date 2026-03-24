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

// ParseContainedPatient unmarshals a contained resource as a Patient.
// Returns an error if the raw JSON is not a Patient resource.
func ParseContainedPatient(raw json.RawMessage) (*Patient, error) {
	rt, err := ContainedResourceType(raw)
	if err != nil {
		return nil, err
	}
	if rt != "Patient" {
		return nil, fmt.Errorf("expected Patient, got %s", rt)
	}
	var p Patient
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// ParseContainedObservation unmarshals a contained resource as an Observation.
func ParseContainedObservation(raw json.RawMessage) (*Observation, error) {
	rt, err := ContainedResourceType(raw)
	if err != nil {
		return nil, err
	}
	if rt != "Observation" {
		return nil, fmt.Errorf("expected Observation, got %s", rt)
	}
	var o Observation
	if err := json.Unmarshal(raw, &o); err != nil {
		return nil, err
	}
	return &o, nil
}

// ParseContainedPractitioner unmarshals a contained resource as a Practitioner.
func ParseContainedPractitioner(raw json.RawMessage) (*Practitioner, error) {
	rt, err := ContainedResourceType(raw)
	if err != nil {
		return nil, err
	}
	if rt != "Practitioner" {
		return nil, fmt.Errorf("expected Practitioner, got %s", rt)
	}
	var p Practitioner
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// ParseContained unmarshals a contained resource as the appropriate type
// based on its resourceType field. Returns the resource as an any value.
// The caller should type-assert to the expected resource type.
func ParseContained(raw json.RawMessage) (any, error) {
	rt, err := ContainedResourceType(raw)
	if err != nil {
		return nil, err
	}

	switch rt {
	case "Patient":
		var r Patient
		return &r, json.Unmarshal(raw, &r)
	case "Observation":
		var r Observation
		return &r, json.Unmarshal(raw, &r)
	case "Encounter":
		var r Encounter
		return &r, json.Unmarshal(raw, &r)
	case "Practitioner":
		var r Practitioner
		return &r, json.Unmarshal(raw, &r)
	case "Condition":
		var r Condition
		return &r, json.Unmarshal(raw, &r)
	case "DiagnosticReport":
		var r DiagnosticReport
		return &r, json.Unmarshal(raw, &r)
	case "MedicationRequest":
		var r MedicationRequest
		return &r, json.Unmarshal(raw, &r)
	case "Organization":
		var r Organization
		return &r, json.Unmarshal(raw, &r)
	case "Location":
		var r Location
		return &r, json.Unmarshal(raw, &r)
	case "Medication":
		var r Medication
		return &r, json.Unmarshal(raw, &r)
	default:
		// Return raw JSON wrapped in a generic map for unsupported types
		var m map[string]any
		return m, json.Unmarshal(raw, &m)
	}
}
