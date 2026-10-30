// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package oidc_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func TestOIDC(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Auth OIDC Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db       *gorp.DB
	otg      *ontology.Ontology
	userSvc  *user.Service
	rbacSvc  *rbac.Service
	provider *fakeProvider
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: db, Ontology: otg,
	}))
	userSvc = MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB: db, Ontology: otg, Group: groupSvc,
	}))
	rbacSvc = MustOpen(rbac.OpenService(ctx, rbac.ServiceConfig{
		DB: db, Ontology: otg, Group: groupSvc, User: userSvc,
	}))
	provider = newFakeProvider()
	DeferCleanup(provider.server.Close)
})

const clientID = "synnax"

// grant is what the fake provider issues for one authorization code.
type grant struct {
	sub       string
	name      string
	verifier  string
	redirect  string
	audience  string
	noIDToken bool
}

// fakeProvider is an OpenID Connect provider that signs ID tokens with its own key and
// redeems the codes a spec registers.
type fakeProvider struct {
	server *httptest.Server
	key    *rsa.PrivateKey
	mu     sync.Mutex
	grants map[string]grant
}

func newFakeProvider() *fakeProvider {
	GinkgoHelper()
	p := &fakeProvider{
		key:    MustSucceed(rsa.GenerateKey(rand.Reader, 2048)),
		grants: make(map[string]grant),
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

func (p *fakeProvider) issuer() string { return p.server.URL }

func (p *fakeProvider) grant(code string, g grant) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.grants[code] = g
}

func (p *fakeProvider) discovery(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                p.issuer(),
		"authorization_endpoint":                p.issuer() + "/authorize",
		"token_endpoint":                        p.issuer() + "/token",
		"jwks_uri":                              p.issuer() + "/keys",
		"id_token_signing_alg_values_supported": []string{"RS256"},
	})
}

func (p *fakeProvider) keys(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
		Key:       &p.key.PublicKey,
		KeyID:     "key",
		Algorithm: string(jose.RS256),
		Use:       "sig",
	}}})
}

func (p *fakeProvider) token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{"error": "invalid_request"},
		)
		return
	}
	p.mu.Lock()
	g, ok := p.grants[r.Form.Get("code")]
	p.mu.Unlock()
	if !ok ||
		g.verifier != r.Form.Get("code_verifier") ||
		g.redirect != r.Form.Get("redirect_uri") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant"})
		return
	}
	res := map[string]any{"access_token": "access", "token_type": "Bearer"}
	if !g.noIDToken {
		res["id_token"] = p.sign(g)
	}
	writeJSON(w, http.StatusOK, res)
}

func (p *fakeProvider) sign(g grant) string {
	GinkgoHelper()
	audience := g.audience
	if audience == "" {
		audience = clientID
	}
	signer := MustSucceed(jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: p.key},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "key"),
	))
	now := time.Now()
	claims := MustSucceed(json.Marshal(map[string]any{
		"iss":  p.issuer(),
		"sub":  g.sub,
		"aud":  audience,
		"iat":  now.Unix(),
		"exp":  now.Add(time.Hour).Unix(),
		"name": g.name,
	}))
	return MustSucceed(MustSucceed(signer.Sign(claims)).CompactSerialize())
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// a broken response fails the spec that reads it
	if err := json.NewEncoder(w).Encode(v); err != nil {
		panic(err)
	}
}
