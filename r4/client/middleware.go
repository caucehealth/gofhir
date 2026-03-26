// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"encoding/base64"
	"net/http"
	"time"
)

// Middleware wraps an http.RoundTripper to intercept requests.
// Chain multiple middlewares with Wrap.
type Middleware func(http.RoundTripper) http.RoundTripper

// Wrap applies middlewares to the client's HTTP transport.
// Middlewares are applied in order: first middleware is outermost.
// Wrap must be called during initialization, not concurrently at runtime.
func (c *Client) Wrap(middlewares ...Middleware) *Client {
	transport := c.httpClient.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	for i := len(middlewares) - 1; i >= 0; i-- {
		transport = middlewares[i](transport)
	}
	// Clone the http.Client to avoid mutating shared state
	clone := *c.httpClient
	clone.Transport = transport
	c.httpClient = &clone
	return c
}

// --- Built-in Middlewares ---

// BearerToken adds an Authorization: Bearer header to every request.
func BearerToken(token string) Middleware {
	return func(next http.RoundTripper) http.RoundTripper {
		return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req = req.Clone(req.Context())
			req.Header.Set("Authorization", "Bearer "+token)
			return next.RoundTrip(req)
		})
	}
}

// BasicAuth adds an Authorization: Basic header to every request.
func BasicAuth(username, password string) Middleware {
	encoded := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	return func(next http.RoundTripper) http.RoundTripper {
		return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req = req.Clone(req.Context())
			req.Header.Set("Authorization", "Basic "+encoded)
			return next.RoundTrip(req)
		})
	}
}

// CustomHeaders adds static headers to every request.
func CustomHeaders(headers map[string]string) Middleware {
	return func(next http.RoundTripper) http.RoundTripper {
		return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req = req.Clone(req.Context())
			for k, v := range headers {
				req.Header.Set(k, v)
			}
			return next.RoundTrip(req)
		})
	}
}

// UserAgent sets the User-Agent header on every request.
func UserAgent(agent string) Middleware {
	return CustomHeaders(map[string]string{"User-Agent": agent})
}

// Retry adds automatic retry with exponential backoff for transient errors
// (5xx status codes and network errors). MaxRetries of 0 means no retries.
func Retry(maxRetries int, initialDelay time.Duration) Middleware {
	return func(next http.RoundTripper) http.RoundTripper {
		return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			var lastErr error
			var lastResp *http.Response
			delay := initialDelay

			for attempt := 0; attempt <= maxRetries; attempt++ {
				if attempt > 0 {
					select {
					case <-req.Context().Done():
						return nil, req.Context().Err()
					case <-time.After(delay):
					}
					delay *= 2 // exponential backoff
				}

				resp, err := next.RoundTrip(req)
				if err != nil {
					lastErr = err
					continue // network error, retry
				}

				// Don't retry client errors (4xx)
				if resp.StatusCode < 500 {
					return resp, nil
				}

				// Server error (5xx) — retry
				lastResp = resp
				lastErr = nil
			}

			if lastResp != nil {
				return lastResp, nil
			}
			return nil, lastErr
		})
	}
}

// roundTripperFunc adapts a function to http.RoundTripper.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
