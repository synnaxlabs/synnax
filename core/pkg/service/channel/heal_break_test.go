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
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// breakDisplay holds up to two channels through a weak link that a heal moves.
type breakDisplay struct {
	Key      string
	Channels []channel.Key
}

func (d breakDisplay) GorpKey() string { return d.Key }

func breakDisplayConfig(db *gorp.DB) gorp.TableConfig[string, breakDisplay] {
	return gorp.TableConfig[string, breakDisplay]{
		DB:   db,
		Name: "break_display",
		References: []gorp.Reference[string, breakDisplay]{gorp.NewKeyReference(
			gorp.KeyReferenceConfig[string, breakDisplay, channel.Key]{
				Name:     "channels",
				Table:    "channel",
				OnDelete: gorp.Weak,
				Kind:     gorp.Use,
				Tag:      1,
				Targets:  func(d *breakDisplay) []channel.Key { return d.Channels },
				Detach: func(d *breakDisplay, k channel.Key) {
					for i := range d.Channels {
						if d.Channels[i] == k {
							d.Channels[i] = 0
						}
					}
				},
				Retarget: func(d *breakDisplay, to map[channel.Key]channel.Key) {
					for i := range d.Channels {
						if k, ok := to[d.Channels[i]]; ok {
							d.Channels[i] = k
						}
					}
				},
			},
		)},
	}
}

var _ = Describe("Heal break", Ordered, ContinueOnFailure, func() {
	var (
		db       *gorp.DB
		svc      *channel.Service
		displays *gorp.Table[string, breakDisplay]
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		cfg, _ := serviceConfig(ctx, testutil.NewNode(ctx), db)
		cfg.Retargets = func(d gorp.Dependent) (bool, error) {
			return d.Holder.Table == "break_display", nil
		}
		svc = MustOpen(channel.OpenService(ctx, cfg))
		displays = MustSucceed(gorp.OpenTable(ctx, breakDisplayConfig(db)))
		DeferCleanup(func() { Expect(displays.Close()).To(Succeed()) })
	})

	createAll := func(ctx SpecContext, chs ...channel.Channel) []channel.Channel {
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
	tombstoned := func(ctx SpecContext, k channel.Key) bool {
		GinkgoHelper()
		var ts []channel.Tombstone
		Expect(gorp.NewRetrieve[channel.Key, channel.Tombstone]().
			Entries(&ts).
			Exec(ctx, db)).To(Succeed())
		for _, t := range ts {
			if t.Key == k {
				return true
			}
		}
		return false
	}
	It("Should leave the tombstone and drop a given list before the service heals",
		func(ctx SpecContext) {
			dead := createAll(ctx, channel.Channel{})[0]
			display := show(ctx, dead.Key())
			del(ctx, dead.Key())
			ch := createAll(ctx, channel.Channel{
				Rewires: []channel.Rewire{{From: 7, At: telem.Now()}},
			})[0]
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Rename(ctx, ch.Key(), dead.Name, false)
			})).To(Succeed())
			Expect(tombstoned(ctx, dead.Key())).To(BeTrue())
			Expect(shown(ctx, display)).To(Equal([]channel.Key{dead.Key()}))
			var stored channel.Channel
			Expect(svc.NewRetrieve().
				Where(channel.MatchKeys(ch.Key())).
				Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			Expect(stored.Rewires).To(BeEmpty())
		})

	It("Should heal a display that opens after a built-in takes the path of the "+
		"channel it held", func(ctx SpecContext) {
		name := UniqueChannelName()
		dead := createAll(ctx, channel.Channel{Name: name})[0]
		display := show(ctx, dead.Key())
		del(ctx, dead.Key())
		Expect(tombstoned(ctx, dead.Key())).To(BeTrue())
		// A Core start opens the channel service, and its Declares, before the
		// tables of displays.
		Expect(displays.Close()).To(Succeed())
		builtIn := channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Declare(ctx, &builtIn)
		})).To(Succeed())
		Expect(tombstoned(ctx, dead.Key())).To(BeTrue())
		displays = MustSucceed(gorp.OpenTable(ctx, breakDisplayConfig(db)))
		Expect(svc.Heal(ctx)).To(Succeed())
		Expect(shown(ctx, display)).To(Equal([]channel.Key{builtIn.Key()}))
		Expect(tombstoned(ctx, dead.Key())).To(BeFalse())
	})
})
