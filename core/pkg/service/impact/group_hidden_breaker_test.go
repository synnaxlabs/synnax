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
	"slices"

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
)

var _ = Describe("Group delete hidden breaker", Ordered, ContinueOnFailure, func() {
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
		proj = project.Project{Name: "impact-ungroup-breaker"}
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
	// deletes it, which leaves its tombstone.
	tombstone := func(ctx SpecContext, name string, folder *group.Group) channel.Channel {
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
		return ch
	}
	rename := func(ctx SpecContext, g group.Group) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Group.NewWriter(tx).Rename(ctx, g.Key, "renamed-"+g.Name)
		})).To(Succeed())
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
	// hide returns a Visible that hides each of hidden.
	hide := func(hidden ...resource.ID) func(resource.ID) bool {
		return func(id resource.ID) bool { return !slices.Contains(hidden, id) }
	}
	changed := MatchError(ContainSubstring(
		"the records that the change touches changed",
	))

	It("Should count a tombstone of a deleted group that the caller cannot see", func(
		ctx SpecContext,
	) {
		folder := createFolder(ctx, nil)
		dead := tombstone(ctx, UniqueChannelName(), &folder)
		p := MustSucceed(layer.Impact.Plan(
			ctx,
			db,
			deleteOf(folder.Key),
			impact.PlanOptions{Visible: hide(channel.ResourceID(dead.Key()))},
		))
		Expect(p.Entries).To(BeEmpty())
		Expect(p.Hidden).To(Equal(impact.Hidden{Cascade: 1}))
	})

	It("Should refuse a group delete token planned before the folder was renamed", func(
		ctx SpecContext,
	) {
		folder := createFolder(ctx, nil)
		dead := tombstone(ctx, UniqueChannelName(), &folder)
		stale := MustSucceed(layer.Impact.Plan(
			ctx, db, deleteOf(folder.Key), impact.PlanOptions{},
		))
		Expect(*stale.Entries[0].Path).To(Equal(folder.Name + "." + dead.Name))
		rename(ctx, folder)
		applied, err := layer.Impact.Apply(
			ctx, db, deleteOf(folder.Key), impact.ApplyOptions{Token: stale.Token},
		)
		Expect(err).To(changed)
		Expect(applied).To(BeZero())
		Expect(exists(ctx, folder.Key)).To(BeTrue())
	})
})
