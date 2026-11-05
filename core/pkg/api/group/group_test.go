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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apigroup "github.com/synnaxlabs/synnax/pkg/api/group"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	create = []access.Action{access.ActionCreate}
	update = []access.Action{access.ActionUpdate}
	groups = resource.ID{Type: resource.TypeGroup}
)

func newGroup(ctx SpecContext, scope resource.Type, parent *group.Group) group.Group {
	GinkgoHelper()
	g := group.Group{Name: "group", Scope: scope}
	if parent != nil {
		g.Parent = new(parent.ResourceID())
	}
	Expect(svc.Group.NewWriter(nil).Create(ctx, &g)).To(Succeed())
	return g
}

func keys(gs []group.Group) []group.Key {
	out := make([]group.Key, len(gs))
	for i, g := range gs {
		out[i] = g.Key
	}
	return out
}

var _ = Describe("Group", func() {
	Describe("Create", func() {
		It("Should need update access on the parent", func(ctx SpecContext) {
			parent := newGroup(ctx, resource.TypeChannel, nil)
			u := newUser(ctx, grant(create, groups))
			Expect(apiSvc.Create(AuthedCtx(ctx, u), nil, apigroup.CreateRequest{
				Name:   "child",
				Parent: new(parent.ResourceID()),
			})).Error().To(MatchError(access.ErrDenied))
		})

		It(
			"Should create a group under a parent the user can update",
			func(ctx SpecContext) {
				parent := newGroup(ctx, resource.TypeChannel, nil)
				u := newUser(
					ctx,
					grant(create, groups),
					grant(update, parent.ResourceID()),
				)
				res := MustSucceed(apiSvc.Create(
					AuthedCtx(ctx, u),
					nil,
					apigroup.CreateRequest{
						Name:   "child",
						Parent: new(parent.ResourceID()),
					},
				))
				Expect(res.Group.Parent).To(Equal(new(parent.ResourceID())))
				Expect(res.Group.Scope).To(Equal(resource.TypeChannel))
			},
		)
	})

	Describe("SetParent", func() {
		var source, dest, moved group.Group
		BeforeEach(func(ctx SpecContext) {
			source = newGroup(ctx, resource.TypeChannel, nil)
			dest = newGroup(ctx, resource.TypeChannel, nil)
			moved = newGroup(ctx, resource.TypeChannel, &source)
		})
		setParent := func(ctx SpecContext, objects ...resource.ID) error {
			u := newUser(ctx, grant(update, objects...))
			_, err := apiSvc.SetParent(
				AuthedCtx(ctx, u),
				nil,
				apigroup.SetParentRequest{
					Keys:   []group.Key{moved.Key},
					Parent: new(dest.ResourceID()),
				},
			)
			return err
		}

		It("Should need update access on the destination", func(ctx SpecContext) {
			Expect(setParent(ctx, moved.ResourceID(), source.ResourceID())).
				To(MatchError(access.ErrDenied))
		})

		It("Should need update access on the source parent", func(ctx SpecContext) {
			Expect(setParent(ctx, moved.ResourceID(), dest.ResourceID())).
				To(MatchError(access.ErrDenied))
		})

		It(
			"Should move a group when the user can update the group and both parents",
			func(ctx SpecContext) {
				Expect(setParent(
					ctx,
					moved.ResourceID(),
					source.ResourceID(),
					dest.ResourceID(),
				)).To(Succeed())
				var g group.Group
				Expect(svc.Group.NewRetrieve().
					Where(group.MatchKeys(moved.Key)).
					Entry(&g).
					Exec(ctx, nil)).To(Succeed())
				Expect(g.Parent).To(Equal(new(dest.ResourceID())))
			},
		)
	})

	Describe("Retrieve", func() {
		var top, nested, devices group.Group
		BeforeEach(func(ctx SpecContext) {
			top = newGroup(ctx, resource.TypeChannel, nil)
			nested = newGroup(ctx, resource.TypeChannel, &top)
			devices = newGroup(ctx, resource.TypeDevice, nil)
		})
		retrieve := func(
			ctx SpecContext,
			req apigroup.RetrieveRequest,
		) apigroup.RetrieveResponse {
			GinkgoHelper()
			u := newUser(ctx, grant([]access.Action{access.ActionRetrieve}, groups))
			return MustSucceed(apiSvc.Retrieve(AuthedCtx(ctx, u), req))
		}
		all := func() []group.Key { return []group.Key{top.Key, nested.Key, devices.Key} }

		It(
			"Should keep only the groups under the given parents",
			func(ctx SpecContext) {
				res := retrieve(ctx, apigroup.RetrieveRequest{
					Parents: []resource.ID{top.ResourceID()},
				})
				Expect(keys(res.Groups)).To(Equal([]group.Key{nested.Key}))
			},
		)

		It("Should keep only the groups with no parent", func(ctx SpecContext) {
			res := retrieve(ctx, apigroup.RetrieveRequest{
				Keys:   all(),
				Nested: new(false),
			})
			Expect(keys(res.Groups)).To(ConsistOf(top.Key, devices.Key))
		})

		It("Should keep only the groups of the given scopes", func(ctx SpecContext) {
			res := retrieve(ctx, apigroup.RetrieveRequest{
				Keys:   all(),
				Scopes: []resource.Type{resource.TypeDevice},
			})
			Expect(keys(res.Groups)).To(Equal([]group.Key{devices.Key}))
		})

		It("Should add the groups above the returned groups", func(ctx SpecContext) {
			res := retrieve(ctx, apigroup.RetrieveRequest{
				Keys:      []group.Key{nested.Key},
				Ancestors: true,
			})
			Expect(keys(res.Groups)).To(Equal([]group.Key{nested.Key}))
			Expect(keys(res.Ancestors)).To(Equal([]group.Key{top.Key}))
		})
	})
})
