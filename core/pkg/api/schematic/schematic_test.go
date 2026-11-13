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
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/gorp"
	apischematic "github.com/synnaxlabs/synnax/pkg/api/schematic"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/spatial"
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

// scopedAction is a short local alias for the schematic's action envelope so
// the test files don't have to spell out the generic parameters at every use.
type scopedAction = actions.Scoped[schematic.Key, schematic.Action]

// createSchematic persists a fresh schematic owned by the suite project and returns it
// with its key populated. The write commits at once so access-control reads can observe
// the new schematic.
func createSchematic(ctx context.Context, name string) schematic.Schematic {
	GinkgoHelper()
	s := schematic.Schematic{Project: &proj.Key, Name: name}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return schematicSvc.NewWriter(tx).Create(ctx, &s)
	})).To(Succeed())
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
			grantUpdateOn(ctx, author.ResourceID(), project.ResourceID(proj.Key))
			rec := &Recorder[schematic.Key, schematic.Action]{}
			DeferCleanup(schematicSvc.OnAction(rec.Record))
			res := MustSucceed(create(
				AuthedCtx(ctx, author),
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
					retrieve(
						AuthedCtx(ctx, author),
						apischematic.RetrieveRequest{
							Where: &schematic.Where{
								Key: &filter.Set[schematic.Key]{
									In: []schematic.Key{s.Key, missing},
								},
							},
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
				missing := uuid.New()
				grantOn(ctx, author.ResourceID(), access.ActionRetrieve,
					schematic.ResourceID(missing))
				res := MustSucceed(
					retrieve(
						AuthedCtx(ctx, author),
						apischematic.RetrieveRequest{
							Where: &schematic.Where{
								Key: &filter.Set[schematic.Key]{
									In: []schematic.Key{missing},
								},
							},
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
				Expect(setProject(
					AuthedCtx(ctx, author),
					apischematic.SetProjectRequest{
						Project: proj.Key,
						Keys:    []schematic.Key{s.Key},
					},
				)).Error().To(MatchError(access.ErrDenied))
			},
		)

		It(
			"Should reject the request without update on the destination project",
			func(ctx SpecContext) {
				dest := project.Project{Name: "denied destination"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return projectSvc.NewWriter(tx).Create(ctx, &dest)
				})).To(Succeed())
				s := createSchematic(ctx, "move-denied")
				grantUpdateOn(
					ctx,
					author.ResourceID(),
					s.ResourceID(),
					project.ResourceID(proj.Key),
				)
				Expect(setProject(
					AuthedCtx(ctx, author),
					apischematic.SetProjectRequest{
						Project: dest.Key,
						Keys:    []schematic.Key{s.Key},
					},
				)).Error().To(MatchError(access.ErrDenied))
				Expect(retrieveSchematic(ctx, s.Key).Project).To(Equal(&proj.Key))
			},
		)

		It("Should move the schematics into the project", func(ctx SpecContext) {
			dest := project.Project{Name: "destination"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return projectSvc.NewWriter(tx).Create(ctx, &dest)
			})).To(Succeed())
			s := createSchematic(ctx, "moved")
			grantUpdateOn(
				ctx,
				author.ResourceID(),
				s.ResourceID(),
				project.ResourceID(proj.Key),
				project.ResourceID(dest.Key),
			)
			Expect(setProject(
				AuthedCtx(ctx, author),
				apischematic.SetProjectRequest{
					Project: dest.Key,
					Keys:    []schematic.Key{s.Key},
				},
			)).Error().To(Succeed())
			var res schematic.Schematic
			Expect(schematicSvc.NewRetrieve().
				Where(schematic.MatchKeys(s.Key)).
				Entry(&res).Exec(ctx, db)).To(Succeed())
			Expect(res.Project).To(HaveValue(Equal(dest.Key)))
		})

		It(
			"Should refuse a move out of a group the subject cannot update",
			func(ctx SpecContext) {
				s := createSchematic(ctx, "grouped")
				g := createGroup(ctx, "grouped")
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return schematicSvc.NewWriter(tx).SetGroup(ctx, &g.Key, s.Key)
				})).
					To(Succeed())
				grantUpdateOn(
					ctx,
					author.ResourceID(),
					s.ResourceID(),
					project.ResourceID(proj.Key),
				)
				Expect(setProject(
					AuthedCtx(ctx, author),
					apischematic.SetProjectRequest{
						Project: proj.Key,
						Keys:    []schematic.Key{s.Key},
					},
				)).Error().To(MatchError(access.ErrDenied))
				Expect(retrieveSchematic(ctx, s.Key).Group).To(Equal(&g.Key))
			},
		)
	})

	Describe("Dispatch", func() {
		Describe("access control", func() {
			It(
				"Should reject the request with access.ErrDenied when the subject has no policy",
				func(ctx SpecContext) {
					s := createSchematic(ctx, "no-policy")
					Expect(
						dispatch(
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
						dispatch(
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
						Entry(&res).Exec(ctx, db)).To(Succeed())
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
						dispatch(
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

		Describe("delegation to Writer.Apply", func() {
			It(
				"Should apply a multi-action sequence to the target schematic",
				func(ctx SpecContext) {
					s := createSchematic(ctx, "multi-action")
					grantUpdateOn(ctx, author.ResourceID(), s.ResourceID())
					Expect(
						dispatch(
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
						Entry(&res).Exec(ctx, db)).To(Succeed())
					Expect(res.Nodes).To(HaveLen(2))
					Expect(res.Edges).To(HaveLen(1))
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
						dispatch(
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
						dispatch(
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
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return groupSvc.NewWriter(tx).Create(ctx, &g)
	})).To(Succeed())
	return g
}

func retrieveSchematic(ctx SpecContext, key schematic.Key) schematic.Schematic {
	GinkgoHelper()
	var s schematic.Schematic
	Expect(schematicSvc.NewRetrieve().
		Where(schematic.MatchKeys(key)).
		Entry(&s).
		Exec(ctx, db)).To(Succeed())
	return s
}

var _ = Describe("SetGroup", func() {
	It(
		"Should refuse the move without update on the destination group",
		func(ctx SpecContext) {
			s := createSchematic(ctx, "Set group denied")
			g := createGroup(ctx, "Set group denied")
			grantUpdateOn(ctx, author.ResourceID(), schematic.ResourceID(s.Key))
			Expect(setGroup(
				AuthedCtx(ctx, author),
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
			Expect(setGroup(
				AuthedCtx(ctx, author),
				apischematic.SetGroupRequest{
					Group: &g.Key,
					Keys:  []schematic.Key{s.Key},
				},
			)).Error().ToNot(HaveOccurred())
			Expect(retrieveSchematic(ctx, s.Key).Group).To(Equal(&g.Key))
		},
	)
	Context("Into a group under another project", func() {
		var (
			u    user.User
			x    schematic.Schematic
			dest project.Project
			g    group.Group
		)
		BeforeEach(func(ctx SpecContext) {
			u = newUser(ctx)
			x = createSchematic(ctx, "Cross project")
			dest = project.Project{Name: "Destination"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return projectSvc.NewWriter(tx).Create(ctx, &dest)
			})).To(Succeed())
			g = group.Group{Name: "Destination", Parent: new(dest.ResourceID())}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return groupSvc.NewWriter(tx).Create(ctx, &g)
			})).To(Succeed())
		})
		move := func(ctx SpecContext) (struct{}, error) {
			return setGroup(
				AuthedCtx(ctx, u),
				apischematic.SetGroupRequest{
					Group: &g.Key,
					Keys:  []schematic.Key{x.Key},
				},
			)
		}
		DescribeTable(
			"Should refuse the move without update on both projects",
			func(ctx SpecContext, projects func() []resource.ID) {
				objects := projects()
				grantUpdateOn(ctx, u.ResourceID(), append(
					objects,
					schematic.ResourceID(x.Key),
					g.ResourceID(),
				)...)
				Expect(move(ctx)).Error().To(MatchError(access.ErrDenied))
				Expect(retrieveSchematic(ctx, x.Key).Project).To(Equal(&proj.Key))
			},
			Entry("without the destination", func() []resource.ID {
				return []resource.ID{project.ResourceID(proj.Key)}
			}),
			Entry("without the source", func() []resource.ID {
				return []resource.ID{dest.ResourceID()}
			}),
		)
		It(
			"Should move the schematic with update on both projects",
			func(ctx SpecContext) {
				objects := []resource.ID{
					project.ResourceID(proj.Key),
					dest.ResourceID(),
				}
				grantUpdateOn(ctx, u.ResourceID(), append(
					objects,
					schematic.ResourceID(x.Key),
					g.ResourceID(),
				)...)
				MustSucceed(move(ctx))
				Expect(retrieveSchematic(ctx, x.Key).Project).To(Equal(&dest.Key))
			},
		)
	})
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
			Expect(create(
				AuthedCtx(ctx, author),
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
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return schematicSvc.NewWriter(tx).Create(ctx, &sc)
			})).To(Succeed())
			ignored := createGroup(ctx, "Create ignores group")
			sc.Name, sc.Group = "Renamed", &ignored.Key
			grantUpdateOn(ctx, author.ResourceID(), sc.ResourceID())
			Expect(create(
				AuthedCtx(ctx, author),
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
		grantUpdateOn(
			ctx,
			author.ResourceID(),
			g.ResourceID(),
			project.ResourceID(proj.Key),
		)
		res := MustSucceed(create(
			AuthedCtx(ctx, author),
			apischematic.CreateRequest{
				Schematics: []schematic.Schematic{
					{Project: &proj.Key, Name: "Granted", Group: &g.Key},
				},
			},
		))
		Expect(retrieveSchematic(ctx, res.Schematics[0].Key).Group).To(Equal(&g.Key))
	})
	Context("Under another project", func() {
		var (
			u    user.User
			dest project.Project
			g    group.Group
		)
		BeforeEach(func(ctx SpecContext) {
			u = newUser(ctx)
			grantOn(
				ctx,
				u.ResourceID(),
				access.ActionCreate,
				resource.ID{Type: resource.TypeSchematic},
			)
			grantUpdateOn(ctx, u.ResourceID(), proj.ResourceID())
			dest = project.Project{Name: "Create destination"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return projectSvc.NewWriter(tx).Create(ctx, &dest)
			})).To(Succeed())
			g = group.Group{Name: "Create destination", Parent: new(dest.ResourceID())}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return groupSvc.NewWriter(tx).Create(ctx, &g)
			})).To(Succeed())
		})
		create := func(ctx SpecContext, carried *project.Key) (apischematic.CreateResponse, error) {
			return create(AuthedCtx(ctx, u), apischematic.CreateRequest{
				Schematics: []schematic.Schematic{
					{Project: carried, Name: "Elsewhere", Group: &g.Key},
				},
			})
		}
		DescribeTable(
			"Should refuse a create in a group without update on the group's project",
			func(ctx SpecContext, carried func() *project.Key) {
				grantUpdateOn(ctx, u.ResourceID(), g.ResourceID())
				Expect(create(ctx, carried())).Error().To(MatchError(access.ErrDenied))
			},
			Entry(
				"with the project it carries",
				func() *project.Key { return &proj.Key },
			),
			Entry("with no project", func() *project.Key { return nil }),
		)
		It(
			"Should create the schematic with update on the group's project",
			func(ctx SpecContext) {
				grantUpdateOn(ctx, u.ResourceID(), g.ResourceID(), dest.ResourceID())
				res := MustSucceed(create(ctx, nil))
				Expect(
					retrieveSchematic(ctx, res.Schematics[0].Key).Project,
				).To(Equal(&dest.Key))
			},
		)
	})
})

var _ = Describe("Owner access", func() {
	It("Should refuse a create without update on the owner", func(ctx SpecContext) {
		u := newUser(ctx)
		grantOn(
			ctx,
			u.ResourceID(),
			access.ActionCreate,
			resource.ID{Type: resource.TypeSchematic},
		)
		Expect(create(
			AuthedCtx(ctx, u),
			apischematic.CreateRequest{
				Schematics: []schematic.Schematic{{Project: &proj.Key, Name: "Denied"}},
			},
		)).Error().To(MatchError(access.ErrDenied))
	})

	It("Should refuse a create without create on the schematic", func(
		ctx SpecContext,
	) {
		u := newUser(ctx)
		grantUpdateOn(ctx, u.ResourceID(), project.ResourceID(proj.Key))
		sc := schematic.Schematic{Key: uuid.New(), Project: &proj.Key, Name: "Denied"}
		Expect(create(AuthedCtx(ctx, u), apischematic.CreateRequest{
			Schematics: []schematic.Schematic{sc},
		})).Error().To(denied(u, access.ActionCreate, sc.ResourceID()))
		Expect(schematicSvc.NewRetrieve().
			Where(schematic.MatchKeys(sc.Key)).
			Entry(&schematic.Schematic{}).
			Exec(ctx, db)).To(MatchError(query.ErrNotFound))
	})

	It("Should refuse to overwrite a schematic without update on it", func(
		ctx SpecContext,
	) {
		u := newUser(ctx)
		grantOn(
			ctx,
			u.ResourceID(),
			access.ActionCreate,
			resource.ID{Type: resource.TypeSchematic},
		)
		stored := createSchematic(ctx, "Stored")
		Expect(create(AuthedCtx(ctx, u), apischematic.CreateRequest{
			Schematics: []schematic.Schematic{
				{Key: stored.Key, Name: "Overwritten", Project: &proj.Key},
			},
		})).Error().To(denied(u, access.ActionUpdate, stored.ResourceID()))
		Expect(retrieveSchematic(ctx, stored.Key).Name).To(Equal("Stored"))
	})

	Describe("Copy", func() {
		var (
			u   user.User
			src schematic.Schematic
		)
		BeforeEach(func(ctx SpecContext) {
			u = newUser(ctx)
			src = createSchematic(ctx, "Copy source")
			grantOn(
				ctx,
				u.ResourceID(),
				access.ActionCreate,
				resource.ID{Type: resource.TypeSchematic},
			)
		})
		copyAs := func(ctx SpecContext) (apischematic.CopyResponse, error) {
			return copySchematic(AuthedCtx(ctx, u), apischematic.CopyRequest{
				Key:  src.Key,
				Name: "Copy",
			})
		}
		It("Should refuse a copy without update on its owner", func(ctx SpecContext) {
			grantOn(ctx, u.ResourceID(), access.ActionRetrieve, src.ResourceID())
			Expect(copyAs(ctx)).Error().
				To(denied(u, access.ActionUpdate, project.ResourceID(proj.Key)))
		})
		It(
			"Should refuse a copy without retrieve on the source",
			func(ctx SpecContext) {
				grantUpdateOn(ctx, u.ResourceID(), project.ResourceID(proj.Key))
				Expect(copyAs(ctx)).Error().
					To(denied(u, access.ActionRetrieve, src.ResourceID()))
			},
		)
		It("Should copy with update on its owner", func(ctx SpecContext) {
			grantOn(ctx, u.ResourceID(), access.ActionRetrieve, src.ResourceID())
			grantUpdateOn(ctx, u.ResourceID(), project.ResourceID(proj.Key))
			res := MustSucceed(copyAs(ctx))
			Expect(retrieveSchematic(ctx, res.Schematic.Key).Name).To(Equal("Copy"))
		})
	})
})

var _ = Describe("CopyRequest", func() {
	It("Should name the schematic it copies as its source", func() {
		key := uuid.New()
		Expect(apischematic.CopyRequest{Key: key}.Sources()).
			To(Equal([]resource.ID{schematic.ResourceID(key)}))
	})
})

var _ = Describe("Delete", func() {
	var (
		u  user.User
		sc schematic.Schematic
	)
	BeforeEach(func(ctx SpecContext) {
		u = newUser(ctx)
		sc = createSchematic(ctx, "Deleted")
	})
	deleteAs := func(ctx SpecContext) (struct{}, error) {
		return del(AuthedCtx(ctx, u), apischematic.DeleteRequest{
			Keys: []schematic.Key{sc.Key},
		})
	}
	schematicID := func() resource.ID { return sc.ResourceID() }
	projectID := func() resource.ID { return project.ResourceID(proj.Key) }
	DescribeTable(
		"Should refuse the delete and keep the schematic",
		func(
			ctx SpecContext,
			granted access.Action,
			grantedOn func() resource.ID,
			refused access.Action,
			refusedOn func() resource.ID,
		) {
			grantOn(ctx, u.ResourceID(), granted, grantedOn())
			Expect(deleteAs(ctx)).Error().To(denied(u, refused, refusedOn()))
			Expect(retrieveSchematic(ctx, sc.Key).Key).To(Equal(sc.Key))
		},
		Entry(
			"without delete on the schematic",
			access.ActionUpdate,
			projectID,
			access.ActionDelete,
			schematicID,
		),
		Entry(
			"without update on its project",
			access.ActionDelete,
			schematicID,
			access.ActionUpdate,
			projectID,
		),
	)
	It(
		"Should delete the schematic with delete on it and update on its project",
		func(ctx SpecContext) {
			grantOn(ctx, u.ResourceID(), access.ActionDelete, schematicID())
			grantUpdateOn(ctx, u.ResourceID(), projectID())
			MustSucceed(deleteAs(ctx))
			Expect(schematicSvc.NewRetrieve().
				Where(schematic.MatchKeys(sc.Key)).
				Entry(&schematic.Schematic{}).
				Exec(ctx, db)).To(MatchError(query.ErrNotFound))
		},
	)
})
