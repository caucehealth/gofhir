// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package validate

import "github.com/caucehealth/gofhir/r4/fhirpath"

func init() {
	// Wire FHIRPath engine into validation invariant evaluation.
	fhirpathEval = func(resource any, expr string) (bool, error) {
		return fhirpath.EvaluateBool(resource, expr)
	}
}
