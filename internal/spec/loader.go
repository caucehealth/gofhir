// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Schema represents the top-level FHIR JSON schema.
type Schema struct {
	Discriminator struct {
		Mapping map[string]string `json:"mapping"`
	} `json:"discriminator"`
	Definitions map[string]Definition `json:"definitions"`
}

// Definition represents a single type definition in the schema.
type Definition struct {
	Description string                `json:"description"`
	Properties  map[string]Property   `json:"properties"`
	Required    []string              `json:"required"`
	Type        string                `json:"type"`
	Enum        []string              `json:"enum"`
	Const       interface{}           `json:"const"`
	Ref         string                `json:"$ref"`
	Items       *Property             `json:"items"`
	OneOf       []map[string]string   `json:"oneOf"`
}

// Property represents a property within a definition.
type Property struct {
	Description string              `json:"description"`
	Ref         string              `json:"$ref"`
	Type        string              `json:"type"`
	Enum        []string            `json:"enum"`
	Pattern     string              `json:"pattern"`
	Const       interface{}         `json:"const"`
	Items       *Property           `json:"items"`
	OneOf       []map[string]string `json:"oneOf"`
}

// FHIRSpec holds the parsed and categorized FHIR specification.
type FHIRSpec struct {
	Resources       map[string]*ResourceDef
	ComplexTypes    map[string]*ComplexTypeDef
	BackboneElements map[string]*ComplexTypeDef
	Primitives      []string
	ResourceNames   []string // sorted
}

// ResourceDef describes a FHIR resource.
type ResourceDef struct {
	Name        string
	Description string
	Fields      []*FieldDef
	Required    map[string]bool
}

// ComplexTypeDef describes a FHIR complex type or backbone element.
type ComplexTypeDef struct {
	Name        string
	Description string
	Fields      []*FieldDef
	Required    map[string]bool
}

// FieldDef describes a single field.
type FieldDef struct {
	Name        string
	JSONName    string
	Description string
	FHIRType    string // e.g. "string", "Reference", "CodeableConcept"
	IsArray     bool
	IsRequired  bool
	Enum        []string // non-nil if this is an enum field
	IsRef       bool     // true if $ref
	RefTarget   string   // e.g. "Quantity"
}

// Load reads and parses a FHIR schema JSON file.
func Load(path string) (*FHIRSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading schema: %w", err)
	}

	var schema Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("parsing schema: %w", err)
	}

	resourceNames := make(map[string]bool)
	for name := range schema.Discriminator.Mapping {
		resourceNames[name] = true
	}

	spec := &FHIRSpec{
		Resources:        make(map[string]*ResourceDef),
		ComplexTypes:     make(map[string]*ComplexTypeDef),
		BackboneElements: make(map[string]*ComplexTypeDef),
	}

	primitiveNames := map[string]bool{
		"base64Binary": true, "boolean": true, "canonical": true, "code": true,
		"date": true, "dateTime": true, "decimal": true, "id": true,
		"instant": true, "integer": true, "markdown": true, "oid": true,
		"positiveInt": true, "string": true, "time": true, "unsignedInt": true,
		"uri": true, "url": true, "uuid": true, "xhtml": true,
	}

	// Abstract types to skip
	abstractTypes := map[string]bool{
		"ResourceList":    true,
		"Resource":        true,
		"DomainResource":  true,
		"MetadataResource": true,
		"Element":         true,
		"BackboneElement": true,
	}

	for name, def := range schema.Definitions {
		if abstractTypes[name] || primitiveNames[name] {
			if primitiveNames[name] {
				spec.Primitives = append(spec.Primitives, name)
			}
			continue
		}

		if name == "ResourceList" {
			continue
		}

		fields := parseFields(def, schema)
		required := make(map[string]bool)
		for _, r := range def.Required {
			required[r] = true
		}

		if resourceNames[name] {
			spec.Resources[name] = &ResourceDef{
				Name:        name,
				Description: def.Description,
				Fields:      fields,
				Required:    required,
			}
		} else if strings.Contains(name, "_") {
			parts := strings.SplitN(name, "_", 2)
			if resourceNames[parts[0]] || isKnownComplexBase(parts[0]) {
				spec.BackboneElements[name] = &ComplexTypeDef{
					Name:        name,
					Description: def.Description,
					Fields:      fields,
					Required:    required,
				}
			}
		} else {
			spec.ComplexTypes[name] = &ComplexTypeDef{
				Name:        name,
				Description: def.Description,
				Fields:      fields,
				Required:    required,
			}
		}
	}

	// Build sorted resource names
	for name := range spec.Resources {
		spec.ResourceNames = append(spec.ResourceNames, name)
	}
	sort.Strings(spec.ResourceNames)
	sort.Strings(spec.Primitives)

	return spec, nil
}

func isKnownComplexBase(name string) bool {
	complexBases := map[string]bool{
		"DataRequirement": true, "Dosage": true, "ElementDefinition": true,
		"Timing": true, "SubstanceAmount": true, "MarketingStatus": true,
		"ProdCharacteristic": true, "ProductShelfLife": true, "Population": true,
	}
	return complexBases[name]
}

func parseFields(def Definition, schema Schema) []*FieldDef {
	var fields []*FieldDef

	for propName, prop := range def.Properties {
		// Skip underscore-prefixed extension properties
		if strings.HasPrefix(propName, "_") {
			continue
		}
		// Skip resourceType - handled separately
		if propName == "resourceType" {
			continue
		}

		field := &FieldDef{
			JSONName:    propName,
			Name:        propName,
			Description: prop.Description,
		}

		if len(prop.Enum) > 0 {
			field.Enum = prop.Enum
			field.FHIRType = "code"
		} else if prop.Ref != "" {
			ref := strings.TrimPrefix(prop.Ref, "#/definitions/")
			field.IsRef = true
			field.RefTarget = ref
			field.FHIRType = ref
		} else if prop.Type == "array" && prop.Items != nil {
			field.IsArray = true
			if prop.Items.Ref != "" {
				ref := strings.TrimPrefix(prop.Items.Ref, "#/definitions/")
				field.IsRef = true
				field.RefTarget = ref
				field.FHIRType = ref
			} else if len(prop.Items.Enum) > 0 {
				field.Enum = prop.Items.Enum
				field.FHIRType = "code"
			} else {
				field.FHIRType = prop.Items.Type
			}
		} else {
			field.FHIRType = prop.Type
			if field.FHIRType == "" {
				field.FHIRType = "string"
			}
		}

		fields = append(fields, field)
	}

	// Mark required fields
	for _, r := range def.Required {
		for _, f := range fields {
			if f.JSONName == r {
				f.IsRequired = true
			}
		}
	}

	// Sort fields for deterministic output
	sort.Slice(fields, func(i, j int) bool {
		return fieldOrder(fields[i].JSONName) < fieldOrder(fields[j].JSONName)
	})

	return fields
}

// fieldOrder returns a sort key that puts common FHIR fields first.
func fieldOrder(name string) string {
	priority := map[string]string{
		"id":                "00",
		"meta":              "01",
		"implicitRules":     "02",
		"language":          "03",
		"text":              "04",
		"contained":         "05",
		"extension":         "06",
		"modifierExtension": "07",
		"identifier":        "08",
		"status":            "09",
	}
	if p, ok := priority[name]; ok {
		return p + name
	}
	return "99" + name
}
