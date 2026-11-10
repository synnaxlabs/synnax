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
	"github.com/synnaxlabs/x/defaults"
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
	It("Should leave out the line plots that the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		u := newUser(ctx)
		granted := createLinePlot(ctx, "leave-out-granted")
		createLinePlot(ctx, "leave-out-denied")
		grant(ctx, access.ActionRetrieve, u.ResourceID(), granted.ResourceID())
		res := MustSucceed(retrieve(AuthedCtx(ctx, u), apilineplot.RetrieveRequest{}))
		Expect(res.LinePlots).To(HaveExactElements(HaveField("Key", granted.Key)))
	})

	It("Should reject the request when retrieve is not granted on a match", func(
		ctx SpecContext,
	) {
		r := defaults.Fill(lineplot.LinePlot{Project: &proj.Key, Name: "unretrievable"})
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.LinePlot.NewWriter(tx).Create(ctx, &r)
		})).To(Succeed())
		Expect(retrieve(AuthedCtx(ctx, author), apilineplot.RetrieveRequest{
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
				LinePlots: defaults.Fill([]lineplot.LinePlot{
					{Name: "broadcast", Project: &proj.Key},
				}),
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
		lp := defaults.Fill(
			lineplot.LinePlot{Key: uuid.New(), Name: "Denied", Project: &proj.Key},
		)
		Expect(create(AuthedCtx(ctx, u), apilineplot.CreateRequest{
			LinePlots: defaults.Fill([]lineplot.LinePlot{lp}),
		})).Error().To(denied(u, access.ActionCreate, lp.ResourceID()))
		Expect(svc.LinePlot.NewRetrieve().
			Where(lineplot.MatchKeys(lp.Key)).
			Entry(&lineplot.LinePlot{}).
			Exec(ctx, db)).To(MatchError(query.ErrNotFound))
	})
	It("Should refuse a line plot whose key exists", func(ctx SpecContext) {
		u := newCreator(ctx)
		stored := createLinePlot(ctx, "Stored")
		Expect(create(AuthedCtx(ctx, u), apilineplot.CreateRequest{
			LinePlots: defaults.Fill([]lineplot.LinePlot{
				{Key: stored.Key, Name: "Overwritten", Project: &proj.Key},
			}),
		})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("lineplot %s already exists", stored.Key)),
		))
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
			LinePlots: defaults.Fill([]lineplot.LinePlot{{
				Name:     "Plotted",
				Project:  &proj.Key,
				Channels: lineplot.Channels{Y1: []channel.Key{ch.Key()}},
			}}),
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
				Exec(ctx, db)).To(MatchError(query.ErrNotFound))
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
	dispatchAs := func(ctx SpecContext, acts ...lineplot.Action) (apilineplot.DispatchResponse, error) {
		return dispatch(AuthedCtx(ctx, u), apilineplot.DispatchRequest{
			Key:         lp.Key,
			DispatchKey: "dk-dispatch",
			Actions:     acts,
		})
	}
	rename := lineplot.NewSetAction(
		lineplot.SetPayload{Path: []string{"name"}, Value: "Renamed"},
	)
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
	lp := defaults.Fill(lineplot.LinePlot{Project: &proj.Key, Name: name})
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
		Exec(ctx, db)).To(Succeed())
	return lp
}

var _ = Describe("Create in a group", func() {
	It(
		"Should refuse the create without update on the group",
		func(ctx SpecContext) {
			u := newCreator(ctx)
			g := createGroup(ctx, "Create denied")
			Expect(create(AuthedCtx(ctx, u), apilineplot.CreateRequest{
				LinePlots: defaults.Fill([]lineplot.LinePlot{
					{Project: &proj.Key, Name: "Denied", Group: &g.Key},
				}),
			})).Error().To(MatchError(access.ErrDenied))
		},
	)
	It("Should create the line plot with update on the group", func(ctx SpecContext) {
		u := newCreator(ctx)
		g := createGroup(ctx, "Create granted")
		grant(ctx, access.ActionUpdate, u.ResourceID(), g.ResourceID())
		res := MustSucceed(
			create(AuthedCtx(ctx, u), apilineplot.CreateRequest{
				LinePlots: defaults.Fill([]lineplot.LinePlot{
					{Project: &proj.Key, Name: "Granted", Group: &g.Key},
				}),
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
				LinePlots: defaults.Fill([]lineplot.LinePlot{
					{Project: carried, Name: "Elsewhere", Group: &g.Key},
				}),
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
			LinePlots: defaults.Fill(
				[]lineplot.LinePlot{{Project: &proj.Key, Name: "Denied"}},
			),
		})).Error().To(MatchError(access.ErrDenied))
	})
})

var _ = Describe("Dispatch of a move", func() {
	var (
		u      user.User
		target lineplot.LinePlot
		g      group.Group
		dest   project.Project
		destG  group.Group
		rec    *Recorder[lineplot.Key, lineplot.Action]
	)
	BeforeEach(func(ctx SpecContext) {
		u = newUser(ctx)
		target = createLinePlot(ctx, "Moved")
		g = createGroup(ctx, "Source")
		dest = project.Project{Name: "Destination"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Project.NewWriter(tx).Create(ctx, &dest)
		})).To(Succeed())
		destG = group.Group{Name: "Destination", Parent: new(dest.ResourceID())}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Group.NewWriter(tx).Create(ctx, &destG)
		})).To(Succeed())
		rec = &Recorder[lineplot.Key, lineplot.Action]{}
	})
	setGroup := func(k group.Key) lineplot.Action {
		return lineplot.NewSetAction(
			lineplot.SetPayload{Path: []string{"group"}, Value: k},
		)
	}
	setProject := func(k project.Key) lineplot.Action {
		return lineplot.NewSetAction(
			lineplot.SetPayload{Path: []string{"project"}, Value: k},
		)
	}
	removeGroup := lineplot.NewRemoveAction(lineplot.RemovePayload{Key: "group"})
	// inGroup moves the line plot into g.
	inGroup := func(ctx SpecContext) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			_, err := svc.LinePlot.NewWriter(tx).Apply(ctx, target.Key, setGroup(g.Key))
			return err
		})).To(Succeed())
	}
	// moveAs grants u update on the line plot and on objects, records the broadcasts
	// that follow, and dispatches acts as u.
	moveAs := func(
		ctx SpecContext,
		acts []lineplot.Action,
		objects ...resource.ID,
	) (apilineplot.DispatchResponse, error) {
		GinkgoHelper()
		grant(
			ctx,
			access.ActionUpdate,
			u.ResourceID(),
			append(objects, target.ResourceID())...,
		)
		DeferCleanup(svc.LinePlot.OnAction(rec.Record))
		return dispatch(AuthedCtx(ctx, u), apilineplot.DispatchRequest{
			Key:         target.Key,
			DispatchKey: "dk-move",
			Actions:     acts,
		})
	}
	It("Should move the line plot into a group", func(ctx SpecContext) {
		Expect(moveAs(ctx, []lineplot.Action{setGroup(g.Key)}, g.ResourceID())).
			To(HaveField("Actions", BeEmpty()))
		Expect(retrieveLinePlot(ctx, target.Key).Group).To(Equal(&g.Key))
		Expect(rec.Snapshot()).To(HaveExactElements(SatisfyAll(
			HaveField("DispatchKey", "dk-move"),
			HaveField("Actions", []lineplot.Action{setGroup(g.Key)}),
		)))
	})
	It(
		"Should move the line plot into the project of a group in another project",
		func(ctx SpecContext) {
			res := MustSucceed(moveAs(
				ctx,
				[]lineplot.Action{setGroup(destG.Key)},
				destG.ResourceID(),
				project.ResourceID(proj.Key),
				dest.ResourceID(),
			))
			Expect(res.Actions).To(Equal([]lineplot.Action{setProject(dest.Key)}))
			moved := retrieveLinePlot(ctx, target.Key)
			Expect(moved.Group).To(Equal(&destG.Key))
			Expect(moved.Project).To(Equal(&dest.Key))
			Expect(rec.Snapshot()).To(HaveExactElements(HaveField(
				"Actions",
				[]lineplot.Action{setGroup(destG.Key), setProject(dest.Key)},
			)))
		},
	)
	It("Should move the line plot out of its group", func(ctx SpecContext) {
		inGroup(ctx)
		Expect(moveAs(ctx, []lineplot.Action{removeGroup}, g.ResourceID())).
			To(HaveField("Actions", BeEmpty()))
		Expect(retrieveLinePlot(ctx, target.Key).Group).To(BeNil())
	})
	It("Should move the line plot into another project", func(ctx SpecContext) {
		Expect(moveAs(
			ctx,
			[]lineplot.Action{setProject(dest.Key)},
			project.ResourceID(proj.Key),
			dest.ResourceID(),
		)).To(HaveField("Actions", BeEmpty()))
		Expect(retrieveLinePlot(ctx, target.Key).Project).To(Equal(&dest.Key))
	})
	It(
		"Should move the line plot out of its group when it moves into another project",
		func(ctx SpecContext) {
			inGroup(ctx)
			res := MustSucceed(moveAs(
				ctx,
				[]lineplot.Action{setProject(dest.Key)},
				g.ResourceID(),
				project.ResourceID(proj.Key),
				dest.ResourceID(),
			))
			Expect(res.Actions).To(Equal([]lineplot.Action{removeGroup}))
			moved := retrieveLinePlot(ctx, target.Key)
			Expect(moved.Group).To(BeNil())
			Expect(moved.Project).To(Equal(&dest.Key))
		},
	)
	It("Should refuse a move into a group that does not exist", func(ctx SpecContext) {
		missing := uuid.New()
		Expect(moveAs(ctx, []lineplot.Action{setGroup(missing)})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("group: group %s does not exist", missing)),
		))
		Expect(retrieveLinePlot(ctx, target.Key).Group).To(BeNil())
		Expect(rec.Snapshot()).To(BeEmpty())
	})
	DescribeTable(
		"Should refuse a move without update on a record it changes",
		func(
			ctx SpecContext,
			grouped bool,
			move func() ([]lineplot.Action, []resource.ID, resource.ID),
		) {
			if grouped {
				inGroup(ctx)
			}
			before := retrieveLinePlot(ctx, target.Key)
			acts, granted, refused := move()
			Expect(moveAs(ctx, acts, granted...)).Error().
				To(denied(u, access.ActionUpdate, refused))
			Expect(retrieveLinePlot(ctx, target.Key)).To(Equal(before))
			Expect(rec.Snapshot()).To(BeEmpty())
		},
		Entry(
			"into a group without the group",
			false,
			func() ([]lineplot.Action, []resource.ID, resource.ID) {
				return []lineplot.Action{setGroup(g.Key)}, nil, g.ResourceID()
			},
		),
		Entry(
			"into a group of another project without that project",
			false,
			func() ([]lineplot.Action, []resource.ID, resource.ID) {
				return []lineplot.Action{setGroup(destG.Key)},
					[]resource.ID{destG.ResourceID(), project.ResourceID(proj.Key)},
					dest.ResourceID()
			},
		),
		Entry(
			"into a group of another project without the source project",
			false,
			func() ([]lineplot.Action, []resource.ID, resource.ID) {
				return []lineplot.Action{setGroup(destG.Key)},
					[]resource.ID{destG.ResourceID(), dest.ResourceID()},
					project.ResourceID(proj.Key)
			},
		),
		Entry(
			"into another project without that project",
			false,
			func() ([]lineplot.Action, []resource.ID, resource.ID) {
				return []lineplot.Action{setProject(dest.Key)},
					[]resource.ID{project.ResourceID(proj.Key)},
					dest.ResourceID()
			},
		),
		Entry(
			"into another project without the source project",
			false,
			func() ([]lineplot.Action, []resource.ID, resource.ID) {
				return []lineplot.Action{setProject(dest.Key)},
					[]resource.ID{dest.ResourceID()},
					project.ResourceID(proj.Key)
			},
		),
		Entry(
			"out of a group without the group",
			true,
			func() ([]lineplot.Action, []resource.ID, resource.ID) {
				return []lineplot.Action{setProject(dest.Key)},
					[]resource.ID{project.ResourceID(proj.Key), dest.ResourceID()},
					g.ResourceID()
			},
		),
	)
})
