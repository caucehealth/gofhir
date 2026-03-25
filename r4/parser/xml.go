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

// inferChildType returns the FHIR type name for a nested element using
// schema-generated metadata. Falls back to a heuristic if no metadata found.
func inferChildType(parentType, fieldName string) string {
	// Use schema-generated field type map
	if ft := FieldType(parentType, fieldName); ft != "" {
		return ft
	}
	// Fallback for backbone elements: ParentFieldName
	return ""
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
	for key, val := range m {
		if strings.HasPrefix(key, "_") {
			continue // handled alongside their primitive via m["_"+key]
		}
		if key == "resourceType" {
			continue // already emitted as root element name
		}
		// XHTML div content — embed as raw XML, not as value attribute
		if key == "div" {
			if s, ok := val.(string); ok {
				e.writeIndent()
				e.buf.WriteString(s)
				e.writeNewline()
				continue
			}
		}
		e.encodeField(key, val, m["_"+key])
	}
}

func (e *xmlEncoder) encodeField(name string, val any, elemExt any) {
	switch v := val.(type) {
	case nil:
		// skip
	case string:
		if elemExt != nil {
			// Primitive with element extensions — container element
			e.writeStart(name, map[string]string{"value": v})
			if ext, ok := elemExt.(map[string]any); ok {
				e.encodeElementExtensions(ext)
			}
			e.writeEnd(name)
		} else {
			e.writeValueElement(name, v)
		}
	case float64:
		s := fmt.Sprintf("%v", v)
		if elemExt != nil {
			e.writeStart(name, map[string]string{"value": s})
			if ext, ok := elemExt.(map[string]any); ok {
				e.encodeElementExtensions(ext)
			}
			e.writeEnd(name)
		} else {
			e.writeValueElement(name, s)
		}
	case bool:
		s := "false"
		if v {
			s = "true"
		}
		if elemExt != nil {
			e.writeStart(name, map[string]string{"value": s})
			if ext, ok := elemExt.(map[string]any); ok {
				e.encodeElementExtensions(ext)
			}
			e.writeEnd(name)
		} else {
			e.writeValueElement(name, s)
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

// encodeElementExtensions writes the id and extension children of an element extension.
func (e *xmlEncoder) encodeElementExtensions(ext map[string]any) {
	if id, ok := ext["id"]; ok {
		if idStr, ok := id.(string); ok {
			e.writeValueElement("id", idStr)
		}
	}
	if exts, ok := ext["extension"]; ok {
		e.encodeField("extension", exts, nil)
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

			if name == "div" {
				divContent, err := readRawXMLElement(decoder, t)
				if err != nil {
					return nil, err
				}
				addToMap(m, "div", divContent)
			} else if hasValue {
				// Primitive with value — may also have child extensions
				addToMap(m, name, coerceXMLValue(resourceType, name, valueAttr))
				elemExt := decodePrimitiveExtensions(decoder)
				if elemExt != nil {
					m["_"+name] = elemExt
				}
			} else {
				childType := inferChildType(resourceType, name)
				child, err := decodeXMLChild(decoder, childType)
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

func decodeXMLChild(decoder *xml.Decoder, typeName string) (any, error) {
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

			if name == "div" {
				divContent, err := readRawXMLElement(decoder, t)
				if err != nil {
					return nil, err
				}
				addToMap(m, "div", divContent)
			} else if hasValue {
				addToMap(m, name, coerceXMLValue(typeName, name, valueAttr))
				elemExt := decodePrimitiveExtensions(decoder)
				if elemExt != nil {
					m["_"+name] = elemExt
				}
			} else {
				childType := inferChildType(typeName, name)
				child, err := decodeXMLChild(decoder, childType)
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

// coerceXMLValue converts string values from XML value="" attributes to
// their appropriate JSON types using schema-generated metadata.
func coerceXMLValue(parentType, fieldName, s string) any {
	// Boolean fields — only when schema says it's boolean or value is exactly true/false
	// and the field is known to be boolean in context
	if (s == "true" || s == "false") && isBooleanContext(parentType, fieldName) {
		return s == "true"
	}
	// Use schema-generated numeric field metadata — ONLY coerce when schema confirms
	if IsNumericField(parentType, fieldName) {
		if len(s) > 0 && (s[0] == '-' || s[0] == '.' || (s[0] >= '0' && s[0] <= '9')) {
			var f float64
			if _, err := fmt.Sscanf(s, "%g", &f); err == nil {
				return f
			}
		}
	}
	return s
}

// isBooleanContext checks if a field is expected to be boolean.
// Uses known FHIR boolean field names.
func isBooleanContext(parentType, fieldName string) bool {
	boolFields := map[string]bool{
		"active": true, "focal": true, "allDay": true,
		"primarySource": true, "isSubpotent": true, "doNotPerform": true,
		"experimental": true, "abstract": true, "required": true,
		"repeats": true, "readOnly": true, "multiple": true,
		"preferred": true, "implicitRules": false,
	}
	if v, ok := boolFields[fieldName]; ok {
		return v
	}
	// Fields starting with "is", "has", "can", or ending with "Boolean"
	if strings.HasPrefix(fieldName, "is") || strings.HasPrefix(fieldName, "has") ||
		strings.HasSuffix(fieldName, "Boolean") {
		return true
	}
	// Check for known value[x] boolean variants
	if fieldName == "deceasedBoolean" || fieldName == "multipleBirthBoolean" ||
		fieldName == "asNeededBoolean" || fieldName == "allowedBoolean" ||
		fieldName == "reportedBoolean" {
		return true
	}
	return false
}

// decodePrimitiveExtensions reads any child elements inside a primitive element
// (id, extension) and returns them as a map for the _field companion.
// If there are no children (self-closing element), returns nil.
func decodePrimitiveExtensions(decoder *xml.Decoder) map[string]any {
	m := make(map[string]any)
	hasContent := false

	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := token.(type) {
		case xml.StartElement:
			hasContent = true
			name := t.Name.Local
			if name == "extension" {
				child, err := decodeXMLChild(decoder, "Extension")
				if err == nil {
					for _, attr := range t.Attr {
						if attr.Name.Local == "url" {
							if cm, ok := child.(map[string]any); ok {
								cm["url"] = attr.Value
							}
						}
					}
					// Always store extensions as array
					if existing, ok := m["extension"]; ok {
						if arr, ok := existing.([]any); ok {
							m["extension"] = append(arr, child)
						}
					} else {
						m["extension"] = []any{child}
					}
				}
			} else if name == "id" {
				for _, attr := range t.Attr {
					if attr.Name.Local == "value" {
						m["id"] = attr.Value
					}
				}
				decoder.Skip()
			} else {
				decoder.Skip()
			}
		case xml.EndElement:
			if !hasContent {
				return nil
			}
			return m
		case xml.CharData:
			// ignore whitespace
		}
	}
	if !hasContent {
		return nil
	}
	return m
}

// readRawXMLElement reads all content of an XML element (including nested elements)
// as a raw XHTML string. Used for the narrative div element.
func readRawXMLElement(decoder *xml.Decoder, start xml.StartElement) (string, error) {
	var buf strings.Builder
	// Reconstruct the opening tag
	buf.WriteByte('<')
	buf.WriteString(start.Name.Local)
	for _, attr := range start.Attr {
		buf.WriteByte(' ')
		if attr.Name.Space != "" {
			buf.WriteString(attr.Name.Space)
			buf.WriteByte(':')
		}
		buf.WriteString(attr.Name.Local)
		buf.WriteString(`="`)
		buf.WriteString(xmlEscape(attr.Value))
		buf.WriteByte('"')
	}
	buf.WriteByte('>')

	depth := 1
	for depth > 0 {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := token.(type) {
		case xml.StartElement:
			depth++
			buf.WriteByte('<')
			buf.WriteString(t.Name.Local)
			for _, attr := range t.Attr {
				buf.WriteByte(' ')
				buf.WriteString(attr.Name.Local)
				buf.WriteString(`="`)
				buf.WriteString(xmlEscape(attr.Value))
				buf.WriteByte('"')
			}
			buf.WriteByte('>')
		case xml.EndElement:
			depth--
			if depth >= 0 {
				buf.WriteString("</")
				buf.WriteString(t.Name.Local)
				buf.WriteByte('>')
			}
		case xml.CharData:
			buf.WriteString(xmlEscape(string(t)))
		}
	}
	return buf.String(), nil
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
