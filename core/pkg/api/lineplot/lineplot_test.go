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
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
)

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

// createProjectGroup creates a project group with no top parent.
func createProjectGroup(ctx SpecContext, name string) group.Group {
	GinkgoHelper()
	g := group.Group{Name: name, Scope: resource.TypeProject}
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
			g := createProjectGroup(ctx, "Set group denied")
			grant(ctx, access.ActionUpdate, u.ResourceID(), lineplot.ResourceID(lp.Key))
			Expect(apiSvc.SetGroup(
				AuthedCtx(ctx, u),
				nil,
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
			g := createProjectGroup(ctx, "Set group granted")
			grant(
				ctx,
				access.ActionUpdate,
				u.ResourceID(),
				lineplot.ResourceID(lp.Key),
				g.ResourceID(),
			)
			Expect(apiSvc.SetGroup(
				AuthedCtx(ctx, u),
				nil,
				apilineplot.SetGroupRequest{
					Group: &g.Key,
					Keys:  []lineplot.Key{lp.Key},
				},
			)).Error().ToNot(HaveOccurred())
			Expect(retrieveLinePlot(ctx, lp.Key).Group).To(Equal(&g.Key))
		},
	)
})

var _ = Describe("Create in a group", func() {
	It(
		"Should refuse the create without update on the group",
		func(ctx SpecContext) {
			u := newCreator(ctx)
			g := createProjectGroup(ctx, "Create denied")
			Expect(apiSvc.Create(AuthedCtx(ctx, u), nil, apilineplot.CreateRequest{
				LinePlots: []lineplot.LinePlot{{Project: &proj.Key, Name: "Denied", Group: &g.Key}},
			})).Error().To(MatchError(access.ErrDenied))
		},
	)
	It("Should create the line plot with update on the group", func(ctx SpecContext) {
		u := newCreator(ctx)
		g := createProjectGroup(ctx, "Create granted")
		grant(ctx, access.ActionUpdate, u.ResourceID(), g.ResourceID())
		res := MustSucceed(
			apiSvc.Create(AuthedCtx(ctx, u), nil, apilineplot.CreateRequest{
				LinePlots: []lineplot.LinePlot{{Project: &proj.Key, Name: "Granted", Group: &g.Key}},
			}),
		)
		Expect(retrieveLinePlot(ctx, res.LinePlots[0].Key).Group).To(Equal(&g.Key))
	})
})
