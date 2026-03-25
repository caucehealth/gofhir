// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package bundle_test

import (
	"encoding/json"
	"testing"

	"github.com/caucehealth/gofhir/r4/bundle"
	dt "github.com/caucehealth/gofhir/r4/datatypes"
	"github.com/caucehealth/gofhir/r4/resources"
)

func TestBundleBuilder(t *testing.T) {
	p, err := resources.NewPatient().
		WithName("John", "Doe").
		WithGender(resources.AdministrativeGenderMale).
		Build()
	if err != nil {
		t.Fatalf("build patient: %v", err)
	}

	b := bundle.New(bundle.TypeSearchset).
		WithID("test-bundle").
		WithTotal(1).
		WithEntry(p).
		Build()

	if b.ResourceType != "Bundle" {
		t.Errorf("resourceType = %q, want Bundle", b.ResourceType)
	}
	if string(b.Type) != "searchset" {
		t.Errorf("type = %q, want searchset", b.Type)
	}
	if b.Total == nil || *b.Total != 1 {
		t.Error("total should be 1")
	}
	if len(b.Entry) != 1 {
		t.Fatal("should have one entry")
	}
	if b.Entry[0].Resource == nil {
		t.Fatal("entry resource should not be nil")
	}
}

func TestBundleRoundTrip(t *testing.T) {
	input := `{
		"resourceType": "Bundle",
		"type": "searchset",
		"total": 1,
		"entry": [{
			"fullUrl": "http://example.com/Patient/1",
			"resource": {
				"resourceType": "Patient",
				"id": "1",
				"name": [{"family": "Doe"}]
			}
		}]
	}`

	var b bundle.Bundle
	if err := json.Unmarshal([]byte(input), &b); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if b.Type != bundle.TypeSearchset {
		t.Errorf("type = %q, want searchset", b.Type)
	}
	if b.Total == nil || *b.Total != 1 {
		t.Error("total should be 1")
	}
	if len(b.Entry) != 1 {
		t.Fatal("should have one entry")
	}

	// Verify we can parse the contained resource
	var patient resources.Patient
	if err := json.Unmarshal(b.Entry[0].Resource, &patient); err != nil {
		t.Fatalf("unmarshal patient from bundle: %v", err)
	}
	if patient.Id == nil || string(*patient.Id) != "1" {
		t.Error("patient id should be 1")
	}

	// Round-trip
	out, err := json.Marshal(&b)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var b2 bundle.Bundle
	if err := json.Unmarshal(out, &b2); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if b2.Type != bundle.TypeSearchset {
		t.Error("round-trip: type mismatch")
	}
	if len(b2.Entry) != 1 {
		t.Error("round-trip: entry count mismatch")
	}
}

func TestBundleMarshalJSON(t *testing.T) {
	p, err := resources.NewPatient().
		WithName("Jane", "Smith").
		Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	b := bundle.New(bundle.TypeCollection).
		WithEntry(p).
		Build()

	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal map: %v", err)
	}

	if _, ok := m["resourceType"]; !ok {
		t.Error("resourceType should be present")
	}
	if _, ok := m["type"]; !ok {
		t.Error("type should be present")
	}
	if _, ok := m["entry"]; !ok {
		t.Error("entry should be present")
	}
}

func TestBundleEmptyEntries(t *testing.T) {
	b := bundle.New(bundle.TypeTransaction).Build()

	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Empty entries should be omitted
	if _, ok := m["entry"]; ok {
		t.Error("empty entry array should be omitted")
	}
}

func TestBundleWithLinks(t *testing.T) {
	b := bundle.New(bundle.TypeSearchset).
		WithTotal(100).
		WithLink("self", "http://example.com/Patient?page=2").
		WithLink("next", "http://example.com/Patient?page=3").
		WithLink("previous", "http://example.com/Patient?page=1").
		Build()

	if len(b.Link) != 3 {
		t.Fatalf("expected 3 links, got %d", len(b.Link))
	}
	if b.Link[0].Relation != "self" {
		t.Error("first link should be self")
	}
	if string(b.Link[1].URL) != "http://example.com/Patient?page=3" {
		t.Error("next link URL mismatch")
	}

	// Round-trip
	data, _ := json.Marshal(b)
	var m map[string]json.RawMessage
	json.Unmarshal(data, &m)
	if _, ok := m["link"]; !ok {
		t.Error("link should be present in JSON")
	}

	var b2 bundle.Bundle
	json.Unmarshal(data, &b2)
	if len(b2.Link) != 3 {
		t.Error("links should survive round-trip")
	}
}

func TestBundleWithLinkRoundTrip(t *testing.T) {
	input := `{
		"resourceType": "Bundle",
		"type": "searchset",
		"total": 50,
		"link": [
			{"relation": "self", "url": "http://example.com/Patient"},
			{"relation": "next", "url": "http://example.com/Patient?page=2"}
		],
		"entry": [{"resource": {"resourceType": "Patient", "id": "1"}}]
	}`

	var b bundle.Bundle
	if err := json.Unmarshal([]byte(input), &b); err != nil {
		t.Fatal(err)
	}
	if len(b.Link) != 2 {
		t.Fatalf("expected 2 links, got %d", len(b.Link))
	}

	out, _ := json.Marshal(&b)
	var b2 bundle.Bundle
	json.Unmarshal(out, &b2)
	if len(b2.Link) != 2 {
		t.Error("links should survive round-trip")
	}
	if b2.Link[1].Relation != "next" {
		t.Error("link relation mismatch")
	}
}

func TestBundleTransaction(t *testing.T) {
	p, _ := resources.NewPatient().
		WithName("Jane", "Doe").
		Build()

	b := bundle.New(bundle.TypeTransaction).
		WithTransactionEntry("POST", "Patient", p).
		Build()

	if len(b.Entry) != 1 {
		t.Fatal("should have one entry")
	}
	if b.Entry[0].Request == nil {
		t.Fatal("entry should have request")
	}
	if b.Entry[0].Request.Method != "POST" {
		t.Error("method should be POST")
	}
	if string(b.Entry[0].Request.URL) != "Patient" {
		t.Error("URL should be Patient")
	}

	// Round-trip
	data, _ := json.Marshal(b)
	var b2 bundle.Bundle
	json.Unmarshal(data, &b2)
	if b2.Entry[0].Request == nil || b2.Entry[0].Request.Method != "POST" {
		t.Error("request should survive round-trip")
	}
}

func TestBundleTransactionResponse(t *testing.T) {
	input := `{
		"resourceType": "Bundle",
		"type": "transaction-response",
		"entry": [{
			"response": {
				"status": "201 Created",
				"location": "Patient/123/_history/1",
				"etag": "W/\"1\"",
				"lastModified": "2024-01-15T10:00:00Z"
			}
		}]
	}`

	var b bundle.Bundle
	if err := json.Unmarshal([]byte(input), &b); err != nil {
		t.Fatal(err)
	}
	if len(b.Entry) != 1 || b.Entry[0].Response == nil {
		t.Fatal("should have entry with response")
	}
	if b.Entry[0].Response.Status != "201 Created" {
		t.Error("status mismatch")
	}
	if b.Entry[0].Response.Etag == nil || *b.Entry[0].Response.Etag != `W/"1"` {
		t.Error("etag mismatch")
	}

	// Round-trip
	out, _ := json.Marshal(&b)
	var b2 bundle.Bundle
	json.Unmarshal(out, &b2)
	if b2.Entry[0].Response == nil || b2.Entry[0].Response.Status != "201 Created" {
		t.Error("response should survive round-trip")
	}
}

func TestBundleWithMeta(t *testing.T) {
	b := bundle.New(bundle.TypeSearchset).
		WithMeta(dt.NewMeta().WithLastUpdated("2024-01-15T10:00:00Z").Build()).
		WithTimestamp("2024-01-15T10:00:00Z").
		Build()

	if b.Meta == nil {
		t.Fatal("meta should be present")
	}

	data, _ := json.Marshal(b)
	var m map[string]json.RawMessage
	json.Unmarshal(data, &m)
	if _, ok := m["meta"]; !ok {
		t.Error("meta should be in JSON")
	}
	if _, ok := m["timestamp"]; !ok {
		t.Error("timestamp should be in JSON")
	}
}
