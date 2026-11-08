// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apitable "github.com/synnaxlabs/synnax/pkg/api/table"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Retrieve", func() {
	It("Should reject the request when retrieve is not granted on a match", func(
		ctx SpecContext,
	) {
		r := table.Table{Name: "unretrievable"}
		Expect(svc.Table.NewWriter(nil).Create(ctx, uuid.Nil(), &r)).To(Succeed())
		Expect(apiSvc.Retrieve(AuthedCtx(ctx, author), apitable.RetrieveRequest{
			Where: &table.Where{Key: &filter.Set[table.Key]{Eq: &r.Key}},
		})).Error().To(MatchError(access.ErrDenied))
	})
})

var _ = Describe("Create", func() {
	It("Should stamp the dispatch key onto the create broadcast", func(
		ctx SpecContext,
	) {
		u := newCreator(ctx)
		rec := &Recorder[table.Key, table.Action]{}
		DeferCleanup(svc.Table.OnAction(rec.Record))
		res := MustSucceed(apiSvc.Create(AuthedCtx(ctx, u), nil, apitable.CreateRequest{
			Project:     proj.Key,
			Tables:      []table.Table{{Name: "broadcast"}},
			DispatchKey: "dk-create",
		}))
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(1))
		Expect(frames[0].Key).To(Equal(res.Tables[0].Key))
		Expect(frames[0].DispatchKey).To(Equal("dk-create"))
	})
})

// createProjectGroup creates a project group with no top parent.
func createProjectGroup(ctx SpecContext, name string) group.Group {
	GinkgoHelper()
	g := group.Group{Name: name, Scope: resource.TypeProject}
	Expect(svc.Group.NewWriter(nil).Create(ctx, &g)).To(Succeed())
	return g
}

func createTable(ctx SpecContext, name string) table.Table {
	GinkgoHelper()
	t := table.Table{Name: name}
	Expect(svc.Table.NewWriter(nil).Create(ctx, proj.Key, &t)).To(Succeed())
	return t
}

func retrieveTable(ctx SpecContext, key table.Key) table.Table {
	GinkgoHelper()
	var t table.Table
	Expect(svc.Table.NewRetrieve().
		Where(table.MatchKeys(key)).
		Entry(&t).
		Exec(ctx, nil)).To(Succeed())
	return t
}

var _ = Describe("SetGroup", func() {
	It(
		"Should refuse the move without update on the destination group",
		func(ctx SpecContext) {
			u := newCreator(ctx)
			t := createTable(ctx, "Set group denied")
			g := createProjectGroup(ctx, "Set group denied")
			grant(ctx, access.ActionUpdate, u.ResourceID(), table.ResourceID(t.Key))
			Expect(apiSvc.SetGroup(
				AuthedCtx(ctx, u), nil,
				apitable.SetGroupRequest{Group: &g.Key, Keys: []table.Key{t.Key}},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(retrieveTable(ctx, t.Key).Group).To(BeNil())
		},
	)
	It(
		"Should move the table with update on the destination group",
		func(ctx SpecContext) {
			u := newCreator(ctx)
			t := createTable(ctx, "Set group granted")
			g := createProjectGroup(ctx, "Set group granted")
			grant(
				ctx,
				access.ActionUpdate,
				u.ResourceID(),
				table.ResourceID(t.Key),
				g.ResourceID(),
			)
			Expect(apiSvc.SetGroup(
				AuthedCtx(ctx, u), nil,
				apitable.SetGroupRequest{Group: &g.Key, Keys: []table.Key{t.Key}},
			)).Error().ToNot(HaveOccurred())
			Expect(retrieveTable(ctx, t.Key).Group).To(Equal(&g.Key))
		},
	)
})

var _ = Describe("Create in a group", func() {
	It(
		"Should refuse the create without update on the group",
		func(ctx SpecContext) {
			u := newCreator(ctx)
			g := createProjectGroup(ctx, "Create denied")
			Expect(apiSvc.Create(AuthedCtx(ctx, u), nil, apitable.CreateRequest{
				Project: proj.Key,
				Tables:  []table.Table{{Name: "Denied", Group: &g.Key}},
			})).Error().To(MatchError(access.ErrDenied))
		},
	)
	It("Should create the table with update on the group", func(ctx SpecContext) {
		u := newCreator(ctx)
		g := createProjectGroup(ctx, "Create granted")
		grant(ctx, access.ActionUpdate, u.ResourceID(), g.ResourceID())
		res := MustSucceed(apiSvc.Create(AuthedCtx(ctx, u), nil, apitable.CreateRequest{
			Project: proj.Key,
			Tables:  []table.Table{{Name: "Granted", Group: &g.Key}},
		}))
		Expect(retrieveTable(ctx, res.Tables[0].Key).Group).To(Equal(&g.Key))
	})
})
