// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package terminology

import "sync"

// InMemory is a terminology service backed by in-memory code system and
// value set definitions. Thread-safe for concurrent use.
type InMemory struct {
	mu        sync.RWMutex
	systems   map[string]*CodeSystem
	valueSets map[string]*ValueSet
}

// CodeSystem represents an in-memory code system.
type CodeSystem struct {
	URL      string
	Name     string
	Concepts map[string]*Concept // code → concept
}

// ValueSet represents an in-memory value set.
type ValueSet struct {
	URL      string
	Name     string
	Includes []ValueSetInclude
	// Expanded is the pre-computed flat list of codes.
	Expanded []Concept
}

// ValueSetInclude defines which codes from a code system are in the value set.
type ValueSetInclude struct {
	System   string
	Concepts []Concept // explicit concept list (if empty, includes all)
}

// NewInMemory creates an empty in-memory terminology service.
func NewInMemory() *InMemory {
	return &InMemory{
		systems:   make(map[string]*CodeSystem),
		valueSets: make(map[string]*ValueSet),
	}
}

// AddCodeSystem registers a code system.
func (m *InMemory) AddCodeSystem(cs *CodeSystem) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.systems[cs.URL] = cs
}

// AddValueSet registers a value set.
func (m *InMemory) AddValueSet(vs *ValueSet) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.valueSets[vs.URL] = vs
}

// ValidateCode checks if a code is valid in a code system or value set.
func (m *InMemory) ValidateCode(params ValidateCodeParams) *ValidateCodeResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// If ValueSetURL is specified, validate against that
	if params.ValueSetURL != "" {
		return m.validateAgainstValueSet(params)
	}

	// Validate against code system directly
	cs, ok := m.systems[params.System]
	if !ok {
		return &ValidateCodeResult{Valid: false, Message: "unknown code system: " + params.System}
	}

	concept, ok := cs.Concepts[params.Code]
	if !ok {
		return &ValidateCodeResult{
			Valid:   false,
			Message: params.Code + " not found in " + params.System,
		}
	}

	result := &ValidateCodeResult{Valid: true, Display: concept.Display}

	// Verify display if provided
	if params.Display != "" && concept.Display != "" && params.Display != concept.Display {
		result.Message = "display mismatch: expected " + concept.Display
	}

	return result
}

func (m *InMemory) validateAgainstValueSet(params ValidateCodeParams) *ValidateCodeResult {
	vs, ok := m.valueSets[params.ValueSetURL]
	if !ok {
		return &ValidateCodeResult{Valid: false, Message: "unknown value set: " + params.ValueSetURL}
	}

	// Check expanded list first
	for _, c := range vs.Expanded {
		if c.Code == params.Code && (params.System == "" || c.System == params.System) {
			return &ValidateCodeResult{Valid: true, Display: c.Display}
		}
	}

	// Check includes
	for _, inc := range vs.Includes {
		if params.System != "" && inc.System != params.System {
			continue
		}
		if len(inc.Concepts) == 0 {
			// Include all codes from system
			cs, ok := m.systems[inc.System]
			if ok {
				if concept, ok := cs.Concepts[params.Code]; ok {
					return &ValidateCodeResult{Valid: true, Display: concept.Display}
				}
			}
		} else {
			for _, c := range inc.Concepts {
				if c.Code == params.Code {
					return &ValidateCodeResult{Valid: true, Display: c.Display}
				}
			}
		}
	}

	return &ValidateCodeResult{
		Valid:   false,
		Message: params.Code + " not in value set " + params.ValueSetURL,
	}
}

// LookupCode returns details about a code.
func (m *InMemory) LookupCode(system, code string) *LookupResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cs, ok := m.systems[system]
	if !ok {
		return &LookupResult{Found: false}
	}
	concept, ok := cs.Concepts[code]
	if !ok {
		return &LookupResult{Found: false}
	}
	return &LookupResult{
		Found:      true,
		Display:    concept.Display,
		Definition: concept.Definition,
	}
}

// ExpandValueSet returns all codes in a value set.
func (m *InMemory) ExpandValueSet(url string) *ExpansionResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	vs, ok := m.valueSets[url]
	if !ok {
		return &ExpansionResult{Error: "unknown value set: " + url}
	}

	if len(vs.Expanded) > 0 {
		return &ExpansionResult{Concepts: vs.Expanded, Total: len(vs.Expanded)}
	}

	// Expand from includes
	var concepts []Concept
	for _, inc := range vs.Includes {
		if len(inc.Concepts) > 0 {
			concepts = append(concepts, inc.Concepts...)
		} else {
			cs, ok := m.systems[inc.System]
			if ok {
				for _, c := range cs.Concepts {
					concepts = append(concepts, *c)
				}
			}
		}
	}
	return &ExpansionResult{Concepts: concepts, Total: len(concepts)}
}

// Subsumes checks if codeA subsumes codeB (always not-subsumed for flat code systems).
func (m *InMemory) Subsumes(system, codeA, codeB string) SubsumptionResult {
	if codeA == codeB {
		return SubsumesEquivalent
	}
	return SubsumesNotSubsumed
}
