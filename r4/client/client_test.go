// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package client_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/caucehealth/gofhir/r4/bundle"
	"github.com/caucehealth/gofhir/r4/client"
	dt "github.com/caucehealth/gofhir/r4/datatypes"
	"github.com/caucehealth/gofhir/r4/resources"
)

func TestReadResource(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Patient/123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if !strings.Contains(r.Header.Get("Accept"), "fhir+json") {
			t.Error("should send fhir+json Accept header")
		}
		w.Header().Set("Content-Type", "application/fhir+json")
		json.NewEncoder(w).Encode(map[string]any{
			"resourceType": "Patient", "id": "123", "gender": "male",
		})
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	p, err := client.ReadAs[resources.Patient](context.Background(), c, "Patient", "123")
	if err != nil {
		t.Fatal(err)
	}
	if string(p.GetId()) != "123" {
		t.Errorf("id = %q, want 123", p.GetId())
	}
	if p.GetGender() != resources.AdministrativeGenderMale {
		t.Errorf("gender = %q, want male", p.GetGender())
	}
}

func TestReadResourceInterface(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"resourceType": "Patient", "id": "456",
		})
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	res, err := client.Read(context.Background(), c, "Patient", "456")
	if err != nil {
		t.Fatal(err)
	}
	if res.GetResourceType() != "Patient" {
		t.Errorf("type = %q, want Patient", res.GetResourceType())
	}
	if string(res.GetId()) != "456" {
		t.Errorf("id = %q, want 456", res.GetId())
	}
}

func TestCreateResource(t *testing.T) {
	var receivedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/Patient" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(201)
		w.Write([]byte(`{"resourceType":"Patient","id":"new-1"}`))
	}))
	defer srv.Close()

	p, _ := resources.NewPatient().WithName("John", "Doe").Build()
	c := client.New(srv.URL)
	outcome, err := client.Create(context.Background(), c, p)
	if err != nil {
		t.Fatal(err)
	}
	if outcome == nil {
		t.Fatal("outcome should not be nil")
	}
	if !strings.Contains(string(receivedBody), "Patient") {
		t.Error("body should contain Patient resourceType")
	}
}

func TestUpdateResource(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/Patient/456" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write([]byte(`{"resourceType":"Patient","id":"456"}`))
	}))
	defer srv.Close()

	p, _ := resources.NewPatient().WithName("Jane", "Doe").Build()
	id := dt.ID("456")
	p.Id = &id

	c := client.New(srv.URL)
	_, err := client.Update(context.Background(), c, p)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateWithoutID(t *testing.T) {
	c := client.New("http://example.com")
	p, _ := resources.NewPatient().Build()
	_, err := client.Update(context.Background(), c, p)
	if err == nil {
		t.Error("should fail without id")
	}
}

func TestDeleteResource(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(204)
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	err := client.Delete(context.Background(), c, "Patient", "789")
	if err != nil {
		t.Fatal(err)
	}
}

func TestSearchFluent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("family") != "Smith" {
			t.Errorf("family = %q, want Smith", q.Get("family"))
		}
		if q.Get("_count") != "10" {
			t.Errorf("_count = %q, want 10", q.Get("_count"))
		}
		if q.Get("_sort") != "birthdate" {
			t.Errorf("_sort = %q, want birthdate", q.Get("_sort"))
		}
		json.NewEncoder(w).Encode(map[string]any{
			"resourceType": "Bundle", "type": "searchset", "total": 1,
			"entry": []map[string]any{
				{"resource": map[string]any{"resourceType": "Patient", "id": "1"}},
			},
		})
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	result, err := c.Search(context.Background(), "Patient").
		Where("family", "Smith").
		Count(10).
		Sort("birthdate").
		Execute()
	if err != nil {
		t.Fatal(err)
	}
	if result.Type != bundle.TypeSearchset {
		t.Errorf("type = %q, want searchset", result.Type)
	}
	if len(result.Entry) != 1 {
		t.Errorf("expected 1 entry, got %d", len(result.Entry))
	}
}

func TestPaging(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"resourceType": "Bundle", "type": "searchset", "total": 20,
			"entry": []map[string]any{
				{"resource": map[string]any{"resourceType": "Patient", "id": "p1"}},
			},
		})
	}))
	defer srv.Close()

	b := &bundle.Bundle{
		ResourceType: "Bundle",
		Type:         bundle.TypeSearchset,
		Link: []bundle.BundleLink{
			{Relation: "next", URL: dt.URL(srv.URL + "/Patient?page=2")},
		},
	}

	c := client.New(srv.URL)
	next, err := client.NextPage(context.Background(), c, b)
	if err != nil {
		t.Fatal(err)
	}
	if next == nil {
		t.Fatal("next page should not be nil")
	}
}

func TestPagingNoLink(t *testing.T) {
	c := client.New("http://example.com")
	b := &bundle.Bundle{Type: bundle.TypeSearchset}
	_, err := client.NextPage(context.Background(), c, b)
	if err == nil {
		t.Error("should fail when no next link")
	}
}

func TestServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(map[string]any{
			"resourceType": "OperationOutcome",
			"issue": []map[string]any{
				{"severity": "error", "code": "not-found", "diagnostics": "not found"},
			},
		})
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	_, err := client.ReadAs[resources.Patient](context.Background(), c, "Patient", "999")
	if err == nil {
		t.Fatal("should return error for 404")
	}

	srvErr, ok := err.(*client.ServerError)
	if !ok {
		t.Fatalf("expected ServerError, got %T", err)
	}
	if srvErr.StatusCode != 404 {
		t.Errorf("status = %d, want 404", srvErr.StatusCode)
	}
	oo := srvErr.OperationOutcome()
	if oo == nil {
		t.Fatal("should parse OperationOutcome from error body")
	}
}

func TestBearerTokenMiddleware(t *testing.T) {
	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(map[string]any{"resourceType": "Patient", "id": "1"})
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	c.Wrap(client.BearerToken("my-secret-token"))
	client.ReadAs[resources.Patient](context.Background(), c, "Patient", "1")

	if authHeader != "Bearer my-secret-token" {
		t.Errorf("auth = %q, want Bearer my-secret-token", authHeader)
	}
}

func TestBasicAuthMiddleware(t *testing.T) {
	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(map[string]any{"resourceType": "Patient", "id": "1"})
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	c.Wrap(client.BasicAuth("user", "pass"))
	client.ReadAs[resources.Patient](context.Background(), c, "Patient", "1")

	if !strings.HasPrefix(authHeader, "Basic ") {
		t.Errorf("auth = %q, should start with Basic", authHeader)
	}
}

func TestCustomHeadersMiddleware(t *testing.T) {
	var customHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		customHeader = r.Header.Get("X-Custom")
		json.NewEncoder(w).Encode(map[string]any{"resourceType": "Patient", "id": "1"})
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	c.Wrap(client.CustomHeaders(map[string]string{"X-Custom": "my-value"}))
	client.ReadAs[resources.Patient](context.Background(), c, "Patient", "1")

	if customHeader != "my-value" {
		t.Errorf("custom header = %q, want my-value", customHeader)
	}
}

func TestTransaction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"resourceType": "Bundle", "type": "transaction-response",
			"entry": []map[string]any{
				{"response": map[string]any{"status": "201 Created"}},
			},
		})
	}))
	defer srv.Close()

	p, _ := resources.NewPatient().WithName("Test", "User").Build()
	b := bundle.New(bundle.TypeTransaction).
		WithTransactionEntry("POST", "Patient", p).
		Build()

	c := client.New(srv.URL)
	resp, err := client.Transaction(context.Background(), c, b)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Type != bundle.TypeTransactionResponse {
		t.Errorf("type = %q, want transaction-response", resp.Type)
	}
}

func TestContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := client.New(srv.URL)
	_, err := client.Read(ctx, c, "Patient", "1")
	if err == nil {
		t.Error("should fail with cancelled context")
	}
}

func TestRetryMiddleware(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(503) // Service Unavailable
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"resourceType": "Patient", "id": "1"})
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	c.Wrap(client.Retry(3, 1*time.Millisecond))

	res, err := client.Read(context.Background(), c, "Patient", "1")
	if err != nil {
		t.Fatalf("should succeed after retries: %v", err)
	}
	if string(res.GetId()) != "1" {
		t.Error("should get patient after retry")
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestOperation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Patient/123/$everything" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"resourceType": "Bundle", "type": "searchset",
		})
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	data, err := client.Operation(context.Background(), c, "Patient/123", "$everything", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "searchset") {
		t.Error("should return bundle")
	}
}

func TestConditionalCreate(t *testing.T) {
	var ifNoneExist string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ifNoneExist = r.Header.Get("If-None-Exist")
		w.WriteHeader(201)
		w.Write([]byte(`{"resourceType":"Patient","id":"new"}`))
	}))
	defer srv.Close()

	p, _ := resources.NewPatient().WithName("Test", "User").Build()
	c := client.New(srv.URL)
	_, err := client.CreateConditional(context.Background(), c, p, "identifier=http://example.org|123")
	if err != nil {
		t.Fatal(err)
	}
	if ifNoneExist != "identifier=http://example.org|123" {
		t.Errorf("If-None-Exist = %q", ifNoneExist)
	}
}

func TestVRead(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Patient/123/_history/2" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"resourceType": "Patient", "id": "123",
			"meta": map[string]any{"versionId": "2"},
		})
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	res, err := client.VRead(context.Background(), c, "Patient", "123", "2")
	if err != nil {
		t.Fatal(err)
	}
	if res.GetResourceType() != "Patient" {
		t.Errorf("type = %q, want Patient", res.GetResourceType())
	}
}

func TestVReadAs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"resourceType": "Patient", "id": "123", "gender": "female",
		})
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	p, err := client.VReadAs[resources.Patient](context.Background(), c, "Patient", "123", "2")
	if err != nil {
		t.Fatal(err)
	}
	if p.GetGender() != resources.AdministrativeGenderFemale {
		t.Errorf("gender = %q, want female", p.GetGender())
	}
}

func TestHistory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Patient/123/_history" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"resourceType": "Bundle", "type": "history", "total": 2,
			"entry": []map[string]any{
				{"resource": map[string]any{"resourceType": "Patient", "id": "123"}},
			},
		})
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	b, err := client.History(context.Background(), c, "Patient", "123")
	if err != nil {
		t.Fatal(err)
	}
	if b.Type != bundle.TypeHistory {
		t.Errorf("type = %q, want history", b.Type)
	}
	if len(b.Entry) != 1 {
		t.Errorf("expected 1 entry, got %d", len(b.Entry))
	}
}

func TestTypeHistory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Patient/_history" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"resourceType": "Bundle", "type": "history",
		})
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	_, err := client.TypeHistory(context.Background(), c, "Patient")
	if err != nil {
		t.Fatal(err)
	}
}

func TestSystemHistory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_history" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"resourceType": "Bundle", "type": "history",
		})
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	_, err := client.SystemHistory(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPatch(t *testing.T) {
	var receivedContentType string
	var receivedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/Patient/123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		receivedContentType = r.Header.Get("Content-Type")
		receivedBody, _ = io.ReadAll(r.Body)
		w.Write([]byte(`{"resourceType":"Patient","id":"123"}`))
	}))
	defer srv.Close()

	patchBody := []byte(`[{"op":"replace","path":"/gender","value":"female"}]`)
	c := client.New(srv.URL)
	_, err := client.Patch(context.Background(), c, "Patient", "123", patchBody, "application/json-patch+json")
	if err != nil {
		t.Fatal(err)
	}
	if receivedContentType != "application/json-patch+json" {
		t.Errorf("content-type = %q", receivedContentType)
	}
	if !strings.Contains(string(receivedBody), "replace") {
		t.Error("body should contain patch operations")
	}
}

func TestLoggingMiddleware(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"resourceType": "Patient", "id": "1"})
	}))
	defer srv.Close()

	var logged bool
	var loggedMethod string
	var loggedStatus int
	c := client.New(srv.URL)
	c.Wrap(client.Logging(func(method, url string, statusCode int, duration time.Duration) {
		logged = true
		loggedMethod = method
		loggedStatus = statusCode
	}))

	client.Read(context.Background(), c, "Patient", "1")
	if !logged {
		t.Error("logging callback should have been called")
	}
	if loggedMethod != "GET" {
		t.Errorf("method = %q, want GET", loggedMethod)
	}
	if loggedStatus != 200 {
		t.Errorf("status = %d, want 200", loggedStatus)
	}
}

func TestETagCacheMiddleware(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if etag := r.Header.Get("If-None-Match"); etag == `"v1"` {
			w.WriteHeader(304)
			return
		}
		w.Header().Set("ETag", `"v1"`)
		json.NewEncoder(w).Encode(map[string]any{"resourceType": "Patient", "id": "1"})
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	c.Wrap(client.ETagCache(100))

	// First request — cache miss
	res1, err := client.Read(context.Background(), c, "Patient", "1")
	if err != nil {
		t.Fatal(err)
	}
	if string(res1.GetId()) != "1" {
		t.Errorf("id = %q, want 1", res1.GetId())
	}

	// Second request — should use cache (304)
	res2, err := client.Read(context.Background(), c, "Patient", "1")
	if err != nil {
		t.Fatal(err)
	}
	if string(res2.GetId()) != "1" {
		t.Errorf("cached id = %q, want 1", res2.GetId())
	}

	if callCount != 2 {
		t.Errorf("expected 2 server calls, got %d", callCount)
	}
}

func TestRetryMiddlewareExhausted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	c.Wrap(client.Retry(2, 1*time.Millisecond))

	_, err := client.Read(context.Background(), c, "Patient", "1")
	if err == nil {
		t.Error("should fail after exhausting retries")
	}
	srvErr, ok := err.(*client.ServerError)
	if !ok {
		t.Fatalf("expected ServerError, got %T", err)
	}
	if srvErr.StatusCode != 503 {
		t.Errorf("status = %d, want 503", srvErr.StatusCode)
	}
}

func TestReadBinary(t *testing.T) {
	pdfContent := []byte("%PDF-1.4 fake content")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Binary/doc-1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Write(pdfContent)
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	data, contentType, err := client.ReadBinary(context.Background(), c, "doc-1")
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "application/pdf" {
		t.Errorf("content-type = %q, want application/pdf", contentType)
	}
	if string(data) != string(pdfContent) {
		t.Error("content mismatch")
	}
}

func TestCreateBinary(t *testing.T) {
	var receivedContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		receivedContentType = r.Header.Get("Content-Type")
		w.WriteHeader(201)
		w.Write([]byte(`{"resourceType":"Binary","id":"new-1"}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	_, err := client.CreateBinary(context.Background(), c, []byte("image data"), "image/png")
	if err != nil {
		t.Fatal(err)
	}
	if receivedContentType != "image/png" {
		t.Errorf("content-type = %q, want image/png", receivedContentType)
	}
}
