// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package bulk_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/caucehealth/gofhir/r4/bulk"
	"github.com/caucehealth/gofhir/r4/resources"
)

func TestSystemExportKickOff(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/$export" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Prefer") != "respond-async" {
			t.Error("should send Prefer: respond-async")
		}
		w.Header().Set("Content-Location", "http://example.com/status/123")
		w.WriteHeader(202)
	}))
	defer srv.Close()

	e := bulk.NewExporter(srv.URL, bulk.WithHTTPClient(srv.Client()))
	job, err := e.SystemExport(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if job.StatusURL != "http://example.com/status/123" {
		t.Errorf("status URL = %q", job.StatusURL)
	}
}

func TestPatientExport(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Patient/$export" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Location", "http://example.com/status/456")
		w.WriteHeader(202)
	}))
	defer srv.Close()

	e := bulk.NewExporter(srv.URL)
	job, err := e.PatientExport(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if job.StatusURL == "" {
		t.Error("should have status URL")
	}
}

func TestGroupExport(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Group/grp-1/$export" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Location", "http://example.com/status/789")
		w.WriteHeader(202)
	}))
	defer srv.Close()

	e := bulk.NewExporter(srv.URL)
	_, err := e.GroupExport(context.Background(), "grp-1", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestExportWithParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("_type") != "Patient,Observation" {
			t.Errorf("_type = %q", q.Get("_type"))
		}
		if q.Get("_since") == "" {
			t.Error("should have _since parameter")
		}
		w.Header().Set("Content-Location", "http://example.com/status/1")
		w.WriteHeader(202)
	}))
	defer srv.Close()

	since := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	e := bulk.NewExporter(srv.URL)
	_, err := e.SystemExport(context.Background(), &bulk.ExportParams{
		Types: []string{"Patient", "Observation"},
		Since: &since,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPollInProgress(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(202)
	}))
	defer srv.Close()

	job := &bulk.Job{StatusURL: srv.URL, HTTPClient: srv.Client()}
	status, err := job.Poll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.Complete {
		t.Error("should not be complete")
	}
	if status.RetryAfter != 5 {
		t.Errorf("retry-after = %d, want 5", status.RetryAfter)
	}
}

func TestPollComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"transactionTime": "2024-01-01T00:00:00Z",
			"request":         "https://example.com/$export",
			"output": []map[string]any{
				{"type": "Patient", "url": "https://example.com/data/patient.ndjson", "count": 100},
				{"type": "Observation", "url": "https://example.com/data/obs.ndjson", "count": 500},
			},
		})
	}))
	defer srv.Close()

	job := &bulk.Job{StatusURL: srv.URL, HTTPClient: srv.Client()}
	status, err := job.Poll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !status.Complete {
		t.Error("should be complete")
	}
	if len(status.Output) != 2 {
		t.Errorf("expected 2 output files, got %d", len(status.Output))
	}
	if status.Output[0].Type != "Patient" {
		t.Errorf("first output type = %q", status.Output[0].Type)
	}
}

func TestDeleteJob(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(202)
	}))
	defer srv.Close()

	job := &bulk.Job{StatusURL: srv.URL, HTTPClient: srv.Client()}
	if err := job.Delete(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestNDJSONReader(t *testing.T) {
	ndjson := `{"resourceType":"Patient","id":"1"}
{"resourceType":"Patient","id":"2"}
{"resourceType":"Observation","id":"3"}
`
	reader := bulk.NewNDJSONReaderFromReader(strings.NewReader(ndjson))

	count := 0
	for reader.Next() {
		count++
		res, err := reader.Resource()
		if err != nil {
			t.Fatal(err)
		}
		if res.GetResourceType() != "Patient" && res.GetResourceType() != "Observation" {
			t.Errorf("unexpected type: %s", res.GetResourceType())
		}
	}
	if reader.Err() != nil {
		t.Fatal(reader.Err())
	}
	if count != 3 {
		t.Errorf("expected 3 resources, got %d", count)
	}
}

func TestNDJSONReaderSkipsEmptyLines(t *testing.T) {
	ndjson := `{"resourceType":"Patient","id":"1"}

{"resourceType":"Patient","id":"2"}

`
	reader := bulk.NewNDJSONReaderFromReader(strings.NewReader(ndjson))

	count := 0
	for reader.Next() {
		count++
	}
	if count != 2 {
		t.Errorf("expected 2 resources, got %d", count)
	}
}

func TestNDJSONReaderDecode(t *testing.T) {
	ndjson := `{"resourceType":"Patient","id":"1","gender":"male"}`
	reader := bulk.NewNDJSONReaderFromReader(strings.NewReader(ndjson))

	if !reader.Next() {
		t.Fatal("should have one entry")
	}

	var p resources.Patient
	if err := reader.Decode(&p); err != nil {
		t.Fatal(err)
	}
	if p.ResourceType != "Patient" {
		t.Errorf("type = %q", p.ResourceType)
	}
}

func TestNDJSONWriter(t *testing.T) {
	var buf bytes.Buffer
	w := bulk.NewNDJSONWriter(&buf)

	p1 := map[string]any{"resourceType": "Patient", "id": "1"}
	p2 := map[string]any{"resourceType": "Patient", "id": "2"}
	w.Write(p1)
	w.Write(p2)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}

	// Verify each line is valid JSON
	for _, line := range lines {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Errorf("invalid JSON: %s", line)
		}
	}
}

func TestNDJSONWriterRaw(t *testing.T) {
	var buf bytes.Buffer
	w := bulk.NewNDJSONWriter(&buf)
	w.WriteRaw(json.RawMessage(`{"resourceType":"Patient","id":"raw"}`))

	if !strings.Contains(buf.String(), `"id":"raw"`) {
		t.Error("should contain raw JSON")
	}
}

func TestDownload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/fhir+ndjson")
		io.WriteString(w, `{"resourceType":"Patient","id":"1"}`+"\n")
		io.WriteString(w, `{"resourceType":"Patient","id":"2"}`+"\n")
	}))
	defer srv.Close()

	job := &bulk.Job{StatusURL: srv.URL, HTTPClient: srv.Client()}
	reader, err := job.Download(context.Background(), bulk.OutputFile{
		Type: "Patient", URL: srv.URL + "/data/patient.ndjson",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	count := 0
	for reader.Next() {
		count++
	}
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestExportKickOffError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	e := bulk.NewExporter(srv.URL)
	_, err := e.SystemExport(context.Background(), nil)
	if err == nil {
		t.Error("should fail on 500")
	}
}
