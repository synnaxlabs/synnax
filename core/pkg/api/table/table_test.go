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
	"github.com/onsi/gomega/types"
	apitable "github.com/synnaxlabs/synnax/pkg/api/table"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
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

var _ = Describe("Retrieve", func() {
	It("Should reject the request when retrieve is not granted on a match", func(
		ctx SpecContext,
	) {
		r := table.Table{Project: &proj.Key, Name: "unretrievable"}
		Expect(svc.Table.NewWriter(nil).Create(ctx, &r)).To(Succeed())
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
		res := MustSucceed(create(AuthedCtx(ctx, u), apitable.CreateRequest{
			Tables:      []table.Table{{Name: "broadcast", Project: &proj.Key}},
			DispatchKey: "dk-create",
		}))
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(1))
		Expect(frames[0].Key).To(Equal(res.Tables[0].Key))
		Expect(frames[0].DispatchKey).To(Equal("dk-create"))
	})
	It("Should refuse the create without create on the table", func(ctx SpecContext) {
		u := newUser(ctx)
		grant(ctx, access.ActionUpdate, u.ResourceID(), project.ResourceID(proj.Key))
		t := table.Table{Key: uuid.New(), Name: "Denied", Project: &proj.Key}
		Expect(create(AuthedCtx(ctx, u), apitable.CreateRequest{
			Tables: []table.Table{t},
		})).Error().To(denied(u, access.ActionCreate, t.ResourceID()))
		Expect(svc.Table.NewRetrieve().
			Where(table.MatchKeys(t.Key)).
			Entry(&table.Table{}).
			Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
	})
	It("Should refuse to overwrite a table without update on it", func(
		ctx SpecContext,
	) {
		u := newCreator(ctx)
		stored := createTable(ctx, "Stored")
		Expect(create(AuthedCtx(ctx, u), apitable.CreateRequest{
			Tables: []table.Table{
				{Key: stored.Key, Name: "Overwritten", Project: &proj.Key},
			},
		})).Error().To(denied(u, access.ActionUpdate, stored.ResourceID()))
		Expect(retrieveTable(ctx, stored.Key).Name).To(Equal("Stored"))
	})
})

var _ = Describe("Delete", func() {
	var (
		u user.User
		t table.Table
	)
	BeforeEach(func(ctx SpecContext) {
		u = newUser(ctx)
		t = createTable(ctx, "Deleted")
	})
	deleteAs := func(ctx SpecContext) (struct{}, error) {
		return del(AuthedCtx(ctx, u), apitable.DeleteRequest{Keys: []table.Key{t.Key}})
	}
	tableID := func() resource.ID { return t.ResourceID() }
	projectID := func() resource.ID { return project.ResourceID(proj.Key) }
	DescribeTable(
		"Should refuse the delete and keep the table",
		func(
			ctx SpecContext,
			granted access.Action,
			grantedOn func() resource.ID,
			refused access.Action,
			refusedOn func() resource.ID,
		) {
			grant(ctx, granted, u.ResourceID(), grantedOn())
			Expect(deleteAs(ctx)).Error().To(denied(u, refused, refusedOn()))
			Expect(retrieveTable(ctx, t.Key).Key).To(Equal(t.Key))
		},
		Entry(
			"without delete on the table",
			access.ActionUpdate,
			projectID,
			access.ActionDelete,
			tableID,
		),
		Entry(
			"without update on its project",
			access.ActionDelete,
			tableID,
			access.ActionUpdate,
			projectID,
		),
	)
	It(
		"Should delete the table with delete on it and update on its project",
		func(ctx SpecContext) {
			grant(ctx, access.ActionDelete, u.ResourceID(), tableID())
			grant(ctx, access.ActionUpdate, u.ResourceID(), projectID())
			MustSucceed(deleteAs(ctx))
			Expect(svc.Table.NewRetrieve().
				Where(table.MatchKeys(t.Key)).
				Entry(&table.Table{}).
				Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
		},
	)
})

var _ = Describe("Dispatch", func() {
	var (
		u user.User
		t table.Table
	)
	BeforeEach(func(ctx SpecContext) {
		u = newUser(ctx)
		t = createTable(ctx, "Dispatched")
	})
	dispatchAs := func(ctx SpecContext) (struct{}, error) {
		return dispatch(AuthedCtx(ctx, u), apitable.DispatchRequest{
			Key: t.Key,
			Actions: []table.Action{
				table.NewRenameAction(table.RenamePayload{Name: "Renamed"}),
			},
		})
	}
	It(
		"Should refuse a dispatch without update on the table and write nothing",
		func(ctx SpecContext) {
			Expect(dispatchAs(ctx)).Error().
				To(denied(u, access.ActionUpdate, t.ResourceID()))
			Expect(retrieveTable(ctx, t.Key).Name).To(Equal("Dispatched"))
		},
	)
	It("Should apply the actions with update on the table", func(ctx SpecContext) {
		grant(ctx, access.ActionUpdate, u.ResourceID(), t.ResourceID())
		MustSucceed(dispatchAs(ctx))
		Expect(retrieveTable(ctx, t.Key).Name).To(Equal("Renamed"))
	})
})

// createGroup creates a group in the suite project.
func createGroup(ctx SpecContext, name string) group.Group {
	GinkgoHelper()
	g := group.Group{Name: name, Parent: new(project.ResourceID(proj.Key))}
	Expect(svc.Group.NewWriter(nil).Create(ctx, &g)).To(Succeed())
	return g
}

func createTable(ctx SpecContext, name string) table.Table {
	GinkgoHelper()
	t := table.Table{Project: &proj.Key, Name: name}
	Expect(svc.Table.NewWriter(nil).Create(ctx, &t)).To(Succeed())
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
			g := createGroup(ctx, "Set group denied")
			grant(ctx, access.ActionUpdate, u.ResourceID(), table.ResourceID(t.Key))
			Expect(setGroup(
				AuthedCtx(ctx, u),
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
			g := createGroup(ctx, "Set group granted")
			grant(
				ctx,
				access.ActionUpdate,
				u.ResourceID(),
				table.ResourceID(t.Key),
				g.ResourceID(),
			)
			Expect(setGroup(
				AuthedCtx(ctx, u),
				apitable.SetGroupRequest{Group: &g.Key, Keys: []table.Key{t.Key}},
			)).Error().ToNot(HaveOccurred())
			Expect(retrieveTable(ctx, t.Key).Group).To(Equal(&g.Key))
		},
	)
	Context("Into a group under another project", func() {
		var (
			u    user.User
			x    table.Table
			dest project.Project
			g    group.Group
		)
		BeforeEach(func(ctx SpecContext) {
			u = newUser(ctx)
			x = createTable(ctx, "Cross project")
			dest = project.Project{Name: "Destination"}
			Expect(svc.Project.NewWriter(nil).Create(ctx, &dest)).To(Succeed())
			g = group.Group{Name: "Destination", Parent: new(dest.ResourceID())}
			Expect(svc.Group.NewWriter(nil).Create(ctx, &g)).To(Succeed())
		})
		move := func(ctx SpecContext) (struct{}, error) {
			return setGroup(
				AuthedCtx(ctx, u),
				apitable.SetGroupRequest{Group: &g.Key, Keys: []table.Key{x.Key}},
			)
		}
		DescribeTable(
			"Should refuse the move without update on both projects",
			func(ctx SpecContext, projects func() []resource.ID) {
				objects := projects()
				grant(ctx, access.ActionUpdate, u.ResourceID(), append(
					objects,
					table.ResourceID(x.Key),
					g.ResourceID(),
				)...)
				Expect(move(ctx)).Error().To(MatchError(access.ErrDenied))
				Expect(retrieveTable(ctx, x.Key).Project).To(Equal(&proj.Key))
			},
			Entry("without the destination", func() []resource.ID {
				return []resource.ID{project.ResourceID(proj.Key)}
			}),
			Entry("without the source", func() []resource.ID {
				return []resource.ID{dest.ResourceID()}
			}),
		)
		It("Should move the table with update on both projects", func(ctx SpecContext) {
			objects := []resource.ID{project.ResourceID(proj.Key), dest.ResourceID()}
			grant(ctx, access.ActionUpdate, u.ResourceID(), append(
				objects,
				table.ResourceID(x.Key),
				g.ResourceID(),
			)...)
			MustSucceed(move(ctx))
			Expect(retrieveTable(ctx, x.Key).Project).To(Equal(&dest.Key))
		})
	})
})

var _ = Describe("Create in a group", func() {
	It(
		"Should refuse the create without update on the group",
		func(ctx SpecContext) {
			u := newCreator(ctx)
			g := createGroup(ctx, "Create denied")
			Expect(create(AuthedCtx(ctx, u), apitable.CreateRequest{
				Tables: []table.Table{
					{Project: &proj.Key, Name: "Denied", Group: &g.Key},
				},
			})).Error().To(MatchError(access.ErrDenied))
		},
	)
	It("Should create the table with update on the group", func(ctx SpecContext) {
		u := newCreator(ctx)
		g := createGroup(ctx, "Create granted")
		grant(ctx, access.ActionUpdate, u.ResourceID(), g.ResourceID())
		res := MustSucceed(create(AuthedCtx(ctx, u), apitable.CreateRequest{
			Tables: []table.Table{{Project: &proj.Key, Name: "Granted", Group: &g.Key}},
		}))
		Expect(retrieveTable(ctx, res.Tables[0].Key).Group).To(Equal(&g.Key))
	})
	Context("Under another project", func() {
		var (
			u    user.User
			dest project.Project
			g    group.Group
		)
		BeforeEach(func(ctx SpecContext) {
			u = newCreator(ctx)
			dest = project.Project{Name: "Create destination"}
			Expect(svc.Project.NewWriter(nil).Create(ctx, &dest)).To(Succeed())
			g = group.Group{Name: "Create destination", Parent: new(dest.ResourceID())}
			Expect(svc.Group.NewWriter(nil).Create(ctx, &g)).To(Succeed())
		})
		createIn := func(
			ctx SpecContext,
			carried *project.Key,
		) (apitable.CreateResponse, error) {
			return create(AuthedCtx(ctx, u), apitable.CreateRequest{
				Tables: []table.Table{
					{Project: carried, Name: "Elsewhere", Group: &g.Key},
				},
			})
		}
		DescribeTable(
			"Should refuse a create in a group without update on the group's project",
			func(ctx SpecContext, carried func() *project.Key) {
				grant(ctx, access.ActionUpdate, u.ResourceID(), g.ResourceID())
				Expect(
					createIn(ctx, carried()),
				).Error().
					To(MatchError(access.ErrDenied))
			},
			Entry(
				"with the project it carries",
				func() *project.Key { return &proj.Key },
			),
			Entry("with no project", func() *project.Key { return nil }),
		)
		It(
			"Should create the table with update on the group's project",
			func(ctx SpecContext) {
				grant(
					ctx,
					access.ActionUpdate,
					u.ResourceID(),
					g.ResourceID(),
					dest.ResourceID(),
				)
				res := MustSucceed(createIn(ctx, nil))
				Expect(
					retrieveTable(ctx, res.Tables[0].Key).Project,
				).To(Equal(&dest.Key))
			},
		)
	})
})

var _ = Describe("Create with an owner", func() {
	It("Should refuse the create without update on the owner", func(ctx SpecContext) {
		u := newUser(ctx)
		grant(
			ctx,
			access.ActionCreate,
			u.ResourceID(),
			resource.ID{Type: resource.TypeTable},
		)
		Expect(create(AuthedCtx(ctx, u), apitable.CreateRequest{
			Tables: []table.Table{{Project: &proj.Key, Name: "Denied"}},
		})).Error().To(MatchError(access.ErrDenied))
	})
})

var _ = Describe("SetProject", func() {
	createProject := func(ctx SpecContext) project.Project {
		GinkgoHelper()
		p := project.Project{Name: "destination"}
		Expect(svc.Project.NewWriter(nil).Create(ctx, &p)).To(Succeed())
		return p
	}
	setProject := func(
		ctx SpecContext,
		u user.User,
		to project.Key,
		key table.Key,
	) (struct{}, error) {
		GinkgoHelper()
		return setProject(AuthedCtx(ctx, u), apitable.SetProjectRequest{
			Project: to,
			Keys:    []table.Key{key},
		})
	}
	It(
		"Should refuse the move without update on the destination project",
		func(ctx SpecContext) {
			u := newCreator(ctx)
			e := createTable(ctx, "Set project denied")
			dest := createProject(ctx)
			grant(ctx, access.ActionUpdate, u.ResourceID(), table.ResourceID(e.Key))
			Expect(
				setProject(ctx, u, dest.Key, e.Key),
			).Error().
				To(MatchError(access.ErrDenied))
			Expect(retrieveTable(ctx, e.Key).Project).To(Equal(&proj.Key))
		},
	)
	It(
		"Should refuse the move without update on the source project",
		func(ctx SpecContext) {
			u := newUser(ctx)
			e := createTable(ctx, "Set project denied")
			dest := createProject(ctx)
			grant(
				ctx,
				access.ActionUpdate,
				u.ResourceID(),
				table.ResourceID(e.Key),
				project.ResourceID(dest.Key),
			)
			Expect(
				setProject(ctx, u, dest.Key, e.Key),
			).Error().
				To(MatchError(access.ErrDenied))
			Expect(retrieveTable(ctx, e.Key).Project).To(Equal(&proj.Key))
		},
	)
	It(
		"Should move the table with update on both projects",
		func(ctx SpecContext) {
			u := newCreator(ctx)
			e := createTable(ctx, "Set project granted")
			dest := createProject(ctx)
			grant(
				ctx,
				access.ActionUpdate,
				u.ResourceID(),
				table.ResourceID(e.Key),
				project.ResourceID(dest.Key),
			)
			MustSucceed(setProject(ctx, u, dest.Key, e.Key))
			Expect(retrieveTable(ctx, e.Key).Project).To(Equal(&dest.Key))
		},
	)
	It(
		"Should refuse a move out of a group the subject cannot update",
		func(ctx SpecContext) {
			u := newCreator(ctx)
			e := createTable(ctx, "Set project grouped")
			g := createGroup(ctx, "Set project grouped")
			Expect(svc.Table.NewWriter(nil).SetGroup(ctx, &g.Key, e.Key)).To(Succeed())
			grant(ctx, access.ActionUpdate, u.ResourceID(), table.ResourceID(e.Key))
			Expect(
				setProject(ctx, u, proj.Key, e.Key),
			).Error().
				To(MatchError(access.ErrDenied))
			Expect(retrieveTable(ctx, e.Key).Group).To(Equal(&g.Key))
		},
	)
})
