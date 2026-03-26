// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

// Package bulk implements the FHIR Bulk Data Access (Flat FHIR) specification,
// including $export kick-off, status polling, NDJSON download and parsing.
//
// Usage:
//
//	exporter := bulk.NewExporter(fhirClient)
//	job, _ := exporter.SystemExport(ctx)       // kick off system-level $export
//	status, _ := job.Poll(ctx)                  // poll until complete
//	for _, output := range status.Output {
//	    reader, _ := job.Download(ctx, output)  // stream NDJSON
//	    for reader.Next() {
//	        resource := reader.Resource()
//	    }
//	}
package bulk

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/caucehealth/gofhir/r4/resources"
)

// ExportType represents the level of a bulk export.
type ExportType string

const (
	ExportSystem  ExportType = "system"
	ExportPatient ExportType = "patient"
	ExportGroup   ExportType = "group"
)

// Exporter initiates and manages FHIR Bulk Data exports.
type Exporter struct {
	baseURL    string
	httpClient *http.Client
}

// NewExporter creates a bulk data exporter.
func NewExporter(baseURL string, opts ...ExporterOption) *Exporter {
	e := &Exporter{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// ExporterOption configures an Exporter.
type ExporterOption func(*Exporter)

// WithHTTPClient sets a custom HTTP client for the exporter.
func WithHTTPClient(c *http.Client) ExporterOption {
	return func(e *Exporter) { e.httpClient = c }
}

// ExportParams configures an export request.
type ExportParams struct {
	// OutputFormat defaults to "application/fhir+ndjson".
	OutputFormat string
	// Since exports only resources updated after this time.
	Since *time.Time
	// Types limits the export to these resource types.
	Types []string
}

// SystemExport kicks off a system-level $export.
func (e *Exporter) SystemExport(ctx context.Context, params *ExportParams) (*Job, error) {
	return e.kickOff(ctx, e.baseURL+"/$export", params)
}

// PatientExport kicks off a patient-level $export.
func (e *Exporter) PatientExport(ctx context.Context, params *ExportParams) (*Job, error) {
	return e.kickOff(ctx, e.baseURL+"/Patient/$export", params)
}

// GroupExport kicks off a group-level $export.
func (e *Exporter) GroupExport(ctx context.Context, groupID string, params *ExportParams) (*Job, error) {
	return e.kickOff(ctx, e.baseURL+"/Group/"+groupID+"/$export", params)
}

func (e *Exporter) kickOff(ctx context.Context, url string, params *ExportParams) (*Job, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/fhir+json")
	req.Header.Set("Prefer", "respond-async")

	if params != nil {
		q := req.URL.Query()
		if params.OutputFormat != "" {
			q.Set("_outputFormat", params.OutputFormat)
		}
		if params.Since != nil {
			q.Set("_since", params.Since.Format(time.RFC3339))
		}
		if len(params.Types) > 0 {
			q.Set("_type", strings.Join(params.Types, ","))
		}
		req.URL.RawQuery = q.Encode()
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("export kick-off: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 202 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("export kick-off: status %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	statusURL := resp.Header.Get("Content-Location")
	if statusURL == "" {
		return nil, fmt.Errorf("export kick-off: no Content-Location header")
	}

	return &Job{
		StatusURL:  statusURL,
		HTTPClient: e.httpClient,
	}, nil
}

// Job represents a running bulk export job.
type Job struct {
	StatusURL  string
	HTTPClient *http.Client
}

// Status represents the status of a bulk export job.
type Status struct {
	// Complete is true when the export has finished.
	Complete bool
	// TransactionTime is the server's transaction time for the export.
	TransactionTime string `json:"transactionTime"`
	// Request is the original kick-off request URL.
	Request string `json:"request"`
	// RequiresAccessToken indicates if output URLs need authentication.
	RequiresAccessToken bool `json:"requiresAccessToken"`
	// Output lists the NDJSON files to download.
	Output []OutputFile `json:"output"`
	// Error lists any error files.
	Error []OutputFile `json:"error"`
	// RetryAfter is the suggested wait time (seconds) before polling again.
	RetryAfter int
}

// OutputFile describes a single output file in a bulk export.
type OutputFile struct {
	Type  string `json:"type"`
	URL   string `json:"url"`
	Count int    `json:"count,omitempty"`
}

// Poll checks the status of the export job.
// Returns a Status with Complete=true when finished.
func (j *Job) Poll(ctx context.Context) (*Status, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", j.StatusURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := j.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("poll export status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 202 {
		// Still in progress
		retryAfter := 10
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if n, err := strconv.Atoi(ra); err == nil {
				retryAfter = n
			}
		}
		return &Status{Complete: false, RetryAfter: retryAfter}, nil
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("poll export status: status %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var status Status
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("parse export status: %w", err)
	}
	status.Complete = true
	return &status, nil
}

// WaitForComplete polls until the export is complete, respecting Retry-After.
func (j *Job) WaitForComplete(ctx context.Context) (*Status, error) {
	for {
		status, err := j.Poll(ctx)
		if err != nil {
			return nil, err
		}
		if status.Complete {
			return status, nil
		}
		wait := time.Duration(status.RetryAfter) * time.Second
		if wait < time.Second {
			wait = time.Second
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
	}
}

// Delete cancels a running export job.
func (j *Job) Delete(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", j.StatusURL, nil)
	if err != nil {
		return err
	}
	resp, err := j.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete export job: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 202 && resp.StatusCode != 204 {
		return fmt.Errorf("delete export job: status %d", resp.StatusCode)
	}
	return nil
}

// Download fetches an NDJSON output file and returns a streaming reader.
func (j *Job) Download(ctx context.Context, output OutputFile) (*NDJSONReader, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", output.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/fhir+ndjson")

	resp, err := j.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download NDJSON: %w", err)
	}

	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, fmt.Errorf("download NDJSON: status %d", resp.StatusCode)
	}

	return NewNDJSONReader(resp.Body), nil
}

// --- NDJSON Reader/Writer ---

// NDJSONReader reads FHIR resources from a newline-delimited JSON stream.
type NDJSONReader struct {
	scanner  *bufio.Scanner
	body     io.ReadCloser
	current  json.RawMessage
	err      error
}

// NewNDJSONReader creates a reader from an io.ReadCloser.
func NewNDJSONReader(r io.ReadCloser) *NDJSONReader {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // up to 10MB per line
	return &NDJSONReader{scanner: scanner, body: r}
}

// NewNDJSONReaderFromReader creates a reader from an io.Reader (no close).
func NewNDJSONReaderFromReader(r io.Reader) *NDJSONReader {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)
	return &NDJSONReader{scanner: scanner}
}

// Next advances to the next resource. Returns false when done.
func (r *NDJSONReader) Next() bool {
	for r.scanner.Scan() {
		line := strings.TrimSpace(r.scanner.Text())
		if line == "" {
			continue
		}
		r.current = json.RawMessage(line)
		return true
	}
	r.err = r.scanner.Err()
	return false
}

// Bytes returns the current line as raw JSON bytes.
func (r *NDJSONReader) Bytes() json.RawMessage {
	return r.current
}

// Resource parses the current line as a typed FHIR resource.
func (r *NDJSONReader) Resource() (resources.Resource, error) {
	return resources.ParseResource(r.current)
}

// Decode unmarshals the current line into the given type.
func (r *NDJSONReader) Decode(v any) error {
	return json.Unmarshal(r.current, v)
}

// Err returns any error from scanning.
func (r *NDJSONReader) Err() error {
	return r.err
}

// Close closes the underlying reader.
func (r *NDJSONReader) Close() error {
	if r.body != nil {
		return r.body.Close()
	}
	return nil
}

// NDJSONWriter writes FHIR resources as newline-delimited JSON.
type NDJSONWriter struct {
	w io.Writer
}

// NewNDJSONWriter creates a writer.
func NewNDJSONWriter(w io.Writer) *NDJSONWriter {
	return &NDJSONWriter{w: w}
}

// Write marshals a resource to JSON and writes it as a single line.
func (w *NDJSONWriter) Write(resource any) error {
	data, err := json.Marshal(resource)
	if err != nil {
		return err
	}
	_, err = w.w.Write(append(data, '\n'))
	return err
}

// WriteRaw writes pre-marshaled JSON as a single line.
func (w *NDJSONWriter) WriteRaw(data json.RawMessage) error {
	_, err := w.w.Write(append([]byte(data), '\n'))
	return err
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
