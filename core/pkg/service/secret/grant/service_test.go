// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package grant_test

import (
	"sync"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/secret/grant"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	xtime "github.com/synnaxlabs/x/time"
	timetestutil "github.com/synnaxlabs/x/time/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("ServiceConfig", func() {
	full := func() grant.ServiceConfig {
		return grant.ServiceConfig{DB: &gorp.DB{}, TTL: time.Minute, Clock: xtime.Real}
	}
	DescribeTable(
		"Should fail when a field is not valid",
		func(change func(*grant.ServiceConfig), msg string) {
			cfg := full()
			change(&cfg)
			Expect(cfg.Validate()).To(SatisfyAll(
				BeAssignableToTypeOf(validate.PathError{}),
				MatchError(msg),
			))
		},
		Entry("db", func(c *grant.ServiceConfig) { c.DB = nil }, "db: must be non-nil"),
		Entry(
			"ttl",
			func(c *grant.ServiceConfig) { c.TTL = 0 },
			"ttl: must be positive",
		),
		Entry(
			"clock",
			func(c *grant.ServiceConfig) { c.Clock = nil },
			"clock: must be non-nil",
		),
	)
	It("Should take the other config's fields when set", func() {
		cfg := full()
		Expect(cfg.Validate()).To(Succeed())
		merged := grant.ServiceConfig{}.Override(cfg)
		Expect(merged.DB).To(BeIdenticalTo(cfg.DB))
		Expect(merged.TTL).To(Equal(time.Minute))
		Expect(merged.Clock).To(BeIdenticalTo(cfg.Clock))
	})
})

var _ = Describe("Service", func() {
	start := time.Unix(1000, 0)
	var (
		db     *gorp.DB
		clock  *timetestutil.Clock
		svc    *grant.Service
		rk     resource.ID
		secret uuid.UUID
	)
	create := func(ctx SpecContext) grant.Grant {
		GinkgoHelper()
		g := grant.Grant{Subject: rk, Secrets: []uuid.UUID{secret}}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		return g
	}
	stored := func(ctx SpecContext) []grant.Grant {
		GinkgoHelper()
		var grants []grant.Grant
		Expect(gorp.NewRetrieve[grant.Key, grant.Grant]().
			Entries(&grants).
			Exec(ctx, db)).To(Succeed())
		return grants
	}
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		clock = timetestutil.NewClock(start)
		svc = MustOpen(grant.OpenService(ctx, grant.ServiceConfig{
			DB:    db,
			TTL:   time.Minute,
			Clock: clock,
		}))
		rk = rack.Key(1).ResourceID()
		secret = uuid.New()
	})

	Describe("OpenService", func() {
		It("Should return an error when the config is invalid", func(ctx SpecContext) {
			Expect(grant.OpenService(ctx, grant.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
		})
	})

	Describe("Writer", func() {
		Describe("Create", func() {
			It("Should store the grant under a new key with an expiry one TTL away",
				func(ctx SpecContext) {
					g := create(ctx)
					Expect(g.Key).ToNot(Equal(uuid.Nil()))
					Expect(stored(ctx)).To(Equal([]grant.Grant{{
						Key:       g.Key,
						Subject:   rk,
						Secrets:   []uuid.UUID{secret},
						ExpiresAt: telem.NewTimeStamp(start.Add(time.Minute)),
					}}))
				},
			)
			DescribeTable("Should reject a grant that is not valid",
				func(ctx SpecContext, change func(*grant.Grant), msg string) {
					g := grant.Grant{Subject: rk, Secrets: []uuid.UUID{secret}}
					change(&g)
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &g)
					})).To(SatisfyAll(
						BeAssignableToTypeOf(validate.PathError{}),
						MatchError(msg),
					))
					Expect(stored(ctx)).To(BeEmpty())
				},
				Entry(
					"a subject that is not a rack",
					func(g *grant.Grant) {
						g.Subject = resource.ID{
							Type: resource.TypeUser,
							Key:  "u",
						}
					},
					"subject: user:u is not a rack",
				),
				Entry(
					"no secrets",
					func(g *grant.Grant) { g.Secrets = nil },
					"secrets: must be non-empty",
				),
			)
			It("Should delete the grants that have expired", func(ctx SpecContext) {
				create(ctx)
				clock.Advance(2 * time.Minute)
				live := create(ctx)
				Expect(stored(ctx)).To(Equal([]grant.Grant{live}))
			})
		})
	})

	Describe("Redeem", func() {
		It("Should return the grant and delete it", func(ctx SpecContext) {
			g := create(ctx)
			Expect(svc.Redeem(ctx, g.Key, rk)).To(Equal(g))
			Expect(stored(ctx)).To(BeEmpty())
		})
		It("Should refuse a grant that a redeem already used", func(ctx SpecContext) {
			g := create(ctx)
			MustSucceed(svc.Redeem(ctx, g.Key, rk))
			Expect(svc.Redeem(ctx, g.Key, rk)).Error().To(SatisfyAll(
				MatchError(access.ErrDenied),
				MatchError(ContainSubstring(
					"grant "+g.Key.String()+" has ended or does not exist",
				)),
			))
		})
		It("Should refuse and delete a grant that has expired", func(ctx SpecContext) {
			g := create(ctx)
			clock.Advance(time.Minute + time.Nanosecond)
			Expect(svc.Redeem(ctx, g.Key, rk)).Error().To(SatisfyAll(
				MatchError(access.ErrDenied),
				MatchError(ContainSubstring(
					"grant "+g.Key.String()+" has ended or does not exist",
				)),
			))
			Expect(stored(ctx)).To(BeEmpty())
		})
		It(
			"Should refuse a grant for another subject and keep it",
			func(ctx SpecContext) {
				g := create(ctx)
				other := rack.Key(2).ResourceID()
				Expect(svc.Redeem(ctx, g.Key, other)).Error().To(SatisfyAll(
					MatchError(access.ErrDenied),
					MatchError(ContainSubstring("grant is not for "+other.String())),
				))
				Expect(svc.Redeem(ctx, g.Key, rk)).To(Equal(g))
			},
		)
		It("Should let only one of several concurrent redeems use a grant",
			func(ctx SpecContext) {
				g := create(ctx)
				const redeems = 10
				var (
					wg        sync.WaitGroup
					mu        sync.Mutex
					succeeded int
				)
				for range redeems {
					wg.Go(func() {
						if _, err := svc.Redeem(ctx, g.Key, rk); err == nil {
							mu.Lock()
							succeeded++
							mu.Unlock()
						}
					})
				}
				wg.Wait()
				Expect(succeeded).To(Equal(1))
			},
		)
	})
})
