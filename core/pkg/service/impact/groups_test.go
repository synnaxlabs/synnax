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
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
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
	apply := func(
		ctx SpecContext,
		opts impact.ApplyOptions,
		keys ...group.Key,
	) (impact.Plan, error) {
		return layer.Impact.Apply(
			ctx,
			db,
			impact.Change{Variant: impact.DeleteGroupsChange{Keys: keys}},
			opts,
		)
	}

	It("Should delete a folder and a folder inside it", func(ctx SpecContext) {
		parent := createFolder(ctx, "impact-parent", nil)
		child := createFolder(ctx, "impact-child", &parent)
		Expect(apply(ctx, impact.ApplyOptions{}, parent.Key, child.Key)).
			To(Equal(impact.Plan{}))
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

	It("Should refuse a pick, since a group delete has no entries", func(
		ctx SpecContext,
	) {
		folder := createFolder(ctx, "impact-picked", nil)
		Expect(apply(ctx, impact.ApplyOptions{
			Picks: map[string]impact.Choice{
				"task:b": impact.ChoiceDelete,
				"task:a": impact.ChoiceCancel,
			},
		}, folder.Key)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("no entry is task:a")),
		))
		Expect(exists(ctx, folder.Key)).To(BeTrue())
	})

	It("Should fail on a change of no known variant", func(ctx SpecContext) {
		Expect(layer.Impact.Apply(ctx, db, impact.Change{}, impact.ApplyOptions{})).
			Error().To(MatchError("[impact] - unknown change"))
	})
})
