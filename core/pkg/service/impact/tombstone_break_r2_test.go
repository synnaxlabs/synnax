// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package impact_test

import (
	"cmp"
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/impact"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Tombstone break r2", Ordered, ContinueOnFailure, func() {
	var (
		layer *service.Layer
		db    *gorp.DB
		proj  project.Project
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		layer = MustOpen(svctestutil.OpenLayer(ctx, node))
		proj = project.Project{Name: "Tombstone break r2"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Project.NewWriter(tx).Create(ctx, &proj)
		})).To(Succeed())
	})
	createFolder := func(ctx SpecContext, parent *group.Group) group.Group {
		GinkgoHelper()
		g := group.Group{Name: UniqueChannelName(), Scope: resource.TypeChannel}
		if parent != nil {
			g.Parent = new(parent.ResourceID())
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Group.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		return g
	}
	createChannels := func(ctx SpecContext, n int, g *group.Key) []channel.Channel {
		GinkgoHelper()
		chs := make([]channel.Channel, n)
		for i := range chs {
			chs[i] = channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
				Group:    g,
			}
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).CreateMany(ctx, &chs)
		})).To(Succeed())
		return chs
	}
	// display creates a schematic with one value symbol for each of keys.
	display := func(ctx SpecContext, keys ...channel.Key) {
		GinkgoHelper()
		sch := schematic.Schematic{Name: UniqueChannelName(), Project: &proj.Key}
		for _, k := range keys {
			var value schematic.ValueNodeConfig
			value.SetDefaults()
			value.Channel = k
			sch.Nodes = append(sch.Nodes, schematic.Node{
				Key:    k.String(),
				Config: schematic.NodeConfig{Variant: value},
			})
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Schematic.NewWriter(tx).Create(ctx, &sch)
		})).To(Succeed())
	}
	deleteChannels := func(ctx SpecContext, keys ...channel.Key) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).DeleteMany(ctx, keys, false)
		})).To(Succeed())
	}
	tombstoneKeys := func(ctx SpecContext) channel.Keys {
		GinkgoHelper()
		var ts []channel.Tombstone
		Expect(gorp.NewRetrieve[channel.Key, channel.Tombstone]().
			Entries(&ts).
			Exec(ctx, db)).To(Succeed())
		keys := make(channel.Keys, len(ts))
		for i, t := range ts {
			keys[i] = t.Key
		}
		return keys
	}
	deleteGroups := func(ctx SpecContext, keys ...group.Key) (impact.Plan, error) {
		return layer.Impact.Apply(
			ctx,
			db,
			impact.Change{Variant: impact.DeleteGroupsChange{Keys: keys}},
			impact.ApplyOptions{},
		)
	}

	It("Should sort the tombstone entries of a folder delete by record", func(
		ctx SpecContext,
	) {
		parent := createFolder(ctx, nil)
		child := createFolder(ctx, &parent)
		chs := append(
			createChannels(ctx, 8, &parent.Key),
			createChannels(ctx, 8, &child.Key)...,
		)
		keys := channel.KeysFromChannels(chs)
		display(ctx, keys...)
		deleteChannels(ctx, keys...)
		p := MustSucceed(deleteGroups(ctx, child.Key, parent.Key))
		want := make([]impact.Entry, len(chs))
		for i, ch := range chs {
			path := parent.Name + "." + ch.Name
			if *ch.Group == child.Key {
				path = parent.Name + "." + child.Name + "." + ch.Name
			}
			want[i] = impact.Entry{
				Record: channel.ResourceID(ch.Key()),
				Effect: impact.EffectCascade,
				Path:   &path,
			}
		}
		slices.SortFunc(want, func(a, b impact.Entry) int {
			return cmp.Or(
				strings.Compare(string(a.Record.Type), string(b.Record.Type)),
				strings.Compare(a.Record.Key, b.Record.Key),
			)
		})
		Expect(p.Entries).To(Equal(want), "Plan.Entries are sorted by record")
	})

	It("Should count a display, and not a tombstone, that keeps a channel", func(
		ctx SpecContext,
	) {
		past := createChannels(ctx, 1, nil)[0]
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
			Rewires:  []channel.Rewire{{From: past.Key(), At: telem.Now()}},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		display(ctx, ch.Key())
		deleteChannels(ctx, ch.Key())
		Expect(tombstoneKeys(ctx)).To(ContainElement(ch.Key()))
		display(ctx, past.Key())
		uses := MustSucceed(layer.Channel.Uses(ctx, db, []channel.Key{past.Key()}))
		Expect(uses[0]).To(HaveLen(1))
		Expect(uses[0][0].Holder.Table).To(Equal("schematic"))
		dev := device.Device{
			Key:      UniqueChannelName(),
			Rack:     layer.Rack.EmbeddedKey,
			Name:     "plc",
			Make:     "modbus",
			Model:    "modbus",
			Location: "127.0.0.1:502",
		}
		tsk := task.Task{
			Name: "modbus_read",
			Type: "modbus_read",
			Config: msgpack.EncodedJSON{
				"device": dev.Key,
				"channels": []any{map[string]any{
					"type":    "coil",
					"key":     "coil-1",
					"channel": past.Key(),
				}},
			},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := layer.Device.NewWriter(tx).Create(ctx, &dev); err != nil {
				return err
			}
			return layer.Task.NewWriter(tx).Create(ctx, &tsk)
		})).To(Succeed())
		var p impact.Plan
		Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
			p, err = layer.Impact.Apply(
				ctx,
				tx,
				impact.Change{Variant: impact.DeleteChannelsChange{
					Keys: []channel.Key{past.Key()},
				}},
				impact.ApplyOptions{Picks: map[string]impact.Choice{
					task.ResourceID(tsk.Key).String(): impact.ChoiceDelete,
				}},
			)
			return err
		})).To(Succeed())
		Expect(p.Kept).To(Equal(uint32(1)))
		Expect(tombstoneKeys(ctx)).To(ContainElement(past.Key()))
	})

	It("Should keep the tombstones of a folder whose delete fails", func(
		ctx SpecContext,
	) {
		parent := createFolder(ctx, nil)
		child := createFolder(ctx, &parent)
		dead := createChannels(ctx, 1, &child.Key)[0]
		display(ctx, dead.Key())
		deleteChannels(ctx, dead.Key())
		Expect(deleteGroups(ctx, parent.Key)).Error().To(MatchError(
			ContainSubstring("not empty"),
		))
		Expect(tombstoneKeys(ctx)).To(ContainElement(dead.Key()))
		live := createChannels(ctx, 1, &child.Key)[0]
		Expect(deleteGroups(ctx, parent.Key, child.Key)).Error().To(MatchError(
			ContainSubstring("not empty"),
		))
		Expect(tombstoneKeys(ctx)).To(ContainElement(dead.Key()))
		deleteChannels(ctx, live.Key())
		path := parent.Name + "." + child.Name + "." + dead.Name
		Expect(deleteGroups(ctx, parent.Key, child.Key)).To(Equal(impact.Plan{
			Change: impact.Change{Variant: impact.DeleteGroupsChange{
				Keys: []group.Key{parent.Key, child.Key},
			}},
			Entries: []impact.Entry{{
				Record: channel.ResourceID(dead.Key()),
				Effect: impact.EffectCascade,
				Path:   &path,
			}},
		}))
		Expect(tombstoneKeys(ctx)).ToNot(ContainElement(dead.Key()))
	})

	It("Should refuse a pick on a tombstone entry and keep the folder", func(
		ctx SpecContext,
	) {
		folder := createFolder(ctx, nil)
		dead := createChannels(ctx, 1, &folder.Key)[0]
		display(ctx, dead.Key())
		deleteChannels(ctx, dead.Key())
		Expect(layer.Impact.Apply(
			ctx,
			db,
			impact.Change{Variant: impact.DeleteGroupsChange{
				Keys: []group.Key{folder.Key},
			}},
			impact.ApplyOptions{Picks: map[string]impact.Choice{
				channel.ResourceID(dead.Key()).String(): impact.ChoiceDelete,
			}},
		)).Error().To(MatchError(ContainSubstring("validation")))
		Expect(tombstoneKeys(ctx)).To(ContainElement(dead.Key()))
		Expect(MustSucceed(layer.Group.NewRetrieve().
			Where(group.MatchKeys(folder.Key)).
			Exists(ctx, db))).To(BeTrue())
	})
})
