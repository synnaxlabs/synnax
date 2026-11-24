// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
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
		apiauth.OIDCLinkRequest{Credentials: grantAccount(sub)},
	)).Link
}

var _ = Describe("OIDC links", func() {
	var (
		bob       user.User
		bobToken  string
		rootToken string
	)
	login := func(ctx context.Context, proof auth.Proof) string {
		GinkgoHelper()
		return MustSucceed(apiSvc.CreateSession(
			requestCtx(ctx),
			apiauth.SessionCreateRequest{Proof: proof},
		)).Token
	}
	BeforeEach(func(ctx SpecContext) {
		bob = createUser(ctx, "bob-"+uuid.New().String(), "p")
		bobToken = login(ctx, passwordProof(bob.Name, "p"))
		rootToken = login(ctx, passwordProof(creds.Username, creds.Password))
	})

	Describe("LinkOIDC", func() {
		It(
			"Should link the account to the caller so it logs in",
			func(ctx SpecContext) {
				sub := uuid.New().String()
				link := linkAccount(ctx, bobToken, sub)
				Expect(link.User).To(Equal(bob.Key))
				Expect(link.Account).To(Equal(sub))
				res := MustSucceed(apiSvc.CreateSession(
					requestCtx(ctx),
					apiauth.SessionCreateRequest{Proof: auth.Proof{
						Variant: auth.OIDCProof{OIDCCredentials: grantAccount(sub)},
					}},
				))
				Expect(res.Session.Subject).To(Equal(bob.ResourceID()))
				Expect(res.Session.Credential).To(Equal(link.Key))
			},
		)
		It("Should refuse an account the provider rejects", func(ctx SpecContext) {
			creds := grantAccount(uuid.New().String())
			creds.Verifier = "wrong"
			Expect(linkOIDC(
				authed(ctx, bobToken),
				apiauth.OIDCLinkRequest{Credentials: creds},
			)).Error().To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should refuse a caller that is not a user", func(ctx SpecContext) {
			r := rack.Rack{Name: "rack-" + uuid.New().String()}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return rackSvc.NewWriter(tx).Create(ctx, &r)
			})).To(Succeed())
			key := MustSucceed(createKey(
				authed(ctx, rootToken),
				apiauth.APIKeyCreateRequest{Subject: r.ResourceID(), Name: "rack"},
			))
			Expect(linkOIDC(
				authed(ctx, login(ctx, keyProof(key.Token))),
				apiauth.OIDCLinkRequest{Credentials: grantAccount(uuid.New().String())},
			)).Error().To(MatchError(validate.ErrValidation))
			Expect(retrieveOIDCLinks(
				authed(ctx, rootToken),
				apiauth.OIDCRetrieveRequest{Subjects: []resource.ID{r.ResourceID()}},
			)).To(Equal(apiauth.OIDCRetrieveResponse{Links: []oidc.Link{}}))
		})
		It("Should refuse an account linked to another user", func(ctx SpecContext) {
			sub := uuid.New().String()
			linkAccount(ctx, bobToken, sub)
			Expect(linkOIDC(
				authed(ctx, rootToken),
				apiauth.OIDCLinkRequest{Credentials: grantAccount(sub)},
			)).Error().To(MatchError(validate.ErrValidation))
		})
	})

	Describe("RetrieveOIDCLinks", func() {
		It("Should list the caller's links by default", func(ctx SpecContext) {
			link := linkAccount(ctx, bobToken, uuid.New().String())
			Expect(retrieveOIDCLinks(
				authed(ctx, bobToken),
				apiauth.OIDCRetrieveRequest{},
			)).To(Equal(apiauth.OIDCRetrieveResponse{Links: []oidc.Link{link}}))
		})
		It(
			"Should return an empty list for a user with no links",
			func(ctx SpecContext) {
				Expect(retrieveOIDCLinks(
					authed(ctx, bobToken),
					apiauth.OIDCRetrieveRequest{},
				)).To(Equal(apiauth.OIDCRetrieveResponse{Links: []oidc.Link{}}))
			},
		)
		It("Should let the owner list another user's links", func(ctx SpecContext) {
			link := linkAccount(ctx, bobToken, uuid.New().String())
			Expect(retrieveOIDCLinks(
				authed(ctx, rootToken),
				apiauth.OIDCRetrieveRequest{Subjects: []resource.ID{bob.ResourceID()}},
			)).To(Equal(apiauth.OIDCRetrieveResponse{Links: []oidc.Link{link}}))
		})
		It("Should deny a user listing another user's links", func(ctx SpecContext) {
			Expect(retrieveOIDCLinks(
				authed(ctx, bobToken),
				apiauth.OIDCRetrieveRequest{Subjects: []resource.ID{root.ResourceID()}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("UnlinkOIDC", func() {
		It("Should remove the link and end its sessions", func(ctx SpecContext) {
			sub := uuid.New().String()
			link := linkAccount(ctx, bobToken, sub)
			linkedToken := login(ctx, auth.Proof{
				Variant: auth.OIDCProof{OIDCCredentials: grantAccount(sub)},
			})
			Expect(unlinkOIDC(
				authed(ctx, bobToken),
				apiauth.OIDCUnlinkRequest{Keys: []oidc.LinkKey{link.Key}},
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
				apiauth.OIDCUnlinkRequest{Keys: []oidc.LinkKey{link.Key}},
			)).Error().To(MatchError(access.ErrDenied))
		})
		It("Should ignore unknown keys", func(ctx SpecContext) {
			Expect(unlinkOIDC(
				authed(ctx, bobToken),
				apiauth.OIDCUnlinkRequest{Keys: []oidc.LinkKey{uuid.New()}},
			)).Error().ToNot(HaveOccurred())
		})
	})
})
