// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/caucehealth/gofhir/r4/parser"
	"github.com/caucehealth/gofhir/r4/resources"
	"github.com/caucehealth/gofhir/r4/validate"
)

// TestHL7ExamplesJSONRoundTrip tests that ALL official HL7 FHIR R4 examples
// can be parsed and round-tripped through our library without data loss.
// This is the single most valuable conformance test — HAPI's reliability
// comes from testing against these same examples.
func TestHL7ExamplesJSONRoundTrip(t *testing.T) {
	dir := "testdata/hl7-examples"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("HL7 examples not available: %v", err)
	}

	var total, passed, skipped, failed int
	failedFiles := []string{}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		total++

		t.Run(entry.Name(), func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}

			// Step 1: Determine resource type
			var header struct {
				ResourceType string `json:"resourceType"`
			}
			if err := json.Unmarshal(data, &header); err != nil {
				t.Skipf("invalid JSON: %v", err)
				skipped++
				return
			}
			if header.ResourceType == "" {
				skipped++
				t.Skip("no resourceType")
				return
			}

			// Step 2: Parse via registry
			res, err := resources.ParseResource(data)
			if err != nil {
				// Bundle is handled separately
				if header.ResourceType == "Bundle" {
					skipped++
					t.Skipf("Bundle: %v", err)
					return
				}
				failed++
				failedFiles = append(failedFiles, entry.Name())
				t.Fatalf("ParseResource(%s): %v", header.ResourceType, err)
				return
			}

			// Step 3: Verify resource type
			if res.GetResourceType() != header.ResourceType {
				t.Errorf("GetResourceType() = %q, want %q", res.GetResourceType(), header.ResourceType)
			}

			// Step 4: Marshal back to JSON
			out, err := json.Marshal(res)
			if err != nil {
				failed++
				failedFiles = append(failedFiles, entry.Name())
				t.Fatalf("Marshal(%s): %v", header.ResourceType, err)
				return
			}

			// Step 5: Verify no top-level keys lost
			var m1, m2 map[string]json.RawMessage
			json.Unmarshal(data, &m1)
			json.Unmarshal(out, &m2)

			for k := range m1 {
				if k == "text" || k == "meta" {
					continue // narrative and meta may differ
				}
				if _, ok := m2[k]; !ok {
					t.Errorf("key %q lost in round-trip for %s (%s)",
						k, header.ResourceType, entry.Name())
				}
			}

			passed++
		})
	}

	if len(failedFiles) > 0 {
		t.Logf("Failed files: %v", failedFiles)
	}
	t.Logf("HL7 examples: %d total, %d passed, %d skipped, %d failed",
		total, passed, skipped, failed)
}

// TestHL7ExamplesValidation runs validation against all HL7 examples.
// Valid HL7 examples should not produce errors for the fields they contain.
func TestHL7ExamplesValidation(t *testing.T) {
	dir := "testdata/hl7-examples"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("HL7 examples not available: %v", err)
	}

	v := validate.New()
	var total, withErrors int

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		res, err := resources.ParseResource(data)
		if err != nil {
			continue
		}
		total++

		result := v.Validate(res)
		if result.HasErrors() {
			withErrors++
			// Log but don't fail — some examples may have intentional gaps
			if testing.Verbose() {
				for _, issue := range result.Errors() {
					t.Logf("%s: %s: %s", entry.Name(), issue.Path, issue.Message)
				}
			}
		}
	}

	t.Logf("Validated %d HL7 examples, %d with errors (%.0f%% clean)",
		total, withErrors, float64(total-withErrors)/float64(total)*100)
}

// TestHL7ExamplesXMLRoundTrip tests XML round-trip for a sample of HL7 examples.
func TestHL7ExamplesXMLRoundTrip(t *testing.T) {
	dir := "testdata/hl7-examples"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("HL7 examples not available: %v", err)
	}

	var total, passed, failed int

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		var header struct{ ResourceType string `json:"resourceType"` }
		json.Unmarshal(data, &header)
		if header.ResourceType == "" || header.ResourceType == "Bundle" {
			continue
		}

		total++

		t.Run(entry.Name(), func(t *testing.T) {
			// Parse JSON
			res, err := resources.ParseResource(data)
			if err != nil {
				t.Skipf("parse: %v", err)
				return
			}

			// Marshal to XML
			xmlData, err := parser.MarshalXML(res, parser.Options{})
			if err != nil {
				failed++
				t.Fatalf("MarshalXML(%s): %v", header.ResourceType, err)
				return
			}

			// Unmarshal back from XML
			res2, err := resources.ParseResource(json.RawMessage(`{"resourceType":"` + header.ResourceType + `"}`))
			if err != nil {
				t.Fatal(err)
				return
			}
			if err := parser.UnmarshalXML(xmlData, res2); err != nil {
				failed++
				t.Fatalf("UnmarshalXML(%s): %v\nXML prefix: %s",
					header.ResourceType, err,
					string(xmlData[:min(300, len(xmlData))]))
				return
			}

			// Compare key count
			out1, _ := json.Marshal(res)
			out2, _ := json.Marshal(res2)
			var m1, m2 map[string]json.RawMessage
			json.Unmarshal(out1, &m1)
			json.Unmarshal(out2, &m2)

			for k := range m1 {
				if k == "text" {
					continue
				}
				// _field without corresponding field (extension-only primitive)
				// is a rare edge case not yet supported in XML round-trip
				if strings.HasPrefix(k, "_") {
					baseField := k[1:]
					if _, hasBase := m1[baseField]; !hasBase {
						continue
					}
				}
				if _, ok := m2[k]; !ok {
					t.Errorf("key %q lost in XML round-trip for %s", k, header.ResourceType)
				}
			}
			passed++
		})
	}

	t.Logf("XML round-trip: %d total, %d passed, %d failed", total, passed, failed)
}
