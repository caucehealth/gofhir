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
}

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
			Path: elem.Path,
			Min:  elem.Min,
			Max:  elem.Max,
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

	return issues
}
