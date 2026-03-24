// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"encoding/json"

	dt "github.com/caucehealth/gofhir/r4/datatypes"
)

// BundleType represents the type of a FHIR Bundle.
type BundleType string

const (
	// TypeDocument represents a FHIR document bundle.
	TypeDocument BundleType = "document"
	// TypeMessage represents a FHIR message bundle.
	TypeMessage BundleType = "message"
	// TypeTransaction represents a FHIR transaction bundle.
	TypeTransaction BundleType = "transaction"
	// TypeTransactionResponse represents a FHIR transaction response bundle.
	TypeTransactionResponse BundleType = "transaction-response"
	// TypeBatch represents a FHIR batch bundle.
	TypeBatch BundleType = "batch"
	// TypeBatchResponse represents a FHIR batch response bundle.
	TypeBatchResponse BundleType = "batch-response"
	// TypeHistory represents a FHIR history bundle.
	TypeHistory BundleType = "history"
	// TypeSearchset represents a FHIR search result bundle.
	TypeSearchset BundleType = "searchset"
	// TypeCollection represents a FHIR collection bundle.
	TypeCollection BundleType = "collection"
)

// Bundle is a FHIR Bundle resource containing a collection of resources.
type Bundle struct {
	// ResourceType is always "Bundle".
	ResourceType string `json:"resourceType"`
	// ID is the logical id of the bundle.
	ID *dt.ID `json:"id,omitempty"`
	// Type indicates the purpose of the bundle.
	Type BundleType `json:"type"`
	// Total is the total number of matches if this is a search result bundle.
	Total *uint32 `json:"total,omitempty"`
	// Entry is the list of entries in the bundle.
	Entry []BundleEntry `json:"entry,omitempty"`
	// Extension contains additional information not part of the basic definition.
	Extension []dt.Extension `json:"extension,omitempty"`
}

// BundleEntry represents a single entry in a FHIR Bundle.
type BundleEntry struct {
	// FullURL is the URI for the entry resource.
	FullURL *dt.URL `json:"fullUrl,omitempty"`
	// Resource contains the entry resource as raw JSON.
	Resource json.RawMessage `json:"resource,omitempty"`
	// Search contains search metadata for this entry.
	Search *BundleSearch `json:"search,omitempty"`
}

// BundleSearch contains search metadata for a Bundle entry.
type BundleSearch struct {
	// Mode indicates whether this entry is in the result set because it matched
	// the search criteria or for some other reason.
	Mode *string `json:"mode,omitempty"`
	// Score is the search ranking score (between 0 and 1).
	Score *float64 `json:"score,omitempty"`
}

// MarshalJSON implements the json.Marshaler interface for Bundle.
func (b Bundle) MarshalJSON() ([]byte, error) {
	b.ResourceType = "Bundle"
	type Alias Bundle
	return json.Marshal((Alias)(b))
}

// UnmarshalJSON implements the json.Unmarshaler interface for Bundle.
func (b *Bundle) UnmarshalJSON(data []byte) error {
	type Alias Bundle
	var alias Alias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	*b = Bundle(alias)
	return nil
}

// BundleBuilder provides a fluent API for constructing Bundle resources.
type BundleBuilder struct {
	bundle Bundle
}

// New creates a new BundleBuilder with the given type.
func New(t BundleType) *BundleBuilder {
	return &BundleBuilder{bundle: Bundle{ResourceType: "Bundle", Type: t}}
}

// WithID sets the bundle ID.
func (b *BundleBuilder) WithID(id string) *BundleBuilder {
	v := dt.ID(id)
	b.bundle.ID = &v
	return b
}

// WithTotal sets the total count.
func (b *BundleBuilder) WithTotal(total uint32) *BundleBuilder {
	b.bundle.Total = &total
	return b
}

// WithEntry adds a resource entry to the bundle. The resource is marshaled
// to JSON and stored as a raw message.
func (b *BundleBuilder) WithEntry(resource any) *BundleBuilder {
	data, err := json.Marshal(resource)
	if err != nil {
		// Store nil resource on marshal error — caller can check the entry
		b.bundle.Entry = append(b.bundle.Entry, BundleEntry{})
		return b
	}
	b.bundle.Entry = append(b.bundle.Entry, BundleEntry{
		Resource: data,
	})
	return b
}

// WithRawEntry adds a pre-marshaled JSON resource entry to the bundle.
func (b *BundleBuilder) WithRawEntry(raw json.RawMessage) *BundleBuilder {
	b.bundle.Entry = append(b.bundle.Entry, BundleEntry{
		Resource: raw,
	})
	return b
}

// Build returns the final Bundle.
func (b *BundleBuilder) Build() *Bundle {
	result := b.bundle
	return &result
}
