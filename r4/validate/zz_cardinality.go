// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package validate

// structDefRequired maps resource types to their required fields (min>=1)
// as defined in FHIR R4 StructureDefinitions. This supplements the JSON schema
// which omits many required field constraints.
var structDefRequired = map[string][]string{
	"Account": {"status"},
	"ActivityDefinition": {"status"},
	"AdverseEvent": {"actuality", "subject"},
	"AllergyIntolerance": {"patient"},
	"Appointment": {"status", "participant"},
	"AppointmentResponse": {"appointment", "participantStatus"},
	"AuditEvent": {"type", "recorded", "agent", "source"},
	"Basic": {"code"},
	"Binary": {"contentType"},
	"BodyStructure": {"patient"},
	"CapabilityStatement": {"status", "date", "kind", "fhirVersion", "format"},
	"CarePlan": {"status", "intent", "subject"},
	"CatalogEntry": {"orderable", "referencedItem"},
	"ChargeItem": {"status", "code", "subject"},
	"ChargeItemDefinition": {"url", "status"},
	"Claim": {"status", "type", "use", "patient", "created", "provider", "priority", "insurance"},
	"ClaimResponse": {"status", "type", "use", "patient", "created", "insurer", "outcome"},
	"ClinicalImpression": {"status", "subject"},
	"CodeSystem": {"status", "content"},
	"Communication": {"status"},
	"CommunicationRequest": {"status"},
	"CompartmentDefinition": {"url", "name", "status", "code", "search"},
	"Composition": {"status", "type", "date", "author", "title"},
	"ConceptMap": {"status"},
	"Condition": {"subject"},
	"Consent": {"status", "scope", "category"},
	"Coverage": {"status", "beneficiary", "payor"},
	"CoverageEligibilityRequest": {"status", "purpose", "patient", "created", "insurer"},
	"CoverageEligibilityResponse": {"status", "purpose", "patient", "created", "request", "outcome", "insurer"},
	"DetectedIssue": {"status"},
	"DeviceMetric": {"type", "category"},
	"DeviceRequest": {"intent", "code", "subject"},
	"DeviceUseStatement": {"status", "subject", "device"},
	"DiagnosticReport": {"status", "code"},
	"DocumentManifest": {"status", "content"},
	"DocumentReference": {"status", "content"},
	"EffectEvidenceSynthesis": {"status", "population", "exposure", "exposureAlternative", "outcome"},
	"Encounter": {"status", "class"},
	"Endpoint": {"status", "connectionType", "payloadType", "address"},
	"EpisodeOfCare": {"status", "patient"},
	"EventDefinition": {"status", "trigger"},
	"Evidence": {"status", "exposureBackground"},
	"EvidenceVariable": {"status", "characteristic"},
	"ExampleScenario": {"status"},
	"ExplanationOfBenefit": {"status", "type", "use", "patient", "created", "insurer", "provider", "outcome", "insurance"},
	"FamilyMemberHistory": {"status", "patient", "relationship"},
	"Flag": {"status", "code", "subject"},
	"Goal": {"lifecycleStatus", "description", "subject"},
	"GraphDefinition": {"name", "status", "start"},
	"Group": {"type", "actual"},
	"GuidanceResponse": {"module", "status"},
	"ImagingStudy": {"status", "subject"},
	"Immunization": {"status", "vaccineCode", "patient", "occurrence"},
	"ImmunizationEvaluation": {"status", "patient", "targetDisease", "immunizationEvent", "doseStatus"},
	"ImmunizationRecommendation": {"patient", "date", "recommendation"},
	"ImplementationGuide": {"url", "name", "status", "packageId", "fhirVersion"},
	"Invoice": {"status"},
	"Library": {"status", "type"},
	"Linkage": {"item"},
	"List": {"status", "mode"},
	"Measure": {"status"},
	"MeasureReport": {"status", "type", "measure", "period"},
	"Media": {"status", "content"},
	"MedicationAdministration": {"status", "medication", "subject", "effective"},
	"MedicationDispense": {"status", "medication"},
	"MedicationRequest": {"status", "intent", "medication", "subject"},
	"MedicationStatement": {"status", "medication", "subject"},
	"MedicinalProduct": {"name"},
	"MedicinalProductIngredient": {"role"},
	"MedicinalProductManufactured": {"manufacturedDoseForm", "quantity"},
	"MedicinalProductPackaged": {"packageItem"},
	"MedicinalProductPharmaceutical": {"administrableDoseForm", "routeOfAdministration"},
	"MessageDefinition": {"status", "date", "event"},
	"MessageHeader": {"event", "source"},
	"MolecularSequence": {"coordinateSystem"},
	"NamingSystem": {"name", "status", "kind", "date", "uniqueId"},
	"NutritionOrder": {"status", "intent", "patient", "dateTime"},
	"Observation": {"status", "code"},
	"ObservationDefinition": {"code"},
	"OperationDefinition": {"name", "status", "kind", "code", "system", "type", "instance"},
	"OperationOutcome": {"issue"},
	"PaymentNotice": {"status", "created", "payment", "recipient", "amount"},
	"PaymentReconciliation": {"status", "created", "paymentDate", "paymentAmount"},
	"PlanDefinition": {"status"},
	"Procedure": {"status", "subject"},
	"Provenance": {"target", "recorded", "agent"},
	"Questionnaire": {"status"},
	"QuestionnaireResponse": {"status"},
	"RelatedPerson": {"patient"},
	"RequestGroup": {"status", "intent"},
	"ResearchDefinition": {"status", "population"},
	"ResearchElementDefinition": {"status", "type", "characteristic"},
	"ResearchStudy": {"status"},
	"ResearchSubject": {"status", "study", "individual"},
	"RiskAssessment": {"status", "subject"},
	"RiskEvidenceSynthesis": {"status", "population", "outcome"},
	"Schedule": {"actor"},
	"SearchParameter": {"url", "name", "status", "description", "code", "base", "type"},
	"ServiceRequest": {"status", "intent", "subject"},
	"Slot": {"schedule", "status", "start", "end"},
	"StructureDefinition": {"url", "name", "status", "kind", "abstract", "type"},
	"StructureMap": {"url", "name", "status", "group"},
	"Subscription": {"status", "reason", "criteria", "channel"},
	"Substance": {"code"},
	"SupplyRequest": {"item", "quantity"},
	"Task": {"status", "intent"},
	"TerminologyCapabilities": {"status", "date", "kind"},
	"TestReport": {"status", "testScript", "result"},
	"TestScript": {"url", "name", "status"},
	"ValueSet": {"status"},
	"VerificationResult": {"status"},
	"VisionPrescription": {"status", "created", "patient", "dateWritten", "prescriber", "lensSpecification"},
}

func init() {
	// Augment generated metadata with StructureDefinition cardinality.
	// The JSON schema misses many required fields; StructureDefs are authoritative.
	for rt, fields := range structDefRequired {
		meta := GetResourceMeta(rt)
		if meta == nil {
			continue
		}
		requiredSet := make(map[string]bool, len(fields))
		for _, f := range fields {
			requiredSet[f] = true
		}
		for i := range meta.Fields {
			if requiredSet[meta.Fields[i].JSONName] {
				meta.Fields[i].Required = true
			}
		}
	}
}
