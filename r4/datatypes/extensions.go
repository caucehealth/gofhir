// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package datatypes

import "sync"

// ExtensionDef describes a registered extension with its expected value type.
type ExtensionDef struct {
	// URL is the canonical URL of the extension.
	URL string
	// Name is a short human-readable name (e.g., "birthTime").
	Name string
	// ValueType is the expected Go type name (e.g., "DateTime", "CodeableConcept").
	ValueType string
	// IsModifier indicates this is a modifier extension.
	IsModifier bool
}

// ExtensionRegistry maps known extension URLs to their definitions.
// Thread-safe for concurrent read/write. Intended to be populated at
// application startup with extensions from profiles (e.g., US Core).
type ExtensionRegistry struct {
	mu   sync.RWMutex
	defs map[string]*ExtensionDef
}

// NewExtensionRegistry creates an empty registry.
func NewExtensionRegistry() *ExtensionRegistry {
	return &ExtensionRegistry{defs: make(map[string]*ExtensionDef)}
}

// Register adds an extension definition to the registry.
func (r *ExtensionRegistry) Register(def ExtensionDef) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defs[def.URL] = &def
}

// Lookup returns the definition for a URL, or nil if not registered.
func (r *ExtensionRegistry) Lookup(url string) *ExtensionDef {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.defs[url]
}

// IsKnown returns true if the extension URL is registered.
func (r *ExtensionRegistry) IsKnown(url string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.defs[url]
	return ok
}

// All returns all registered extension definitions.
func (r *ExtensionRegistry) All() []ExtensionDef {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ExtensionDef, 0, len(r.defs))
	for _, d := range r.defs {
		result = append(result, *d)
	}
	return result
}

// GetExtensionValue extracts a typed value from an extension by URL.
// Returns nil if the extension is not found or has no matching value.
func GetExtensionValue[T any](exts []Extension, url string) *T {
	ext := ExtensionByURL(exts, url)
	if ext == nil {
		return nil
	}
	return extractValue[T](ext)
}

func extractValue[T any](ext *Extension) *T {
	// Try each value field — the caller's type parameter determines the match
	candidates := []any{
		ext.ValueString, ext.ValueBoolean, ext.ValueInteger,
		ext.ValueDecimal, ext.ValueUri, ext.ValueCode,
		ext.ValueDate, ext.ValueDateTime, ext.ValueInstant,
		ext.ValueTime, ext.ValueId, ext.ValueMarkdown,
		ext.ValueOid, ext.ValueUuid, ext.ValueUrl,
		ext.ValueCanonical, ext.ValuePositiveInt, ext.ValueUnsignedInt,
	}
	for _, c := range candidates {
		if c == nil {
			continue
		}
		if v, ok := c.(*T); ok && v != nil {
			return v
		}
	}
	// Try complex types
	complexCandidates := []any{
		ext.ValueCoding, ext.ValueCodeableConcept,
		ext.ValueQuantity, ext.ValueReference,
		ext.ValuePeriod, ext.ValueRange,
		ext.ValueIdentifier, ext.ValueAddress,
		ext.ValueHumanName, ext.ValueContactPoint,
		ext.ValueAttachment, ext.ValueAnnotation,
		ext.ValueMoney, ext.ValueAge,
		ext.ValueSignature,
	}
	for _, c := range complexCandidates {
		if c == nil {
			continue
		}
		if v, ok := c.(*T); ok && v != nil {
			return v
		}
	}
	return nil
}

// USCoreExtensions returns a registry pre-populated with common US Core extensions.
func USCoreExtensions() *ExtensionRegistry {
	r := NewExtensionRegistry()
	r.Register(ExtensionDef{
		URL:       "http://hl7.org/fhir/us/core/StructureDefinition/us-core-race",
		Name:      "race",
		ValueType: "CodeableConcept",
	})
	r.Register(ExtensionDef{
		URL:       "http://hl7.org/fhir/us/core/StructureDefinition/us-core-ethnicity",
		Name:      "ethnicity",
		ValueType: "CodeableConcept",
	})
	r.Register(ExtensionDef{
		URL:       "http://hl7.org/fhir/us/core/StructureDefinition/us-core-birthsex",
		Name:      "birthsex",
		ValueType: "Code",
	})
	r.Register(ExtensionDef{
		URL:       "http://hl7.org/fhir/us/core/StructureDefinition/us-core-genderIdentity",
		Name:      "genderIdentity",
		ValueType: "CodeableConcept",
	})
	r.Register(ExtensionDef{
		URL:       "http://hl7.org/fhir/StructureDefinition/patient-birthTime",
		Name:      "birthTime",
		ValueType: "DateTime",
	})
	return r
}
