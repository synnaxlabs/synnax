// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package invite_test

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc/invite"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	timetestutil "github.com/synnaxlabs/x/time/testutil"
	"github.com/synnaxlabs/x/validate"
)

func noop(gorp.Tx, invite.Invite) error { return nil }

var _ = Describe("Service", func() {
	var (
		clock *timetestutil.Clock
		svc   *invite.Service
	)
	exists := func(ctx SpecContext, u user.Key) bool {
		GinkgoHelper()
		return MustSucceed(
			svc.NewRetrieve().Where(invite.MatchUsers(u)).Exists(ctx, db),
		)
	}
	create := func(ctx SpecContext, u user.Key) (invite.Invite, string) {
		GinkgoHelper()
		return MustSucceed2(svc.NewWriter(db).Create(ctx, u))
	}
	BeforeEach(func(ctx SpecContext) {
		clock = timetestutil.NewClock(time.Unix(1_700_000_000, 0))
		svc = MustOpen(invite.OpenService(ctx, invite.ServiceConfig{
			DB:    db,
			Clock: clock,
		}))
	})

	Describe("OpenService", func() {
		It("Should default to the system clock", func(ctx SpecContext) {
			other := DeferClose(gorp.Wrap(memkv.New()))
			StartRun(other.Run)
			MustOpen(invite.OpenService(ctx, invite.ServiceConfig{DB: other}))
		})
		It("Should fail without a DB", func(ctx SpecContext) {
			Expect(invite.OpenService(ctx)).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
		})
	})

	Describe("Create", func() {
		It(
			"Should store only a hash of a token of 256 random bits",
			func(ctx SpecContext) {
				u := createUser(ctx)
				i, token := create(ctx, u)
				Expect(i.User).To(Equal(u))
				Expect(i.Hash).To(HaveLen(32))
				raw := MustSucceed(base64.RawURLEncoding.DecodeString(
					strings.TrimPrefix(token, "syi_"),
				))
				Expect(raw).To(HaveLen(16 + 32))
				Expect(raw[16:]).ToNot(Equal(i.Hash))
			},
		)
		It("Should expire a day after creation", func(ctx SpecContext) {
			i, _ := create(ctx, createUser(ctx))
			Expect(i.Expiry.Time()).To(BeTemporally(
				"==",
				clock.Now().Add(invite.Lifetime),
			))
			Expect(invite.Lifetime).To(Equal(24 * time.Hour))
		})
		It("Should replace a user's invite", func(ctx SpecContext) {
			u := createUser(ctx)
			first, old := create(ctx, u)
			second, current := create(ctx, u)
			Expect(second.Key).To(Equal(first.Key))
			Expect(MustSucceed(
				svc.NewRetrieve().Where(invite.MatchUsers(u)).Count(ctx, db),
			)).To(Equal(1))
			Expect(
				svc.Redeem(ctx, old, noop),
			).To(MatchError(auth.ErrInvalidCredentials))
			Expect(svc.Redeem(ctx, current, noop)).To(Succeed())
		})
		It("Should give each user its own invite", func(ctx SpecContext) {
			a, _ := create(ctx, createUser(ctx))
			b, _ := create(ctx, createUser(ctx))
			Expect(a.Key).ToNot(Equal(b.Key))
		})
		It("Should reject an unknown user", func(ctx SpecContext) {
			Expect(svc.NewWriter(db).Create(ctx, uuid.New())).Error().
				To(MatchError(validate.ErrValidation))
		})
	})

	Describe("Writer", func() {
		It("Should delete invites by key", func(ctx SpecContext) {
			u := createUser(ctx)
			i, _ := create(ctx, u)
			Expect(svc.NewWriter(db).Delete(ctx, i.Key, uuid.New())).To(Succeed())
			Expect(exists(ctx, u)).To(BeFalse())
		})
		It("Should delete the invite of a deleted user", func(ctx SpecContext) {
			u, kept := createUser(ctx), createUser(ctx)
			create(ctx, u)
			create(ctx, kept)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return userSvc.NewWriter(tx).Delete(ctx, u)
			})).To(Succeed())
			Expect(exists(ctx, u)).To(BeFalse())
			Expect(exists(ctx, kept)).To(BeTrue())
		})
	})

	Describe("MatchLive", func() {
		It("Should leave out an invite at its expiry", func(ctx SpecContext) {
			u := createUser(ctx)
			create(ctx, u)
			live := func() bool {
				GinkgoHelper()
				return MustSucceed(svc.NewRetrieve().
					Where(invite.MatchUsers(u)).
					Where(svc.MatchLive()).
					Exists(ctx, db))
			}
			clock.Advance(invite.Lifetime - time.Second)
			Expect(live()).To(BeTrue())
			clock.Advance(time.Second)
			Expect(live()).To(BeFalse())
		})
	})

	Describe("Redeem", func() {
		It("Should run f with the invite and delete it", func(ctx SpecContext) {
			u := createUser(ctx)
			i, token := create(ctx, u)
			var seen invite.Invite
			Expect(svc.Redeem(ctx, token, func(_ gorp.Tx, got invite.Invite) error {
				seen = got
				return nil
			})).To(Succeed())
			Expect(seen.Key).To(Equal(i.Key))
			Expect(exists(ctx, u)).To(BeFalse())
		})
		It("Should work once", func(ctx SpecContext) {
			_, token := create(ctx, createUser(ctx))
			Expect(svc.Redeem(ctx, token, noop)).To(Succeed())
			Expect(svc.Redeem(ctx, token, noop)).
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should work once when redeemed at the same time", func(ctx SpecContext) {
			_, token := create(ctx, createUser(ctx))
			var (
				wg       sync.WaitGroup
				redeemed atomic.Int32
			)
			for range 16 {
				wg.Go(func() {
					defer GinkgoRecover()
					if err := svc.Redeem(ctx, token, noop); err == nil {
						redeemed.Add(1)
					} else {
						Expect(err).To(MatchError(auth.ErrInvalidCredentials))
					}
				})
			}
			wg.Wait()
			Expect(redeemed.Load()).To(Equal(int32(1)))
		})
		It("Should keep the invite when f fails", func(ctx SpecContext) {
			u := createUser(ctx)
			_, token := create(ctx, u)
			failed := errors.New("failed")
			Expect(svc.Redeem(ctx, token, func(gorp.Tx, invite.Invite) error {
				return failed
			})).To(MatchError(failed))
			Expect(exists(ctx, u)).To(BeTrue())
		})
		It("Should keep an invite created during a redeem", func(ctx SpecContext) {
			u := createUser(ctx)
			_, old := create(ctx, u)
			created := make(chan string, 1)
			Expect(svc.Redeem(ctx, old, func(gorp.Tx, invite.Invite) error {
				go func() {
					defer GinkgoRecover()
					_, token := create(ctx, u)
					created <- token
				}()
				Consistently(created, "50ms").ShouldNot(Receive())
				return nil
			})).To(Succeed())
			var current string
			Eventually(created).Should(Receive(&current))
			Expect(svc.Redeem(ctx, current, noop)).To(Succeed())
		})
		DescribeTable(
			"Should give the same error for each token it refuses",
			func(ctx SpecContext, token func(SpecContext) string) {
				err := svc.Redeem(ctx, token(ctx), noop)
				Expect(err).To(BeIdenticalTo(auth.ErrInvalidCredentials))
			},
			Entry("expired", func(ctx SpecContext) string {
				GinkgoHelper()
				_, token := create(ctx, createUser(ctx))
				clock.Advance(invite.Lifetime)
				return token
			}),
			Entry("wrong secret", func(ctx SpecContext) string {
				GinkgoHelper()
				_, token := create(ctx, createUser(ctx))
				secret := make([]byte, 32)
				MustSucceed(rand.Read(secret))
				key := MustSucceed(base64.RawURLEncoding.DecodeString(
					strings.TrimPrefix(token, "syi_"),
				))[:16]
				return "syi_" + base64.RawURLEncoding.EncodeToString(
					append(key, secret...),
				)
			}),
			Entry("unknown", func(SpecContext) string {
				GinkgoHelper()
				token, _ := MustSucceed2(auth.NewToken("syi_", uuid.New()))
				return token
			}),
			Entry("malformed", func(SpecContext) string { return "syi_malformed" }),
			Entry("empty", func(SpecContext) string { return "" }),
		)
	})
})
