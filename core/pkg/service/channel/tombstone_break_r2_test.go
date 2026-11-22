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

var _ = Describe("Tombstone break r2", Ordered, ContinueOnFailure, func() {
	var (
		db       *gorp.DB
		svc      *channel.Service
		groupSvc *group.Service
		holders  *gorp.Table[string, displayHolder]
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		var cfg channel.ServiceConfig
		cfg, groupSvc = serviceConfig(ctx, testutil.NewNode(ctx), db)
		svc = MustOpen(channel.OpenService(ctx, cfg))
		holders = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, displayHolder]{
			DB:   db,
			Name: "display_holder",
			References: []gorp.Reference[string, displayHolder]{
				gorp.NewKeyReference(
					gorp.KeyReferenceConfig[string, displayHolder, channel.Key]{
						Name:     "channel",
						Table:    "channel",
						OnDelete: gorp.Weak,
						Kind:     gorp.Use,
						Tag:      1,
						Targets: func(h *displayHolder) []channel.Key {
							return []channel.Key{h.Channel}
						},
						Detach: func(h *displayHolder, _ channel.Key) { h.Channel = 0 },
					},
				),
			},
		}))
		Expect(svc.Heal(ctx)).To(Succeed())
	})
	create := func(ctx SpecContext, ch channel.Channel) channel.Channel {
		GinkgoHelper()
		if ch.Name == "" {
			ch.Name = UniqueChannelName()
		}
		ch.DataType, ch.Virtual = telem.Float32T, true
		rewires := ch.Rewires
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		if len(rewires) > 0 {
			// A create drops the list it gets, so the table writes it.
			ch.Rewires = rewires
			Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).Set(ctx, ch)).
				To(Succeed())
		}
		return ch
	}
	hold := func(ctx SpecContext, k channel.Key) {
		GinkgoHelper()
		Expect(holders.NewCreate().
			Entry(&displayHolder{Key: UniqueChannelName(), Channel: k}).
			Exec(ctx, db)).To(Succeed())
	}
	createFolder := func(ctx SpecContext) group.Group {
		GinkgoHelper()
		g := group.Group{Name: UniqueChannelName(), Scope: resource.TypeChannel}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		return g
	}

	tombstoneOf := func(ctx SpecContext, k channel.Key) (channel.Tombstone, bool) {
		GinkgoHelper()
		for _, t := range allTombstones(ctx, db) {
			if t.Key == k {
				return t, true
			}
		}
		return channel.Tombstone{}, false
	}
	deleteHeld := func(ctx SpecContext, ch channel.Channel) channel.Channel {
		GinkgoHelper()
		ch = create(ctx, ch)
		hold(ctx, ch.Key())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Delete(ctx, ch.Key(), false)
		})).To(Succeed())
		return ch
	}
	retrieveName := func(ctx SpecContext, k channel.Key) string {
		GinkgoHelper()
		var ch channel.Channel
		Expect(svc.NewRetrieve().
			Where(channel.MatchKeys(k)).
			Entry(&ch).
			Exec(ctx, db)).To(Succeed())
		return ch.Name
	}

	It("Should rehome a nested folder channel past only its own folder tombstones",
		func(ctx SpecContext) {
			parent := createFolder(ctx)
			child := group.Group{
				Name:   UniqueChannelName(),
				Scope:  resource.TypeChannel,
				Parent: new(parent.ResourceID()),
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return groupSvc.NewWriter(tx).Create(ctx, &child)
			})).To(Succeed())
			name := UniqueChannelName()
			inParent := deleteHeld(
				ctx,
				channel.Channel{Name: name + "_2", Group: &parent.Key},
			)
			inChild := deleteHeld(
				ctx,
				channel.Channel{Name: name + "_3", Group: &child.Key},
			)
			atRoot := deleteHeld(ctx, channel.Channel{Name: name + "_4"})
			user := create(ctx, channel.Channel{Name: name, Group: &child.Key})
			builtIn := channel.Channel{Name: name, DataType: telem.JSONT, Virtual: true}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Declare(ctx, &builtIn)
			})).To(Succeed())
			Expect(retrieveName(ctx, user.Key())).To(Equal(name + "_2"))
			for _, k := range []channel.Key{inParent.Key(), inChild.Key(), atRoot.Key()} {
				_, ok := tombstoneOf(ctx, k)
				Expect(ok).To(BeTrue())
			}
		})

	It("Should rehome a root channel past a root tombstone and not a folder one",
		func(ctx SpecContext) {
			g := createFolder(ctx)
			name := UniqueChannelName()
			deleteHeld(ctx, channel.Channel{Name: name + "_2"})
			deleteHeld(ctx, channel.Channel{Name: name + "_3", Group: &g.Key})
			user := create(ctx, channel.Channel{Name: name})
			builtIn := channel.Channel{
				Name:     name,
				DataType: telem.StringT,
				Virtual:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Declare(ctx, &builtIn)
			})).To(Succeed())
			Expect(retrieveName(ctx, user.Key())).To(Equal(name + "_3"))
		})

	DescribeTable(
		"Should end the list with a gap no earlier than the last at",
		func(ctx SpecContext, last channel.Rewire, wantAt telem.TimeStamp, n int) {
			past := create(ctx, channel.Channel{})
			if last.From == 1 {
				last.From = past.Key()
			}
			ch := deleteHeld(ctx, channel.Channel{
				Rewires: []channel.Rewire{{From: past.Key(), At: 1}, last},
			})
			t, ok := tombstoneOf(ctx, ch.Key())
			Expect(ok).To(BeTrue())
			Expect(t.Rewires).To(HaveLen(n))
			Expect(t.Rewires[n-1]).To(Equal(channel.Rewire{At: wantAt}))
		},
		Entry(
			"a holder one before the max",
			channel.Rewire{From: 1, At: telem.TimeStampMax - 1},
			telem.TimeStampMax,
			3,
		),
		Entry(
			"a gap one before the max",
			channel.Rewire{At: telem.TimeStampMax - 1},
			telem.TimeStampMax,
			2,
		),
		Entry(
			"a gap at the max",
			channel.Rewire{At: telem.TimeStampMax},
			telem.TimeStampMax,
			2,
		),
	)

	It("Should delete many channels that nothing holds and keep no paths", func(
		ctx SpecContext,
	) {
		g := createFolder(ctx)
		keys := make([]channel.Key, 0, 5)
		for range 5 {
			keys = append(keys, create(ctx, channel.Channel{Group: &g.Key}).Key())
		}
		before := len(allTombstones(ctx, db))
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).DeleteMany(ctx, keys, false)
		})).To(Succeed())
		Expect(allTombstones(ctx, db)).To(HaveLen(before))
		Expect(svc.NewRetrieve().
			Where(channel.MatchKeys(keys...)).
			Exists(ctx, db)).To(BeFalse())
	})

	It("Should leave a tombstone out of uses and keep a display of the same channel",
		func(ctx SpecContext) {
			past := create(ctx, channel.Channel{})
			ch := create(ctx, channel.Channel{
				Rewires: []channel.Rewire{{From: past.Key(), At: telem.Now()}},
			})
			hold(ctx, ch.Key())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, ch.Key(), false)
			})).To(Succeed())
			hold(ctx, past.Key())
			uses := MustSucceed(svc.Uses(ctx, db, []channel.Key{past.Key(), ch.Key()}))
			Expect(uses[0]).To(HaveLen(1))
			Expect(uses[0][0].Holder.Table).To(Equal("display_holder"))
			Expect(uses[1]).To(HaveLen(1))
			Expect(uses[1][0].Holder.Table).To(Equal("display_holder"))
		})
})
