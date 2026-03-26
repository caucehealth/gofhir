// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

// Package diff compares two FHIR resources and produces a structured
// list of changes. Useful for auditing, generating patches, and
// displaying resource version differences.
package diff

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
)

// ChangeType describes the kind of change.
type ChangeType string

const (
	Added    ChangeType = "added"
	Removed  ChangeType = "removed"
	Modified ChangeType = "modified"
)

// Change represents a single difference between two resource versions.
type Change struct {
	// Type is the kind of change (added, removed, modified).
	Type ChangeType `json:"type"`
	// Path is the JSON pointer-style path (e.g. "name/0/family").
	Path string `json:"path"`
	// OldValue is the previous value (nil for additions).
	OldValue any `json:"oldValue,omitempty"`
	// NewValue is the new value (nil for removals).
	NewValue any `json:"newValue,omitempty"`
}

// Result holds the complete diff between two resources.
type Result struct {
	Changes []Change `json:"changes"`
}

// HasChanges returns true if there are any differences.
func (r *Result) HasChanges() bool {
	return len(r.Changes) > 0
}

// Additions returns only added fields.
func (r *Result) Additions() []Change {
	return r.filter(Added)
}

// Removals returns only removed fields.
func (r *Result) Removals() []Change {
	return r.filter(Removed)
}

// Modifications returns only modified fields.
func (r *Result) Modifications() []Change {
	return r.filter(Modified)
}

func (r *Result) filter(t ChangeType) []Change {
	var out []Change
	for _, c := range r.Changes {
		if c.Type == t {
			out = append(out, c)
		}
	}
	return out
}

// Compare diffs two FHIR resources by comparing their JSON representations.
// Both arguments should be FHIR resource structs or pointers to structs.
func Compare(old, new any) (*Result, error) {
	oldJSON, err := toMap(old)
	if err != nil {
		return nil, fmt.Errorf("marshal old: %w", err)
	}
	newJSON, err := toMap(new)
	if err != nil {
		return nil, fmt.Errorf("marshal new: %w", err)
	}

	result := &Result{}
	compareValues(result, "", oldJSON, newJSON)
	sort.Slice(result.Changes, func(i, j int) bool {
		return result.Changes[i].Path < result.Changes[j].Path
	})
	return result, nil
}

// CompareJSON diffs two raw JSON representations of FHIR resources.
func CompareJSON(oldJSON, newJSON json.RawMessage) (*Result, error) {
	var oldMap, newMap any
	if err := json.Unmarshal(oldJSON, &oldMap); err != nil {
		return nil, fmt.Errorf("unmarshal old: %w", err)
	}
	if err := json.Unmarshal(newJSON, &newMap); err != nil {
		return nil, fmt.Errorf("unmarshal new: %w", err)
	}

	result := &Result{}
	compareValues(result, "", oldMap, newMap)
	sort.Slice(result.Changes, func(i, j int) bool {
		return result.Changes[i].Path < result.Changes[j].Path
	})
	return result, nil
}

func toMap(v any) (any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m any
	err = json.Unmarshal(data, &m)
	return m, err
}

func joinPath(base, key string) string {
	if base == "" {
		return key
	}
	return base + "/" + key
}

func compareValues(result *Result, path string, old, new any) {
	if old == nil && new == nil {
		return
	}
	if old == nil {
		result.Changes = append(result.Changes, Change{Type: Added, Path: path, NewValue: new})
		return
	}
	if new == nil {
		result.Changes = append(result.Changes, Change{Type: Removed, Path: path, OldValue: old})
		return
	}

	oldMap, oldIsMap := old.(map[string]any)
	newMap, newIsMap := new.(map[string]any)
	if oldIsMap && newIsMap {
		compareMaps(result, path, oldMap, newMap)
		return
	}

	oldArr, oldIsArr := old.([]any)
	newArr, newIsArr := new.([]any)
	if oldIsArr && newIsArr {
		compareArrays(result, path, oldArr, newArr)
		return
	}

	if !reflect.DeepEqual(old, new) {
		result.Changes = append(result.Changes, Change{
			Type: Modified, Path: path, OldValue: old, NewValue: new,
		})
	}
}

func compareMaps(result *Result, path string, old, new map[string]any) {
	// Keys in old
	for key, oldVal := range old {
		newVal, exists := new[key]
		childPath := joinPath(path, key)
		if !exists {
			result.Changes = append(result.Changes, Change{
				Type: Removed, Path: childPath, OldValue: oldVal,
			})
		} else {
			compareValues(result, childPath, oldVal, newVal)
		}
	}
	// Keys only in new
	for key, newVal := range new {
		if _, exists := old[key]; !exists {
			childPath := joinPath(path, key)
			result.Changes = append(result.Changes, Change{
				Type: Added, Path: childPath, NewValue: newVal,
			})
		}
	}
}

func compareArrays(result *Result, path string, old, new []any) {
	maxLen := len(old)
	if len(new) > maxLen {
		maxLen = len(new)
	}
	for i := 0; i < maxLen; i++ {
		childPath := joinPath(path, fmt.Sprintf("%d", i))
		if i >= len(old) {
			result.Changes = append(result.Changes, Change{
				Type: Added, Path: childPath, NewValue: new[i],
			})
		} else if i >= len(new) {
			result.Changes = append(result.Changes, Change{
				Type: Removed, Path: childPath, OldValue: old[i],
			})
		} else {
			compareValues(result, childPath, old[i], new[i])
		}
	}
}
