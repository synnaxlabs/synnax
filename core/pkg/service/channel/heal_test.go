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

// healHolder holds a channel through a weak link that can move.
type healHolder struct {
	Key     string
	Channel channel.Key
}

func (h healHolder) GorpKey() string { return h.Key }

// healKeeper holds a channel through a weak link that the heal policy keeps.
type healKeeper struct {
	Key     string
	Channel channel.Key
}

func (h healKeeper) GorpKey() string { return h.Key }

func weakChannelReference[E gorp.Entry[string]](
	channelOf func(*E) *channel.Key,
) []gorp.Reference[string, E] {
	return []gorp.Reference[string, E]{gorp.NewKeyReference(
		gorp.KeyReferenceConfig[string, E, channel.Key]{
			Name:     "channel",
			Table:    "channel",
			OnDelete: gorp.Weak,
			Kind:     gorp.Use,
			Tag:      1,
			Targets: func(e *E) []channel.Key {
				return []channel.Key{*channelOf(e)}
			},
			Detach: func(e *E, _ channel.Key) { *channelOf(e) = 0 },
			Retarget: func(e *E, to map[channel.Key]channel.Key) {
				if k, ok := to[*channelOf(e)]; ok {
					*channelOf(e) = k
				}
			},
		},
	)}
}

var _ = Describe("Heal", Ordered, func() {
	var (
		db       *gorp.DB
		svc      *channel.Service
		groupSvc *group.Service
		displays *gorp.Table[string, healHolder]
		keepers  *gorp.Table[string, healKeeper]
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		var cfg channel.ServiceConfig
		cfg, groupSvc = serviceConfig(ctx, testutil.NewNode(ctx), db)
		cfg.Retargets = func(d gorp.Dependent) (bool, error) {
			return d.Holder.Table == "display", nil
		}
		svc = MustOpen(channel.OpenService(ctx, cfg))
		displays = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, healHolder]{
			DB:   db,
			Name: "display",
			References: weakChannelReference(func(h *healHolder) *channel.Key {
				return &h.Channel
			}),
		}))
		keepers = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, healKeeper]{
			DB:   db,
			Name: "keeper",
			References: weakChannelReference(func(h *healKeeper) *channel.Key {
				return &h.Channel
			}),
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
	displayOf := func(ctx SpecContext, key string) channel.Key {
		GinkgoHelper()
		var h healHolder
		Expect(displays.NewRetrieve().
			Where(gorp.MatchKeys[string, healHolder](key)).
			Entry(&h).
			Exec(ctx, db)).To(Succeed())
		return h.Channel
	}
	keeperOf := func(ctx SpecContext, key string) channel.Key {
		GinkgoHelper()
		var h healKeeper
		Expect(keepers.NewRetrieve().
			Where(gorp.MatchKeys[string, healKeeper](key)).
			Entry(&h).
			Exec(ctx, db)).To(Succeed())
		return h.Channel
	}
	tombstones := func(ctx SpecContext) []channel.Tombstone {
		GinkgoHelper()
		var ts []channel.Tombstone
		Expect(gorp.NewRetrieve[channel.Key, channel.Tombstone]().
			Entries(&ts).
			Exec(ctx, db)).To(Succeed())
		return ts
	}
	// held creates a channel at the path of ch that a display and a keeper hold,
	// deletes it, and returns its key and the keys of its holders.
	held := func(
		ctx SpecContext,
		ch channel.Channel,
	) (dead channel.Key, display, keeper string) {
		GinkgoHelper()
		dead = create(ctx, ch)[0].Key()
		display, keeper = UniqueChannelName(), UniqueChannelName()
		Expect(displays.NewCreate().
			Entry(&healHolder{Key: display, Channel: dead}).
			Exec(ctx, db)).To(Succeed())
		Expect(keepers.NewCreate().
			Entry(&healKeeper{Key: keeper, Channel: dead}).
			Exec(ctx, db)).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Delete(ctx, dead, false)
		})).To(Succeed())
		return dead, display, keeper
	}
	expectHealed := func(
		ctx SpecContext,
		dead channel.Key,
		display, keeper string,
		ch channel.Channel,
	) {
		GinkgoHelper()
		Expect(displayOf(ctx, display)).To(Equal(ch.Key()))
		Expect(keeperOf(ctx, keeper)).To(Equal(dead))
		healed := retrieve(ctx, ch.Key())
		Expect(healed.Rewires).To(HaveLen(1))
		Expect(healed.Rewires[0].From).To(BeZero())
		Expect(tombstones(ctx)).ToNot(ContainElement(HaveField("Key", dead)))
	}

	It("Should heal the display links of a create at a tombstone path", func(
		ctx SpecContext,
	) {
		name := UniqueChannelName()
		dead, display, keeper := held(ctx, channel.Channel{Name: name})
		ch := create(ctx, channel.Channel{Name: name})[0]
		expectHealed(ctx, dead, display, keeper, ch)
	})

	It("Should heal the display links of a rename onto a tombstone path", func(
		ctx SpecContext,
	) {
		name := UniqueChannelName()
		dead, display, keeper := held(ctx, channel.Channel{Name: name})
		ch := create(ctx, channel.Channel{})[0]
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Rename(ctx, ch.Key(), name, false)
		})).To(Succeed())
		expectHealed(ctx, dead, display, keeper, ch)
	})

	It("Should heal the display links of a move onto a tombstone path", func(
		ctx SpecContext,
	) {
		g := group.Group{Name: UniqueChannelName(), Scope: resource.TypeChannel}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		name := UniqueChannelName()
		dead, display, keeper := held(ctx, channel.Channel{Name: name, Group: &g.Key})
		ch := create(ctx, channel.Channel{Name: name})[0]
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).SetGroup(ctx, &g.Key, ch.Key())
		})).To(Succeed())
		expectHealed(ctx, dead, display, keeper, ch)
	})

	It("Should heal two tombstone paths in one create", func(ctx SpecContext) {
		a, b := UniqueChannelName(), UniqueChannelName()
		deadA, displayA, keeperA := held(ctx, channel.Channel{Name: a})
		deadB, displayB, keeperB := held(ctx, channel.Channel{Name: b})
		chs := create(ctx, channel.Channel{Name: a}, channel.Channel{Name: b})
		expectHealed(ctx, deadA, displayA, keeperA, chs[0])
		expectHealed(ctx, deadB, displayB, keeperB, chs[1])
	})

	It("Should leave the path history of another channel on the dead key", func(
		ctx SpecContext,
	) {
		name := UniqueChannelName()
		dead, _, _ := held(ctx, channel.Channel{Name: name})
		past := create(ctx, channel.Channel{})[0]
		past.Rewires = []channel.Rewire{{From: dead, At: telem.Now()}}
		Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).Set(ctx, past)).
			To(Succeed())
		create(ctx, channel.Channel{Name: name})
		Expect(retrieve(ctx, past.Key()).Rewires[0].From).To(Equal(dead))
	})

	It("Should keep a tombstone of a channel that holds rewires and no display", func(
		ctx SpecContext,
	) {
		name := UniqueChannelName()
		held(ctx, channel.Channel{Name: name})
		ch := create(ctx, channel.Channel{Name: name})[0]
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Delete(ctx, ch.Key(), false)
		})).To(Succeed())
		Expect(tombstones(ctx)).To(ContainElement(SatisfyAll(
			HaveField("Key", ch.Key()),
			HaveField("Rewires", HaveLen(1)),
		)))
	})

	It("Should drop the rewires that the caller of a create gives", func(
		ctx SpecContext,
	) {
		ch := create(ctx, channel.Channel{
			Rewires: []channel.Rewire{{From: 7, At: telem.Now()}},
		})[0]
		Expect(retrieve(ctx, ch.Key()).Rewires).To(BeEmpty())
	})
})
