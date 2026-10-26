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
	"sync"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
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

var _ = Describe("ServiceConfig", func() {
	var (
		cfg   auth.ServiceConfig
		auths map[auth.Method]auth.Authenticator
	)
	BeforeEach(func(ctx SpecContext) {
		pw := MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
		auths = map[auth.Method]auth.Authenticator{auth.MethodPassword: pw}
		cfg = auth.ServiceConfig{
			DB:                db,
			Authenticators:    auths,
			IdleTimeout:       time.Hour,
			Clock:             xtime.Real,
			HoldCheckInterval: time.Second,
		}
	})
	Describe("Override", func() {
		It("Should take each field from the override when the base is zero", func() {
			result := auth.ServiceConfig{}.Override(cfg)
			Expect(result.DB).To(Equal(db))
			Expect(result.Authenticators).To(Equal(auths))
			Expect(result.IdleTimeout).To(Equal(time.Hour))
			Expect(result.Clock).To(Equal(xtime.Real))
			Expect(result.HoldCheckInterval).To(Equal(time.Second))
		})
		It("Should keep each field from the base when the override is zero", func() {
			result := cfg.Override(auth.ServiceConfig{})
			Expect(result.DB).To(Equal(db))
			Expect(result.Authenticators).To(Equal(auths))
			Expect(result.IdleTimeout).To(Equal(time.Hour))
			Expect(result.Clock).To(Equal(xtime.Real))
			Expect(result.HoldCheckInterval).To(Equal(time.Second))
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
			Entry(
				"hold check interval",
				func(c *auth.ServiceConfig) { c.HoldCheckInterval = 0 },
				"hold_check_interval",
			),
		)
	})
})

var _ = Describe("Service", func() {
	const idle = time.Hour
	var (
		db       *gorp.DB
		svc      *auth.Service
		pw       *password.Service
		clock    *xtime.Fake
		alice    ontology.ID
		username string
	)
	login := func(ctx SpecContext) (auth.Session, string) {
		GinkgoHelper()
		return MustSucceed2(svc.Login(ctx, proof(username, "secret")))
	}
	BeforeEach(func(ctx SpecContext) {
		// Every spec gets its own db so sweeps and clocks never cross.
		db = DeferClose(gorp.Wrap(memkv.New()))
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
				Expect(token).To(HavePrefix("sys_"))
				Expect(session.Subject).To(Equal(alice))
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
		It("Should store the token's hash, never the token", func(ctx SpecContext) {
			session, token := login(ctx)
			var stored auth.Session
			Expect(
				svc.NewRetrieve().Where(auth.MatchKeys(session.Key)).Entry(&stored).
					Exec(ctx, nil),
			).To(Succeed())
			Expect(stored.Hash).To(HaveLen(32))
			Expect(stored.Hash).To(Equal(session.Hash))
			Expect(string(stored.Hash)).ToNot(ContainSubstring(token))
		})
		It("Should return the authenticator's error", func(ctx SpecContext) {
			Expect(svc.Login(ctx, proof(username, "wrong"))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should reject a proof with no method", func(ctx SpecContext) {
			Expect(svc.Login(ctx, auth.Proof{})).Error().
				To(MatchError(validate.ErrValidation))
		})
		It("Should reject a method with no authenticator", func(ctx SpecContext) {
			lonely := MustOpen(auth.OpenService(ctx, auth.ServiceConfig{
				DB:             db,
				Authenticators: map[auth.Method]auth.Authenticator{"other": pw},
			}))
			Expect(
				lonely.Login(ctx, proof(username, "secret")),
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
				func(t string) string { return t[len("sys_"):] },
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
	Describe("Hold", func() {
		It("Should reject a session that does not exist", func(ctx SpecContext) {
			Expect(
				svc.Hold(ctx, uuid.New()),
			).Error().
				To(MatchError(auth.ErrInvalidToken))
		})
		It(
			"Should end a hold whose session storage lost behind the feed",
			func(ctx SpecContext) {
				session, _ := login(ctx)
				ended, release := MustSucceed2(svc.Hold(ctx, session.Key))
				defer release()
				Expect(gorp.WrapWriter[auth.SessionKey, auth.Session](db).
					Delete(ctx, session.Key)).To(Succeed())
				Expect(svc.CheckHolds(ctx)).To(Succeed())
				Eventually(ended).Should(BeClosed())
			},
		)
		It("Should check holds when nothing is held", func(ctx SpecContext) {
			Expect(svc.CheckHolds(ctx)).To(Succeed())
		})
		It(
			"Should finish a sweep when a held session is gone",
			func(ctx SpecContext) {
				session, _ := login(ctx)
				ended, release := MustSucceed2(svc.Hold(ctx, session.Key))
				defer release()
				stale, _ := login(ctx)
				Expect(gorp.WrapWriter[auth.SessionKey, auth.Session](db).
					Delete(ctx, session.Key)).To(Succeed())
				clock.Advance(idle*5/4 + time.Second)
				Expect(svc.Sweep(ctx)).To(Succeed())
				Eventually(ended).Should(BeClosed())
				Expect(
					svc.NewRetrieve().Where(auth.MatchKeys(stale.Key)).Exists(ctx, nil),
				).
					To(BeFalse())
			},
		)
		It(
			"Should close the channel when the session is logged out",
			func(ctx SpecContext) {
				session, _ := login(ctx)
				ended, release := MustSucceed2(svc.Hold(ctx, session.Key))
				defer release()
				Consistently(ended).ShouldNot(BeClosed())
				Expect(svc.Logout(ctx, session.Key)).To(Succeed())
				Eventually(ended).Should(BeClosed())
			},
		)
		It(
			"Should close the channel when the subject is deleted",
			func(ctx SpecContext) {
				session, _ := login(ctx)
				ended, release := MustSucceed2(svc.Hold(ctx, session.Key))
				defer release()
				Expect(svc.NewWriter(nil).DeleteFor(ctx, alice)).To(Succeed())
				Eventually(ended).Should(BeClosed())
			},
		)
		It(
			"Should close the channel when the credential is revoked",
			func(ctx SpecContext) {
				session, _ := login(ctx)
				ended, release := MustSucceed2(svc.Hold(ctx, session.Key))
				defer release()
				Expect(svc.NewWriter(nil).RevokeCredential(
					ctx,
					nil,
					session.Credential,
				)).To(Succeed())
				Eventually(ended).Should(BeClosed())
			},
		)
		It(
			"Should share one channel between holders of a session",
			func(ctx SpecContext) {
				session, _ := login(ctx)
				first, releaseFirst := MustSucceed2(svc.Hold(ctx, session.Key))
				second, releaseSecond := MustSucceed2(svc.Hold(ctx, session.Key))
				releaseFirst()
				releaseFirst()
				Expect(svc.Logout(ctx, session.Key)).To(Succeed())
				Eventually(first).Should(BeClosed())
				Eventually(second).Should(BeClosed())
				releaseSecond()
			},
		)
		It(
			"Should keep a held session alive past the idle timeout",
			func(ctx SpecContext) {
				session, token := login(ctx)
				_, release := MustSucceed2(svc.Hold(ctx, session.Key))
				clock.Advance(idle / 2)
				Expect(svc.Sweep(ctx)).To(Succeed())
				var refreshed auth.Session
				Expect(svc.NewRetrieve().Where(auth.MatchKeys(session.Key)).
					Entry(&refreshed).Exec(ctx, nil)).To(Succeed())
				Expect(refreshed.LastActive).To(Equal(telem.NewTimeStamp(clock.Now())))
				clock.Advance(idle + time.Second)
				Expect(svc.Sweep(ctx)).To(Succeed())
				Expect(
					MustSucceed(svc.Authenticate(ctx, token)).Key,
				).To(Equal(session.Key))
				release()
				clock.Advance(idle*5/4 + time.Second)
				Expect(svc.Sweep(ctx)).To(Succeed())
				Expect(svc.Authenticate(ctx, token)).Error().
					To(MatchError(auth.ErrInvalidToken))
			},
		)
	})
	Describe("Sweep", func() {
		It("Should store the activity this node saw instead of ending the session",
			func(ctx SpecContext) {
				session, token := login(ctx)
				clock.Advance(idle/4 - time.Minute)
				MustSucceed(svc.Authenticate(ctx, token))
				seen := telem.NewTimeStamp(clock.Now())
				clock.Advance(idle + 2*time.Minute)
				Expect(svc.Sweep(ctx)).To(Succeed())
				var stored auth.Session
				Expect(
					svc.NewRetrieve().Where(auth.MatchKeys(session.Key)).Entry(&stored).
						Exec(ctx, nil),
				).To(Succeed())
				Expect(stored.LastActive).To(Equal(seen))
			},
		)
		It(
			"Should keep a session another node may have seen within the last quarter",
			func(ctx SpecContext) {
				session, _ := login(ctx)
				clock.Advance(idle + time.Minute)
				Expect(svc.Sweep(ctx)).To(Succeed())
				Expect(
					svc.NewRetrieve().
						Where(auth.MatchKeys(session.Key)).
						Exists(ctx, nil),
				).To(BeTrue())
				clock.Advance(idle / 4)
				Expect(svc.Sweep(ctx)).To(Succeed())
				Expect(
					svc.NewRetrieve().
						Where(auth.MatchKeys(session.Key)).
						Exists(ctx, nil),
				).To(BeFalse())
			},
		)
		It(
			"Should mark the other held sessions when one ends during the sweep",
			func(ctx SpecContext) {
				kept, _ := login(ctx)
				_, release := MustSucceed2(svc.Hold(ctx, kept.Key))
				defer release()
				for range 50 {
					ended, _ := login(ctx)
					_, releaseEnded := MustSucceed2(svc.Hold(ctx, ended.Key))
					clock.Advance(time.Second)
					var wg sync.WaitGroup
					wg.Go(func() {
						defer GinkgoRecover()
						Expect(svc.Logout(ctx, ended.Key)).To(Succeed())
					})
					Expect(svc.Sweep(ctx)).To(Succeed())
					wg.Wait()
					releaseEnded()
					var stored auth.Session
					Expect(svc.NewRetrieve().Where(auth.MatchKeys(kept.Key)).
						Entry(&stored).Exec(ctx, nil)).To(Succeed())
					Expect(stored.LastActive).To(Equal(telem.NewTimeStamp(clock.Now())))
				}
			},
		)
		It("Should end idle sessions and keep active ones", func(ctx SpecContext) {
			stale, staleToken := login(ctx)
			clock.Advance(idle / 2)
			fresh, freshToken := login(ctx)
			clock.Advance(idle*3/4 + time.Second)
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
					_, bobToken := MustSucceed2(svc.Login(ctx, proof(bobName, "p")))
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
				session, token := login(ctx)
				tx := db.OpenTx()
				Expect(svc.NewWriter(tx).DeleteFor(ctx, alice)).To(Succeed())
				Expect(svc.NewRetrieve().Where(auth.MatchKeys(session.Key)).
					Exists(ctx, tx)).To(BeFalse())
				Expect(pw.NewRetrieve().Where(password.MatchSubjects(alice)).
					Exists(ctx, tx)).To(BeFalse())
				MustSucceed(svc.Authenticate(ctx, token))
				Expect(tx.Close()).To(Succeed())
				MustSucceed(svc.Authenticate(ctx, token))
				Expect(pw.NewRetrieve().Where(password.MatchSubjects(alice)).
					Exists(ctx, nil)).To(BeTrue())
			})
		})
		Describe("RevokeCredential", func() {
			It("Should end the credential's other sessions", func(ctx SpecContext) {
				kept, keptToken := login(ctx)
				_, otherToken := login(ctx)
				Expect(svc.NewWriter(nil).RevokeCredential(
					ctx,
					[]auth.SessionKey{kept.Key},
					kept.Credential,
				)).To(Succeed())
				MustSucceed(svc.Authenticate(ctx, keptToken))
				Expect(svc.Authenticate(ctx, otherToken)).Error().
					To(MatchError(auth.ErrInvalidToken))
			})
			It("Should end the sessions of several credentials", func(ctx SpecContext) {
				a, aToken := login(ctx)
				bobName := uuid.New().String()
				Expect(pw.NewWriter(nil).Set(ctx, subject(), auth.Credentials{
					Username: bobName,
					Password: "secret",
				})).To(Succeed())
				b, bToken := MustSucceed2(
					svc.Login(ctx, proof(bobName, "secret")),
				)
				Expect(svc.NewWriter(nil).RevokeCredential(
					ctx,
					nil,
					a.Credential,
					b.Credential,
				)).To(Succeed())
				for _, token := range []string{aToken, bToken} {
					Expect(svc.Authenticate(ctx, token)).Error().
						To(MatchError(auth.ErrInvalidToken))
				}
			})
			It("Should leave other credentials alone", func(ctx SpecContext) {
				_, token := login(ctx)
				Expect(svc.NewWriter(nil).RevokeCredential(
					ctx,
					nil,
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
