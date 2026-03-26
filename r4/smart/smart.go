// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

// Package smart implements the SMART on FHIR authorization framework,
// supporting standalone launch, EHR launch, backend services (client
// credentials), and token management.
//
// Usage (standalone launch):
//
//	cfg, _ := smart.Discover(ctx, "https://fhir.example.org")
//	launcher := smart.NewStandaloneLauncher(cfg, smart.ClientConfig{
//	    ClientID:    "my-app",
//	    RedirectURI: "http://localhost:8080/callback",
//	    Scopes:      []string{"patient/Patient.read", "launch/patient"},
//	})
//	authURL := launcher.AuthURL("state-123")
//	// redirect user to authURL, handle callback:
//	token, _ := launcher.Exchange(ctx, code)
//
// Usage (backend services / client credentials):
//
//	token, _ := smart.BackendAuth(ctx, cfg, smart.BackendConfig{
//	    ClientID: "service-app",
//	    KeyFile:  "private-key.pem",
//	    Scopes:   []string{"system/*.read"},
//	})
package smart

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Configuration holds the SMART on FHIR server endpoints discovered
// from .well-known/smart-configuration or the CapabilityStatement.
type Configuration struct {
	// AuthorizationEndpoint is the OAuth2 authorization URL.
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	// TokenEndpoint is the OAuth2 token URL.
	TokenEndpoint string `json:"token_endpoint"`
	// RegistrationEndpoint is the dynamic client registration URL.
	RegistrationEndpoint string `json:"registration_endpoint,omitempty"`
	// ManagementEndpoint is the token management URL.
	ManagementEndpoint string `json:"management_endpoint,omitempty"`
	// IntrospectionEndpoint is the token introspection URL.
	IntrospectionEndpoint string `json:"introspection_endpoint,omitempty"`
	// RevocationEndpoint is the token revocation URL.
	RevocationEndpoint string `json:"revocation_endpoint,omitempty"`
	// ScopesSupported lists supported OAuth scopes.
	ScopesSupported []string `json:"scopes_supported,omitempty"`
	// ResponseTypesSupported lists supported OAuth response types.
	ResponseTypesSupported []string `json:"response_types_supported,omitempty"`
	// Capabilities lists SMART capabilities (e.g. "launch-ehr", "client-public").
	Capabilities []string `json:"capabilities,omitempty"`
	// CodeChallengeMethodsSupported lists supported PKCE methods.
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported,omitempty"`
}

// Discover fetches the SMART configuration from the FHIR server's
// .well-known/smart-configuration endpoint.
func Discover(ctx context.Context, baseURL string) (*Configuration, error) {
	return DiscoverWith(ctx, baseURL, http.DefaultClient)
}

// DiscoverWith fetches the SMART configuration using a custom HTTP client.
func DiscoverWith(ctx context.Context, baseURL string, httpClient *http.Client) (*Configuration, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	wellKnownURL := baseURL + "/.well-known/smart-configuration"

	req, err := http.NewRequestWithContext(ctx, "GET", wellKnownURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("discover SMART config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discover SMART config: status %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var cfg Configuration
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse SMART config: %w", err)
	}
	return &cfg, nil
}

// HasCapability checks if the server supports a specific SMART capability.
func (c *Configuration) HasCapability(cap string) bool {
	for _, s := range c.Capabilities {
		if s == cap {
			return true
		}
	}
	return false
}

// SupportsPKCE returns true if the server supports PKCE (S256).
func (c *Configuration) SupportsPKCE() bool {
	for _, m := range c.CodeChallengeMethodsSupported {
		if m == "S256" {
			return true
		}
	}
	return false
}

// --- Token ---

// Token represents an OAuth2 token response from the SMART authorization server.
type Token struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
	// SMART-specific context
	Patient      string `json:"patient,omitempty"`
	Encounter    string `json:"encounter,omitempty"`
	NeedPatientBanner bool `json:"need_patient_banner,omitempty"`
	SmartStyleURL string `json:"smart_style_url,omitempty"`
	// IDToken is the OpenID Connect ID token if requested.
	IDToken string `json:"id_token,omitempty"`
	// ExpiresAt is computed from ExpiresIn at token receipt time.
	ExpiresAt time.Time `json:"-"`
}

// IsExpired returns true if the token has expired (with a 30-second buffer).
func (t *Token) IsExpired() bool {
	return time.Now().After(t.ExpiresAt.Add(-30 * time.Second))
}

// Valid returns true if the token has a non-empty access token and is not expired.
func (t *Token) Valid() bool {
	return t.AccessToken != "" && !t.IsExpired()
}

// --- Client Config ---

// ClientConfig holds client application credentials.
type ClientConfig struct {
	ClientID     string
	ClientSecret string // empty for public clients
	RedirectURI  string
	Scopes       []string
}

// --- Standalone Launch ---

// StandaloneLauncher handles the SMART standalone launch flow (authorization code).
type StandaloneLauncher struct {
	config *Configuration
	client ClientConfig
	pkce   *pkceParams
}

// NewStandaloneLauncher creates a launcher for the standalone launch flow.
func NewStandaloneLauncher(config *Configuration, client ClientConfig) *StandaloneLauncher {
	launcher := &StandaloneLauncher{config: config, client: client}
	if config.SupportsPKCE() {
		launcher.pkce = generatePKCE()
	}
	return launcher
}

// AuthURL returns the authorization URL that the user should be redirected to.
func (l *StandaloneLauncher) AuthURL(state string) string {
	params := url.Values{
		"response_type": {"code"},
		"client_id":     {l.client.ClientID},
		"redirect_uri":  {l.client.RedirectURI},
		"scope":         {strings.Join(l.client.Scopes, " ")},
		"state":         {state},
		"aud":           {l.config.TokenEndpoint},
	}
	if l.pkce != nil {
		params.Set("code_challenge", l.pkce.challenge)
		params.Set("code_challenge_method", "S256")
	}
	return l.config.AuthorizationEndpoint + "?" + params.Encode()
}

// Exchange trades an authorization code for a token.
func (l *StandaloneLauncher) Exchange(ctx context.Context, code string) (*Token, error) {
	return l.ExchangeWith(ctx, code, http.DefaultClient)
}

// ExchangeWith trades an authorization code for a token using a custom HTTP client.
func (l *StandaloneLauncher) ExchangeWith(ctx context.Context, code string, httpClient *http.Client) (*Token, error) {
	params := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {l.client.RedirectURI},
		"client_id":    {l.client.ClientID},
	}
	if l.pkce != nil {
		params.Set("code_verifier", l.pkce.verifier)
	}
	return doTokenRequest(ctx, httpClient, l.config.TokenEndpoint, params, l.client.ClientSecret)
}

// --- EHR Launch ---

// EHRLauncher handles the SMART EHR launch flow.
type EHRLauncher struct {
	config *Configuration
	client ClientConfig
	pkce   *pkceParams
}

// NewEHRLauncher creates a launcher for the EHR launch flow.
func NewEHRLauncher(config *Configuration, client ClientConfig) *EHRLauncher {
	launcher := &EHRLauncher{config: config, client: client}
	if config.SupportsPKCE() {
		launcher.pkce = generatePKCE()
	}
	return launcher
}

// AuthURL returns the authorization URL including the launch parameter from the EHR.
func (l *EHRLauncher) AuthURL(state, launch string) string {
	params := url.Values{
		"response_type": {"code"},
		"client_id":     {l.client.ClientID},
		"redirect_uri":  {l.client.RedirectURI},
		"scope":         {strings.Join(l.client.Scopes, " ")},
		"state":         {state},
		"launch":        {launch},
		"aud":           {l.config.TokenEndpoint},
	}
	if l.pkce != nil {
		params.Set("code_challenge", l.pkce.challenge)
		params.Set("code_challenge_method", "S256")
	}
	return l.config.AuthorizationEndpoint + "?" + params.Encode()
}

// Exchange trades an authorization code for a token.
func (l *EHRLauncher) Exchange(ctx context.Context, code string) (*Token, error) {
	return l.ExchangeWith(ctx, code, http.DefaultClient)
}

// ExchangeWith trades an authorization code for a token using a custom HTTP client.
func (l *EHRLauncher) ExchangeWith(ctx context.Context, code string, httpClient *http.Client) (*Token, error) {
	params := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {l.client.RedirectURI},
		"client_id":    {l.client.ClientID},
	}
	if l.pkce != nil {
		params.Set("code_verifier", l.pkce.verifier)
	}
	return doTokenRequest(ctx, httpClient, l.config.TokenEndpoint, params, l.client.ClientSecret)
}

// --- Token Refresh ---

// Refresh exchanges a refresh token for a new access token.
func Refresh(ctx context.Context, config *Configuration, clientID, clientSecret, refreshToken string) (*Token, error) {
	return RefreshWith(ctx, config, clientID, clientSecret, refreshToken, http.DefaultClient)
}

// RefreshWith exchanges a refresh token for a new access token using a custom HTTP client.
func RefreshWith(ctx context.Context, config *Configuration, clientID, clientSecret, refreshToken string, httpClient *http.Client) (*Token, error) {
	params := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {clientID},
	}
	return doTokenRequest(ctx, httpClient, config.TokenEndpoint, params, clientSecret)
}

// --- Backend Services (Client Credentials) ---

// BackendConfig holds configuration for backend services authorization.
type BackendConfig struct {
	ClientID string
	// PrivateKey is a PEM-encoded RSA private key for JWT assertion.
	PrivateKey []byte
	Scopes     []string
}

// BackendAuth performs the SMART Backend Services authorization flow
// using a signed JWT client assertion.
func BackendAuth(ctx context.Context, config *Configuration, bc BackendConfig) (*Token, error) {
	return BackendAuthWith(ctx, config, bc, http.DefaultClient)
}

// BackendAuthWith performs backend auth with a custom HTTP client.
func BackendAuthWith(ctx context.Context, config *Configuration, bc BackendConfig, httpClient *http.Client) (*Token, error) {
	assertion, err := buildClientAssertion(bc.ClientID, config.TokenEndpoint, bc.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("build client assertion: %w", err)
	}

	params := url.Values{
		"grant_type":            {"client_credentials"},
		"scope":                 {strings.Join(bc.Scopes, " ")},
		"client_assertion_type": {"urn:ietf:params:oauth:client-assertion-type:jwt-bearer"},
		"client_assertion":      {assertion},
	}
	return doTokenRequest(ctx, httpClient, config.TokenEndpoint, params, "")
}

// --- PKCE ---

type pkceParams struct {
	verifier  string
	challenge string
}

func generatePKCE() *pkceParams {
	b := make([]byte, 32)
	rand.Read(b)
	verifier := base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])
	return &pkceParams{verifier: verifier, challenge: challenge}
}

// --- JWT client assertion ---

func buildClientAssertion(clientID, tokenEndpoint string, privateKeyPEM []byte) (string, error) {
	block, _ := pem.Decode(privateKeyPEM)
	if block == nil {
		return "", fmt.Errorf("failed to decode PEM block")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS1
		rsaKey, err2 := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err2 != nil {
			return "", fmt.Errorf("parse private key: %w", err)
		}
		key = rsaKey
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("private key is not RSA")
	}

	now := time.Now()
	header := map[string]string{"alg": "RS384", "typ": "JWT"}
	claims := map[string]any{
		"iss": clientID,
		"sub": clientID,
		"aud": tokenEndpoint,
		"exp": now.Add(5 * time.Minute).Unix(),
		"jti": generateJTI(),
	}

	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	signingInput := headerB64 + "." + claimsB64
	h := sha256.Sum256([]byte(signingInput))
	// Use SHA-384 for RS384
	h384 := crypto.SHA384
	hasher := h384.New()
	hasher.Write([]byte(signingInput))
	hashed := hasher.Sum(nil)

	sig, err := rsa.SignPKCS1v15(rand.Reader, rsaKey, h384, hashed)
	if err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}
	_ = h // suppress unused

	sigB64 := base64.RawURLEncoding.EncodeToString(sig)
	return signingInput + "." + sigB64, nil
}

func generateJTI() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// --- Token request helper ---

func doTokenRequest(ctx context.Context, httpClient *http.Client, tokenEndpoint string, params url.Values, clientSecret string) (*Token, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", tokenEndpoint, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	if clientSecret != "" {
		req.SetBasicAuth(params.Get("client_id"), clientSecret)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("token request failed: status %d: %s", resp.StatusCode, truncate(string(body), 300))
	}

	var token Token
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}
	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	return &token, nil
}

// --- Scope helpers ---

// PatientScope creates a patient-level scope string (e.g. "patient/Patient.read").
func PatientScope(resourceType, access string) string {
	return "patient/" + resourceType + "." + access
}

// UserScope creates a user-level scope string (e.g. "user/Observation.read").
func UserScope(resourceType, access string) string {
	return "user/" + resourceType + "." + access
}

// SystemScope creates a system-level scope string (e.g. "system/*.read").
func SystemScope(resourceType, access string) string {
	return "system/" + resourceType + "." + access
}

// --- SMART middleware for FHIR client ---

// TokenSource provides tokens for authenticated FHIR requests.
type TokenSource interface {
	Token(ctx context.Context) (*Token, error)
}

// StaticTokenSource returns a fixed token. Useful for testing or short-lived scripts.
func StaticTokenSource(token *Token) TokenSource {
	return &staticTokenSource{token: token}
}

type staticTokenSource struct {
	token *Token
}

func (s *staticTokenSource) Token(_ context.Context) (*Token, error) {
	return s.token, nil
}

// RefreshingTokenSource automatically refreshes tokens when they expire.
type RefreshingTokenSource struct {
	Config       *Configuration
	ClientID     string
	ClientSecret string
	current      *Token
}

// NewRefreshingTokenSource creates a token source that auto-refreshes.
func NewRefreshingTokenSource(config *Configuration, clientID, clientSecret string, initial *Token) *RefreshingTokenSource {
	return &RefreshingTokenSource{
		Config:       config,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		current:      initial,
	}
}

// Token returns a valid token, refreshing if needed.
func (r *RefreshingTokenSource) Token(ctx context.Context) (*Token, error) {
	if r.current.Valid() {
		return r.current, nil
	}
	if r.current.RefreshToken == "" {
		return nil, fmt.Errorf("token expired and no refresh token available")
	}
	newToken, err := Refresh(ctx, r.Config, r.ClientID, r.ClientSecret, r.current.RefreshToken)
	if err != nil {
		return nil, err
	}
	r.current = newToken
	return r.current, nil
}

// --- Helpers ---

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

