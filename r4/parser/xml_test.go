// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package parser_test

import (
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
	if obs.Value.Quantity.Value == nil || *obs.Value.Quantity.Value != 120 {
		t.Errorf("quantity.value = %v, want 120", obs.Value.Quantity.Value)
	}
	if obs.Value.Quantity.Unit == nil || *obs.Value.Quantity.Unit != "mmHg" {
		t.Errorf("quantity.unit = %v, want mmHg", obs.Value.Quantity.Unit)
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
