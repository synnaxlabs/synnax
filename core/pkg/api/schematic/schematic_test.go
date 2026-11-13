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
											Variant:     schematic.EdgeVariantPipe,
											StrokeWidth: 2,
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

	It("Should refuse a schematic whose key exists", func(ctx SpecContext) {
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
		})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("schematic %s already exists", stored.Key)),
		))
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

var _ = Describe("Dispatch of a move", func() {
	var (
		u      user.User
		target schematic.Schematic
		g      group.Group
		dest   project.Project
		destG  group.Group
		rec    *Recorder[schematic.Key, schematic.Action]
	)
	BeforeEach(func(ctx SpecContext) {
		u = newUser(ctx)
		target = createSchematic(ctx, "Moved")
		g = createGroup(ctx, "Source")
		dest = project.Project{Name: "Destination"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return projectSvc.NewWriter(tx).Create(ctx, &dest)
		})).To(Succeed())
		destG = group.Group{Name: "Destination", Parent: new(dest.ResourceID())}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &destG)
		})).To(Succeed())
		rec = &Recorder[schematic.Key, schematic.Action]{}
	})
	setGroup := func(k group.Key) schematic.Action {
		return schematic.NewSetAction(
			schematic.SetPayload{Path: []string{"group"}, Value: k},
		)
	}
	setProject := func(k project.Key) schematic.Action {
		return schematic.NewSetAction(
			schematic.SetPayload{Path: []string{"project"}, Value: k},
		)
	}
	removeGroup := schematic.NewRemoveAction(schematic.RemovePayload{Key: "group"})
	// inGroup moves the schematic into g.
	inGroup := func(ctx SpecContext) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			_, err := schematicSvc.NewWriter(tx).Apply(ctx, target.Key, setGroup(g.Key))
			return err
		})).To(Succeed())
	}
	// moveAs grants u update on the schematic and on objects, records the broadcasts
	// that follow, and dispatches acts as u.
	moveAs := func(
		ctx SpecContext,
		acts []schematic.Action,
		objects ...resource.ID,
	) (apischematic.DispatchResponse, error) {
		GinkgoHelper()
		grantUpdateOn(ctx, u.ResourceID(), append(objects, target.ResourceID())...)
		DeferCleanup(schematicSvc.OnAction(rec.Record))
		return dispatch(AuthedCtx(ctx, u), apischematic.DispatchRequest{
			Key:         target.Key,
			DispatchKey: "dk-move",
			Actions:     acts,
		})
	}
	It("Should move the schematic into a group", func(ctx SpecContext) {
		Expect(moveAs(ctx, []schematic.Action{setGroup(g.Key)}, g.ResourceID())).
			To(HaveField("Actions", BeEmpty()))
		Expect(retrieveSchematic(ctx, target.Key).Group).To(Equal(&g.Key))
		Expect(rec.Snapshot()).To(HaveExactElements(SatisfyAll(
			HaveField("DispatchKey", "dk-move"),
			HaveField("Actions", []schematic.Action{setGroup(g.Key)}),
		)))
	})
	It(
		"Should move the schematic into the project of a group in another project",
		func(ctx SpecContext) {
			res := MustSucceed(moveAs(
				ctx,
				[]schematic.Action{setGroup(destG.Key)},
				destG.ResourceID(),
				project.ResourceID(proj.Key),
				dest.ResourceID(),
			))
			Expect(res.Actions).To(Equal([]schematic.Action{setProject(dest.Key)}))
			moved := retrieveSchematic(ctx, target.Key)
			Expect(moved.Group).To(Equal(&destG.Key))
			Expect(moved.Project).To(Equal(&dest.Key))
			Expect(rec.Snapshot()).To(HaveExactElements(HaveField(
				"Actions",
				[]schematic.Action{setGroup(destG.Key), setProject(dest.Key)},
			)))
		},
	)
	It("Should move the schematic out of its group", func(ctx SpecContext) {
		inGroup(ctx)
		Expect(moveAs(ctx, []schematic.Action{removeGroup}, g.ResourceID())).
			To(HaveField("Actions", BeEmpty()))
		Expect(retrieveSchematic(ctx, target.Key).Group).To(BeNil())
	})
	It("Should move the schematic into another project", func(ctx SpecContext) {
		Expect(moveAs(
			ctx,
			[]schematic.Action{setProject(dest.Key)},
			project.ResourceID(proj.Key),
			dest.ResourceID(),
		)).To(HaveField("Actions", BeEmpty()))
		Expect(retrieveSchematic(ctx, target.Key).Project).To(Equal(&dest.Key))
	})
	It(
		"Should move the schematic out of its group when it moves into another project",
		func(ctx SpecContext) {
			inGroup(ctx)
			res := MustSucceed(moveAs(
				ctx,
				[]schematic.Action{setProject(dest.Key)},
				g.ResourceID(),
				project.ResourceID(proj.Key),
				dest.ResourceID(),
			))
			Expect(res.Actions).To(Equal([]schematic.Action{removeGroup}))
			moved := retrieveSchematic(ctx, target.Key)
			Expect(moved.Group).To(BeNil())
			Expect(moved.Project).To(Equal(&dest.Key))
		},
	)
	It("Should refuse a move into a group that does not exist", func(ctx SpecContext) {
		missing := uuid.New()
		Expect(
			moveAs(ctx, []schematic.Action{setGroup(missing)}),
		).Error().
			To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("group: group %s does not exist", missing)),
			))
		Expect(retrieveSchematic(ctx, target.Key).Group).To(BeNil())
		Expect(rec.Snapshot()).To(BeEmpty())
	})
	DescribeTable(
		"Should refuse a move without update on a record it changes",
		func(
			ctx SpecContext,
			grouped bool,
			move func() ([]schematic.Action, []resource.ID, resource.ID),
		) {
			if grouped {
				inGroup(ctx)
			}
			before := retrieveSchematic(ctx, target.Key)
			acts, granted, refused := move()
			Expect(moveAs(ctx, acts, granted...)).Error().
				To(denied(u, access.ActionUpdate, refused))
			Expect(retrieveSchematic(ctx, target.Key)).To(Equal(before))
			Expect(rec.Snapshot()).To(BeEmpty())
		},
		Entry(
			"into a group without the group",
			false,
			func() ([]schematic.Action, []resource.ID, resource.ID) {
				return []schematic.Action{setGroup(g.Key)}, nil, g.ResourceID()
			},
		),
		Entry(
			"into a group of another project without that project",
			false,
			func() ([]schematic.Action, []resource.ID, resource.ID) {
				return []schematic.Action{setGroup(destG.Key)},
					[]resource.ID{destG.ResourceID(), project.ResourceID(proj.Key)},
					dest.ResourceID()
			},
		),
		Entry(
			"into a group of another project without the source project",
			false,
			func() ([]schematic.Action, []resource.ID, resource.ID) {
				return []schematic.Action{setGroup(destG.Key)},
					[]resource.ID{destG.ResourceID(), dest.ResourceID()},
					project.ResourceID(proj.Key)
			},
		),
		Entry(
			"into another project without that project",
			false,
			func() ([]schematic.Action, []resource.ID, resource.ID) {
				return []schematic.Action{setProject(dest.Key)},
					[]resource.ID{project.ResourceID(proj.Key)},
					dest.ResourceID()
			},
		),
		Entry(
			"into another project without the source project",
			false,
			func() ([]schematic.Action, []resource.ID, resource.ID) {
				return []schematic.Action{setProject(dest.Key)},
					[]resource.ID{dest.ResourceID()},
					project.ResourceID(proj.Key)
			},
		),
		Entry(
			"out of a group without the group",
			true,
			func() ([]schematic.Action, []resource.ID, resource.ID) {
				return []schematic.Action{setProject(dest.Key)},
					[]resource.ID{project.ResourceID(proj.Key), dest.ResourceID()},
					g.ResourceID()
			},
		),
	)
})
