// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package datatypes

// HumanNameBuilder provides a fluent API for constructing HumanName values.
type HumanNameBuilder struct {
	name HumanName
}

// NewHumanName creates a new HumanNameBuilder.
func NewHumanName() *HumanNameBuilder {
	return &HumanNameBuilder{}
}

// WithFamily sets the family (last) name.
func (b *HumanNameBuilder) WithFamily(family string) *HumanNameBuilder {
	b.name.Family = &family
	return b
}

// WithGiven adds a given (first/middle) name.
func (b *HumanNameBuilder) WithGiven(given string) *HumanNameBuilder {
	b.name.Given = append(b.name.Given, given)
	return b
}

// WithPrefix adds a name prefix (e.g. "Dr.", "Mr.").
func (b *HumanNameBuilder) WithPrefix(prefix string) *HumanNameBuilder {
	b.name.Prefix = append(b.name.Prefix, prefix)
	return b
}

// WithSuffix adds a name suffix (e.g. "Jr.", "III").
func (b *HumanNameBuilder) WithSuffix(suffix string) *HumanNameBuilder {
	b.name.Suffix = append(b.name.Suffix, suffix)
	return b
}

// WithUse sets the name use (e.g. "official", "nickname").
func (b *HumanNameBuilder) WithUse(use string) *HumanNameBuilder {
	b.name.Use = &use
	return b
}

// WithText sets the full text representation of the name.
func (b *HumanNameBuilder) WithText(text string) *HumanNameBuilder {
	b.name.Text = &text
	return b
}

// WithPeriod sets the period during which this name was/is in use.
func (b *HumanNameBuilder) WithPeriod(period Period) *HumanNameBuilder {
	b.name.Period = &period
	return b
}

// Build returns the constructed HumanName.
func (b *HumanNameBuilder) Build() HumanName {
	return b.name
}

// CodingBuilder provides a fluent API for constructing Coding values.
type CodingBuilder struct {
	coding Coding
}

// NewCoding creates a new CodingBuilder.
func NewCoding() *CodingBuilder {
	return &CodingBuilder{}
}

// WithSystem sets the code system URI.
func (b *CodingBuilder) WithSystem(system string) *CodingBuilder {
	s := URI(system)
	b.coding.System = &s
	return b
}

// WithCode sets the code value.
func (b *CodingBuilder) WithCode(code string) *CodingBuilder {
	c := Code(code)
	b.coding.Code = &c
	return b
}

// WithDisplay sets the display text.
func (b *CodingBuilder) WithDisplay(display string) *CodingBuilder {
	b.coding.Display = &display
	return b
}

// Build returns the constructed Coding.
func (b *CodingBuilder) Build() Coding {
	return b.coding
}

// CodeableConceptBuilder provides a fluent API for constructing CodeableConcept values.
type CodeableConceptBuilder struct {
	cc CodeableConcept
}

// NewCodeableConcept creates a new CodeableConceptBuilder.
func NewCodeableConcept() *CodeableConceptBuilder {
	return &CodeableConceptBuilder{}
}

// WithCoding adds a Coding to the CodeableConcept.
func (b *CodeableConceptBuilder) WithCoding(system, code, display string) *CodeableConceptBuilder {
	b.cc.Coding = append(b.cc.Coding, NewCoding().
		WithSystem(system).
		WithCode(code).
		WithDisplay(display).
		Build())
	return b
}

// WithText sets the text representation.
func (b *CodeableConceptBuilder) WithText(text string) *CodeableConceptBuilder {
	b.cc.Text = &text
	return b
}

// Build returns the constructed CodeableConcept.
func (b *CodeableConceptBuilder) Build() CodeableConcept {
	return b.cc
}
