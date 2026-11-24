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
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apioidc "github.com/synnaxlabs/synnax/pkg/api/auth/oidc"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

const callbackURL = "http://127.0.0.1:4242/callback"

// grantAccount registers a code for sub at the fake provider and returns credentials
// that redeem it.
func grantAccount(sub string) auth.OIDCCredentials {
	code := uuid.New().String()
	provider.Grant(code, testutil.Grant{
		Subject:     sub,
		Verifier:    "v",
		RedirectURI: callbackURL,
	})
	return auth.OIDCCredentials{
		Provider:    "idp",
		Code:        code,
		Verifier:    "v",
		RedirectURI: callbackURL,
	}
}

func linkAccount(ctx context.Context, token, sub string) oidc.Link {
	GinkgoHelper()
	return MustSucceed(linkOIDC(
		authed(ctx, token),
		apioidc.LinkRequest{Credentials: grantAccount(sub)},
	)).Link
}

var _ = Describe("ServiceConfig", func() {
	full := func() apioidc.ServiceConfig {
		return apioidc.ServiceConfig{Auth: &auth.Service{}, OIDC: &oidc.Service{}}
	}

	Describe("Validate", func() {
		It("Should succeed when every service is set", func() {
			Expect(full().Validate()).To(Succeed())
		})

		DescribeTable(
			"Should fail when a service is missing",
			func(unset func(*apioidc.ServiceConfig), msg string) {
				cfg := full()
				unset(&cfg)
				Expect(cfg.Validate()).To(MatchError(msg))
			},
			Entry(
				"auth",
				func(c *apioidc.ServiceConfig) { c.Auth = nil },
				"auth: must be non-nil",
			),
			Entry(
				"oidc",
				func(c *apioidc.ServiceConfig) { c.OIDC = nil },
				"oidc: must be non-nil",
			),
		)
	})

	Describe("Override", func() {
		It("Should take the other config's services when set", func() {
			cfg := full()
			merged := apioidc.ServiceConfig{}.Override(cfg)
			Expect(merged.Auth).To(BeIdenticalTo(cfg.Auth))
			Expect(merged.OIDC).To(BeIdenticalTo(cfg.OIDC))
		})

		It("Should keep the receiver's services when the other's are nil", func() {
			cfg := full()
			merged := cfg.Override(apioidc.ServiceConfig{})
			Expect(merged.Auth).To(BeIdenticalTo(cfg.Auth))
			Expect(merged.OIDC).To(BeIdenticalTo(cfg.OIDC))
		})
	})
})

var _ = Describe("NewService", func() {
	It("Should return an error when the config is invalid", func() {
		Expect(apioidc.NewService(apioidc.ServiceConfig{})).Error().
			To(MatchError(ContainSubstring("auth: must be non-nil")))
	})
})

var _ = Describe("Disabled OIDC method", func() {
	It("Should refuse to link a provider account", func(ctx SpecContext) {
		other := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(other.Run)
		disabled := newService(MustOpen(auth.OpenService(ctx, auth.ServiceConfig{
			DB: other,
			Authenticators: map[auth.Method]auth.Authenticator{
				auth.MethodPassword: passwordSvc,
			},
		})))
		Expect(disabled.PrepareLink(
			requestCtx(ctx),
			apioidc.LinkRequest{},
		)).Error().To(MatchError(validate.ErrValidation))
	})
})

var _ = Describe("OIDC links", func() {
	var (
		bob       user.User
		bobToken  string
		rootToken string
	)
	BeforeEach(func(ctx SpecContext) {
		bob = createUser(ctx, "bob-"+uuid.New().String(), "p")
		bobToken = login(ctx, passwordProof(bob.Name, "p"))
		rootToken = login(ctx, passwordProof(creds.Username, creds.Password))
	})

	Describe("Link", func() {
		It(
			"Should link the account to the caller so it logs in",
			func(ctx SpecContext) {
				sub := uuid.New().String()
				link := linkAccount(ctx, bobToken, sub)
				Expect(link.User).To(Equal(bob.Key))
				Expect(link.Account).To(Equal(sub))
				session, _ := MustSucceed2(authSvc.Login(ctx, auth.Proof{
					Variant: auth.OIDCProof{OIDCCredentials: grantAccount(sub)},
				}))
				Expect(session.Subject).To(Equal(bob.ResourceID()))
				Expect(session.Credential).To(Equal(link.Key))
			},
		)
		It("Should refuse an account the provider rejects", func(ctx SpecContext) {
			creds := grantAccount(uuid.New().String())
			creds.Verifier = "wrong"
			Expect(linkOIDC(
				authed(ctx, bobToken),
				apioidc.LinkRequest{Credentials: creds},
			)).Error().To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should refuse a caller that is not a user", func(ctx SpecContext) {
			r := rack.Rack{Name: "rack-" + uuid.New().String()}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return rackSvc.NewWriter(tx).Create(ctx, &r)
			})).To(Succeed())
			key := createKey(ctx, r.ResourceID())
			Expect(linkOIDC(
				authed(ctx, login(ctx, keyProof(key))),
				apioidc.LinkRequest{Credentials: grantAccount(uuid.New().String())},
			)).Error().To(MatchError(validate.ErrValidation))
			Expect(retrieveOIDCLinks(
				authed(ctx, rootToken),
				apioidc.RetrieveRequest{Subjects: []resource.ID{r.ResourceID()}},
			)).To(Equal(apioidc.RetrieveResponse{Links: []oidc.Link{}}))
		})
		It("Should refuse an account linked to another user", func(ctx SpecContext) {
			sub := uuid.New().String()
			linkAccount(ctx, bobToken, sub)
			Expect(linkOIDC(
				authed(ctx, rootToken),
				apioidc.LinkRequest{Credentials: grantAccount(sub)},
			)).Error().To(MatchError(validate.ErrValidation))
		})
	})

	Describe("Retrieve", func() {
		It("Should list the caller's links by default", func(ctx SpecContext) {
			link := linkAccount(ctx, bobToken, uuid.New().String())
			Expect(retrieveOIDCLinks(
				authed(ctx, bobToken),
				apioidc.RetrieveRequest{},
			)).To(Equal(apioidc.RetrieveResponse{Links: []oidc.Link{link}}))
		})
		It(
			"Should return an empty list for a user with no links",
			func(ctx SpecContext) {
				Expect(retrieveOIDCLinks(
					authed(ctx, bobToken),
					apioidc.RetrieveRequest{},
				)).To(Equal(apioidc.RetrieveResponse{Links: []oidc.Link{}}))
			},
		)
		It("Should let the owner list another user's links", func(ctx SpecContext) {
			link := linkAccount(ctx, bobToken, uuid.New().String())
			Expect(retrieveOIDCLinks(
				authed(ctx, rootToken),
				apioidc.RetrieveRequest{Subjects: []resource.ID{bob.ResourceID()}},
			)).To(Equal(apioidc.RetrieveResponse{Links: []oidc.Link{link}}))
		})
		It("Should deny a user listing another user's links", func(ctx SpecContext) {
			Expect(retrieveOIDCLinks(
				authed(ctx, bobToken),
				apioidc.RetrieveRequest{Subjects: []resource.ID{root.ResourceID()}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Unlink", func() {
		It("Should remove the link and end its sessions", func(ctx SpecContext) {
			sub := uuid.New().String()
			link := linkAccount(ctx, bobToken, sub)
			linkedToken := login(ctx, auth.Proof{
				Variant: auth.OIDCProof{OIDCCredentials: grantAccount(sub)},
			})
			Expect(unlinkOIDC(
				authed(ctx, bobToken),
				apioidc.UnlinkRequest{Keys: []oidc.LinkKey{link.Key}},
			)).Error().ToNot(HaveOccurred())
			Expect(oidcSvc.NewRetrieve().Where(oidc.MatchKeys(link.Key)).
				Entry(&oidc.Link{}).Exec(ctx, db)).To(MatchError(query.ErrNotFound))
			Expect(authSvc.Authenticate(ctx, linkedToken)).Error().
				To(MatchError(auth.ErrInvalidToken))
		})
		It("Should deny a user removing another user's link", func(ctx SpecContext) {
			alice := createUser(ctx, "alice-"+uuid.New().String(), "p")
			link := linkAccount(
				ctx,
				login(ctx, passwordProof(alice.Name, "p")),
				uuid.New().String(),
			)
			Expect(unlinkOIDC(
				authed(ctx, bobToken),
				apioidc.UnlinkRequest{Keys: []oidc.LinkKey{link.Key}},
			)).Error().To(MatchError(access.ErrDenied))
		})
		It("Should ignore unknown keys", func(ctx SpecContext) {
			Expect(unlinkOIDC(
				authed(ctx, bobToken),
				apioidc.UnlinkRequest{Keys: []oidc.LinkKey{uuid.New()}},
			)).Error().ToNot(HaveOccurred())
		})
	})
})
