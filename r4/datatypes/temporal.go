// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package datatypes

import (
	"strings"
	"time"
)

// DatePrecision represents the precision level of a FHIR date/dateTime value.
type DatePrecision int

const (
	PrecisionYear   DatePrecision = iota // YYYY
	PrecisionMonth                       // YYYY-MM
	PrecisionDay                         // YYYY-MM-DD
	PrecisionSecond                      // YYYY-MM-DDThh:mm:ss
	PrecisionMilli                       // YYYY-MM-DDThh:mm:ss.sss
)

// String returns a human-readable name for the precision level.
func (p DatePrecision) String() string {
	switch p {
	case PrecisionYear:
		return "year"
	case PrecisionMonth:
		return "month"
	case PrecisionDay:
		return "day"
	case PrecisionSecond:
		return "second"
	case PrecisionMilli:
		return "millisecond"
	default:
		return "unknown"
	}
}

// Precision returns the precision level of this Date value.
func (d Date) Precision() DatePrecision {
	return detectPrecision(string(d))
}

// Time parses the Date into a time.Time at the detected precision.
func (d Date) Time() (time.Time, error) {
	return parseTemporalValue(string(d))
}

// Precision returns the precision level of this DateTime value.
func (d DateTime) Precision() DatePrecision {
	return detectPrecision(string(d))
}

// Time parses the DateTime into a time.Time at the detected precision.
func (d DateTime) Time() (time.Time, error) {
	return parseTemporalValue(string(d))
}

// detectPrecision determines the precision of a FHIR date/dateTime string.
func detectPrecision(s string) DatePrecision {
	// Remove timezone suffix for length analysis
	base := s
	if idx := strings.IndexAny(s, "Z+-"); idx > 10 {
		base = s[:idx]
	} else if strings.HasSuffix(s, "Z") {
		base = s[:len(s)-1]
	}

	switch {
	case strings.Contains(base, "."):
		return PrecisionMilli
	case strings.Contains(base, "T"):
		return PrecisionSecond
	case strings.Count(base, "-") >= 2:
		return PrecisionDay
	case strings.Count(base, "-") == 1:
		return PrecisionMonth
	default:
		return PrecisionYear
	}
}

// parseTemporalValue attempts to parse a FHIR date/dateTime string.
func parseTemporalValue(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02T15:04:05.999Z07:00",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02",
		"2006-01",
		"2006",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, &time.ParseError{Value: s, Message: "unrecognized FHIR date format"}
}
