// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package datatypes

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Decimal represents a FHIR decimal value with precision preservation.
// FHIR requires that decimal precision is maintained exactly: 1.0 and 1.00
// are semantically different values. This type stores the exact string
// representation and marshals to/from JSON as a bare number.
type Decimal string

// NewDecimal creates a Decimal from a float64 value.
func NewDecimal(f float64) Decimal {
	return Decimal(strconv.FormatFloat(f, 'f', -1, 64))
}

// NewDecimalFromInt creates a Decimal from an integer value.
func NewDecimalFromInt(i int) Decimal {
	return Decimal(strconv.Itoa(i))
}

// Float64 returns the decimal value as a float64.
// Precision may be lost in the conversion.
func (d Decimal) Float64() float64 {
	f, _ := strconv.ParseFloat(string(d), 64)
	return f
}

// String returns the exact string representation of the decimal.
func (d Decimal) String() string {
	return string(d)
}

// Equal returns true if two decimals represent the same numeric value,
// regardless of trailing zeros. Uses string normalization to avoid
// float64 precision loss.
func (d Decimal) Equal(other Decimal) bool {
	return normalizeDecimal(string(d)) == normalizeDecimal(string(other))
}

// normalizeDecimal removes trailing zeros for comparison.
// "1.00" → "1", "1.10" → "1.1", "100" → "100"
func normalizeDecimal(s string) string {
	if s == "" {
		return "0"
	}
	// Only normalize if there's a decimal point
	dot := -1
	for i, c := range s {
		if c == '.' {
			dot = i
			break
		}
	}
	if dot < 0 {
		return s
	}
	// Trim trailing zeros after decimal point
	end := len(s)
	for end > dot+1 && s[end-1] == '0' {
		end--
	}
	// If only the dot remains, remove it too
	if end == dot+1 {
		end = dot
	}
	return s[:end]
}

// MarshalJSON writes the decimal as a bare JSON number (no quotes).
func (d Decimal) MarshalJSON() ([]byte, error) {
	s := string(d)
	if s == "" {
		return []byte("null"), nil
	}
	// Validate the string is a valid finite number
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid decimal value: %q", s)
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return nil, fmt.Errorf("invalid decimal value: %q (must be finite)", s)
	}
	return []byte(s), nil
}

// UnmarshalJSON reads a JSON number and preserves its exact representation.
func (d *Decimal) UnmarshalJSON(data []byte) error {
	s := strings.TrimSpace(string(data))
	if s == "null" {
		return nil
	}
	// Remove quotes if present (some implementations quote decimals)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	// Validate it's a valid finite number (reject NaN, Infinity)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("invalid decimal value: %q", s)
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return fmt.Errorf("invalid decimal value: %q (must be finite)", s)
	}
	*d = Decimal(s)
	return nil
}
