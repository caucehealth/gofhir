// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

// Package client provides a FHIR R4 REST client with a fluent API.
//
// Usage:
//
//	c := client.New("https://hapi.fhir.org/baseR4")
//	patient, err := client.Read[resources.Patient](ctx, c, "123")
//	results, err := c.Search(ctx, "Patient").
//	    Where("family", "Smith").
//	    Where("birthdate", "gt2000-01-01").
//	    Count(10).
//	    Execute()
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/caucehealth/gofhir/r4/bundle"
	"github.com/caucehealth/gofhir/r4/resources"
)


// Client is a FHIR R4 REST client.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets a custom http.Client (for timeouts, TLS, proxies).
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// New creates a FHIR client for the given base URL.
func New(baseURL string, opts ...Option) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	c := &Client{
		baseURL:    baseURL,
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// BaseURL returns the server base URL.
func (c *Client) BaseURL() string { return c.baseURL }

// --- CRUD Operations ---

// Read fetches a resource by type and ID and returns it as the Resource interface.
func Read(ctx context.Context, c *Client, resourceType, id string) (resources.Resource, error) {
	reqURL := fmt.Sprintf("%s/%s/%s", c.baseURL, resourceType, id)

	data, err := c.doGet(ctx, reqURL)
	if err != nil {
		return nil, err
	}

	return resources.ParseResource(data)
}

// ReadAs fetches a resource and unmarshals it into the given type.
func ReadAs[T any](ctx context.Context, c *Client, resourceType, id string) (*T, error) {
	reqURL := fmt.Sprintf("%s/%s/%s", c.baseURL, resourceType, id)

	data, err := c.doGet(ctx, reqURL)
	if err != nil {
		return nil, err
	}

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", resourceType, err)
	}
	return &result, nil
}

// Create sends a new resource to the server (POST).
// Returns the server-assigned ID and the response body.
func Create(ctx context.Context, c *Client, resource resources.Resource) (*MethodOutcome, error) {
	rt := resource.GetResourceType()
	url := fmt.Sprintf("%s/%s", c.baseURL, rt)

	data, err := json.Marshal(resource)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, "POST", url, data)
	if err != nil {
		return nil, err
	}

	return parseMethodOutcome(resp)
}

// Update sends an existing resource to the server (PUT).
func Update(ctx context.Context, c *Client, resource resources.Resource) (*MethodOutcome, error) {
	rt := resource.GetResourceType()
	id := string(resource.GetId())
	if id == "" {
		return nil, fmt.Errorf("resource %s has no id for update", rt)
	}
	url := fmt.Sprintf("%s/%s/%s", c.baseURL, rt, id)

	data, err := json.Marshal(resource)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, "PUT", url, data)
	if err != nil {
		return nil, err
	}

	return parseMethodOutcome(resp)
}

// Delete removes a resource by type and ID.
func Delete(ctx context.Context, c *Client, resourceType, id string) error {
	reqURL := fmt.Sprintf("%s/%s/%s", c.baseURL, resourceType, id)
	_, err := c.doRequest(ctx, "DELETE", reqURL, nil)
	return err
}

// --- Conditional CRUD ---

// CreateConditional creates a resource only if no match exists (If-None-Exist).
func CreateConditional(ctx context.Context, c *Client, resource resources.Resource, ifNoneExist string) (*MethodOutcome, error) {
	rt := resource.GetResourceType()
	reqURL := fmt.Sprintf("%s/%s", c.baseURL, rt)
	data, err := json.Marshal(resource)
	if err != nil {
		return nil, err
	}
	resp, err := c.doRequestWithHeaders(ctx, "POST", reqURL, data, map[string]string{
		"If-None-Exist": ifNoneExist,
	})
	if err != nil {
		return nil, err
	}
	return parseMethodOutcome(resp)
}

// UpdateConditional updates a resource with optimistic locking (If-Match).
func UpdateConditional(ctx context.Context, c *Client, resource resources.Resource, ifMatch string) (*MethodOutcome, error) {
	rt := resource.GetResourceType()
	id := string(resource.GetId())
	if id == "" {
		return nil, fmt.Errorf("resource %s has no id for update", rt)
	}
	reqURL := fmt.Sprintf("%s/%s/%s", c.baseURL, rt, id)
	data, err := json.Marshal(resource)
	if err != nil {
		return nil, err
	}
	resp, err := c.doRequestWithHeaders(ctx, "PUT", reqURL, data, map[string]string{
		"If-Match": ifMatch,
	})
	if err != nil {
		return nil, err
	}
	return parseMethodOutcome(resp)
}

// --- Operations ---

// Operation invokes a FHIR operation (e.g., $validate, $everything, $expand).
// For type-level operations: Operation(ctx, c, "Patient", "$everything", params)
// For instance-level: Operation(ctx, c, "Patient/123", "$everything", params)
func Operation(ctx context.Context, c *Client, target, operation string, params url.Values) (json.RawMessage, error) {
	reqURL := fmt.Sprintf("%s/%s/%s", c.baseURL, target, operation)
	if params != nil && len(params) > 0 {
		reqURL += "?" + params.Encode()
	}
	return c.doGet(ctx, reqURL)
}

// OperationPost invokes a FHIR operation with a POST body.
func OperationPost(ctx context.Context, c *Client, target, operation string, body any) (json.RawMessage, error) {
	reqURL := fmt.Sprintf("%s/%s/%s", c.baseURL, target, operation)
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return c.doRequest(ctx, "POST", reqURL, data)
}

// Transaction sends a Bundle of type transaction to the server.
func Transaction(ctx context.Context, c *Client, b *bundle.Bundle) (*bundle.Bundle, error) {
	data, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, "POST", c.baseURL, data)
	if err != nil {
		return nil, err
	}

	var result bundle.Bundle
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal transaction response: %w", err)
	}
	return &result, nil
}

// --- Search ---

// Search starts a fluent search query for the given resource type.
func (c *Client) Search(ctx context.Context, resourceType string) *SearchBuilder {
	return &SearchBuilder{
		client:       c,
		ctx:          ctx,
		resourceType: resourceType,
		params:       url.Values{},
	}
}

// SearchBuilder constructs a FHIR search query.
type SearchBuilder struct {
	client       *Client
	ctx          context.Context
	resourceType string
	params       url.Values
}

// Where adds a search parameter.
func (s *SearchBuilder) Where(param, value string) *SearchBuilder {
	s.params.Add(param, value)
	return s
}

// Count sets the _count parameter (page size).
func (s *SearchBuilder) Count(n int) *SearchBuilder {
	s.params.Set("_count", fmt.Sprintf("%d", n))
	return s
}

// Sort adds a _sort parameter.
func (s *SearchBuilder) Sort(field string) *SearchBuilder {
	s.params.Add("_sort", field)
	return s
}

// SortDesc adds a descending _sort parameter.
func (s *SearchBuilder) SortDesc(field string) *SearchBuilder {
	s.params.Add("_sort", "-"+field)
	return s
}

// Include adds an _include parameter.
func (s *SearchBuilder) Include(param string) *SearchBuilder {
	s.params.Add("_include", param)
	return s
}

// RevInclude adds a _revinclude parameter.
func (s *SearchBuilder) RevInclude(param string) *SearchBuilder {
	s.params.Add("_revinclude", param)
	return s
}

// Execute runs the search and returns the result bundle.
func (s *SearchBuilder) Execute() (*bundle.Bundle, error) {
	searchURL := fmt.Sprintf("%s/%s?%s", s.client.baseURL, s.resourceType, s.params.Encode())

	data, err := s.client.doGet(s.ctx, searchURL)
	if err != nil {
		return nil, err
	}

	var b bundle.Bundle
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, fmt.Errorf("unmarshal search results: %w", err)
	}
	return &b, nil
}

// --- Paging ---

// NextPage fetches the next page of results from a search bundle.
func NextPage(ctx context.Context, c *Client, b *bundle.Bundle) (*bundle.Bundle, error) {
	return loadPage(ctx, c, b, "next")
}

// PreviousPage fetches the previous page of results.
func PreviousPage(ctx context.Context, c *Client, b *bundle.Bundle) (*bundle.Bundle, error) {
	return loadPage(ctx, c, b, "previous")
}

func loadPage(ctx context.Context, c *Client, b *bundle.Bundle, relation string) (*bundle.Bundle, error) {
	for _, link := range b.Link {
		if link.Relation == relation {
			data, err := c.doGet(ctx, string(link.URL))
			if err != nil {
				return nil, err
			}
			var result bundle.Bundle
			if err := json.Unmarshal(data, &result); err != nil {
				return nil, err
			}
			return &result, nil
		}
	}
	return nil, fmt.Errorf("no %q link in bundle", relation)
}

// --- MethodOutcome ---

// MethodOutcome contains the result of a create/update/delete operation.
type MethodOutcome struct {
	// ID is the server-assigned resource ID.
	ID string
	// StatusCode is the HTTP response status.
	StatusCode int
	// Location is the Location header (for creates).
	Location string
	// Resource is the response body, if any.
	Resource json.RawMessage
}

func parseMethodOutcome(data []byte) (*MethodOutcome, error) {
	// For now, just return the raw body
	return &MethodOutcome{Resource: data}, nil
}

// --- HTTP helpers ---

// Get performs a raw GET request to the given URL. Used by terminology
// and other services that need direct HTTP access.
func (c *Client) Get(ctx context.Context, url string) ([]byte, error) {
	return c.doGet(ctx, url)
}

func (c *Client) doGet(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/fhir+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, &ServerError{StatusCode: resp.StatusCode, Body: body}
	}
	return body, nil
}

func (c *Client) doRequestWithHeaders(ctx context.Context, method, reqURL string, body []byte, headers map[string]string) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/fhir+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/fhir+json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, reqURL, err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, &ServerError{StatusCode: resp.StatusCode, Body: respBody}
	}
	return respBody, nil
}

func (c *Client) doRequest(ctx context.Context, method, reqURL string, body []byte) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/fhir+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/fhir+json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, reqURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, &ServerError{StatusCode: resp.StatusCode, Body: respBody}
	}
	return respBody, nil
}

// ServerError represents an HTTP error from the FHIR server.
type ServerError struct {
	StatusCode int
	Body       []byte
}

func (e *ServerError) Error() string {
	return fmt.Sprintf("FHIR server error %d: %s", e.StatusCode, truncate(string(e.Body), 200))
}

// OperationOutcome attempts to parse the error body as a FHIR OperationOutcome.
func (e *ServerError) OperationOutcome() *resources.OperationOutcome {
	var oo resources.OperationOutcome
	if err := json.Unmarshal(e.Body, &oo); err != nil || oo.ResourceType != "OperationOutcome" {
		return nil
	}
	return &oo
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

