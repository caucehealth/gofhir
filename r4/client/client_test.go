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
