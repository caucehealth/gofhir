// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package terminology_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/caucehealth/gofhir/r4/client"
	"github.com/caucehealth/gofhir/r4/terminology"
)

func commonSvc() *terminology.InMemory {
	svc := terminology.NewInMemory()
	svc.LoadCommonCodeSystems()
	return svc
}

// === ValidateCode ===

func TestValidateCodeValid(t *testing.T) {
	svc := commonSvc()
	result := svc.ValidateCode(terminology.ValidateCodeParams{
		System: "http://hl7.org/fhir/administrative-gender",
		Code:   "male",
	})
	if !result.Valid {
		t.Error("male should be valid")
	}
	if result.Display != "Male" {
		t.Errorf("display = %q, want Male", result.Display)
	}
}

func TestValidateCodeInvalid(t *testing.T) {
	svc := commonSvc()
	result := svc.ValidateCode(terminology.ValidateCodeParams{
		System: "http://hl7.org/fhir/administrative-gender",
		Code:   "invalid",
	})
	if result.Valid {
		t.Error("invalid should not be valid")
	}
}

func TestValidateCodeUnknownSystem(t *testing.T) {
	svc := commonSvc()
	result := svc.ValidateCode(terminology.ValidateCodeParams{
		System: "http://unknown-system",
		Code:   "test",
	})
	if result.Valid {
		t.Error("unknown system should fail")
	}
	if result.Message == "" {
		t.Error("should have error message")
	}
}

func TestValidateCodeDisplayMismatch(t *testing.T) {
	svc := commonSvc()
	result := svc.ValidateCode(terminology.ValidateCodeParams{
		System:  "http://hl7.org/fhir/administrative-gender",
		Code:    "male",
		Display: "Wrong Display",
	})
	if !result.Valid {
		t.Error("code itself should be valid")
	}
	if result.Message == "" {
		t.Error("should warn about display mismatch")
	}
}

func TestValidateCodeAgainstValueSet(t *testing.T) {
	svc := commonSvc()
	svc.AddValueSet(&terminology.ValueSet{
		URL:  "http://example.org/vs/test",
		Name: "TestValueSet",
		Includes: []terminology.ValueSetInclude{
			{System: "http://hl7.org/fhir/administrative-gender",
				Concepts: []terminology.Concept{
					{Code: "male", Display: "Male"},
					{Code: "female", Display: "Female"},
				}},
		},
	})

	result := svc.ValidateCode(terminology.ValidateCodeParams{
		Code:        "male",
		ValueSetURL: "http://example.org/vs/test",
	})
	if !result.Valid {
		t.Error("male should be in value set")
	}

	result = svc.ValidateCode(terminology.ValidateCodeParams{
		Code:        "other",
		ValueSetURL: "http://example.org/vs/test",
	})
	if result.Valid {
		t.Error("other should not be in value set")
	}
}

func TestValidateCodeValueSetIncludeAll(t *testing.T) {
	svc := commonSvc()
	svc.AddValueSet(&terminology.ValueSet{
		URL:  "http://example.org/vs/all-genders",
		Name: "AllGenders",
		Includes: []terminology.ValueSetInclude{
			{System: "http://hl7.org/fhir/administrative-gender"},
		},
	})

	// "other" exists in the gender code system and should be included
	result := svc.ValidateCode(terminology.ValidateCodeParams{
		System:      "http://hl7.org/fhir/administrative-gender",
		Code:        "other",
		ValueSetURL: "http://example.org/vs/all-genders",
	})
	if !result.Valid {
		t.Error("other should be valid when all codes included")
	}
}

// === LookupCode ===

func TestLookupCodeFound(t *testing.T) {
	svc := commonSvc()
	result := svc.LookupCode("http://hl7.org/fhir/observation-status", "final")
	if !result.Found {
		t.Fatal("final should be found")
	}
	if result.Display != "Final" {
		t.Errorf("display = %q, want Final", result.Display)
	}
}

func TestLookupCodeNotFound(t *testing.T) {
	svc := commonSvc()
	result := svc.LookupCode("http://hl7.org/fhir/observation-status", "nonexistent")
	if result.Found {
		t.Error("should not be found")
	}
}

func TestLookupUCUM(t *testing.T) {
	svc := commonSvc()
	result := svc.LookupCode("http://unitsofmeasure.org", "mm[Hg]")
	if !result.Found {
		t.Fatal("mm[Hg] should be found in UCUM")
	}
	if result.Display != "millimeter of mercury" {
		t.Errorf("display = %q", result.Display)
	}
}

func TestLookupLanguage(t *testing.T) {
	svc := commonSvc()
	result := svc.LookupCode("urn:ietf:bcp:47", "en-US")
	if !result.Found {
		t.Fatal("en-US should be found")
	}
}

func TestLookupCountry(t *testing.T) {
	svc := commonSvc()
	result := svc.LookupCode("urn:iso:std:iso:3166", "US")
	if !result.Found || result.Display != "United States" {
		t.Errorf("US lookup: found=%v display=%q", result.Found, result.Display)
	}
}

// === ExpandValueSet ===

func TestExpandValueSet(t *testing.T) {
	svc := commonSvc()
	svc.AddValueSet(&terminology.ValueSet{
		URL:  "http://example.org/vs/obs-status",
		Name: "ObservationStatus",
		Includes: []terminology.ValueSetInclude{
			{System: "http://hl7.org/fhir/observation-status"},
		},
	})

	result := svc.ExpandValueSet("http://example.org/vs/obs-status")
	if result.Error != "" {
		t.Fatal(result.Error)
	}
	if result.Total < 5 {
		t.Errorf("expected >= 5 concepts, got %d", result.Total)
	}
}

func TestExpandValueSetExplicit(t *testing.T) {
	svc := commonSvc()
	svc.AddValueSet(&terminology.ValueSet{
		URL:  "http://example.org/vs/two-codes",
		Name: "TwoCodes",
		Expanded: []terminology.Concept{
			{System: "http://example.org", Code: "a", Display: "Alpha"},
			{System: "http://example.org", Code: "b", Display: "Beta"},
		},
	})

	result := svc.ExpandValueSet("http://example.org/vs/two-codes")
	if result.Total != 2 {
		t.Errorf("expected 2 concepts, got %d", result.Total)
	}
}

func TestExpandUnknownValueSet(t *testing.T) {
	svc := commonSvc()
	result := svc.ExpandValueSet("http://unknown")
	if result.Error == "" {
		t.Error("should error for unknown value set")
	}
}

// === Subsumes ===

func TestSubsumesEquivalent(t *testing.T) {
	svc := commonSvc()
	result := svc.Subsumes("http://hl7.org/fhir/administrative-gender", "male", "male")
	if result != terminology.SubsumesEquivalent {
		t.Errorf("same code should be equivalent, got %s", result)
	}
}

// === Chain ===

func TestChain(t *testing.T) {
	local := terminology.NewInMemory()
	local.AddCodeSystem(&terminology.CodeSystem{
		URL:  "http://local.org/cs",
		Name: "Local",
		Concepts: map[string]*terminology.Concept{
			"L1": {Code: "L1", Display: "Local One"},
		},
	})

	common := commonSvc()
	chain := terminology.NewChain(local, common)

	// Local code found
	result := chain.ValidateCode(terminology.ValidateCodeParams{
		System: "http://local.org/cs", Code: "L1",
	})
	if !result.Valid {
		t.Error("L1 should be valid from local")
	}

	// Common code found via fallback
	result = chain.ValidateCode(terminology.ValidateCodeParams{
		System: "http://hl7.org/fhir/administrative-gender", Code: "female",
	})
	if !result.Valid {
		t.Error("female should be valid from common (fallback)")
	}

	// Unknown in both
	result = chain.ValidateCode(terminology.ValidateCodeParams{
		System: "http://unknown", Code: "x",
	})
	if result.Valid {
		t.Error("unknown should fail in chain")
	}
}

func TestChainLookup(t *testing.T) {
	chain := terminology.NewChain(commonSvc())
	result := chain.LookupCode("http://unitsofmeasure.org", "kg")
	if !result.Found || result.Display != "kilogram" {
		t.Errorf("kg: found=%v display=%q", result.Found, result.Display)
	}
}

func TestChainExpand(t *testing.T) {
	svc := commonSvc()
	svc.AddValueSet(&terminology.ValueSet{
		URL:      "http://example.org/vs/test",
		Expanded: []terminology.Concept{{Code: "a"}},
	})
	chain := terminology.NewChain(svc)
	result := chain.ExpandValueSet("http://example.org/vs/test")
	if result.Error != "" || result.Total != 1 {
		t.Errorf("expand via chain: error=%q total=%d", result.Error, result.Total)
	}
}

// === Common code systems coverage ===

func TestCommonCodeSystemsCoverage(t *testing.T) {
	svc := commonSvc()
	systems := []struct {
		url  string
		code string
	}{
		{"http://hl7.org/fhir/administrative-gender", "male"},
		{"http://hl7.org/fhir/observation-status", "final"},
		{"http://hl7.org/fhir/encounter-status", "finished"},
		{"http://terminology.hl7.org/CodeSystem/condition-clinical", "active"},
		{"http://terminology.hl7.org/CodeSystem/condition-ver-status", "confirmed"},
		{"http://hl7.org/fhir/request-status", "active"},
		{"http://hl7.org/fhir/request-intent", "order"},
		{"http://hl7.org/fhir/bundle-type", "searchset"},
		{"http://hl7.org/fhir/narrative-status", "generated"},
		{"http://hl7.org/fhir/publication-status", "active"},
		{"http://hl7.org/fhir/contact-point-system", "phone"},
		{"http://hl7.org/fhir/contact-point-use", "home"},
		{"http://hl7.org/fhir/address-use", "home"},
		{"http://hl7.org/fhir/name-use", "official"},
		{"http://hl7.org/fhir/identifier-use", "usual"},
		{"http://unitsofmeasure.org", "mg"},
		{"urn:ietf:bcp:13", "application/json"},
		{"urn:ietf:bcp:47", "en"},
		{"urn:iso:std:iso:3166", "US"},
	}

	for _, tt := range systems {
		t.Run(tt.url+"/"+tt.code, func(t *testing.T) {
			result := svc.ValidateCode(terminology.ValidateCodeParams{
				System: tt.url, Code: tt.code,
			})
			if !result.Valid {
				t.Errorf("%s#%s should be valid", tt.url, tt.code)
			}
		})
	}
}

// === Remote (mock server) ===

func TestRemoteValidateCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		valid := code == "male"
		json.NewEncoder(w).Encode(map[string]any{
			"resourceType": "Parameters",
			"parameter": []map[string]any{
				{"name": "result", "valueBoolean": valid},
				{"name": "display", "valueString": "Male"},
			},
		})
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	remote := terminology.NewRemote(c).WithContext(context.Background())

	result := remote.ValidateCode(terminology.ValidateCodeParams{
		System: "http://hl7.org/fhir/administrative-gender",
		Code:   "male",
	})
	if !result.Valid {
		t.Error("remote: male should be valid")
	}

	result = remote.ValidateCode(terminology.ValidateCodeParams{
		System: "http://hl7.org/fhir/administrative-gender",
		Code:   "invalid",
	})
	if result.Valid {
		t.Error("remote: invalid should not be valid")
	}
}

func TestRemoteExpandValueSet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"resourceType": "ValueSet",
			"expansion": map[string]any{
				"total": 2,
				"contains": []map[string]any{
					{"system": "http://example.org", "code": "a", "display": "Alpha"},
					{"system": "http://example.org", "code": "b", "display": "Beta"},
				},
			},
		})
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	remote := terminology.NewRemote(c)

	result := remote.ExpandValueSet("http://example.org/vs/test")
	if result.Error != "" {
		t.Fatal(result.Error)
	}
	if len(result.Concepts) != 2 {
		t.Errorf("expected 2 concepts, got %d", len(result.Concepts))
	}
}
