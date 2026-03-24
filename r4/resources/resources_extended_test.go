// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package resources_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/caucehealth/gofhir/r4/resources"
)

// ============================================================================
// Inline HL7 example JSON
// ============================================================================

const organizationExampleJSON = `{"resourceType":"Organization","id":"hl7","name":"Health Level Seven International","alias":["HL7 International"],"telecom":[{"system":"phone","value":"(+1) 734-677-7777"},{"system":"fax","value":"(+1) 734-677-6622"},{"system":"email","value":"hq@HL7.org"}],"address":[{"line":["3300 Washtenaw Avenue, Suite 227"],"city":"Ann Arbor","state":"MI","postalCode":"48104","country":"USA"}],"endpoint":[{"reference":"Endpoint/example"}]}`

const locationExampleJSON = `{"resourceType":"Location","id":"1","status":"active","name":"South Wing, second floor","alias":["MC, SW, F2","South Wing 2nd floor"],"description":"Second floor of the Old South Wing","mode":"instance","telecom":[{"system":"phone","value":"2328","use":"work"}],"address":{"use":"work","line":["Galapagosweg 91, Building A"],"city":"Den Burg","postalCode":"9105 PZ","country":"NLD"},"position":{"longitude":-83.6945691,"latitude":42.25475478,"altitude":0},"managingOrganization":{"reference":"Organization/f001"}}`

// ============================================================================
// Organization tests
// ============================================================================

func TestOrganizationHL7ExampleRoundTrip(t *testing.T) {
	data := []byte(organizationExampleJSON)
	var org resources.Organization
	assertJSONRoundTrip(t, data, org)

	if err := json.Unmarshal(data, &org); err != nil {
		t.Fatal(err)
	}
	if org.GetId() != "hl7" {
		t.Error("id should be 'hl7'")
	}
	if org.GetName() != "Health Level Seven International" {
		t.Error("name mismatch")
	}
	if len(org.Alias) != 1 {
		t.Error("should have 1 alias")
	}
	if len(org.Telecom) != 3 {
		t.Errorf("expected 3 telecom, got %d", len(org.Telecom))
	}
	if len(org.Address) != 1 {
		t.Error("should have 1 address")
	}
}

// ============================================================================
// Location tests
// ============================================================================

func TestLocationHL7ExampleRoundTrip(t *testing.T) {
	data := []byte(locationExampleJSON)
	var loc resources.Location
	assertJSONRoundTrip(t, data, loc)

	if err := json.Unmarshal(data, &loc); err != nil {
		t.Fatal(err)
	}
	if loc.GetName() != "South Wing, second floor" {
		t.Error("name mismatch")
	}
	if loc.Status == nil || string(*loc.Status) != "active" {
		t.Error("status should be active")
	}
	if len(loc.Alias) != 2 {
		t.Errorf("expected 2 aliases, got %d", len(loc.Alias))
	}
	if loc.Position == nil || loc.Position.Longitude == nil {
		t.Fatal("position should be present with longitude")
	}
	if *loc.Position.Longitude != -83.6945691 {
		t.Error("longitude mismatch")
	}
	if loc.Position.Altitude == nil || *loc.Position.Altitude != 0 {
		t.Error("altitude 0 should not be omitted")
	}
}

func TestLocationHoursOfOperation(t *testing.T) {
	input := `{"resourceType":"Location","name":"Clinic","status":"active","hoursOfOperation":[{"daysOfWeek":["mon","tue","wed"],"allDay":false,"openingTime":"08:00:00","closingTime":"17:00:00"}]}`
	var loc resources.Location
	if err := json.Unmarshal([]byte(input), &loc); err != nil {
		t.Fatal(err)
	}
	if len(loc.HoursOfOperation) != 1 {
		t.Fatal("should have 1 hoursOfOperation")
	}
	if len(loc.HoursOfOperation[0].DaysOfWeek) != 3 {
		t.Error("should have 3 days")
	}
	if loc.HoursOfOperation[0].AllDay == nil || *loc.HoursOfOperation[0].AllDay != false {
		t.Error("allDay:false should be preserved")
	}

	// Round-trip
	out, _ := json.Marshal(&loc)
	var m map[string]json.RawMessage
	json.Unmarshal(out, &m)
	if _, ok := m["hoursOfOperation"]; !ok {
		t.Error("hoursOfOperation should survive round-trip")
	}
}

// ============================================================================
// Procedure tests (separate performed[x] fields, not polymorphic union)
// ============================================================================

func TestProcedureRoundTrip(t *testing.T) {
	input := `{"resourceType":"Procedure","id":"example","status":"completed","code":{"coding":[{"system":"http://snomed.info/sct","code":"80146002","display":"Appendectomy"}]},"subject":{"reference":"Patient/example"},"performedDateTime":"2013-04-05","performer":[{"actor":{"reference":"Practitioner/example"}}],"reasonCode":[{"text":"Acute appendicitis"}]}`

	var proc resources.Procedure
	if err := json.Unmarshal([]byte(input), &proc); err != nil {
		t.Fatal(err)
	}

	if proc.Status == nil || string(*proc.Status) != "completed" {
		t.Error("status should be completed")
	}
	if proc.PerformedDateTime == nil || *proc.PerformedDateTime != "2013-04-05" {
		t.Error("performedDateTime should be 2013-04-05")
	}
	if len(proc.Performer) != 1 {
		t.Error("should have 1 performer")
	}
	if len(proc.ReasonCode) != 1 {
		t.Error("should have 1 reasonCode")
	}

	out, _ := json.Marshal(&proc)
	var reparsed resources.Procedure
	json.Unmarshal(out, &reparsed)
	if reparsed.PerformedDateTime == nil || *reparsed.PerformedDateTime != "2013-04-05" {
		t.Error("performedDateTime should survive round-trip")
	}
}

func TestProcedurePerformedPeriod(t *testing.T) {
	input := `{"resourceType":"Procedure","status":"completed","subject":{"reference":"Patient/1"},"performedPeriod":{"start":"2024-01-01","end":"2024-01-05"}}`
	var proc resources.Procedure
	if err := json.Unmarshal([]byte(input), &proc); err != nil {
		t.Fatal(err)
	}
	if proc.PerformedPeriod == nil || proc.PerformedPeriod.Start == nil {
		t.Error("performedPeriod.start should be set")
	}

	out, _ := json.Marshal(&proc)
	var m map[string]json.RawMessage
	json.Unmarshal(out, &m)
	if _, ok := m["performedPeriod"]; !ok {
		t.Error("performedPeriod should survive round-trip")
	}
}

// ============================================================================
// Immunization tests (polymorphic occurrence[x], boolean fields)
// ============================================================================

func TestImmunizationRoundTrip(t *testing.T) {
	input := `{"resourceType":"Immunization","id":"example","status":"completed","vaccineCode":{"coding":[{"system":"urn:oid:1.2.36.1.2001.1005.17","code":"FLUVAX"}],"text":"Fluvax (Influenza)"},"patient":{"reference":"Patient/example"},"occurrenceDateTime":"2013-01-10","primarySource":true,"lotNumber":"AAJN11K","isSubpotent":true,"performer":[{"actor":{"reference":"Practitioner/example"}}],"doseQuantity":{"value":5,"system":"http://unitsofmeasure.org","code":"mg"}}`

	var imm resources.Immunization
	if err := json.Unmarshal([]byte(input), &imm); err != nil {
		t.Fatal(err)
	}

	if imm.Status == nil || string(*imm.Status) != "completed" {
		t.Error("status should be completed")
	}
	if imm.Occurrence == nil || imm.Occurrence.DateTime == nil {
		t.Fatal("occurrenceDateTime should be present")
	}
	if *imm.Occurrence.DateTime != "2013-01-10" {
		t.Error("occurrenceDateTime mismatch")
	}
	if imm.PrimarySource == nil || !*imm.PrimarySource {
		t.Error("primarySource should be true")
	}
	if imm.DoseQuantity == nil || imm.DoseQuantity.Value == nil || *imm.DoseQuantity.Value != 5 {
		t.Error("doseQuantity.value should be 5")
	}

	out, _ := json.Marshal(&imm)
	var m map[string]json.RawMessage
	json.Unmarshal(out, &m)
	if _, ok := m["occurrenceDateTime"]; !ok {
		t.Error("occurrenceDateTime should survive round-trip")
	}
}

func TestImmunizationBooleanFalse(t *testing.T) {
	input := `{"resourceType":"Immunization","status":"completed","vaccineCode":{"text":"test"},"patient":{"reference":"Patient/1"},"occurrenceDateTime":"2024-01-01","primarySource":false,"isSubpotent":false}`
	var imm resources.Immunization
	json.Unmarshal([]byte(input), &imm)

	if imm.PrimarySource == nil || *imm.PrimarySource != false {
		t.Error("primarySource should be false")
	}

	out, _ := json.Marshal(&imm)
	var m map[string]json.RawMessage
	json.Unmarshal(out, &m)
	if _, ok := m["primarySource"]; !ok {
		t.Error("primarySource:false must NOT be omitted")
	}
	if _, ok := m["isSubpotent"]; !ok {
		t.Error("isSubpotent:false must NOT be omitted")
	}
}

func TestImmunizationOccurrenceString(t *testing.T) {
	input := `{"resourceType":"Immunization","status":"completed","vaccineCode":{"text":"test"},"patient":{"reference":"Patient/1"},"occurrenceString":"Summer 2023"}`
	var imm resources.Immunization
	json.Unmarshal([]byte(input), &imm)

	if imm.Occurrence == nil || imm.Occurrence.String == nil || *imm.Occurrence.String != "Summer 2023" {
		t.Error("occurrenceString should be parsed")
	}

	out, _ := json.Marshal(&imm)
	var m map[string]json.RawMessage
	json.Unmarshal(out, &m)
	if _, ok := m["occurrenceString"]; !ok {
		t.Error("occurrenceString should survive round-trip")
	}
	if _, ok := m["occurrenceDateTime"]; ok {
		t.Error("occurrenceDateTime should NOT be present")
	}
}

// ============================================================================
// Claim tests (deep nesting, many required fields)
// ============================================================================

func TestClaimRoundTrip(t *testing.T) {
	input := `{"resourceType":"Claim","id":"100150","status":"active","type":{"coding":[{"system":"http://terminology.hl7.org/CodeSystem/claim-type","code":"oral"}]},"use":"claim","patient":{"reference":"Patient/1"},"created":"2014-08-16","insurer":{"reference":"Organization/2"},"provider":{"reference":"Organization/1"},"priority":{"coding":[{"code":"normal"}]},"insurance":[{"sequence":1,"focal":true,"coverage":{"reference":"Coverage/9876B1"}}],"item":[{"sequence":1,"productOrService":{"coding":[{"code":"1200"}]},"servicedDate":"2014-08-16","unitPrice":{"value":135.57,"currency":"USD"},"net":{"value":135.57,"currency":"USD"}}]}`

	var claim resources.Claim
	if err := json.Unmarshal([]byte(input), &claim); err != nil {
		t.Fatal(err)
	}
	if claim.Status == nil || string(*claim.Status) != "active" {
		t.Error("status should be active")
	}
	if claim.Use == nil || string(*claim.Use) != "claim" {
		t.Error("use should be claim")
	}
	if len(claim.Insurance) != 1 {
		t.Fatal("should have 1 insurance")
	}
	if claim.Insurance[0].Focal == nil || !*claim.Insurance[0].Focal {
		t.Error("insurance focal should be true")
	}
	if len(claim.Item) != 1 {
		t.Fatal("should have 1 item")
	}

	out, _ := json.Marshal(&claim)
	var reparsed resources.Claim
	if err := json.Unmarshal(out, &reparsed); err != nil {
		t.Fatal(err)
	}
	if len(reparsed.Item) != 1 || reparsed.Item[0].Sequence == nil {
		t.Error("item should survive round-trip")
	}
}

func TestClaimDeepNesting(t *testing.T) {
	input := `{"resourceType":"Claim","status":"active","type":{"coding":[{"code":"oral"}]},"use":"claim","patient":{"reference":"Patient/1"},"provider":{"reference":"Organization/1"},"priority":{"coding":[{"code":"normal"}]},"insurance":[{"sequence":1,"focal":true,"coverage":{"reference":"Coverage/1"}}],"item":[{"sequence":1,"productOrService":{"coding":[{"code":"exam"}]},"detail":[{"sequence":1,"productOrService":{"coding":[{"code":"detail"}]},"unitPrice":{"value":50.00,"currency":"USD"}}]}]}`

	var claim resources.Claim
	if err := json.Unmarshal([]byte(input), &claim); err != nil {
		t.Fatal(err)
	}
	if len(claim.Item[0].Detail) != 1 {
		t.Fatal("should have 1 detail")
	}

	out, _ := json.Marshal(&claim)
	var reparsed resources.Claim
	json.Unmarshal(out, &reparsed)
	if len(reparsed.Item[0].Detail) != 1 {
		t.Error("detail should survive round-trip")
	}
}

// ============================================================================
// MedicationAdministration polymorphic fields
// ============================================================================

func TestMedicationAdministrationPolymorphic(t *testing.T) {
	input := `{"resourceType":"MedicationAdministration","status":"completed","subject":{"reference":"Patient/1"},"medicationCodeableConcept":{"coding":[{"system":"http://www.nlm.nih.gov/research/umls/rxnorm","code":"1049502"}],"text":"Tylenol"},"effectiveDateTime":"2024-01-15T10:30:00Z"}`

	var ma resources.MedicationAdministration
	if err := json.Unmarshal([]byte(input), &ma); err != nil {
		t.Fatal(err)
	}
	if ma.Medication == nil || ma.Medication.CodeableConcept == nil {
		t.Fatal("medicationCodeableConcept should be parsed")
	}
	if ma.Effective == nil || ma.Effective.DateTime == nil {
		t.Fatal("effectiveDateTime should be parsed")
	}

	out, _ := json.Marshal(&ma)
	var m map[string]json.RawMessage
	json.Unmarshal(out, &m)
	if _, ok := m["medicationCodeableConcept"]; !ok {
		t.Error("medicationCodeableConcept should survive round-trip")
	}
	if _, ok := m["effectiveDateTime"]; !ok {
		t.Error("effectiveDateTime should survive round-trip")
	}
}

// ============================================================================
// ParseResource registry — ALL 145 resource types
// ============================================================================

func TestParseResourceRegistryAllTypes(t *testing.T) {
	allResourceTypes := []string{
		"Account", "ActivityDefinition", "AdverseEvent", "AllergyIntolerance",
		"Appointment", "AppointmentResponse", "AuditEvent", "Basic", "Binary",
		"BiologicallyDerivedProduct", "BodyStructure", "CapabilityStatement",
		"CarePlan", "CareTeam", "CatalogEntry", "ChargeItem",
		"ChargeItemDefinition", "Claim", "ClaimResponse", "ClinicalImpression",
		"CodeSystem", "Communication", "CommunicationRequest",
		"CompartmentDefinition", "Composition", "ConceptMap", "Condition",
		"Consent", "Contract", "Coverage", "CoverageEligibilityRequest",
		"CoverageEligibilityResponse", "DetectedIssue", "Device",
		"DeviceDefinition", "DeviceMetric", "DeviceRequest",
		"DeviceUseStatement", "DiagnosticReport", "DocumentManifest",
		"DocumentReference", "EffectEvidenceSynthesis", "Encounter", "Endpoint",
		"EnrollmentRequest", "EnrollmentResponse", "EpisodeOfCare",
		"EventDefinition", "Evidence", "EvidenceVariable", "ExampleScenario",
		"ExplanationOfBenefit", "FamilyMemberHistory", "Flag", "Goal",
		"GraphDefinition", "Group", "GuidanceResponse", "HealthcareService",
		"ImagingStudy", "Immunization", "ImmunizationEvaluation",
		"ImmunizationRecommendation", "ImplementationGuide", "InsurancePlan",
		"Invoice", "Library", "Linkage", "List", "Location", "Measure",
		"MeasureReport", "Media", "Medication", "MedicationAdministration",
		"MedicationDispense", "MedicationKnowledge", "MedicationRequest",
		"MedicationStatement", "MedicinalProduct",
		"MedicinalProductAuthorization", "MedicinalProductContraindication",
		"MedicinalProductIndication", "MedicinalProductIngredient",
		"MedicinalProductInteraction", "MedicinalProductManufactured",
		"MedicinalProductPackaged", "MedicinalProductPharmaceutical",
		"MedicinalProductUndesirableEffect", "MessageDefinition",
		"MessageHeader", "MolecularSequence", "NamingSystem",
		"NutritionOrder", "Observation", "ObservationDefinition",
		"OperationDefinition", "OperationOutcome", "Organization",
		"OrganizationAffiliation", "Parameters", "Patient", "PaymentNotice",
		"PaymentReconciliation", "Person", "PlanDefinition", "Practitioner",
		"PractitionerRole", "Procedure", "Provenance", "Questionnaire",
		"QuestionnaireResponse", "RelatedPerson", "RequestGroup",
		"ResearchDefinition", "ResearchElementDefinition", "ResearchStudy",
		"ResearchSubject", "RiskAssessment", "RiskEvidenceSynthesis",
		"Schedule", "SearchParameter", "ServiceRequest", "Slot", "Specimen",
		"SpecimenDefinition", "StructureDefinition", "StructureMap",
		"Subscription", "Substance", "SubstanceNucleicAcid",
		"SubstancePolymer", "SubstanceProtein",
		"SubstanceReferenceInformation", "SubstanceSourceMaterial",
		"SubstanceSpecification", "SupplyDelivery", "SupplyRequest", "Task",
		"TerminologyCapabilities", "TestReport", "TestScript", "ValueSet",
		"VerificationResult", "VisionPrescription",
	}

	for _, rt := range allResourceTypes {
		t.Run(rt, func(t *testing.T) {
			input := fmt.Sprintf(`{"resourceType":"%s"}`, rt)
			result, err := resources.ParseResource(json.RawMessage(input))
			if err != nil {
				t.Fatalf("ParseResource(%s): %v", rt, err)
			}
			if result == nil {
				t.Fatalf("ParseResource(%s) returned nil", rt)
			}

			// Marshal back — must not panic
			data, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("Marshal(%s): %v", rt, err)
			}

			var m map[string]json.RawMessage
			json.Unmarshal(data, &m)
			var rtStr string
			json.Unmarshal(m["resourceType"], &rtStr)
			if rtStr != rt {
				t.Errorf("resourceType = %q, want %q", rtStr, rt)
			}
		})
	}
}

func TestParseResourceInvalidJSON(t *testing.T) {
	_, err := resources.ParseResource(json.RawMessage(`{not valid}`))
	if err == nil {
		t.Error("should fail for invalid JSON")
	}
}

func TestParseResourceWithData(t *testing.T) {
	result, err := resources.ParseResource(json.RawMessage(organizationExampleJSON))
	if err != nil {
		t.Fatal(err)
	}
	org, ok := result.(*resources.Organization)
	if !ok {
		t.Fatalf("expected *Organization, got %T", result)
	}
	if org.GetName() != "Health Level Seven International" {
		t.Error("name mismatch")
	}
}

// ============================================================================
// Empty collections omitted
// ============================================================================

func TestEmptyCollectionsOmitted(t *testing.T) {
	loc := resources.Location{ResourceType: "Location"}
	data, _ := json.Marshal(&loc)
	var m map[string]json.RawMessage
	json.Unmarshal(data, &m)

	for _, field := range []string{"telecom", "identifier", "alias", "endpoint", "hoursOfOperation"} {
		if _, ok := m[field]; ok {
			t.Errorf("empty %s should be omitted", field)
		}
	}
}

// ============================================================================
// Nil-safe getters
// ============================================================================

func TestGettersReturnZeroOnNil(t *testing.T) {
	var org resources.Organization
	if org.GetId() != "" {
		t.Error("nil GetId should return empty")
	}
	if org.GetName() != "" {
		t.Error("nil GetName should return empty")
	}
	if org.GetActive() != false {
		t.Error("nil GetActive should return false")
	}

	var loc resources.Location
	if loc.GetName() != "" {
		t.Error("nil GetName should return empty")
	}

	// After population
	json.Unmarshal([]byte(organizationExampleJSON), &org)
	if org.GetName() != "Health Level Seven International" {
		t.Error("GetName mismatch after unmarshal")
	}
}

// ============================================================================
// Unknown fields on additional resources
// ============================================================================

func TestOrganizationUnknownFieldsPreserved(t *testing.T) {
	input := `{"resourceType":"Organization","id":"test","name":"Test","futureField":[1,2,3],"anotherNew":"hello"}`
	var org resources.Organization
	json.Unmarshal([]byte(input), &org)

	if len(org.Extra) != 2 {
		t.Fatalf("expected 2 extra fields, got %d", len(org.Extra))
	}

	out, _ := json.Marshal(&org)
	if !strings.Contains(string(out), "futureField") {
		t.Error("futureField should survive round-trip")
	}
}

