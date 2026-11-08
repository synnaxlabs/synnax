// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot_test

import (
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/gorp"
	apilineplot "github.com/synnaxlabs/synnax/pkg/api/lineplot"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
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
	It("Should reject the request when retrieve is not granted on a match", func(
		ctx SpecContext,
	) {
		r := lineplot.LinePlot{Project: &proj.Key, Name: "unretrievable"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.LinePlot.NewWriter(tx).Create(ctx, &r)
		})).To(Succeed())
		Expect(apiSvc.Retrieve(AuthedCtx(ctx, author), apilineplot.RetrieveRequest{
			Where: &lineplot.Where{Key: &filter.Set[lineplot.Key]{Eq: &r.Key}},
		})).Error().To(MatchError(access.ErrDenied))
	})
})

var _ = Describe("Create", func() {
	It("Should stamp the dispatch key onto the create broadcast", func(
		ctx SpecContext,
	) {
		u := newCreator(ctx)
		rec := &Recorder[lineplot.Key, lineplot.Action]{}
		DeferCleanup(svc.LinePlot.OnAction(rec.Record))
		res := MustSucceed(
			create(AuthedCtx(ctx, u), apilineplot.CreateRequest{
				LinePlots: []lineplot.LinePlot{
					{Name: "broadcast", Project: &proj.Key},
				},
				DispatchKey: "dk-create",
			}),
		)
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(1))
		Expect(frames[0].Key).To(Equal(res.LinePlots[0].Key))
		Expect(frames[0].DispatchKey).To(Equal("dk-create"))
	})
	It("Should refuse the create without create on the line plot", func(
		ctx SpecContext,
	) {
		u := newUser(ctx)
		grant(ctx, access.ActionUpdate, u.ResourceID(), project.ResourceID(proj.Key))
		lp := lineplot.LinePlot{Key: uuid.New(), Name: "Denied", Project: &proj.Key}
		Expect(create(AuthedCtx(ctx, u), apilineplot.CreateRequest{
			LinePlots: []lineplot.LinePlot{lp},
		})).Error().To(denied(u, access.ActionCreate, lp.ResourceID()))
		Expect(svc.LinePlot.NewRetrieve().
			Where(lineplot.MatchKeys(lp.Key)).
			Entry(&lineplot.LinePlot{}).
			Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
	})
	It("Should refuse to overwrite a line plot without update on it", func(
		ctx SpecContext,
	) {
		u := newCreator(ctx)
		stored := createLinePlot(ctx, "Stored")
		Expect(create(AuthedCtx(ctx, u), apilineplot.CreateRequest{
			LinePlots: []lineplot.LinePlot{
				{Key: stored.Key, Name: "Overwritten", Project: &proj.Key},
			},
		})).Error().To(denied(u, access.ActionUpdate, stored.ResourceID()))
		Expect(retrieveLinePlot(ctx, stored.Key).Name).To(Equal("Stored"))
	})
	It("Should require retrieve on each channel the line plot plots", func(
		ctx SpecContext,
	) {
		u := newCreator(ctx)
		ch := channel.Channel{
			Name:     "plotted_" + strings.ReplaceAll(uuid.New().String(), "-", "_"),
			DataType: telem.Float64T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Channel.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		req := apilineplot.CreateRequest{
			LinePlots: []lineplot.LinePlot{{
				Name:     "Plotted",
				Project:  &proj.Key,
				Channels: lineplot.Channels{Y1: []channel.Key{ch.Key()}},
			}},
		}
		Expect(create(AuthedCtx(ctx, u), req)).Error().
			To(denied(u, access.ActionRetrieve, channel.ResourceID(ch.Key())))
		grant(ctx, access.ActionRetrieve, u.ResourceID(), channel.ResourceID(ch.Key()))
		MustSucceed(create(AuthedCtx(ctx, u), req))
	})
})

var _ = Describe("Delete", func() {
	var (
		u  user.User
		lp lineplot.LinePlot
	)
	BeforeEach(func(ctx SpecContext) {
		u = newUser(ctx)
		lp = createLinePlot(ctx, "Deleted")
	})
	deleteAs := func(ctx SpecContext) (struct{}, error) {
		return del(AuthedCtx(ctx, u), apilineplot.DeleteRequest{
			Keys: []lineplot.Key{lp.Key},
		})
	}
	linePlotID := func() resource.ID { return lp.ResourceID() }
	projectID := func() resource.ID { return project.ResourceID(proj.Key) }
	DescribeTable(
		"Should refuse the delete and keep the line plot",
		func(
			ctx SpecContext,
			granted access.Action,
			grantedOn func() resource.ID,
			refused access.Action,
			refusedOn func() resource.ID,
		) {
			grant(ctx, granted, u.ResourceID(), grantedOn())
			Expect(deleteAs(ctx)).Error().To(denied(u, refused, refusedOn()))
			Expect(retrieveLinePlot(ctx, lp.Key).Key).To(Equal(lp.Key))
		},
		Entry(
			"without delete on the line plot",
			access.ActionUpdate,
			projectID,
			access.ActionDelete,
			linePlotID,
		),
		Entry(
			"without update on its project",
			access.ActionDelete,
			linePlotID,
			access.ActionUpdate,
			projectID,
		),
	)
	It(
		"Should delete the line plot with delete on it and update on its project",
		func(ctx SpecContext) {
			grant(ctx, access.ActionDelete, u.ResourceID(), linePlotID())
			grant(ctx, access.ActionUpdate, u.ResourceID(), projectID())
			MustSucceed(deleteAs(ctx))
			Expect(svc.LinePlot.NewRetrieve().
				Where(lineplot.MatchKeys(lp.Key)).
				Entry(&lineplot.LinePlot{}).
				Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
		},
	)
})

var _ = Describe("Dispatch", func() {
	var (
		u   user.User
		lp  lineplot.LinePlot
		rec *Recorder[lineplot.Key, lineplot.Action]
	)
	BeforeEach(func(ctx SpecContext) {
		u = newUser(ctx)
		lp = createLinePlot(ctx, "Dispatched")
		rec = &Recorder[lineplot.Key, lineplot.Action]{}
		DeferCleanup(svc.LinePlot.OnAction(rec.Record))
	})
	dispatchAs := func(ctx SpecContext, acts ...lineplot.Action) (struct{}, error) {
		return dispatch(AuthedCtx(ctx, u), apilineplot.DispatchRequest{
			Key:         lp.Key,
			DispatchKey: "dk-dispatch",
			Actions:     acts,
		})
	}
	rename := lineplot.NewRenameAction(lineplot.RenamePayload{Name: "Renamed"})
	It(
		"Should refuse a dispatch without update on the line plot and write nothing",
		func(ctx SpecContext) {
			Expect(dispatchAs(ctx, rename)).Error().
				To(denied(u, access.ActionUpdate, lp.ResourceID()))
			Expect(retrieveLinePlot(ctx, lp.Key).Name).To(Equal("Dispatched"))
			Expect(rec.Snapshot()).To(BeEmpty())
		},
	)
	It("Should stamp the dispatch key onto the action batch", func(ctx SpecContext) {
		grant(ctx, access.ActionUpdate, u.ResourceID(), lp.ResourceID())
		MustSucceed(dispatchAs(ctx, rename))
		Expect(retrieveLinePlot(ctx, lp.Key).Name).To(Equal("Renamed"))
		Expect(rec.Snapshot()).To(HaveExactElements(SatisfyAll(
			HaveField("Key", lp.Key),
			HaveField("DispatchKey", "dk-dispatch"),
			HaveField("Actions", []lineplot.Action{rename}),
		)))
	})
	DescribeTable(
		"Should refuse an action that only the Core dispatches",
		func(ctx SpecContext, act func() lineplot.Action, what string) {
			grant(ctx, access.ActionUpdate, u.ResourceID(), lp.ResourceID())
			Expect(dispatchAs(ctx, act())).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("only the Core dispatches %s", what)),
			))
			Expect(retrieveLinePlot(ctx, lp.Key).Project).To(Equal(&proj.Key))
			Expect(rec.Snapshot()).To(BeEmpty())
		},
		Entry(
			"a set group",
			func() lineplot.Action {
				g := uuid.New()
				return lineplot.NewSetGroupAction(lineplot.SetGroupPayload{Group: &g})
			},
			"a line plot group change",
		),
		Entry(
			"a set project",
			func() lineplot.Action {
				return lineplot.NewSetProjectAction(
					lineplot.SetProjectPayload{Project: uuid.New()},
				)
			},
			"a line plot project change",
		),
	)
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

func createLinePlot(ctx SpecContext, name string) lineplot.LinePlot {
	GinkgoHelper()
	lp := lineplot.LinePlot{Project: &proj.Key, Name: name}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.LinePlot.NewWriter(tx).Create(ctx, &lp)
	})).To(Succeed())
	return lp
}

func retrieveLinePlot(ctx SpecContext, key lineplot.Key) lineplot.LinePlot {
	GinkgoHelper()
	var lp lineplot.LinePlot
	Expect(svc.LinePlot.NewRetrieve().
		Where(lineplot.MatchKeys(key)).
		Entry(&lp).
		Exec(ctx, nil)).To(Succeed())
	return lp
}

var _ = Describe("SetGroup", func() {
	It(
		"Should refuse the move without update on the destination group",
		func(ctx SpecContext) {
			u := newCreator(ctx)
			lp := createLinePlot(ctx, "Set group denied")
			g := createGroup(ctx, "Set group denied")
			grant(ctx, access.ActionUpdate, u.ResourceID(), lineplot.ResourceID(lp.Key))
			Expect(setGroup(
				AuthedCtx(ctx, u),
				apilineplot.SetGroupRequest{
					Group: &g.Key,
					Keys:  []lineplot.Key{lp.Key},
				},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(retrieveLinePlot(ctx, lp.Key).Group).To(BeNil())
		},
	)
	It(
		"Should move the line plot with update on the destination group",
		func(ctx SpecContext) {
			u := newCreator(ctx)
			lp := createLinePlot(ctx, "Set group granted")
			g := createGroup(ctx, "Set group granted")
			grant(
				ctx,
				access.ActionUpdate,
				u.ResourceID(),
				lineplot.ResourceID(lp.Key),
				g.ResourceID(),
			)
			Expect(setGroup(
				AuthedCtx(ctx, u),
				apilineplot.SetGroupRequest{
					Group: &g.Key,
					Keys:  []lineplot.Key{lp.Key},
				},
			)).Error().ToNot(HaveOccurred())
			Expect(retrieveLinePlot(ctx, lp.Key).Group).To(Equal(&g.Key))
		},
	)
	Context("Into a group under another project", func() {
		var (
			u    user.User
			x    lineplot.LinePlot
			dest project.Project
			g    group.Group
		)
		BeforeEach(func(ctx SpecContext) {
			u = newUser(ctx)
			x = createLinePlot(ctx, "Cross project")
			dest = project.Project{Name: "Destination"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Project.NewWriter(tx).Create(ctx, &dest)
			})).To(Succeed())
			g = group.Group{Name: "Destination", Parent: new(dest.ResourceID())}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Group.NewWriter(tx).Create(ctx, &g)
			})).To(Succeed())
		})
		move := func(ctx SpecContext) (struct{}, error) {
			return setGroup(
				AuthedCtx(ctx, u),
				apilineplot.SetGroupRequest{Group: &g.Key, Keys: []lineplot.Key{x.Key}},
			)
		}
		DescribeTable(
			"Should refuse the move without update on both projects",
			func(ctx SpecContext, projects func() []resource.ID) {
				objects := projects()
				grant(ctx, access.ActionUpdate, u.ResourceID(), append(
					objects,
					lineplot.ResourceID(x.Key),
					g.ResourceID(),
				)...)
				Expect(move(ctx)).Error().To(MatchError(access.ErrDenied))
				Expect(retrieveLinePlot(ctx, x.Key).Project).To(Equal(&proj.Key))
			},
			Entry("without the destination", func() []resource.ID {
				return []resource.ID{project.ResourceID(proj.Key)}
			}),
			Entry("without the source", func() []resource.ID {
				return []resource.ID{dest.ResourceID()}
			}),
		)
		It(
			"Should move the line plot with update on both projects",
			func(ctx SpecContext) {
				objects := []resource.ID{
					project.ResourceID(proj.Key),
					dest.ResourceID(),
				}
				grant(ctx, access.ActionUpdate, u.ResourceID(), append(
					objects,
					lineplot.ResourceID(x.Key),
					g.ResourceID(),
				)...)
				MustSucceed(move(ctx))
				Expect(retrieveLinePlot(ctx, x.Key).Project).To(Equal(&dest.Key))
			},
		)
	})
})

var _ = Describe("Create in a group", func() {
	It(
		"Should refuse the create without update on the group",
		func(ctx SpecContext) {
			u := newCreator(ctx)
			g := createGroup(ctx, "Create denied")
			Expect(create(AuthedCtx(ctx, u), apilineplot.CreateRequest{
				LinePlots: []lineplot.LinePlot{
					{Project: &proj.Key, Name: "Denied", Group: &g.Key},
				},
			})).Error().To(MatchError(access.ErrDenied))
		},
	)
	It("Should create the line plot with update on the group", func(ctx SpecContext) {
		u := newCreator(ctx)
		g := createGroup(ctx, "Create granted")
		grant(ctx, access.ActionUpdate, u.ResourceID(), g.ResourceID())
		res := MustSucceed(
			create(AuthedCtx(ctx, u), apilineplot.CreateRequest{
				LinePlots: []lineplot.LinePlot{
					{Project: &proj.Key, Name: "Granted", Group: &g.Key},
				},
			}),
		)
		Expect(retrieveLinePlot(ctx, res.LinePlots[0].Key).Group).To(Equal(&g.Key))
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
		) (apilineplot.CreateResponse, error) {
			return create(AuthedCtx(ctx, u), apilineplot.CreateRequest{
				LinePlots: []lineplot.LinePlot{
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
			"Should create the line plot with update on the group's project",
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
					retrieveLinePlot(ctx, res.LinePlots[0].Key).Project,
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
			resource.ID{Type: resource.TypeLineplot},
		)
		Expect(create(AuthedCtx(ctx, u), apilineplot.CreateRequest{
			LinePlots: []lineplot.LinePlot{{Project: &proj.Key, Name: "Denied"}},
		})).Error().To(MatchError(access.ErrDenied))
	})
})

var _ = Describe("SetProject", func() {
	createProject := func(ctx SpecContext) project.Project {
		GinkgoHelper()
		p := project.Project{Name: "destination"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Project.NewWriter(tx).Create(ctx, &p)
		})).To(Succeed())
		return p
	}
	setProject := func(
		ctx SpecContext,
		u user.User,
		to project.Key,
		key lineplot.Key,
	) (struct{}, error) {
		GinkgoHelper()
		return setProject(
			AuthedCtx(ctx, u),
			apilineplot.SetProjectRequest{
				Project: to,
				Keys:    []lineplot.Key{key},
			},
		)
	}
	It(
		"Should refuse the move without update on the destination project",
		func(ctx SpecContext) {
			u := newCreator(ctx)
			e := createLinePlot(ctx, "Set project denied")
			dest := createProject(ctx)
			grant(ctx, access.ActionUpdate, u.ResourceID(), lineplot.ResourceID(e.Key))
			Expect(
				setProject(ctx, u, dest.Key, e.Key),
			).Error().
				To(MatchError(access.ErrDenied))
			Expect(retrieveLinePlot(ctx, e.Key).Project).To(Equal(&proj.Key))
		},
	)
	It(
		"Should refuse the move without update on the source project",
		func(ctx SpecContext) {
			u := newUser(ctx)
			e := createLinePlot(ctx, "Set project denied")
			dest := createProject(ctx)
			grant(
				ctx,
				access.ActionUpdate,
				u.ResourceID(),
				lineplot.ResourceID(e.Key),
				project.ResourceID(dest.Key),
			)
			Expect(
				setProject(ctx, u, dest.Key, e.Key),
			).Error().
				To(MatchError(access.ErrDenied))
			Expect(retrieveLinePlot(ctx, e.Key).Project).To(Equal(&proj.Key))
		},
	)
	It(
		"Should move the line plot with update on both projects",
		func(ctx SpecContext) {
			u := newCreator(ctx)
			e := createLinePlot(ctx, "Set project granted")
			dest := createProject(ctx)
			grant(
				ctx,
				access.ActionUpdate,
				u.ResourceID(),
				lineplot.ResourceID(e.Key),
				project.ResourceID(dest.Key),
			)
			MustSucceed(setProject(ctx, u, dest.Key, e.Key))
			Expect(retrieveLinePlot(ctx, e.Key).Project).To(Equal(&dest.Key))
		},
	)
	It(
		"Should refuse a move out of a group the subject cannot update",
		func(ctx SpecContext) {
			u := newCreator(ctx)
			e := createLinePlot(ctx, "Set project grouped")
			g := createGroup(ctx, "Set project grouped")
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.LinePlot.NewWriter(tx).SetGroup(ctx, &g.Key, e.Key)
			})).
				To(Succeed())
			grant(ctx, access.ActionUpdate, u.ResourceID(), lineplot.ResourceID(e.Key))
			Expect(
				setProject(ctx, u, proj.Key, e.Key),
			).Error().
				To(MatchError(access.ErrDenied))
			Expect(retrieveLinePlot(ctx, e.Key).Group).To(Equal(&g.Key))
		},
	)
})
