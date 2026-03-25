// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package terminology

// LoadCommonCodeSystems loads well-known FHIR code systems into the service.
// Includes: administrative gender, observation status, encounter status,
// common UCUM units, BCP-47 languages, MIME types, and ISO 3166 countries.
func (m *InMemory) LoadCommonCodeSystems() {
	m.loadAdministrativeGender()
	m.loadObservationStatus()
	m.loadEncounterStatus()
	m.loadConditionClinicalStatus()
	m.loadConditionVerificationStatus()
	m.loadRequestStatus()
	m.loadRequestIntent()
	m.loadBundleType()
	m.loadNarrativeStatus()
	m.loadPublicationStatus()
	m.loadContactPointSystem()
	m.loadContactPointUse()
	m.loadAddressUse()
	m.loadNameUse()
	m.loadIdentifierUse()
	m.loadUCUM()
	m.loadMIMETypes()
	m.loadLanguages()
	m.loadCountries()
}

func (m *InMemory) loadAdministrativeGender() {
	m.AddCodeSystem(&CodeSystem{
		URL:  "http://hl7.org/fhir/administrative-gender",
		Name: "AdministrativeGender",
		Concepts: concepts(
			"male", "Male",
			"female", "Female",
			"other", "Other",
			"unknown", "Unknown",
		),
	})
}

func (m *InMemory) loadObservationStatus() {
	m.AddCodeSystem(&CodeSystem{
		URL:  "http://hl7.org/fhir/observation-status",
		Name: "ObservationStatus",
		Concepts: concepts(
			"registered", "Registered",
			"preliminary", "Preliminary",
			"final", "Final",
			"amended", "Amended",
			"corrected", "Corrected",
			"cancelled", "Cancelled",
			"entered-in-error", "Entered in Error",
			"unknown", "Unknown",
		),
	})
}

func (m *InMemory) loadEncounterStatus() {
	m.AddCodeSystem(&CodeSystem{
		URL:  "http://hl7.org/fhir/encounter-status",
		Name: "EncounterStatus",
		Concepts: concepts(
			"planned", "Planned",
			"arrived", "Arrived",
			"triaged", "Triaged",
			"in-progress", "In Progress",
			"onleave", "On Leave",
			"finished", "Finished",
			"cancelled", "Cancelled",
			"entered-in-error", "Entered in Error",
			"unknown", "Unknown",
		),
	})
}

func (m *InMemory) loadConditionClinicalStatus() {
	m.AddCodeSystem(&CodeSystem{
		URL:  "http://terminology.hl7.org/CodeSystem/condition-clinical",
		Name: "ConditionClinicalStatusCodes",
		Concepts: concepts(
			"active", "Active",
			"recurrence", "Recurrence",
			"relapse", "Relapse",
			"inactive", "Inactive",
			"remission", "Remission",
			"resolved", "Resolved",
		),
	})
}

func (m *InMemory) loadConditionVerificationStatus() {
	m.AddCodeSystem(&CodeSystem{
		URL:  "http://terminology.hl7.org/CodeSystem/condition-ver-status",
		Name: "ConditionVerificationStatus",
		Concepts: concepts(
			"unconfirmed", "Unconfirmed",
			"provisional", "Provisional",
			"differential", "Differential",
			"confirmed", "Confirmed",
			"refuted", "Refuted",
			"entered-in-error", "Entered in Error",
		),
	})
}

func (m *InMemory) loadRequestStatus() {
	m.AddCodeSystem(&CodeSystem{
		URL:  "http://hl7.org/fhir/request-status",
		Name: "RequestStatus",
		Concepts: concepts(
			"draft", "Draft",
			"active", "Active",
			"on-hold", "On Hold",
			"revoked", "Revoked",
			"completed", "Completed",
			"entered-in-error", "Entered in Error",
			"unknown", "Unknown",
		),
	})
}

func (m *InMemory) loadRequestIntent() {
	m.AddCodeSystem(&CodeSystem{
		URL:  "http://hl7.org/fhir/request-intent",
		Name: "RequestIntent",
		Concepts: concepts(
			"proposal", "Proposal",
			"plan", "Plan",
			"directive", "Directive",
			"order", "Order",
			"original-order", "Original Order",
			"reflex-order", "Reflex Order",
			"filler-order", "Filler Order",
			"instance-order", "Instance Order",
			"option", "Option",
		),
	})
}

func (m *InMemory) loadBundleType() {
	m.AddCodeSystem(&CodeSystem{
		URL:  "http://hl7.org/fhir/bundle-type",
		Name: "BundleType",
		Concepts: concepts(
			"document", "Document",
			"message", "Message",
			"transaction", "Transaction",
			"transaction-response", "Transaction Response",
			"batch", "Batch",
			"batch-response", "Batch Response",
			"history", "History",
			"searchset", "Search Results",
			"collection", "Collection",
		),
	})
}

func (m *InMemory) loadNarrativeStatus() {
	m.AddCodeSystem(&CodeSystem{
		URL:  "http://hl7.org/fhir/narrative-status",
		Name: "NarrativeStatus",
		Concepts: concepts(
			"generated", "Generated",
			"extensions", "Extensions",
			"additional", "Additional",
			"empty", "Empty",
		),
	})
}

func (m *InMemory) loadPublicationStatus() {
	m.AddCodeSystem(&CodeSystem{
		URL:  "http://hl7.org/fhir/publication-status",
		Name: "PublicationStatus",
		Concepts: concepts(
			"draft", "Draft",
			"active", "Active",
			"retired", "Retired",
			"unknown", "Unknown",
		),
	})
}

func (m *InMemory) loadContactPointSystem() {
	m.AddCodeSystem(&CodeSystem{
		URL:  "http://hl7.org/fhir/contact-point-system",
		Name: "ContactPointSystem",
		Concepts: concepts(
			"phone", "Phone",
			"fax", "Fax",
			"email", "Email",
			"pager", "Pager",
			"url", "URL",
			"sms", "SMS",
			"other", "Other",
		),
	})
}

func (m *InMemory) loadContactPointUse() {
	m.AddCodeSystem(&CodeSystem{
		URL:  "http://hl7.org/fhir/contact-point-use",
		Name: "ContactPointUse",
		Concepts: concepts(
			"home", "Home",
			"work", "Work",
			"temp", "Temp",
			"old", "Old",
			"mobile", "Mobile",
		),
	})
}

func (m *InMemory) loadAddressUse() {
	m.AddCodeSystem(&CodeSystem{
		URL:  "http://hl7.org/fhir/address-use",
		Name: "AddressUse",
		Concepts: concepts(
			"home", "Home",
			"work", "Work",
			"temp", "Temporary",
			"old", "Old / Incorrect",
			"billing", "Billing",
		),
	})
}

func (m *InMemory) loadNameUse() {
	m.AddCodeSystem(&CodeSystem{
		URL:  "http://hl7.org/fhir/name-use",
		Name: "NameUse",
		Concepts: concepts(
			"usual", "Usual",
			"official", "Official",
			"temp", "Temp",
			"nickname", "Nickname",
			"anonymous", "Anonymous",
			"old", "Old",
			"maiden", "Name changed for Marriage",
		),
	})
}

func (m *InMemory) loadIdentifierUse() {
	m.AddCodeSystem(&CodeSystem{
		URL:  "http://hl7.org/fhir/identifier-use",
		Name: "IdentifierUse",
		Concepts: concepts(
			"usual", "Usual",
			"official", "Official",
			"temp", "Temp",
			"secondary", "Secondary",
			"old", "Old",
		),
	})
}

func (m *InMemory) loadUCUM() {
	m.AddCodeSystem(&CodeSystem{
		URL:  "http://unitsofmeasure.org",
		Name: "UCUM",
		Concepts: concepts(
			"mm[Hg]", "millimeter of mercury",
			"mg", "milligram",
			"g", "gram",
			"kg", "kilogram",
			"mL", "milliliter",
			"L", "liter",
			"cm", "centimeter",
			"m", "meter",
			"km", "kilometer",
			"min", "minute",
			"h", "hour",
			"d", "day",
			"wk", "week",
			"mo", "month",
			"a", "year",
			"/min", "per minute",
			"%", "percent",
			"mmol/L", "millimole per liter",
			"mg/dL", "milligram per deciliter",
			"g/dL", "gram per deciliter",
			"ng/mL", "nanogram per milliliter",
			"U/L", "unit per liter",
			"10*3/uL", "thousand per microliter",
			"10*6/uL", "million per microliter",
			"10*9/L", "billion per liter",
			"fL", "femtoliter",
			"pg", "picogram",
			"Cel", "degree Celsius",
			"[degF]", "degree Fahrenheit",
			"meq/L", "milliequivalent per liter",
			"umol/L", "micromole per liter",
			"kcal", "kilocalorie",
			"[lb_av]", "pound",
			"[in_i]", "inch",
			"[ft_i]", "foot",
			"bpm", "beats per minute",
		),
	})
}

func (m *InMemory) loadMIMETypes() {
	m.AddCodeSystem(&CodeSystem{
		URL:  "urn:ietf:bcp:13",
		Name: "MIMETypes",
		Concepts: concepts(
			"application/json", "JSON",
			"application/xml", "XML",
			"application/fhir+json", "FHIR JSON",
			"application/fhir+xml", "FHIR XML",
			"application/pdf", "PDF",
			"text/plain", "Plain Text",
			"text/html", "HTML",
			"image/jpeg", "JPEG",
			"image/png", "PNG",
		),
	})
}

func (m *InMemory) loadLanguages() {
	m.AddCodeSystem(&CodeSystem{
		URL:  "urn:ietf:bcp:47",
		Name: "Languages",
		Concepts: concepts(
			"en", "English",
			"en-US", "English (US)",
			"en-GB", "English (UK)",
			"es", "Spanish",
			"fr", "French",
			"de", "German",
			"zh", "Chinese",
			"ja", "Japanese",
			"ko", "Korean",
			"pt", "Portuguese",
			"it", "Italian",
			"ru", "Russian",
			"ar", "Arabic",
			"hi", "Hindi",
			"nl", "Dutch",
		),
	})
}

func (m *InMemory) loadCountries() {
	m.AddCodeSystem(&CodeSystem{
		URL:  "urn:iso:std:iso:3166",
		Name: "ISO3166Countries",
		Concepts: concepts(
			"US", "United States",
			"GB", "United Kingdom",
			"CA", "Canada",
			"AU", "Australia",
			"DE", "Germany",
			"FR", "France",
			"ES", "Spain",
			"IT", "Italy",
			"JP", "Japan",
			"CN", "China",
			"IN", "India",
			"BR", "Brazil",
			"MX", "Mexico",
		),
	})
}

// concepts creates a Concept map from alternating code, display pairs.
func concepts(pairs ...string) map[string]*Concept {
	m := make(map[string]*Concept, len(pairs)/2)
	for i := 0; i < len(pairs)-1; i += 2 {
		m[pairs[i]] = &Concept{Code: pairs[i], Display: pairs[i+1]}
	}
	return m
}
