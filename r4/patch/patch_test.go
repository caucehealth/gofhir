// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package patch_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/caucehealth/gofhir/r4/patch"
)

func TestJSONPatchAdd(t *testing.T) {
	p := patch.NewJSONPatch().
		Add("/name/0/family", "Smith")

	data, err := p.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, `"op":"add"`) {
		t.Error("should contain add op")
	}
	if !strings.Contains(s, `"path":"/name/0/family"`) {
		t.Error("should contain path")
	}
	if !strings.Contains(s, `"Smith"`) {
		t.Error("should contain value")
	}
}

func TestJSONPatchReplace(t *testing.T) {
	p := patch.NewJSONPatch().
		Replace("/gender", "female")

	ops := p.Operations()
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
	if ops[0].Op != patch.OpReplace {
		t.Errorf("op = %q, want replace", ops[0].Op)
	}
}

func TestJSONPatchRemove(t *testing.T) {
	p := patch.NewJSONPatch().Remove("/telecom/0")
	data, err := p.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"op":"remove"`) {
		t.Error("should contain remove op")
	}
}

func TestJSONPatchMove(t *testing.T) {
	p := patch.NewJSONPatch().Move("/name/0", "/name/1")
	ops := p.Operations()
	if ops[0].From != "/name/0" {
		t.Errorf("from = %q, want /name/0", ops[0].From)
	}
}

func TestJSONPatchCopy(t *testing.T) {
	p := patch.NewJSONPatch().Copy("/name/0", "/name/1")
	ops := p.Operations()
	if ops[0].Op != patch.OpCopy {
		t.Errorf("op = %q, want copy", ops[0].Op)
	}
}

func TestJSONPatchTest(t *testing.T) {
	p := patch.NewJSONPatch().Test("/gender", "male")
	ops := p.Operations()
	if ops[0].Op != patch.OpTest {
		t.Errorf("op = %q, want test", ops[0].Op)
	}
}

func TestJSONPatchChaining(t *testing.T) {
	p := patch.NewJSONPatch().
		Test("/gender", "male").
		Replace("/gender", "female").
		Add("/telecom/-", map[string]string{"system": "phone", "value": "555-1234"}).
		Remove("/address/0")

	if len(p.Operations()) != 4 {
		t.Errorf("expected 4 ops, got %d", len(p.Operations()))
	}

	data, err := p.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	var ops []map[string]any
	json.Unmarshal(data, &ops)
	if len(ops) != 4 {
		t.Errorf("JSON should have 4 ops, got %d", len(ops))
	}
}

func TestJSONPatchMustMarshal(t *testing.T) {
	p := patch.NewJSONPatch().Add("/active", true)
	data := p.MustMarshal()
	if len(data) == 0 {
		t.Error("should produce output")
	}
}

func TestFHIRPatchAdd(t *testing.T) {
	p := patch.NewFHIRPatch().
		Add("Patient", "birthDate", "1990-01-01")

	data, err := p.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, `"resourceType":"Parameters"`) {
		t.Error("should be a Parameters resource")
	}
	if !strings.Contains(s, `"valueCode":"add"`) {
		t.Error("should contain add type")
	}
	if !strings.Contains(s, `"1990-01-01"`) {
		t.Error("should contain value")
	}
}

func TestFHIRPatchReplace(t *testing.T) {
	p := patch.NewFHIRPatch().
		Replace("Patient.gender", "female")

	data, err := p.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"valueCode":"replace"`) {
		t.Error("should contain replace type")
	}
}

func TestFHIRPatchDelete(t *testing.T) {
	p := patch.NewFHIRPatch().Delete("Patient.telecom")
	data, err := p.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"valueCode":"delete"`) {
		t.Error("should contain delete type")
	}
}

func TestFHIRPatchMove(t *testing.T) {
	p := patch.NewFHIRPatch().Move("Patient.name", 0, 1)
	data, err := p.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, `"valueCode":"move"`) {
		t.Error("should contain move type")
	}
	if !strings.Contains(s, `"source"`) {
		t.Error("should contain source")
	}
}

func TestFHIRPatchInsert(t *testing.T) {
	p := patch.NewFHIRPatch().Insert("Patient.name", 0, "test")
	data, err := p.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, `"valueCode":"insert"`) {
		t.Error("should contain insert type")
	}
	if !strings.Contains(s, `"index"`) {
		t.Error("should contain index")
	}
}

func TestFHIRPatchValueTypes(t *testing.T) {
	p := patch.NewFHIRPatch().
		Add("Patient", "active", true).
		Add("Patient", "multipleBirthInteger", 2).
		Add("Patient", "birthDate", "1990-01-01")

	data, err := p.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "valueBoolean") {
		t.Error("should use valueBoolean for bool")
	}
	if !strings.Contains(s, "valueInteger") {
		t.Error("should use valueInteger for int")
	}
	if !strings.Contains(s, "valueString") {
		t.Error("should use valueString for string")
	}
}
