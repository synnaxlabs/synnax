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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var errP2Rollback = errors.New("roll back")

var _ = Describe("Heal break p2", Ordered, ContinueOnFailure, func() {
	var (
		db       *gorp.DB
		svc      *channel.Service
		groupSvc *group.Service
		displays *gorp.Table[string, breakDisplay]
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		var cfg channel.ServiceConfig
		cfg, groupSvc = serviceConfig(ctx, testutil.NewNode(ctx), db)
		cfg.Retargets = func(d gorp.Dependent) (bool, error) {
			return d.Holder.Table == "break_display", nil
		}
		svc = MustOpen(channel.OpenService(ctx, cfg))
		displays = MustSucceed(gorp.OpenTable(ctx, breakDisplayConfig(db)))
		DeferCleanup(func() { Expect(displays.Close()).To(Succeed()) })
		Expect(svc.Heal(ctx)).To(Succeed())
	})

	template := func(name string, g *group.Key) channel.Channel {
		if name == "" {
			name = UniqueChannelName()
		}
		return channel.Channel{
			Name:     name,
			Group:    g,
			DataType: telem.Float32T,
			Virtual:  true,
		}
	}
	createAll := func(ctx SpecContext, chs ...channel.Channel) []channel.Channel {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).CreateMany(ctx, &chs)
		})).To(Succeed())
		return chs
	}
	folder := func(ctx SpecContext) group.Group {
		GinkgoHelper()
		g := group.Group{Name: UniqueChannelName(), Scope: resource.TypeChannel}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		return g
	}
	show := func(ctx SpecContext, keys ...channel.Key) string {
		GinkgoHelper()
		d := breakDisplay{Key: UniqueChannelName(), Channels: keys}
		Expect(displays.NewCreate().Entry(&d).Exec(ctx, db)).To(Succeed())
		return d.Key
	}
	shown := func(ctx SpecContext, key string) []channel.Key {
		GinkgoHelper()
		var d breakDisplay
		Expect(gorp.NewRetrieve[string, breakDisplay]().
			Where(gorp.MatchKeys[string, breakDisplay](key)).
			Entry(&d).
			Exec(ctx, db)).To(Succeed())
		return d.Channels
	}
	del := func(ctx SpecContext, keys ...channel.Key) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).DeleteMany(ctx, keys, false)
		})).To(Succeed())
	}
	tombstoneOf := func(ctx SpecContext, k channel.Key) (channel.Tombstone, bool) {
		GinkgoHelper()
		var ts []channel.Tombstone
		Expect(gorp.NewRetrieve[channel.Key, channel.Tombstone]().
			Entries(&ts).
			Exec(ctx, db)).To(Succeed())
		for _, t := range ts {
			if t.Key == k {
				return t, true
			}
		}
		return channel.Tombstone{}, false
	}
	stored := func(ctx SpecContext, k channel.Key) channel.Channel {
		GinkgoHelper()
		var ch channel.Channel
		Expect(svc.NewRetrieve().
			Where(channel.MatchKeys(k)).
			Entry(&ch).
			Exec(ctx, db)).To(Succeed())
		return ch
	}
	// dead creates a channel at the path of tpl that a display shows, deletes it, and
	// returns it and the display.
	dead := func(ctx SpecContext, tpl channel.Channel) (channel.Channel, string) {
		GinkgoHelper()
		ch := createAll(ctx, tpl)[0]
		display := show(ctx, ch.Key())
		del(ctx, ch.Key())
		_, ok := tombstoneOf(ctx, ch.Key())
		Expect(ok).To(BeTrue())
		return ch, display
	}

	It("Should heal again after a heal whose transaction rolls back", func(
		ctx SpecContext,
	) {
		d, display := dead(ctx, template("", nil))
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			ch := template(d.Name, nil)
			if err := svc.NewWriter(tx).Create(ctx, &ch); err != nil {
				return err
			}
			return errP2Rollback
		})).To(MatchError(errP2Rollback))
		Expect(shown(ctx, display)).To(Equal([]channel.Key{d.Key()}))
		_, ok := tombstoneOf(ctx, d.Key())
		Expect(ok).To(BeTrue())
		ch := createAll(ctx, template(d.Name, nil))[0]
		Expect(shown(ctx, display)).To(Equal([]channel.Key{ch.Key()}))
		_, ok = tombstoneOf(ctx, d.Key())
		Expect(ok).To(BeFalse())
	})

	It("Should heal a create after the display is deleted in the same transaction",
		func(ctx SpecContext) {
			d, display := dead(ctx, template("", nil))
			other := show(ctx, d.Key())
			var ch channel.Channel
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := displays.NewDelete().
					Where(gorp.MatchKeys[string, breakDisplay](display)).
					Exec(ctx, tx); err != nil {
					return err
				}
				ch = template(d.Name, nil)
				return svc.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			Expect(shown(ctx, other)).To(Equal([]channel.Key{ch.Key()}))
			_, ok := tombstoneOf(ctx, d.Key())
			Expect(ok).To(BeFalse())
		})

	It("Should heal a display created in the transaction of the create", func(
		ctx SpecContext,
	) {
		d, _ := dead(ctx, template("", nil))
		var (
			ch    channel.Channel
			fresh = breakDisplay{
				Key:      UniqueChannelName(),
				Channels: []channel.Key{d.Key()},
			}
		)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := displays.NewCreate().Entry(&fresh).Exec(ctx, tx); err != nil {
				return err
			}
			ch = template(d.Name, nil)
			return svc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		Expect(shown(ctx, fresh.Key)).To(Equal([]channel.Key{ch.Key()}))
	})

	It("Should leave a display that stops showing the dead key in the transaction",
		func(ctx SpecContext) {
			d, display := dead(ctx, template("", nil))
			keep := createAll(ctx, template("", nil))[0]
			var ch channel.Channel
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := displays.NewCreate().
					Entry(&breakDisplay{
						Key:      display,
						Channels: []channel.Key{keep.Key()},
					}).
					Exec(ctx, tx); err != nil {
					return err
				}
				ch = template(d.Name, nil)
				return svc.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			Expect(shown(ctx, display)).To(Equal([]channel.Key{keep.Key()}))
		})

	It("Should tombstone a healed channel that is deleted in the same transaction",
		func(ctx SpecContext) {
			d, display := dead(ctx, template("", nil))
			var ch channel.Channel
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				w := svc.NewWriter(tx)
				ch = template(d.Name, nil)
				if err := w.Create(ctx, &ch); err != nil {
					return err
				}
				return w.Delete(ctx, ch.Key(), false)
			})).To(Succeed())
			Expect(shown(ctx, display)).To(Equal([]channel.Key{ch.Key()}))
			t, ok := tombstoneOf(ctx, ch.Key())
			Expect(ok).To(BeTrue())
			Expect(t.Name).To(Equal(d.Name))
			_, ok = tombstoneOf(ctx, d.Key())
			Expect(ok).To(BeFalse())
		})

	It("Should heal a rename in the transaction that deletes the dead channel", func(
		ctx SpecContext,
	) {
		chs := createAll(ctx, template("", nil), template("", nil))
		gone, live := chs[0], chs[1]
		display := show(ctx, gone.Key())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := svc.NewWriter(tx)
			if err := w.Delete(ctx, gone.Key(), false); err != nil {
				return err
			}
			return w.Rename(ctx, live.Key(), gone.Name, false)
		})).To(Succeed())
		Expect(shown(ctx, display)).To(Equal([]channel.Key{live.Key()}))
		_, ok := tombstoneOf(ctx, gone.Key())
		Expect(ok).To(BeFalse())
	})

	It("Should heal two renames onto two tombstone paths in one call", func(
		ctx SpecContext,
	) {
		a, displayA := dead(ctx, template("", nil))
		b, displayB := dead(ctx, template("", nil))
		chs := createAll(ctx, template("", nil), template("", nil))
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).RenameMany(
				ctx,
				[]channel.Key{chs[0].Key(), chs[1].Key()},
				[]string{a.Name, b.Name},
				false,
			)
		})).To(Succeed())
		Expect(shown(ctx, displayA)).To(Equal([]channel.Key{chs[0].Key()}))
		Expect(shown(ctx, displayB)).To(Equal([]channel.Key{chs[1].Key()}))
		Expect(stored(ctx, chs[0].Key()).Rewires).To(HaveLen(1))
		Expect(stored(ctx, chs[1].Key()).Rewires).To(HaveLen(1))
	})

	It("Should heal a rename and a move onto two tombstone paths in one Apply", func(
		ctx SpecContext,
	) {
		g := folder(ctx)
		d, display := dead(ctx, template("", &g.Key))
		ch := createAll(ctx, template("", nil))[0]
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			_, err := svc.NewWriter(tx).Apply(
				ctx,
				ch.Key(),
				channel.NewRenameAction(channel.RenamePayload{Name: d.Name}),
				channel.NewSetAction(channel.SetPayload{
					Path:  []string{"group"},
					Value: &g.Key,
				}),
			)
			return err
		})).To(Succeed())
		Expect(shown(ctx, display)).To(Equal([]channel.Key{ch.Key()}))
		_, ok := tombstoneOf(ctx, d.Key())
		Expect(ok).To(BeFalse())
	})

	It("Should heal a move that names one channel twice", func(ctx SpecContext) {
		g := folder(ctx)
		d, display := dead(ctx, template("", &g.Key))
		ch := createAll(ctx, template(d.Name, nil))[0]
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).SetGroup(ctx, &g.Key, ch.Key(), ch.Key())
		})).To(Succeed())
		Expect(shown(ctx, display)).To(Equal([]channel.Key{ch.Key()}))
		Expect(stored(ctx, ch.Key()).Rewires).To(HaveLen(1))
	})

	It("Should keep both positions of a display that shows the dead key twice", func(
		ctx SpecContext,
	) {
		ch := createAll(ctx, template("", nil))[0]
		display := show(ctx, ch.Key(), ch.Key())
		del(ctx, ch.Key())
		healed := createAll(ctx, template(ch.Name, nil))[0]
		Expect(shown(ctx, display)).To(Equal([]channel.Key{healed.Key(), healed.Key()}))
	})

	It("Should heal an index channel that takes a tombstone path", func(
		ctx SpecContext,
	) {
		d, display := dead(ctx, template("", nil))
		idx := createAll(ctx, channel.Channel{
			Name:     d.Name,
			DataType: telem.TimestampT,
			IsIndex:  true,
		})[0]
		Expect(shown(ctx, display)).To(Equal([]channel.Key{idx.Key()}))
	})

	It("Should heal a path that a channel leaves and another takes in one transaction",
		func(ctx SpecContext) {
			d, display := dead(ctx, template("", nil))
			chs := createAll(ctx, template("", nil), template("", nil))
			mover, taker := chs[0], chs[1]
			moverDisplay := show(ctx, mover.Key())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				w := svc.NewWriter(tx)
				if err := w.Rename(ctx, mover.Key(), d.Name, false); err != nil {
					return err
				}
				return w.Rename(ctx, taker.Key(), mover.Name, false)
			})).To(Succeed())
			Expect(shown(ctx, display)).To(Equal([]channel.Key{mover.Key()}))
			Expect(shown(ctx, moverDisplay)).To(Equal([]channel.Key{mover.Key()}))
			Expect(stored(ctx, taker.Key()).Rewires).To(BeEmpty())
		})

	It("Should heal a calculated channel and its index that take two tombstone paths",
		func(ctx SpecContext) {
			name := UniqueChannelName()
			d, display := dead(ctx, template(name, nil))
			dIdx, idxDisplay := dead(ctx, template(name+"_time", nil))
			calc := channel.Channel{Name: name, Expression: "return 1.0 + 2.0"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &calc)
			})).To(Succeed())
			Expect(shown(ctx, display)).To(Equal([]channel.Key{calc.Key()}))
			Expect(shown(ctx, idxDisplay)).To(Equal([]channel.Key{calc.Index()}))
			_, ok := tombstoneOf(ctx, d.Key())
			Expect(ok).To(BeFalse())
			_, ok = tombstoneOf(ctx, dIdx.Key())
			Expect(ok).To(BeFalse())
		})

	It("Should heal a calculated channel that a display held after a delete with its "+
		"index", func(ctx SpecContext) {
		calc := channel.Channel{Name: UniqueChannelName(), Expression: "return 1.0"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &calc)
		})).To(Succeed())
		display := show(ctx, calc.Key(), calc.Index())
		del(ctx, calc.Key())
		again := channel.Channel{Name: calc.Name, Expression: "return 2.0"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &again)
		})).To(Succeed())
		Expect(shown(ctx, display)).To(Equal([]channel.Key{again.Key(), again.Index()}))
	})
})

var _ = Describe("Heal break p2 before the service heals", Ordered, ContinueOnFailure,
	func() {
		var (
			db       *gorp.DB
			svc      *channel.Service
			groupSvc *group.Service
			displays *gorp.Table[string, breakDisplay]
		)
		BeforeAll(func(ctx SpecContext) {
			ShouldNotLeakGoroutines()
			db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
			StartRun(db.Run)
			var cfg channel.ServiceConfig
			cfg, groupSvc = serviceConfig(ctx, testutil.NewNode(ctx), db)
			cfg.Retargets = func(d gorp.Dependent) (bool, error) {
				return d.Holder.Table == "break_display", nil
			}
			svc = MustOpen(channel.OpenService(ctx, cfg))
			displays = MustSucceed(gorp.OpenTable(ctx, breakDisplayConfig(db)))
			DeferCleanup(func() { Expect(displays.Close()).To(Succeed()) })
		})

		template := func(name string, g *group.Key) channel.Channel {
			if name == "" {
				name = UniqueChannelName()
			}
			return channel.Channel{
				Name:     name,
				Group:    g,
				DataType: telem.Float32T,
				Virtual:  true,
			}
		}
		write := func(ctx SpecContext, f func(w channel.Writer) error) {
			GinkgoHelper()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return f(svc.NewWriter(tx))
			})).To(Succeed())
		}
		show := func(ctx SpecContext, keys ...channel.Key) string {
			GinkgoHelper()
			d := breakDisplay{Key: UniqueChannelName(), Channels: keys}
			Expect(displays.NewCreate().Entry(&d).Exec(ctx, db)).To(Succeed())
			return d.Key
		}
		shown := func(ctx SpecContext, key string) []channel.Key {
			GinkgoHelper()
			var d breakDisplay
			Expect(gorp.NewRetrieve[string, breakDisplay]().
				Where(gorp.MatchKeys[string, breakDisplay](key)).
				Entry(&d).
				Exec(ctx, db)).To(Succeed())
			return d.Channels
		}

		It("Should heal a rename and a move that took tombstone paths before Heal",
			func(ctx SpecContext) {
				g := group.Group{Name: UniqueChannelName(), Scope: resource.TypeChannel}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return groupSvc.NewWriter(tx).Create(ctx, &g)
				})).To(Succeed())
				deadA, deadB := template("", nil), template("", &g.Key)
				write(ctx, func(w channel.Writer) error {
					return w.CreateMany(ctx, &[]channel.Channel{deadA, deadB})
				})
				var stored []channel.Channel
				Expect(svc.NewRetrieve().
					Where(channel.MatchNames(deadA.Name, deadB.Name)).
					Entries(&stored).
					Exec(ctx, db)).To(Succeed())
				Expect(stored).To(HaveLen(2))
				keyOf := map[string]channel.Key{}
				for _, c := range stored {
					keyOf[c.Name] = c.Key()
				}
				displayA := show(ctx, keyOf[deadA.Name])
				displayB := show(ctx, keyOf[deadB.Name])
				write(ctx, func(w channel.Writer) error {
					return w.DeleteMany(
						ctx,
						[]channel.Key{keyOf[deadA.Name], keyOf[deadB.Name]},
						false,
					)
				})
				chs := []channel.Channel{template("", nil), template(deadB.Name, nil)}
				write(ctx, func(w channel.Writer) error {
					return w.CreateMany(ctx, &chs)
				})
				write(ctx, func(w channel.Writer) error {
					err := w.Rename(ctx, chs[0].Key(), deadA.Name, false)
					if err != nil {
						return err
					}
					return w.SetGroup(ctx, &g.Key, chs[1].Key())
				})
				Expect(shown(ctx, displayA)).To(Equal([]channel.Key{keyOf[deadA.Name]}))
				Expect(svc.Heal(ctx)).To(Succeed())
				Expect(shown(ctx, displayA)).To(Equal([]channel.Key{chs[0].Key()}))
				Expect(shown(ctx, displayB)).To(Equal([]channel.Key{chs[1].Key()}))
			})
	})
