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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// pairHolder holds two channels through one weak link that can move.
type pairHolder struct {
	Key  string
	A, B channel.Key
}

func (h pairHolder) GorpKey() string { return h.Key }

var _ = Describe("Checker heal r1", Ordered, ContinueOnFailure, func() {
	var (
		db       *gorp.DB
		svc      *channel.Service
		groupSvc *group.Service
		pairs    *gorp.Table[string, pairHolder]
		rewrites int
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		var cfg channel.ServiceConfig
		cfg, groupSvc = serviceConfig(ctx, testutil.NewNode(ctx), db)
		cfg.Retargets = func(gorp.Dependent) (bool, error) { return true, nil }
		svc = MustOpen(channel.OpenService(ctx, cfg))
		pairs = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, pairHolder]{
			DB:   db,
			Name: "pair",
			References: []gorp.Reference[string, pairHolder]{gorp.NewKeyReference(
				gorp.KeyReferenceConfig[string, pairHolder, channel.Key]{
					Name:     "channels",
					Table:    "channel",
					OnDelete: gorp.Weak,
					Kind:     gorp.Use,
					Targets: func(h *pairHolder) []channel.Key {
						return []channel.Key{h.A, h.B}
					},
					Detach: func(h *pairHolder, k channel.Key) {
						if h.A == k {
							h.A = 0
						}
						if h.B == k {
							h.B = 0
						}
					},
					Retarget: func(h *pairHolder, to map[channel.Key]channel.Key) {
						if k, ok := to[h.A]; ok {
							h.A = k
						}
						if k, ok := to[h.B]; ok {
							h.B = k
						}
					},
				},
			)},
			OnRewrite: func(context.Context, gorp.Tx, pairHolder) error {
				rewrites++
				return nil
			},
		}))
		Expect(svc.Heal(ctx)).To(Succeed())
	})

	create := func(ctx SpecContext, chs ...channel.Channel) []channel.Channel {
		GinkgoHelper()
		for i := range chs {
			if chs[i].Name == "" {
				chs[i].Name = UniqueChannelName()
			}
			chs[i].DataType, chs[i].Virtual = telem.Float32T, true
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).CreateMany(ctx, &chs)
		})).To(Succeed())
		return chs
	}
	retrieve := func(ctx SpecContext, k channel.Key) channel.Channel {
		GinkgoHelper()
		var ch channel.Channel
		Expect(svc.NewRetrieve().
			Where(channel.MatchKeys(k)).
			Entry(&ch).
			Exec(ctx, db)).To(Succeed())
		return ch
	}
	remove := func(ctx SpecContext, keys ...channel.Key) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).DeleteMany(ctx, keys, false)
		})).To(Succeed())
	}

	It("Should move a path history link when Retargets accepts it", func(
		ctx SpecContext,
	) {
		name := UniqueChannelName()
		dead := create(ctx, channel.Channel{Name: name})[0].Key()
		Expect(pairs.NewCreate().
			Entry(&pairHolder{Key: UniqueChannelName(), A: dead, B: dead}).
			Exec(ctx, db)).To(Succeed())
		remove(ctx, dead)
		past := create(ctx, channel.Channel{})[0]
		past.Rewires = []channel.Rewire{{From: dead, At: telem.Now()}}
		Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).Set(ctx, past)).
			To(Succeed())
		ch := create(ctx, channel.Channel{Name: name})[0]
		Expect(retrieve(ctx, past.Key()).Rewires[0].From).To(Equal(ch.Key()),
			"the policy is the one source of what a heal moves, and it accepts "+
				"every link")
	})

	It("Should write a holder of two dead keys once when one move heals both", func(
		ctx SpecContext,
	) {
		g := group.Group{Name: UniqueChannelName(), Scope: resource.TypeChannel}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		a, b := UniqueChannelName(), UniqueChannelName()
		dead := create(
			ctx,
			channel.Channel{Name: a, Group: &g.Key},
			channel.Channel{Name: b, Group: &g.Key},
		)
		holder := pairHolder{
			Key: UniqueChannelName(),
			A:   dead[0].Key(),
			B:   dead[1].Key(),
		}
		Expect(pairs.NewCreate().Entry(&holder).Exec(ctx, db)).To(Succeed())
		remove(ctx, dead[0].Key(), dead[1].Key())
		moved := create(ctx, channel.Channel{Name: a}, channel.Channel{Name: b})
		rewrites = 0
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).
				SetGroup(ctx, &g.Key, moved[0].Key(), moved[1].Key())
		})).To(Succeed())
		var got pairHolder
		Expect(pairs.NewRetrieve().
			Where(gorp.MatchKeys[string, pairHolder](holder.Key)).
			Entry(&got).
			Exec(ctx, db)).To(Succeed())
		Expect(got).To(Equal(pairHolder{
			Key: holder.Key,
			A:   moved[0].Key(),
			B:   moved[1].Key(),
		}))
		Expect(rewrites).To(Equal(1),
			"one move heals both dead keys, so one retarget writes the holder once")
	})
})
