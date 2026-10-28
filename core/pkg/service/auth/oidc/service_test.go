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
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
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

// login registers a code for sub at the fake provider and proves it.
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

func roles(ctx SpecContext, subject ontology.ID) []ontology.Resource {
	GinkgoHelper()
	var parents []ontology.Resource
	Expect(otg.NewRetrieve().
		WhereIDs(subject).
		ExcludeFieldData(true).
		TraverseTo(ontology.ParentsTraverser).
		ExcludeFieldData(true).
		WhereTypes(ontology.ResourceTypeRole).
		Entries(&parents).
		Exec(ctx, nil)).To(Succeed())
	return parents
}

var _ = Describe("Service", func() {
	var svc *oidc.Service
	open := func(ctx SpecContext, providers ...oidc.Provider) *oidc.Service {
		GinkgoHelper()
		return MustOpen(oidc.OpenService(ctx, oidc.ServiceConfig{
			DB:          db,
			User:        userSvc,
			Role:        rbacSvc.Role,
			NewUserRole: rbacSvc.Builtin.ViewerKey,
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
				Role:        rbacSvc.Role,
				NewUserRole: rbacSvc.Builtin.ViewerKey,
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
			Entry(
				"no role service",
				func(c *oidc.ServiceConfig) { c.Role = nil },
				"role: must be non-nil",
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

	Describe("Verify", func() {
		It("Should return the account the ID token names", func(ctx SpecContext) {
			provider.grant("verify", grant{
				sub: "ada", name: "Ada Lovelace", verifier: "v", redirect: redirect,
			})
			Expect(
				svc.Verify(ctx, credentials("idp", "verify", "v")),
			).To(Equal(oidc.Account{
				Provider: "idp",
				ID:       "ada",
				Name:     "Ada Lovelace",
			}))
		})
		It("Should reject an unknown provider", func(ctx SpecContext) {
			Expect(svc.Verify(ctx, credentials("nope", "c", "v"))).Error().
				To(MatchError(validate.ErrValidation))
		})
		It("Should reject incomplete credentials", func(ctx SpecContext) {
			Expect(svc.Verify(ctx, auth.OIDCCredentials{Provider: "idp"})).Error().
				To(MatchError(ContainSubstring("code")))
		})
		It("Should reject a code the provider does not know", func(ctx SpecContext) {
			Expect(svc.Verify(ctx, credentials("idp", "unknown", "v"))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should reject a verifier that does not match", func(ctx SpecContext) {
			provider.grant("pkce", grant{sub: "ada", verifier: "v", redirect: redirect})
			Expect(svc.Verify(ctx, credentials("idp", "pkce", "other"))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should reject an ID token for another client", func(ctx SpecContext) {
			provider.grant("aud", grant{
				sub: "ada", verifier: "v", redirect: redirect, audience: "other",
			})
			Expect(svc.Verify(ctx, credentials("idp", "aud", "v"))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should reject a token response without an ID token", func(ctx SpecContext) {
			provider.grant("bare", grant{
				sub: "ada", verifier: "v", redirect: redirect, noIDToken: true,
			})
			Expect(svc.Verify(ctx, credentials("idp", "bare", "v"))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should report a provider it cannot reach", func(ctx SpecContext) {
			down := open(ctx, oidc.Provider{
				Name:     "down",
				Issuer:   "http://127.0.0.1:1",
				ClientID: clientID,
			})
			Expect(down.Verify(ctx, credentials("down", "c", "v"))).Error().
				To(MatchError(ContainSubstring(`discover provider "down"`)))
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
					Exec(ctx, nil)).To(Succeed())
				Expect(u.FirstName).To(Equal("Ada Lovelace"))
				Expect(roles(ctx, identity.Subject)).To(ConsistOf(
					HaveField("ID", role.OntologyID(rbacSvc.Builtin.ViewerKey)),
				))
				var link oidc.Link
				Expect(svc.NewRetrieve().
					Where(oidc.MatchKeys(identity.Credential)).
					Entry(&link).
					Exec(ctx, nil)).To(Succeed())
				Expect(link.Provider).To(Equal("idp"))
				Expect(link.Account).To(Equal(sub))
				Expect(link.Subject).To(Equal(identity.Subject))
			},
		)
		It("Should log a linked account in as its user", func(ctx SpecContext) {
			sub := uuid.New().String()
			first := MustSucceed(login(svc, ctx, "idp", sub))
			Expect(login(svc, ctx, "idp", sub)).To(Equal(first))
		})
		It("Should log in a linked account when user creation is off", func(
			ctx SpecContext,
		) {
			u := MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{
				Username: uuid.New().String(),
			}))
			link := MustSucceed(
				svc.NewWriter(nil).Create(ctx, u.OntologyID(), oidc.Account{
					Provider: "closed",
					ID:       "grace",
				}),
			)
			Expect(login(svc, ctx, "closed", "grace")).To(Equal(auth.Identity{
				Subject:    u.OntologyID(),
				Credential: link.Key,
			}))
		})
		It("Should refuse an unlinked account when user creation is off", func(
			ctx SpecContext,
		) {
			sub := uuid.New().String()
			Expect(login(svc, ctx, "closed", sub)).Error().
				To(MatchError(auth.ErrInvalidCredentials))
			Expect(svc.NewRetrieve().Where(oidc.MatchAccounts(sub)).Exists(ctx, nil)).
				To(BeFalse())
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
			var link oidc.Link
			Expect(svc.NewRetrieve().
				Where(oidc.MatchAccounts(sub)).
				Entry(&link).
				Exec(ctx, nil)).To(Succeed())
			Expect(session.Subject).To(Equal(link.Subject))
			Expect(session.Credential).To(Equal(link.Key))
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

	Describe("Writer", func() {
		var subject ontology.ID
		BeforeEach(func(ctx SpecContext) {
			subject = MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{
				Username: uuid.New().String(),
			})).OntologyID()
		})
		It("Should refuse a second link to one account", func(ctx SpecContext) {
			account := oidc.Account{Provider: "idp", ID: uuid.New().String()}
			MustSucceed(svc.NewWriter(nil).Create(ctx, subject, account))
			other := MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{
				Username: uuid.New().String(),
			})).OntologyID()
			Expect(svc.NewWriter(nil).Create(ctx, other, account)).Error().
				To(MatchError(validate.ErrValidation))
		})
		It("Should refuse a second link from one user to a provider", func(
			ctx SpecContext,
		) {
			MustSucceed(svc.NewWriter(nil).Create(ctx, subject, oidc.Account{
				Provider: "idp", ID: uuid.New().String(),
			}))
			Expect(svc.NewWriter(nil).Create(ctx, subject, oidc.Account{
				Provider: "idp", ID: uuid.New().String(),
			})).Error().To(MatchError(validate.ErrValidation))
		})
		It("Should reject a link without an account", func(ctx SpecContext) {
			Expect(
				svc.NewWriter(nil).Create(ctx, subject, oidc.Account{Provider: "idp"}),
			).
				Error().
				To(MatchError(ContainSubstring("account")))
		})
		It("Should delete links by key", func(ctx SpecContext) {
			l := MustSucceed(svc.NewWriter(nil).Create(ctx, subject, oidc.Account{
				Provider: "idp", ID: uuid.New().String(),
			}))
			Expect(svc.NewWriter(nil).Delete(ctx, l.Key)).To(Succeed())
			Expect(svc.NewRetrieve().Where(oidc.MatchKeys(l.Key)).Entry(&oidc.Link{}).
				Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
		})
		It("Should delete every link of a subject", func(ctx SpecContext) {
			for _, p := range []string{"idp", "closed"} {
				MustSucceed(svc.NewWriter(nil).Create(ctx, subject, oidc.Account{
					Provider: p, ID: uuid.New().String(),
				}))
			}
			Expect(svc.DeleteFor(ctx, nil, subject)).To(Succeed())
			Expect(
				svc.NewRetrieve().Where(oidc.MatchSubjects(subject)).Exists(ctx, nil),
			).
				To(BeFalse())
		})
	})
})
