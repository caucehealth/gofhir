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
	// Meta contains resource metadata.
	Meta *dt.Meta `json:"meta,omitempty"`
	// ImplicitRules is a reference to the implementation rules.
	ImplicitRules *dt.URI `json:"implicitRules,omitempty"`
	// Language is the language of the resource content.
	Language *dt.Code `json:"language,omitempty"`
	// Identifier is a persistent identifier for the bundle.
	Identifier *dt.Identifier `json:"identifier,omitempty"`
	// Type indicates the purpose of the bundle.
	Type BundleType `json:"type"`
	// Timestamp is when the bundle was assembled.
	Timestamp *dt.Instant `json:"timestamp,omitempty"`
	// Total is the total number of matches if this is a search result bundle.
	Total *uint32 `json:"total,omitempty"`
	// Link provides navigation links for paginated results.
	Link []BundleLink `json:"link,omitempty"`
	// Entry is the list of entries in the bundle.
	Entry []BundleEntry `json:"entry,omitempty"`
	// Signature is a digital signature for the bundle.
	Signature *dt.Signature `json:"signature,omitempty"`
	// Extension contains additional information not part of the basic definition.
	Extension []dt.Extension `json:"extension,omitempty"`
}

// BundleLink represents a navigation link in a Bundle (e.g. next, previous).
type BundleLink struct {
	// Relation describes the link relationship (e.g. "self", "next", "previous").
	Relation string `json:"relation"`
	// URL is the reference details for the link.
	URL dt.URL `json:"url"`
}

// BundleEntry represents a single entry in a FHIR Bundle.
type BundleEntry struct {
	// Link provides links related to this entry.
	Link []BundleLink `json:"link,omitempty"`
	// FullURL is the URI for the entry resource.
	FullURL *dt.URL `json:"fullUrl,omitempty"`
	// Resource contains the entry resource as raw JSON.
	Resource json.RawMessage `json:"resource,omitempty"`
	// Search contains search metadata for this entry.
	Search *BundleSearch `json:"search,omitempty"`
	// Request contains transaction/batch request details.
	Request *BundleRequest `json:"request,omitempty"`
	// Response contains transaction/batch response details.
	Response *BundleResponse `json:"response,omitempty"`
}

// BundleSearch contains search metadata for a Bundle entry.
type BundleSearch struct {
	// Mode indicates whether this entry is in the result set because it matched
	// the search criteria or for some other reason.
	Mode *string `json:"mode,omitempty"`
	// Score is the search ranking score (between 0 and 1).
	Score *float64 `json:"score,omitempty"`
}

// BundleRequest contains HTTP request details for transaction/batch entries.
type BundleRequest struct {
	// Method is the HTTP method (GET, HEAD, POST, PUT, DELETE, PATCH).
	Method string `json:"method"`
	// URL is the URL for the request relative to the server root.
	URL dt.URL `json:"url"`
	// IfNoneMatch is used for conditional read (ETag).
	IfNoneMatch *string `json:"ifNoneMatch,omitempty"`
	// IfModifiedSince is used for conditional read (Last-Modified).
	IfModifiedSince *dt.Instant `json:"ifModifiedSince,omitempty"`
	// IfMatch is used for conditional updates (ETag).
	IfMatch *string `json:"ifMatch,omitempty"`
	// IfNoneExist is used for conditional creates.
	IfNoneExist *string `json:"ifNoneExist,omitempty"`
}

// BundleResponse contains HTTP response details for transaction/batch responses.
type BundleResponse struct {
	// Status is the HTTP status code (e.g. "200 OK", "201 Created").
	Status string `json:"status"`
	// Location is the location header value (for creates).
	Location *dt.URL `json:"location,omitempty"`
	// Etag is the ETag for the resource version.
	Etag *string `json:"etag,omitempty"`
	// LastModified is the server's date/time modified.
	LastModified *dt.Instant `json:"lastModified,omitempty"`
	// Outcome is an OperationOutcome for this entry (as raw JSON).
	Outcome json.RawMessage `json:"outcome,omitempty"`
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

// WithMeta sets the bundle metadata.
func (b *BundleBuilder) WithMeta(meta dt.Meta) *BundleBuilder {
	b.bundle.Meta = &meta
	return b
}

// WithTotal sets the total count.
func (b *BundleBuilder) WithTotal(total uint32) *BundleBuilder {
	b.bundle.Total = &total
	return b
}

// WithTimestamp sets the bundle timestamp.
func (b *BundleBuilder) WithTimestamp(ts string) *BundleBuilder {
	v := dt.Instant(ts)
	b.bundle.Timestamp = &v
	return b
}

// WithLink adds a navigation link to the bundle.
func (b *BundleBuilder) WithLink(relation, url string) *BundleBuilder {
	b.bundle.Link = append(b.bundle.Link, BundleLink{
		Relation: relation,
		URL:      dt.URL(url),
	})
	return b
}

// WithEntry adds a resource entry to the bundle. The resource is marshaled
// to JSON and stored as a raw message.
func (b *BundleBuilder) WithEntry(resource any) *BundleBuilder {
	data, err := json.Marshal(resource)
	if err != nil {
		b.bundle.Entry = append(b.bundle.Entry, BundleEntry{})
		return b
	}
	b.bundle.Entry = append(b.bundle.Entry, BundleEntry{
		Resource: data,
	})
	return b
}

// WithFullURLEntry adds a resource entry with a full URL.
func (b *BundleBuilder) WithFullURLEntry(fullURL string, resource any) *BundleBuilder {
	data, err := json.Marshal(resource)
	if err != nil {
		b.bundle.Entry = append(b.bundle.Entry, BundleEntry{})
		return b
	}
	u := dt.URL(fullURL)
	b.bundle.Entry = append(b.bundle.Entry, BundleEntry{
		FullURL:  &u,
		Resource: data,
	})
	return b
}

// WithTransactionEntry adds a transaction entry with method and URL.
func (b *BundleBuilder) WithTransactionEntry(method, url string, resource any) *BundleBuilder {
	data, err := json.Marshal(resource)
	if err != nil {
		b.bundle.Entry = append(b.bundle.Entry, BundleEntry{})
		return b
	}
	b.bundle.Entry = append(b.bundle.Entry, BundleEntry{
		Resource: data,
		Request: &BundleRequest{
			Method: method,
			URL:    dt.URL(url),
		},
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
