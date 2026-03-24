// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package datatypes

import "strings"

// ExtensionsByURL returns all extensions matching the given URL.
func ExtensionsByURL(exts []Extension, url string) []Extension {
	var result []Extension
	for _, ext := range exts {
		if string(ext.Url) == url {
			result = append(result, ext)
		}
	}
	return result
}

// ExtensionByURL returns the first extension matching the given URL, or nil.
func ExtensionByURL(exts []Extension, url string) *Extension {
	for i := range exts {
		if string(exts[i].Url) == url {
			return &exts[i]
		}
	}
	return nil
}

// ResourceID represents a parsed FHIR resource identity.
type ResourceID struct {
	// Type is the resource type (e.g. "Patient").
	Type string
	// ID is the logical id (e.g. "123").
	ID string
	// Version is the version id, if present (e.g. "2").
	Version string
}

// String returns the ResourceID as a FHIR reference string.
func (r ResourceID) String() string {
	if r.Version != "" {
		return r.Type + "/" + r.ID + "/_history/" + r.Version
	}
	return r.Type + "/" + r.ID
}

// ParseResourceID parses a FHIR reference string like "Patient/123" or
// "Patient/123/_history/2" into its component parts. Returns an empty
// ResourceID if the format is not recognized.
func ParseResourceID(ref string) ResourceID {
	ref = strings.TrimSpace(ref)

	// Strip leading URL (e.g. "http://example.com/fhir/Patient/123")
	if idx := strings.LastIndex(ref, "/"); idx != -1 {
		// Check if this is a full URL with resource type
		parts := strings.Split(ref, "/")
		// Find the resource type / id pattern working backwards
		for i := len(parts) - 1; i >= 1; i-- {
			if parts[i-1] == "_history" && i >= 3 {
				return ResourceID{
					Type:    parts[i-3],
					ID:      parts[i-2],
					Version: parts[i],
				}
			}
		}
		// Simple Type/ID pattern — find last two segments
		if len(parts) >= 2 {
			resType := parts[len(parts)-2]
			id := parts[len(parts)-1]
			if resType != "" && id != "" && resType[0] >= 'A' && resType[0] <= 'Z' {
				return ResourceID{Type: resType, ID: id}
			}
		}
	}
	return ResourceID{}
}

// NewReference creates a Reference from a resource type and ID.
func NewReference(resourceType, id string) Reference {
	ref := resourceType + "/" + id
	return Reference{Reference: &ref}
}

// NewReferenceWithDisplay creates a Reference with a display string.
func NewReferenceWithDisplay(resourceType, id, display string) Reference {
	ref := resourceType + "/" + id
	return Reference{Reference: &ref, Display: &display}
}
