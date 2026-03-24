// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package resources

import "encoding/json"

// DeepCopy creates a deep copy of any FHIR resource by marshaling to JSON
// and unmarshaling into a new instance. All pointer fields and slices in
// the copy are independent from the original.
func DeepCopy[T any](resource *T) (*T, error) {
	data, err := json.Marshal(resource)
	if err != nil {
		return nil, err
	}
	var copy T
	if err := json.Unmarshal(data, &copy); err != nil {
		return nil, err
	}
	return &copy, nil
}
