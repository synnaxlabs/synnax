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
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apiimpact "github.com/synnaxlabs/synnax/pkg/api/impact"
	apipanel "github.com/synnaxlabs/synnax/pkg/api/panel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/impact"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Ungroup routes breaker", Ordered, ContinueOnFailure, func() {
	var (
		layer *service.Layer
		db    *gorp.DB
		proj  project.Project
		plan  freighter.UnaryHandler[apiimpact.PlanRequest, impact.Plan]
		apply freighter.UnaryHandler[apiimpact.ApplyRequest, impact.Plan]
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		layer = MustOpen(svctestutil.OpenLayer(ctx, node))
		svc := MustSucceed(apiimpact.NewService(apiimpact.ServiceConfig{
			RBAC:   layer.RBAC,
			Impact: layer.Impact,
		}))
		binder := MustSucceed(bind.NewService(bind.ServiceConfig{
			DB:          db,
			RBAC:        layer.RBAC,
			Held:        apiauth.HeldTables(),
			Owned:       apipanel.OwnedTables(),
			OwnedPlaces: apipanel.OwnedPlaces(),
			Mirrors:     channel.MirrorTables(),
		}))
		plan = bind.Read(binder, svc.Plan)
		apply = svc.ApplyHandler(binder)
		proj = project.Project{Name: "impact-ungroup-routes"}
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
	createUser := func(ctx SpecContext) user.User {
		GinkgoHelper()
		return MustSucceed(rbactestutil.CreateUser(
			ctx,
			db,
			layer.RBAC,
			layer.User,
			user.User{Name: UniqueChannelName()},
		))
	}
	grant := func(
		ctx SpecContext,
		u user.User,
		actions []access.Action,
		objects ...resource.ID,
	) {
		GinkgoHelper()
		MustSucceed(rbactestutil.Grant(
			ctx,
			db,
			layer.RBAC,
			layer.User,
			u.Key,
			actions,
			objects...,
		))
	}
	exists := func(ctx SpecContext, k group.Key) bool {
		GinkgoHelper()
		return MustSucceed(layer.Group.NewRetrieve().
			Where(group.MatchKeys(k)).
			Exists(ctx, db))
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
	ungroupOf := func(keys ...group.Key) impact.Change {
		return impact.Change{Variant: impact.UngroupGroupsChange{Keys: keys}}
	}
	retrieve := []access.Action{access.ActionRetrieve}

	It("Should hide a dropped tombstone of a channel that the subject cannot see", func(
		ctx SpecContext,
	) {
		parent := createFolder(ctx, nil)
		folder := createFolder(ctx, &parent)
		name := UniqueChannelName()
		tombstone(ctx, name, &folder)
		createChannel(ctx, name, &parent)
		u := createUser(ctx)
		grant(ctx, u, retrieve, folder.ResourceID())
		p := MustSucceed(plan(AuthedCtx(ctx, u), apiimpact.PlanRequest{
			Change: ungroupOf(folder.Key),
		}))
		Expect(p.Entries).To(BeEmpty())
		Expect(p.Hidden).To(Equal(impact.Hidden{Cascade: 1}))
	})

	It("Should refuse an ungroup into a parent the subject cannot update", func(
		ctx SpecContext,
	) {
		parent := createFolder(ctx, nil)
		folder := createFolder(ctx, &parent)
		tombstone(ctx, UniqueChannelName(), &folder)
		u := createUser(ctx)
		grant(
			ctx,
			u,
			[]access.Action{access.ActionRetrieve, access.ActionDelete},
			folder.ResourceID(),
		)
		authed := AuthedCtx(ctx, u)
		p := MustSucceed(plan(authed, apiimpact.PlanRequest{
			Change: ungroupOf(folder.Key),
		}))
		Expect(apply(authed, apiimpact.ApplyRequest{
			Change: ungroupOf(folder.Key),
			Token:  p.Token,
		})).Error().To(MatchError(access.ErrDenied))
		Expect(exists(ctx, folder.Key)).To(BeTrue())
	})

	It("Should refuse an ungroup that moves a channel the subject cannot update", func(
		ctx SpecContext,
	) {
		parent := createFolder(ctx, nil)
		folder := createFolder(ctx, &parent)
		live := createChannel(ctx, UniqueChannelName(), &folder)
		u := createUser(ctx)
		grant(
			ctx,
			u,
			[]access.Action{access.ActionRetrieve, access.ActionDelete},
			folder.ResourceID(),
		)
		grant(
			ctx,
			u,
			[]access.Action{access.ActionRetrieve, access.ActionUpdate},
			parent.ResourceID(),
		)
		authed := AuthedCtx(ctx, u)
		p := MustSucceed(plan(authed, apiimpact.PlanRequest{
			Change: ungroupOf(folder.Key),
		}))
		Expect(apply(authed, apiimpact.ApplyRequest{
			Change: ungroupOf(folder.Key),
			Token:  p.Token,
		})).Error().To(MatchError(access.ErrDenied))
		Expect(exists(ctx, folder.Key)).To(BeTrue())
		Expect(groupOf(ctx, live.Key())).To(Equal(&folder.Key))
	})

	It("Should refuse to plan an ungroup of a folder the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		seen := createFolder(ctx, nil)
		hidden := createFolder(ctx, nil)
		u := createUser(ctx)
		grant(ctx, u, retrieve, seen.ResourceID())
		Expect(plan(AuthedCtx(ctx, u), apiimpact.PlanRequest{
			Change: ungroupOf(seen.Key, hidden.Key),
		})).Error().To(MatchError(access.ErrDenied))
	})

	It("Should hide a dropped tombstone in the plan that an ungroup applies", func(
		ctx SpecContext,
	) {
		parent := createFolder(ctx, nil)
		folder := createFolder(ctx, &parent)
		name := UniqueChannelName()
		tombstone(ctx, name, &folder)
		createChannel(ctx, name, &parent)
		u := createUser(ctx)
		grant(
			ctx,
			u,
			[]access.Action{access.ActionRetrieve, access.ActionDelete},
			folder.ResourceID(),
		)
		grant(
			ctx,
			u,
			[]access.Action{access.ActionRetrieve, access.ActionUpdate},
			parent.ResourceID(),
		)
		authed := AuthedCtx(ctx, u)
		p := MustSucceed(plan(authed, apiimpact.PlanRequest{
			Change: ungroupOf(folder.Key),
		}))
		Expect(p.Entries).To(BeEmpty())
		Expect(p.Hidden).To(Equal(impact.Hidden{Cascade: 1}))
		Expect(apply(authed, apiimpact.ApplyRequest{
			Change: ungroupOf(folder.Key),
			Token:  p.Token,
		})).To(Equal(p))
		Expect(exists(ctx, folder.Key)).To(BeFalse())
	})
})
