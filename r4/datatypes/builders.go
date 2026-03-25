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

// AddressBuilder provides a fluent API for constructing Address values.
type AddressBuilder struct {
	addr Address
}

// NewAddress creates a new AddressBuilder.
func NewAddress() *AddressBuilder {
	return &AddressBuilder{}
}

// WithUse sets the address use (e.g. "home", "work").
func (b *AddressBuilder) WithUse(use string) *AddressBuilder {
	b.addr.Use = &use
	return b
}

// WithLine adds an address line.
func (b *AddressBuilder) WithLine(line string) *AddressBuilder {
	b.addr.Line = append(b.addr.Line, line)
	return b
}

// WithCity sets the city name.
func (b *AddressBuilder) WithCity(city string) *AddressBuilder {
	b.addr.City = &city
	return b
}

// WithState sets the state or province.
func (b *AddressBuilder) WithState(state string) *AddressBuilder {
	b.addr.State = &state
	return b
}

// WithPostalCode sets the postal code.
func (b *AddressBuilder) WithPostalCode(postalCode string) *AddressBuilder {
	b.addr.PostalCode = &postalCode
	return b
}

// WithCountry sets the country.
func (b *AddressBuilder) WithCountry(country string) *AddressBuilder {
	b.addr.Country = &country
	return b
}

// Build returns the constructed Address.
func (b *AddressBuilder) Build() Address {
	return b.addr
}

// ContactPointBuilder provides a fluent API for constructing ContactPoint values.
type ContactPointBuilder struct {
	cp ContactPoint
}

// NewContactPoint creates a new ContactPointBuilder.
func NewContactPoint() *ContactPointBuilder {
	return &ContactPointBuilder{}
}

// WithSystem sets the contact point system (e.g. "phone", "email").
func (b *ContactPointBuilder) WithSystem(system string) *ContactPointBuilder {
	b.cp.System = &system
	return b
}

// WithValue sets the contact point value.
func (b *ContactPointBuilder) WithValue(value string) *ContactPointBuilder {
	b.cp.Value = &value
	return b
}

// WithUse sets the contact point use (e.g. "home", "work", "mobile").
func (b *ContactPointBuilder) WithUse(use string) *ContactPointBuilder {
	b.cp.Use = &use
	return b
}

// WithRank sets the preference order.
func (b *ContactPointBuilder) WithRank(rank uint32) *ContactPointBuilder {
	b.cp.Rank = &rank
	return b
}

// Build returns the constructed ContactPoint.
func (b *ContactPointBuilder) Build() ContactPoint {
	return b.cp
}

// IdentifierBuilder provides a fluent API for constructing Identifier values.
type IdentifierBuilder struct {
	id Identifier
}

// NewIdentifier creates a new IdentifierBuilder.
func NewIdentifier() *IdentifierBuilder {
	return &IdentifierBuilder{}
}

// WithSystem sets the identifier system URI.
func (b *IdentifierBuilder) WithSystem(system string) *IdentifierBuilder {
	s := URI(system)
	b.id.System = &s
	return b
}

// WithValue sets the identifier value.
func (b *IdentifierBuilder) WithValue(value string) *IdentifierBuilder {
	b.id.Value = &value
	return b
}

// WithUse sets the identifier use (e.g. "official", "usual").
func (b *IdentifierBuilder) WithUse(use string) *IdentifierBuilder {
	b.id.Use = &use
	return b
}

// WithType sets the identifier type as a CodeableConcept.
func (b *IdentifierBuilder) WithType(system, code, display string) *IdentifierBuilder {
	b.id.Type = &CodeableConcept{
		Coding: []Coding{NewCoding().WithSystem(system).WithCode(code).WithDisplay(display).Build()},
	}
	return b
}

// Build returns the constructed Identifier.
func (b *IdentifierBuilder) Build() Identifier {
	return b.id
}

// PeriodBuilder provides a fluent API for constructing Period values.
type PeriodBuilder struct {
	p Period
}

// NewPeriod creates a new PeriodBuilder.
func NewPeriod() *PeriodBuilder {
	return &PeriodBuilder{}
}

// WithStart sets the period start date/time.
func (b *PeriodBuilder) WithStart(start string) *PeriodBuilder {
	s := DateTime(start)
	b.p.Start = &s
	return b
}

// WithEnd sets the period end date/time.
func (b *PeriodBuilder) WithEnd(end string) *PeriodBuilder {
	e := DateTime(end)
	b.p.End = &e
	return b
}

// Build returns the constructed Period.
func (b *PeriodBuilder) Build() Period {
	return b.p
}

// QuantityBuilder provides a fluent API for constructing Quantity values.
type QuantityBuilder struct {
	q Quantity
}

// NewQuantity creates a new QuantityBuilder.
func NewQuantity() *QuantityBuilder {
	return &QuantityBuilder{}
}

// WithValue sets the numeric value.
func (b *QuantityBuilder) WithValue(value float64) *QuantityBuilder {
	d := NewDecimal(value)
	b.q.Value = &d
	return b
}

// WithUnit sets the human-readable unit.
func (b *QuantityBuilder) WithUnit(unit string) *QuantityBuilder {
	b.q.Unit = &unit
	return b
}

// WithSystem sets the coding system for the unit.
func (b *QuantityBuilder) WithSystem(system string) *QuantityBuilder {
	s := URI(system)
	b.q.System = &s
	return b
}

// WithCode sets the coded form of the unit.
func (b *QuantityBuilder) WithCode(code string) *QuantityBuilder {
	c := Code(code)
	b.q.Code = &c
	return b
}

// Build returns the constructed Quantity.
func (b *QuantityBuilder) Build() Quantity {
	return b.q
}

// MetaBuilder provides a fluent API for constructing Meta values.
type MetaBuilder struct {
	m Meta
}

// NewMeta creates a new MetaBuilder.
func NewMeta() *MetaBuilder {
	return &MetaBuilder{}
}

// WithVersionId sets the version identifier.
func (b *MetaBuilder) WithVersionId(versionId string) *MetaBuilder {
	id := ID(versionId)
	b.m.VersionId = &id
	return b
}

// WithLastUpdated sets the last updated timestamp.
func (b *MetaBuilder) WithLastUpdated(lastUpdated string) *MetaBuilder {
	i := Instant(lastUpdated)
	b.m.LastUpdated = &i
	return b
}

// WithProfile adds a profile URI.
func (b *MetaBuilder) WithProfile(profile string) *MetaBuilder {
	c := Canonical(profile)
	b.m.Profile = append(b.m.Profile, c)
	return b
}

// WithTag adds a tag coding.
func (b *MetaBuilder) WithTag(system, code string) *MetaBuilder {
	b.m.Tag = append(b.m.Tag, NewCoding().WithSystem(system).WithCode(code).Build())
	return b
}

// Build returns the constructed Meta.
func (b *MetaBuilder) Build() Meta {
	return b.m
}

// AnnotationBuilder provides a fluent API for constructing Annotation values.
type AnnotationBuilder struct {
	a Annotation
}

// NewAnnotation creates a new AnnotationBuilder.
func NewAnnotation() *AnnotationBuilder {
	return &AnnotationBuilder{}
}

// WithText sets the annotation text.
func (b *AnnotationBuilder) WithText(text string) *AnnotationBuilder {
	md := Markdown(text)
	b.a.Text = &md
	return b
}

// WithAuthorReference sets the author as a reference.
func (b *AnnotationBuilder) WithAuthorReference(reference string) *AnnotationBuilder {
	b.a.AuthorReference = &Reference{Reference: &reference}
	return b
}

// WithTime sets the time the annotation was made.
func (b *AnnotationBuilder) WithTime(time string) *AnnotationBuilder {
	dt := DateTime(time)
	b.a.Time = &dt
	return b
}

// Build returns the constructed Annotation.
func (b *AnnotationBuilder) Build() Annotation {
	return b.a
}
