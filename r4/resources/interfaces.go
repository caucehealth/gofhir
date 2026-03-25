// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"encoding/json"

	dt "github.com/caucehealth/gofhir/r4/datatypes"
)

// Resource is the common interface satisfied by all FHIR resource types.
// Every generated resource struct implements this interface, enabling
// type-safe polymorphic handling without type assertions to any.
type Resource interface {
	GetResourceType() string
	GetId() dt.ID
	GetMeta() dt.Meta
	GetImplicitRules() dt.URI
	GetLanguage() dt.Code
}

// DomainResource is the interface satisfied by FHIR resources that extend
// DomainResource (all resources except Binary and Parameters). It adds
// narrative text, contained resources, extensions, and modifier extensions.
type DomainResource interface {
	Resource
	GetText() dt.Narrative
	GetContained() []json.RawMessage
	GetExtension() []dt.Extension
	GetModifierExtension() []dt.Extension
}
