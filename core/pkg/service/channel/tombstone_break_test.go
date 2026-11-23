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

var _ = Describe("Tombstone break", Ordered, ContinueOnFailure, func() {
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
		if ch.DataType == "" {
			ch.DataType = telem.Float32T
		}
		ch.Virtual = true
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	hold := func(ctx SpecContext, k channel.Key) {
		GinkgoHelper()
		Expect(holders.NewCreate().
			Entry(&displayHolder{Key: UniqueChannelName(), Channel: k}).
			Exec(ctx, db)).To(Succeed())
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

	It("Should not rehome a folder channel onto a tombstone in its folder", func(
		ctx SpecContext,
	) {
		g := createFolder(ctx)
		name := UniqueChannelName()
		dead := create(ctx, channel.Channel{Name: name + "_2", Group: &g.Key})
		hold(ctx, dead.Key())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Delete(ctx, dead.Key(), false)
		})).To(Succeed())
		_, ok := tombstoneOf(ctx, dead.Key())
		Expect(ok).To(BeTrue())
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
		_, ok = tombstoneOf(ctx, dead.Key())
		Expect(
			ok,
		).To(BeTrue(), "the rehome deleted the tombstone at %s/%s_2", g.Name, name)
		Expect(renamed.Name).To(Equal(name + "_3"))
	})

	It("Should end the list with a gap later than a last at of TimeStampMax", func(
		ctx SpecContext,
	) {
		past := create(ctx, channel.Channel{})
		ch := create(ctx, channel.Channel{Rewires: []channel.Rewire{
			{From: past.Key(), At: telem.TimeStampMax},
		}})
		hold(ctx, ch.Key())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Delete(ctx, ch.Key(), false)
		})).To(Succeed())
		t, ok := tombstoneOf(ctx, ch.Key())
		Expect(ok).To(BeTrue())
		Expect(t.Rewires).To(HaveLen(2))
		Expect(t.Rewires[1].At).To(BeNumerically(">=", t.Rewires[0].At))
	})

	It("Should write one tombstone each for a mixed delete in one transaction", func(
		ctx SpecContext,
	) {
		f1, f2 := createFolder(ctx), createFolder(ctx)
		a := create(ctx, channel.Channel{Group: &f1.Key})
		b := create(ctx, channel.Channel{Group: &f2.Key})
		c := create(ctx, channel.Channel{})
		hold(ctx, a.Key())
		hold(ctx, a.Key())
		hold(ctx, a.Key())
		hold(ctx, c.Key())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).DeleteMany(
				ctx, []channel.Key{a.Key(), b.Key(), c.Key()}, false,
			)
		})).To(Succeed())
		ta, ok := tombstoneOf(ctx, a.Key())
		Expect(ok).To(BeTrue())
		Expect(ta.Group).To(Equal(&f1.Key))
		Expect(ta.Rewires).To(HaveLen(1))
		_, ok = tombstoneOf(ctx, b.Key())
		Expect(ok).To(BeFalse())
		_, ok = tombstoneOf(ctx, c.Key())
		Expect(ok).To(BeTrue())
		// The same name deleted again in another folder keeps both paths.
		name := a.Name
		d := create(ctx, channel.Channel{Name: name, Group: &f2.Key})
		hold(ctx, d.Key())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Delete(ctx, d.Key(), false)
		})).To(Succeed())
		_, ok = tombstoneOf(ctx, a.Key())
		Expect(ok).To(BeTrue())
		_, ok = tombstoneOf(ctx, d.Key())
		Expect(ok).To(BeTrue())
		// The same path deleted twice keeps one tombstone.
		e := create(ctx, channel.Channel{Name: name, Group: &f1.Key})
		hold(ctx, e.Key())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Delete(ctx, e.Key(), false)
		})).To(Succeed())
		_, ok = tombstoneOf(ctx, a.Key())
		Expect(ok).To(BeFalse())
		_, ok = tombstoneOf(ctx, e.Key())
		Expect(ok).To(BeTrue())
	})

	It("Should delete a folder in the transaction that tombstones its channel", func(
		ctx SpecContext,
	) {
		parent := createFolder(ctx)
		child := group.Group{
			Name:   UniqueChannelName(),
			Scope:  resource.TypeChannel,
			Parent: new(parent.ResourceID()),
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &child)
		})).To(Succeed())
		a := create(ctx, channel.Channel{Group: &child.Key})
		b := create(ctx, channel.Channel{Group: &parent.Key})
		hold(ctx, a.Key())
		hold(ctx, b.Key())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := svc.NewWriter(tx)
			if err := w.DeleteMany(
				ctx,
				[]channel.Key{a.Key(), b.Key()},
				false,
			); err != nil {
				return err
			}
			if _, err := w.DeleteTombstones(ctx, parent.Key, child.Key); err != nil {
				return err
			}
			return groupSvc.NewWriter(tx).Delete(ctx, parent.Key, child.Key)
		})).To(Succeed())
		Expect(allTombstones(ctx, db)).ToNot(ContainElement(
			HaveField("Key", BeElementOf(a.Key(), b.Key())),
		))
	})

	It("Should leave a tombstone out of the uses of a channel in its rewires", func(
		ctx SpecContext,
	) {
		past := create(ctx, channel.Channel{})
		ch := create(ctx, channel.Channel{
			Rewires: []channel.Rewire{{From: past.Key(), At: telem.Now()}},
		})
		hold(ctx, ch.Key())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Delete(ctx, ch.Key(), false)
		})).To(Succeed())
		_, ok := tombstoneOf(ctx, ch.Key())
		Expect(ok).To(BeTrue())
		uses := MustSucceed(svc.Uses(ctx, db, []channel.Key{past.Key()}))
		Expect(uses).To(HaveLen(1))
		Expect(uses[0]).ToNot(ContainElement(
			HaveField("Holder.Table", "channel_tombstone"),
		))
	})

	It("Should write one tombstone for a key named twice in one delete", func(
		ctx SpecContext,
	) {
		a := create(ctx, channel.Channel{})
		hold(ctx, a.Key())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).
				DeleteMany(ctx, []channel.Key{a.Key(), a.Key()}, false)
		})).To(Succeed())
		_, ok := tombstoneOf(ctx, a.Key())
		Expect(ok).To(BeTrue())
	})

	It("Should delete and remake a held path in one transaction", func(
		ctx SpecContext,
	) {
		g := createFolder(ctx)
		a := create(ctx, channel.Channel{Group: &g.Key})
		hold(ctx, a.Key())
		var b channel.Channel
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := svc.NewWriter(tx)
			if err := w.Delete(ctx, a.Key(), false); err != nil {
				return err
			}
			b = channel.Channel{
				Name: a.Name, Group: &g.Key, DataType: telem.Float32T, Virtual: true,
			}
			return w.Create(ctx, &b)
		})).To(Succeed())
		_, ok := tombstoneOf(ctx, a.Key())
		Expect(ok).To(BeFalse())
		hold(ctx, b.Key())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := svc.NewWriter(tx)
			if err := w.Rename(ctx, b.Key(), UniqueChannelName(), false); err != nil {
				return err
			}
			if err := w.Rename(ctx, b.Key(), a.Name, false); err != nil {
				return err
			}
			return w.Delete(ctx, b.Key(), false)
		})).To(Succeed())
		tb, ok := tombstoneOf(ctx, b.Key())
		Expect(ok).To(BeTrue())
		Expect(tb.Name).To(Equal(a.Name))
	})
})

// allTombstones returns every tombstone that db holds.
func allTombstones(ctx SpecContext, db *gorp.DB) []channel.Tombstone {
	GinkgoHelper()
	var tombstones []channel.Tombstone
	Expect(gorp.NewRetrieve[channel.Key, channel.Tombstone]().
		Entries(&tombstones).
		Exec(ctx, db)).To(Succeed())
	return tombstones
}
