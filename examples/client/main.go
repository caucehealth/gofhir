// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

// Example client demonstrates the FHIR REST client with search and middleware.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/caucehealth/gofhir/r4/client"
	"github.com/caucehealth/gofhir/r4/resources"
)

func main() {
	// Create a client pointing to a public FHIR server
	c := client.New("https://hapi.fhir.org/baseR4")

	// Add middleware
	c.Wrap(
		client.Retry(3, time.Second),
		client.Logging(func(method, url string, status int, dur time.Duration) {
			fmt.Printf("  %s %s -> %d (%s)\n", method, url, status, dur.Round(time.Millisecond))
		}),
	)

	ctx := context.Background()

	// Read a Patient by ID
	fmt.Println("Reading Patient/592912...")
	p, err := client.ReadAs[resources.Patient](ctx, c, "Patient", "592912")
	if err != nil {
		log.Printf("Read failed (expected if server is down): %v", err)
	} else {
		fmt.Printf("  Got: %s %s\n", p.GetResourceType(), string(p.GetId()))
	}

	// Search for patients
	fmt.Println("\nSearching for patients named 'Smith'...")
	results, err := c.Search(ctx, "Patient").
		Where("family", "Smith").
		Count(5).
		Sort("-_lastUpdated").
		Execute()
	if err != nil {
		log.Printf("Search failed: %v", err)
		return
	}

	fmt.Printf("  Found %d entries\n", len(results.Entry))

	// Page through results
	if len(results.Link) > 0 {
		for _, link := range results.Link {
			fmt.Printf("  Link: %s -> %s\n", link.Relation, link.URL)
		}
	}
}
