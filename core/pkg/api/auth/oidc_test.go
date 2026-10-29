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
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

const oidcRedirect = "http://127.0.0.1:4242/callback"

// grantAccount registers a code for sub at the fake provider and returns credentials
// that redeem it.
func grantAccount(sub string) auth.OIDCCredentials {
	code := uuid.New().String()
	provider.Grant(code, testutil.Grant{
		Subject:     sub,
		Verifier:    "v",
		RedirectURI: oidcRedirect,
	})
	return auth.OIDCCredentials{
		Provider:    "idp",
		Code:        code,
		Verifier:    "v",
		RedirectURI: oidcRedirect,
	}
}

func linkAccount(ctx context.Context, token, sub string) oidc.Link {
	GinkgoHelper()
	return MustSucceed(apiSvc.LinkOIDC(
		authed(ctx, token),
		nil,
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
				Expect(link.Subject).To(Equal(bob.OntologyID()))
				Expect(link.Account).To(Equal(sub))
				res := MustSucceed(apiSvc.CreateSession(
					requestCtx(ctx),
					apiauth.SessionCreateRequest{Proof: auth.Proof{
						Variant: auth.OIDCProof{OIDCCredentials: grantAccount(sub)},
					}},
				))
				Expect(res.Session.Subject).To(Equal(bob.OntologyID()))
				Expect(res.Session.Credential).To(Equal(link.Key))
			},
		)
		It("Should refuse an account the provider rejects", func(ctx SpecContext) {
			creds := grantAccount(uuid.New().String())
			creds.Verifier = "wrong"
			Expect(apiSvc.LinkOIDC(
				authed(ctx, bobToken),
				nil,
				apiauth.OIDCLinkRequest{Credentials: creds},
			)).Error().To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should refuse a caller that is not a user", func(ctx SpecContext) {
			rack := ontology.ID{
				Type: ontology.ResourceTypeRack,
				Key:  uuid.New().String(),
			}
			Expect(otg.NewWriter(nil).DefineResources(ctx, rack)).To(Succeed())
			key := MustSucceed(apiSvc.CreateAPIKey(
				authed(ctx, rootToken),
				nil,
				apiauth.APIKeyCreateRequest{Subject: rack, Name: "rack"},
			))
			Expect(apiSvc.LinkOIDC(
				authed(ctx, login(ctx, keyProof(key.Token))),
				nil,
				apiauth.OIDCLinkRequest{Credentials: grantAccount(uuid.New().String())},
			)).Error().To(MatchError(validate.ErrValidation))
			Expect(apiSvc.RetrieveOIDCLinks(
				authed(ctx, rootToken),
				apiauth.OIDCRetrieveRequest{Subjects: []ontology.ID{rack}},
			)).To(Equal(apiauth.OIDCRetrieveResponse{Links: []oidc.Link{}}))
		})
		It("Should refuse an account linked to another user", func(ctx SpecContext) {
			sub := uuid.New().String()
			linkAccount(ctx, bobToken, sub)
			Expect(apiSvc.LinkOIDC(
				authed(ctx, rootToken),
				nil,
				apiauth.OIDCLinkRequest{Credentials: grantAccount(sub)},
			)).Error().To(MatchError(validate.ErrValidation))
		})
	})

	Describe("RetrieveOIDCLinks", func() {
		It("Should list the caller's links by default", func(ctx SpecContext) {
			link := linkAccount(ctx, bobToken, uuid.New().String())
			Expect(apiSvc.RetrieveOIDCLinks(
				authed(ctx, bobToken),
				apiauth.OIDCRetrieveRequest{},
			)).To(Equal(apiauth.OIDCRetrieveResponse{Links: []oidc.Link{link}}))
		})
		It(
			"Should return an empty list for a user with no links",
			func(ctx SpecContext) {
				Expect(apiSvc.RetrieveOIDCLinks(
					authed(ctx, bobToken),
					apiauth.OIDCRetrieveRequest{},
				)).To(Equal(apiauth.OIDCRetrieveResponse{Links: []oidc.Link{}}))
			},
		)
		It("Should let the owner list another user's links", func(ctx SpecContext) {
			link := linkAccount(ctx, bobToken, uuid.New().String())
			Expect(apiSvc.RetrieveOIDCLinks(
				authed(ctx, rootToken),
				apiauth.OIDCRetrieveRequest{Subjects: []ontology.ID{bob.OntologyID()}},
			)).To(Equal(apiauth.OIDCRetrieveResponse{Links: []oidc.Link{link}}))
		})
		It("Should deny a user listing another user's links", func(ctx SpecContext) {
			Expect(apiSvc.RetrieveOIDCLinks(
				authed(ctx, bobToken),
				apiauth.OIDCRetrieveRequest{Subjects: []ontology.ID{root.OntologyID()}},
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
			Expect(apiSvc.UnlinkOIDC(
				authed(ctx, bobToken),
				nil,
				apiauth.OIDCUnlinkRequest{Keys: []oidc.Key{link.Key}},
			)).Error().ToNot(HaveOccurred())
			Expect(oidcSvc.NewRetrieve().Where(oidc.MatchKeys(link.Key)).
				Entry(&oidc.Link{}).Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
			Expect(authSvc.Authenticate(ctx, linkedToken)).Error().
				To(MatchError(auth.ErrInvalidToken))
		})
		It("Should deny a user removing another user's link", func(ctx SpecContext) {
			link := linkAccount(ctx, rootToken, uuid.New().String())
			Expect(apiSvc.UnlinkOIDC(
				authed(ctx, bobToken),
				nil,
				apiauth.OIDCUnlinkRequest{Keys: []oidc.Key{link.Key}},
			)).Error().To(MatchError(access.ErrDenied))
		})
		It("Should ignore unknown keys", func(ctx SpecContext) {
			Expect(apiSvc.UnlinkOIDC(
				authed(ctx, bobToken),
				nil,
				apiauth.OIDCUnlinkRequest{Keys: []oidc.Key{uuid.New()}},
			)).Error().ToNot(HaveOccurred())
		})
	})
})
