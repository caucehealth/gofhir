// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

// Package synthetic generates random FHIR R4 resources for testing,
// development, and demonstration purposes. These are the core primitives —
// random names, dates, codes, identifiers, and resource builders.
//
// Usage:
//
//	gen := synthetic.New()
//	patient := gen.Patient()
//	observation := gen.Observation(patient)
//	bundle := gen.Bundle(10) // 10 random patients with observations
package synthetic

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	dt "github.com/caucehealth/gofhir/r4/datatypes"
	"github.com/caucehealth/gofhir/r4/resources"
)

// Generator produces random FHIR resources.
type Generator struct {
	rng *rand.Rand
}

// New creates a Generator with a random seed.
func New() *Generator {
	return &Generator{rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

// NewWithSeed creates a Generator with a fixed seed for reproducible output.
func NewWithSeed(seed int64) *Generator {
	return &Generator{rng: rand.New(rand.NewSource(seed))}
}

// --- Primitive generators ---

func (g *Generator) pick(items []string) string {
	return items[g.rng.Intn(len(items))]
}

func (g *Generator) pickN(items []string, n int) []string {
	if n >= len(items) {
		return items
	}
	perm := g.rng.Perm(len(items))
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = items[perm[i]]
	}
	return out
}

func (g *Generator) chance(pct int) bool {
	return g.rng.Intn(100) < pct
}

func (g *Generator) dateInRange(from, to time.Time) time.Time {
	delta := to.Sub(from)
	return from.Add(time.Duration(g.rng.Int63n(int64(delta))))
}

// ID generates a random FHIR-compliant ID.
func (g *Generator) ID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8+g.rng.Intn(8))
	for i := range b {
		b[i] = chars[g.rng.Intn(len(chars))]
	}
	return string(b)
}

// --- Name data ---

var (
	maleFirstNames = []string{
		"James", "John", "Robert", "Michael", "David", "William",
		"Richard", "Joseph", "Thomas", "Christopher", "Daniel", "Matthew",
		"Anthony", "Mark", "Steven", "Andrew", "Joshua", "Kenneth",
		"Kevin", "Brian", "George", "Timothy", "Ronald", "Edward",
	}
	femaleFirstNames = []string{
		"Mary", "Patricia", "Jennifer", "Linda", "Barbara", "Elizabeth",
		"Susan", "Jessica", "Sarah", "Karen", "Lisa", "Nancy",
		"Betty", "Margaret", "Sandra", "Ashley", "Dorothy", "Kimberly",
		"Emily", "Donna", "Michelle", "Carol", "Amanda", "Melissa",
	}
	lastNames = []string{
		"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia",
		"Miller", "Davis", "Rodriguez", "Martinez", "Hernandez", "Lopez",
		"Gonzalez", "Wilson", "Anderson", "Thomas", "Taylor", "Moore",
		"Jackson", "Martin", "Lee", "Perez", "Thompson", "White",
		"Harris", "Sanchez", "Clark", "Ramirez", "Lewis", "Robinson",
	}
	streetNames = []string{
		"Main St", "Oak Ave", "Elm St", "Park Blvd", "Cedar Lane",
		"Maple Dr", "Pine St", "Washington Ave", "Lake Rd", "Hill St",
	}
	cities = []string{
		"New York", "Los Angeles", "Chicago", "Houston", "Phoenix",
		"Philadelphia", "San Antonio", "San Diego", "Dallas", "Austin",
		"Boston", "Seattle", "Denver", "Portland", "Atlanta",
	}
	states = []string{
		"NY", "CA", "IL", "TX", "AZ", "PA", "FL", "OH", "GA", "NC",
		"WA", "CO", "OR", "MA", "MI",
	}
	loincCodes = []struct {
		Code    string
		Display string
		Unit    string
		Low     float64
		High    float64
	}{
		{"8302-2", "Body height", "cm", 140, 200},
		{"29463-7", "Body weight", "kg", 40, 150},
		{"8867-4", "Heart rate", "/min", 50, 120},
		{"8480-6", "Systolic blood pressure", "mmHg", 90, 180},
		{"8462-4", "Diastolic blood pressure", "mmHg", 50, 110},
		{"8310-5", "Body temperature", "Cel", 35.5, 40.0},
		{"2093-3", "Total cholesterol", "mg/dL", 100, 300},
		{"2085-9", "HDL cholesterol", "mg/dL", 20, 100},
		{"2571-8", "Triglycerides", "mg/dL", 50, 400},
		{"4548-4", "Hemoglobin A1c", "%", 4.0, 14.0},
		{"2339-0", "Glucose", "mg/dL", 60, 300},
		{"718-7", "Hemoglobin", "g/dL", 8.0, 18.0},
		{"6690-2", "WBC count", "10*3/uL", 2.0, 15.0},
		{"789-8", "RBC count", "10*6/uL", 3.0, 6.5},
		{"9279-1", "Respiratory rate", "/min", 10, 30},
		{"59408-5", "Oxygen saturation", "%", 85, 100},
	}
	conditionCodes = []struct {
		Code    string
		Display string
		System  string
	}{
		{"44054006", "Type 2 diabetes mellitus", "http://snomed.info/sct"},
		{"38341003", "Hypertensive disorder", "http://snomed.info/sct"},
		{"195967001", "Asthma", "http://snomed.info/sct"},
		{"73211009", "Diabetes mellitus", "http://snomed.info/sct"},
		{"40930008", "Hypothyroidism", "http://snomed.info/sct"},
		{"13645005", "COPD", "http://snomed.info/sct"},
		{"84757009", "Epilepsy", "http://snomed.info/sct"},
		{"49436004", "Atrial fibrillation", "http://snomed.info/sct"},
		{"235595009", "Gastroesophageal reflux", "http://snomed.info/sct"},
		{"396275006", "Osteoarthritis", "http://snomed.info/sct"},
	}
	encounterTypes = []struct {
		Code    string
		Display string
	}{
		{"AMB", "ambulatory"},
		{"EMER", "emergency"},
		{"IMP", "inpatient encounter"},
		{"SS", "short stay"},
		{"VR", "virtual"},
	}
)

// --- Resource generators ---

// Patient generates a random Patient resource.
func (g *Generator) Patient() *resources.Patient {
	id := dt.ID(g.ID())
	isMale := g.chance(50)

	var firstName string
	var genderCode resources.AdministrativeGender
	if isMale {
		firstName = g.pick(maleFirstNames)
		genderCode = resources.AdministrativeGenderMale
	} else {
		firstName = g.pick(femaleFirstNames)
		genderCode = resources.AdministrativeGenderFemale
	}
	lastName := g.pick(lastNames)

	birthDate := dt.Date(g.dateInRange(
		time.Date(1940, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC),
	).Format("2006-01-02"))

	p := &resources.Patient{
		ResourceType: "Patient",
		Id:           &id,
		Gender:       &genderCode,
		BirthDate:    &birthDate,
		Name: []dt.HumanName{{
			Family: &lastName,
			Given:  []string{firstName},
		}},
	}

	// Add address (80% chance)
	if g.chance(80) {
		city := g.pick(cities)
		state := g.pick(states)
		line := fmt.Sprintf("%d %s", 100+g.rng.Intn(9900), g.pick(streetNames))
		zip := fmt.Sprintf("%05d", 10000+g.rng.Intn(89999))
		p.Address = []dt.Address{{
			Line:       []string{line},
			City:       &city,
			State:      &state,
			PostalCode: &zip,
			Country:    strPtr("US"),
		}}
	}

	// Add phone (70% chance)
	if g.chance(70) {
		phone := fmt.Sprintf("(%03d) %03d-%04d", 200+g.rng.Intn(800), g.rng.Intn(1000), g.rng.Intn(10000))
		system := "phone"
		use := "home"
		p.Telecom = append(p.Telecom, dt.ContactPoint{
			System: &system,
			Value:  &phone,
			Use:    &use,
		})
	}

	// Add SSN identifier (60% chance)
	if g.chance(60) {
		ssnSystem := dt.URI("http://hl7.org/fhir/sid/us-ssn")
		ssn := fmt.Sprintf("%03d-%02d-%04d", g.rng.Intn(900)+100, g.rng.Intn(100), g.rng.Intn(10000))
		p.Identifier = append(p.Identifier, dt.Identifier{
			System: &ssnSystem,
			Value:  &ssn,
		})
	}

	// Add MRN identifier
	mrnSystem := dt.URI("http://hospital.example.org/mrn")
	mrn := fmt.Sprintf("MRN-%08d", g.rng.Intn(100000000))
	p.Identifier = append(p.Identifier, dt.Identifier{
		System: &mrnSystem,
		Value:  &mrn,
	})

	return p
}

// Observation generates a random Observation for the given patient.
func (g *Generator) Observation(patientID string) *resources.Observation {
	id := dt.ID(g.ID())
	loinc := loincCodes[g.rng.Intn(len(loincCodes))]

	status := resources.ObservationStatusFinal
	effectiveDate := dt.DateTime(g.dateInRange(
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Now(),
	).Format("2006-01-02T15:04:05Z"))

	value := loinc.Low + g.rng.Float64()*(loinc.High-loinc.Low)
	decVal := dt.Decimal(fmt.Sprintf("%.1f", value))
	unitStr := loinc.Unit
	loincSystem := dt.URI("http://loinc.org")
	ucumSystem := dt.URI("http://unitsofmeasure.org")

	patRef := "Patient/" + patientID

	loincCode := dt.Code(loinc.Code)
	unitCode := dt.Code(loinc.Unit)
	effectiveDateStr := string(effectiveDate)

	obs := &resources.Observation{
		ResourceType: "Observation",
		Id:           &id,
		Status:       &status,
		Code: dt.CodeableConcept{
			Coding: []dt.Coding{{
				System:  &loincSystem,
				Code:    &loincCode,
				Display: strPtr(loinc.Display),
			}},
			Text: strPtr(loinc.Display),
		},
		Subject: &dt.Reference{Reference: &patRef},
		Effective: &resources.ObservationEffective{
			DateTime: &effectiveDateStr,
		},
		Value: &resources.ObservationValue{
			Quantity: &dt.Quantity{
				Value:  &decVal,
				Unit:   &unitStr,
				System: &ucumSystem,
				Code:   &unitCode,
			},
		},
	}

	return obs
}

// Condition generates a random Condition for the given patient.
func (g *Generator) Condition(patientID string) *resources.Condition {
	id := dt.ID(g.ID())
	cond := conditionCodes[g.rng.Intn(len(conditionCodes))]

	snomedSystem := dt.URI(cond.System)
	patRef := "Patient/" + patientID
	onsetDate := dt.DateTime(g.dateInRange(
		time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Now(),
	).Format("2006-01-02"))

	clinicalStatus := dt.URI("http://terminology.hl7.org/CodeSystem/condition-clinical")
	verificationStatus := dt.URI("http://terminology.hl7.org/CodeSystem/condition-ver-status")

	onsetStr := string(onsetDate)

	c := &resources.Condition{
		ResourceType: "Condition",
		Id:           &id,
		ClinicalStatus: &dt.CodeableConcept{
			Coding: []dt.Coding{{
				System:  &clinicalStatus,
				Code:    codePtr("active"),
				Display: strPtr("Active"),
			}},
		},
		VerificationStatus: &dt.CodeableConcept{
			Coding: []dt.Coding{{
				System:  &verificationStatus,
				Code:    codePtr("confirmed"),
				Display: strPtr("Confirmed"),
			}},
		},
		Code: &dt.CodeableConcept{
			Coding: []dt.Coding{{
				System:  &snomedSystem,
				Code:    codePtr(cond.Code),
				Display: strPtr(cond.Display),
			}},
			Text: strPtr(cond.Display),
		},
		Subject: dt.Reference{Reference: &patRef},
		Onset:   &resources.ConditionOnset{DateTime: &onsetStr},
	}

	return c
}

// Encounter generates a random Encounter for the given patient.
func (g *Generator) Encounter(patientID string) *resources.Encounter {
	id := dt.ID(g.ID())
	enc := encounterTypes[g.rng.Intn(len(encounterTypes))]

	status := resources.EncounterStatusFinished
	v3System := dt.URI("http://terminology.hl7.org/CodeSystem/v3-ActCode")
	patRef := "Patient/" + patientID

	start := g.dateInRange(
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Now(),
	)
	end := start.Add(time.Duration(30+g.rng.Intn(480)) * time.Minute)
	startStr := dt.DateTime(start.Format("2006-01-02T15:04:05Z"))
	endStr := dt.DateTime(end.Format("2006-01-02T15:04:05Z"))

	e := &resources.Encounter{
		ResourceType: "Encounter",
		Id:           &id,
		Status:       &status,
		Class: dt.Coding{
			System:  &v3System,
			Code:    codePtr(enc.Code),
			Display: strPtr(enc.Display),
		},
		Subject: &dt.Reference{Reference: &patRef},
		Period: &dt.Period{
			Start: &startStr,
			End:   &endStr,
		},
	}

	return e
}

// PatientBundle generates a bundle of n random patients, each with
// a random number of observations, conditions, and encounters.
func (g *Generator) PatientBundle(n int) []*resources.Patient {
	patients := make([]*resources.Patient, n)
	for i := 0; i < n; i++ {
		patients[i] = g.Patient()
	}
	return patients
}

// PopulatedBundle generates a complete set of resources for n patients.
// Returns patients, observations, conditions, and encounters.
func (g *Generator) PopulatedBundle(nPatients int) (
	patients []*resources.Patient,
	observations []*resources.Observation,
	conditions []*resources.Condition,
	encounters []*resources.Encounter,
) {
	for i := 0; i < nPatients; i++ {
		p := g.Patient()
		patients = append(patients, p)
		pid := string(*p.Id)

		// 2-8 observations per patient
		nObs := 2 + g.rng.Intn(7)
		for j := 0; j < nObs; j++ {
			observations = append(observations, g.Observation(pid))
		}

		// 0-4 conditions
		nCond := g.rng.Intn(5)
		for j := 0; j < nCond; j++ {
			conditions = append(conditions, g.Condition(pid))
		}

		// 1-3 encounters
		nEnc := 1 + g.rng.Intn(3)
		for j := 0; j < nEnc; j++ {
			encounters = append(encounters, g.Encounter(pid))
		}
	}
	return
}

func strPtr(s string) *string     { return &s }
func codePtr(s string) *dt.Code   { v := dt.Code(s); return &v }

// suppress unused import
var _ = strings.TrimSpace
