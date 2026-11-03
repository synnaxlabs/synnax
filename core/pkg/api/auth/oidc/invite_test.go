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
	"encoding/json"
	"sync"
	"sync/atomic"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	apioidc "github.com/synnaxlabs/synnax/pkg/api/auth/oidc"
	"github.com/synnaxlabs/synnax/pkg/api/auth/session"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc/invite"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func redeem(
	ctx context.Context,
	token string,
	creds auth.OIDCCredentials,
) (session.CreateResponse, error) {
	return apiSvc.RedeemInvite(
		requestCtx(ctx),
		apioidc.RedeemInviteRequest{Token: token, Credentials: creds},
	)
}

func oidcProof(creds auth.OIDCCredentials) auth.Proof {
	return auth.Proof{Variant: auth.OIDCProof{OIDCCredentials: creds}}
}

var _ = Describe("Disabled OIDC method for invites", func() {
	var disabled *apioidc.Service
	BeforeEach(func(ctx SpecContext) {
		disabled = newService(MustOpen(auth.OpenService(ctx, auth.ServiceConfig{
			DB: node.DB,
			Authenticators: map[auth.Method]auth.Authenticator{
				auth.MethodPassword: passwordSvc,
			},
		})))
	})
	It("Should refuse to create an invite", func(ctx SpecContext) {
		Expect(disabled.CreateInvite(
			requestCtx(ctx),
			nil,
			apioidc.CreateInviteRequest{},
		)).Error().To(MatchError(validate.ErrValidation))
	})
	It("Should refuse to retrieve invites", func(ctx SpecContext) {
		Expect(disabled.RetrieveInvites(
			requestCtx(ctx),
			apioidc.RetrieveInvitesRequest{},
		)).Error().To(MatchError(validate.ErrValidation))
	})
	It("Should refuse to delete invites", func(ctx SpecContext) {
		Expect(disabled.DeleteInvites(
			requestCtx(ctx),
			nil,
			apioidc.DeleteInvitesRequest{},
		)).Error().To(MatchError(validate.ErrValidation))
	})
	It("Should refuse to redeem an invite", func(ctx SpecContext) {
		Expect(disabled.RedeemInvite(
			requestCtx(ctx),
			apioidc.RedeemInviteRequest{},
		)).Error().To(MatchError(validate.ErrValidation))
	})
})

var _ = Describe("OIDC invites", func() {
	var (
		carol     user.User
		subject   ontology.ID
		bobToken  string
		rootToken string
	)
	create := func(ctx SpecContext) apioidc.CreateInviteResponse {
		GinkgoHelper()
		return MustSucceed(apiSvc.CreateInvite(
			authed(ctx, rootToken),
			nil,
			apioidc.CreateInviteRequest{Subject: subject},
		))
	}
	retrieve := func(ctx SpecContext) []invite.Invite {
		GinkgoHelper()
		return MustSucceed(apiSvc.RetrieveInvites(
			authed(ctx, rootToken),
			apioidc.RetrieveInvitesRequest{Subjects: []ontology.ID{subject}},
		)).Invites
	}
	BeforeEach(func(ctx SpecContext) {
		carol = MustSucceed(
			userSvc.NewWriter(nil).Create(ctx, user.User{Name: "Carol"}),
		)
		subject = carol.OntologyID()
		bob := createUser(ctx, "bob-"+uuid.New().String(), "p")
		bobToken = login(ctx, passwordProof(bob.Name, "p"))
		rootToken = login(ctx, passwordProof(creds.Username, creds.Password))
	})

	Describe("CreateInvite", func() {
		It(
			"Should return a token that only a hash of is stored",
			func(ctx SpecContext) {
				res := create(ctx)
				Expect(res.Token).To(HavePrefix("syi_"))
				Expect(res.Invite.Subject).To(Equal(subject))
				Expect(string(res.Invite.Hash)).ToNot(ContainSubstring(res.Token))
			},
		)
		It("Should replace the user's invite", func(ctx SpecContext) {
			old := create(ctx)
			current := create(ctx)
			Expect(retrieve(ctx)).To(HaveLen(1))
			Expect(redeem(ctx, old.Token, grantAccount(uuid.New().String()))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
			MustSucceed(redeem(ctx, current.Token, grantAccount(uuid.New().String())))
		})
		It("Should deny a caller that may not update the user", func(ctx SpecContext) {
			Expect(apiSvc.CreateInvite(
				authed(ctx, bobToken),
				nil,
				apioidc.CreateInviteRequest{Subject: subject},
			)).Error().To(MatchError(access.ErrDenied))
		})
		It("Should deny a caller inviting itself without the permission", func(
			ctx SpecContext,
		) {
			caller := authed(ctx, bobToken)
			Expect(apiSvc.CreateInvite(caller, nil, apioidc.CreateInviteRequest{
				Subject: apiauth.GetSubject(caller),
			})).Error().To(MatchError(access.ErrDenied))
		})
		It("Should refuse a rack", func(ctx SpecContext) {
			Expect(apiSvc.CreateInvite(
				authed(ctx, rootToken),
				nil,
				apioidc.CreateInviteRequest{Subject: ontology.ID{
					Type: ontology.ResourceTypeRack,
					Key:  uuid.New().String(),
				}},
			)).Error().To(MatchError(validate.ErrValidation))
		})
		It("Should not find an unknown user", func(ctx SpecContext) {
			Expect(apiSvc.CreateInvite(
				authed(ctx, rootToken),
				nil,
				apioidc.CreateInviteRequest{Subject: user.OntologyID(uuid.New())},
			)).Error().To(MatchError(query.ErrNotFound))
		})
	})

	Describe("RetrieveInvites", func() {
		It("Should list live invites without a token or a hash", func(ctx SpecContext) {
			res := create(ctx)
			invites := retrieve(ctx)
			Expect(invites).To(HaveLen(1))
			Expect(invites[0].Key).To(Equal(res.Invite.Key))
			Expect(invites[0].Subject).To(Equal(subject))
			Expect(invites[0].Expiry).To(Equal(res.Invite.Expiry))
			body := MustSucceed(json.Marshal(apioidc.RetrieveInvitesResponse{
				Invites: invites,
			}))
			Expect(string(body)).ToNot(ContainSubstring(res.Token))
			Expect(string(body)).ToNot(ContainSubstring("hash"))
		})
		It("Should leave out an expired invite", func(ctx SpecContext) {
			create(ctx)
			clock.Advance(invite.Lifetime)
			Expect(retrieve(ctx)).To(BeEmpty())
		})
		It("Should list nothing for no subjects", func(ctx SpecContext) {
			create(ctx)
			Expect(apiSvc.RetrieveInvites(
				authed(ctx, rootToken),
				apioidc.RetrieveInvitesRequest{},
			)).To(Equal(apioidc.RetrieveInvitesResponse{Invites: []invite.Invite{}}))
		})
		It(
			"Should deny a caller that may not retrieve the user",
			func(ctx SpecContext) {
				Expect(apiSvc.RetrieveInvites(
					authed(ctx, bobToken),
					apioidc.RetrieveInvitesRequest{
						Subjects: []ontology.ID{root.OntologyID()},
					},
				)).Error().To(MatchError(access.ErrDenied))
			},
		)
	})

	Describe("DeleteInvites", func() {
		It("Should revoke the invite", func(ctx SpecContext) {
			res := create(ctx)
			Expect(apiSvc.DeleteInvites(
				authed(ctx, rootToken),
				nil,
				apioidc.DeleteInvitesRequest{Keys: []invite.Key{res.Invite.Key}},
			)).Error().ToNot(HaveOccurred())
			Expect(retrieve(ctx)).To(BeEmpty())
			Expect(redeem(ctx, res.Token, grantAccount(uuid.New().String()))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should deny a caller that may not update the user", func(ctx SpecContext) {
			res := create(ctx)
			Expect(apiSvc.DeleteInvites(
				authed(ctx, bobToken),
				nil,
				apioidc.DeleteInvitesRequest{Keys: []invite.Key{res.Invite.Key}},
			)).Error().To(MatchError(access.ErrDenied))
		})
		It("Should ignore unknown keys", func(ctx SpecContext) {
			Expect(apiSvc.DeleteInvites(
				authed(ctx, rootToken),
				nil,
				apioidc.DeleteInvitesRequest{Keys: []invite.Key{uuid.New()}},
			)).Error().ToNot(HaveOccurred())
		})
	})

	Describe("RedeemInvite", func() {
		It("Should link the account and open a session", func(ctx SpecContext) {
			sub := uuid.New().String()
			res := MustSucceed(redeem(ctx, create(ctx).Token, grantAccount(sub)))
			Expect(res.Session.Subject).To(Equal(subject))
			Expect(res.User.Key).To(Equal(carol.Key))
			Expect(MustSucceed(authSvc.Authenticate(ctx, res.Token)).Subject).
				To(Equal(subject))
			Expect(retrieve(ctx)).To(BeEmpty())
			next := login(ctx, oidcProof(grantAccount(sub)))
			Expect(
				MustSucceed(authSvc.Authenticate(ctx, next)).Subject,
			).To(Equal(subject))
		})
		It(
			"Should bring in an account that user creation refuses",
			func(ctx SpecContext) {
				closed := func(sub string) auth.OIDCCredentials {
					c := grantAccount(sub)
					c.Provider = "closed"
					return c
				}
				sub := uuid.New().String()
				Expect(authSvc.Login(ctx, oidcProof(closed(sub)))).Error().
					To(MatchError(auth.ErrInvalidCredentials))
				res := MustSucceed(redeem(ctx, create(ctx).Token, closed(sub)))
				Expect(res.Session.Subject).To(Equal(subject))
				next := login(ctx, oidcProof(closed(sub)))
				Expect(
					MustSucceed(authSvc.Authenticate(ctx, next)).Subject,
				).To(Equal(subject))
			},
		)
		It("Should put the session token in a cookie", func(ctx SpecContext) {
			token := create(ctx).Token
			fctx := requestCtx(ctx)
			res := MustSucceed(apiSvc.RedeemInvite(fctx, apioidc.RedeemInviteRequest{
				Token:       token,
				Credentials: grantAccount(uuid.New().String()),
				Cookie:      true,
			}))
			Expect(res.Token).To(BeEmpty())
			cookie := MustBeOk(fctx.Get(apiauth.SessionCookieParam)).(apiauth.SessionCookie)
			Expect(MustSucceed(authSvc.Authenticate(ctx, cookie.Token)).Subject).
				To(Equal(subject))
		})
		It("Should work once", func(ctx SpecContext) {
			token := create(ctx).Token
			MustSucceed(redeem(ctx, token, grantAccount(uuid.New().String())))
			Expect(redeem(ctx, token, grantAccount(uuid.New().String()))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
			Expect(MustSucceed(oidcSvc.NewRetrieve().
				Where(oidc.MatchSubjects(subject)).
				Count(ctx, nil))).To(Equal(1))
		})
		It("Should link one account when redeemed at the same time", func(
			ctx SpecContext,
		) {
			token := create(ctx).Token
			var (
				wg       sync.WaitGroup
				redeemed atomic.Int32
			)
			for range 8 {
				account := grantAccount(uuid.New().String())
				wg.Go(func() {
					if _, err := redeem(ctx, token, account); err == nil {
						redeemed.Add(1)
					}
				})
			}
			wg.Wait()
			Expect(redeemed.Load()).To(Equal(int32(1)))
			Expect(MustSucceed(oidcSvc.NewRetrieve().
				Where(oidc.MatchSubjects(subject)).
				Count(ctx, nil))).To(Equal(1))
		})
		It("Should give one error for wrong and expired tokens", func(ctx SpecContext) {
			expired := create(ctx).Token
			clock.Advance(invite.Lifetime)
			_, expiredErr := redeem(ctx, expired, grantAccount(uuid.New().String()))
			Expect(expiredErr).To(MatchError(auth.ErrInvalidCredentials))
			_, wrongErr := redeem(ctx, "syi_wrong", grantAccount(uuid.New().String()))
			Expect(wrongErr).To(BeIdenticalTo(expiredErr))
		})
		It("Should refuse an account the provider rejects", func(ctx SpecContext) {
			res := create(ctx)
			creds := grantAccount(uuid.New().String())
			creds.Verifier = "wrong"
			Expect(redeem(ctx, res.Token, creds)).Error().
				To(MatchError(auth.ErrInvalidCredentials))
			Expect(retrieve(ctx)).To(HaveLen(1))
		})
		It("Should refuse a linked account and keep the invite", func(ctx SpecContext) {
			sub := uuid.New().String()
			linkAccount(ctx, bobToken, sub)
			Expect(redeem(ctx, create(ctx).Token, grantAccount(sub))).Error().
				To(MatchError(validate.ErrValidation))
			Expect(retrieve(ctx)).To(HaveLen(1))
		})
		It("Should refuse a user linked to the provider", func(ctx SpecContext) {
			MustSucceed(oidcSvc.NewWriter(nil).Create(ctx, subject, oidc.Account{
				Provider: "idp",
				ID:       uuid.New().String(),
			}))
			Expect(redeem(ctx, create(ctx).Token, grantAccount(uuid.New().String()))).
				Error().To(MatchError(validate.ErrValidation))
		})
	})

	It("Should delete a user's invites with the user", func(ctx SpecContext) {
		create(ctx)
		Expect(authSvc.NewWriter(nil).DeleteFor(ctx, subject)).To(Succeed())
		Expect(retrieve(ctx)).To(BeEmpty())
	})
})
