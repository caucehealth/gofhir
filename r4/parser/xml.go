// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package parser

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
)

const fhirNamespace = "http://hl7.org/fhir"

// MarshalXML serializes a FHIR resource to XML. The resource is first
// marshaled to JSON (the canonical in-memory format), then converted to
// FHIR-conformant XML with the hl7.org/fhir namespace.
func MarshalXML(resource any, opts Options) ([]byte, error) {
	// Marshal to JSON first (canonical format)
	jsonData, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("xml: json marshal: %w", err)
	}

	var m map[string]any
	if err := json.Unmarshal(jsonData, &m); err != nil {
		return nil, fmt.Errorf("xml: json unmarshal: %w", err)
	}

	if opts.SuppressNarrative {
		delete(m, "text")
	}

	resourceType, _ := m["resourceType"].(string)
	if resourceType == "" {
		return nil, fmt.Errorf("xml: missing resourceType")
	}
	delete(m, "resourceType")

	var buf strings.Builder
	buf.WriteString(xml.Header)

	indent := ""
	if opts.PrettyPrint {
		indent = "  "
	}

	encoder := &xmlEncoder{
		buf:    &buf,
		indent: indent,
		level:  0,
		pretty: opts.PrettyPrint,
	}

	encoder.writeStart(resourceType, map[string]string{"xmlns": fhirNamespace})
	encoder.encodeMap(m)
	encoder.writeEnd(resourceType)

	return []byte(buf.String()), nil
}

// UnmarshalXML deserializes FHIR XML into a resource. The XML is first
// converted to FHIR JSON, then unmarshaled into the target struct.
// Single XML elements that map to array fields are automatically wrapped.
func UnmarshalXML(data []byte, resource any) error {
	m, err := xmlToMap(data)
	if err != nil {
		return fmt.Errorf("xml: parse: %w", err)
	}

	// Use schema metadata to fix array fields
	resourceType, _ := m["resourceType"].(string)
	fixArrayFields(m, resourceType)

	jsonData, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("xml: json re-convert: %w", err)
	}

	return json.Unmarshal(jsonData, resource)
}

// fixArrayFields uses schema metadata to wrap single values in arrays where
// the FHIR spec defines the field as repeating. Recurses into nested maps.
func fixArrayFields(m map[string]any, typeName string) {
	for k, v := range m {
		if strings.HasPrefix(k, "_") || k == "resourceType" {
			continue
		}
		switch val := v.(type) {
		case map[string]any:
			// Determine child type name for recursion
			childType := inferChildType(typeName, k)
			fixArrayFields(val, childType)
			if IsArrayField(typeName, k) {
				m[k] = []any{val}
			}
		case string:
			if IsArrayField(typeName, k) {
				m[k] = []any{val}
			}
		case float64:
			if IsArrayField(typeName, k) {
				m[k] = []any{val}
			}
		case bool:
			if IsArrayField(typeName, k) {
				m[k] = []any{val}
			}
		case []any:
			for _, item := range val {
				if sub, ok := item.(map[string]any); ok {
					childType := inferChildType(typeName, k)
					fixArrayFields(sub, childType)
				}
			}
		}
	}
}

// inferChildType guesses the FHIR type name for a nested element.
func inferChildType(parentType, fieldName string) string {
	// Common complex types
	switch fieldName {
	case "name":
		return "HumanName"
	case "telecom":
		return "ContactPoint"
	case "address":
		return "Address"
	case "identifier":
		return "Identifier"
	case "coding":
		return "Coding"
	case "code":
		if parentType != "" {
			return "CodeableConcept"
		}
	case "meta":
		return "Meta"
	case "text":
		return "Narrative"
	case "period":
		return "Period"
	case "quantity", "valueQuantity":
		return "Quantity"
	}
	// Default: try ParentFieldName pattern
	return parentType + exportFieldName(fieldName)
}

func exportFieldName(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = []rune(strings.ToUpper(string(r[0])))[0]
	return string(r)
}

type xmlEncoder struct {
	buf    *strings.Builder
	indent string
	level  int
	pretty bool
}

func (e *xmlEncoder) writeIndent() {
	if e.pretty {
		for i := 0; i < e.level; i++ {
			e.buf.WriteString(e.indent)
		}
	}
}

func (e *xmlEncoder) writeNewline() {
	if e.pretty {
		e.buf.WriteByte('\n')
	}
}

func (e *xmlEncoder) writeStart(name string, attrs map[string]string) {
	e.writeIndent()
	e.buf.WriteByte('<')
	e.buf.WriteString(name)
	for k, v := range attrs {
		e.buf.WriteByte(' ')
		e.buf.WriteString(k)
		e.buf.WriteString(`="`)
		e.buf.WriteString(xmlEscape(v))
		e.buf.WriteByte('"')
	}
	e.buf.WriteByte('>')
	e.writeNewline()
	e.level++
}

func (e *xmlEncoder) writeEnd(name string) {
	e.level--
	e.writeIndent()
	e.buf.WriteString("</")
	e.buf.WriteString(name)
	e.buf.WriteByte('>')
	e.writeNewline()
}

func (e *xmlEncoder) writeValueElement(name, value string) {
	e.writeIndent()
	e.buf.WriteByte('<')
	e.buf.WriteString(name)
	e.buf.WriteString(` value="`)
	e.buf.WriteString(xmlEscape(value))
	e.buf.WriteString(`"/>`)
	e.writeNewline()
}

func (e *xmlEncoder) encodeMap(m map[string]any) {
	// FHIR XML field order matters less, but we process in insertion order
	for key, val := range m {
		if strings.HasPrefix(key, "_") {
			continue // element extensions handled alongside their primitive
		}
		e.encodeField(key, val, m["_"+key])
	}
}

func (e *xmlEncoder) encodeField(name string, val any, elemExt any) {
	switch v := val.(type) {
	case nil:
		// skip
	case string:
		e.writeValueElement(name, v)
	case float64:
		e.writeValueElement(name, fmt.Sprintf("%v", v))
	case bool:
		if v {
			e.writeValueElement(name, "true")
		} else {
			e.writeValueElement(name, "false")
		}
	case map[string]any:
		e.writeStart(name, nil)
		e.encodeMap(v)
		e.writeEnd(name)
	case []any:
		for _, item := range v {
			e.encodeField(name, item, nil)
		}
	}
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// xmlToMap converts FHIR XML to a JSON-compatible map.
func xmlToMap(data []byte) (map[string]any, error) {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	m, err := decodeXMLElement(decoder)
	if err != nil {
		return nil, err
	}
	result, ok := m.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected map, got %T", m)
	}
	return result, nil
}

func decodeXMLElement(decoder *xml.Decoder) (any, error) {
	m := make(map[string]any)
	var resourceType string

	for {
		token, err := decoder.Token()
		if err != nil {
			return m, nil
		}

		switch t := token.(type) {
		case xml.StartElement:
			name := t.Name.Local
			if resourceType == "" {
				resourceType = name
				m["resourceType"] = name
				// Process attributes
				for _, attr := range t.Attr {
					if attr.Name.Local == "xmlns" {
						continue
					}
				}
				continue
			}

			// Check for value attribute (FHIR primitives)
			var valueAttr string
			hasValue := false
			for _, attr := range t.Attr {
				if attr.Name.Local == "value" {
					valueAttr = attr.Value
					hasValue = true
				}
			}

			if hasValue {
				// Self-closing primitive element
				decoder.Skip()
				addToMap(m, name, valueAttr)
			} else {
				// Complex child element
				child, err := decodeXMLChild(decoder)
				if err != nil {
					return nil, err
				}
				addToMap(m, name, child)
			}

		case xml.EndElement:
			return m, nil
		}
	}
}

func decodeXMLChild(decoder *xml.Decoder) (any, error) {
	m := make(map[string]any)

	for {
		token, err := decoder.Token()
		if err != nil {
			return m, nil
		}

		switch t := token.(type) {
		case xml.StartElement:
			name := t.Name.Local

			var valueAttr string
			hasValue := false
			for _, attr := range t.Attr {
				if attr.Name.Local == "value" {
					valueAttr = attr.Value
					hasValue = true
				}
			}

			if hasValue {
				decoder.Skip()
				addToMap(m, name, valueAttr)
			} else {
				child, err := decodeXMLChild(decoder)
				if err != nil {
					return nil, err
				}
				addToMap(m, name, child)
			}

		case xml.EndElement:
			return m, nil

		case xml.CharData:
			// Text content in FHIR XML (e.g. div content)
			text := strings.TrimSpace(string(t))
			if text != "" {
				m["div"] = text
			}
		}
	}
}

func addToMap(m map[string]any, key string, val any) {
	existing, ok := m[key]
	if !ok {
		m[key] = val
		return
	}
	// Multiple elements with same name → make array
	switch e := existing.(type) {
	case []any:
		m[key] = append(e, val)
	default:
		m[key] = []any{e, val}
	}
}
