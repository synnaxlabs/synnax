// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"math/rand/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

type breakerChannel struct {
	ch         channel.Channel
	calculated bool
}

func createBreakerChannels(ctx SpecContext) []breakerChannel {
	GinkgoHelper()
	index := channel.Channel{
		Name:     UniqueChannelName(),
		DataType: telem.TimestampT,
		IsIndex:  true,
	}
	var out []breakerChannel
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		w := channelSvc.NewWriter(tx)
		if err := w.Create(ctx, &index); err != nil {
			return err
		}
		data := channel.Channel{
			Name:       UniqueChannelName(),
			DataType:   telem.Float32T,
			LocalIndex: index.LocalKey,
		}
		if err := w.Create(ctx, &data); err != nil {
			return err
		}
		out = append(out, breakerChannel{ch: index}, breakerChannel{ch: data})
		for i := range 4 {
			v := channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}
			if err := w.Create(ctx, &v); err != nil {
				return err
			}
			calculated := i%2 == 0
			if calculated {
				c := calculation.Calculation{Expression: "return 2", Output: v.Key()}
				if err := layer.Calculation.NewWriter(tx).Create(ctx, &c); err != nil {
					return err
				}
			}
			out = append(out, breakerChannel{ch: v, calculated: calculated})
		}
		return nil
	})).To(Succeed())
	return out
}

// randomWhere returns a random tree of calculated, virtual, key, and is_index lines,
// with and, or, and not at any depth.
func randomWhere(r *rand.Rand, chs []breakerChannel, depth int) channel.Where {
	var w channel.Where
	if r.IntN(3) == 0 {
		w.Calculated = new(r.IntN(2) == 0)
	}
	if r.IntN(4) == 0 {
		w.Virtual = new(r.IntN(2) == 0)
	}
	if r.IntN(5) == 0 {
		w.IsIndex = &filter.Bool{Eq: new(r.IntN(2) == 0)}
	}
	if r.IntN(5) == 0 {
		keys := lo.FilterMap(chs, func(c breakerChannel, _ int) (channel.Key, bool) {
			return c.ch.Key(), r.IntN(2) == 0
		})
		w.Key = &filter.Set[channel.Key]{In: keys}
	}
	if depth > 0 {
		if r.IntN(3) == 0 {
			for range r.IntN(3) + 1 {
				w.And = append(w.And, randomWhere(r, chs, depth-1))
			}
		}
		if r.IntN(3) == 0 {
			for range r.IntN(3) + 1 {
				w.Or = append(w.Or, randomWhere(r, chs, depth-1))
			}
		}
		if r.IntN(3) == 0 {
			not := randomWhere(r, chs, depth-1)
			w.Not = &not
		}
	}
	return w
}

// matchesWhere checks w against c with the channel API's meaning of each line.
func matchesWhere(c breakerChannel, w channel.Where) bool {
	for _, a := range w.And {
		if !matchesWhere(c, a) {
			return false
		}
	}
	if w.Or != nil && !lo.ContainsBy(w.Or, func(o channel.Where) bool {
		return matchesWhere(c, o)
	}) {
		return false
	}
	if w.Not != nil && matchesWhere(c, *w.Not) {
		return false
	}
	if w.Calculated != nil && c.calculated != *w.Calculated {
		return false
	}
	if w.Virtual != nil && (c.ch.Virtual && !c.calculated) != *w.Virtual {
		return false
	}
	if w.IsIndex != nil && c.ch.IsIndex != *w.IsIndex.Eq {
		return false
	}
	if w.Key != nil && !lo.Contains(w.Key.In, c.ch.Key()) {
		return false
	}
	return true
}

var _ = Describe("Calculated line breaker", Ordered, func() {
	var (
		admin, reader user.User
		chs           []breakerChannel
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		admin = MustSucceed(rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc,
			user.User{Name: "breaker-admin-" + uuid.New().String()}))
		grantOn(ctx, admin.ResourceID(), access.AllActions,
			resource.ID{Type: resource.TypeChannel},
			resource.ID{Type: resource.TypeCalculation},
			resource.ID{Type: resource.TypeFramer},
		)
		reader = MustSucceed(rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc,
			user.User{Name: "breaker-reader-" + uuid.New().String()}))
		grantOn(ctx, reader.ResourceID(), []access.Action{access.ActionRetrieve},
			resource.ID{Type: resource.TypeChannel},
			resource.ID{Type: resource.TypeFramer},
		)
		chs = createBreakerChannels(ctx)
	})

	for _, subject := range []string{"admin", "reader"} {
		It("Should match random trees over 64 seeds for the "+subject, func(
			ctx SpecContext,
		) {
			u := lo.Ternary(subject == "admin", admin, reader)
			keys := lo.Map(chs, func(c breakerChannel, _ int) channel.Key {
				return c.ch.Key()
			})
			for seed := range uint64(64) {
				r := rand.New(rand.NewPCG(seed, 7))
				for range 8 {
					w := randomWhere(r, chs, 3)
					where := channel.Where{And: []channel.Where{
						w,
						{Key: &filter.Set[channel.Key]{In: keys}},
					}}
					res := MustSucceed(retrieveChannels(
						AuthedCtx(ctx, u),
						apichannel.RetrieveRequest{Where: &where},
					))
					expected := lo.FilterMap(
						chs,
						func(c breakerChannel, _ int) (channel.Key, bool) {
							return c.ch.Key(), matchesWhere(c, w)
						},
					)
					Expect(lo.Map(
						res.Channels,
						func(ch apichannel.Channel, _ int) channel.Key { return ch.Key },
					)).To(ConsistOf(expected), "seed %d where %+v", seed, w)
				}
			}
		})
	}

	It("Should not tell a subject which unreadable channels are calculated", func(
		ctx SpecContext,
	) {
		outsider := MustSucceed(rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc,
			user.User{Name: "breaker-outsider-" + uuid.New().String()}))
		readable := chs[1].ch
		grantOn(
			ctx,
			outsider.ResourceID(),
			[]access.Action{access.ActionRetrieve},
			append(
				framer.ResourceIDs(channel.Keys{readable.Key()}),
				channel.ResourceID(readable.Key()),
			)...,
		)
		output, plain := chs[2].ch, chs[3].ch
		Expect(chs[2].calculated).To(BeTrue())
		Expect(chs[3].calculated).To(BeFalse())
		ask := func(name string) error {
			where := channel.Where{
				Name:       &filter.String{Eq: new(name)},
				Calculated: new(true),
			}
			_, err := retrieveChannels(
				AuthedCtx(ctx, outsider),
				apichannel.RetrieveRequest{Where: &where},
			)
			return err
		}
		// Neither channel is readable, so the answer must not depend on which one a
		// calculation writes.
		plainErr, outputErr := ask(plain.Name), ask(output.Name)
		Expect(errors.Is(outputErr, access.ErrDenied)).
			To(Equal(errors.Is(plainErr, access.ErrDenied)),
				"plain: %v, output: %v", plainErr, outputErr)
	})
})
