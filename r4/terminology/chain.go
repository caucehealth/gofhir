// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package terminology

// Chain delegates to multiple terminology services in order.
// The first service to return a definitive result wins.
type Chain struct {
	services []Service
}

// NewChain creates a chain of terminology services.
// Services are consulted in order; the first definitive answer wins.
func NewChain(services ...Service) *Chain {
	return &Chain{services: services}
}

func (c *Chain) ValidateCode(params ValidateCodeParams) *ValidateCodeResult {
	for _, svc := range c.services {
		result := svc.ValidateCode(params)
		if result.Valid || result.Message == "" {
			return result
		}
	}
	return &ValidateCodeResult{Valid: false, Message: "code not found in any terminology service"}
}

func (c *Chain) LookupCode(system, code string) *LookupResult {
	for _, svc := range c.services {
		result := svc.LookupCode(system, code)
		if result.Found {
			return result
		}
	}
	return &LookupResult{Found: false}
}

func (c *Chain) ExpandValueSet(url string) *ExpansionResult {
	for _, svc := range c.services {
		result := svc.ExpandValueSet(url)
		if result.Error == "" {
			return result
		}
	}
	return &ExpansionResult{Error: "value set not found in any terminology service"}
}

func (c *Chain) Subsumes(system, codeA, codeB string) SubsumptionResult {
	for _, svc := range c.services {
		result := svc.Subsumes(system, codeA, codeB)
		if result != SubsumesNotSubsumed {
			return result
		}
	}
	return SubsumesNotSubsumed
}
