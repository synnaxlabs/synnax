// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package apikey_test

import (
	"context"
	"strings"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	xtime "github.com/synnaxlabs/x/time"
	timetestutil "github.com/synnaxlabs/x/time/testutil"
	"github.com/synnaxlabs/x/validate"
)

func create(
	ctx context.Context,
	svc *apikey.Service,
	sub resource.ID,
	name string,
) (apikey.APIKey, string) {
	GinkgoHelper()
	var (
		k     apikey.APIKey
		token string
	)
	Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
		k, token, err = svc.NewWriter(tx).Create(ctx, sub, name, 0)
		return err
	})).To(Succeed())
	return k, token
}

func proof(key string) auth.Proof {
	return auth.Proof{Variant: auth.APIKeyProof{
		APIKeyCredentials: auth.APIKeyCredentials{Key: key},
	}}
}

var _ = Describe("ServiceConfig", func() {
	var cfg apikey.ServiceConfig
	BeforeEach(func() {
		cfg = apikey.ServiceConfig{
			DB:              db,
			Clock:           xtime.Real,
			Instrumentation: alamos.New("apikey-config-test"),
		}
	})
	Describe("Override", func() {
		It("Should take each field from the override when the base is zero", func() {
			result := apikey.ServiceConfig{}.Override(cfg)
			Expect(result.DB).To(Equal(db))
			Expect(result.Clock).To(Equal(xtime.Real))
			Expect(result.Instrumentation).To(Equal(cfg.Instrumentation))
		})
		It("Should keep each field from the base when the override is zero", func() {
			result := cfg.Override(apikey.ServiceConfig{})
			Expect(result.DB).To(Equal(db))
			Expect(result.Clock).To(Equal(xtime.Real))
			Expect(result.Instrumentation).To(Equal(cfg.Instrumentation))
		})
	})
})

var _ = Describe("Service", func() {
	var (
		svc   *apikey.Service
		clock *timetestutil.Clock
	)
	BeforeEach(func(ctx SpecContext) {
		clock = timetestutil.NewClock(time.Unix(1_700_000_000, 0))
		svc = MustOpen(apikey.OpenService(ctx, apikey.ServiceConfig{
			DB:    db,
			Clock: clock,
		}))
	})
	It("Should require a DB", func(ctx SpecContext) {
		Expect(apikey.OpenService(ctx)).Error().
			To(MatchError(ContainSubstring("db: must be non-nil")))
	})
	Describe("Create", func() {
		It("Should return a key that starts with syk_", func(ctx SpecContext) {
			sub := subject()
			k, token := create(ctx, svc, sub, "ci")
			Expect(token).To(HavePrefix("syk_"))
			Expect(k.Subject).To(Equal(sub))
			Expect(k.Name).To(Equal("ci"))
			Expect(k.Created).ToNot(BeZero())
			var stored apikey.APIKey
			Expect(svc.NewRetrieve().Where(apikey.MatchKeys(k.Key)).Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			Expect(stored.Hash).To(HaveLen(32))
		})
		It("Should reject an invalid subject", func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				_, _, err := svc.NewWriter(tx).Create(ctx, resource.ID{}, "", 0)
				return err
			})).
				To(MatchError(ContainSubstring("subject")))
		})
	})
	Describe("Authenticate", func() {
		It("Should prove the key's subject and credential", func(ctx SpecContext) {
			sub := subject()
			k, token := create(ctx, svc, sub, "")
			Expect(svc.Authenticate(ctx, proof(token))).To(Equal(auth.Identity{
				Subject:    sub,
				Credential: k.Key,
				Version:    k.Created,
			}))
		})
		It("Should reject a wrong secret", func(ctx SpecContext) {
			k, _ := create(ctx, svc, subject(), "")
			wrong, _ := MustSucceed2(auth.NewToken("syk_", k.Key))
			Expect(svc.Authenticate(ctx, proof(wrong))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should reject an unknown key", func(ctx SpecContext) {
			token, _ := MustSucceed2(auth.NewToken("syk_", uuid.New()))
			Expect(svc.Authenticate(ctx, proof(token))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should reject a malformed key", func(ctx SpecContext) {
			_, token := create(ctx, svc, subject(), "")
			session := strings.Replace(token, "syk_", "sys_", 1)
			Expect(svc.Authenticate(ctx, proof(session))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should reject an empty key", func(ctx SpecContext) {
			Expect(svc.Authenticate(ctx, proof(""))).Error().
				To(MatchError(ContainSubstring("key: required")))
		})
		It("Should reject a proof for another method", func(ctx SpecContext) {
			Expect(svc.Authenticate(ctx, auth.Proof{Variant: auth.PasswordProof{}})).
				Error().To(MatchError(validate.ErrValidation))
		})
	})
	Describe("Version", func() {
		It("Should return when the key was created", func(ctx SpecContext) {
			k, _ := create(ctx, svc, subject(), "")
			Expect(svc.Version(ctx, db, k.Key)).To(Equal(k.Created))
		})
		It("Should return a not found error for an unknown key", func(ctx SpecContext) {
			Expect(svc.Version(ctx, db, uuid.New())).Error().
				To(MatchError(query.ErrNotFound))
		})
	})
	Describe("Delete", func() {
		It("Should remove only the given keys", func(ctx SpecContext) {
			sub := subject()
			gone, goneToken := create(ctx, svc, sub, "gone")
			_, keptToken := create(ctx, svc, sub, "kept")
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, gone.Key)
			})).To(Succeed())
			Expect(svc.NewRetrieve().Where(apikey.MatchKeys(gone.Key)).
				Exec(ctx, db)).To(MatchError(query.ErrNotFound))
			Expect(svc.Authenticate(ctx, proof(goneToken))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
			MustSucceed(svc.Authenticate(ctx, proof(keptToken)))
		})
		It("Should ignore unknown keys", func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, uuid.New())
			})).To(Succeed())
		})
	})
	Describe("Expiry", func() {
		var (
			expires telem.TimeStamp
			k       apikey.APIKey
			token   string
		)
		BeforeEach(func(ctx SpecContext) {
			expires = telem.NewTimeStamp(clock.Now().Add(time.Hour))
			sub := subject()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
				k, token, err = svc.NewWriter(tx).
					Create(ctx, sub, "expiring", expires)
				return err
			})).To(Succeed())
		})
		It("Should store the expiry", func() {
			Expect(k.Expires).To(Equal(expires))
		})
		It("Should log in until the key expires", func(ctx SpecContext) {
			clock.Advance(time.Hour - time.Second)
			MustSucceed(svc.Authenticate(ctx, proof(token)))
			Expect(svc.Version(ctx, db, k.Key)).To(Equal(k.Created))
		})
		It("Should reject an expired key", func(ctx SpecContext) {
			clock.Advance(time.Hour)
			Expect(svc.Authenticate(ctx, proof(token))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
			Expect(svc.Version(ctx, db, k.Key)).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should reject an expiry that is not in the future", func(ctx SpecContext) {
			sub, now := subject(), telem.NewTimeStamp(clock.Now())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				_, _, err := svc.NewWriter(tx).Create(ctx, sub, "", now)
				return err
			})).To(MatchError(ContainSubstring("expires: must be in the future")))
		})
		It("Should give a key with a zero expiry the maximum expiry", func(
			ctx SpecContext,
		) {
			never, _ := create(ctx, svc, subject(), "never")
			Expect(never.Expires).To(Equal(telem.TimeStampMax))
		})
		It("Should never expire a key with the maximum expiry", func(ctx SpecContext) {
			sub := subject()
			var never string
			Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
				_, never, err = svc.NewWriter(tx).
					Create(ctx, sub, "never", telem.TimeStampMax)
				return err
			})).To(Succeed())
			clock.Advance(100 * 365 * 24 * time.Hour)
			MustSucceed(svc.Authenticate(ctx, proof(never)))
		})
	})
	Describe("ValidateLifetime", func() {
		It("Should accept any expiry when there is no maximum lifetime", func() {
			Expect(svc.MaxLifetime()).To(Equal(telem.TimeSpan(0)))
			Expect(svc.ValidateLifetime(0)).To(Succeed())
			Expect(svc.ValidateLifetime(telem.TimeStampMax)).To(Succeed())
		})
		Describe("With a maximum lifetime", func() {
			var (
				capped *apikey.Service
				limit  telem.TimeStamp
			)
			BeforeEach(func(ctx SpecContext) {
				other := DeferClose(gorp.Wrap(memkv.New()))
				StartRun(other.Run)
				capped = MustOpen(apikey.OpenService(ctx, apikey.ServiceConfig{
					DB:          other,
					Clock:       clock,
					MaxLifetime: telem.Day,
				}))
				limit = telem.NewTimeStamp(clock.Now()).Add(telem.Day)
			})
			It("Should report the maximum lifetime", func() {
				Expect(capped.MaxLifetime()).To(Equal(telem.Day))
			})
			It("Should accept an expiry at the maximum lifetime", func() {
				Expect(capped.ValidateLifetime(limit)).To(Succeed())
			})
			DescribeTable(
				"Should reject an expiry past the maximum lifetime",
				func(expires func() telem.TimeStamp) {
					Expect(capped.ValidateLifetime(expires())).To(MatchError(
						ContainSubstring("expires: must be at most"),
					))
				},
				Entry("no expiry", func() telem.TimeStamp { return 0 }),
				Entry("a second past the limit", func() telem.TimeStamp {
					return limit.Add(telem.Second)
				}),
				Entry("the maximum timestamp", func() telem.TimeStamp {
					return telem.TimeStampMax
				}),
			)
		})
	})
	Describe("Subject", func() {
		It("Should delete every key of a subject when the subject is deleted", func(
			ctx SpecContext,
		) {
			gone, kept := subject(), subject()
			create(ctx, svc, gone, "a")
			create(ctx, svc, gone, "b")
			create(ctx, svc, kept, "c")
			deleteSubject(ctx, gone)
			var keys []apikey.APIKey
			Expect(svc.NewRetrieve().Where(apikey.MatchSubjects(gone, kept)).
				Entries(&keys).Exec(ctx, db)).To(Succeed())
			Expect(keys).To(HaveLen(1))
			Expect(keys[0].Subject).To(Equal(kept))
		})
	})
	Describe("Rotate", func() {
		rotate := func(
			ctx context.Context,
			key apikey.Key,
		) (apikey.APIKey, string, error) {
			var (
				k     apikey.APIKey
				token string
			)
			err := db.WithTx(ctx, func(tx gorp.Tx) (err error) {
				k, token, err = svc.NewWriter(tx).Rotate(ctx, key)
				return err
			})
			return k, token, err
		}
		createExpiring := func(
			ctx context.Context,
			lifetime time.Duration,
		) apikey.APIKey {
			GinkgoHelper()
			sub, expires := subject(), telem.NewTimeStamp(clock.Now().Add(lifetime))
			var k apikey.APIKey
			Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
				k, _, err = svc.NewWriter(tx).Create(ctx, sub, "rotated", expires)
				return err
			})).To(Succeed())
			return k
		}
		It("Should give the key a new secret and keep its key, subject, and name", func(
			ctx SpecContext,
		) {
			old, oldToken := create(ctx, svc, subject(), "rotated")
			clock.Advance(time.Hour)
			k, token := MustSucceed2(rotate(ctx, old.Key))
			Expect(k.Key).To(Equal(old.Key))
			Expect(k.Created).To(Equal(old.Created.Add(telem.Hour)))
			Expect(k.Subject).To(Equal(old.Subject))
			Expect(k.Name).To(Equal("rotated"))
			Expect(k.Expires).To(Equal(telem.TimeStampMax))
			Expect(svc.Authenticate(ctx, proof(oldToken))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
			Expect(MustSucceed(svc.Authenticate(ctx, proof(token))).Credential).
				To(Equal(k.Key))
		})
		It(
			"Should give the new key the lifetime of the old key",
			func(ctx SpecContext) {
				old := createExpiring(ctx, 30*24*time.Hour)
				clock.Advance(20 * 24 * time.Hour)
				k, _ := MustSucceed2(rotate(ctx, old.Key))
				Expect(k.Created).To(Equal(telem.NewTimeStamp(clock.Now())))
				lifetime := old.Created.Span(old.Expires)
				Expect(k.Expires).To(Equal(k.Created.Add(lifetime)))
			},
		)
		It("Should refuse an expired key", func(ctx SpecContext) {
			old := createExpiring(ctx, time.Hour)
			clock.Advance(time.Hour)
			Expect(rotate(ctx, old.Key)).Error().
				To(MatchError(auth.ErrInvalidCredentials))
			Expect(svc.NewRetrieve().Where(apikey.MatchKeys(old.Key)).Exists(ctx, db)).
				To(BeTrue())
		})
		It("Should return not found for an unknown key", func(ctx SpecContext) {
			Expect(rotate(ctx, uuid.New())).Error().To(MatchError(query.ErrNotFound))
		})
	})
})

var _ = Describe("Service with a maximum lifetime", func() {
	var (
		svc   *apikey.Service
		clock *timetestutil.Clock
	)
	BeforeEach(func(ctx SpecContext) {
		clock = timetestutil.NewClock(time.Unix(1_700_000_000, 0))
		svc = MustOpen(apikey.OpenService(ctx, apikey.ServiceConfig{
			DB:          db,
			Clock:       clock,
			MaxLifetime: telem.Day,
		}))
	})
	DescribeTable(
		"Should give a rotated key the longest expiry",
		func(ctx SpecContext, lifetime telem.TimeSpan) {
			expires := telem.TimeStampMax
			if lifetime != 0 {
				expires = telem.NewTimeStamp(clock.Now()).Add(lifetime)
			}
			sub := subject()
			var k apikey.APIKey
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				w := svc.NewWriter(tx)
				old, _, err := w.Create(ctx, sub, "rotated", expires)
				if err != nil {
					return err
				}
				k, _, err = w.Rotate(ctx, old.Key)
				return err
			})).To(Succeed())
			Expect(k.Expires).To(Equal(k.Created.Add(telem.Day - telem.Hour)))
		},
		Entry("for a key with a shorter lifetime", telem.Hour),
		Entry("for a key that never expires", telem.TimeSpan(0)),
	)
	It("Should stop the longest expiry an hour short of the maximum lifetime", func() {
		Expect(svc.LongestExpiry()).
			To(Equal(telem.NewTimeStamp(clock.Now()).Add(telem.Day - telem.Hour)))
		Expect(svc.ValidateLifetime(svc.LongestExpiry())).To(Succeed())
	})
	It("Should stop half a short maximum lifetime short", func(ctx SpecContext) {
		other := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(other.Run)
		short := MustOpen(apikey.OpenService(ctx, apikey.ServiceConfig{
			DB:          other,
			Clock:       clock,
			MaxLifetime: 90 * telem.Minute,
		}))
		Expect(short.LongestExpiry()).
			To(Equal(telem.NewTimeStamp(clock.Now()).Add(45 * telem.Minute)))
	})
})
