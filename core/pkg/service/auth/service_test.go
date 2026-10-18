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
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	xtime "github.com/synnaxlabs/x/time"
	"github.com/synnaxlabs/x/validate"
)

func subject() ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeUser, Key: uuid.New().String()}
}

func proof(username, pwd string) auth.Proof {
	return auth.Proof{Variant: auth.PasswordProof{
		Credentials: auth.Credentials{Username: username, Password: pwd},
	}}
}

var _ = Describe("Proof", func() {
	Describe("Method", func() {
		It("Should name the method of a password proof", func() {
			Expect(proof("u", "p").Method()).To(Equal(auth.MethodPassword))
		})
		It("Should reject a proof with no variant", func() {
			Expect(auth.Proof{}.Method()).Error().To(MatchError(validate.ErrValidation))
		})
	})
})

var _ = Describe("ServiceConfig", func() {
	var (
		cfg   auth.ServiceConfig
		auths map[auth.Method]auth.Authenticator
	)
	BeforeEach(func(ctx SpecContext) {
		pw := MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
		auths = map[auth.Method]auth.Authenticator{auth.MethodPassword: pw}
		cfg = auth.ServiceConfig{
			DB:             db,
			Authenticators: auths,
			IdleTimeout:    time.Hour,
			Clock:          xtime.Real,
		}
	})
	Describe("Override", func() {
		It("Should take each field from the override when the base is zero", func() {
			result := auth.ServiceConfig{}.Override(cfg)
			Expect(result.DB).To(Equal(db))
			Expect(result.Authenticators).To(Equal(auths))
			Expect(result.IdleTimeout).To(Equal(time.Hour))
			Expect(result.Clock).To(Equal(xtime.Real))
		})
		It("Should keep each field from the base when the override is zero", func() {
			result := cfg.Override(auth.ServiceConfig{})
			Expect(result.DB).To(Equal(db))
			Expect(result.Authenticators).To(Equal(auths))
			Expect(result.IdleTimeout).To(Equal(time.Hour))
			Expect(result.Clock).To(Equal(xtime.Real))
		})
	})
	Describe("Validate", func() {
		It("Should succeed when every required field is set", func() {
			Expect(cfg.Validate()).To(Succeed())
		})
		DescribeTable(
			"Should return an error when a field is invalid",
			func(mutate func(*auth.ServiceConfig), errorMsg string) {
				mutate(&cfg)
				Expect(cfg.Validate()).To(MatchError(ContainSubstring(errorMsg)))
			},
			Entry(
				"db",
				func(c *auth.ServiceConfig) { c.DB = nil },
				"db: must be non-nil",
			),
			Entry(
				"authenticators",
				func(c *auth.ServiceConfig) { c.Authenticators = nil },
				"authenticators: must not be empty",
			),
			Entry(
				"idle timeout",
				func(c *auth.ServiceConfig) { c.IdleTimeout = 0 },
				"idle_timeout",
			),
			Entry("clock", func(c *auth.ServiceConfig) { c.Clock = nil }, "clock"),
		)
	})
})

var _ = Describe("Service", func() {
	const idle = time.Hour
	var (
		svc      *auth.Service
		pw       *password.Service
		clock    *xtime.Fake
		alice    ontology.ID
		username string
	)
	login := func(ctx SpecContext) (auth.Session, string) {
		GinkgoHelper()
		return MustSucceed2(svc.Login(ctx, proof(username, "secret"), "10.0.0.1:1234"))
	}
	BeforeEach(func(ctx SpecContext) {
		// Every spec gets its own db so sweeps and clocks never cross.
		db := DeferClose(gorp.Wrap(memkv.New()))
		pw = MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
		clock = xtime.NewFake(time.Unix(1_700_000_000, 0))
		svc = MustOpen(auth.OpenService(ctx, auth.ServiceConfig{
			DB:             db,
			Authenticators: map[auth.Method]auth.Authenticator{auth.MethodPassword: pw},
			IdleTimeout:    idle,
			Clock:          clock,
		}))
		alice = subject()
		username = uuid.New().String()
		Expect(pw.NewWriter(nil).Set(ctx, alice, auth.Credentials{
			Username: username,
			Password: "secret",
		})).To(Succeed())
	})
	Describe("OpenService", func() {
		It("Should return an error when the config is invalid", func(ctx SpecContext) {
			Expect(auth.OpenService(ctx, auth.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
		})
	})
	Describe("Login", func() {
		It(
			"Should open a session for the identity the proof proves",
			func(ctx SpecContext) {
				session, token := login(ctx)
				Expect(token).To(HavePrefix(auth.TokenPrefix))
				Expect(session.Subject).To(Equal(alice))
				Expect(session.Method).To(Equal(auth.MethodPassword))
				Expect(session.Address).To(Equal("10.0.0.1:1234"))
				Expect(session.Created).To(Equal(session.LastActive))
				var stored auth.Session
				Expect(
					svc.NewRetrieve().Where(auth.MatchKeys(session.Key)).Entry(&stored).
						Exec(ctx, nil),
				).To(Succeed())
				Expect(stored).To(Equal(session))
				var p password.Password
				Expect(pw.NewRetrieve().Where(password.MatchSubjects(alice)).Entry(&p).
					Exec(ctx, nil)).To(Succeed())
				Expect(session.Credential).To(Equal(p.Key))
			},
		)
		It("Should never store the token", func(ctx SpecContext) {
			session, token := login(ctx)
			Expect(string(session.Hash)).ToNot(ContainSubstring(token))
			Expect(session.Hash).To(HaveLen(32))
		})
		It("Should return the authenticator's error", func(ctx SpecContext) {
			Expect(svc.Login(ctx, proof(username, "wrong"), "")).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should reject a proof with no method", func(ctx SpecContext) {
			Expect(svc.Login(ctx, auth.Proof{}, "")).Error().
				To(MatchError(validate.ErrValidation))
		})
		It("Should reject a method with no authenticator", func(ctx SpecContext) {
			lonely := MustOpen(auth.OpenService(ctx, auth.ServiceConfig{
				DB:             db,
				Authenticators: map[auth.Method]auth.Authenticator{"other": pw},
			}))
			Expect(
				lonely.Login(ctx, proof(username, "secret"), ""),
			).Error().
				To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("not accepted")),
				))
		})
	})
	Describe("Authenticate", func() {
		It("Should return the session a token belongs to", func(ctx SpecContext) {
			session, token := login(ctx)
			Expect(svc.Authenticate(ctx, token)).To(Equal(session))
		})
		DescribeTable(
			"Should reject a token that matches no session",
			func(ctx SpecContext, mangle func(string) string) {
				_, token := login(ctx)
				Expect(svc.Authenticate(ctx, mangle(token))).Error().
					To(MatchError(auth.ErrInvalidToken))
			},
			Entry("empty", func(string) string { return "" }),
			Entry(
				"no prefix",
				func(t string) string { return t[len(auth.TokenPrefix):] },
			),
			Entry("not base64", func(t string) string { return t + "!" }),
			Entry("too short", func(t string) string { return t[:len(t)-8] }),
			Entry("wrong secret", func(t string) string {
				b := []byte(t)
				last := len(b) - 1
				if b[last] == 'A' {
					b[last] = 'B'
				} else {
					b[last] = 'A'
				}
				return string(b)
			}),
		)
		It(
			"Should reject the token of a session that was logged out",
			func(ctx SpecContext) {
				session, token := login(ctx)
				Expect(svc.Logout(ctx, session.Key)).To(Succeed())
				Expect(
					svc.Authenticate(ctx, token),
				).Error().
					To(MatchError(auth.ErrInvalidToken))
			},
		)
		It("Should end a session idle past the timeout", func(ctx SpecContext) {
			session, token := login(ctx)
			clock.Advance(idle + time.Second)
			Expect(
				svc.Authenticate(ctx, token),
			).Error().
				To(MatchError(auth.ErrExpiredToken))
			Expect(
				svc.NewRetrieve().Where(auth.MatchKeys(session.Key)).Exists(ctx, nil),
			).
				To(BeFalse())
		})
		It(
			"Should record activity once the stored mark is a quarter period old",
			func(ctx SpecContext) {
				session, token := login(ctx)
				clock.Advance(idle / 8)
				Expect(MustSucceed(svc.Authenticate(ctx, token)).LastActive).
					To(Equal(session.LastActive))
				clock.Advance(idle / 4)
				touched := MustSucceed(svc.Authenticate(ctx, token))
				Expect(touched.LastActive).To(BeNumerically(">", session.LastActive))
				var stored auth.Session
				Expect(
					svc.NewRetrieve().Where(auth.MatchKeys(session.Key)).Entry(&stored).
						Exec(ctx, nil),
				).To(Succeed())
				Expect(stored.LastActive).To(Equal(touched.LastActive))
			},
		)
		It(
			"Should count a request made before the stored mark moved",
			func(ctx SpecContext) {
				session, token := login(ctx)
				clock.Advance(idle/4 - time.Minute)
				MustSucceed(svc.Authenticate(ctx, token))
				clock.Advance(idle*3/4 + 2*time.Minute)
				MustSucceed(svc.Authenticate(ctx, token))
				clock.Advance(idle/4 - time.Minute)
				MustSucceed(svc.Authenticate(ctx, token))
				Expect(svc.Sweep(ctx)).To(Succeed())
				Expect(
					svc.NewRetrieve().
						Where(auth.MatchKeys(session.Key)).
						Exists(ctx, nil),
				).To(BeTrue())
			},
		)
		It(
			"Should keep an active session alive past the timeout",
			func(ctx SpecContext) {
				_, token := login(ctx)
				for range 5 {
					clock.Advance(idle / 2)
					MustSucceed(svc.Authenticate(ctx, token))
				}
			},
		)
	})
	Describe("Logout", func() {
		It("Should ignore unknown keys", func(ctx SpecContext) {
			Expect(svc.Logout(ctx, uuid.New())).To(Succeed())
		})
	})
	Describe("Sweep", func() {
		It("Should store the activity this node saw instead of ending the session",
			func(ctx SpecContext) {
				session, token := login(ctx)
				clock.Advance(idle/4 - time.Minute)
				MustSucceed(svc.Authenticate(ctx, token))
				seen := telem.NewTimeStamp(clock.Now())
				clock.Advance(idle*3/4 + 2*time.Minute)
				Expect(svc.Sweep(ctx)).To(Succeed())
				var stored auth.Session
				Expect(
					svc.NewRetrieve().Where(auth.MatchKeys(session.Key)).Entry(&stored).
						Exec(ctx, nil),
				).To(Succeed())
				Expect(stored.LastActive).To(Equal(seen))
			},
		)
		It("Should end idle sessions and keep active ones", func(ctx SpecContext) {
			stale, staleToken := login(ctx)
			clock.Advance(idle / 2)
			fresh, freshToken := login(ctx)
			clock.Advance(idle/2 + time.Second)
			Expect(svc.Sweep(ctx)).To(Succeed())
			Expect(svc.NewRetrieve().Where(auth.MatchKeys(stale.Key)).Exists(ctx, nil)).
				To(BeFalse())
			Expect(svc.Authenticate(ctx, staleToken)).Error().
				To(MatchError(auth.ErrInvalidToken))
			Expect(
				MustSucceed(svc.Authenticate(ctx, freshToken)).Key,
			).To(Equal(fresh.Key))
		})
	})
	Describe("Writer", func() {
		Describe("DeleteFor", func() {
			It(
				"Should remove the subject's credentials and sessions",
				func(ctx SpecContext) {
					session, token := login(ctx)
					bob := subject()
					bobName := uuid.New().String()
					Expect(pw.NewWriter(nil).Set(ctx, bob, auth.Credentials{
						Username: bobName,
						Password: "p",
					})).To(Succeed())
					_, bobToken := MustSucceed2(svc.Login(ctx, proof(bobName, "p"), ""))
					Expect(svc.NewWriter(nil).DeleteFor(ctx, alice)).To(Succeed())
					Expect(svc.Authenticate(ctx, token)).Error().
						To(MatchError(auth.ErrInvalidToken))
					Expect(svc.NewRetrieve().Where(auth.MatchSubjects(session.Subject)).
						Exists(ctx, nil)).To(BeFalse())
					Expect(pw.Authenticate(ctx, proof(username, "secret"))).Error().
						To(MatchError(auth.ErrInvalidCredentials))
					MustSucceed(svc.Authenticate(ctx, bobToken))
				},
			)
			It("Should run in the caller's transaction", func(ctx SpecContext) {
				_, token := login(ctx)
				tx := db.OpenTx()
				Expect(svc.NewWriter(tx).DeleteFor(ctx, alice)).To(Succeed())
				MustSucceed(svc.Authenticate(ctx, token))
				Expect(tx.Close()).To(Succeed())
				MustSucceed(svc.Authenticate(ctx, token))
			})
		})
		Describe("RevokeCredential", func() {
			It("Should end the credential's other sessions", func(ctx SpecContext) {
				kept, keptToken := login(ctx)
				_, otherToken := login(ctx)
				Expect(svc.NewWriter(nil).RevokeCredential(
					ctx,
					kept.Method,
					kept.Credential,
					kept.Key,
				)).To(Succeed())
				MustSucceed(svc.Authenticate(ctx, keptToken))
				Expect(svc.Authenticate(ctx, otherToken)).Error().
					To(MatchError(auth.ErrInvalidToken))
			})
			It("Should leave other credentials alone", func(ctx SpecContext) {
				_, token := login(ctx)
				Expect(svc.NewWriter(nil).RevokeCredential(
					ctx,
					auth.MethodPassword,
					uuid.New(),
				)).To(Succeed())
				MustSucceed(svc.Authenticate(ctx, token))
			})
		})
	})
	It("Should not find a session by a stale retrieve", func(ctx SpecContext) {
		Expect(
			svc.NewRetrieve().Where(auth.MatchKeys(uuid.New())).Entry(&auth.Session{}).
				Exec(ctx, nil),
		).To(MatchError(query.ErrNotFound))
	})
})
