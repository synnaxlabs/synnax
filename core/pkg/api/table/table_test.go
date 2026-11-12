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
	"github.com/synnaxlabs/gorp"
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
	"github.com/synnaxlabs/x/validate"
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
	It("Should leave out the tables that the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		u := newUser(ctx)
		granted := createTable(ctx, "leave-out-granted")
		createTable(ctx, "leave-out-denied")
		grant(ctx, access.ActionRetrieve, u.ResourceID(), granted.ResourceID())
		res := MustSucceed(retrieve(AuthedCtx(ctx, u), apitable.RetrieveRequest{}))
		Expect(res.Tables).To(HaveExactElements(HaveField("Key", granted.Key)))
	})

	It("Should reject the request when retrieve is not granted on a match", func(
		ctx SpecContext,
	) {
		r := table.Table{Project: &proj.Key, Name: "unretrievable"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Table.NewWriter(tx).Create(ctx, &r)
		})).To(Succeed())
		Expect(retrieve(AuthedCtx(ctx, author), apitable.RetrieveRequest{
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
			Exec(ctx, db)).To(MatchError(query.ErrNotFound))
	})
	It("Should refuse a table whose key exists", func(ctx SpecContext) {
		u := newCreator(ctx)
		stored := createTable(ctx, "Stored")
		Expect(create(AuthedCtx(ctx, u), apitable.CreateRequest{
			Tables: []table.Table{
				{Key: stored.Key, Name: "Overwritten", Project: &proj.Key},
			},
		})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("table %s already exists", stored.Key)),
		))
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
				Exec(ctx, db)).To(MatchError(query.ErrNotFound))
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
	dispatchAs := func(ctx SpecContext) (apitable.DispatchResponse, error) {
		return dispatch(AuthedCtx(ctx, u), apitable.DispatchRequest{
			Key: t.Key,
			Actions: []table.Action{
				table.NewSetAction(
					table.SetPayload{Path: []string{"name"}, Value: "Renamed"},
				),
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
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Group.NewWriter(tx).Create(ctx, &g)
	})).To(Succeed())
	return g
}

func createTable(ctx SpecContext, name string) table.Table {
	GinkgoHelper()
	t := table.Table{Project: &proj.Key, Name: name}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Table.NewWriter(tx).Create(ctx, &t)
	})).To(Succeed())
	return t
}

func retrieveTable(ctx SpecContext, key table.Key) table.Table {
	GinkgoHelper()
	var t table.Table
	Expect(svc.Table.NewRetrieve().
		Where(table.MatchKeys(key)).
		Entry(&t).
		Exec(ctx, db)).To(Succeed())
	return t
}

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
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Project.NewWriter(tx).Create(ctx, &dest)
			})).To(Succeed())
			g = group.Group{Name: "Create destination", Parent: new(dest.ResourceID())}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Group.NewWriter(tx).Create(ctx, &g)
			})).To(Succeed())
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

var _ = Describe("Dispatch of a move", func() {
	var (
		u      user.User
		target table.Table
		g      group.Group
		dest   project.Project
		destG  group.Group
		rec    *Recorder[table.Key, table.Action]
	)
	BeforeEach(func(ctx SpecContext) {
		u = newUser(ctx)
		target = createTable(ctx, "Moved")
		g = createGroup(ctx, "Source")
		dest = project.Project{Name: "Destination"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Project.NewWriter(tx).Create(ctx, &dest)
		})).To(Succeed())
		destG = group.Group{Name: "Destination", Parent: new(dest.ResourceID())}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Group.NewWriter(tx).Create(ctx, &destG)
		})).To(Succeed())
		rec = &Recorder[table.Key, table.Action]{}
	})
	setGroup := func(k group.Key) table.Action {
		return table.NewSetAction(table.SetPayload{Path: []string{"group"}, Value: k})
	}
	setProject := func(k project.Key) table.Action {
		return table.NewSetAction(table.SetPayload{Path: []string{"project"}, Value: k})
	}
	removeGroup := table.NewRemoveAction(table.RemovePayload{Key: "group"})
	// inGroup moves the table into g.
	inGroup := func(ctx SpecContext) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			_, err := svc.Table.NewWriter(tx).Apply(ctx, target.Key, setGroup(g.Key))
			return err
		})).To(Succeed())
	}
	// moveAs grants u update on the table and on objects, records the broadcasts that
	// follow, and dispatches acts as u.
	moveAs := func(
		ctx SpecContext,
		acts []table.Action,
		objects ...resource.ID,
	) (apitable.DispatchResponse, error) {
		GinkgoHelper()
		grant(
			ctx,
			access.ActionUpdate,
			u.ResourceID(),
			append(objects, target.ResourceID())...,
		)
		DeferCleanup(svc.Table.OnAction(rec.Record))
		return dispatch(AuthedCtx(ctx, u), apitable.DispatchRequest{
			Key:         target.Key,
			DispatchKey: "dk-move",
			Actions:     acts,
		})
	}
	It("Should move the table into a group", func(ctx SpecContext) {
		Expect(moveAs(ctx, []table.Action{setGroup(g.Key)}, g.ResourceID())).
			To(HaveField("Actions", BeEmpty()))
		Expect(retrieveTable(ctx, target.Key).Group).To(Equal(&g.Key))
		Expect(rec.Snapshot()).To(HaveExactElements(SatisfyAll(
			HaveField("DispatchKey", "dk-move"),
			HaveField("Actions", []table.Action{setGroup(g.Key)}),
		)))
	})
	It(
		"Should move the table into the project of a group in another project",
		func(ctx SpecContext) {
			res := MustSucceed(moveAs(
				ctx,
				[]table.Action{setGroup(destG.Key)},
				destG.ResourceID(),
				project.ResourceID(proj.Key),
				dest.ResourceID(),
			))
			Expect(res.Actions).To(Equal([]table.Action{setProject(dest.Key)}))
			moved := retrieveTable(ctx, target.Key)
			Expect(moved.Group).To(Equal(&destG.Key))
			Expect(moved.Project).To(Equal(&dest.Key))
			Expect(rec.Snapshot()).To(HaveExactElements(HaveField(
				"Actions",
				[]table.Action{setGroup(destG.Key), setProject(dest.Key)},
			)))
		},
	)
	It("Should move the table out of its group", func(ctx SpecContext) {
		inGroup(ctx)
		Expect(moveAs(ctx, []table.Action{removeGroup}, g.ResourceID())).
			To(HaveField("Actions", BeEmpty()))
		Expect(retrieveTable(ctx, target.Key).Group).To(BeNil())
	})
	It("Should move the table into another project", func(ctx SpecContext) {
		Expect(moveAs(
			ctx,
			[]table.Action{setProject(dest.Key)},
			project.ResourceID(proj.Key),
			dest.ResourceID(),
		)).To(HaveField("Actions", BeEmpty()))
		Expect(retrieveTable(ctx, target.Key).Project).To(Equal(&dest.Key))
	})
	It(
		"Should move the table out of its group when it moves into another project",
		func(ctx SpecContext) {
			inGroup(ctx)
			res := MustSucceed(moveAs(
				ctx,
				[]table.Action{setProject(dest.Key)},
				g.ResourceID(),
				project.ResourceID(proj.Key),
				dest.ResourceID(),
			))
			Expect(res.Actions).To(Equal([]table.Action{removeGroup}))
			moved := retrieveTable(ctx, target.Key)
			Expect(moved.Group).To(BeNil())
			Expect(moved.Project).To(Equal(&dest.Key))
		},
	)
	It("Should refuse a move into a group that does not exist", func(ctx SpecContext) {
		missing := uuid.New()
		Expect(moveAs(ctx, []table.Action{setGroup(missing)})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("group: group %s does not exist", missing)),
		))
		Expect(retrieveTable(ctx, target.Key).Group).To(BeNil())
		Expect(rec.Snapshot()).To(BeEmpty())
	})
	DescribeTable(
		"Should refuse a move without update on a record it changes",
		func(
			ctx SpecContext,
			grouped bool,
			move func() ([]table.Action, []resource.ID, resource.ID),
		) {
			if grouped {
				inGroup(ctx)
			}
			before := retrieveTable(ctx, target.Key)
			acts, granted, refused := move()
			Expect(moveAs(ctx, acts, granted...)).Error().
				To(denied(u, access.ActionUpdate, refused))
			Expect(retrieveTable(ctx, target.Key)).To(Equal(before))
			Expect(rec.Snapshot()).To(BeEmpty())
		},
		Entry(
			"into a group without the group",
			false,
			func() ([]table.Action, []resource.ID, resource.ID) {
				return []table.Action{setGroup(g.Key)}, nil, g.ResourceID()
			},
		),
		Entry(
			"into a group of another project without that project",
			false,
			func() ([]table.Action, []resource.ID, resource.ID) {
				return []table.Action{setGroup(destG.Key)},
					[]resource.ID{destG.ResourceID(), project.ResourceID(proj.Key)},
					dest.ResourceID()
			},
		),
		Entry(
			"into a group of another project without the source project",
			false,
			func() ([]table.Action, []resource.ID, resource.ID) {
				return []table.Action{setGroup(destG.Key)},
					[]resource.ID{destG.ResourceID(), dest.ResourceID()},
					project.ResourceID(proj.Key)
			},
		),
		Entry(
			"into another project without that project",
			false,
			func() ([]table.Action, []resource.ID, resource.ID) {
				return []table.Action{setProject(dest.Key)},
					[]resource.ID{project.ResourceID(proj.Key)},
					dest.ResourceID()
			},
		),
		Entry(
			"into another project without the source project",
			false,
			func() ([]table.Action, []resource.ID, resource.ID) {
				return []table.Action{setProject(dest.Key)},
					[]resource.ID{dest.ResourceID()},
					project.ResourceID(proj.Key)
			},
		),
		Entry(
			"out of a group without the group",
			true,
			func() ([]table.Action, []resource.ID, resource.ID) {
				return []table.Action{setProject(dest.Key)},
					[]resource.ID{project.ResourceID(proj.Key), dest.ResourceID()},
					g.ResourceID()
			},
		),
	)
})
