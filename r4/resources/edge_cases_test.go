// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/caucehealth/gofhir/r4/resources"
)

// ============================================================================
// Observation.component with all value[x] variants
// ============================================================================

func TestObservationComponentValueQuantity(t *testing.T) {
	input := `{"resourceType":"Observation","status":"final","code":{"text":"BP"},
		"component":[{"code":{"text":"systolic"},"valueQuantity":{"value":120,"unit":"mmHg"}},
		             {"code":{"text":"diastolic"},"valueQuantity":{"value":80,"unit":"mmHg"}}]}`
	var obs resources.Observation
	json.Unmarshal([]byte(input), &obs)
	if len(obs.Component) != 2 {
		t.Fatalf("expected 2 components, got %d", len(obs.Component))
	}
	if obs.Component[0].Value == nil || obs.Component[0].Value.Quantity == nil {
		t.Fatal("component[0].valueQuantity should be set")
	}
	if obs.Component[0].Value.Quantity.Value.Float64() != 120 {
		t.Error("systolic should be 120")
	}
	// Round-trip
	out, _ := json.Marshal(&obs)
	if !strings.Contains(string(out), `"valueQuantity"`) {
		t.Error("valueQuantity should survive round-trip in component")
	}
}

func TestObservationComponentValueString(t *testing.T) {
	input := `{"resourceType":"Observation","status":"final","code":{"text":"Note"},
		"component":[{"code":{"text":"finding"},"valueString":"Normal"}]}`
	var obs resources.Observation
	json.Unmarshal([]byte(input), &obs)
	if obs.Component[0].Value == nil || obs.Component[0].Value.String == nil {
		t.Fatal("component[0].valueString should be set")
	}
	if *obs.Component[0].Value.String != "Normal" {
		t.Error("valueString should be Normal")
	}
}

func TestObservationComponentValueCodeableConcept(t *testing.T) {
	input := `{"resourceType":"Observation","status":"final","code":{"text":"test"},
		"component":[{"code":{"text":"interp"},"valueCodeableConcept":{"text":"High"}}]}`
	var obs resources.Observation
	json.Unmarshal([]byte(input), &obs)
	if obs.Component[0].Value == nil || obs.Component[0].Value.CodeableConcept == nil {
		t.Fatal("component[0].valueCodeableConcept should be set")
	}
	if *obs.Component[0].Value.CodeableConcept.Text != "High" {
		t.Error("text should be High")
	}
}

func TestObservationComponentValueBoolean(t *testing.T) {
	input := `{"resourceType":"Observation","status":"final","code":{"text":"test"},
		"component":[{"code":{"text":"detected"},"valueBoolean":true}]}`
	var obs resources.Observation
	json.Unmarshal([]byte(input), &obs)
	if obs.Component[0].Value == nil || obs.Component[0].Value.Boolean == nil {
		t.Fatal("component[0].valueBoolean should be set")
	}
	if *obs.Component[0].Value.Boolean != true {
		t.Error("valueBoolean should be true")
	}
}

// ============================================================================
// Extension-on-Extension nesting
// ============================================================================

func TestExtensionOnExtension(t *testing.T) {
	input := `{"resourceType":"Patient","id":"1",
		"extension":[{
			"url":"http://hl7.org/fhir/us/core/StructureDefinition/us-core-race",
			"extension":[
				{"url":"ombCategory","valueCoding":{"system":"urn:oid:2.16.840.1.113883.6.238","code":"2106-3","display":"White"}},
				{"url":"text","valueString":"White"}
			]
		}]}`

	var p resources.Patient
	if err := json.Unmarshal([]byte(input), &p); err != nil {
		t.Fatal(err)
	}

	if len(p.Extension) != 1 {
		t.Fatalf("expected 1 top extension, got %d", len(p.Extension))
	}
	if len(p.Extension[0].Extension) != 2 {
		t.Fatalf("expected 2 nested extensions, got %d", len(p.Extension[0].Extension))
	}
	if p.Extension[0].Extension[1].ValueString == nil || *p.Extension[0].Extension[1].ValueString != "White" {
		t.Error("nested extension valueString should be White")
	}

	// Round-trip
	out, _ := json.Marshal(&p)
	s := string(out)
	if !strings.Contains(s, "us-core-race") {
		t.Error("top extension URL should survive")
	}
	if !strings.Contains(s, "ombCategory") {
		t.Error("nested extension URL should survive")
	}
	if !strings.Contains(s, "White") {
		t.Error("nested text extension should survive")
	}
}

func TestDeeplyNestedExtension(t *testing.T) {
	// 3 levels deep: extension → extension → extension
	input := `{"resourceType":"Patient","id":"1",
		"extension":[{
			"url":"http://level1",
			"extension":[{
				"url":"http://level2",
				"extension":[{
					"url":"http://level3",
					"valueString":"deep"
				}]
			}]
		}]}`

	var p resources.Patient
	json.Unmarshal([]byte(input), &p)

	ext3 := p.Extension[0].Extension[0].Extension[0]
	if *ext3.ValueString != "deep" {
		t.Error("3-level deep extension should be accessible")
	}

	out, _ := json.Marshal(&p)
	if !strings.Contains(string(out), "level3") {
		t.Error("deep extension should survive round-trip")
	}
}

// ============================================================================
// Contained resource nesting
// ============================================================================

func TestContainedWithinContained(t *testing.T) {
	// Patient contains Practitioner which itself has contained resources
	input := `{"resourceType":"Patient","id":"p1",
		"contained":[{
			"resourceType":"Practitioner","id":"prac1",
			"contained":[{
				"resourceType":"Organization","id":"org1","name":"Test Org"
			}],
			"name":[{"family":"Smith"}]
		}],
		"generalPractitioner":[{"reference":"#prac1"}]}`

	var p resources.Patient
	if err := json.Unmarshal([]byte(input), &p); err != nil {
		t.Fatal(err)
	}

	if len(p.Contained) != 1 {
		t.Fatal("should have 1 contained")
	}

	// Parse the contained Practitioner
	prac, err := resources.ParseContainedPractitioner(p.Contained[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(prac.Contained) != 1 {
		t.Fatal("practitioner should have 1 nested contained")
	}

	// Round-trip
	out, _ := json.Marshal(&p)
	if !strings.Contains(string(out), "Test Org") {
		t.Error("nested contained org name should survive round-trip")
	}
}

// ============================================================================
// All value[x] type variants on Observation
// ============================================================================

func TestAllValueXVariantsRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		json  string
		check func(*testing.T, resources.Observation)
	}{
		{"valueQuantity", `{"resourceType":"Observation","status":"final","code":{"text":"t"},"valueQuantity":{"value":120,"unit":"mg"}}`,
			func(t *testing.T, o resources.Observation) {
				if o.Value == nil || o.Value.Quantity == nil {
					t.Fatal("expected Quantity")
				}
			}},
		{"valueString", `{"resourceType":"Observation","status":"final","code":{"text":"t"},"valueString":"test"}`,
			func(t *testing.T, o resources.Observation) {
				if o.Value == nil || o.Value.String == nil || *o.Value.String != "test" {
					t.Fatal("expected String")
				}
			}},
		{"valueBoolean", `{"resourceType":"Observation","status":"final","code":{"text":"t"},"valueBoolean":true}`,
			func(t *testing.T, o resources.Observation) {
				if o.Value == nil || o.Value.Boolean == nil || *o.Value.Boolean != true {
					t.Fatal("expected Boolean")
				}
			}},
		{"valueInteger", `{"resourceType":"Observation","status":"final","code":{"text":"t"},"valueInteger":42}`,
			func(t *testing.T, o resources.Observation) {
				if o.Value == nil || o.Value.Integer == nil || *o.Value.Integer != 42 {
					t.Fatal("expected Integer 42")
				}
			}},
		{"valueRange", `{"resourceType":"Observation","status":"final","code":{"text":"t"},"valueRange":{"low":{"value":10},"high":{"value":20}}}`,
			func(t *testing.T, o resources.Observation) {
				if o.Value == nil || o.Value.Range == nil {
					t.Fatal("expected Range")
				}
			}},
		{"valueRatio", `{"resourceType":"Observation","status":"final","code":{"text":"t"},"valueRatio":{"numerator":{"value":1},"denominator":{"value":128}}}`,
			func(t *testing.T, o resources.Observation) {
				if o.Value == nil || o.Value.Ratio == nil {
					t.Fatal("expected Ratio")
				}
			}},
		{"valuePeriod", `{"resourceType":"Observation","status":"final","code":{"text":"t"},"valuePeriod":{"start":"2024-01-01","end":"2024-12-31"}}`,
			func(t *testing.T, o resources.Observation) {
				if o.Value == nil || o.Value.Period == nil {
					t.Fatal("expected Period")
				}
			}},
		{"valueCodeableConcept", `{"resourceType":"Observation","status":"final","code":{"text":"t"},"valueCodeableConcept":{"text":"positive"}}`,
			func(t *testing.T, o resources.Observation) {
				if o.Value == nil || o.Value.CodeableConcept == nil {
					t.Fatal("expected CodeableConcept")
				}
			}},
		{"valueDateTime", `{"resourceType":"Observation","status":"final","code":{"text":"t"},"valueDateTime":"2024-03-15"}`,
			func(t *testing.T, o resources.Observation) {
				if o.Value == nil || o.Value.DateTime == nil {
					t.Fatal("expected DateTime")
				}
			}},
		{"valueTime", `{"resourceType":"Observation","status":"final","code":{"text":"t"},"valueTime":"14:30:00"}`,
			func(t *testing.T, o resources.Observation) {
				if o.Value == nil || o.Value.Time == nil {
					t.Fatal("expected Time")
				}
			}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var obs resources.Observation
			if err := json.Unmarshal([]byte(tt.json), &obs); err != nil {
				t.Fatal(err)
			}
			tt.check(t, obs)

			// Round-trip
			out, _ := json.Marshal(&obs)
			var obs2 resources.Observation
			json.Unmarshal(out, &obs2)
			tt.check(t, obs2)
		})
	}
}

// ============================================================================
// Unicode and special characters
// ============================================================================

func TestUnicodeInNames(t *testing.T) {
	tests := []struct {
		name   string
		family string
	}{
		{"CJK", "\u5f20\u4e09"},          // 张三 (Chinese)
		{"Arabic", "\u0645\u062d\u0645\u062f"}, // محمد
		{"Emoji", "Dr. \U0001F600"},
		{"Accented", "Müller-Böhm"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := `{"resourceType":"Patient","name":[{"family":"` + tt.family + `"}]}`
			var p resources.Patient
			json.Unmarshal([]byte(input), &p)

			if p.Name[0].Family == nil || *p.Name[0].Family != tt.family {
				t.Errorf("family = %q, want %q", *p.Name[0].Family, tt.family)
			}

			out, _ := json.Marshal(&p)
			var p2 resources.Patient
			json.Unmarshal(out, &p2)
			if *p2.Name[0].Family != tt.family {
				t.Errorf("round-trip family = %q, want %q", *p2.Name[0].Family, tt.family)
			}
		})
	}
}

// ============================================================================
// Race condition safety
// ============================================================================

func TestConcurrentParsing(t *testing.T) {
	input := `{"resourceType":"Patient","id":"1","gender":"male","name":[{"family":"Doe"}]}`
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < 100; j++ {
				res, err := resources.ParseResource(json.RawMessage(input))
				if err != nil {
					t.Errorf("concurrent parse error: %v", err)
					return
				}
				if res.GetResourceType() != "Patient" {
					t.Error("wrong resource type in concurrent parse")
					return
				}
				json.Marshal(res)
			}
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
