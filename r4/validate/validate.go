// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

// Package validate provides FHIR resource validation with composable rules.
// Validation results map directly to OperationOutcome for API responses.
//
// Usage:
//
//	v := validate.New()
//	result := v.Validate(patient)
//	if result.HasErrors() {
//	    outcome := result.ToOperationOutcome()
//	}
//
// Custom rules:
//
//	v := validate.New(validate.WithRules(myCustomRule))
package validate

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	dt "github.com/caucehealth/gofhir/r4/datatypes"
	"github.com/caucehealth/gofhir/r4/resources"
)

// FHIR primitive format patterns
var (
	idPattern       = regexp.MustCompile(`^[A-Za-z0-9\-.]{1,64}$`)
	datePattern     = regexp.MustCompile(`^\d{4}(-\d{2}(-\d{2})?)?$`)
	dateTimePattern = regexp.MustCompile(`^\d{4}(-\d{2}(-\d{2}(T\d{2}:\d{2}(:\d{2}(\.\d+)?)?(Z|[+-]\d{2}:\d{2})?)?)?)?$`)
	instantPattern  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$`)
	oidPattern      = regexp.MustCompile(`^urn:oid:[0-2](\.\d+)+$`)
)

// Validator validates FHIR resources using a chain of rules.
type Validator struct {
	rules []Rule
}

// Rule is a single validation rule that checks a resource.
type Rule interface {
	Validate(resource resources.Resource) []Issue
}

// RuleFunc adapts a function to the Rule interface.
type RuleFunc func(resources.Resource) []Issue

// Validate implements the Rule interface.
func (f RuleFunc) Validate(resource resources.Resource) []Issue {
	return f(resource)
}

// Issue represents a single validation issue.
type Issue struct {
	Severity   Severity
	Code       IssueCode
	Path       string // FHIRPath expression (e.g., "Patient.name")
	Message    string
	ResourceID string
}

// Severity levels for validation issues.
type Severity string

const (
	SeverityFatal       Severity = "fatal"
	SeverityError       Severity = "error"
	SeverityWarning     Severity = "warning"
	SeverityInformation Severity = "information"
)

// IssueCode categorizes validation issues.
type IssueCode string

const (
	CodeRequired    IssueCode = "required"
	CodeValue       IssueCode = "value"
	CodeInvariant   IssueCode = "invariant"
	CodeStructure   IssueCode = "structure"
	CodeCodeInvalid IssueCode = "code-invalid"
	CodeProcessing  IssueCode = "processing"
)

// Result holds all issues from validating a resource.
type Result struct {
	Issues []Issue
}

// HasErrors returns true if there are any error or fatal issues.
func (r *Result) HasErrors() bool {
	for _, i := range r.Issues {
		if i.Severity == SeverityError || i.Severity == SeverityFatal {
			return true
		}
	}
	return false
}

// Errors returns only error and fatal issues.
func (r *Result) Errors() []Issue {
	var errs []Issue
	for _, i := range r.Issues {
		if i.Severity == SeverityError || i.Severity == SeverityFatal {
			errs = append(errs, i)
		}
	}
	return errs
}

// Warnings returns only warning issues.
func (r *Result) Warnings() []Issue {
	var warns []Issue
	for _, i := range r.Issues {
		if i.Severity == SeverityWarning {
			warns = append(warns, i)
		}
	}
	return warns
}

// ToOperationOutcome converts the validation result to a FHIR OperationOutcome.
func (r *Result) ToOperationOutcome() *resources.OperationOutcome {
	b := resources.NewOutcome()
	for _, i := range r.Issues {
		severity := resources.OperationOutcomeIssueSeverity(i.Severity)
		code := resources.OperationOutcomeIssueCode(i.Code)
		if i.Path != "" {
			b.WithIssueAt(severity, code, i.Message, i.Path)
		} else {
			b.WithIssue(severity, code, i.Message)
		}
	}
	return b.Build()
}

// Option configures a Validator.
type Option func(*Validator)

// WithRules adds custom rules to the validator.
func WithRules(rules ...Rule) Option {
	return func(v *Validator) {
		v.rules = append(v.rules, rules...)
	}
}

// New creates a Validator with the default rule set (required fields, enum
// bindings, cardinality) plus any additional rules provided via options.
func New(opts ...Option) *Validator {
	v := &Validator{
		rules: []Rule{
			&requiredFieldRule{},
			&enumBindingRule{},
			&cardinalityRule{},
			&primitiveFormatRule{},
		},
	}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// NewEmpty creates a Validator with no default rules. Use WithRules to add.
func NewEmpty(opts ...Option) *Validator {
	v := &Validator{}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// Validate checks a resource against all registered rules.
func (v *Validator) Validate(resource resources.Resource) *Result {
	result := &Result{}
	for _, rule := range v.rules {
		issues := rule.Validate(resource)
		result.Issues = append(result.Issues, issues...)
	}
	return result
}

// --- Built-in rules ---

// requiredFieldRule validates that required fields (cardinality 1..1) are present.
type requiredFieldRule struct{}

func (r *requiredFieldRule) Validate(resource resources.Resource) []Issue {
	rt := resource.GetResourceType()
	meta := GetResourceMeta(rt)
	if meta == nil {
		return nil
	}

	v := reflect.ValueOf(resource)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	var issues []Issue
	for _, fm := range meta.Fields {
		if !fm.Required {
			continue
		}
		if fm.JSONName == "resourceType" {
			continue
		}

		fieldVal := findFieldByJSON(v, fm.JSONName)
		if !fieldVal.IsValid() {
			continue
		}

		if isEmpty(fieldVal) {
			issues = append(issues, Issue{
				Severity: SeverityError,
				Code:     CodeRequired,
				Path:     rt + "." + fm.JSONName,
				Message:  fmt.Sprintf("%s.%s: minimum required (1)", rt, fm.JSONName),
			})
		}
	}
	return issues
}

// enumBindingRule validates that coded fields contain valid enum values.
type enumBindingRule struct{}

func (r *enumBindingRule) Validate(resource resources.Resource) []Issue {
	rt := resource.GetResourceType()
	meta := GetResourceMeta(rt)
	if meta == nil {
		return nil
	}

	v := reflect.ValueOf(resource)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	var issues []Issue
	for _, fm := range meta.Fields {
		if len(fm.Enum) == 0 {
			continue
		}

		fieldVal := findFieldByJSON(v, fm.JSONName)
		if !fieldVal.IsValid() || isEmpty(fieldVal) {
			continue
		}

		// Dereference pointer
		if fieldVal.Kind() == reflect.Ptr {
			fieldVal = fieldVal.Elem()
		}

		val := fmt.Sprintf("%v", fieldVal.Interface())
		valid := false
		for _, e := range fm.Enum {
			if e == val {
				valid = true
				break
			}
		}
		if !valid {
			issues = append(issues, Issue{
				Severity: SeverityError,
				Code:     CodeCodeInvalid,
				Path:     rt + "." + fm.JSONName,
				Message:  fmt.Sprintf("%s.%s: value %q is not in the required value set %v", rt, fm.JSONName, val, fm.Enum),
			})
		}
	}
	return issues
}

// cardinalityRule checks min/max cardinality for array fields.
type cardinalityRule struct{}

func (r *cardinalityRule) Validate(resource resources.Resource) []Issue {
	rt := resource.GetResourceType()
	meta := GetResourceMeta(rt)
	if meta == nil {
		return nil
	}

	v := reflect.ValueOf(resource)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	var issues []Issue
	for _, fm := range meta.Fields {
		if !fm.IsArray || !fm.Required {
			continue
		}
		if fm.JSONName == "resourceType" {
			continue
		}

		fieldVal := findFieldByJSON(v, fm.JSONName)
		if !fieldVal.IsValid() {
			continue
		}

		if fieldVal.Kind() == reflect.Slice && fieldVal.Len() == 0 {
			issues = append(issues, Issue{
				Severity: SeverityError,
				Code:     CodeRequired,
				Path:     rt + "." + fm.JSONName,
				Message:  fmt.Sprintf("%s.%s: minimum required (1), but array is empty", rt, fm.JSONName),
			})
		}
	}
	return issues
}

// primitiveFormatRule validates format constraints on primitive types.
type primitiveFormatRule struct{}

func (r *primitiveFormatRule) Validate(resource resources.Resource) []Issue {
	rt := resource.GetResourceType()
	v := reflect.ValueOf(resource)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	var issues []Issue
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldVal := v.Field(i)

		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		if strings.HasPrefix(name, "_") {
			continue
		}

		// Skip nil/empty
		if fieldVal.Kind() == reflect.Ptr {
			if fieldVal.IsNil() {
				continue
			}
			fieldVal = fieldVal.Elem()
		}

		path := rt + "." + name
		typeName := fieldVal.Type().Name()

		switch typeName {
		case "ID":
			id := string(fieldVal.Interface().(dt.ID))
			if len(id) > 64 {
				issues = append(issues, Issue{
					Severity: SeverityError, Code: CodeValue, Path: path,
					Message: fmt.Sprintf("%s: id length %d exceeds maximum 64", path, len(id)),
				})
			}
			if id != "" && !idPattern.MatchString(id) {
				issues = append(issues, Issue{
					Severity: SeverityError, Code: CodeValue, Path: path,
					Message: fmt.Sprintf("%s: id %q does not match pattern [A-Za-z0-9\\-.]{1,64}", path, id),
				})
			}
		case "Date":
			s := string(fieldVal.Interface().(dt.Date))
			if s != "" && !datePattern.MatchString(s) {
				issues = append(issues, Issue{
					Severity: SeverityError, Code: CodeValue, Path: path,
					Message: fmt.Sprintf("%s: date %q does not match YYYY, YYYY-MM, or YYYY-MM-DD", path, s),
				})
			}
		case "DateTime":
			s := string(fieldVal.Interface().(dt.DateTime))
			if s != "" && !dateTimePattern.MatchString(s) {
				issues = append(issues, Issue{
					Severity: SeverityError, Code: CodeValue, Path: path,
					Message: fmt.Sprintf("%s: dateTime %q is not valid", path, s),
				})
			}
		case "Instant":
			s := string(fieldVal.Interface().(dt.Instant))
			if s != "" && !instantPattern.MatchString(s) {
				issues = append(issues, Issue{
					Severity: SeverityError, Code: CodeValue, Path: path,
					Message: fmt.Sprintf("%s: instant %q must include timezone", path, s),
				})
			}
		case "OID":
			s := string(fieldVal.Interface().(dt.OID))
			if s != "" && !oidPattern.MatchString(s) {
				issues = append(issues, Issue{
					Severity: SeverityError, Code: CodeValue, Path: path,
					Message: fmt.Sprintf("%s: oid %q must start with urn:oid:", path, s),
				})
			}
		case "Code":
			s := string(fieldVal.Interface().(dt.Code))
			if s != "" && strings.ContainsAny(s, " \t\n\r") {
				issues = append(issues, Issue{
					Severity: SeverityError, Code: CodeValue, Path: path,
					Message: fmt.Sprintf("%s: code %q must not contain whitespace", path, s),
				})
			}
		}
	}
	return issues
}

// --- Helpers ---

func findFieldByJSON(v reflect.Value, jsonName string) reflect.Value {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("json")
		if tag == "" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		if name == jsonName {
			return v.Field(i)
		}
	}
	return reflect.Value{}
}

func isEmpty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Map:
		return v.IsNil() || v.Len() == 0
	case reflect.String:
		return v.String() == ""
	case reflect.Struct:
		// For required struct fields (like Observation.Code), check if it's a zero value
		return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
	default:
		return false
	}
}

// --- Metadata lookup (populated by generated code) ---

// ResourceMeta holds validation metadata for a resource type.
type ResourceMeta struct {
	Name   string
	Fields []FieldMeta
}

// FieldMeta holds validation metadata for a single field.
type FieldMeta struct {
	JSONName string
	FHIRType string
	Required bool
	IsArray  bool
	Enum     []string
}

var resourceMetaRegistry = map[string]*ResourceMeta{}

// RegisterResourceMeta registers validation metadata for a resource type.
// Called by generated init code.
func RegisterResourceMeta(meta *ResourceMeta) {
	resourceMetaRegistry[meta.Name] = meta
}

// GetResourceMeta returns validation metadata for a resource type.
func GetResourceMeta(name string) *ResourceMeta {
	return resourceMetaRegistry[name]
}

// ValidateJSON validates raw JSON as a FHIR resource without needing to
// unmarshal it first. Parses, validates, and returns the result.
func ValidateJSON(data json.RawMessage) (*Result, error) {
	resource, err := resources.ParseResource(data)
	if err != nil {
		return nil, err
	}
	v := New()
	return v.Validate(resource), nil
}
