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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

const redirect = "http://127.0.0.1:4242/callback"

func credentials(providerName, code, verifier string) auth.OIDCCredentials {
	return auth.OIDCCredentials{
		Provider:    providerName,
		Code:        code,
		Verifier:    verifier,
		RedirectURI: redirect,
	}
}

func proof(creds auth.OIDCCredentials) auth.Proof {
	return auth.Proof{Variant: auth.OIDCProof{OIDCCredentials: creds}}
}

func login(
	svc *oidc.Service,
	ctx SpecContext,
	providerName, sub string,
) (auth.Identity, error) {
	code := uuid.New().String()
	provider.grant(
		code,
		grant{sub: sub, name: "Ada Lovelace", verifier: "v", redirect: redirect},
	)
	return svc.Authenticate(ctx, proof(credentials(providerName, code, "v")))
}

var _ = Describe("Service", func() {
	var svc *oidc.Service
	open := func(ctx SpecContext, providers ...oidc.Provider) *oidc.Service {
		GinkgoHelper()
		return MustOpen(oidc.OpenService(ctx, oidc.ServiceConfig{
			DB:          db,
			User:        userSvc,
			NewUserRole: viewer,
			Providers:   providers,
		}))
	}
	BeforeEach(func(ctx SpecContext) {
		svc = open(ctx,
			oidc.Provider{Name: "idp", Issuer: provider.issuer(), ClientID: clientID},
			oidc.Provider{
				Name:                 "closed",
				Issuer:               provider.issuer(),
				ClientID:             clientID,
				UserCreationDisabled: true,
			},
		)
	})

	Describe("Config", func() {
		valid := func() oidc.ServiceConfig {
			return oidc.ServiceConfig{
				DB:          db,
				User:        userSvc,
				NewUserRole: viewer,
			}
		}
		DescribeTable(
			"Should reject an invalid config",
			func(
				ctx SpecContext,
				mutate func(*oidc.ServiceConfig),
				msg string,
			) {
				cfg := valid()
				mutate(&cfg)
				Expect(oidc.OpenService(ctx, cfg)).Error().
					To(MatchError(ContainSubstring(msg)))
			},
			Entry(
				"no DB",
				func(c *oidc.ServiceConfig) { c.DB = nil },
				"db: must be non-nil",
			),
			Entry(
				"no user service",
				func(c *oidc.ServiceConfig) { c.User = nil },
				"user: must be non-nil",
			),
			Entry("no new user role", func(c *oidc.ServiceConfig) {
				c.NewUserRole = uuid.Nil()
			}, "new_user_role: must be set"),
			Entry("a provider without a name", func(c *oidc.ServiceConfig) {
				c.Providers = []oidc.Provider{{Issuer: "https://idp", ClientID: "c"}}
			}, "name must be set"),
			Entry("a provider without an issuer", func(c *oidc.ServiceConfig) {
				c.Providers = []oidc.Provider{{Name: "a", ClientID: "c"}}
			}, `"a" needs an issuer`),
			Entry("a provider without a client ID", func(c *oidc.ServiceConfig) {
				c.Providers = []oidc.Provider{{Name: "a", Issuer: "https://idp"}}
			}, `"a" needs a client ID`),
			Entry("two providers with one name", func(c *oidc.ServiceConfig) {
				p := oidc.Provider{Name: "a", Issuer: "https://idp", ClientID: "c"}
				c.Providers = []oidc.Provider{p, p}
			}, `duplicate name "a"`),
		)
	})

	Describe("Provider exchange", func() {
		It("Should reject an unknown provider", func(ctx SpecContext) {
			Expect(svc.Authenticate(ctx, proof(credentials("nope", "c", "v")))).Error().
				To(MatchError(validate.ErrValidation))
		})
		It("Should reject incomplete credentials", func(ctx SpecContext) {
			Expect(svc.Authenticate(ctx, proof(auth.OIDCCredentials{Provider: "idp"}))).
				Error().To(MatchError(ContainSubstring("code")))
		})
		It("Should reject a code the provider does not know", func(ctx SpecContext) {
			Expect(svc.Authenticate(ctx, proof(credentials("idp", "unknown", "v")))).
				Error().To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should reject a verifier that does not match", func(ctx SpecContext) {
			provider.grant("pkce", grant{sub: "ada", verifier: "v", redirect: redirect})
			Expect(svc.Authenticate(ctx, proof(credentials("idp", "pkce", "other")))).
				Error().To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should reject an ID token for another client", func(ctx SpecContext) {
			provider.grant("aud", grant{
				sub: "ada", verifier: "v", redirect: redirect, audience: "other",
			})
			Expect(svc.Authenticate(ctx, proof(credentials("idp", "aud", "v")))).
				Error().To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should reject a token response without an ID token", func(ctx SpecContext) {
			provider.grant("bare", grant{
				sub: "ada", verifier: "v", redirect: redirect, noIDToken: true,
			})
			Expect(svc.Authenticate(ctx, proof(credentials("idp", "bare", "v")))).
				Error().To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should report a provider it cannot reach", func(ctx SpecContext) {
			other := DeferClose(gorp.Wrap(memkv.New()))
			otherUsers, otherViewer := openUsers(ctx, other)
			down := MustOpen(oidc.OpenService(ctx, oidc.ServiceConfig{
				DB:          other,
				User:        otherUsers,
				NewUserRole: otherViewer,
				Providers: []oidc.Provider{
					{Name: "down", Issuer: "http://127.0.0.1:1", ClientID: clientID},
				},
			}))
			Expect(down.Authenticate(ctx, proof(credentials("down", "c", "v")))).
				Error().To(MatchError(ContainSubstring(`discover provider "down"`)))
		})
	})

	Describe("Authenticate", func() {
		It(
			"Should create a viewer user at an account's first login",
			func(ctx SpecContext) {
				sub := uuid.New().String()
				identity := MustSucceed(login(svc, ctx, "idp", sub))
				var u user.User
				Expect(userSvc.NewRetrieve().
					Where(user.MatchKeys(uuid.MustParse(identity.Subject.Key))).
					Entry(&u).
					Exec(ctx, db)).To(Succeed())
				Expect(u.Name).To(Equal("Ada Lovelace"))
				Expect(u.Role).To(Equal(viewer))
				Expect(svc.Version(ctx, db, identity.Credential)).To(BeZero())
			},
		)
		It("Should name a user without a name claim", func(ctx SpecContext) {
			code := uuid.New().String()
			provider.grant(code, grant{
				sub: uuid.New().String(), verifier: "v", redirect: redirect,
			})
			identity := MustSucceed(
				svc.Authenticate(ctx, proof(credentials("idp", code, "v"))),
			)
			var u user.User
			Expect(userSvc.NewRetrieve().
				Where(user.MatchKeys(uuid.MustParse(identity.Subject.Key))).
				Entry(&u).
				Exec(ctx, db)).To(Succeed())
			Expect(u.Name).To(Equal("Unnamed user"))
		})
		It(
			"Should treat the same account ID at another issuer as another account",
			func(
				ctx SpecContext,
			) {
				other := DeferClose(gorp.Wrap(memkv.New()))
				otherUsers, otherViewer := openUsers(ctx, other)
				moved := newFakeProvider()
				DeferCleanup(moved.server.Close)
				cfg := oidc.ServiceConfig{
					DB:          other,
					User:        otherUsers,
					NewUserRole: otherViewer,
					Providers: []oidc.Provider{
						{Name: "idp", Issuer: provider.issuer(), ClientID: clientID},
					},
				}
				before := MustSucceed(oidc.OpenService(ctx, cfg))
				sub := uuid.New().String()
				identity := MustSucceed(login(before, ctx, "idp", sub))
				Expect(before.Close()).To(Succeed())
				cfg.Providers[0].Issuer = moved.issuer()
				after := MustOpen(oidc.OpenService(ctx, cfg))
				moved.grant("moved", grant{sub: sub, verifier: "v", redirect: redirect})
				Expect(
					after.Authenticate(ctx, proof(credentials("idp", "moved", "v"))),
				).
					ToNot(HaveField("Subject", identity.Subject))
				Expect(after.Version(ctx, other, identity.Credential)).Error().
					To(MatchError(query.ErrNotFound))
			},
		)
		It("Should keep the links of a renamed provider", func(ctx SpecContext) {
			other := DeferClose(gorp.Wrap(memkv.New()))
			otherUsers, otherViewer := openUsers(ctx, other)
			cfg := oidc.ServiceConfig{
				DB:          other,
				User:        otherUsers,
				NewUserRole: otherViewer,
				Providers: []oidc.Provider{
					{Name: "idp", Issuer: provider.issuer(), ClientID: clientID},
				},
			}
			before := MustSucceed(oidc.OpenService(ctx, cfg))
			sub := uuid.New().String()
			identity := MustSucceed(login(before, ctx, "idp", sub))
			Expect(before.Close()).To(Succeed())
			cfg.Providers[0].Name = "renamed"
			after := MustOpen(oidc.OpenService(ctx, cfg))
			Expect(after.Version(ctx, other, identity.Credential)).Error().
				ToNot(HaveOccurred())
			Expect(login(after, ctx, "renamed", sub)).To(Equal(identity))
		})
		It("Should log a linked account in as its user", func(ctx SpecContext) {
			sub := uuid.New().String()
			first := MustSucceed(login(svc, ctx, "idp", sub))
			Expect(login(svc, ctx, "idp", sub)).To(Equal(first))
		})
		It("Should refuse an unlinked account when user creation is off", func(
			ctx SpecContext,
		) {
			sub := uuid.New().String()
			Expect(login(svc, ctx, "closed", sub)).Error().
				To(MatchError(auth.ErrInvalidCredentials))
			Expect(login(svc, ctx, "closed", sub)).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should open a session through the auth service", func(ctx SpecContext) {
			authSvc := MustOpen(auth.OpenService(ctx, auth.ServiceConfig{
				DB: db,
				Authenticators: map[auth.Method]auth.Authenticator{
					auth.MethodOIDC: svc,
				},
			}))
			sub := uuid.New().String()
			code := uuid.New().String()
			provider.grant(code, grant{sub: sub, verifier: "v", redirect: redirect})
			session, token := MustSucceed2(
				authSvc.Login(ctx, proof(credentials("idp", code, "v"))),
			)
			Expect(token).ToNot(BeEmpty())
			Expect(login(svc, ctx, "idp", sub)).To(Equal(auth.Identity{
				Subject:    session.Subject,
				Credential: session.Credential,
			}))
		})
		It("Should reject a proof of another method", func(ctx SpecContext) {
			Expect(svc.Authenticate(ctx, auth.Proof{Variant: auth.APIKeyProof{}})).
				Error().To(MatchError(validate.ErrValidation))
		})
		It("Should return the provider's rejection", func(ctx SpecContext) {
			Expect(svc.Authenticate(ctx, proof(credentials("idp", "unknown", "v")))).
				Error().To(MatchError(auth.ErrInvalidCredentials))
		})
	})

	Describe("Version", func() {
		It("Should not find the link of a deleted user", func(ctx SpecContext) {
			identity := MustSucceed(login(svc, ctx, "idp", uuid.New().String()))
			key := MustSucceed(user.KeyFromResourceID(identity.Subject))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return userSvc.NewWriter(tx).Delete(ctx, key)
			})).To(Succeed())
			Expect(svc.Version(ctx, db, identity.Credential)).Error().
				To(MatchError(query.ErrNotFound))
		})
		It("Should end the sessions of a removed provider", func(ctx SpecContext) {
			other := DeferClose(gorp.Wrap(memkv.New()))
			otherUsers, otherViewer := openUsers(ctx, other)
			cfg := oidc.ServiceConfig{
				DB:          other,
				User:        otherUsers,
				NewUserRole: otherViewer,
				Providers: []oidc.Provider{
					{Name: "idp", Issuer: provider.issuer(), ClientID: clientID},
				},
			}
			withIDP := MustSucceed(oidc.OpenService(ctx, cfg))
			on := MustSucceed(auth.OpenService(ctx, auth.ServiceConfig{
				DB: other,
				Authenticators: map[auth.Method]auth.Authenticator{
					auth.MethodOIDC: withIDP,
				},
			}))
			code := uuid.New().String()
			provider.grant(
				code,
				grant{sub: uuid.New().String(), verifier: "v", redirect: redirect},
			)
			session, token := MustSucceed2(
				on.Login(ctx, proof(credentials("idp", code, "v"))),
			)
			Expect(on.Close()).To(Succeed())
			Expect(withIDP.Close()).To(Succeed())
			cfg.Providers = nil
			without := MustOpen(oidc.OpenService(ctx, cfg))
			Expect(without.Version(ctx, other, session.Credential)).Error().
				To(MatchError(query.ErrNotFound))
			off := MustOpen(auth.OpenService(ctx, auth.ServiceConfig{
				DB: other,
				Authenticators: map[auth.Method]auth.Authenticator{
					auth.MethodOIDC: without,
				},
			}))
			Expect(off.Authenticate(ctx, token)).Error().
				To(MatchError(auth.ErrInvalidToken))
			Expect(off.NewRetrieve().
				Where(auth.MatchKeys(session.Key)).
				Exec(ctx, other)).To(MatchError(query.ErrNotFound))
		})
		It("Should not find an unknown credential", func(ctx SpecContext) {
			Expect(svc.Version(ctx, db, uuid.New())).Error().
				To(MatchError(query.ErrNotFound))
		})
	})
})
