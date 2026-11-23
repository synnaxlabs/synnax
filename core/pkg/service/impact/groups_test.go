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
	"fmt"

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
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Delete groups", Ordered, func() {
	var (
		layer *service.Layer
		db    *gorp.DB
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		layer = MustOpen(svctestutil.OpenLayer(ctx, node))
	})

	createFolder := func(ctx SpecContext, name string, parent *group.Group) group.Group {
		GinkgoHelper()
		g := group.Group{Name: name, Scope: resource.TypeChannel}
		if parent != nil {
			g.Parent = new(parent.ResourceID())
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Group.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		return g
	}
	exists := func(ctx SpecContext, k group.Key) bool {
		GinkgoHelper()
		return MustSucceed(layer.Group.NewRetrieve().
			Where(group.MatchKeys(k)).
			Exists(ctx, db))
	}
	deleteOf := func(keys ...group.Key) impact.Change {
		return impact.Change{Variant: impact.DeleteGroupsChange{Keys: keys}}
	}
	apply := func(
		ctx SpecContext,
		opts impact.ApplyOptions,
		keys ...group.Key,
	) (impact.Plan, error) {
		return layer.Impact.Apply(ctx, db, deleteOf(keys...), opts)
	}
	plan := func(
		ctx SpecContext,
		opts impact.PlanOptions,
		keys ...group.Key,
	) (impact.Plan, error) {
		return layer.Impact.Plan(ctx, db, deleteOf(keys...), opts)
	}

	It("Should delete a folder and a folder inside it", func(ctx SpecContext) {
		parent := createFolder(ctx, "impact-parent", nil)
		child := createFolder(ctx, "impact-child", &parent)
		Expect(apply(ctx, impact.ApplyOptions{}, parent.Key, child.Key)).
			To(Equal(impact.Plan{Change: impact.Change{
				Variant: impact.DeleteGroupsChange{
					Keys: []group.Key{parent.Key, child.Key},
				},
			}}))
		Expect(exists(ctx, parent.Key)).To(BeFalse())
		Expect(exists(ctx, child.Key)).To(BeFalse())
	})

	// tombstone deletes a channel in folder that a schematic holds, which leaves its
	// tombstone in folder.
	tombstone := func(ctx SpecContext, folder group.Group) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
			Group:    &folder.Key,
		}
		proj := project.Project{Name: "impact-tombstones"}
		var value schematic.ValueNodeConfig
		value.SetDefaults()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := layer.Project.NewWriter(tx).Create(ctx, &proj); err != nil {
				return err
			}
			if err := layer.Channel.NewWriter(tx).Create(ctx, &ch); err != nil {
				return err
			}
			value.Channel = ch.Key()
			sch := schematic.Schematic{
				Name:    "pid",
				Project: &proj.Key,
				Nodes: []schematic.Node{{
					Key:    "value",
					Config: schematic.NodeConfig{Variant: value},
				}},
			}
			return layer.Schematic.NewWriter(tx).Create(ctx, &sch)
		})).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).Delete(ctx, ch.Key(), false)
		})).To(Succeed())
		return ch
	}

	cascade := func(ch channel.Channel, path string) impact.Entry {
		return impact.Entry{
			Record: channel.ResourceID(ch.Key()),
			Effect: impact.EffectCascade,
			Path:   &path,
		}
	}

	It("Should delete the tombstones of a folder first and list them", func(
		ctx SpecContext,
	) {
		folder := createFolder(ctx, "impact-tombstoned", nil)
		ch := tombstone(ctx, folder)
		Expect(apply(ctx, impact.ApplyOptions{}, folder.Key)).To(Equal(impact.Plan{
			Change:  deleteOf(folder.Key),
			Entries: []impact.Entry{cascade(ch, "impact-tombstoned."+ch.Name)},
		}))
		Expect(exists(ctx, folder.Key)).To(BeFalse())
	})

	It("Should give each tombstone its path through the deleted folders", func(
		ctx SpecContext,
	) {
		outer := createFolder(ctx, "impact-outer", nil)
		inner := createFolder(ctx, "impact-inner", &outer)
		inOuter := tombstone(ctx, outer)
		inInner := tombstone(ctx, inner)
		plan := MustSucceed(apply(ctx, impact.ApplyOptions{}, outer.Key, inner.Key))
		Expect(plan.Entries).To(ConsistOf(
			cascade(inOuter, "impact-outer."+inOuter.Name),
			cascade(inInner, "impact-outer.impact-inner."+inInner.Name),
		))
	})

	It("Should start a path at the deleted folder, not at a kept folder above it", func(
		ctx SpecContext,
	) {
		kept := createFolder(ctx, "impact-kept", nil)
		gone := createFolder(ctx, "impact-gone", &kept)
		ch := tombstone(ctx, gone)
		Expect(apply(ctx, impact.ApplyOptions{}, gone.Key)).To(Equal(impact.Plan{
			Change:  deleteOf(gone.Key),
			Entries: []impact.Entry{cascade(ch, "impact-gone."+ch.Name)},
		}))
		Expect(exists(ctx, kept.Key)).To(BeTrue())
	})

	It("Should refuse a folder that holds a channel", func(ctx SpecContext) {
		folder := createFolder(ctx, "impact-full", nil)
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
			Group:    &folder.Key,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		Expect(apply(ctx, impact.ApplyOptions{}, folder.Key)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(
				"cannot delete group impact-full because it is not empty",
			)),
		))
		Expect(exists(ctx, folder.Key)).To(BeTrue())
	})

	It("Should refuse an apply of no change as invalid", func(ctx SpecContext) {
		Expect(layer.Impact.Apply(ctx, db, impact.Change{}, impact.ApplyOptions{})).
			Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("unknown change")),
		))
	})

	It("Should plan a group delete with no entries and a token of its keys", func(
		ctx SpecContext,
	) {
		parent := createFolder(ctx, "impact-planned", nil)
		child := createFolder(ctx, "impact-planned-child", &parent)
		p := MustSucceed(plan(ctx, impact.PlanOptions{}, parent.Key, child.Key))
		Expect(p.Entries).To(BeEmpty())
		Expect(p.Kept).To(BeZero())
		Expect(p.Hidden).To(BeZero())
		Expect(p.Change).To(Equal(deleteOf(parent.Key, child.Key)))
		Expect(p.Token).To(HaveLen(16))
		Expect(plan(ctx, impact.PlanOptions{}, child.Key, parent.Key, child.Key)).
			To(HaveField("Token", p.Token))
		Expect(plan(ctx, impact.PlanOptions{}, child.Key)).
			To(HaveField("Token", Not(Equal(p.Token))))
		Expect(exists(ctx, parent.Key)).To(BeTrue())
	})

	It("Should apply a group delete with the token of its plan", func(
		ctx SpecContext,
	) {
		folder := createFolder(ctx, "impact-tokened", nil)
		p := MustSucceed(plan(ctx, impact.PlanOptions{}, folder.Key))
		Expect(apply(ctx, impact.ApplyOptions{Token: p.Token}, folder.Key)).
			To(Equal(p))
		Expect(exists(ctx, folder.Key)).To(BeFalse())
	})

	It("Should refuse the token of another plan and delete nothing", func(
		ctx SpecContext,
	) {
		folder := createFolder(ctx, "impact-stale", nil)
		other := createFolder(ctx, "impact-other", nil)
		stale := MustSucceed(plan(ctx, impact.PlanOptions{}, other.Key))
		applied, err := apply(ctx, impact.ApplyOptions{Token: stale.Token}, folder.Key)
		Expect(applied).To(BeZero())
		Expect(err).To(MatchError("the records that the change touches changed"))
		changed := MustBeOk(errors.AsType[impact.PlanChangedError](err))
		Expect(changed.Plan).To(Equal(MustSucceed(
			plan(ctx, impact.PlanOptions{}, folder.Key),
		)))
		Expect(exists(ctx, folder.Key)).To(BeTrue())
	})

	It("Should plan each tombstone of a group delete as the apply lists it", func(
		ctx SpecContext,
	) {
		folder := createFolder(ctx, "impact-planned-tombstones", nil)
		ch := tombstone(ctx, folder)
		p := MustSucceed(plan(ctx, impact.PlanOptions{}, folder.Key))
		Expect(p.Entries).To(Equal([]impact.Entry{
			cascade(ch, "impact-planned-tombstones."+ch.Name),
		}))
		Expect(exists(ctx, folder.Key)).To(BeTrue())
		Expect(apply(ctx, impact.ApplyOptions{Token: p.Token}, folder.Key)).
			To(Equal(p))
		Expect(exists(ctx, folder.Key)).To(BeFalse())
	})

	It("Should refuse a token planned before a tombstone joined the folder", func(
		ctx SpecContext,
	) {
		folder := createFolder(ctx, "impact-late-tombstone", nil)
		stale := MustSucceed(plan(ctx, impact.PlanOptions{}, folder.Key))
		ch := tombstone(ctx, folder)
		applied, err := apply(ctx, impact.ApplyOptions{Token: stale.Token}, folder.Key)
		Expect(applied).To(BeZero())
		changed := MustBeOk(errors.AsType[impact.PlanChangedError](err))
		Expect(changed.Plan.Entries).To(Equal([]impact.Entry{
			cascade(ch, "impact-late-tombstone."+ch.Name),
		}))
		Expect(changed.Plan).To(Equal(MustSucceed(
			plan(ctx, impact.PlanOptions{}, folder.Key),
		)))
		Expect(exists(ctx, folder.Key)).To(BeTrue())
	})

	It("Should refuse a pick on a tombstone entry, which takes none", func(
		ctx SpecContext,
	) {
		folder := createFolder(ctx, "impact-picked-tombstone", nil)
		ch := tombstone(ctx, folder)
		record := channel.ResourceID(ch.Key()).String()
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

	It("Should not reach a hidden tombstone through a pick", func(ctx SpecContext) {
		folder := createFolder(ctx, "impact-hidden-pick", nil)
		ch := tombstone(ctx, folder)
		record := channel.ResourceID(ch.Key())
		visible := func(id resource.ID) bool { return id != record }
		picks := map[string]impact.Choice{record.String(): impact.ChoiceDelete}
		Expect(
			plan(ctx, impact.PlanOptions{Picks: picks, Visible: visible}, folder.Key),
		).To(Equal(MustSucceed(
			plan(ctx, impact.PlanOptions{Visible: visible}, folder.Key),
		)))
		MustSucceed(apply(
			ctx,
			impact.ApplyOptions{Picks: picks, Visible: visible},
			folder.Key,
		))
		Expect(exists(ctx, folder.Key)).To(BeFalse())
	})

	It("Should name the pick with the smallest key among the picks it refuses", func(
		ctx SpecContext,
	) {
		folder := createFolder(ctx, "impact-smallest", nil)
		invalid := make(map[string]impact.Choice, 8)
		unparsed := make(map[string]impact.Choice, 8)
		for c := 'a'; c <= 'h'; c++ {
			invalid[fmt.Sprintf("task:%c", c)] = impact.Choice(
				fmt.Sprintf("keep_%c", c),
			)
			unparsed[fmt.Sprintf("%c", c)] = impact.ChoiceDelete
		}
		for range 32 {
			Expect(plan(ctx, impact.PlanOptions{Picks: invalid}, folder.Key)).
				Error().To(MatchError(ContainSubstring("keep_a is not a choice")))
			Expect(plan(ctx, impact.PlanOptions{Picks: unparsed}, folder.Key)).
				Error().To(MatchError(ContainSubstring("a is not a record")))
		}
	})

	It("Should ignore a pick for no entry", func(ctx SpecContext) {
		folder := createFolder(ctx, "impact-ignored", nil)
		picks := map[string]impact.Choice{"task:a": impact.ChoiceDelete}
		Expect(plan(ctx, impact.PlanOptions{Picks: picks}, folder.Key)).
			To(Equal(MustSucceed(plan(ctx, impact.PlanOptions{}, folder.Key))))
		MustSucceed(apply(ctx, impact.ApplyOptions{Picks: picks}, folder.Key))
		Expect(exists(ctx, folder.Key)).To(BeFalse())
	})

	DescribeTable("Should refuse a pick in a plan or an apply", func(
		ctx SpecContext,
		picks map[string]impact.Choice,
		hidden *impact.Choice,
		message string,
	) {
		folder := createFolder(ctx, "impact-refused", nil)
		refused := And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(message)),
		)
		Expect(plan(ctx, impact.PlanOptions{Picks: picks, Hidden: hidden}, folder.Key)).
			Error().To(refused)
		Expect(apply(
			ctx,
			impact.ApplyOptions{Picks: picks, Hidden: hidden},
			folder.Key,
		)).Error().To(refused)
		Expect(exists(ctx, folder.Key)).To(BeTrue())
	},
		Entry(
			"a hidden pick that is not a choice",
			nil,
			new(impact.Choice("keep")),
			"keep is not a choice",
		),
	)
})
