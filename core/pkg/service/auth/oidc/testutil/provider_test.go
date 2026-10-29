// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-jose/go-jose/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

func redeem(p *testutil.Provider, form url.Values) (int, map[string]any) {
	GinkgoHelper()
	res := MustSucceed(http.Post(
		p.Issuer()+"/token",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	))
	defer func() { Expect(res.Body.Close()).To(Succeed()) }()
	var body map[string]any
	Expect(json.NewDecoder(res.Body).Decode(&body)).To(Succeed())
	return res.StatusCode, body
}

// claims verifies an ID token against the provider's published keys and returns its
// claims.
func claims(p *testutil.Provider, idToken string) map[string]any {
	GinkgoHelper()
	res := MustSucceed(http.Get(p.Issuer() + "/keys"))
	defer func() { Expect(res.Body.Close()).To(Succeed()) }()
	var keys jose.JSONWebKeySet
	Expect(json.NewDecoder(res.Body).Decode(&keys)).To(Succeed())
	signed := MustSucceed(jose.ParseSigned(
		idToken,
		[]jose.SignatureAlgorithm{jose.RS256},
	))
	key := keys.Key(signed.Signatures[0].Header.KeyID)
	Expect(key).To(HaveLen(1))
	var c map[string]any
	Expect(json.Unmarshal(MustSucceed(signed.Verify(key[0])), &c)).To(Succeed())
	return c
}

var _ = Describe("Provider", func() {
	var p *testutil.Provider
	BeforeEach(func() {
		p = testutil.NewProvider()
		DeferCleanup(p.Close)
	})
	It("Should serve a discovery document for its issuer", func() {
		res := MustSucceed(http.Get(p.Issuer() + "/.well-known/openid-configuration"))
		defer func() { Expect(res.Body.Close()).To(Succeed()) }()
		var doc map[string]any
		Expect(json.NewDecoder(res.Body).Decode(&doc)).To(Succeed())
		Expect(doc).To(HaveKeyWithValue("issuer", p.Issuer()))
		Expect(doc).To(HaveKeyWithValue("token_endpoint", p.Issuer()+"/token"))
	})
	It("Should redeem a granted code for an ID token", func() {
		p.Grant("code", testutil.Grant{Subject: "ada", Verifier: "v", RedirectURI: "r"})
		status, body := redeem(p, url.Values{
			"code": {"code"}, "code_verifier": {"v"}, "redirect_uri": {"r"},
		})
		Expect(status).To(Equal(http.StatusOK))
		c := claims(p, body["id_token"].(string))
		Expect(c).To(HaveKeyWithValue("iss", p.Issuer()))
		Expect(c).To(HaveKeyWithValue("sub", "ada"))
		Expect(c).To(HaveKeyWithValue("aud", testutil.ClientID))
	})
	It("Should issue the ID token to the audience the grant names", func() {
		p.Grant(
			"code",
			testutil.Grant{Verifier: "v", RedirectURI: "r", Audience: "other"},
		)
		_, body := redeem(p, url.Values{
			"code": {"code"}, "code_verifier": {"v"}, "redirect_uri": {"r"},
		})
		Expect(claims(p, body["id_token"].(string))).
			To(HaveKeyWithValue("aud", "other"))
	})
	It("Should leave the ID token out when the grant says so", func() {
		p.Grant(
			"code",
			testutil.Grant{Verifier: "v", RedirectURI: "r", IDTokenOmitted: true},
		)
		_, body := redeem(p, url.Values{
			"code": {"code"}, "code_verifier": {"v"}, "redirect_uri": {"r"},
		})
		Expect(body).ToNot(HaveKey("id_token"))
	})
	It("Should refuse a code with the wrong verifier", func() {
		p.Grant("code", testutil.Grant{Verifier: "v", RedirectURI: "r"})
		status, body := redeem(p, url.Values{
			"code": {"code"}, "code_verifier": {"x"}, "redirect_uri": {"r"},
		})
		Expect(status).To(Equal(http.StatusBadRequest))
		Expect(body).To(HaveKeyWithValue("error", "invalid_grant"))
	})
})
