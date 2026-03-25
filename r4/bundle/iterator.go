// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"encoding/json"
	"fmt"
	"io"
)

// EntryIterator provides streaming access to Bundle entries without loading
// the entire bundle into memory. This is critical for processing large
// search result bundles with thousands of entries.
type EntryIterator struct {
	decoder *json.Decoder
	header  IteratorHeader
	started bool
	inArray bool
	done    bool
}

// IteratorHeader contains the Bundle-level fields parsed before streaming entries.
type IteratorHeader struct {
	ResourceType string     `json:"resourceType"`
	Type         BundleType `json:"type"`
	Total        *uint32    `json:"total,omitempty"`
	Link         []BundleLink `json:"link,omitempty"`
}

// NewEntryIterator creates a streaming iterator over Bundle entries from a reader.
// It reads the Bundle header (type, total, links) eagerly, then yields entries
// one at a time via Next().
func NewEntryIterator(r io.Reader) (*EntryIterator, error) {
	decoder := json.NewDecoder(r)
	decoder.UseNumber()

	// Read opening '{'
	tok, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("bundle iterator: %w", err)
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return nil, fmt.Errorf("bundle iterator: expected '{', got %v", tok)
	}

	it := &EntryIterator{decoder: decoder}

	// Read top-level fields until we find "entry"
	for decoder.More() {
		tok, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("bundle iterator: %w", err)
		}
		key, ok := tok.(string)
		if !ok {
			continue
		}

		switch key {
		case "resourceType":
			var v string
			if err := decoder.Decode(&v); err != nil {
				return nil, err
			}
			it.header.ResourceType = v
		case "type":
			var v string
			if err := decoder.Decode(&v); err != nil {
				return nil, err
			}
			it.header.Type = BundleType(v)
		case "total":
			var v uint32
			if err := decoder.Decode(&v); err != nil {
				return nil, err
			}
			it.header.Total = &v
		case "link":
			if err := decoder.Decode(&it.header.Link); err != nil {
				return nil, err
			}
		case "entry":
			// Start streaming entries
			tok, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			if delim, ok := tok.(json.Delim); !ok || delim != '[' {
				return nil, fmt.Errorf("bundle iterator: expected '[' for entry array, got %v", tok)
			}
			it.inArray = true
			return it, nil
		default:
			// Skip unknown fields
			var skip json.RawMessage
			if err := decoder.Decode(&skip); err != nil {
				return nil, err
			}
		}
	}

	// No entry array found
	it.done = true
	return it, nil
}

// Header returns the Bundle-level metadata (type, total, links).
func (it *EntryIterator) Header() IteratorHeader {
	return it.header
}

// Next returns the next BundleEntry. Returns io.EOF when no more entries.
func (it *EntryIterator) Next() (*BundleEntry, error) {
	if it.done {
		return nil, io.EOF
	}
	if !it.inArray {
		return nil, io.EOF
	}

	if !it.decoder.More() {
		it.done = true
		return nil, io.EOF
	}

	var entry BundleEntry
	if err := it.decoder.Decode(&entry); err != nil {
		return nil, fmt.Errorf("bundle iterator: decode entry: %w", err)
	}

	return &entry, nil
}
