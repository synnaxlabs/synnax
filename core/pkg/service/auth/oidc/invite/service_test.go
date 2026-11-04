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
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	timetestutil "github.com/synnaxlabs/x/time/testutil"
)

func subject() ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeUser, Key: uuid.New().String()}
}

func noop(gorp.Tx, invite.Invite) error { return nil }

var _ = Describe("Service", func() {
	var (
		clock *timetestutil.Clock
		svc   *invite.Service
	)
	exists := func(ctx SpecContext, s ontology.ID) bool {
		GinkgoHelper()
		return MustSucceed(
			svc.NewRetrieve().Where(invite.MatchSubjects(s)).Exists(ctx, nil),
		)
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
			MustOpen(invite.OpenService(ctx, invite.ServiceConfig{DB: db}))
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
				s := subject()
				i, token := MustSucceed2(svc.Create(ctx, s))
				Expect(i.Subject).To(Equal(s))
				Expect(i.Hash).To(HaveLen(32))
				raw := MustSucceed(base64.RawURLEncoding.DecodeString(
					strings.TrimPrefix(token, "syi_"),
				))
				Expect(raw).To(HaveLen(16 + 32))
				Expect(raw[16:]).ToNot(Equal(i.Hash))
			},
		)
		It("Should expire a day after creation", func(ctx SpecContext) {
			i, _ := MustSucceed2(svc.Create(ctx, subject()))
			Expect(i.Expiry.Time()).To(BeTemporally(
				"==",
				clock.Now().Add(invite.Lifetime),
			))
			Expect(invite.Lifetime).To(Equal(24 * time.Hour))
		})
		It("Should replace a subject's invite", func(ctx SpecContext) {
			s := subject()
			first, old := MustSucceed2(svc.Create(ctx, s))
			second, current := MustSucceed2(svc.Create(ctx, s))
			Expect(second.Key).To(Equal(first.Key))
			Expect(MustSucceed(
				svc.NewRetrieve().Where(invite.MatchSubjects(s)).Count(ctx, nil),
			)).To(Equal(1))
			Expect(
				svc.Redeem(ctx, old, noop),
			).To(MatchError(auth.ErrInvalidCredentials))
			Expect(svc.Redeem(ctx, current, noop)).To(Succeed())
		})
		It("Should give each subject its own invite", func(ctx SpecContext) {
			a, _ := MustSucceed2(svc.Create(ctx, subject()))
			b, _ := MustSucceed2(svc.Create(ctx, subject()))
			Expect(a.Key).ToNot(Equal(b.Key))
		})
		It("Should reject an invalid subject", func(ctx SpecContext) {
			Expect(svc.Create(ctx, ontology.ID{})).Error().
				To(MatchError(ContainSubstring("subject")))
		})
	})

	Describe("Writer", func() {
		It("Should delete invites by key", func(ctx SpecContext) {
			s := subject()
			i, _ := MustSucceed2(svc.Create(ctx, s))
			Expect(svc.NewWriter(nil).Delete(ctx, i.Key, uuid.New())).To(Succeed())
			Expect(exists(ctx, s)).To(BeFalse())
		})
		It("Should delete every invite of the given subjects", func(ctx SpecContext) {
			s, kept := subject(), subject()
			MustSucceed2(svc.Create(ctx, s))
			MustSucceed2(svc.Create(ctx, kept))
			Expect(svc.NewWriter(nil).DeleteFor(ctx, s)).To(Succeed())
			Expect(exists(ctx, s)).To(BeFalse())
			Expect(exists(ctx, kept)).To(BeTrue())
		})
	})

	Describe("MatchLive", func() {
		It("Should leave out an invite at its expiry", func(ctx SpecContext) {
			s := subject()
			MustSucceed2(svc.Create(ctx, s))
			live := func() bool {
				GinkgoHelper()
				return MustSucceed(svc.NewRetrieve().
					Where(invite.MatchSubjects(s)).
					Where(svc.MatchLive()).
					Exists(ctx, nil))
			}
			clock.Advance(invite.Lifetime - time.Second)
			Expect(live()).To(BeTrue())
			clock.Advance(time.Second)
			Expect(live()).To(BeFalse())
		})
	})

	Describe("Redeem", func() {
		It("Should run f with the invite and delete it", func(ctx SpecContext) {
			s := subject()
			i, token := MustSucceed2(svc.Create(ctx, s))
			var seen invite.Invite
			Expect(svc.Redeem(ctx, token, func(_ gorp.Tx, got invite.Invite) error {
				seen = got
				return nil
			})).To(Succeed())
			Expect(seen.Key).To(Equal(i.Key))
			Expect(exists(ctx, s)).To(BeFalse())
		})
		It("Should work once", func(ctx SpecContext) {
			_, token := MustSucceed2(svc.Create(ctx, subject()))
			Expect(svc.Redeem(ctx, token, noop)).To(Succeed())
			Expect(svc.Redeem(ctx, token, noop)).
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should work once when redeemed at the same time", func(ctx SpecContext) {
			_, token := MustSucceed2(svc.Create(ctx, subject()))
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
			s := subject()
			_, token := MustSucceed2(svc.Create(ctx, s))
			failed := errors.New("failed")
			Expect(svc.Redeem(ctx, token, func(gorp.Tx, invite.Invite) error {
				return failed
			})).To(MatchError(failed))
			Expect(exists(ctx, s)).To(BeTrue())
		})
		It("Should keep an invite created during a redeem", func(ctx SpecContext) {
			s := subject()
			_, old := MustSucceed2(svc.Create(ctx, s))
			created := make(chan string, 1)
			Expect(svc.Redeem(ctx, old, func(gorp.Tx, invite.Invite) error {
				go func() {
					defer GinkgoRecover()
					_, token := MustSucceed2(svc.Create(ctx, s))
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
				_, token := MustSucceed2(svc.Create(ctx, subject()))
				clock.Advance(invite.Lifetime)
				return token
			}),
			Entry("wrong secret", func(ctx SpecContext) string {
				GinkgoHelper()
				_, token := MustSucceed2(svc.Create(ctx, subject()))
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
