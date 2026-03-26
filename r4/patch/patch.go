// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

// Package patch provides JSON Patch (RFC 6902) and FHIR Patch builders
// for partial resource updates.
package patch

import (
	"encoding/json"
	"fmt"
)

// ContentTypeJSONPatch is the MIME type for JSON Patch (RFC 6902).
const ContentTypeJSONPatch = "application/json-patch+json"

// ContentTypeFHIRPatch is the MIME type for FHIR Patch (Parameters-based).
const ContentTypeFHIRPatch = "application/fhir+json"

// Op represents a JSON Patch operation type.
type Op string

const (
	OpAdd     Op = "add"
	OpRemove  Op = "remove"
	OpReplace Op = "replace"
	OpMove    Op = "move"
	OpCopy    Op = "copy"
	OpTest    Op = "test"
)

// Operation is a single JSON Patch (RFC 6902) operation.
type Operation struct {
	Op    Op              `json:"op"`
	Path  string          `json:"path"`
	From  string          `json:"from,omitempty"`
	Value json.RawMessage `json:"value,omitempty"`
}

// JSONPatch builds a list of RFC 6902 JSON Patch operations.
type JSONPatch struct {
	ops []Operation
}

// NewJSONPatch creates a new JSON Patch builder.
func NewJSONPatch() *JSONPatch {
	return &JSONPatch{}
}

// Add appends an "add" operation.
func (p *JSONPatch) Add(path string, value any) *JSONPatch {
	p.ops = append(p.ops, Operation{Op: OpAdd, Path: path, Value: marshal(value)})
	return p
}

// Remove appends a "remove" operation.
func (p *JSONPatch) Remove(path string) *JSONPatch {
	p.ops = append(p.ops, Operation{Op: OpRemove, Path: path})
	return p
}

// Replace appends a "replace" operation.
func (p *JSONPatch) Replace(path string, value any) *JSONPatch {
	p.ops = append(p.ops, Operation{Op: OpReplace, Path: path, Value: marshal(value)})
	return p
}

// Move appends a "move" operation.
func (p *JSONPatch) Move(from, path string) *JSONPatch {
	p.ops = append(p.ops, Operation{Op: OpMove, Path: path, From: from})
	return p
}

// Copy appends a "copy" operation.
func (p *JSONPatch) Copy(from, path string) *JSONPatch {
	p.ops = append(p.ops, Operation{Op: OpCopy, Path: path, From: from})
	return p
}

// Test appends a "test" operation.
func (p *JSONPatch) Test(path string, value any) *JSONPatch {
	p.ops = append(p.ops, Operation{Op: OpTest, Path: path, Value: marshal(value)})
	return p
}

// Operations returns the list of operations.
func (p *JSONPatch) Operations() []Operation {
	return p.ops
}

// Marshal serializes the patch to JSON.
func (p *JSONPatch) Marshal() ([]byte, error) {
	return json.Marshal(p.ops)
}

// MustMarshal serializes the patch to JSON, panicking on error.
func (p *JSONPatch) MustMarshal() []byte {
	data, err := p.Marshal()
	if err != nil {
		panic(fmt.Sprintf("patch marshal: %v", err))
	}
	return data
}

func marshal(v any) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}

// --- FHIR Patch (Parameters-based) ---

// FHIRPatchType represents a FHIR Patch operation type.
type FHIRPatchType string

const (
	FHIRAdd     FHIRPatchType = "add"
	FHIRInsert  FHIRPatchType = "insert"
	FHIRDelete  FHIRPatchType = "delete"
	FHIRReplace FHIRPatchType = "replace"
	FHIRMove    FHIRPatchType = "move"
)

// FHIRPatch builds a FHIR Patch (Parameters resource) for partial updates.
type FHIRPatch struct {
	operations []fhirPatchOp
}

type fhirPatchOp struct {
	Type    FHIRPatchType
	Path    string
	Name    string // for add/insert
	Value   any    // for add/insert/replace
	Source  int    // for move: source index
	Dest    int    // for move: destination index
	Index   int    // for insert: index position
}

// NewFHIRPatch creates a new FHIR Patch builder.
func NewFHIRPatch() *FHIRPatch {
	return &FHIRPatch{}
}

// Add appends an "add" operation to set a value at a path.
func (p *FHIRPatch) Add(path, name string, value any) *FHIRPatch {
	p.operations = append(p.operations, fhirPatchOp{
		Type: FHIRAdd, Path: path, Name: name, Value: value,
	})
	return p
}

// Insert appends an "insert" operation at a specific index.
func (p *FHIRPatch) Insert(path string, index int, value any) *FHIRPatch {
	p.operations = append(p.operations, fhirPatchOp{
		Type: FHIRInsert, Path: path, Index: index, Value: value,
	})
	return p
}

// Delete appends a "delete" operation.
func (p *FHIRPatch) Delete(path string) *FHIRPatch {
	p.operations = append(p.operations, fhirPatchOp{Type: FHIRDelete, Path: path})
	return p
}

// Replace appends a "replace" operation.
func (p *FHIRPatch) Replace(path string, value any) *FHIRPatch {
	p.operations = append(p.operations, fhirPatchOp{
		Type: FHIRReplace, Path: path, Value: value,
	})
	return p
}

// Move appends a "move" operation.
func (p *FHIRPatch) Move(path string, source, destination int) *FHIRPatch {
	p.operations = append(p.operations, fhirPatchOp{
		Type: FHIRMove, Path: path, Source: source, Dest: destination,
	})
	return p
}

// Marshal serializes the FHIR Patch as a Parameters resource JSON.
func (p *FHIRPatch) Marshal() ([]byte, error) {
	params := map[string]any{
		"resourceType": "Parameters",
		"parameter":    p.toParameters(),
	}
	return json.Marshal(params)
}

func (p *FHIRPatch) toParameters() []map[string]any {
	var params []map[string]any
	for _, op := range p.operations {
		param := map[string]any{
			"name": "operation",
			"part": buildParts(op),
		}
		params = append(params, param)
	}
	return params
}

func buildParts(op fhirPatchOp) []map[string]any {
	parts := []map[string]any{
		{"name": "type", "valueCode": string(op.Type)},
		{"name": "path", "valueString": op.Path},
	}

	if op.Name != "" {
		parts = append(parts, map[string]any{"name": "name", "valueString": op.Name})
	}

	if op.Value != nil {
		parts = append(parts, valuePart(op.Value))
	}

	if op.Type == FHIRInsert {
		parts = append(parts, map[string]any{"name": "index", "valueInteger": op.Index})
	}

	if op.Type == FHIRMove {
		parts = append(parts, map[string]any{"name": "source", "valueInteger": op.Source})
		parts = append(parts, map[string]any{"name": "destination", "valueInteger": op.Dest})
	}

	return parts
}

func valuePart(v any) map[string]any {
	switch val := v.(type) {
	case string:
		return map[string]any{"name": "value", "valueString": val}
	case int, int32, int64:
		return map[string]any{"name": "value", "valueInteger": val}
	case float64:
		return map[string]any{"name": "value", "valueDecimal": val}
	case bool:
		return map[string]any{"name": "value", "valueBoolean": val}
	default:
		// For complex types, marshal to JSON
		data, _ := json.Marshal(val)
		return map[string]any{"name": "value", "valueString": string(data)}
	}
}
