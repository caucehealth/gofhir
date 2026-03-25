// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package terminology

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/caucehealth/gofhir/r4/client"
)

// Remote delegates terminology operations to a FHIR terminology server
// via the $validate-code, $lookup, and $expand operations.
type Remote struct {
	client *client.Client
	ctx    context.Context
}

// NewRemote creates a remote terminology service using the given FHIR client.
func NewRemote(c *client.Client) *Remote {
	return &Remote{client: c, ctx: context.Background()}
}

// WithContext sets the context for remote operations.
func (r *Remote) WithContext(ctx context.Context) *Remote {
	r.ctx = ctx
	return r
}

func (r *Remote) ValidateCode(params ValidateCodeParams) *ValidateCodeResult {
	q := url.Values{}
	q.Set("system", params.System)
	q.Set("code", params.Code)
	if params.Display != "" {
		q.Set("display", params.Display)
	}
	if params.ValueSetURL != "" {
		q.Set("url", params.ValueSetURL)
	}

	reqURL := fmt.Sprintf("%s/ValueSet/$validate-code?%s", r.client.BaseURL(), q.Encode())
	data, err := doGet(r.ctx, r.client, reqURL)
	if err != nil {
		return &ValidateCodeResult{Valid: false, Message: err.Error()}
	}

	var resp struct {
		Parameter []struct {
			Name         string `json:"name"`
			ValueBoolean *bool  `json:"valueBoolean,omitempty"`
			ValueString  string `json:"valueString,omitempty"`
		} `json:"parameter"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return &ValidateCodeResult{Valid: false, Message: err.Error()}
	}

	result := &ValidateCodeResult{}
	for _, p := range resp.Parameter {
		switch p.Name {
		case "result":
			if p.ValueBoolean != nil {
				result.Valid = *p.ValueBoolean
			}
		case "display":
			result.Display = p.ValueString
		case "message":
			result.Message = p.ValueString
		}
	}
	return result
}

func (r *Remote) LookupCode(system, code string) *LookupResult {
	q := url.Values{}
	q.Set("system", system)
	q.Set("code", code)

	reqURL := fmt.Sprintf("%s/CodeSystem/$lookup?%s", r.client.BaseURL(), q.Encode())
	data, err := doGet(r.ctx, r.client, reqURL)
	if err != nil {
		return &LookupResult{Found: false}
	}

	var resp struct {
		Parameter []struct {
			Name        string `json:"name"`
			ValueString string `json:"valueString,omitempty"`
		} `json:"parameter"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return &LookupResult{Found: false}
	}

	result := &LookupResult{Found: true}
	for _, p := range resp.Parameter {
		switch p.Name {
		case "display":
			result.Display = p.ValueString
		case "definition":
			result.Definition = p.ValueString
		}
	}
	return result
}

func (r *Remote) ExpandValueSet(vsURL string) *ExpansionResult {
	q := url.Values{}
	q.Set("url", vsURL)

	reqURL := fmt.Sprintf("%s/ValueSet/$expand?%s", r.client.BaseURL(), q.Encode())
	data, err := doGet(r.ctx, r.client, reqURL)
	if err != nil {
		return &ExpansionResult{Error: err.Error()}
	}

	var resp struct {
		Expansion struct {
			Total    int `json:"total"`
			Contains []struct {
				System  string `json:"system"`
				Code    string `json:"code"`
				Display string `json:"display"`
			} `json:"contains"`
		} `json:"expansion"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return &ExpansionResult{Error: err.Error()}
	}

	var concepts []Concept
	for _, c := range resp.Expansion.Contains {
		concepts = append(concepts, Concept{
			System:  c.System,
			Code:    c.Code,
			Display: c.Display,
		})
	}
	return &ExpansionResult{Concepts: concepts, Total: resp.Expansion.Total}
}

func (r *Remote) Subsumes(system, codeA, codeB string) SubsumptionResult {
	// Not commonly used, return not-subsumed
	return SubsumesNotSubsumed
}

func doGet(ctx context.Context, c *client.Client, reqURL string) ([]byte, error) {
	return c.Get(ctx, reqURL)
}
