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
	"github.com/synnaxlabs/x/validate"
)

// displayHolder holds a channel through a weak link, as a schematic symbol does.
type displayHolder struct {
	Key     string
	Channel channel.Key
}

func (h displayHolder) GorpKey() string { return h.Key }

var _ = Describe("Tombstone", Ordered, func() {
	var (
		db       *gorp.DB
		svc      *channel.Service
		groupSvc *group.Service
		holders  *gorp.Table[string, displayHolder]
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		// The holder has no generated codec, so the store encodes with msgpack.
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

	createFolder := func(ctx SpecContext) group.Group {
		GinkgoHelper()
		g := group.Group{Name: UniqueChannelName(), Scope: resource.TypeChannel}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		return g
	}
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
	del := func(ctx SpecContext, k channel.Key) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Delete(ctx, k, false)
		})).To(Succeed())
	}
	tombstoneOf := func(ctx SpecContext, k channel.Key) (channel.Tombstone, bool) {
		GinkgoHelper()
		var tombstones []channel.Tombstone
		Expect(gorp.NewRetrieve[channel.Key, channel.Tombstone]().
			Entries(&tombstones).
			Exec(ctx, db)).To(Succeed())
		for _, t := range tombstones {
			if t.Key == k {
				return t, true
			}
		}
		return channel.Tombstone{}, false
	}
	// deleteHeld creates a channel at the path of ch that a display holds, deletes it,
	// and returns it.
	deleteHeld := func(ctx SpecContext, ch channel.Channel) channel.Channel {
		GinkgoHelper()
		ch = create(ctx, ch)
		hold(ctx, ch.Key())
		del(ctx, ch.Key())
		return ch
	}

	It("Should keep the path of a deleted channel that a display holds", func(
		ctx SpecContext,
	) {
		g := createFolder(ctx)
		before := telem.Now()
		ch := deleteHeld(ctx, channel.Channel{Group: &g.Key})
		t := MustBeOk(tombstoneOf(ctx, ch.Key()))
		Expect(t.Name).To(Equal(ch.Name))
		Expect(t.Group).To(Equal(&g.Key))
		Expect(t.Rewires).To(HaveLen(1))
		Expect(t.Rewires[0].From).To(BeZero())
		Expect(t.Rewires[0].At).To(BeNumerically(">=", before))
	})

	It("Should keep no path of a deleted channel that nothing holds", func(
		ctx SpecContext,
	) {
		ch := create(ctx, channel.Channel{})
		del(ctx, ch.Key())
		_, ok := tombstoneOf(ctx, ch.Key())
		Expect(ok).To(BeFalse())
	})

	It("Should keep no path of a channel that only a rewires list holds", func(
		ctx SpecContext,
	) {
		past := create(ctx, channel.Channel{})
		create(ctx, channel.Channel{
			Rewires: []channel.Rewire{{From: past.Key(), At: telem.Now()}},
		})
		del(ctx, past.Key())
		_, ok := tombstoneOf(ctx, past.Key())
		Expect(ok).To(BeFalse())
	})

	It("Should end the list in one gap after a gap, later than the last one", func(
		ctx SpecContext,
	) {
		past := create(ctx, channel.Channel{})
		late := telem.Now() + telem.TimeStamp(telem.Hour)
		ch := deleteHeld(ctx, channel.Channel{Rewires: []channel.Rewire{
			{From: past.Key(), At: telem.Now()},
			{At: late},
		}})
		t := MustBeOk(tombstoneOf(ctx, ch.Key()))
		Expect(t.Rewires).To(HaveLen(2))
		Expect(t.Rewires[0].From).To(Equal(past.Key()))
		Expect(t.Rewires[1]).To(Equal(channel.Rewire{At: late + 1}))
	})

	Describe("A channel that takes the path of a tombstone", func() {
		It("Should remove the tombstone on a create", func(ctx SpecContext) {
			g := createFolder(ctx)
			dead := deleteHeld(ctx, channel.Channel{Group: &g.Key})
			create(ctx, channel.Channel{Name: dead.Name, Group: &g.Key})
			_, ok := tombstoneOf(ctx, dead.Key())
			Expect(ok).To(BeFalse())
		})

		It("Should remove the tombstone on a rename", func(ctx SpecContext) {
			dead := deleteHeld(ctx, channel.Channel{})
			ch := create(ctx, channel.Channel{})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Rename(ctx, ch.Key(), dead.Name, false)
			})).To(Succeed())
			_, ok := tombstoneOf(ctx, dead.Key())
			Expect(ok).To(BeFalse())
		})

		It("Should remove the tombstone on a move", func(ctx SpecContext) {
			g := createFolder(ctx)
			dead := deleteHeld(ctx, channel.Channel{Group: &g.Key})
			ch := create(ctx, channel.Channel{Name: dead.Name})
			_, ok := tombstoneOf(ctx, dead.Key())
			Expect(ok).To(BeTrue())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).SetGroup(ctx, &g.Key, ch.Key())
			})).To(Succeed())
			_, ok = tombstoneOf(ctx, dead.Key())
			Expect(ok).To(BeFalse())
		})
	})

	It(
		"Should refuse the delete of a folder that holds a tombstone until it goes",
		func(
			ctx SpecContext,
		) {
			g := createFolder(ctx)
			dead := deleteHeld(ctx, channel.Channel{Group: &g.Key})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return groupSvc.NewWriter(tx).Delete(ctx, g.Key)
			})).To(MatchError(validate.ErrValidation))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				gone, err := svc.NewWriter(tx).DeleteTombstones(ctx, g.Key)
				if err != nil {
					return err
				}
				Expect(gone).To(HaveLen(1))
				Expect(gone[0].Key).To(Equal(dead.Key()))
				Expect(gone[0].Name).To(Equal(dead.Name))
				Expect(gone[0].Group).To(Equal(&g.Key))
				return groupSvc.NewWriter(tx).Delete(ctx, g.Key)
			})).To(Succeed())
			_, ok := tombstoneOf(ctx, dead.Key())
			Expect(ok).To(BeFalse())
		},
	)

	It("Should return the tombstones in folders and leave them in place", func(
		ctx SpecContext,
	) {
		g := createFolder(ctx)
		other := createFolder(ctx)
		dead := deleteHeld(ctx, channel.Channel{Group: &g.Key})
		deleteHeld(ctx, channel.Channel{Group: &other.Key})
		found := MustSucceed(svc.Tombstones(ctx, db, g.Key))
		Expect(found).To(HaveLen(1))
		Expect(found[0].Key).To(Equal(dead.Key()))
		Expect(found[0].Name).To(Equal(dead.Name))
		Expect(found[0].Group).To(Equal(&g.Key))
		_, ok := tombstoneOf(ctx, dead.Key())
		Expect(ok).To(BeTrue())
		Expect(svc.Tombstones(ctx, db, createFolder(ctx).Key)).To(BeEmpty())
	})

	It("Should name a tombstone among the records that sit in a folder", func(
		ctx SpecContext,
	) {
		g := createFolder(ctx)
		dead := deleteHeld(ctx, channel.Channel{Group: &g.Key})
		live := create(ctx, channel.Channel{Group: &g.Key})
		children := MustSucceed(groupSvc.Children(ctx, db, g.Key))
		Expect(children).To(HaveLen(2))
		for _, id := range children {
			Expect(channel.IsTombstone(id)).To(Equal(id.Key == dead.Key().String()))
		}
		Expect(channel.IsTombstone(channel.ResourceID(live.Key()))).To(BeFalse())
	})

	Describe("MoveTombstones", func() {
		keysOf := func(ts []channel.Tombstone) channel.Keys {
			keys := make(channel.Keys, len(ts))
			for i, t := range ts {
				keys[i] = t.Key
			}
			return keys
		}
		move := func(
			ctx SpecContext,
			moves map[group.Key]*group.Key,
		) []channel.Tombstone {
			GinkgoHelper()
			var dropped []channel.Tombstone
			Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
				dropped, err = svc.NewWriter(tx).MoveTombstones(ctx, moves)
				return err
			})).To(Succeed())
			return dropped
		}

		It("Should move tombstones and drop the ones whose new paths are taken", func(
			ctx SpecContext,
		) {
			from, to := createFolder(ctx), createFolder(ctx)
			moved := deleteHeld(ctx, channel.Channel{Group: &from.Key})
			byChannel := deleteHeld(ctx, channel.Channel{Group: &from.Key})
			create(ctx, channel.Channel{Name: byChannel.Name, Group: &to.Key})
			name := UniqueChannelName()
			kept := deleteHeld(ctx, channel.Channel{Name: name, Group: &to.Key})
			byTombstone := deleteHeld(
				ctx,
				channel.Channel{Name: name, Group: &from.Key},
			)
			moves := map[group.Key]*group.Key{from.Key: &to.Key}
			Expect(keysOf(MustSucceed(svc.DroppedTombstones(ctx, db, moves)))).
				To(ConsistOf(byChannel.Key(), byTombstone.Key()))
			Expect(keysOf(move(ctx, moves))).
				To(ConsistOf(byChannel.Key(), byTombstone.Key()))
			Expect(MustBeOk(tombstoneOf(ctx, moved.Key())).Group).To(Equal(&to.Key))
			Expect(MustBeOk(tombstoneOf(ctx, kept.Key())).Group).To(Equal(&to.Key))
			_, ok := tombstoneOf(ctx, byChannel.Key())
			Expect(ok).To(BeFalse())
			_, ok = tombstoneOf(ctx, byTombstone.Key())
			Expect(ok).To(BeFalse())
		})

		It("Should move a tombstone out of any folder for nil", func(ctx SpecContext) {
			from := createFolder(ctx)
			dead := deleteHeld(ctx, channel.Channel{Group: &from.Key})
			Expect(move(ctx, map[group.Key]*group.Key{from.Key: nil})).To(BeEmpty())
			Expect(MustBeOk(tombstoneOf(ctx, dead.Key())).Group).To(BeNil())
		})

		It("Should count a channel in a moved folder at the folder it moves to", func(
			ctx SpecContext,
		) {
			a, b, to := createFolder(ctx), createFolder(ctx), createFolder(ctx)
			dead := deleteHeld(ctx, channel.Channel{Group: &b.Key})
			create(ctx, channel.Channel{Name: dead.Name, Group: &a.Key})
			moves := map[group.Key]*group.Key{a.Key: &to.Key, b.Key: &to.Key}
			Expect(keysOf(MustSucceed(svc.DroppedTombstones(ctx, db, moves)))).
				To(ConsistOf(dead.Key()))
		})

		It("Should drop the tombstone of the later folder when two take one path", func(
			ctx SpecContext,
		) {
			a, b, to := createFolder(ctx), createFolder(ctx), createFolder(ctx)
			name := UniqueChannelName()
			inA := deleteHeld(ctx, channel.Channel{Name: name, Group: &a.Key})
			inB := deleteHeld(ctx, channel.Channel{Name: name, Group: &b.Key})
			later := inB
			if string(b.Key[:]) < string(a.Key[:]) {
				later = inA
			}
			moves := map[group.Key]*group.Key{a.Key: &to.Key, b.Key: &to.Key}
			Expect(keysOf(move(ctx, moves))).To(ConsistOf(later.Key()))
		})
	})

	It("Should skip a tombstone path in the folder of a channel it rehomes", func(
		ctx SpecContext,
	) {
		g := createFolder(ctx)
		name := UniqueChannelName()
		dead := deleteHeld(ctx, channel.Channel{Name: name + "_2", Group: &g.Key})
		user := create(ctx, channel.Channel{Name: name, Group: &g.Key})
		builtIn := channel.Channel{Name: name, DataType: telem.JSONT, Virtual: true}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Declare(ctx, &builtIn)
		})).To(Succeed())
		var renamed channel.Channel
		Expect(svc.NewRetrieve().
			Where(channel.MatchKeys(user.Key())).
			Entry(&renamed).
			Exec(ctx, db)).To(Succeed())
		Expect(renamed.Name).To(Equal(name + "_3"))
		_, ok := tombstoneOf(ctx, dead.Key())
		Expect(ok).To(BeTrue())
	})

	It("Should skip a tombstone path when it picks a suffix for a built-in", func(
		ctx SpecContext,
	) {
		name := UniqueChannelName()
		deleteHeld(ctx, channel.Channel{Name: name + "_2"})
		user := create(ctx, channel.Channel{Name: name})
		builtIn := channel.Channel{Name: name, DataType: telem.JSONT, Virtual: true}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Declare(ctx, &builtIn)
		})).To(Succeed())
		var renamed channel.Channel
		Expect(svc.NewRetrieve().
			Where(channel.MatchKeys(user.Key())).
			Entry(&renamed).
			Exec(ctx, db)).To(Succeed())
		Expect(renamed.Name).To(Equal(name + "_3"))
	})

	It(
		"Should skip a folder tombstone path when it picks a suffix for a built-in",
		func(
			ctx SpecContext,
		) {
			g := createFolder(ctx)
			name := UniqueChannelName()
			dead := deleteHeld(ctx, channel.Channel{Name: name + "_2", Group: &g.Key})
			user := create(ctx, channel.Channel{Name: name, Group: &g.Key})
			builtIn := channel.Channel{Name: name, DataType: telem.JSONT, Virtual: true}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Declare(ctx, &builtIn)
			})).To(Succeed())
			var renamed channel.Channel
			Expect(svc.NewRetrieve().
				Where(channel.MatchKeys(user.Key())).
				Entry(&renamed).
				Exec(ctx, db)).To(Succeed())
			Expect(renamed.Name).To(Equal(name + "_3"))
			_, ok := tombstoneOf(ctx, dead.Key())
			Expect(ok).To(BeTrue())
		},
	)
})
