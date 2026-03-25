// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package datatypes_test

import (
	"encoding/json"
	"testing"

	dt "github.com/caucehealth/gofhir/r4/datatypes"
)

func TestAddressBuilder(t *testing.T) {
	addr := dt.NewAddress().
		WithUse("home").
		WithLine("123 Main St").
		WithLine("Apt 4B").
		WithCity("Springfield").
		WithState("IL").
		WithPostalCode("62701").
		WithCountry("US").
		Build()

	if addr.Use == nil || *addr.Use != "home" {
		t.Error("use should be home")
	}
	if len(addr.Line) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(addr.Line))
	}
	if addr.City == nil || *addr.City != "Springfield" {
		t.Error("city should be Springfield")
	}

	// Round-trip
	data, _ := json.Marshal(addr)
	var reparsed dt.Address
	json.Unmarshal(data, &reparsed)
	if len(reparsed.Line) != 2 {
		t.Error("lines should survive round-trip")
	}
}

func TestContactPointBuilder(t *testing.T) {
	cp := dt.NewContactPoint().
		WithSystem("phone").
		WithValue("555-1234").
		WithUse("home").
		Build()

	if cp.System == nil || *cp.System != "phone" {
		t.Error("system should be phone")
	}
	if cp.Value == nil || *cp.Value != "555-1234" {
		t.Error("value should be 555-1234")
	}
}

func TestIdentifierBuilder(t *testing.T) {
	id := dt.NewIdentifier().
		WithSystem("http://example.org/mrn").
		WithValue("MRN-12345").
		WithUse("official").
		Build()

	if id.System == nil || string(*id.System) != "http://example.org/mrn" {
		t.Error("system mismatch")
	}
	if id.Value == nil || *id.Value != "MRN-12345" {
		t.Error("value mismatch")
	}
}

func TestPeriodBuilder(t *testing.T) {
	p := dt.NewPeriod().
		WithStart("2023-01-01").
		WithEnd("2023-12-31").
		Build()

	if p.Start == nil || string(*p.Start) != "2023-01-01" {
		t.Error("start mismatch")
	}
	if p.End == nil || string(*p.End) != "2023-12-31" {
		t.Error("end mismatch")
	}
}

func TestQuantityBuilder(t *testing.T) {
	q := dt.NewQuantity().
		WithValue(120.0).
		WithUnit("mmHg").
		WithSystem("http://unitsofmeasure.org").
		WithCode("mm[Hg]").
		Build()

	if q.Value == nil || q.Value.Float64() != 120.0 {
		t.Error("value should be 120")
	}
	if q.Unit == nil || *q.Unit != "mmHg" {
		t.Error("unit should be mmHg")
	}
}

func TestMetaBuilder(t *testing.T) {
	m := dt.NewMeta().
		WithVersionId("1").
		WithLastUpdated("2023-01-15T10:00:00Z").
		WithProfile("http://hl7.org/fhir/us/core/StructureDefinition/us-core-patient").
		WithTag("http://example.org/tags", "important").
		Build()

	if m.VersionId == nil || string(*m.VersionId) != "1" {
		t.Error("versionId mismatch")
	}
	if len(m.Profile) != 1 {
		t.Error("should have one profile")
	}
	if len(m.Tag) != 1 {
		t.Error("should have one tag")
	}
}

func TestAnnotationBuilder(t *testing.T) {
	a := dt.NewAnnotation().
		WithText("Patient is allergic to penicillin").
		WithTime("2023-06-15T10:30:00Z").
		Build()

	if a.Text == nil || string(*a.Text) != "Patient is allergic to penicillin" {
		t.Error("text mismatch")
	}
	if a.Time == nil || string(*a.Time) != "2023-06-15T10:30:00Z" {
		t.Error("time mismatch")
	}
}

func TestDecimalPrecisionPreservation(t *testing.T) {
	tests := []struct {
		json     string
		wantStr  string
		wantF64  float64
	}{
		{`{"value":1.0}`, "1.0", 1.0},
		{`{"value":1.00}`, "1.00", 1.0},
		{`{"value":120}`, "120", 120.0},
		{`{"value":0.123456789}`, "0.123456789", 0.123456789},
		{`{"value":100.0}`, "100.0", 100.0},
	}

	for _, tt := range tests {
		var q dt.Quantity
		if err := json.Unmarshal([]byte(tt.json), &q); err != nil {
			t.Fatalf("unmarshal %s: %v", tt.json, err)
		}
		if q.Value == nil {
			t.Fatalf("value nil for %s", tt.json)
		}
		if q.Value.String() != tt.wantStr {
			t.Errorf("precision lost: got %q, want %q", q.Value.String(), tt.wantStr)
		}
		if q.Value.Float64() != tt.wantF64 {
			t.Errorf("float64: got %v, want %v", q.Value.Float64(), tt.wantF64)
		}

		// Round-trip: marshal and verify precision preserved
		out, err := json.Marshal(&q)
		if err != nil {
			t.Fatal(err)
		}
		var q2 dt.Quantity
		json.Unmarshal(out, &q2)
		if q2.Value.String() != tt.wantStr {
			t.Errorf("round-trip precision lost: got %q, want %q", q2.Value.String(), tt.wantStr)
		}
	}
}

func TestDecimalMarshalAsNumber(t *testing.T) {
	// Decimal should marshal as a bare JSON number, not a quoted string
	d := dt.NewDecimal(3.14)
	out, err := json.Marshal(&d)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "3.14" {
		t.Errorf("marshal = %s, want 3.14 (bare number)", out)
	}
}

func TestDecimalEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantStr string
	}{
		{"scientific", `{"value":1.23e-4}`, "1.23e-4"},
		{"negative", `{"value":-120.5}`, "-120.5"},
		{"large_int", `{"value":99999999999999}`, "99999999999999"},
		{"small", `{"value":0.000001}`, "0.000001"},
		{"negative_zero", `{"value":-0}`, "-0"},
		{"zero", `{"value":0}`, "0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var q dt.Quantity
			if err := json.Unmarshal([]byte(tt.json), &q); err != nil {
				t.Fatal(err)
			}
			if q.Value == nil {
				t.Fatal("value nil")
			}
			if q.Value.String() != tt.wantStr {
				t.Errorf("got %q, want %q", q.Value.String(), tt.wantStr)
			}
			// Round-trip
			out, _ := json.Marshal(&q)
			var q2 dt.Quantity
			json.Unmarshal(out, &q2)
			if q2.Value.String() != tt.wantStr {
				t.Errorf("round-trip: got %q, want %q", q2.Value.String(), tt.wantStr)
			}
		})
	}
}

func TestDecimalInvalidValues(t *testing.T) {
	var d dt.Decimal
	if err := d.UnmarshalJSON([]byte(`"not_a_number"`)); err == nil {
		t.Error("should reject non-numeric string")
	}
	if err := d.UnmarshalJSON([]byte(`"NaN"`)); err == nil {
		t.Error("should reject NaN")
	}
	if err := d.UnmarshalJSON([]byte(`"Infinity"`)); err == nil {
		t.Error("should reject Infinity")
	}
}

func TestDecimalMarshalInvalid(t *testing.T) {
	d := dt.Decimal("not_a_number")
	_, err := d.MarshalJSON()
	if err == nil {
		t.Error("should reject invalid decimal on marshal")
	}
}
