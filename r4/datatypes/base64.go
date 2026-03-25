// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package datatypes

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

// Base64Binary represents a FHIR base64Binary value. Unlike Go's default
// []byte JSON handling, this type accepts base64 with whitespace (spaces,
// newlines) which is valid per the FHIR specification.
type Base64Binary []byte

// MarshalJSON encodes the binary data as a JSON base64 string.
func (b Base64Binary) MarshalJSON() ([]byte, error) {
	if b == nil {
		return []byte("null"), nil
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(b))
	return []byte(`"` + encoded + `"`), nil
}

// UnmarshalJSON decodes a JSON base64 string, tolerating whitespace.
func (b *Base64Binary) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*b = nil
		return nil
	}
	// Use Go's JSON string decoding to handle unicode escapes (\u003d etc.)
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	// Strip all whitespace (FHIR allows line breaks in base64)
	s = strings.Map(func(r rune) rune {
		if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
			return -1
		}
		return r
	}, s)

	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		// Try with padding
		decoded, err = base64.RawStdEncoding.DecodeString(s)
		if err != nil {
			return err
		}
	}
	*b = decoded
	return nil
}
