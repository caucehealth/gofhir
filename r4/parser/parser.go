// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package parser

import (
	"encoding/json"
	"fmt"
)

// Options controls how FHIR resources are serialized and deserialized.
type Options struct {
	// PrettyPrint enables indented JSON output.
	PrettyPrint bool

	// SuppressNarrative removes the "text" field from serialized output.
	SuppressNarrative bool

	// SummaryMode includes only fields marked as summary elements in the
	// FHIR specification. When true, only id, meta, and fields defined as
	// summary in the spec are included. This is a simplified implementation
	// that removes large fields (text, contained, extension).
	SummaryMode bool

	// IncludeElements, if non-empty, restricts output to only these top-level
	// field names (plus resourceType which is always included).
	IncludeElements []string

	// ExcludeElements removes these top-level field names from output.
	ExcludeElements []string
}

// Marshal serializes a FHIR resource to JSON with the given options.
func Marshal(resource any, opts Options) ([]byte, error) {
	data, err := json.Marshal(resource)
	if err != nil {
		return nil, err
	}

	if !opts.needsPostProcess() {
		if opts.PrettyPrint {
			return prettyPrint(data)
		}
		return data, nil
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("post-processing: %w", err)
	}

	applyOptions(m, opts)

	if opts.PrettyPrint {
		return json.MarshalIndent(m, "", "  ")
	}
	return json.Marshal(m)
}

// Unmarshal deserializes FHIR JSON into the given resource.
// This is a convenience wrapper around json.Unmarshal.
func Unmarshal(data []byte, resource any) error {
	return json.Unmarshal(data, resource)
}

func (o Options) needsPostProcess() bool {
	return o.SuppressNarrative || o.SummaryMode ||
		len(o.IncludeElements) > 0 || len(o.ExcludeElements) > 0
}

func applyOptions(m map[string]json.RawMessage, opts Options) {
	if opts.SuppressNarrative {
		delete(m, "text")
	}

	if opts.SummaryMode {
		// Summary mode keeps: resourceType, id, meta, and removes
		// large/verbose fields
		summaryExclude := []string{
			"text", "contained", "extension", "modifierExtension",
		}
		for _, key := range summaryExclude {
			delete(m, key)
		}
	}

	if len(opts.IncludeElements) > 0 {
		include := make(map[string]bool)
		include["resourceType"] = true
		for _, e := range opts.IncludeElements {
			include[e] = true
		}
		for key := range m {
			if !include[key] {
				delete(m, key)
			}
		}
	}

	for _, key := range opts.ExcludeElements {
		delete(m, key)
	}
}

func prettyPrint(data []byte) ([]byte, error) {
	var m any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return json.MarshalIndent(m, "", "  ")
}
