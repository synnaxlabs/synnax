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
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc/testutil"
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
	provider.Grant(
		code,
		testutil.Grant{
			Subject:     sub,
			Name:        "Ada Lovelace",
			Verifier:    "v",
			RedirectURI: redirect,
		},
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
	// openFresh opens a service on a new DB, since a DB holds one open link table.
	openFresh := func(ctx SpecContext, cfg oidc.ServiceConfig) *oidc.Service {
		GinkgoHelper()
		cfg.DB = DeferClose(gorp.Wrap(memkv.New()))
		cfg.User, cfg.NewUserRole = openUsers(ctx, cfg.DB)
		return MustOpen(oidc.OpenService(ctx, cfg))
	}
	// openOther opens a second service with the given providers.
	openOther := func(ctx SpecContext, providers ...oidc.Provider) *oidc.Service {
		GinkgoHelper()
		return openFresh(ctx, oidc.ServiceConfig{Providers: providers})
	}
	BeforeEach(func(ctx SpecContext) {
		svc = open(
			ctx,
			oidc.Provider{
				Name:     "idp",
				Issuer:   provider.Issuer(),
				ClientID: testutil.ClientID,
			},
			oidc.Provider{
				Name:                 "closed",
				Issuer:               provider.Issuer(),
				ClientID:             testutil.ClientID,
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
			Entry("a negative discovery timeout", func(c *oidc.ServiceConfig) {
				c.DiscoveryTimeout = -time.Second
			}, "discovery_timeout: must be positive"),
		)
		It("Should default an unset discovery timeout", func(ctx SpecContext) {
			Expect(valid().DiscoveryTimeout).To(BeZero())
			openFresh(ctx, valid())
		})
	})

	Describe("Verify", func() {
		It("Should return the account the ID token names", func(ctx SpecContext) {
			provider.Grant("verify", testutil.Grant{
				Subject:     "ada",
				Name:        "Ada Lovelace",
				Verifier:    "v",
				RedirectURI: redirect,
			})
			Expect(
				svc.Verify(ctx, credentials("idp", "verify", "v")),
			).To(Equal(oidc.Account{
				Provider: "idp", Issuer: provider.Issuer(),
				ID:   "ada",
				Name: "Ada Lovelace",
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
			provider.Grant(
				"pkce",
				testutil.Grant{Subject: "ada", Verifier: "v", RedirectURI: redirect},
			)
			Expect(svc.Verify(ctx, credentials("idp", "pkce", "other"))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should reject an ID token for another client", func(ctx SpecContext) {
			provider.Grant("aud", testutil.Grant{
				Subject: "ada", Verifier: "v", RedirectURI: redirect, Audience: "other",
			})
			Expect(svc.Verify(ctx, credentials("idp", "aud", "v"))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should reject a token response without an ID token", func(ctx SpecContext) {
			provider.Grant("bare", testutil.Grant{
				Subject:        "ada",
				Verifier:       "v",
				RedirectURI:    redirect,
				IDTokenOmitted: true,
			})
			Expect(svc.Verify(ctx, credentials("idp", "bare", "v"))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should report a provider it cannot reach", func(ctx SpecContext) {
			down := openOther(ctx, oidc.Provider{
				Name:     "down",
				Issuer:   "http://127.0.0.1:1",
				ClientID: testutil.ClientID,
			})
			Expect(down.Verify(ctx, credentials("down", "c", "v"))).Error().
				To(MatchError(ContainSubstring(`discover provider "down"`)))
		})
		It("Should stop waiting on another discovery when its context ends", func(
			ctx SpecContext,
		) {
			reached := make(chan struct{}, 1)
			release := make(chan struct{})
			stalled := httptest.NewServer(http.HandlerFunc(
				func(http.ResponseWriter, *http.Request) {
					select {
					case reached <- struct{}{}:
					default:
					}
					select {
					case <-release:
					case <-time.After(10 * time.Second):
					}
				},
			))
			DeferCleanup(stalled.Close)
			releaseOnce := sync.OnceFunc(func() { close(release) })
			DeferCleanup(releaseOnce)
			stalledSvc := openOther(ctx, oidc.Provider{
				Name:     "stalled",
				Issuer:   stalled.URL,
				ClientID: testutil.ClientID,
			})
			creds := credentials("stalled", "c", "v")
			discovered := make(chan struct{})
			go func() {
				defer GinkgoRecover()
				defer close(discovered)
				Expect(stalledSvc.Verify(ctx, creds)).Error().
					To(MatchError(ContainSubstring(`discover provider "stalled"`)))
			}()
			Eventually(reached).Should(Receive())
			cancelled, cancel := context.WithCancel(ctx)
			cancel()
			Expect(stalledSvc.Verify(cancelled, creds)).Error().
				To(MatchError(context.Canceled))
			releaseOnce()
			Eventually(discovered).Should(BeClosed())
		})
	})

	Describe("Logins", func() {
		// stall returns the address of a provider that never answers, and the number
		// of requests it has received.
		stall := func() (string, func() int32) {
			var requests atomic.Int32
			release := make(chan struct{})
			stalled := httptest.NewServer(http.HandlerFunc(
				func(http.ResponseWriter, *http.Request) {
					requests.Add(1)
					<-release
				},
			))
			DeferCleanup(stalled.Close)
			DeferCleanup(func() { close(release) })
			return stalled.URL, requests.Load
		}
		withStalled := func(
			ctx SpecContext,
			timeout time.Duration,
			stalled string,
		) *oidc.Service {
			GinkgoHelper()
			return openFresh(ctx, oidc.ServiceConfig{
				DiscoveryTimeout: timeout,
				Providers: []oidc.Provider{
					{Name: "stalled", Issuer: stalled, ClientID: "s"},
					{Name: "up", Issuer: provider.Issuer(), ClientID: "u"},
				},
			})
		}
		It(
			"Should list each reachable provider's login address",
			func(ctx SpecContext) {
				withDown := openOther(
					ctx,
					oidc.Provider{
						Name:     "b",
						Issuer:   provider.Issuer(),
						ClientID: "b-client",
					},
					oidc.Provider{
						Name:     "a",
						Issuer:   "http://127.0.0.1:1",
						ClientID: "a",
					},
					oidc.Provider{
						Name:     "c",
						Issuer:   provider.Issuer(),
						ClientID: "c-client",
					},
				)
				Eventually(withDown.Logins).Should(Equal([]oidc.Login{
					{
						Provider:              "b",
						AuthorizationEndpoint: provider.Issuer() + "/authorize",
						ClientID:              "b-client",
					},
					{
						Provider:              "c",
						AuthorizationEndpoint: provider.Issuer() + "/authorize",
						ClientID:              "c-client",
					},
				}))
			},
		)
		It("Should answer at once while a provider never answers", func(
			ctx SpecContext,
		) {
			stalled, _ := stall()
			svc := withStalled(ctx, 5*time.Second, stalled)
			for range 2 {
				start := time.Now()
				svc.Logins()
				Expect(time.Since(start)).To(BeNumerically("<", 100*time.Millisecond))
			}
			Eventually(svc.Logins).Should(ConsistOf(HaveField("Provider", "up")))
		})
		It("Should not read a failed provider again within the retry interval", func(
			ctx SpecContext,
		) {
			stalled, requests := stall()
			svc := withStalled(ctx, 50*time.Millisecond, stalled)
			Eventually(func(g Gomega) {
				g.Expect(svc.Authenticate(ctx, proof(credentials("stalled", "c", "v")))).
					Error().
					To(MatchError(ContainSubstring(`discover provider "stalled"`)))
			}).Should(Succeed())
			start := time.Now()
			Expect(svc.Authenticate(ctx, proof(credentials("stalled", "c", "v")))).
				Error().To(MatchError(ContainSubstring(`discover provider "stalled"`)))
			Expect(time.Since(start)).To(BeNumerically("<", 50*time.Millisecond))
			Consistently(func() int32 {
				svc.Logins()
				return requests()
			}, 200*time.Millisecond).Should(BeEquivalentTo(1))
		})
		It("Should not wait on a login stalled in discovery", func(ctx SpecContext) {
			stalled, requests := stall()
			svc := withStalled(ctx, 5*time.Second, stalled)
			Eventually(requests).Should(BeEquivalentTo(1))
			loggedIn := make(chan struct{})
			loginCtx, cancel := context.WithCancel(ctx)
			go func() {
				defer GinkgoRecover()
				defer close(loggedIn)
				Expect(svc.Authenticate(
					loginCtx,
					proof(credentials("stalled", "code", "v")),
				)).Error().To(MatchError(context.Canceled))
			}()
			Eventually(svc.Logins).Should(ConsistOf(HaveField("Provider", "up")))
			cancel()
			Eventually(loggedIn).Should(BeClosed())
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
				var link oidc.Link
				Expect(svc.NewRetrieve().
					Where(oidc.MatchKeys(identity.Credential)).
					Entry(&link).
					Exec(ctx, db)).To(Succeed())
				Expect(link.Issuer).To(Equal(provider.Issuer()))
				Expect(link.Account).To(Equal(sub))
				Expect(user.ResourceID(link.User)).To(Equal(identity.Subject))
			},
		)
		It("Should name a user without a name claim", func(ctx SpecContext) {
			code := uuid.New().String()
			provider.Grant(code, testutil.Grant{
				Subject: uuid.New().String(), Verifier: "v", RedirectURI: redirect,
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
			"Should keep the same account ID at two issuers apart",
			func(ctx SpecContext) {
				other := DeferClose(gorp.Wrap(memkv.New()))
				otherUsers, otherViewer := openUsers(ctx, other)
				moved := testutil.NewProvider()
				DeferCleanup(moved.Close)
				cfg := oidc.ServiceConfig{
					DB:          other,
					User:        otherUsers,
					NewUserRole: otherViewer,
					Providers: []oidc.Provider{
						{
							Name:     "idp",
							Issuer:   provider.Issuer(),
							ClientID: testutil.ClientID,
						},
					},
				}
				before := MustSucceed(oidc.OpenService(ctx, cfg))
				sub := uuid.New().String()
				identity := MustSucceed(login(before, ctx, "idp", sub))
				Expect(before.Close()).To(Succeed())
				cfg.Providers[0].Issuer = moved.Issuer()
				after := MustOpen(oidc.OpenService(ctx, cfg))
				moved.Grant("moved", testutil.Grant{
					Subject: sub, Verifier: "v", RedirectURI: redirect,
				})
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
					{
						Name:     "idp",
						Issuer:   provider.Issuer(),
						ClientID: testutil.ClientID,
					},
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
		It("Should log in a linked account when user creation is off", func(
			ctx SpecContext,
		) {
			subject := createUser(ctx)
			l := MustSucceed(link(ctx, svc, subject, oidc.Account{
				Provider: "closed", Issuer: provider.Issuer(),
				ID: "grace",
			}))
			Expect(login(svc, ctx, "closed", "grace")).To(Equal(auth.Identity{
				Subject:    user.ResourceID(subject),
				Credential: l.Key,
			}))
		})
		It("Should refuse an unlinked account when user creation is off", func(
			ctx SpecContext,
		) {
			sub := uuid.New().String()
			Expect(login(svc, ctx, "closed", sub)).Error().
				To(MatchError(auth.ErrInvalidCredentials))
			Expect(svc.NewRetrieve().Where(oidc.MatchAccounts(sub)).Exists(ctx, db)).
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
			provider.Grant(code, testutil.Grant{
				Subject: sub, Verifier: "v", RedirectURI: redirect,
			})
			session, token := MustSucceed2(
				authSvc.Login(ctx, proof(credentials("idp", code, "v"))),
			)
			Expect(token).ToNot(BeEmpty())
			var link oidc.Link
			Expect(svc.NewRetrieve().
				Where(oidc.MatchAccounts(sub)).
				Entry(&link).
				Exec(ctx, db)).To(Succeed())
			Expect(session.Subject).To(Equal(user.ResourceID(link.User)))
			Expect(session.Credential).To(Equal(link.Key))
		})
		It("Should create one user when two first logins of one account race", func(
			ctx SpecContext,
		) {
			sub := uuid.New().String()
			name := uuid.New().String()
			codes := []string{uuid.New().String(), uuid.New().String()}
			for _, code := range codes {
				provider.Grant(code, testutil.Grant{
					Subject: sub, Name: name, Verifier: "v", RedirectURI: redirect,
				})
			}
			identities := make([]auth.Identity, len(codes))
			var wg sync.WaitGroup
			for i, code := range codes {
				wg.Go(func() {
					defer GinkgoRecover()
					identities[i] = MustSucceed(
						svc.Authenticate(ctx, proof(credentials("idp", code, "v"))),
					)
				})
			}
			wg.Wait()
			Expect(identities[0]).To(Equal(identities[1]))
			Expect(svc.Version(ctx, db, identities[0].Credential)).
				To(Equal(identities[0].Version))
			var users []user.User
			Expect(userSvc.NewRetrieve().
				Where(user.Match(func(_ gorp.Context, _ user.Retrieve, u *user.User) (bool, error) {
					return u.Name == name, nil
				})).
				Entries(&users).
				Exec(ctx, db)).To(Succeed())
			Expect(users).To(HaveLen(1))
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
		var subject user.Key
		BeforeEach(func(ctx SpecContext) { subject = createUser(ctx) })
		It("Should refuse a second link to one account", func(ctx SpecContext) {
			account := oidc.Account{
				Provider: "idp",
				Issuer:   provider.Issuer(),
				ID:       uuid.New().String(),
			}
			MustSucceed(link(ctx, svc, subject, account))
			Expect(link(ctx, svc, createUser(ctx), account)).Error().
				To(MatchError(validate.ErrValidation))
		})
		It("Should refuse a second link from one user to a provider", func(
			ctx SpecContext,
		) {
			MustSucceed(link(ctx, svc, subject, oidc.Account{
				Provider: "idp", Issuer: provider.Issuer(), ID: uuid.New().String(),
			}))
			Expect(link(ctx, svc, subject, oidc.Account{
				Provider: "idp", Issuer: provider.Issuer(), ID: uuid.New().String(),
			})).Error().To(MatchError(validate.ErrValidation))
		})
		It("Should refuse the second of two links to one account made at once", func(
			ctx SpecContext,
		) {
			account := oidc.Account{
				Provider: "idp",
				Issuer:   provider.Issuer(),
				ID:       uuid.New().String(),
			}
			subjects := []user.Key{subject, createUser(ctx)}
			errs := make([]error, len(subjects))
			var wg sync.WaitGroup
			for i, sub := range subjects {
				wg.Go(func() {
					_, errs[i] = link(ctx, svc, sub, account)
				})
			}
			wg.Wait()
			Expect(errs).To(ConsistOf(Succeed(), MatchError(validate.ErrValidation)))
			var links []oidc.Link
			Expect(svc.NewRetrieve().Where(oidc.MatchUsers(subjects...)).
				Entries(&links).Exec(ctx, db)).To(Succeed())
			Expect(links).To(HaveLen(1))
		})
		It("Should reject a link without an account", func(ctx SpecContext) {
			Expect(link(ctx, svc, subject, oidc.Account{Provider: "idp"})).Error().
				To(MatchError(ContainSubstring("account")))
		})
		It("Should delete links by key", func(ctx SpecContext) {
			l := MustSucceed(link(ctx, svc, subject, oidc.Account{
				Provider: "idp", Issuer: provider.Issuer(), ID: uuid.New().String(),
			}))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, l.Key)
			})).To(Succeed())
			Expect(svc.NewRetrieve().Where(oidc.MatchKeys(l.Key)).Entry(&oidc.Link{}).
				Exec(ctx, db)).To(MatchError(query.ErrNotFound))
		})
		It("Should delete every link of a deleted user", func(ctx SpecContext) {
			for _, issuer := range []string{provider.Issuer(), "https://other.example"} {
				MustSucceed(link(ctx, svc, subject, oidc.Account{
					Provider: "idp", Issuer: issuer, ID: uuid.New().String(),
				}))
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return userSvc.NewWriter(tx).Delete(ctx, subject)
			})).To(Succeed())
			Expect(
				svc.NewRetrieve().Where(oidc.MatchUsers(subject)).Exists(ctx, db),
			).
				To(BeFalse())
		})
	})

	Describe("Version", func() {
		var l oidc.Link
		BeforeEach(func(ctx SpecContext) {
			l = MustSucceed(link(ctx, svc, createUser(ctx), oidc.Account{
				Provider: "idp", Issuer: provider.Issuer(), ID: uuid.New().String(),
			}))
		})
		It("Should return the fixed version of a link", func(ctx SpecContext) {
			Expect(svc.Version(ctx, db, l.Key)).To(BeZero())
		})
		It("Should end a session of an unlinked account that links again", func(
			ctx SpecContext,
		) {
			other := DeferClose(gorp.Wrap(memkv.New()))
			otherUsers, otherViewer := openUsers(ctx, other)
			linked := MustOpen(oidc.OpenService(ctx, oidc.ServiceConfig{
				DB:          other,
				User:        otherUsers,
				NewUserRole: otherViewer,
				Providers: []oidc.Provider{{
					Name:     "idp",
					Issuer:   provider.Issuer(),
					ClientID: testutil.ClientID,
				}},
			}))
			authSvc := MustOpen(auth.OpenService(ctx, auth.ServiceConfig{
				DB: other,
				Authenticators: map[auth.Method]auth.Authenticator{
					auth.MethodOIDC: linked,
				},
			}))
			sub := uuid.New().String()
			code := uuid.New().String()
			provider.Grant(code, testutil.Grant{
				Subject: sub, Verifier: "v", RedirectURI: redirect,
			})
			session, token := MustSucceed2(
				authSvc.Login(ctx, proof(credentials("idp", code, "v"))),
			)
			// the session outlives the unlink, as a login in flight does
			Expect(other.WithTx(ctx, func(tx gorp.Tx) error {
				return linked.NewWriter(tx).Delete(ctx, session.Credential)
			})).To(Succeed())
			Expect(other.WithTx(ctx, func(tx gorp.Tx) error {
				_, err := linked.NewWriter(tx).Create(
					ctx,
					MustSucceed(user.KeyFromResourceID(session.Subject)),
					oidc.Account{Provider: "idp", Issuer: provider.Issuer(), ID: sub},
				)
				return err
			})).To(Succeed())
			Expect(authSvc.Authenticate(ctx, token)).Error().
				To(MatchError(auth.ErrInvalidToken))
		})
		It("Should not find a deleted link", func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, l.Key)
			})).To(Succeed())
			Expect(svc.Version(ctx, db, l.Key)).Error().
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
					{
						Name:     "idp",
						Issuer:   provider.Issuer(),
						ClientID: testutil.ClientID,
					},
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
			provider.Grant(code, testutil.Grant{
				Subject:     uuid.New().String(),
				Verifier:    "v",
				RedirectURI: redirect,
			})
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
