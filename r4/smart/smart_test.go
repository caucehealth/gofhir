// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package smart_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/caucehealth/gofhir/r4/smart"
)

func TestDiscover(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/smart-configuration" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"authorization_endpoint":             "https://auth.example.org/authorize",
			"token_endpoint":                     "https://auth.example.org/token",
			"scopes_supported":                   []string{"patient/*.read", "launch"},
			"capabilities":                       []string{"launch-ehr", "launch-standalone", "client-public", "client-confidential-symmetric"},
			"code_challenge_methods_supported":    []string{"S256"},
		})
	}))
	defer srv.Close()

	cfg, err := smart.Discover(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AuthorizationEndpoint != "https://auth.example.org/authorize" {
		t.Errorf("auth endpoint = %q", cfg.AuthorizationEndpoint)
	}
	if cfg.TokenEndpoint != "https://auth.example.org/token" {
		t.Errorf("token endpoint = %q", cfg.TokenEndpoint)
	}
	if !cfg.HasCapability("launch-standalone") {
		t.Error("should have launch-standalone capability")
	}
	if cfg.HasCapability("nonexistent") {
		t.Error("should not have nonexistent capability")
	}
	if !cfg.SupportsPKCE() {
		t.Error("should support PKCE")
	}
}

func TestDiscoverError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	_, err := smart.Discover(context.Background(), srv.URL)
	if err == nil {
		t.Error("should fail on 404")
	}
}

func TestStandaloneLauncherAuthURL(t *testing.T) {
	cfg := &smart.Configuration{
		AuthorizationEndpoint:         "https://auth.example.org/authorize",
		TokenEndpoint:                 "https://auth.example.org/token",
		CodeChallengeMethodsSupported: []string{"S256"},
	}

	launcher := smart.NewStandaloneLauncher(cfg, smart.ClientConfig{
		ClientID:    "my-app",
		RedirectURI: "http://localhost:8080/callback",
		Scopes:      []string{"patient/Patient.read", "launch/patient"},
	})

	authURL := launcher.AuthURL("test-state")
	if !strings.Contains(authURL, "response_type=code") {
		t.Error("should contain response_type=code")
	}
	if !strings.Contains(authURL, "client_id=my-app") {
		t.Error("should contain client_id")
	}
	if !strings.Contains(authURL, "state=test-state") {
		t.Error("should contain state")
	}
	if !strings.Contains(authURL, "code_challenge=") {
		t.Error("should contain PKCE code_challenge")
	}
	if !strings.Contains(authURL, "code_challenge_method=S256") {
		t.Error("should contain S256 method")
	}
}

func TestStandaloneLauncherExchange(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Form.Get("grant_type") != "authorization_code" {
			t.Errorf("grant_type = %q", r.Form.Get("grant_type"))
		}
		if r.Form.Get("code") != "test-code" {
			t.Errorf("code = %q", r.Form.Get("code"))
		}
		json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "access-123",
			"token_type":    "Bearer",
			"expires_in":    3600,
			"refresh_token": "refresh-456",
			"scope":         "patient/Patient.read",
			"patient":       "Patient/789",
		})
	}))
	defer srv.Close()

	cfg := &smart.Configuration{
		AuthorizationEndpoint: "https://auth.example.org/authorize",
		TokenEndpoint:         srv.URL,
	}
	launcher := smart.NewStandaloneLauncher(cfg, smart.ClientConfig{
		ClientID:    "my-app",
		RedirectURI: "http://localhost:8080/callback",
		Scopes:      []string{"patient/Patient.read"},
	})

	token, err := launcher.Exchange(context.Background(), "test-code")
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "access-123" {
		t.Errorf("access_token = %q", token.AccessToken)
	}
	if token.RefreshToken != "refresh-456" {
		t.Errorf("refresh_token = %q", token.RefreshToken)
	}
	if token.Patient != "Patient/789" {
		t.Errorf("patient = %q", token.Patient)
	}
	if token.IsExpired() {
		t.Error("token should not be expired")
	}
	if !token.Valid() {
		t.Error("token should be valid")
	}
}

func TestEHRLauncherAuthURL(t *testing.T) {
	cfg := &smart.Configuration{
		AuthorizationEndpoint: "https://auth.example.org/authorize",
		TokenEndpoint:         "https://auth.example.org/token",
	}

	launcher := smart.NewEHRLauncher(cfg, smart.ClientConfig{
		ClientID:    "ehr-app",
		RedirectURI: "http://localhost:8080/callback",
		Scopes:      []string{"launch", "patient/Patient.read"},
	})

	authURL := launcher.AuthURL("state-1", "launch-context-xyz")
	if !strings.Contains(authURL, "launch=launch-context-xyz") {
		t.Error("should contain launch parameter")
	}
}

func TestRefresh(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Form.Get("grant_type") != "refresh_token" {
			t.Errorf("grant_type = %q", r.Form.Get("grant_type"))
		}
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "new-access",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer srv.Close()

	cfg := &smart.Configuration{TokenEndpoint: srv.URL}
	token, err := smart.Refresh(context.Background(), cfg, "my-app", "", "old-refresh-token")
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "new-access" {
		t.Errorf("access_token = %q", token.AccessToken)
	}
}

func TestBackendAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Form.Get("grant_type") != "client_credentials" {
			t.Errorf("grant_type = %q", r.Form.Get("grant_type"))
		}
		if r.Form.Get("client_assertion_type") != "urn:ietf:params:oauth:client-assertion-type:jwt-bearer" {
			t.Errorf("assertion_type = %q", r.Form.Get("client_assertion_type"))
		}
		assertion := r.Form.Get("client_assertion")
		if assertion == "" {
			t.Error("missing client_assertion")
		}
		// Verify it's a JWT (3 dot-separated parts)
		parts := strings.Split(assertion, ".")
		if len(parts) != 3 {
			t.Errorf("assertion should be a JWT, got %d parts", len(parts))
		}

		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "backend-token",
			"token_type":   "Bearer",
			"expires_in":   300,
		})
	}))
	defer srv.Close()

	// Generate a test RSA key
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keyBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	pemBlock := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})

	cfg := &smart.Configuration{TokenEndpoint: srv.URL}
	token, err := smart.BackendAuth(context.Background(), cfg, smart.BackendConfig{
		ClientID:   "backend-service",
		PrivateKey: pemBlock,
		Scopes:     []string{"system/*.read"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "backend-token" {
		t.Errorf("access_token = %q", token.AccessToken)
	}
}

func TestTokenExpiry(t *testing.T) {
	token := &smart.Token{
		AccessToken: "test",
		ExpiresAt:   time.Now().Add(-1 * time.Minute),
	}
	if !token.IsExpired() {
		t.Error("should be expired")
	}
	if token.Valid() {
		t.Error("expired token should not be valid")
	}

	token2 := &smart.Token{
		AccessToken: "test",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	}
	if token2.IsExpired() {
		t.Error("should not be expired")
	}
	if !token2.Valid() {
		t.Error("should be valid")
	}
}

func TestStaticTokenSource(t *testing.T) {
	token := &smart.Token{AccessToken: "static", ExpiresAt: time.Now().Add(time.Hour)}
	src := smart.StaticTokenSource(token)

	got, err := src.Token(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "static" {
		t.Errorf("token = %q", got.AccessToken)
	}
}

func TestRefreshingTokenSource(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "refreshed",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer srv.Close()

	cfg := &smart.Configuration{TokenEndpoint: srv.URL}
	initial := &smart.Token{
		AccessToken:  "expired",
		RefreshToken: "refresh-me",
		ExpiresAt:    time.Now().Add(-1 * time.Minute), // already expired
	}

	src := smart.NewRefreshingTokenSource(cfg, "my-app", "", initial)
	got, err := src.Token(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "refreshed" {
		t.Errorf("token = %q", got.AccessToken)
	}
	if callCount != 1 {
		t.Errorf("expected 1 refresh call, got %d", callCount)
	}
}

func TestScopeHelpers(t *testing.T) {
	if s := smart.PatientScope("Patient", "read"); s != "patient/Patient.read" {
		t.Errorf("patient scope = %q", s)
	}
	if s := smart.UserScope("Observation", "write"); s != "user/Observation.write" {
		t.Errorf("user scope = %q", s)
	}
	if s := smart.SystemScope("*", "read"); s != "system/*.read" {
		t.Errorf("system scope = %q", s)
	}
}

func TestTokenExchangeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer srv.Close()

	cfg := &smart.Configuration{
		AuthorizationEndpoint: "https://auth.example.org/authorize",
		TokenEndpoint:         srv.URL,
	}
	launcher := smart.NewStandaloneLauncher(cfg, smart.ClientConfig{
		ClientID:    "my-app",
		RedirectURI: "http://localhost:8080/callback",
	})

	_, err := launcher.Exchange(context.Background(), "bad-code")
	if err == nil {
		t.Error("should fail with invalid_grant")
	}
}
