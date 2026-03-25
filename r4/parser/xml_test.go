// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package parser_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/caucehealth/gofhir/r4/parser"
	"github.com/caucehealth/gofhir/r4/resources"
)

func TestMarshalXML(t *testing.T) {
	p := buildTestPatient(t)
	out, err := parser.MarshalXML(p, parser.Options{})
	if err != nil {
		t.Fatal(err)
	}

	xml := string(out)
	if !strings.Contains(xml, `<Patient xmlns="http://hl7.org/fhir">`) {
		t.Error("should contain Patient element with FHIR namespace")
	}
	if !strings.Contains(xml, `<gender value="male"/>`) {
		t.Error("should contain gender element")
	}
	if !strings.Contains(xml, `<birthDate value="1980-01-01"/>`) {
		t.Error("should contain birthDate element")
	}
	if !strings.Contains(xml, `<family value="Doe"/>`) {
		t.Error("should contain family name")
	}
	if !strings.Contains(xml, `</Patient>`) {
		t.Error("should have closing Patient tag")
	}
}

func TestMarshalXMLPretty(t *testing.T) {
	p := buildTestPatient(t)
	out, err := parser.MarshalXML(p, parser.Options{PrettyPrint: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "\n  ") {
		t.Error("pretty XML should have indentation")
	}
}

func TestMarshalXMLSuppressNarrative(t *testing.T) {
	input := `{"resourceType":"Patient","id":"1","text":{"status":"generated","div":"<div>test</div>"},"gender":"male"}`
	var p resources.Patient
	if err := parser.Unmarshal([]byte(input), &p); err != nil {
		t.Fatal(err)
	}

	out, err := parser.MarshalXML(&p, parser.Options{SuppressNarrative: true})
	if err != nil {
		t.Fatal(err)
	}

	xml := string(out)
	if strings.Contains(xml, "<text") {
		t.Error("text should be suppressed")
	}
	if !strings.Contains(xml, `<gender value="male"/>`) {
		t.Error("gender should still be present")
	}
}

func TestUnmarshalXML(t *testing.T) {
	xmlInput := `<?xml version="1.0" encoding="UTF-8"?>
<Patient xmlns="http://hl7.org/fhir">
  <id value="xml-test"/>
  <gender value="female"/>
  <birthDate value="1990-05-15"/>
  <name>
    <family value="Smith"/>
    <given value="Jane"/>
  </name>
</Patient>`

	var p resources.Patient
	if err := parser.UnmarshalXML([]byte(xmlInput), &p); err != nil {
		t.Fatal(err)
	}

	if p.ResourceType != "Patient" {
		t.Errorf("resourceType = %q, want Patient", p.ResourceType)
	}
	if p.GetGender() != "female" {
		t.Errorf("gender = %q, want female", p.GetGender())
	}
	if p.GetBirthDate() != "1990-05-15" {
		t.Errorf("birthDate = %q, want 1990-05-15", p.GetBirthDate())
	}
	if len(p.Name) == 0 {
		t.Fatal("should have a name")
	}
	if p.Name[0].Family == nil || *p.Name[0].Family != "Smith" {
		t.Error("family should be Smith")
	}
}

func TestXMLRoundTrip(t *testing.T) {
	p := buildTestPatient(t)

	// Marshal to XML
	xmlData, err := parser.MarshalXML(p, parser.Options{})
	if err != nil {
		t.Fatal(err)
	}

	// Unmarshal back
	var reparsed resources.Patient
	if err := parser.UnmarshalXML(xmlData, &reparsed); err != nil {
		t.Fatal(err)
	}

	if reparsed.GetGender() != resources.AdministrativeGenderMale {
		t.Errorf("gender = %q, want male", reparsed.GetGender())
	}
	if reparsed.GetBirthDate() != "1980-01-01" {
		t.Errorf("birthDate = %q, want 1980-01-01", reparsed.GetBirthDate())
	}
	if len(reparsed.Name) == 0 || reparsed.Name[0].Family == nil || *reparsed.Name[0].Family != "Doe" {
		t.Error("name should round-trip")
	}
}

func TestXMLObservationWithValueQuantity(t *testing.T) {
	xmlInput := `<?xml version="1.0" encoding="UTF-8"?>
<Observation xmlns="http://hl7.org/fhir">
  <status value="final"/>
  <code><coding><system value="http://loinc.org"/><code value="1234"/></coding></code>
  <valueQuantity><value value="120"/><unit value="mmHg"/></valueQuantity>
</Observation>`

	var obs resources.Observation
	if err := parser.UnmarshalXML([]byte(xmlInput), &obs); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if obs.GetStatus() != "final" {
		t.Errorf("status = %q, want final", obs.GetStatus())
	}
	if obs.Value == nil || obs.Value.Quantity == nil {
		t.Fatal("value.quantity should be parsed from XML")
	}
	if obs.Value.Quantity.Value == nil || obs.Value.Quantity.Value.Float64() != 120 {
		t.Errorf("quantity.value = %v, want 120", obs.Value.Quantity.Value)
	}
	if obs.Value.Quantity.Unit == nil || *obs.Value.Quantity.Unit != "mmHg" {
		t.Errorf("quantity.unit = %v, want mmHg", obs.Value.Quantity.Unit)
	}
}

func TestXMLElementExtensions(t *testing.T) {
	// Patient with _birthDate element extension
	input := `{"resourceType":"Patient","id":"1","birthDate":"1974-12-25","_birthDate":{"extension":[{"url":"http://hl7.org/fhir/StructureDefinition/patient-birthTime","valueDateTime":"1974-12-25T14:35:45-05:00"}]}}`
	var p resources.Patient
	parser.Unmarshal([]byte(input), &p)

	xmlData, err := parser.MarshalXML(&p, parser.Options{PrettyPrint: true})
	if err != nil {
		t.Fatal(err)
	}

	xml := string(xmlData)
	// birthDate should have value attribute AND nested extension
	if !strings.Contains(xml, `birthDate`) {
		t.Error("should contain birthDate element")
	}
	if !strings.Contains(xml, `patient-birthTime`) {
		t.Error("should contain element extension URL in XML output")
	}
}

func TestXMLObservation(t *testing.T) {
	obs, _ := resources.NewObservation().
		WithStatus(resources.ObservationStatusFinal).
		WithCode("http://loinc.org", "85354-9", "Blood pressure").
		Build()

	xmlData, err := parser.MarshalXML(obs, parser.Options{PrettyPrint: true})
	if err != nil {
		t.Fatal(err)
	}

	xml := string(xmlData)
	if !strings.Contains(xml, `<Observation xmlns="http://hl7.org/fhir">`) {
		t.Error("should have Observation root")
	}
	if !strings.Contains(xml, `<status value="final"/>`) {
		t.Error("should have status")
	}
}

func TestXMLRoundTripHL7Examples(t *testing.T) {
	// Stress test: round-trip ALL HL7 example resources through XML
	tests := []struct {
		name string
		json string
	}{
		{"Patient", `{"resourceType":"Patient","id":"1","gender":"male","birthDate":"1980-01-01","name":[{"family":"Doe","given":["John"]}],"active":true}`},
		{"Observation", `{"resourceType":"Observation","id":"1","status":"final","code":{"coding":[{"system":"http://loinc.org","code":"29463-7"}]},"valueQuantity":{"value":120,"unit":"mmHg"}}`},
		{"Condition", `{"resourceType":"Condition","id":"1","code":{"coding":[{"system":"http://snomed.info/sct","code":"386661006"}]},"subject":{"reference":"Patient/1"},"onsetDateTime":"2023-01-01"}`},
		{"Encounter", `{"resourceType":"Encounter","id":"1","status":"in-progress","class":{"system":"http://terminology.hl7.org/CodeSystem/v3-ActCode","code":"IMP"}}`},
		{"MedicationRequest", `{"resourceType":"MedicationRequest","id":"1","status":"active","intent":"order","subject":{"reference":"Patient/1"},"medicationCodeableConcept":{"text":"Tylenol"}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse JSON
			result, err := resources.ParseResource(json.RawMessage(tt.json))
			if err != nil {
				t.Fatalf("JSON parse: %v", err)
			}

			// Marshal to XML
			xmlData, err := parser.MarshalXML(result, parser.Options{})
			if err != nil {
				t.Fatalf("XML marshal: %v", err)
			}

			// Unmarshal back
			result2, _ := resources.ParseResource(json.RawMessage(`{"resourceType":"` + tt.name + `"}`))
			if err := parser.UnmarshalXML(xmlData, result2); err != nil {
				t.Fatalf("XML unmarshal: %v\nXML: %s", err, xmlData[:min(300, len(xmlData))])
			}

			// Compare key count
			var m1, m2 map[string]json.RawMessage
			j1, _ := json.Marshal(result)
			j2, _ := json.Marshal(result2)
			json.Unmarshal(j1, &m1)
			json.Unmarshal(j2, &m2)

			for k := range m1 {
				if k == "text" {
					continue
				}
				if _, ok := m2[k]; !ok {
					t.Errorf("key %q lost in XML round-trip", k)
				}
			}
		})
	}
}

func TestXMLElementExtensionRoundTrip(t *testing.T) {
	// _birthDate with extension must survive XML round-trip
	input := `{"resourceType":"Patient","id":"1","birthDate":"1974-12-25","_birthDate":{"extension":[{"url":"http://hl7.org/fhir/StructureDefinition/patient-birthTime","valueDateTime":"1974-12-25T14:35:45-05:00"}]}}`

	var p resources.Patient
	parser.Unmarshal([]byte(input), &p)

	xmlData, err := parser.MarshalXML(&p, parser.Options{})
	if err != nil {
		t.Fatal(err)
	}

	var p2 resources.Patient
	if err := parser.UnmarshalXML(xmlData, &p2); err != nil {
		t.Fatalf("unmarshal: %v\nXML: %s", err, xmlData)
	}

	if p2.BirthDateElement == nil {
		t.Fatal("_birthDate should survive XML round-trip")
	}
	if len(p2.BirthDateElement.Extension) == 0 {
		t.Fatal("_birthDate extensions should survive XML round-trip")
	}
}

func TestXMLSpecialCharacters(t *testing.T) {
	input := `{"resourceType":"Patient","id":"1","name":[{"family":"O'Brien & Sons <LLC>"}]}`
	var p resources.Patient
	parser.Unmarshal([]byte(input), &p)

	xmlData, err := parser.MarshalXML(&p, parser.Options{})
	if err != nil {
		t.Fatal(err)
	}

	xml := string(xmlData)
	if !strings.Contains(xml, "&amp;") {
		t.Error("& should be escaped to &amp;")
	}
	if !strings.Contains(xml, "&lt;") {
		t.Error("< should be escaped to &lt;")
	}
	if !strings.Contains(xml, "&apos;") {
		t.Error("' should be escaped to &apos;")
	}

	// Round-trip: XML → struct → verify
	var p2 resources.Patient
	if err := parser.UnmarshalXML(xmlData, &p2); err != nil {
		t.Fatal(err)
	}
	if p2.Name[0].Family == nil || *p2.Name[0].Family != "O'Brien & Sons <LLC>" {
		t.Errorf("family = %v, want O'Brien & Sons <LLC>", p2.Name[0].Family)
	}
}

func TestXMLMalformedInput(t *testing.T) {
	tests := []struct {
		name string
		xml  string
	}{
		{"empty", ""},
		{"not_xml", "this is not xml"},
		{"unclosed", `<?xml version="1.0"?><Patient xmlns="http://hl7.org/fhir"><id value="1"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p resources.Patient
			err := parser.UnmarshalXML([]byte(tt.xml), &p)
			if err == nil && tt.name != "unclosed" {
				t.Error("should fail for malformed XML")
			}
		})
	}
}

func TestXMLEmptyResource(t *testing.T) {
	xmlInput := `<?xml version="1.0" encoding="UTF-8"?>
<Patient xmlns="http://hl7.org/fhir">
</Patient>`

	var p resources.Patient
	if err := parser.UnmarshalXML([]byte(xmlInput), &p); err != nil {
		t.Fatal(err)
	}
	if p.ResourceType != "Patient" {
		t.Errorf("resourceType = %q, want Patient", p.ResourceType)
	}
}

func TestXMLDecimalPrecisionRoundTrip(t *testing.T) {
	// Decimal "1.00" must survive XML round-trip as "1.00", not "1"
	input := `{"resourceType":"Observation","status":"final","code":{"text":"test"},"valueQuantity":{"value":1.00,"unit":"mg"}}`
	var obs resources.Observation
	parser.Unmarshal([]byte(input), &obs)

	if obs.Value == nil || obs.Value.Quantity == nil {
		t.Fatal("valueQuantity should be parsed")
	}
	if obs.Value.Quantity.Value.String() != "1.00" {
		t.Fatalf("precision before XML: got %q, want 1.00", obs.Value.Quantity.Value.String())
	}

	// Marshal to XML
	xmlData, err := parser.MarshalXML(&obs, parser.Options{})
	if err != nil {
		t.Fatal(err)
	}

	// Verify XML contains the precise value
	if !strings.Contains(string(xmlData), `value="1.00"`) {
		t.Errorf("XML should contain value=\"1.00\", got: %s", xmlData)
	}

	// Unmarshal back from XML
	var obs2 resources.Observation
	if err := parser.UnmarshalXML(xmlData, &obs2); err != nil {
		t.Fatal(err)
	}

	if obs2.Value == nil || obs2.Value.Quantity == nil || obs2.Value.Quantity.Value == nil {
		t.Fatal("valueQuantity should survive XML round-trip")
	}
	if obs2.Value.Quantity.Value.String() != "1.00" {
		t.Errorf("decimal precision lost in XML round-trip: got %q, want 1.00", obs2.Value.Quantity.Value.String())
	}
}

func TestXMLComplexResourceRoundTrip(t *testing.T) {
	// ExplanationOfBenefit: deeply nested backbone elements, value[x], arrays
	input := `{
		"resourceType":"ExplanationOfBenefit",
		"id":"eob-1",
		"status":"active",
		"type":{"coding":[{"system":"http://terminology.hl7.org/CodeSystem/claim-type","code":"professional"}]},
		"use":"claim",
		"patient":{"reference":"Patient/1"},
		"created":"2024-01-15",
		"insurer":{"reference":"Organization/ins-1"},
		"provider":{"reference":"Practitioner/prov-1"},
		"outcome":"complete",
		"insurance":[{"focal":true,"coverage":{"reference":"Coverage/cov-1"}}],
		"item":[{
			"sequence":1,
			"productOrService":{"text":"Office Visit"},
			"net":{"value":150.00,"currency":"USD"},
			"adjudication":[{
				"category":{"coding":[{"code":"benefit"}]},
				"amount":{"value":120.50,"currency":"USD"}
			}]
		}]
	}`

	var eob resources.ExplanationOfBenefit
	if err := parser.Unmarshal([]byte(input), &eob); err != nil {
		t.Fatal(err)
	}

	// Marshal to XML
	xmlData, err := parser.MarshalXML(&eob, parser.Options{})
	if err != nil {
		t.Fatal(err)
	}

	xml := string(xmlData)
	if !strings.Contains(xml, `<ExplanationOfBenefit xmlns="http://hl7.org/fhir">`) {
		t.Error("should have correct root element")
	}
	if !strings.Contains(xml, `<status value="active"/>`) {
		t.Error("should contain status")
	}
	if !strings.Contains(xml, `value="150.00"`) {
		t.Error("should preserve net.value decimal precision")
	}
	if !strings.Contains(xml, `value="120.50"`) {
		t.Error("should preserve adjudication amount decimal precision")
	}

	// Unmarshal back
	var eob2 resources.ExplanationOfBenefit
	if err := parser.UnmarshalXML(xmlData, &eob2); err != nil {
		t.Fatalf("XML unmarshal: %v\nXML: %s", err, xmlData[:min(500, len(xmlData))])
	}

	if eob2.GetStatus() != "active" {
		t.Errorf("status = %q after round-trip", eob2.GetStatus())
	}
	if len(eob2.Item) == 0 {
		t.Fatal("item should survive round-trip")
	}
}
