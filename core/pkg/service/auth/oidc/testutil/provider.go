// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil runs a fake OpenID Connect provider for specs.
package testutil

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/samber/lo"
)

// ClientID is the client ID the fake provider issues ID tokens to.
const ClientID = "synnax"

// Grant is what the fake provider issues for one authorization code.
type Grant struct {
	// Subject is the ID token's `sub` claim.
	Subject string
	// Name is the ID token's `name` claim.
	Name string
	// Verifier is the PKCE code verifier the token request must present.
	Verifier string
	// RedirectURI is the redirect URI the token request must present.
	RedirectURI string
	// Audience overrides the ID token's audience. Empty means [ClientID].
	Audience string
	// IDTokenOmitted leaves the ID token out of the token response.
	IDTokenOmitted bool
}

// Provider is an OpenID Connect provider that signs ID tokens with its own key and
// redeems the codes a spec grants. It serves plain HTTP and closes every connection
// after one request, so it leaves no idle goroutines behind.
type Provider struct {
	server *httptest.Server
	key    *rsa.PrivateKey
	mu     sync.Mutex
	grants map[string]Grant
}

// NewProvider starts a Provider. Close it when done.
func NewProvider() *Provider {
	p := &Provider{
		key:    lo.Must(rsa.GenerateKey(rand.Reader, 2048)),
		grants: make(map[string]Grant),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/openid-configuration", p.discovery)
	mux.HandleFunc("GET /keys", p.keys)
	mux.HandleFunc("POST /token", p.token)
	p.server = httptest.NewUnstartedServer(mux)
	p.server.Config.SetKeepAlivesEnabled(false)
	p.server.Start()
	return p
}

// Issuer returns the provider's issuer URL.
func (p *Provider) Issuer() string { return p.server.URL }

// Grant registers code so that the token endpoint redeems it for g.
func (p *Provider) Grant(code string, g Grant) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.grants[code] = g
}

// Close stops the provider.
func (p *Provider) Close() { p.server.Close() }

func (p *Provider) discovery(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                p.Issuer(),
		"authorization_endpoint":                p.Issuer() + "/authorize",
		"token_endpoint":                        p.Issuer() + "/token",
		"jwks_uri":                              p.Issuer() + "/keys",
		"id_token_signing_alg_values_supported": []string{string(jose.RS256)},
	})
}

func (p *Provider) keys(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
		Key:       &p.key.PublicKey,
		KeyID:     "key",
		Algorithm: string(jose.RS256),
		Use:       "sig",
	}}})
}

func (p *Provider) token(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	g, ok := p.grants[r.PostFormValue("code")]
	p.mu.Unlock()
	if !ok ||
		g.Verifier != r.PostFormValue("code_verifier") ||
		g.RedirectURI != r.PostFormValue("redirect_uri") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant"})
		return
	}
	res := map[string]any{"access_token": "access", "token_type": "Bearer"}
	if !g.IDTokenOmitted {
		res["id_token"] = p.sign(g)
	}
	writeJSON(w, http.StatusOK, res)
}

func (p *Provider) sign(g Grant) string {
	audience := g.Audience
	if audience == "" {
		audience = ClientID
	}
	signer := lo.Must(jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: p.key},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "key"),
	))
	now := time.Now()
	claims := lo.Must(json.Marshal(map[string]any{
		"iss":  p.Issuer(),
		"sub":  g.Subject,
		"aud":  audience,
		"iat":  now.Unix(),
		"exp":  now.Add(time.Hour).Unix(),
		"name": g.Name,
	}))
	return lo.Must(lo.Must(signer.Sign(claims)).CompactSerialize())
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// a broken response fails the spec that reads it
	if err := json.NewEncoder(w).Encode(v); err != nil {
		panic(err)
	}
}
