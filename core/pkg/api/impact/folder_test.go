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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apiimpact "github.com/synnaxlabs/synnax/pkg/api/impact"
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
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Folder deletes", Ordered, func() {
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
			Tree:   layer.Tree,
		}))
		binder := MustSucceed(bind.NewService(bind.ServiceConfig{
			DB:      db,
			RBAC:    layer.RBAC,
			Held:    apiauth.HeldTables(),
			Mirrors: channel.MirrorTables(),
		}))
		plan = folderPlanOf(bind.Read(binder, svc.Plan))
		apply = folderPlanOf(svc.ApplyHandler(binder))
		proj = project.Project{Name: "folders"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Project.NewWriter(tx).Create(ctx, &proj)
		})).To(Succeed())
	})

	createFolder := func(ctx SpecContext) group.Group {
		GinkgoHelper()
		g := group.Group{Name: UniqueChannelName(), Scope: resource.TypeChannel}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Group.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		return g
	}
	// deleteShown creates a channel in folder g, shows it in a schematic, and deletes
	// it, so a tombstone keeps its path.
	deleteShown := func(ctx SpecContext, g group.Group) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
			Group:    &g.Key,
		}
		var value schematic.ValueNodeConfig
		value.SetDefaults()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := layer.Channel.NewWriter(tx).Create(ctx, &ch); err != nil {
				return err
			}
			value.Channel = ch.Key()
			if err := layer.Schematic.NewWriter(tx).Create(ctx, &schematic.Schematic{
				Name:    "pid",
				Project: &proj.Key,
				Nodes: []schematic.Node{{
					Key:    "value",
					Config: schematic.NodeConfig{Variant: value},
				}},
			}); err != nil {
				return err
			}
			return layer.Channel.NewWriter(tx).Delete(ctx, ch.Key(), false)
		})).To(Succeed())
		return ch
	}
	createUser := func(
		ctx SpecContext,
		actions []access.Action,
		objects ...resource.ID,
	) user.User {
		GinkgoHelper()
		u := MustSucceed(rbactestutil.CreateUser(
			ctx,
			db,
			layer.RBAC,
			layer.User,
			user.User{Name: uuid.New().String()},
		))
		MustSucceed(rbactestutil.Grant(
			ctx,
			db,
			layer.RBAC,
			layer.User,
			u.Key,
			actions,
			objects...,
		))
		return u
	}
	deleteGroups := func(keys ...group.Key) apiimpact.Change {
		return apiimpact.Change{Variant: apiimpact.DeleteGroupsChange{Groups: keys}}
	}
	folderExists := func(ctx SpecContext, k group.Key) bool {
		GinkgoHelper()
		return MustSucceed(layer.Group.NewRetrieve().
			Where(group.MatchKeys(k)).
			Exists(ctx, db))
	}

	It("Should plan and apply the delete of a folder and its tombstones", func(
		ctx SpecContext,
	) {
		g := createFolder(ctx)
		dead := deleteShown(ctx, g)
		u := createUser(
			ctx,
			[]access.Action{access.ActionRetrieve, access.ActionDelete},
			g.ResourceID(),
			channel.ResourceID(dead.Key()),
		)
		planned := MustSucceed(plan(
			AuthedCtx(ctx, u),
			apiimpact.PlanRequest{Change: deleteGroups(g.Key)},
		))
		Expect(planned.Entries).To(Equal([]impact.Entry{{
			Record: channel.ResourceID(dead.Key()),
			Effect: impact.EffectDelete,
			Path:   new(g.Name + "." + dead.Name),
		}}))
		applied := MustSucceed(apply(AuthedCtx(ctx, u), apiimpact.ApplyRequest{
			Change: deleteGroups(g.Key),
			Token:  planned.Token,
		}))
		Expect(applied).To(Equal(planned))
		Expect(folderExists(ctx, g.Key)).To(BeFalse())
	})

	It("Should refuse a folder that the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		g := createFolder(ctx)
		u := createUser(ctx, []access.Action{access.ActionRetrieve})
		Expect(plan(
			AuthedCtx(ctx, u),
			apiimpact.PlanRequest{Change: deleteGroups(g.Key)},
		)).Error().To(MatchError(access.ErrDenied))
		Expect(apply(
			AuthedCtx(ctx, u),
			apiimpact.ApplyRequest{Change: deleteGroups(g.Key)},
		)).Error().To(MatchError(access.ErrDenied))
		Expect(folderExists(ctx, g.Key)).To(BeTrue())
	})

	DescribeTable(
		"Should refuse a folder delete that holds a pick",
		func(
			ctx SpecContext,
			picks map[string]impact.Effect,
			hidden *impact.Effect,
		) {
			g := createFolder(ctx)
			u := createUser(
				ctx,
				[]access.Action{access.ActionRetrieve, access.ActionDelete},
				g.ResourceID(),
			)
			Expect(plan(AuthedCtx(ctx, u), apiimpact.PlanRequest{
				Change: deleteGroups(g.Key),
				Picks:  picks,
				Hidden: hidden,
			})).Error().To(MatchError(
				ContainSubstring("a folder delete takes no picks"),
			))
			Expect(apply(AuthedCtx(ctx, u), apiimpact.ApplyRequest{
				Change: deleteGroups(g.Key),
				Picks:  picks,
				Hidden: hidden,
			})).Error().To(MatchError(validate.ErrValidation))
			Expect(folderExists(ctx, g.Key)).To(BeTrue())
		},
		Entry(
			"a pick",
			map[string]impact.Effect{"channel:1": impact.EffectDelete},
			nil,
		),
		Entry("a hidden pick", nil, new(impact.EffectCancel)),
	)
})
