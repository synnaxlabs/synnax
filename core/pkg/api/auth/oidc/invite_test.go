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
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	apioidc "github.com/synnaxlabs/synnax/pkg/api/auth/oidc"
	"github.com/synnaxlabs/synnax/pkg/api/auth/session"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc/invite"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
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
		other := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(other.Run)
		disabled = newService(MustOpen(auth.OpenService(ctx, auth.ServiceConfig{
			DB: other,
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
		bob       user.User
		bobToken  string
		rootToken string
	)
	create := func(ctx SpecContext) apioidc.CreateInviteResponse {
		GinkgoHelper()
		return MustSucceed(createInvite(
			authed(ctx, rootToken),
			apioidc.CreateInviteRequest{User: carol.Key},
		))
	}
	retrieve := func(ctx SpecContext) []invite.Invite {
		GinkgoHelper()
		return MustSucceed(retrieveInvites(
			authed(ctx, rootToken),
			apioidc.RetrieveInvitesRequest{
				Subjects: []resource.ID{carol.ResourceID()},
			},
		)).Invites
	}
	countLinks := func(ctx SpecContext) int {
		GinkgoHelper()
		return MustSucceed(oidcSvc.NewRetrieve().
			Where(oidc.MatchUsers(carol.Key)).
			Count(ctx, db))
	}
	BeforeEach(func(ctx SpecContext) {
		carol = createUser(ctx, "carol-"+uuid.New().String())
		bob = createUser(ctx, "bob-"+uuid.New().String())
		bobToken = login(ctx, passwordProof(bob.Name, "p"))
		rootToken = login(ctx, passwordProof(creds.Username, creds.Password))
	})

	Describe("CreateInvite", func() {
		It(
			"Should return a token that only a hash of is stored",
			func(ctx SpecContext) {
				res := create(ctx)
				Expect(res.Token).To(HavePrefix("syi_"))
				Expect(res.Invite.User).To(Equal(carol.Key))
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
		It("Should let a user invite itself", func(ctx SpecContext) {
			Expect(MustSucceed(createInvite(
				authed(ctx, bobToken),
				apioidc.CreateInviteRequest{User: bob.Key},
			)).Invite.User).To(Equal(bob.Key))
		})
		It("Should deny a user inviting another user", func(ctx SpecContext) {
			Expect(createInvite(
				authed(ctx, bobToken),
				apioidc.CreateInviteRequest{User: carol.Key},
			)).Error().To(MatchError(access.ErrDenied))
		})
		It("Should refuse an unknown user", func(ctx SpecContext) {
			Expect(createInvite(
				authed(ctx, rootToken),
				apioidc.CreateInviteRequest{User: uuid.New()},
			)).Error().To(MatchError(validate.ErrValidation))
		})
	})

	Describe("RetrieveInvites", func() {
		It("Should list live invites without a token or a hash", func(ctx SpecContext) {
			res := create(ctx)
			invites := retrieve(ctx)
			Expect(invites).To(HaveLen(1))
			Expect(invites[0].Key).To(Equal(res.Invite.Key))
			Expect(invites[0].User).To(Equal(carol.Key))
			Expect(invites[0].Expiry).To(Equal(res.Invite.Expiry))
			body := MustSucceed(json.Marshal(apioidc.RetrieveInvitesResponse{
				Invites: invites,
			}))
			Expect(string(body)).ToNot(ContainSubstring(res.Token))
			Expect(string(body)).ToNot(ContainSubstring("hash"))
		})
		It("Should leave out an expired invite", func(ctx SpecContext) {
			create(ctx)
			clock.Advance(24 * time.Hour)
			Expect(retrieve(ctx)).To(BeEmpty())
		})
		It("Should list the caller's invite by default", func(ctx SpecContext) {
			res := MustSucceed(createInvite(
				authed(ctx, bobToken),
				apioidc.CreateInviteRequest{User: bob.Key},
			))
			Expect(retrieveInvites(
				authed(ctx, bobToken),
				apioidc.RetrieveInvitesRequest{},
			)).To(Equal(apioidc.RetrieveInvitesResponse{
				Invites: []invite.Invite{res.Invite},
			}))
		})
		It("Should deny a user listing another user's invites", func(ctx SpecContext) {
			Expect(retrieveInvites(
				authed(ctx, bobToken),
				apioidc.RetrieveInvitesRequest{
					Subjects: []resource.ID{root.ResourceID()},
				},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("DeleteInvites", func() {
		It("Should revoke the invite", func(ctx SpecContext) {
			res := create(ctx)
			Expect(deleteInvites(
				authed(ctx, rootToken),
				apioidc.DeleteInvitesRequest{Keys: []invite.Key{res.Invite.Key}},
			)).Error().ToNot(HaveOccurred())
			Expect(retrieve(ctx)).To(BeEmpty())
			Expect(redeem(ctx, res.Token, grantAccount(uuid.New().String()))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should deny a user revoking another user's invite", func(ctx SpecContext) {
			res := create(ctx)
			Expect(deleteInvites(
				authed(ctx, bobToken),
				apioidc.DeleteInvitesRequest{Keys: []invite.Key{res.Invite.Key}},
			)).Error().To(MatchError(access.ErrDenied))
		})
		It("Should ignore unknown keys", func(ctx SpecContext) {
			Expect(deleteInvites(
				authed(ctx, bobToken),
				apioidc.DeleteInvitesRequest{Keys: []invite.Key{uuid.New()}},
			)).Error().ToNot(HaveOccurred())
		})
	})

	Describe("RedeemInvite", func() {
		It("Should link the account and open a session", func(ctx SpecContext) {
			sub := uuid.New().String()
			res := MustSucceed(redeem(ctx, create(ctx).Token, grantAccount(sub)))
			Expect(res.Session.Subject).To(Equal(carol.ResourceID()))
			Expect(res.User.Key).To(Equal(carol.Key))
			Expect(MustSucceed(authSvc.Authenticate(ctx, res.Token)).Subject).
				To(Equal(carol.ResourceID()))
			Expect(retrieve(ctx)).To(BeEmpty())
			var link oidc.Link
			Expect(oidcSvc.NewRetrieve().
				Where(oidc.MatchKeys(res.Session.Credential)).
				Entry(&link).
				Exec(ctx, db)).To(Succeed())
			Expect(link.Label).To(Equal(sub + "@example.com"))
			next := login(ctx, oidcProof(grantAccount(sub)))
			Expect(MustSucceed(authSvc.Authenticate(ctx, next)).Subject).
				To(Equal(carol.ResourceID()))
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
				Expect(res.Session.Subject).To(Equal(carol.ResourceID()))
				next := login(ctx, oidcProof(closed(sub)))
				Expect(MustSucceed(authSvc.Authenticate(ctx, next)).Subject).
					To(Equal(carol.ResourceID()))
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
			cookie, ok := MustBeOk(
				fctx.Get(apiauth.SessionCookieParam),
			).(apiauth.SessionCookie)
			Expect(ok).To(BeTrue())
			Expect(MustSucceed(authSvc.Authenticate(ctx, cookie.Token)).Subject).
				To(Equal(carol.ResourceID()))
		})
		It("Should work once", func(ctx SpecContext) {
			token := create(ctx).Token
			MustSucceed(redeem(ctx, token, grantAccount(uuid.New().String())))
			Expect(redeem(ctx, token, grantAccount(uuid.New().String()))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
			Expect(countLinks(ctx)).To(Equal(1))
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
			Expect(countLinks(ctx)).To(Equal(1))
		})
		It("Should give one error for wrong and expired tokens", func(ctx SpecContext) {
			expired := create(ctx).Token
			clock.Advance(24 * time.Hour)
			Expect(redeem(ctx, expired, grantAccount(uuid.New().String()))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
			Expect(redeem(ctx, "syi_wrong", grantAccount(uuid.New().String()))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
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
				To(MatchError(ContainSubstring("linked to another user")))
			Expect(retrieve(ctx)).To(HaveLen(1))
		})
		It("Should refuse a user linked to the provider", func(ctx SpecContext) {
			linkAccount(
				ctx,
				login(ctx, passwordProof(carol.Name, "p")),
				uuid.New().String(),
			)
			Expect(redeem(ctx, create(ctx).Token, grantAccount(uuid.New().String()))).
				Error().To(MatchError(ContainSubstring("already has an account")))
		})
	})
})
