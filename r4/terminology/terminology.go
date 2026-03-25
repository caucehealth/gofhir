// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

// Package terminology provides FHIR terminology services: code validation,
// ValueSet expansion, CodeSystem lookup, and concept translation.
//
// Usage:
//
//	svc := terminology.NewInMemory()
//	svc.LoadCommonCodeSystems() // UCUM, languages, MIME types, countries
//	result := svc.ValidateCode("http://hl7.org/fhir/administrative-gender", "male")
//	if !result.Valid {
//	    // handle invalid code
//	}
//
// Chain multiple sources:
//
//	svc := terminology.NewChain(
//	    terminology.NewInMemory(),      // local codes
//	    terminology.NewRemote(client),  // external terminology server
//	)
package terminology

// Service is the core terminology service interface.
// Implementations can be in-memory, remote, or chained.
type Service interface {
	// ValidateCode checks if a code is valid within a code system or value set.
	ValidateCode(params ValidateCodeParams) *ValidateCodeResult

	// LookupCode returns details about a code in a code system.
	LookupCode(system, code string) *LookupResult

	// ExpandValueSet returns all codes in a value set.
	ExpandValueSet(url string) *ExpansionResult

	// Subsumes checks if code A subsumes code B in a code system.
	Subsumes(system, codeA, codeB string) SubsumptionResult
}

// ValidateCodeParams contains parameters for the $validate-code operation.
type ValidateCodeParams struct {
	// System is the code system URL.
	System string
	// Code is the code value to validate.
	Code string
	// Display is the optional display text to verify.
	Display string
	// ValueSetURL is an optional ValueSet URL to validate against.
	ValueSetURL string
}

// ValidateCodeResult contains the result of a $validate-code operation.
type ValidateCodeResult struct {
	Valid   bool
	Display string
	Message string
}

// LookupResult contains the result of a $lookup operation.
type LookupResult struct {
	Found      bool
	Display    string
	Definition string
	Properties map[string]string
}

// ExpansionResult contains the result of a $expand operation.
type ExpansionResult struct {
	Concepts []Concept
	Total    int
	Error    string
}

// Concept represents a code in a code system or value set expansion.
type Concept struct {
	System     string
	Code       string
	Display    string
	Definition string
}

// SubsumptionResult represents the outcome of a $subsumes operation.
type SubsumptionResult string

const (
	SubsumesEquivalent    SubsumptionResult = "equivalent"
	SubsumesSubsumes      SubsumptionResult = "subsumes"
	SubsumesSubsumedBy    SubsumptionResult = "subsumed-by"
	SubsumesNotSubsumed   SubsumptionResult = "not-subsumed"
)
