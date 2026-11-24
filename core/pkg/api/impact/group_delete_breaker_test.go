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
	"encoding/json/v2"
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
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/impact"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Group delete routes breaker", Ordered, ContinueOnFailure, func() {
	var (
		layer *service.Layer
		db    *gorp.DB
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
			DB:   db,
			RBAC: layer.RBAC,
			Held: apiauth.HeldTables(),
		}))
		plan = bind.Read(binder, svc.Plan)
		apply = svc.ApplyHandler(binder)
	})

	createFolder := func(ctx SpecContext) group.Group {
		GinkgoHelper()
		g := group.Group{Name: uuid.New().String(), Scope: resource.TypeChannel}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Group.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		return g
	}
	createUser := func(ctx SpecContext) user.User {
		GinkgoHelper()
		return MustSucceed(rbactestutil.CreateUser(
			ctx,
			db,
			layer.RBAC,
			layer.User,
			user.User{Name: uuid.New().String()},
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
	groupsOf := func(keys ...group.Key) impact.Change {
		return impact.Change{Variant: impact.DeleteGroupsChange{Keys: keys}}
	}

	It("Should let a subject with delete and no retrieve apply but not plan", func(
		ctx SpecContext,
	) {
		g := createFolder(ctx)
		u := createUser(ctx)
		grant(ctx, u, []access.Action{access.ActionDelete}, g.ResourceID())
		authed := AuthedCtx(ctx, u)
		Expect(plan(authed, apiimpact.PlanRequest{Change: groupsOf(g.Key)})).
			Error().To(MatchError(access.ErrDenied))
		_, err := apply(authed, apiimpact.ApplyRequest{
			Change: groupsOf(g.Key),
			Token:  "0000000000000000",
		})
		changed, ok := errors.AsType[impact.PlanChangedError](err)
		Expect(ok).To(BeTrue())
		Expect(exists(ctx, g.Key)).To(BeTrue())
		Expect(apply(authed, apiimpact.ApplyRequest{
			Change: groupsOf(g.Key),
			Token:  changed.Plan.Token,
		})).To(HaveField("Token", changed.Plan.Token))
		Expect(exists(ctx, g.Key)).To(BeFalse())
	})

	It("Should refuse a subject with retrieve and no delete on apply", func(
		ctx SpecContext,
	) {
		g := createFolder(ctx)
		u := createUser(ctx)
		grant(ctx, u, []access.Action{access.ActionRetrieve}, g.ResourceID())
		authed := AuthedCtx(ctx, u)
		p := MustSucceed(plan(authed, apiimpact.PlanRequest{Change: groupsOf(g.Key)}))
		Expect(apply(authed, apiimpact.ApplyRequest{
			Change: groupsOf(g.Key),
			Token:  p.Token,
		})).Error().To(And(
			MatchError(access.ErrDenied),
			MatchError(ContainSubstring(g.ResourceID().String())),
		))
		Expect(exists(ctx, g.Key)).To(BeTrue())
	})

	It("Should refuse a plan that names a hidden group twice", func(ctx SpecContext) {
		seen := createFolder(ctx)
		hidden := createFolder(ctx)
		u := createUser(ctx)
		grant(ctx, u, []access.Action{access.ActionRetrieve}, seen.ResourceID())
		Expect(plan(AuthedCtx(ctx, u), apiimpact.PlanRequest{
			Change: groupsOf(seen.Key, hidden.Key, seen.Key, hidden.Key),
		})).Error().To(MatchError(access.ErrDenied))
	})

	It("Should refuse a plan request with no change as a validation error", func(
		ctx SpecContext,
	) {
		g := createFolder(ctx)
		u := createUser(ctx)
		grant(
			ctx,
			u,
			[]access.Action{access.ActionRetrieve, access.ActionDelete},
			g.ResourceID(),
		)
		var req apiimpact.PlanRequest
		Expect(json.Unmarshal([]byte(`{"change":null}`), &req)).To(Succeed())
		Expect(plan(AuthedCtx(ctx, u), req)).
			Error().To(MatchError(validate.ErrValidation))
	})

	It("Should refuse an apply request with no change as a validation error", func(
		ctx SpecContext,
	) {
		g := createFolder(ctx)
		u := createUser(ctx)
		grant(
			ctx,
			u,
			[]access.Action{access.ActionRetrieve, access.ActionDelete},
			g.ResourceID(),
		)
		var req apiimpact.ApplyRequest
		Expect(json.Unmarshal([]byte(`{"change":null}`), &req)).To(Succeed())
		Expect(apply(AuthedCtx(ctx, u), req)).
			Error().To(MatchError(validate.ErrValidation))
		Expect(exists(ctx, g.Key)).To(BeTrue())
	})
})
