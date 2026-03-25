// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package datatypes_test

import (
	"testing"

	dt "github.com/caucehealth/gofhir/r4/datatypes"
)

func TestExtensionRegistry(t *testing.T) {
	reg := dt.NewExtensionRegistry()
	reg.Register(dt.ExtensionDef{
		URL:       "http://example.org/ext/test",
		Name:      "test",
		ValueType: "String",
	})

	if !reg.IsKnown("http://example.org/ext/test") {
		t.Error("registered extension should be known")
	}
	if reg.IsKnown("http://example.org/ext/unknown") {
		t.Error("unregistered extension should not be known")
	}

	def := reg.Lookup("http://example.org/ext/test")
	if def == nil || def.Name != "test" {
		t.Error("lookup should return registered definition")
	}

	all := reg.All()
	if len(all) != 1 {
		t.Errorf("expected 1 definition, got %d", len(all))
	}
}

func TestUSCoreExtensions(t *testing.T) {
	reg := dt.USCoreExtensions()

	expectedURLs := []string{
		"http://hl7.org/fhir/us/core/StructureDefinition/us-core-race",
		"http://hl7.org/fhir/us/core/StructureDefinition/us-core-ethnicity",
		"http://hl7.org/fhir/us/core/StructureDefinition/us-core-birthsex",
		"http://hl7.org/fhir/StructureDefinition/patient-birthTime",
	}

	for _, url := range expectedURLs {
		if !reg.IsKnown(url) {
			t.Errorf("US Core registry should contain %s", url)
		}
	}
}

func TestGetExtensionValue(t *testing.T) {
	code := dt.Code("M")
	exts := []dt.Extension{
		{Url: "http://example.org/other", ValueString: strPtr("irrelevant")},
		{Url: "http://example.org/birthsex", ValueCode: &code},
	}

	val := dt.GetExtensionValue[dt.Code](exts, "http://example.org/birthsex")
	if val == nil {
		t.Fatal("should find extension value")
	}
	if *val != "M" {
		t.Errorf("value = %q, want M", *val)
	}

	// Non-existent
	missing := dt.GetExtensionValue[string](exts, "http://example.org/missing")
	if missing != nil {
		t.Error("missing extension should return nil")
	}
}
