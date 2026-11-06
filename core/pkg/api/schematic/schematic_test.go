// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apischematic "github.com/synnaxlabs/synnax/pkg/api/schematic"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// scopedAction is a short local alias for the schematic's action envelope so
// the test files don't have to spell out the generic parameters at every use.
type scopedAction = actions.Scoped[schematic.Key, schematic.Action]

// createSchematic persists a fresh schematic owned by the suite project and
// returns it with its key populated. Writes commit immediately (nil tx) so
// access-control reads can observe the new ontology resource.
func createSchematic(ctx context.Context, name string) schematic.Schematic {
	GinkgoHelper()
	s := schematic.Schematic{Project: &proj.Key, Name: name}
	Expect(schematicSvc.NewWriter(nil).Create(ctx, &s)).To(Succeed())
	return s
}

var _ = Describe("Service", func() {
	Describe("Create", func() {
		It("Should stamp the dispatch key onto the create broadcast", func(
			ctx SpecContext,
		) {
			grantOn(
				ctx,
				author.ResourceID(),
				access.ActionCreate,
				resource.ID{Type: resource.TypeSchematic},
			)
			rec := &Recorder[schematic.Key, schematic.Action]{}
			DeferCleanup(schematicSvc.OnAction(rec.Record))
			res := MustSucceed(apiSvc.Create(
				AuthedCtx(ctx, author),
				db,
				apischematic.CreateRequest{
					Schematics: []schematic.Schematic{
						{Name: "broadcast", Project: &proj.Key},
					},
					DispatchKey: "dk-create",
				},
			))
			frames := rec.Snapshot()
			Expect(frames).To(HaveLen(1))
			Expect(frames[0].Key).To(Equal(res.Schematics[0].Key))
			Expect(frames[0].DispatchKey).To(Equal("dk-create"))
		})
	})

	Describe("Retrieve", func() {
		It(
			"Should omit missing keys from a multi-key retrieve instead of failing",
			func(ctx SpecContext) {
				s := createSchematic(ctx, "partial-survivor")
				missing := uuid.New()
				grantOn(ctx, author.ResourceID(), access.ActionRetrieve,
					schematic.ResourceID(s.Key), schematic.ResourceID(missing))
				res := MustSucceed(
					apiSvc.Retrieve(
						AuthedCtx(ctx, author),
						apischematic.RetrieveRequest{
							Keys:                []schematic.Key{s.Key, missing},
							IgnoreNotFoundError: true,
						},
					),
				)
				Expect(res.Schematics).To(HaveLen(1))
				Expect(res.Schematics[0].Key).To(Equal(s.Key))
			},
		)

		It(
			"Should return an empty result when every requested key is missing",
			func(ctx SpecContext) {
				res := MustSucceed(
					apiSvc.Retrieve(
						AuthedCtx(ctx, author),
						apischematic.RetrieveRequest{
							Keys:                []schematic.Key{uuid.New()},
							IgnoreNotFoundError: true,
						},
					),
				)
				Expect(res.Schematics).To(BeEmpty())
			},
		)
	})

	Describe("SetProject", func() {
		It(
			"Should reject the request when the subject has no update policy",
			func(ctx SpecContext) {
				s := createSchematic(ctx, "move-denied")
				Expect(apiSvc.SetProject(
					AuthedCtx(ctx, author),
					nil,
					apischematic.SetProjectRequest{
						Project: proj.Key,
						Keys:    []schematic.Key{s.Key},
					},
				)).Error().To(MatchError(access.ErrDenied))
			},
		)

		It("Should move the schematics into the project", func(ctx SpecContext) {
			dest := project.Project{Name: "destination"}
			Expect(projectSvc.NewWriter(nil).Create(ctx, &dest)).To(Succeed())
			s := createSchematic(ctx, "moved")
			grantUpdateOn(ctx, author.ResourceID(), s.ResourceID())
			Expect(apiSvc.SetProject(
				AuthedCtx(ctx, author),
				nil,
				apischematic.SetProjectRequest{
					Project: dest.Key,
					Keys:    []schematic.Key{s.Key},
				},
			)).Error().To(Succeed())
			var res schematic.Schematic
			Expect(schematicSvc.NewRetrieve().
				Where(schematic.MatchKeys(s.Key)).
				Entry(&res).Exec(ctx, nil)).To(Succeed())
			Expect(res.Project).To(HaveValue(Equal(dest.Key)))
		})
	})

	Describe("Dispatch", func() {
		Describe("access control", func() {
			It(
				"Should reject the request with access.ErrDenied when the subject has no policy",
				func(ctx SpecContext) {
					s := createSchematic(ctx, "no-policy")
					Expect(
						apiSvc.Dispatch(
							AuthedCtx(ctx, author),
							apischematic.DispatchRequest{
								Key:         s.Key,
								DispatchKey: "sess-1",
								Actions: []schematic.Action{
									schematic.NewRemoveNodeAction(
										schematic.RemoveNodePayload{
											Key: "n1",
										},
									),
								},
							},
						),
					).Error().
						To(MatchError(access.ErrDenied))
				},
			)

			It(
				"Should accept the request when the subject's policy permits update on the target schematic",
				func(ctx SpecContext) {
					s := createSchematic(ctx, "with-policy")
					grantUpdateOn(ctx, author.ResourceID(), s.ResourceID())
					Expect(
						apiSvc.Dispatch(
							AuthedCtx(ctx, author),
							apischematic.DispatchRequest{
								Key:         s.Key,
								DispatchKey: "sess-1",
								Actions: []schematic.Action{
									schematic.NewSetNodeAction(schematic.SetNodePayload{
										Node: schematic.Node{
											Key:      "n1",
											Position: spatial.XY{X: 1, Y: 2},
										},
									}),
								},
							},
						),
					).Error().
						To(Succeed())
					var res schematic.Schematic
					Expect(schematicSvc.NewRetrieve().
						Where(schematic.MatchKeys(s.Key)).
						Entry(&res).Exec(ctx, nil)).To(Succeed())
					Expect(res.Nodes).To(HaveLen(1))
					Expect(res.Nodes[0].Position).To(Equal(spatial.XY{X: 1, Y: 2}))
				},
			)

			It(
				"Should reject when the subject's policy targets a different schematic",
				func(ctx SpecContext) {
					a := createSchematic(ctx, "policy-target")
					b := createSchematic(ctx, "no-policy-target")
					grantUpdateOn(ctx, author.ResourceID(), a.ResourceID())
					Expect(
						apiSvc.Dispatch(
							AuthedCtx(ctx, author),
							apischematic.DispatchRequest{
								Key:         b.Key,
								DispatchKey: "sess-1",
								Actions: []schematic.Action{
									schematic.NewRemoveNodeAction(
										schematic.RemoveNodePayload{
											Key: "n1",
										},
									),
								},
							},
						),
					).Error().
						To(MatchError(access.ErrDenied))
				},
			)
		})

		Describe("delegation to Writer.Dispatch", func() {
			It(
				"Should apply a multi-action sequence to the target schematic",
				func(ctx SpecContext) {
					s := createSchematic(ctx, "multi-action")
					grantUpdateOn(ctx, author.ResourceID(), s.ResourceID())
					Expect(
						apiSvc.Dispatch(
							AuthedCtx(ctx, author),
							apischematic.DispatchRequest{
								Key:         s.Key,
								DispatchKey: "sess-1",
								Actions: []schematic.Action{
									schematic.NewSetNodeAction(schematic.SetNodePayload{
										Node: schematic.Node{
											Key:      "n1",
											Position: spatial.XY{X: 1, Y: 2},
										},
									}),
									schematic.NewSetNodeAction(schematic.SetNodePayload{
										Node: schematic.Node{
											Key:      "n2",
											Position: spatial.XY{X: 3, Y: 4},
										},
									}),
									schematic.NewAddEdgeAction(
										schematic.AddEdgePayload{Edge: schematic.Edge{
											Key: "e1",
											Source: schematic.Handle{
												Node:  "n1",
												Param: "out",
											},
											Target: schematic.Handle{
												Node:  "n2",
												Param: "in",
											},
										}},
									),
								},
							},
						),
					).Error().
						To(Succeed())
					var res schematic.Schematic
					Expect(schematicSvc.NewRetrieve().
						Where(schematic.MatchKeys(s.Key)).
						Entry(&res).Exec(ctx, nil)).To(Succeed())
					Expect(res.Nodes).To(HaveLen(2))
					Expect(res.Edges).To(HaveLen(1))
				},
			)

			It(
				"Should bubble up validate.ErrValidation when dispatching to a snapshot",
				func(ctx SpecContext) {
					s := createSchematic(ctx, "snap-source")
					var snap schematic.Schematic
					Expect(
						schematicSvc.NewWriter(nil).
							Copy(ctx, s.Key, "snap", true, nil, &snap),
					).To(Succeed())
					grantUpdateOn(ctx, author.ResourceID(), snap.ResourceID())
					Expect(
						apiSvc.Dispatch(
							AuthedCtx(ctx, author),
							apischematic.DispatchRequest{
								Key:         snap.Key,
								DispatchKey: "sess-1",
								Actions: []schematic.Action{
									schematic.NewRemoveNodeAction(
										schematic.RemoveNodePayload{
											Key: "n1",
										},
									),
								},
							},
						),
					).Error().
						To(MatchError(validate.ErrValidation))
				},
			)

			It(
				"Should bubble up query.ErrNotFound when the target schematic does not exist",
				func(ctx SpecContext) {
					missing := uuid.New()
					grantUpdateOn(
						ctx,
						author.ResourceID(),
						schematic.ResourceID(missing),
					)
					Expect(
						apiSvc.Dispatch(
							AuthedCtx(ctx, author),
							apischematic.DispatchRequest{
								Key:         missing,
								DispatchKey: "sess-1",
								Actions: []schematic.Action{
									schematic.NewRemoveNodeAction(
										schematic.RemoveNodePayload{
											Key: "n1",
										},
									),
								},
							},
						),
					).Error().
						To(MatchError(query.ErrNotFound))
				},
			)
		})

		Describe("subject identity propagation", func() {
			It(
				"Should pass the DispatchKey verbatim into the action observer",
				func(ctx SpecContext) {
					s := createSchematic(ctx, "session-propagation")
					grantUpdateOn(ctx, author.ResourceID(), s.ResourceID())
					seen := make(chan scopedAction, 1)
					disconnect := schematicSvc.OnAction(
						func(_ context.Context, sa scopedAction) {
							seen <- sa
						},
					)
					DeferCleanup(disconnect)
					Expect(
						apiSvc.Dispatch(
							AuthedCtx(ctx, author),
							apischematic.DispatchRequest{
								Key:         s.Key,
								DispatchKey: "session-marker-xyz",
								Actions: []schematic.Action{
									schematic.NewSetNodeAction(schematic.SetNodePayload{
										Node: schematic.Node{Key: "n1"},
									}),
								},
							},
						),
					).Error().
						To(Succeed())
					var got scopedAction
					Eventually(seen).Should(Receive(&got))
					Expect(got.Key).To(Equal(s.Key))
					Expect(got.DispatchKey).To(Equal("session-marker-xyz"))
					Expect(got.Seq).To(BeNumerically(">", uint64(0)))
					Expect(got.Actions).To(HaveLen(1))
				},
			)
		})
	})
})

// createGroup creates a group in the suite project.
func createGroup(ctx SpecContext, name string) group.Group {
	GinkgoHelper()
	g := group.Group{Name: name, Parent: new(project.ResourceID(proj.Key))}
	Expect(groupSvc.NewWriter(nil).Create(ctx, &g)).To(Succeed())
	return g
}

func retrieveSchematic(ctx SpecContext, key schematic.Key) schematic.Schematic {
	GinkgoHelper()
	var s schematic.Schematic
	Expect(schematicSvc.NewRetrieve().
		Where(schematic.MatchKeys(key)).
		Entry(&s).
		Exec(ctx, nil)).To(Succeed())
	return s
}

var _ = Describe("SetGroup", func() {
	It(
		"Should refuse the move without update on the destination group",
		func(ctx SpecContext) {
			s := createSchematic(ctx, "Set group denied")
			g := createGroup(ctx, "Set group denied")
			grantUpdateOn(ctx, author.ResourceID(), schematic.ResourceID(s.Key))
			Expect(apiSvc.SetGroup(
				AuthedCtx(ctx, author),
				db,
				apischematic.SetGroupRequest{
					Group: &g.Key,
					Keys:  []schematic.Key{s.Key},
				},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(retrieveSchematic(ctx, s.Key).Group).To(BeNil())
		},
	)
	It(
		"Should move the schematic with update on the destination group",
		func(ctx SpecContext) {
			s := createSchematic(ctx, "Set group granted")
			g := createGroup(ctx, "Set group granted")
			grantUpdateOn(
				ctx,
				author.ResourceID(),
				schematic.ResourceID(s.Key),
				g.ResourceID(),
			)
			Expect(apiSvc.SetGroup(
				AuthedCtx(ctx, author),
				db,
				apischematic.SetGroupRequest{
					Group: &g.Key,
					Keys:  []schematic.Key{s.Key},
				},
			)).Error().ToNot(HaveOccurred())
			Expect(retrieveSchematic(ctx, s.Key).Group).To(Equal(&g.Key))
		},
	)
})

var _ = Describe("Create in a group", func() {
	BeforeEach(func(ctx SpecContext) {
		grantOn(
			ctx,
			author.ResourceID(),
			access.ActionCreate,
			resource.ID{Type: resource.TypeSchematic},
		)
	})
	It(
		"Should refuse the create without update on the group",
		func(ctx SpecContext) {
			g := createGroup(ctx, "Create denied")
			Expect(apiSvc.Create(
				AuthedCtx(ctx, author), db,
				apischematic.CreateRequest{
					Schematics: []schematic.Schematic{
						{Project: &proj.Key, Name: "Denied", Group: &g.Key},
					},
				},
			)).Error().To(MatchError(access.ErrDenied))
		},
	)
	It(
		"Should keep an existing schematic in its group without update on a group",
		func(ctx SpecContext) {
			g := createGroup(ctx, "Create keeps group")
			sc := schematic.Schematic{Project: &proj.Key, Name: "Kept", Group: &g.Key}
			Expect(schematicSvc.NewWriter(nil).Create(ctx, &sc)).To(Succeed())
			ignored := createGroup(ctx, "Create ignores group")
			sc.Name, sc.Group = "Renamed", &ignored.Key
			Expect(apiSvc.Create(
				AuthedCtx(ctx, author), db,
				apischematic.CreateRequest{
					Schematics: []schematic.Schematic{sc},
				},
			)).Error().ToNot(HaveOccurred())
			stored := retrieveSchematic(ctx, sc.Key)
			Expect(stored.Name).To(Equal("Renamed"))
			Expect(stored.Group).To(Equal(&g.Key))
		},
	)
	It("Should create the schematic with update on the group", func(ctx SpecContext) {
		g := createGroup(ctx, "Create granted")
		grantUpdateOn(ctx, author.ResourceID(), g.ResourceID())
		res := MustSucceed(apiSvc.Create(
			AuthedCtx(ctx, author), db,
			apischematic.CreateRequest{
				Schematics: []schematic.Schematic{
					{Project: &proj.Key, Name: "Granted", Group: &g.Key},
				},
			},
		))
		Expect(retrieveSchematic(ctx, res.Schematics[0].Key).Group).To(Equal(&g.Key))
	})
})
