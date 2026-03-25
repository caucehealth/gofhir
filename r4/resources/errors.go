// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"fmt"
	"strings"
)

// FHIRError is an error that carries a FHIR OperationOutcome.
// All FHIR-level errors (parsing, validation, not-found, etc.) should use
// this type so callers can extract the OperationOutcome for API responses.
type FHIRError struct {
	// StatusCode is the suggested HTTP status code (e.g., 400, 404, 422).
	StatusCode int
	// Outcome contains the structured error details.
	Outcome *OperationOutcome
}

func (e *FHIRError) Error() string {
	if e.Outcome == nil || len(e.Outcome.Issue) == 0 {
		return fmt.Sprintf("FHIR error (HTTP %d)", e.StatusCode)
	}
	msgs := make([]string, 0, len(e.Outcome.Issue))
	for _, issue := range e.Outcome.Issue {
		if issue.Diagnostics != nil {
			msgs = append(msgs, *issue.Diagnostics)
		}
	}
	return strings.Join(msgs, "; ")
}

// IssueSeverity constants for convenience.
var (
	SeverityFatal       = OperationOutcomeIssueSeverityFatal
	SeverityError       = OperationOutcomeIssueSeverityError
	SeverityWarning     = OperationOutcomeIssueSeverityWarning
	SeverityInformation = OperationOutcomeIssueSeverityInformation
)

// IssueCode constants for convenience.
var (
	IssueCodeInvalid    = OperationOutcomeIssueCodeInvalid
	IssueCodeStructure  = OperationOutcomeIssueCodeStructure
	IssueCodeRequired   = OperationOutcomeIssueCodeRequired
	IssueCodeValue      = OperationOutcomeIssueCodeValue
	IssueCodeNotFound   = OperationOutcomeIssueCodeNotfound
	IssueCodeProcessing = OperationOutcomeIssueCodeProcessing
	IssueCodeSecurity   = OperationOutcomeIssueCodeSecurity
)

// OutcomeBuilder provides a fluent API for constructing OperationOutcome resources.
type OutcomeBuilder struct {
	outcome OperationOutcome
}

// NewOutcome creates a new OutcomeBuilder.
func NewOutcome() *OutcomeBuilder {
	return &OutcomeBuilder{outcome: OperationOutcome{ResourceType: "OperationOutcome"}}
}

// WithIssue adds an issue to the outcome.
func (b *OutcomeBuilder) WithIssue(severity OperationOutcomeIssueSeverity, code OperationOutcomeIssueCode, diagnostics string) *OutcomeBuilder {
	b.outcome.Issue = append(b.outcome.Issue, OperationOutcomeIssue{
		Severity:    &severity,
		Code:        &code,
		Diagnostics: &diagnostics,
	})
	return b
}

// WithIssueAt adds an issue with a FHIRPath expression location.
func (b *OutcomeBuilder) WithIssueAt(severity OperationOutcomeIssueSeverity, code OperationOutcomeIssueCode, diagnostics string, expression string) *OutcomeBuilder {
	b.outcome.Issue = append(b.outcome.Issue, OperationOutcomeIssue{
		Severity:    &severity,
		Code:        &code,
		Diagnostics: &diagnostics,
		Expression:  []string{expression},
	})
	return b
}

// Build returns the final OperationOutcome.
func (b *OutcomeBuilder) Build() *OperationOutcome {
	result := b.outcome
	return &result
}

// NewFHIRError creates a FHIRError from an OutcomeBuilder with the given HTTP status.
func NewFHIRError(statusCode int, outcome *OperationOutcome) *FHIRError {
	return &FHIRError{StatusCode: statusCode, Outcome: outcome}
}

// ErrNotFound creates a 404 FHIRError.
func ErrNotFound(resourceType, id string) *FHIRError {
	return NewFHIRError(404, NewOutcome().
		WithIssue(SeverityError, IssueCodeNotFound,
			fmt.Sprintf("%s/%s not found", resourceType, id)).
		Build())
}

// ErrInvalidResource creates a 400 FHIRError for malformed input.
func ErrInvalidResource(diagnostics string) *FHIRError {
	return NewFHIRError(400, NewOutcome().
		WithIssue(SeverityError, IssueCodeStructure, diagnostics).
		Build())
}

// ErrValidation creates a 422 FHIRError for validation failures.
func ErrValidation(diagnostics string) *FHIRError {
	return NewFHIRError(422, NewOutcome().
		WithIssue(SeverityError, IssueCodeProcessing, diagnostics).
		Build())
}

// HasErrors returns true if the outcome contains any error or fatal issues.
func (o *OperationOutcome) HasErrors() bool {
	for _, issue := range o.Issue {
		if issue.Severity != nil && (*issue.Severity == SeverityError || *issue.Severity == SeverityFatal) {
			return true
		}
	}
	return false
}
