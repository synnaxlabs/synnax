// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Resources", func() {
	var (
		db  *gorp.DB
		svc *group.Service
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		svc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB: db,
		}))
	})

	It("Should name each group by its name", func(ctx SpecContext) {
		g := group.Group{Name: "Sensors", Scope: resource.TypeChannel}
		Expect(svc.NewWriter(nil).Create(ctx, &g)).To(Succeed())
		Expect(svc.Resources().Retrieve(ctx, db, []resource.ID{g.ResourceID()})).
			To(Equal([]resource.Resource{{ID: g.ResourceID(), Name: g.Name}}))
	})

	It("Should return a validation error for a malformed key", func(ctx SpecContext) {
		Expect(svc.Resources().Retrieve(
			ctx,
			nil,
			[]resource.ID{{Type: resource.TypeGroup, Key: "pump"}},
		)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`invalid group key "pump"`)),
		))
	})

	It("Should leave out an ID that names no group", func(ctx SpecContext) {
		g := group.Group{Name: "Sensors", Scope: resource.TypeChannel}
		Expect(svc.NewWriter(nil).Create(ctx, &g)).To(Succeed())
		missing := resource.ID{Type: resource.TypeGroup, Key: uuid.New().String()}
		Expect(svc.Resources().Retrieve(
			ctx,
			db,
			[]resource.ID{missing, g.ResourceID()},
		)).To(Equal([]resource.Resource{{ID: g.ResourceID(), Name: g.Name}}))
	})

	It("Should list each group by its name", func(ctx SpecContext) {
		g := group.Group{Name: "Sensors", Scope: resource.TypeChannel}
		Expect(svc.NewWriter(nil).Create(ctx, &g)).To(Succeed())
		Expect(svc.Resources().List(ctx, db, 10, nil)).
			To(Equal([]resource.Resource{{ID: g.ResourceID(), Name: g.Name}}))
	})
})

var _ = Describe("ResourceIDsFromGroups", func() {
	It("Should map a slice of groups to resource IDs", func() {
		a := group.Group{Key: uuid.New(), Name: "a"}
		b := group.Group{Key: uuid.New(), Name: "b"}
		Expect(group.ResourceIDsFromGroups([]group.Group{a, b})).
			To(ConsistOf(a.ResourceID(), b.ResourceID()))
	})
})

var _ = Describe("KeyFromResourceID", func() {
	It("Should return the key of a group ID", func() {
		key := uuid.New()
		Expect(group.KeyFromResourceID(group.ResourceID(key))).To(Equal(key))
	})
	DescribeTable("Should reject an ID that is not a group",
		func(id resource.ID, msg string) {
			Expect(group.KeyFromResourceID(id)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(msg)),
			))
		},
		Entry("a project",
			resource.ID{Type: resource.TypeProject, Key: uuid.New().String()},
			`must be a group, got "project"`),
		Entry("a zero ID", resource.ID{}, `must be a group, got ""`),
		Entry("a group whose key is not a UUID",
			resource.ID{Type: resource.TypeGroup, Key: "not-a-uuid"},
			`invalid group key "not-a-uuid"`),
		Entry("a group with an empty key",
			resource.ID{Type: resource.TypeGroup},
			`invalid group key ""`),
	)
})
