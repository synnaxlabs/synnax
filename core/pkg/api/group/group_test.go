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
	"github.com/onsi/gomega/types"
	"github.com/samber/lo"
	apigroup "github.com/synnaxlabs/synnax/pkg/api/group"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	createAction = []access.Action{access.ActionCreate}
	update       = []access.Action{access.ActionUpdate}
	remove       = []access.Action{access.ActionDelete}
	groups       = resource.ID{Type: resource.TypeGroup}
)

// denied matches an error that denies u the action on object.
func denied(u user.User, action access.Action, object resource.ID) types.GomegaMatcher {
	return SatisfyAll(
		MatchError(access.ErrDenied),
		MatchError(ContainSubstring(
			"%s cannot %s %s",
			u.ResourceID(),
			action,
			object,
		)),
	)
}

func retrieve(ctx SpecContext, key group.Key) group.Group {
	GinkgoHelper()
	var g group.Group
	Expect(svc.Group.NewRetrieve().
		Where(group.MatchKeys(key)).
		Entry(&g).
		Exec(ctx, nil)).To(Succeed())
	return g
}

func exists(ctx SpecContext, key group.Key) bool {
	GinkgoHelper()
	return MustSucceed(svc.Group.NewRetrieve().
		Where(group.MatchKeys(key)).
		Exists(ctx, nil))
}

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
		It("Should need create access on the group", func(ctx SpecContext) {
			u := newUser(ctx)
			key := uuid.New()
			Expect(create(AuthedCtx(ctx, u), apigroup.CreateRequest{
				Key:   key,
				Name:  "denied",
				Scope: resource.TypeChannel,
			})).Error().To(denied(u, access.ActionCreate, group.ResourceID(key)))
			Expect(exists(ctx, key)).To(BeFalse())
		})

		It("Should need update access on the parent", func(ctx SpecContext) {
			parent := newGroup(ctx, resource.TypeChannel, nil)
			u := newUser(ctx, grant(createAction, groups))
			Expect(create(AuthedCtx(ctx, u), apigroup.CreateRequest{
				Name:   "child",
				Parent: new(parent.ResourceID()),
			})).Error().To(denied(u, access.ActionUpdate, parent.ResourceID()))
		})

		It(
			"Should create a group under a parent the user can update",
			func(ctx SpecContext) {
				parent := newGroup(ctx, resource.TypeChannel, nil)
				u := newUser(
					ctx,
					grant(createAction, groups),
					grant(update, parent.ResourceID()),
				)
				res := MustSucceed(create(
					AuthedCtx(ctx, u),
					apigroup.CreateRequest{
						Name:   "child",
						Parent: new(parent.ResourceID()),
					},
				))
				Expect(res.Group.Parent).To(Equal(new(parent.ResourceID())))
				Expect(res.Group.Scope).To(Equal(resource.TypeChannel))
			},
		)

		It(
			"Should need update access to overwrite an existing group",
			func(ctx SpecContext) {
				g := newGroup(ctx, resource.TypeChannel, nil)
				u := newUser(ctx, grant(createAction, groups))
				Expect(create(
					AuthedCtx(ctx, u),
					apigroup.CreateRequest{Key: g.Key, Name: "renamed"},
				)).Error().To(denied(u, access.ActionUpdate, g.ResourceID()))
			},
		)

		It(
			"Should keep the parent of an existing group that it overwrites",
			func(ctx SpecContext) {
				parent := newGroup(ctx, resource.TypeChannel, nil)
				g := newGroup(ctx, resource.TypeChannel, &parent)
				u := newUser(ctx, grant(update, g.ResourceID()))
				res := MustSucceed(create(
					AuthedCtx(ctx, u),
					apigroup.CreateRequest{Key: g.Key, Name: g.Name},
				))
				Expect(res.Group.Parent).To(Equal(new(parent.ResourceID())))
				Expect(retrieve(ctx, g.Key).Parent).
					To(Equal(new(parent.ResourceID())))
			},
		)
	})

	Describe("Delete", func() {
		var parent, g group.Group
		BeforeEach(func(ctx SpecContext) {
			parent = newGroup(ctx, resource.TypeChannel, nil)
			g = newGroup(ctx, resource.TypeChannel, &parent)
		})
		deleteAs := func(
			ctx SpecContext,
			u user.User,
			keys ...group.Key,
		) (struct{}, error) {
			return del(AuthedCtx(ctx, u), apigroup.DeleteRequest{Keys: keys})
		}

		It("Should need delete access on the group", func(ctx SpecContext) {
			u := newUser(ctx, grant(update, parent.ResourceID()))
			Expect(deleteAs(ctx, u, g.Key)).
				Error().To(denied(u, access.ActionDelete, g.ResourceID()))
			Expect(exists(ctx, g.Key)).To(BeTrue())
		})

		It("Should need update access on the parent", func(ctx SpecContext) {
			u := newUser(ctx, grant(remove, g.ResourceID()))
			Expect(deleteAs(ctx, u, g.Key)).
				Error().To(denied(u, access.ActionUpdate, parent.ResourceID()))
			Expect(exists(ctx, g.Key)).To(BeTrue())
		})

		It(
			"Should delete a group with delete on it and update on its parent",
			func(ctx SpecContext) {
				u := newUser(
					ctx,
					grant(remove, g.ResourceID()),
					grant(update, parent.ResourceID()),
				)
				MustSucceed(deleteAs(ctx, u, g.Key))
				Expect(exists(ctx, g.Key)).To(BeFalse())
			},
		)

		It(
			"Should need delete access on a nested group that it deletes",
			func(ctx SpecContext) {
				nested := newGroup(ctx, resource.TypeChannel, &g)
				u := newUser(
					ctx,
					grant(remove, g.ResourceID()),
					grant(update, parent.ResourceID(), g.ResourceID()),
				)
				Expect(deleteAs(ctx, u, g.Key, nested.Key)).
					Error().To(denied(u, access.ActionDelete, nested.ResourceID()))
				Expect(exists(ctx, nested.Key)).To(BeTrue())
			},
		)
	})

	Describe("Rename", func() {
		var g group.Group
		BeforeEach(func(ctx SpecContext) {
			g = newGroup(ctx, resource.TypeChannel, nil)
		})
		renameAs := func(ctx SpecContext, u user.User) (struct{}, error) {
			return rename(
				AuthedCtx(ctx, u),
				apigroup.RenameRequest{Key: g.Key, Name: "renamed"},
			)
		}

		It("Should need update access on the group", func(ctx SpecContext) {
			u := newUser(ctx)
			Expect(renameAs(ctx, u)).
				Error().To(denied(u, access.ActionUpdate, g.ResourceID()))
			Expect(retrieve(ctx, g.Key).Name).To(Equal("group"))
		})

		It("Should rename a group the user can update", func(ctx SpecContext) {
			u := newUser(ctx, grant(update, g.ResourceID()))
			MustSucceed(renameAs(ctx, u))
			Expect(retrieve(ctx, g.Key).Name).To(Equal("renamed"))
		})
	})

	Describe("SetParent", func() {
		var source, dest, moved group.Group
		BeforeEach(func(ctx SpecContext) {
			source = newGroup(ctx, resource.TypeChannel, nil)
			dest = newGroup(ctx, resource.TypeChannel, nil)
			moved = newGroup(ctx, resource.TypeChannel, &source)
		})
		setParentAs := func(ctx SpecContext, u user.User) (struct{}, error) {
			return setParent(
				AuthedCtx(ctx, u),
				apigroup.SetParentRequest{
					Keys:   []group.Key{moved.Key},
					Parent: new(dest.ResourceID()),
				},
			)
		}

		It("Should need update access on the destination", func(ctx SpecContext) {
			u := newUser(ctx, grant(update, moved.ResourceID(), source.ResourceID()))
			Expect(setParentAs(ctx, u)).
				Error().To(denied(u, access.ActionUpdate, dest.ResourceID()))
		})

		It("Should need update access on the source parent", func(ctx SpecContext) {
			u := newUser(ctx, grant(update, moved.ResourceID(), dest.ResourceID()))
			Expect(setParentAs(ctx, u)).
				Error().To(denied(u, access.ActionUpdate, source.ResourceID()))
		})

		It(
			"Should move a group when the user can update the group and both parents",
			func(ctx SpecContext) {
				u := newUser(ctx, grant(
					update,
					moved.ResourceID(),
					source.ResourceID(),
					dest.ResourceID(),
				))
				MustSucceed(setParentAs(ctx, u))
				Expect(retrieve(ctx, moved.Key).Parent).
					To(Equal(new(dest.ResourceID())))
			},
		)
	})

	Describe("SetParent to another project", func() {
		var (
			from, to             project.Project
			source, dest, nested group.Group
			sc                   schematic.Schematic
		)
		BeforeEach(func(ctx SpecContext) {
			from, to = project.Project{Name: "From"}, project.Project{Name: "To"}
			Expect(svc.Project.NewWriter(nil).Create(ctx, &from)).To(Succeed())
			Expect(svc.Project.NewWriter(nil).Create(ctx, &to)).To(Succeed())
			source = group.Group{Name: "source", Parent: new(from.ResourceID())}
			Expect(svc.Group.NewWriter(nil).Create(ctx, &source)).To(Succeed())
			dest = group.Group{Name: "dest", Parent: new(to.ResourceID())}
			Expect(svc.Group.NewWriter(nil).Create(ctx, &dest)).To(Succeed())
			nested = newGroup(ctx, resource.TypeProject, &source)
		})
		groupIDs := func() []resource.ID {
			return []resource.ID{
				nested.ResourceID(),
				source.ResourceID(),
				dest.ResourceID(),
			}
		}
		move := func(ctx SpecContext, u user.User) (struct{}, error) {
			return setParent(
				AuthedCtx(ctx, u),
				apigroup.SetParentRequest{
					Keys:   []group.Key{nested.Key},
					Parent: new(dest.ResourceID()),
				},
			)
		}

		It(
			"Should move an empty group with update on the groups alone",
			func(ctx SpecContext) {
				u := newUser(ctx, grant(update, groupIDs()...))
				MustSucceed(move(ctx, u))
				Expect(retrieve(ctx, nested.Key).Parent).
					To(Equal(new(dest.ResourceID())))
			},
		)

		Context("Holding a schematic", func() {
			BeforeEach(func(ctx SpecContext) {
				sc = schematic.Schematic{
					Name:    "moved",
					Project: &from.Key,
					Group:   &nested.Key,
				}
				Expect(svc.Schematic.NewWriter(nil).Create(ctx, &sc)).To(Succeed())
			})

			DescribeTable(
				"Should refuse the move and keep the schematic in its project",
				func(ctx SpecContext, refusedOn func() resource.ID) {
					objects := append(
						groupIDs(),
						from.ResourceID(),
						to.ResourceID(),
						sc.ResourceID(),
					)
					refused := refusedOn()
					u := newUser(ctx, grant(update, lo.Without(objects, refused)...))
					Expect(move(ctx, u)).
						Error().To(denied(u, access.ActionUpdate, refused))
					var stored schematic.Schematic
					Expect(svc.Schematic.NewRetrieve().
						Where(schematic.MatchKeys(sc.Key)).
						Entry(&stored).
						Exec(ctx, nil)).To(Succeed())
					Expect(stored.Project).To(Equal(&from.Key))
				},
				Entry("without update on the schematic", func() resource.ID {
					return sc.ResourceID()
				}),
				Entry("without update on the source project", func() resource.ID {
					return from.ResourceID()
				}),
				Entry(
					"without update on the destination project",
					func() resource.ID { return to.ResourceID() },
				),
			)

			It(
				"Should move the schematic with update on it and both projects",
				func(ctx SpecContext) {
					u := newUser(ctx, grant(update, append(
						groupIDs(),
						from.ResourceID(),
						to.ResourceID(),
						sc.ResourceID(),
					)...))
					MustSucceed(move(ctx, u))
					var stored schematic.Schematic
					Expect(svc.Schematic.NewRetrieve().
						Where(schematic.MatchKeys(sc.Key)).
						Entry(&stored).
						Exec(ctx, nil)).To(Succeed())
					Expect(stored.Project).To(Equal(&to.Key))
				},
			)
		})
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
					Where: &group.Where{Parents: []resource.ID{top.ResourceID()}},
				})
				Expect(keys(res.Groups)).To(Equal([]group.Key{nested.Key}))
			},
		)

		It("Should keep only the groups with no parent", func(ctx SpecContext) {
			res := retrieve(ctx, apigroup.RetrieveRequest{
				Where: &group.Where{
					Key:    &filter.Set[group.Key]{In: all()},
					Nested: new(false),
				},
			})
			Expect(keys(res.Groups)).To(ConsistOf(top.Key, devices.Key))
		})

		It("Should keep only the groups of the given scopes", func(ctx SpecContext) {
			res := retrieve(ctx, apigroup.RetrieveRequest{
				Where: &group.Where{
					Key:   &filter.Set[group.Key]{In: all()},
					Scope: &filter.Set[resource.Type]{Eq: new(resource.TypeDevice)},
				},
			})
			Expect(keys(res.Groups)).To(Equal([]group.Key{devices.Key}))
		})

		It(
			"Should reject the request when retrieve is not granted on a match",
			func(ctx SpecContext) {
				u := newUser(ctx)
				Expect(apiSvc.Retrieve(AuthedCtx(ctx, u), apigroup.RetrieveRequest{
					Where: &group.Where{Key: &filter.Set[group.Key]{Eq: &top.Key}},
				})).Error().To(MatchError(access.ErrDenied))
			},
		)
	})

	Describe("RetrieveAncestors", func() {
		It("Should return the groups above the given groups", func(ctx SpecContext) {
			top := newGroup(ctx, resource.TypeChannel, nil)
			mid := newGroup(ctx, resource.TypeChannel, &top)
			leaf := newGroup(ctx, resource.TypeChannel, &mid)
			u := newUser(ctx, grant([]access.Action{access.ActionRetrieve}, groups))
			res := MustSucceed(apiSvc.RetrieveAncestors(
				AuthedCtx(ctx, u),
				apigroup.RetrieveAncestorsRequest{Keys: []group.Key{leaf.Key}},
			))
			Expect(keys(res.Groups)).To(ConsistOf(top.Key, mid.Key))
		})

		It(
			"Should reject the request when retrieve is not granted",
			func(ctx SpecContext) {
				top := newGroup(ctx, resource.TypeChannel, nil)
				leaf := newGroup(ctx, resource.TypeChannel, &top)
				Expect(apiSvc.RetrieveAncestors(
					AuthedCtx(ctx, newUser(ctx)),
					apigroup.RetrieveAncestorsRequest{Keys: []group.Key{leaf.Key}},
				)).Error().To(MatchError(access.ErrDenied))
			},
		)
	})
})
