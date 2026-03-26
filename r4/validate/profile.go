// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/caucehealth/gofhir/r4/fhirpath"
	"github.com/caucehealth/gofhir/r4/resources"
)

// ProfileRegistry manages loaded StructureDefinition profiles.
type ProfileRegistry struct {
	mu       sync.RWMutex
	profiles map[string]*Profile // URL → profile
}

// Profile is a parsed StructureDefinition with its validation-relevant elements.
type Profile struct {
	URL      string
	Name     string
	Type     string // base resource type (e.g., "Patient")
	Elements []ProfileElement
}

// ProfileElement describes constraints on a single element path.
type ProfileElement struct {
	Path       string
	Min        int
	Max        string // "*" for unbounded, or "1", "0", etc.
	Types      []string
	BindingURL string
	Strength   string // "required", "extensible", "preferred", "example"
	FixedValue any
	Invariants []ProfileInvariant
	Slicing    *Slicing // non-nil if this element introduces slicing
	SliceName  string   // non-empty if this element is a named slice
}

// Slicing describes how a repeating element is divided into slices.
type Slicing struct {
	Discriminators []Discriminator
	Rules          SlicingRules // "open", "closed", "openAtEnd"
	Ordered        bool
}

// Discriminator identifies how array elements are assigned to slices.
type Discriminator struct {
	Type DiscriminatorType
	Path string
}

// DiscriminatorType is the method used to match elements to slices.
type DiscriminatorType string

const (
	DiscriminatorValue   DiscriminatorType = "value"
	DiscriminatorPattern DiscriminatorType = "pattern"
	DiscriminatorType_   DiscriminatorType = "type"
	DiscriminatorProfile DiscriminatorType = "profile"
	DiscriminatorExists  DiscriminatorType = "exists"
)

// SlicingRules defines whether additional content is allowed beyond defined slices.
type SlicingRules string

const (
	SlicingOpen      SlicingRules = "open"
	SlicingClosed    SlicingRules = "closed"
	SlicingOpenAtEnd SlicingRules = "openAtEnd"
)

// ProfileInvariant is a FHIRPath constraint on an element.
type ProfileInvariant struct {
	Key        string
	Expression string
	Severity   string // "error" or "warning"
	Human      string // human-readable description
}

// NewProfileRegistry creates an empty profile registry.
func NewProfileRegistry() *ProfileRegistry {
	return &ProfileRegistry{profiles: make(map[string]*Profile)}
}

// Load parses a StructureDefinition JSON and registers it.
func (r *ProfileRegistry) Load(data json.RawMessage) error {
	profile, err := parseProfile(data)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.profiles[profile.URL] = profile
	return nil
}

// Register adds a pre-built Profile.
func (r *ProfileRegistry) Register(p *Profile) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.profiles[p.URL] = p
}

// Get returns a profile by URL.
func (r *ProfileRegistry) Get(url string) *Profile {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.profiles[url]
}

// parseProfile extracts validation-relevant elements from a StructureDefinition.
func parseProfile(data json.RawMessage) (*Profile, error) {
	var sd struct {
		URL  string `json:"url"`
		Name string `json:"name"`
		Type string `json:"type"`
		Snapshot struct {
			Element []struct {
				Path    string `json:"path"`
				Min     int    `json:"min"`
				Max     string `json:"max"`
				Type    []struct {
					Code string `json:"code"`
				} `json:"type"`
				Binding *struct {
					Strength string `json:"strength"`
					ValueSet string `json:"valueSet"`
				} `json:"binding"`
				Fixed      json.RawMessage `json:"fixedValue,omitempty"`
				SliceName  string          `json:"sliceName,omitempty"`
				Slicing    *struct {
					Discriminator []struct {
						Type string `json:"type"`
						Path string `json:"path"`
					} `json:"discriminator"`
					Rules   string `json:"rules"`
					Ordered bool   `json:"ordered"`
				} `json:"slicing,omitempty"`
				Constraint []struct {
					Key        string `json:"key"`
					Expression string `json:"expression"`
					Severity   string `json:"severity"`
					Human      string `json:"human"`
				} `json:"constraint"`
			} `json:"element"`
		} `json:"snapshot"`
	}

	if err := json.Unmarshal(data, &sd); err != nil {
		return nil, fmt.Errorf("parse StructureDefinition: %w", err)
	}

	if sd.URL == "" {
		return nil, fmt.Errorf("StructureDefinition has no url")
	}

	profile := &Profile{
		URL:  sd.URL,
		Name: sd.Name,
		Type: sd.Type,
	}

	for _, elem := range sd.Snapshot.Element {
		pe := ProfileElement{
			Path:      elem.Path,
			Min:       elem.Min,
			Max:       elem.Max,
			SliceName: elem.SliceName,
		}

		for _, t := range elem.Type {
			pe.Types = append(pe.Types, t.Code)
		}

		if elem.Binding != nil {
			pe.BindingURL = elem.Binding.ValueSet
			pe.Strength = elem.Binding.Strength
		}

		if elem.Fixed != nil {
			pe.FixedValue = elem.Fixed
		}

		if elem.Slicing != nil {
			s := &Slicing{
				Rules:   SlicingRules(elem.Slicing.Rules),
				Ordered: elem.Slicing.Ordered,
			}
			for _, d := range elem.Slicing.Discriminator {
				s.Discriminators = append(s.Discriminators, Discriminator{
					Type: DiscriminatorType(d.Type),
					Path: d.Path,
				})
			}
			pe.Slicing = s
		}

		for _, c := range elem.Constraint {
			pe.Invariants = append(pe.Invariants, ProfileInvariant{
				Key:        c.Key,
				Expression: c.Expression,
				Severity:   c.Severity,
				Human:      c.Human,
			})
		}

		profile.Elements = append(profile.Elements, pe)
	}

	return profile, nil
}

// --- Profile validation rule ---

// WithProfile adds profile-based validation against a StructureDefinition.
func WithProfile(registry *ProfileRegistry, profileURL string) Option {
	return func(v *Validator) {
		v.rules = append(v.rules, &profileRule{
			registry:   registry,
			profileURL: profileURL,
		})
	}
}

type profileRule struct {
	registry   *ProfileRegistry
	profileURL string
}

func (r *profileRule) Validate(resource resources.Resource) []Issue {
	profile := r.registry.Get(r.profileURL)
	if profile == nil {
		return []Issue{{
			Severity: SeverityWarning,
			Code:     CodeProcessing,
			Message:  fmt.Sprintf("profile %s not found", r.profileURL),
		}}
	}

	// Verify resource type matches
	if profile.Type != "" && profile.Type != resource.GetResourceType() {
		return []Issue{{
			Severity: SeverityError,
			Code:     CodeStructure,
			Message:  fmt.Sprintf("resource type %s does not match profile type %s", resource.GetResourceType(), profile.Type),
		}}
	}

	var issues []Issue

	for _, elem := range profile.Elements {
		// Skip the root element (e.g., "Patient")
		if !strings.Contains(elem.Path, ".") {
			continue
		}

		// Convert path to FHIRPath-navigable form
		// "Patient.name.family" → "name.family"
		parts := strings.SplitN(elem.Path, ".", 2)
		if len(parts) < 2 {
			continue
		}
		fieldPath := parts[1]

		// Check cardinality via FHIRPath
		if elem.Min > 0 {
			result, err := fhirpath.Evaluate(resource, fieldPath+".count()")
			if err == nil && len(result) > 0 {
				count := fhirpath.ToFloat(result[0])
				if int(count) < elem.Min {
					issues = append(issues, Issue{
						Severity: SeverityError,
						Code:     CodeRequired,
						Path:     elem.Path,
						Message:  fmt.Sprintf("%s: minimum cardinality %d not met (found %d)", elem.Path, elem.Min, int(count)),
					})
				}
			}
		}

		if elem.Max != "*" && elem.Max != "" && elem.Max != "0" {
			result, err := fhirpath.Evaluate(resource, fieldPath+".count()")
			if err == nil && len(result) > 0 {
				count := fhirpath.ToFloat(result[0])
				var maxVal int
				fmt.Sscanf(elem.Max, "%d", &maxVal)
				if maxVal > 0 && int(count) > maxVal {
					issues = append(issues, Issue{
						Severity: SeverityError,
						Code:     CodeStructure,
						Path:     elem.Path,
						Message:  fmt.Sprintf("%s: maximum cardinality %s exceeded (found %d)", elem.Path, elem.Max, int(count)),
					})
				}
			}
		}

		// Evaluate FHIRPath invariants
		for _, inv := range elem.Invariants {
			if inv.Expression == "" {
				continue
			}
			// Evaluate the constraint against the resource
			result, err := fhirpath.EvaluateBool(resource, inv.Expression)
			if err != nil {
				continue // skip expressions we can't evaluate
			}
			if !result {
				sev := SeverityError
				if inv.Severity == "warning" {
					sev = SeverityWarning
				}
				issues = append(issues, Issue{
					Severity: sev,
					Code:     CodeInvariant,
					Path:     elem.Path,
					Message:  fmt.Sprintf("%s: invariant %s failed: %s", elem.Path, inv.Key, inv.Human),
				})
			}
		}
	}

	// Validate slicing constraints
	issues = append(issues, validateSlicing(resource, profile)...)

	return issues
}

// validateSlicing checks that array elements satisfy slice definitions.
func validateSlicing(resource resources.Resource, profile *Profile) []Issue {
	var issues []Issue

	// Find elements that define slicing
	slicedPaths := map[string]*Slicing{}     // base path → slicing definition
	slices := map[string][]ProfileElement{}   // base path → slice elements

	for i := range profile.Elements {
		elem := &profile.Elements[i]
		if elem.Slicing != nil {
			slicedPaths[elem.Path] = elem.Slicing
		}
		if elem.SliceName != "" {
			// Find the base path (remove :sliceName from the path)
			basePath := elem.Path
			if idx := strings.LastIndex(basePath, ":"); idx > 0 {
				basePath = basePath[:idx]
			} else {
				// The slice element may share the same path as the sliced element
				// (e.g. Patient.identifier with sliceName="SSN")
				basePath = elem.Path
			}
			slices[basePath] = append(slices[basePath], *elem)
		}
	}

	for basePath, slicing := range slicedPaths {
		sliceElems, hasSlices := slices[basePath]
		if !hasSlices {
			continue
		}

		// Get the array via FHIRPath
		parts := strings.SplitN(basePath, ".", 2)
		if len(parts) < 2 {
			continue
		}
		fieldPath := parts[1]

		result, err := fhirpath.Evaluate(resource, fieldPath)
		if err != nil || len(result) == 0 {
			// Check if any slice has min > 0
			for _, slice := range sliceElems {
				if slice.Min > 0 {
					issues = append(issues, Issue{
						Severity: SeverityError,
						Code:     CodeRequired,
						Path:     basePath + ":" + slice.SliceName,
						Message:  fmt.Sprintf("%s: slice %q requires min %d elements but array is empty", basePath, slice.SliceName, slice.Min),
					})
				}
			}
			continue
		}

		// Marshal each array element to JSON for matching
		arrayJSON := marshalArrayElements(result)

		// Match array elements to slices
		matched := make([]string, len(arrayJSON)) // which slice each element matched
		sliceCounts := map[string]int{}

		for i, elemJSON := range arrayJSON {
			for _, slice := range sliceElems {
				if matchesSlice(elemJSON, slice, slicing.Discriminators) {
					matched[i] = slice.SliceName
					sliceCounts[slice.SliceName]++
					break
				}
			}
		}

		// Check slice cardinality
		for _, slice := range sliceElems {
			count := sliceCounts[slice.SliceName]
			if slice.Min > 0 && count < slice.Min {
				issues = append(issues, Issue{
					Severity: SeverityError,
					Code:     CodeRequired,
					Path:     basePath + ":" + slice.SliceName,
					Message:  fmt.Sprintf("%s: slice %q requires min %d, found %d", basePath, slice.SliceName, slice.Min, count),
				})
			}
			if slice.Max != "*" && slice.Max != "" {
				var maxVal int
				fmt.Sscanf(slice.Max, "%d", &maxVal)
				if maxVal > 0 && count > maxVal {
					issues = append(issues, Issue{
						Severity: SeverityError,
						Code:     CodeStructure,
						Path:     basePath + ":" + slice.SliceName,
						Message:  fmt.Sprintf("%s: slice %q allows max %s, found %d", basePath, slice.SliceName, slice.Max, count),
					})
				}
			}
		}

		// For closed slicing, unmatched elements are an error
		if slicing.Rules == SlicingClosed {
			for i, sliceName := range matched {
				if sliceName == "" {
					issues = append(issues, Issue{
						Severity: SeverityError,
						Code:     CodeStructure,
						Path:     fmt.Sprintf("%s[%d]", basePath, i),
						Message:  fmt.Sprintf("%s[%d]: element does not match any defined slice (closed slicing)", basePath, i),
					})
				}
			}
		}
	}

	return issues
}

// marshalArrayElements converts FHIRPath results to JSON maps for matching.
func marshalArrayElements(results []any) []map[string]any {
	var out []map[string]any
	for _, r := range results {
		data, err := json.Marshal(r)
		if err != nil {
			out = append(out, nil)
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			out = append(out, nil)
			continue
		}
		out = append(out, m)
	}
	return out
}

// matchesSlice checks if a JSON element matches a slice definition based on discriminators.
func matchesSlice(elemJSON map[string]any, slice ProfileElement, discriminators []Discriminator) bool {
	if elemJSON == nil {
		return false
	}
	for _, disc := range discriminators {
		switch disc.Type {
		case DiscriminatorValue:
			if !matchDiscriminatorValue(elemJSON, disc.Path, slice) {
				return false
			}
		case DiscriminatorPattern:
			if !matchDiscriminatorPattern(elemJSON, disc.Path, slice) {
				return false
			}
		case DiscriminatorType_:
			if !matchDiscriminatorType(elemJSON, slice) {
				return false
			}
		case DiscriminatorExists:
			if !matchDiscriminatorExists(elemJSON, disc.Path) {
				return false
			}
		}
	}
	return true
}

// matchDiscriminatorValue matches by exact value at a path.
func matchDiscriminatorValue(elem map[string]any, path string, slice ProfileElement) bool {
	if slice.FixedValue == nil {
		return false
	}
	val := navigateJSON(elem, path)
	if val == nil {
		return false
	}
	// Compare via JSON serialization
	expectedJSON, _ := json.Marshal(slice.FixedValue)
	actualJSON, _ := json.Marshal(val)
	return string(expectedJSON) == string(actualJSON)
}

// matchDiscriminatorPattern matches by partial object match at a path.
func matchDiscriminatorPattern(elem map[string]any, path string, slice ProfileElement) bool {
	if slice.FixedValue == nil {
		return false
	}
	val := navigateJSON(elem, path)
	if val == nil {
		return false
	}
	// Pattern match: all fields in the pattern must match
	patternJSON, _ := json.Marshal(slice.FixedValue)
	var pattern map[string]any
	if err := json.Unmarshal(patternJSON, &pattern); err != nil {
		// Simple value pattern
		actualJSON, _ := json.Marshal(val)
		return string(patternJSON) == string(actualJSON)
	}
	actualMap, ok := val.(map[string]any)
	if !ok {
		return false
	}
	return mapContains(actualMap, pattern)
}

// matchDiscriminatorType matches by the type of the element.
func matchDiscriminatorType(elem map[string]any, slice ProfileElement) bool {
	if len(slice.Types) == 0 {
		return false
	}
	// Check if the element has a resourceType or system that matches
	for _, t := range slice.Types {
		if rt, ok := elem["resourceType"].(string); ok && rt == t {
			return true
		}
		// For CodeableConcept/Coding discriminators, check system
		if sys, ok := elem["system"].(string); ok && strings.Contains(sys, t) {
			return true
		}
	}
	return false
}

// matchDiscriminatorExists matches by whether a path exists.
func matchDiscriminatorExists(elem map[string]any, path string) bool {
	return navigateJSON(elem, path) != nil
}

// navigateJSON walks a JSON object by a dot-separated path.
func navigateJSON(obj map[string]any, path string) any {
	parts := strings.Split(path, ".")
	var current any = obj
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = m[part]
		if current == nil {
			return nil
		}
	}
	return current
}

// mapContains returns true if all keys in pattern exist in actual with equal values.
func mapContains(actual, pattern map[string]any) bool {
	for k, pv := range pattern {
		av, exists := actual[k]
		if !exists {
			return false
		}
		pMap, pIsMap := pv.(map[string]any)
		aMap, aIsMap := av.(map[string]any)
		if pIsMap && aIsMap {
			if !mapContains(aMap, pMap) {
				return false
			}
		} else {
			pJSON, _ := json.Marshal(pv)
			aJSON, _ := json.Marshal(av)
			if string(pJSON) != string(aJSON) {
				return false
			}
		}
	}
	return true
}
