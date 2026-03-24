// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package datatypes_test

import (
	"testing"

	dt "github.com/helixfhir/gofhir/r4/datatypes"
)

func TestExtensionsByURL(t *testing.T) {
	url := dt.URI("http://example.org/ext/a")
	url2 := dt.URI("http://example.org/ext/b")
	exts := []dt.Extension{
		{Url: url, ValueString: strPtr("one")},
		{Url: url2, ValueString: strPtr("two")},
		{Url: url, ValueString: strPtr("three")},
	}

	matches := dt.ExtensionsByURL(exts, "http://example.org/ext/a")
	if len(matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matches))
	}

	single := dt.ExtensionByURL(exts, "http://example.org/ext/b")
	if single == nil || single.ValueString == nil || *single.ValueString != "two" {
		t.Error("expected to find ext/b with value 'two'")
	}

	none := dt.ExtensionByURL(exts, "http://example.org/ext/missing")
	if none != nil {
		t.Error("expected nil for missing URL")
	}
}

func TestParseResourceID(t *testing.T) {
	tests := []struct {
		input   string
		resType string
		id      string
		version string
	}{
		{"Patient/123", "Patient", "123", ""},
		{"Patient/123/_history/2", "Patient", "123", "2"},
		{"http://example.com/fhir/Patient/123", "Patient", "123", ""},
		{"http://example.com/fhir/Patient/123/_history/5", "Patient", "123", "5"},
		{"", "", "", ""},
		{"garbage", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			rid := dt.ParseResourceID(tt.input)
			if rid.Type != tt.resType {
				t.Errorf("Type = %q, want %q", rid.Type, tt.resType)
			}
			if rid.ID != tt.id {
				t.Errorf("ID = %q, want %q", rid.ID, tt.id)
			}
			if rid.Version != tt.version {
				t.Errorf("Version = %q, want %q", rid.Version, tt.version)
			}
		})
	}
}

func TestResourceIDString(t *testing.T) {
	rid := dt.ResourceID{Type: "Patient", ID: "123"}
	if s := rid.String(); s != "Patient/123" {
		t.Errorf("got %q, want Patient/123", s)
	}

	rid.Version = "2"
	if s := rid.String(); s != "Patient/123/_history/2" {
		t.Errorf("got %q, want Patient/123/_history/2", s)
	}
}

func TestNewReference(t *testing.T) {
	ref := dt.NewReference("Patient", "123")
	if ref.Reference == nil || *ref.Reference != "Patient/123" {
		t.Error("expected Patient/123")
	}

	ref2 := dt.NewReferenceWithDisplay("Patient", "123", "John Doe")
	if ref2.Display == nil || *ref2.Display != "John Doe" {
		t.Error("expected display John Doe")
	}
}

func strPtr(s string) *string {
	return &s
}
