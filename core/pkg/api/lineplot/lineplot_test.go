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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apilineplot "github.com/synnaxlabs/synnax/pkg/api/lineplot"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Retrieve", func() {
	It("Should reject the request when retrieve is not granted on a match", func(
		ctx SpecContext,
	) {
		r := lineplot.LinePlot{Project: &proj.Key, Name: "unretrievable"}
		Expect(svc.LinePlot.NewWriter(nil).Create(ctx, &r)).To(Succeed())
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
			apiSvc.Create(AuthedCtx(ctx, u), nil, apilineplot.CreateRequest{
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
})

// createGroup creates a group in the suite project.
func createGroup(ctx SpecContext, name string) group.Group {
	GinkgoHelper()
	g := group.Group{Name: name, Parent: new(project.ResourceID(proj.Key))}
	Expect(svc.Group.NewWriter(nil).Create(ctx, &g)).To(Succeed())
	return g
}

func createLinePlot(ctx SpecContext, name string) lineplot.LinePlot {
	GinkgoHelper()
	lp := lineplot.LinePlot{Project: &proj.Key, Name: name}
	Expect(svc.LinePlot.NewWriter(nil).Create(ctx, &lp)).To(Succeed())
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
			Expect(apiSvc.SetGroup(
				AuthedCtx(ctx, u),
				db,
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
			Expect(apiSvc.SetGroup(
				AuthedCtx(ctx, u),
				db,
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
			Expect(svc.Project.NewWriter(nil).Create(ctx, &dest)).To(Succeed())
			g = group.Group{Name: "Destination", Parent: new(dest.ResourceID())}
			Expect(svc.Group.NewWriter(nil).Create(ctx, &g)).To(Succeed())
		})
		move := func(ctx SpecContext) (struct{}, error) {
			return apiSvc.SetGroup(
				AuthedCtx(ctx, u),
				db,
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
			Expect(apiSvc.Create(AuthedCtx(ctx, u), nil, apilineplot.CreateRequest{
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
			apiSvc.Create(AuthedCtx(ctx, u), nil, apilineplot.CreateRequest{
				LinePlots: []lineplot.LinePlot{
					{Project: &proj.Key, Name: "Granted", Group: &g.Key},
				},
			}),
		)
		Expect(retrieveLinePlot(ctx, res.LinePlots[0].Key).Group).To(Equal(&g.Key))
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
		Expect(apiSvc.Create(AuthedCtx(ctx, u), nil, apilineplot.CreateRequest{
			LinePlots: []lineplot.LinePlot{{Project: &proj.Key, Name: "Denied"}},
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
	setProject := func(ctx SpecContext, u user.User, to project.Key, key lineplot.Key) error {
		GinkgoHelper()
		_, err := apiSvc.SetProject(
			AuthedCtx(ctx, u),
			nil,
			apilineplot.SetProjectRequest{
				Project: to,
				Keys:    []lineplot.Key{key},
			},
		)
		return err
	}
	It(
		"Should refuse the move without update on the destination project",
		func(ctx SpecContext) {
			u := newCreator(ctx)
			e := createLinePlot(ctx, "Set project denied")
			dest := createProject(ctx)
			grant(ctx, access.ActionUpdate, u.ResourceID(), lineplot.ResourceID(e.Key))
			Expect(setProject(ctx, u, dest.Key, e.Key)).To(MatchError(access.ErrDenied))
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
			Expect(setProject(ctx, u, dest.Key, e.Key)).To(MatchError(access.ErrDenied))
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
			Expect(setProject(ctx, u, dest.Key, e.Key)).To(Succeed())
			Expect(retrieveLinePlot(ctx, e.Key).Project).To(Equal(&dest.Key))
		},
	)
})
