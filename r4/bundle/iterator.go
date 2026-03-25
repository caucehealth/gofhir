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
	decoder       *json.Decoder
	header        IteratorHeader
	inArray       bool
	done          bool
	trailingRead  bool
}

// IteratorHeader contains the Bundle-level fields parsed before streaming entries.
type IteratorHeader struct {
	ResourceType string       `json:"resourceType"`
	Type         BundleType   `json:"type"`
	Total        *uint32      `json:"total,omitempty"`
	Link         []BundleLink `json:"link,omitempty"`
}

// NewEntryIterator creates a streaming iterator over Bundle entries from a reader.
// It reads the Bundle header (type, total, links) eagerly, then yields entries
// one at a time via Next(). Fields appearing after the entry array are read
// when the iterator is exhausted.
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
	if err := it.readFields(); err != nil {
		return nil, err
	}

	return it, nil
}

// readFields reads top-level Bundle fields, stopping when it finds "entry"
// or runs out of fields.
func (it *EntryIterator) readFields() error {
	for it.decoder.More() {
		tok, err := it.decoder.Token()
		if err != nil {
			return fmt.Errorf("bundle iterator: %w", err)
		}
		key, ok := tok.(string)
		if !ok {
			continue
		}

		switch key {
		case "resourceType":
			var v string
			if err := it.decoder.Decode(&v); err != nil {
				return err
			}
			it.header.ResourceType = v
		case "type":
			var v string
			if err := it.decoder.Decode(&v); err != nil {
				return err
			}
			it.header.Type = BundleType(v)
		case "total":
			var v uint32
			if err := it.decoder.Decode(&v); err != nil {
				return err
			}
			it.header.Total = &v
		case "link":
			if err := it.decoder.Decode(&it.header.Link); err != nil {
				return err
			}
		case "entry":
			tok, err := it.decoder.Token()
			if err != nil {
				return err
			}
			if delim, ok := tok.(json.Delim); !ok || delim != '[' {
				return fmt.Errorf("bundle iterator: expected '[' for entry, got %v", tok)
			}
			it.inArray = true
			return nil
		default:
			var skip json.RawMessage
			if err := it.decoder.Decode(&skip); err != nil {
				return err
			}
		}
	}

	it.done = true
	return nil
}

// Header returns the Bundle-level metadata (type, total, links).
// Note: if fields appear after the entry array in the JSON, they are
// populated after the last call to Next() returns io.EOF.
func (it *EntryIterator) Header() IteratorHeader {
	return it.header
}

// Next returns the next BundleEntry. Returns io.EOF when no more entries.
// After returning io.EOF, any trailing Bundle fields (type, total, link)
// that appeared after the entry array are available via Header().
func (it *EntryIterator) Next() (*BundleEntry, error) {
	if it.done {
		return nil, io.EOF
	}
	if !it.inArray {
		return nil, io.EOF
	}

	if !it.decoder.More() {
		// End of entry array — consume closing ']'
		it.decoder.Token()
		it.done = true

		// Read any trailing fields after the entry array
		if !it.trailingRead {
			it.trailingRead = true
			it.readFields()
		}

		return nil, io.EOF
	}

	var entry BundleEntry
	if err := it.decoder.Decode(&entry); err != nil {
		return nil, fmt.Errorf("bundle iterator: decode entry: %w", err)
	}

	return &entry, nil
}
