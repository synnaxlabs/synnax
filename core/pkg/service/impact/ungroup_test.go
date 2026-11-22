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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/impact"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Ungroup groups", Ordered, func() {
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
		proj = project.Project{Name: "impact-ungroup"}
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
	createChannel := func(ctx SpecContext, name string, folder *group.Group) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
		if folder != nil {
			ch.Group = &folder.Key
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	// tombstone creates a channel of name in folder that a schematic shows, then
	// deletes it, which leaves its tombstone. It returns the channel and the schematic.
	tombstone := func(
		ctx SpecContext,
		name string,
		folder *group.Group,
	) (channel.Channel, schematic.Key) {
		GinkgoHelper()
		ch := createChannel(ctx, name, folder)
		var value schematic.ValueNodeConfig
		value.SetDefaults()
		value.Channel = ch.Key()
		sch := schematic.Schematic{
			Name:    UniqueChannelName(),
			Project: &proj.Key,
			Nodes: []schematic.Node{{
				Key:    "value",
				Config: schematic.NodeConfig{Variant: value},
			}},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Schematic.NewWriter(tx).Create(ctx, &sch)
		})).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).Delete(ctx, ch.Key(), false)
		})).To(Succeed())
		return ch, sch.Key
	}
	// shown returns the channel that the value node of the schematic with key shows.
	shown := func(ctx SpecContext, key schematic.Key) channel.Key {
		GinkgoHelper()
		var sch schematic.Schematic
		Expect(layer.Schematic.NewRetrieve().
			Where(schematic.MatchKeys(key)).
			Entry(&sch).
			Exec(ctx, db)).To(Succeed())
		value, ok := sch.Nodes[0].Config.Variant.(schematic.ValueNodeConfig)
		Expect(ok).To(BeTrue())
		return value.Channel
	}
	tombstonesIn := func(ctx SpecContext, folder group.Group) channel.Keys {
		GinkgoHelper()
		ts := MustSucceed(layer.Channel.Tombstones(ctx, db, folder.Key))
		keys := make(channel.Keys, len(ts))
		for i, t := range ts {
			keys[i] = t.Key
		}
		return keys
	}
	groupOf := func(ctx SpecContext, k channel.Key) *group.Key {
		GinkgoHelper()
		var ch channel.Channel
		Expect(layer.Channel.NewRetrieve().
			Where(channel.MatchKeys(k)).
			Entry(&ch).
			Exec(ctx, db)).To(Succeed())
		return ch.Group
	}
	retrieveFolder := func(ctx SpecContext, k group.Key) (group.Group, error) {
		var g group.Group
		err := layer.Group.NewRetrieve().
			Where(group.MatchKeys(k)).
			Entry(&g).
			Exec(ctx, db)
		return g, err
	}
	exists := func(ctx SpecContext, k group.Key) bool {
		GinkgoHelper()
		return MustSucceed(layer.Group.NewRetrieve().
			Where(group.MatchKeys(k)).
			Exists(ctx, db))
	}
	ungroupOf := func(keys ...group.Key) impact.Change {
		return impact.Change{Variant: impact.UngroupGroupsChange{Keys: keys}}
	}
	apply := func(
		ctx SpecContext,
		opts impact.ApplyOptions,
		keys ...group.Key,
	) (impact.Plan, error) {
		return layer.Impact.Apply(ctx, db, ungroupOf(keys...), opts)
	}
	plan := func(
		ctx SpecContext,
		opts impact.PlanOptions,
		keys ...group.Key,
	) (impact.Plan, error) {
		return layer.Impact.Plan(ctx, db, ungroupOf(keys...), opts)
	}
	dropped := func(ch channel.Channel, folder group.Group) impact.Entry {
		path := folder.Name + "." + ch.Name
		return impact.Entry{
			Record: channel.ResourceID(ch.Key()),
			Effect: impact.EffectCascade,
			Path:   &path,
		}
	}

	It("Should move a folder's channels, folders, and tombstones to its parent", func(
		ctx SpecContext,
	) {
		parent := createFolder(ctx, nil)
		folder := createFolder(ctx, &parent)
		inner := createFolder(ctx, &folder)
		live := createChannel(ctx, UniqueChannelName(), &folder)
		dead, _ := tombstone(ctx, UniqueChannelName(), &folder)
		Expect(apply(ctx, impact.ApplyOptions{}, folder.Key)).
			To(Equal(impact.Plan{Change: ungroupOf(folder.Key)}))
		Expect(exists(ctx, folder.Key)).To(BeFalse())
		Expect(groupOf(ctx, live.Key())).To(Equal(&parent.Key))
		Expect(MustSucceed(retrieveFolder(ctx, inner.Key)).Parent).
			To(Equal(new(parent.ResourceID())))
		Expect(tombstonesIn(ctx, parent)).To(ConsistOf(dead.Key()))
	})

	It("Should take members out of any folder when the parent is not a folder", func(
		ctx SpecContext,
	) {
		folder := createFolder(ctx, nil)
		live := createChannel(ctx, UniqueChannelName(), &folder)
		dead, sch := tombstone(ctx, UniqueChannelName(), &folder)
		Expect(apply(ctx, impact.ApplyOptions{}, folder.Key)).
			To(Equal(impact.Plan{Change: ungroupOf(folder.Key)}))
		Expect(groupOf(ctx, live.Key())).To(BeNil())
		healed := createChannel(ctx, dead.Name, nil)
		Expect(shown(ctx, sch)).To(Equal(healed.Key()))
	})

	It("Should heal a parent's tombstone when a moved channel takes its path", func(
		ctx SpecContext,
	) {
		parent := createFolder(ctx, nil)
		folder := createFolder(ctx, &parent)
		name := UniqueChannelName()
		dead, sch := tombstone(ctx, name, &parent)
		live := createChannel(ctx, name, &folder)
		Expect(apply(ctx, impact.ApplyOptions{}, folder.Key)).
			To(Equal(impact.Plan{Change: ungroupOf(folder.Key)}))
		Expect(shown(ctx, sch)).To(Equal(live.Key()))
		Expect(tombstonesIn(ctx, parent)).ToNot(ContainElement(dead.Key()))
	})

	It("Should drop and list a tombstone whose new path holds a channel", func(
		ctx SpecContext,
	) {
		parent := createFolder(ctx, nil)
		folder := createFolder(ctx, &parent)
		name := UniqueChannelName()
		dead, sch := tombstone(ctx, name, &folder)
		createChannel(ctx, name, &parent)
		p := MustSucceed(plan(ctx, impact.PlanOptions{}, folder.Key))
		Expect(p.Entries).To(Equal([]impact.Entry{dropped(dead, folder)}))
		Expect(exists(ctx, folder.Key)).To(BeTrue())
		Expect(apply(ctx, impact.ApplyOptions{Token: p.Token}, folder.Key)).To(Equal(p))
		Expect(tombstonesIn(ctx, parent)).To(BeEmpty())
		Expect(shown(ctx, sch)).To(Equal(dead.Key()))
	})

	It("Should drop and list a tombstone whose new path holds a tombstone", func(
		ctx SpecContext,
	) {
		parent := createFolder(ctx, nil)
		folder := createFolder(ctx, &parent)
		name := UniqueChannelName()
		kept, _ := tombstone(ctx, name, &parent)
		dead, _ := tombstone(ctx, name, &folder)
		Expect(apply(ctx, impact.ApplyOptions{}, folder.Key)).To(Equal(impact.Plan{
			Change:  ungroupOf(folder.Key),
			Entries: []impact.Entry{dropped(dead, folder)},
		}))
		Expect(tombstonesIn(ctx, parent)).To(ConsistOf(kept.Key()))
	})

	It("Should drop a tombstone whose path a channel that the change moves takes", func(
		ctx SpecContext,
	) {
		parent := createFolder(ctx, nil)
		first := createFolder(ctx, &parent)
		second := createFolder(ctx, &parent)
		name := UniqueChannelName()
		dead, _ := tombstone(ctx, name, &second)
		live := createChannel(ctx, name, &first)
		Expect(apply(ctx, impact.ApplyOptions{}, first.Key, second.Key)).To(Equal(
			impact.Plan{
				Change:  ungroupOf(first.Key, second.Key),
				Entries: []impact.Entry{dropped(dead, second)},
			},
		))
		Expect(groupOf(ctx, live.Key())).To(Equal(&parent.Key))
		Expect(tombstonesIn(ctx, parent)).To(BeEmpty())
	})

	It("Should drop the tombstone of the later folder when two move to one path", func(
		ctx SpecContext,
	) {
		parent := createFolder(ctx, nil)
		a := createFolder(ctx, &parent)
		b := createFolder(ctx, &parent)
		name := UniqueChannelName()
		inA, _ := tombstone(ctx, name, &a)
		inB, _ := tombstone(ctx, name, &b)
		kept, gone := inA, dropped(inB, b)
		if string(b.Key[:]) < string(a.Key[:]) {
			kept, gone = inB, dropped(inA, a)
		}
		Expect(apply(ctx, impact.ApplyOptions{}, a.Key, b.Key)).To(Equal(impact.Plan{
			Change:  ungroupOf(a.Key, b.Key),
			Entries: []impact.Entry{gone},
		}))
		Expect(tombstonesIn(ctx, parent)).To(ConsistOf(kept.Key()))
	})

	It("Should move a symbol to the parent folder of its folder", func(
		ctx SpecContext,
	) {
		parent := group.Group{Name: "symbols", Scope: resource.TypeSchematicSymbol}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Group.NewWriter(tx).Create(ctx, &parent)
		})).To(Succeed())
		folder := group.Group{
			Name:   "valves",
			Scope:  resource.TypeSchematicSymbol,
			Parent: new(parent.ResourceID()),
		}
		sym := defaults.Fill(symbol.Symbol{
			Name: "Inlet",
			Data: symbol.Spec{SVG: "<svg/>", Variant: "valve"},
		})
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := layer.Group.NewWriter(tx).Create(ctx, &folder); err != nil {
				return err
			}
			sym.Group = &folder.Key
			return layer.Schematic.Symbol.NewWriter(tx).Create(ctx, &sym)
		})).To(Succeed())
		MustSucceed(apply(ctx, impact.ApplyOptions{}, folder.Key))
		var moved symbol.Symbol
		Expect(layer.Schematic.Symbol.NewRetrieve().
			Where(symbol.MatchKeys(sym.Key)).
			Entry(&moved).
			Exec(ctx, db)).To(Succeed())
		Expect(moved.Group).To(Equal(&parent.Key))
		Expect(exists(ctx, folder.Key)).To(BeFalse())
	})

	It("Should take a schematic out of its folder when the parent is a project", func(
		ctx SpecContext,
	) {
		folder := group.Group{
			Name:   "pids",
			Scope:  resource.TypeProject,
			Parent: new(proj.ResourceID()),
		}
		sch := schematic.Schematic{Name: UniqueChannelName(), Project: &proj.Key}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := layer.Group.NewWriter(tx).Create(ctx, &folder); err != nil {
				return err
			}
			sch.Group = &folder.Key
			return layer.Schematic.NewWriter(tx).Create(ctx, &sch)
		})).To(Succeed())
		MustSucceed(apply(ctx, impact.ApplyOptions{}, folder.Key))
		var moved schematic.Schematic
		Expect(layer.Schematic.NewRetrieve().
			Where(schematic.MatchKeys(sch.Key)).
			Entry(&moved).
			Exec(ctx, db)).To(Succeed())
		Expect(moved.Group).To(BeNil())
		Expect(moved.Project).To(Equal(&proj.Key))
		Expect(exists(ctx, folder.Key)).To(BeFalse())
	})

	It("Should refuse a folder whose parent the change also ungroups", func(
		ctx SpecContext,
	) {
		parent := createFolder(ctx, nil)
		folder := createFolder(ctx, &parent)
		refused := And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(
				"cannot ungroup %s and %s, which holds it, in one change",
				folder.Name,
				parent.Name,
			)),
		)
		Expect(plan(ctx, impact.PlanOptions{}, parent.Key, folder.Key)).
			Error().To(refused)
		Expect(apply(ctx, impact.ApplyOptions{}, parent.Key, folder.Key)).
			Error().To(refused)
		Expect(exists(ctx, parent.Key)).To(BeTrue())
		Expect(exists(ctx, folder.Key)).To(BeTrue())
	})

	It("Should refuse a token planned before a tombstone's new path was taken", func(
		ctx SpecContext,
	) {
		parent := createFolder(ctx, nil)
		folder := createFolder(ctx, &parent)
		name := UniqueChannelName()
		dead, _ := tombstone(ctx, name, &folder)
		stale := MustSucceed(plan(ctx, impact.PlanOptions{}, folder.Key))
		Expect(stale.Entries).To(BeEmpty())
		Expect(stale.Token).To(HaveLen(16))
		Expect(plan(ctx, impact.PlanOptions{}, folder.Key)).
			To(HaveField("Token", stale.Token))
		createChannel(ctx, name, &parent)
		applied, err := apply(ctx, impact.ApplyOptions{Token: stale.Token}, folder.Key)
		Expect(applied).To(BeZero())
		changed := MustBeOk(errors.AsType[impact.PlanChangedError](err))
		Expect(changed.Plan.Entries).To(Equal([]impact.Entry{dropped(dead, folder)}))
		Expect(changed.Plan).To(Equal(MustSucceed(
			plan(ctx, impact.PlanOptions{}, folder.Key),
		)))
		Expect(exists(ctx, folder.Key)).To(BeTrue())
	})

	It("Should give an ungroup a token apart from a delete of the same folders", func(
		ctx SpecContext,
	) {
		folder := createFolder(ctx, nil)
		ungroup := MustSucceed(plan(ctx, impact.PlanOptions{}, folder.Key))
		del := MustSucceed(layer.Impact.Plan(
			ctx,
			db,
			impact.Change{
				Variant: impact.DeleteGroupsChange{Keys: []group.Key{folder.Key}},
			},
			impact.PlanOptions{},
		))
		Expect(ungroup.Token).ToNot(Equal(del.Token))
	})

	It("Should refuse a pick on a dropped tombstone, which takes none", func(
		ctx SpecContext,
	) {
		parent := createFolder(ctx, nil)
		folder := createFolder(ctx, &parent)
		name := UniqueChannelName()
		dead, _ := tombstone(ctx, name, &folder)
		createChannel(ctx, name, &parent)
		record := channel.ResourceID(dead.Key()).String()
		picks := map[string]impact.Choice{record: impact.ChoiceDelete}
		refused := And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("entry %s cannot take delete", record)),
		)
		Expect(plan(ctx, impact.PlanOptions{Picks: picks}, folder.Key)).
			Error().To(refused)
		Expect(apply(ctx, impact.ApplyOptions{Picks: picks}, folder.Key)).
			Error().To(refused)
		Expect(exists(ctx, folder.Key)).To(BeTrue())
	})
})
