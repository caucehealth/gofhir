// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package parser

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ErrorMode controls how parsing errors are handled.
type ErrorMode int

const (
	// Lenient silently captures unknown fields and ignores minor issues.
	// This is the default behavior.
	Lenient ErrorMode = iota
	// Strict returns errors for unknown fields in the input.
	Strict
)

// ParseError represents a parser-level error with categorization.
type ParseError struct {
	// Type categorizes the error.
	Type ParseErrorType
	// Field is the JSON field name that caused the error, if applicable.
	Field string
	// ResourceType is the FHIR resource type being parsed.
	ResourceType string
	// Message is a human-readable description.
	Message string
}

func (e *ParseError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s.%s: %s", e.Type, e.ResourceType, e.Field, e.Message)
	}
	return fmt.Sprintf("%s: %s: %s", e.Type, e.ResourceType, e.Message)
}

// ParseErrorType categorizes parser errors.
type ParseErrorType string

const (
	// ErrorUnknownField indicates an unrecognized JSON field.
	ErrorUnknownField ParseErrorType = "unknown_field"
	// ErrorInvalidValue indicates a value that doesn't match the expected type.
	ErrorInvalidValue ParseErrorType = "invalid_value"
)

// ParseErrors is a collection of parse errors.
type ParseErrors []ParseError

func (pe ParseErrors) Error() string {
	msgs := make([]string, len(pe))
	for i, e := range pe {
		msgs[i] = e.Error()
	}
	return strings.Join(msgs, "; ")
}

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

	// ErrorMode controls strictness of parsing. Default is Lenient.
	ErrorMode ErrorMode

	// StripVersionsFromReferences removes version suffixes from reference
	// values (e.g., "Patient/123/_history/1" becomes "Patient/123").
	StripVersionsFromReferences bool

	// OmitResourceId removes the "id" field from serialized output.
	// Useful when creating new resources via POST where the server assigns the id.
	OmitResourceId bool

	// OmitDefaults removes fields that contain their zero/default values
	// (false for booleans, 0 for numbers, "" for strings). Reduces payload size.
	OmitDefaults bool
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

// UnmarshalWithOptions deserializes FHIR JSON with configurable error handling.
// In Strict mode, returns ParseErrors if unknown fields are present.
func UnmarshalWithOptions(data []byte, resource any, opts Options) error {
	if err := json.Unmarshal(data, resource); err != nil {
		return err
	}

	if opts.ErrorMode == Strict {
		if errs := checkUnknownFields(data, resource); len(errs) > 0 {
			return errs
		}
	}

	return nil
}

// extraHolder is implemented by generated resources that capture unknown fields.
type extraHolder interface {
	GetExtra() map[string]json.RawMessage
}

// checkUnknownFields returns parse errors for any unknown fields in the input.
func checkUnknownFields(data []byte, resource any) ParseErrors {
	holder, ok := resource.(extraHolder)
	if !ok {
		return nil
	}
	extra := holder.GetExtra()
	if len(extra) == 0 {
		return nil
	}

	// Determine resource type
	var header struct {
		ResourceType string `json:"resourceType"`
	}
	json.Unmarshal(data, &header)

	var errs ParseErrors
	for field := range extra {
		errs = append(errs, ParseError{
			Type:         ErrorUnknownField,
			Field:        field,
			ResourceType: header.ResourceType,
			Message:      fmt.Sprintf("unrecognized field %q", field),
		})
	}
	return errs
}

func (o Options) needsPostProcess() bool {
	return o.SuppressNarrative || o.SummaryMode ||
		len(o.IncludeElements) > 0 || len(o.ExcludeElements) > 0 ||
		o.StripVersionsFromReferences || o.OmitResourceId || o.OmitDefaults
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

	if opts.OmitResourceId {
		delete(m, "id")
		delete(m, "_id")
	}

	if opts.OmitDefaults {
		omitDefaults(m)
	}

	if opts.StripVersionsFromReferences {
		stripVersions(m)
	}
}

// omitDefaults removes fields with default/zero values: false, 0, "".
func omitDefaults(m map[string]json.RawMessage) {
	for key, val := range m {
		s := strings.TrimSpace(string(val))
		if s == "false" || s == "0" || s == `""` || s == "0.0" {
			delete(m, key)
			// Also remove companion _field if present
			delete(m, "_"+key)
		}
	}
}

// stripVersions removes /_history/N from reference values in the resource.
func stripVersions(m map[string]json.RawMessage) {
	for key, val := range m {
		s := string(val)
		// Check if this looks like a reference field containing /_history/
		if strings.Contains(s, "/_history/") {
			if len(s) >= 2 && s[0] == '"' {
				// Simple string value — strip version
				unquoted := s[1 : len(s)-1]
				if idx := strings.Index(unquoted, "/_history/"); idx != -1 {
					stripped := unquoted[:idx]
					m[key], _ = json.Marshal(stripped)
				}
			} else if s[0] == '{' {
				// Nested object — recurse
				var nested map[string]json.RawMessage
				if json.Unmarshal(val, &nested) == nil {
					stripVersions(nested)
					m[key], _ = json.Marshal(nested)
				}
			}
		}
	}
}

func prettyPrint(data []byte) ([]byte, error) {
	var m any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return json.MarshalIndent(m, "", "  ")
}
