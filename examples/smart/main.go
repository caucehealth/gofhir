// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

// Example smart demonstrates SMART on FHIR discovery and authorization setup.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/caucehealth/gofhir/r4/smart"
)

func main() {
	ctx := context.Background()

	// Discover SMART endpoints from a FHIR server
	// (Using a public test server — may not always be available)
	fmt.Println("Discovering SMART configuration...")
	cfg, err := smart.Discover(ctx, "https://launch.smarthealthit.org/v/r4/fhir")
	if err != nil {
		log.Printf("Discovery failed (expected if server unavailable): %v", err)
		// Demonstrate with manual config
		cfg = &smart.Configuration{
			AuthorizationEndpoint:         "https://launch.smarthealthit.org/v/r4/auth/authorize",
			TokenEndpoint:                 "https://launch.smarthealthit.org/v/r4/auth/token",
			CodeChallengeMethodsSupported: []string{"S256"},
			Capabilities:                  []string{"launch-standalone", "client-public"},
		}
		fmt.Println("Using manual configuration")
	}

	fmt.Printf("  Authorization: %s\n", cfg.AuthorizationEndpoint)
	fmt.Printf("  Token:         %s\n", cfg.TokenEndpoint)
	fmt.Printf("  PKCE:          %v\n", cfg.SupportsPKCE())
	fmt.Printf("  Capabilities:  %v\n", cfg.Capabilities)

	// Set up a standalone launcher
	launcher := smart.NewStandaloneLauncher(cfg, smart.ClientConfig{
		ClientID:    "my-app",
		RedirectURI: "http://localhost:8080/callback",
		Scopes: []string{
			smart.PatientScope("Patient", "read"),
			smart.PatientScope("Observation", "read"),
			"launch/patient",
			"openid",
			"fhirUser",
		},
	})

	// Generate the authorization URL
	authURL := launcher.AuthURL("random-state-value")
	fmt.Printf("\nAuthorization URL:\n  %s\n", authURL)
	fmt.Println("\nRedirect the user to this URL. After they authorize,")
	fmt.Println("exchange the code with launcher.Exchange(ctx, code)")
}
