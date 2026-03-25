// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package parser

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"reflect"
	"strings"
)

const fhirNamespace = "http://hl7.org/fhir"

// MarshalXML serializes a FHIR resource to XML using direct struct reflection.
// No JSON intermediary is used — struct fields are walked directly.
func MarshalXML(resource any, opts Options) ([]byte, error) {
	v := reflect.ValueOf(resource)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, fmt.Errorf("xml: nil resource")
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("xml: expected struct, got %s", v.Kind())
	}

	// Get resourceType from the struct's ResourceType field
	rtField := v.FieldByName("ResourceType")
	if !rtField.IsValid() || rtField.String() == "" {
		return nil, fmt.Errorf("xml: missing ResourceType field")
	}
	resourceType := rtField.String()

	var buf strings.Builder
	buf.WriteString(xml.Header)

	indent := ""
	if opts.PrettyPrint {
		indent = "  "
	}

	e := &xmlEncoder{
		buf:    &buf,
		indent: indent,
		level:  0,
		pretty: opts.PrettyPrint,
	}

	e.writeStart(resourceType, map[string]string{"xmlns": fhirNamespace})
	e.encodeStruct(v, opts)
	e.writeEnd(resourceType)

	return []byte(buf.String()), nil
}

// encodeStruct writes all fields of a struct as XML child elements.
func (e *xmlEncoder) encodeStruct(v reflect.Value, opts Options) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldVal := v.Field(i)

		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			// Handle json:"-" fields: Extra and value[x] unions
			e.handleDashField(field, fieldVal, opts)
			continue
		}

		name, omitempty := parseTag(tag)
		if name == "resourceType" {
			continue
		}

		// Skip _field companions — they're handled with their primitive
		if strings.HasPrefix(name, "_") {
			continue
		}

		// Options filtering
		if opts.SuppressNarrative && name == "text" {
			continue
		}

		if omitempty && isZero(fieldVal) {
			continue
		}

		// Find companion _field for element extensions
		companion := findCompanion(v, t, name)

		e.encodeFieldValue(name, fieldVal, companion, opts)
	}
}

// handleDashField handles json:"-" fields: Extra map and value[x] unions.
func (e *xmlEncoder) handleDashField(field reflect.StructField, val reflect.Value, opts Options) {
	if field.Name == "Extra" {
		// Encode unknown fields from Extra map
		if val.IsNil() {
			return
		}
		iter := val.MapRange()
		for iter.Next() {
			key := iter.Key().String()
			raw := iter.Value().Interface().(json.RawMessage)
			e.encodeRawJSON(key, raw, opts)
		}
		return
	}

	// value[x] union fields — has MarshalJSON that produces flat key-value pairs
	if val.Kind() == reflect.Ptr && val.IsNil() {
		return
	}

	// Call MarshalJSON on the union struct
	marshaler, ok := val.Interface().(json.Marshaler)
	if !ok && val.Kind() == reflect.Ptr {
		marshaler, ok = val.Elem().Interface().(json.Marshaler)
	}
	if !ok {
		return
	}

	data, err := marshaler.MarshalJSON()
	if err != nil {
		return
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return
	}

	for key, raw := range m {
		e.encodeRawJSON(key, raw, opts)
	}
}

// encodeRawJSON encodes a raw JSON value as XML.
// Uses json.Decoder with UseNumber to preserve decimal precision.
func (e *xmlEncoder) encodeRawJSON(name string, raw json.RawMessage, opts Options) {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	var val any
	if err := dec.Decode(&val); err != nil {
		return
	}
	e.encodeAnyValue(name, val, nil, opts)
}

// encodeFieldValue writes a single struct field as XML.
func (e *xmlEncoder) encodeFieldValue(name string, val reflect.Value, companion reflect.Value, opts Options) {
	// Dereference pointer
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.String:
		s := val.String()
		if s == "" {
			return
		}
		e.encodePrimitive(name, s, companion)

	case reflect.Bool:
		s := "false"
		if val.Bool() {
			s = "true"
		}
		e.encodePrimitive(name, s, companion)

	case reflect.Int32:
		s := fmt.Sprintf("%d", val.Int())
		e.encodePrimitive(name, s, companion)

	case reflect.Uint32:
		s := fmt.Sprintf("%d", val.Uint())
		e.encodePrimitive(name, s, companion)

	case reflect.Float64:
		s := fmt.Sprintf("%v", val.Float())
		e.encodePrimitive(name, s, companion)

	case reflect.Struct:
		e.writeStart(name, nil)
		e.encodeStruct(val, opts)
		e.writeEnd(name)

	case reflect.Slice:
		if val.Type().Elem().Kind() == reflect.Uint8 {
			// []byte — base64 encoded, but json.Marshal handles this
			// For XML, encode as value attribute
			data, _ := json.Marshal(val.Interface())
			// Remove quotes from JSON string
			s := strings.Trim(string(data), `"`)
			e.encodePrimitive(name, s, companion)
		} else {
			for j := 0; j < val.Len(); j++ {
				e.encodeFieldValue(name, val.Index(j), reflect.Value{}, opts)
			}
		}

	case reflect.Interface:
		// json.RawMessage stored as []byte in interface
		if raw, ok := val.Interface().(json.RawMessage); ok {
			e.encodeRawJSON(name, raw, opts)
		}

	case reflect.Map:
		// Skip maps (like Extra) — handled separately
	}
}

// encodePrimitive writes a FHIR primitive as <name value="..."/> or with
// element extensions if a companion _field exists.
func (e *xmlEncoder) encodePrimitive(name, value string, companion reflect.Value) {
	// Check for XHTML div — embed raw, not as value attribute
	if name == "div" {
		e.writeIndent()
		e.buf.WriteString(value)
		e.writeNewline()
		return
	}

	hasCompanion := companion.IsValid() && companion.Kind() == reflect.Ptr && !companion.IsNil()
	if hasCompanion {
		e.writeStart(name, map[string]string{"value": value})
		e.encodeElementExtension(companion.Elem())
		e.writeEnd(name)
	} else {
		e.writeValueElement(name, value)
	}
}

// encodeElementExtension writes the id and extension children from an Element struct.
func (e *xmlEncoder) encodeElementExtension(elem reflect.Value) {
	if elem.Kind() != reflect.Struct {
		return
	}

	// Element has Id *string and Extension []Extension
	idField := elem.FieldByName("Id")
	if idField.IsValid() && idField.Kind() == reflect.Ptr && !idField.IsNil() {
		e.writeValueElement("id", idField.Elem().String())
	}

	extField := elem.FieldByName("Extension")
	if extField.IsValid() && extField.Kind() == reflect.Slice && extField.Len() > 0 {
		for i := 0; i < extField.Len(); i++ {
			ext := extField.Index(i)
			e.writeStart("extension", nil)
			e.encodeStruct(ext, Options{})
			e.writeEnd("extension")
		}
	}
}

// encodeAnyValue encodes a generic any value (from JSON unmarshal) as XML.
// Used for Extra fields and value[x] union content.
func (e *xmlEncoder) encodeAnyValue(name string, val any, elemExt any, opts Options) {
	switch v := val.(type) {
	case nil:
		// skip
	case string:
		if name == "div" {
			e.writeIndent()
			e.buf.WriteString(v)
			e.writeNewline()
		} else if elemExt != nil {
			e.writeStart(name, map[string]string{"value": v})
			if ext, ok := elemExt.(map[string]any); ok {
				e.encodeAnyElementExtensions(ext)
			}
			e.writeEnd(name)
		} else {
			e.writeValueElement(name, v)
		}
	case json.Number:
		// Preserves exact decimal precision (e.g., "1.00" stays "1.00")
		e.writeValueElement(name, v.String())
	case float64:
		s := fmt.Sprintf("%v", v)
		e.writeValueElement(name, s)
	case bool:
		s := "false"
		if v {
			s = "true"
		}
		e.writeValueElement(name, s)
	case map[string]any:
		e.writeStart(name, nil)
		e.encodeAnyMap(v, opts)
		e.writeEnd(name)
	case []any:
		for _, item := range v {
			e.encodeAnyValue(name, item, nil, opts)
		}
	}
}

// encodeAnyMap encodes a generic map as XML children.
func (e *xmlEncoder) encodeAnyMap(m map[string]any, opts Options) {
	for key, val := range m {
		if strings.HasPrefix(key, "_") || key == "resourceType" {
			continue
		}
		e.encodeAnyValue(key, val, m["_"+key], opts)
	}
}

// encodeAnyElementExtensions writes element extension children from a map.
func (e *xmlEncoder) encodeAnyElementExtensions(ext map[string]any) {
	if id, ok := ext["id"]; ok {
		if idStr, ok := id.(string); ok {
			e.writeValueElement("id", idStr)
		}
	}
	if exts, ok := ext["extension"]; ok {
		e.encodeAnyValue("extension", exts, nil, Options{})
	}
}

// findCompanion looks for a _fieldName companion Element field.
func findCompanion(v reflect.Value, t reflect.Type, fieldName string) reflect.Value {
	companionTag := "_" + fieldName
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("json")
		name, _ := parseTag(tag)
		if name == companionTag {
			return v.Field(i)
		}
	}
	return reflect.Value{}
}

func parseTag(tag string) (name string, omitempty bool) {
	parts := strings.Split(tag, ",")
	name = parts[0]
	for _, p := range parts[1:] {
		if p == "omitempty" {
			omitempty = true
		}
	}
	return
}

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Map:
		return v.IsNil() || v.Len() == 0
	case reflect.String:
		return v.String() == ""
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	default:
		return false
	}
}

// --- XML encoder primitives ---

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

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// --- XML decoder (XML → map → JSON → struct) ---
// The decoder uses schema metadata to correctly interpret XML elements.

// UnmarshalXML deserializes FHIR XML into a resource.
func UnmarshalXML(data []byte, resource any) error {
	m, err := xmlToMap(data)
	if err != nil {
		return fmt.Errorf("xml: parse: %w", err)
	}

	resourceType, _ := m["resourceType"].(string)
	fixArrayFields(m, resourceType)

	jsonData, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("xml: json re-convert: %w", err)
	}

	return json.Unmarshal(jsonData, resource)
}

// fixArrayFields uses schema metadata to wrap single values in arrays where
// the FHIR spec defines the field as repeating.
func fixArrayFields(m map[string]any, typeName string) {
	for k, v := range m {
		if k == "resourceType" {
			continue
		}
		// For _field element extensions, recurse into their content
		if strings.HasPrefix(k, "_") {
			if sub, ok := v.(map[string]any); ok {
				fixExtensionArrays(sub)
			}
			continue
		}
		switch val := v.(type) {
		case map[string]any:
			childType := inferChildType(typeName, k)
			// If no type found but this looks like an extension (has "url"), treat as Extension
			if childType == "" {
				if _, hasURL := val["url"]; hasURL {
					childType = "Extension"
				}
			}
			fixArrayFields(val, childType)
			if IsArrayField(typeName, k) {
				m[k] = []any{val}
			}
		case string:
			if IsArrayField(typeName, k) {
				m[k] = []any{val}
			}
		case json.Number:
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

// fixExtensionArrays handles array fixing within _field element extension maps.
// These maps have the structure: {"id": "...", "extension": [{...}]}
func fixExtensionArrays(m map[string]any) {
	// Fix the extension field itself — always an array
	if ext, ok := m["extension"]; ok {
		if sub, ok := ext.(map[string]any); ok {
			// Single extension — wrap in array
			fixArrayFields(sub, "Extension")
			m["extension"] = []any{sub}
		} else if arr, ok := ext.([]any); ok {
			for _, item := range arr {
				if sub, ok := item.(map[string]any); ok {
					fixArrayFields(sub, "Extension")
				}
			}
		}
	}
}

func inferChildType(parentType, fieldName string) string {
	if ft := FieldType(parentType, fieldName); ft != "" {
		return ft
	}
	return ""
}

// xmlToMap converts FHIR XML to a JSON-compatible map.
func xmlToMap(data []byte) (map[string]any, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty XML input")
	}
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	m, err := decodeXMLElement(decoder)
	if err != nil {
		return nil, err
	}
	result, ok := m.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected map, got %T", m)
	}
	if _, hasRT := result["resourceType"]; !hasRT {
		return nil, fmt.Errorf("no FHIR resource root element found")
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
				continue
			}

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

func coerceXMLValue(parentType, fieldName, s string) any {
	if (s == "true" || s == "false") && isBooleanContext(parentType, fieldName) {
		return s == "true"
	}
	if IsNumericField(parentType, fieldName) {
		if len(s) > 0 && (s[0] == '-' || s[0] == '.' || (s[0] >= '0' && s[0] <= '9')) {
			// Use json.Number to preserve exact decimal precision (e.g., "1.00" stays "1.00")
			return json.Number(s)
		}
	}
	return s
}

func isBooleanContext(parentType, fieldName string) bool {
	// Use schema-generated metadata for accurate boolean detection.
	// Falls back to name-based heuristic for value[x] Boolean variants.
	if IsBooleanField(parentType, fieldName) {
		return true
	}
	// value[x] Boolean variants
	if strings.HasSuffix(fieldName, "Boolean") {
		return true
	}
	return false
}

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

func readRawXMLElement(decoder *xml.Decoder, start xml.StartElement) (string, error) {
	var buf strings.Builder
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
	switch e := existing.(type) {
	case []any:
		m[key] = append(e, val)
	default:
		m[key] = []any{e, val}
	}
}
