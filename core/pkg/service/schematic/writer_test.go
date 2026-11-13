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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/union"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create a Schematic", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			s := schematic.Schematic{Project: &proj.Key, Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
			Expect(s.Key).ToNot(Equal(uuid.Nil()))
		})
		It(
			"Should return a validation error when the name is empty",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				s := schematic.Schematic{Project: &proj.Key}
				Expect(svc.NewWriter(tx).Create(ctx, &s)).
					To(MatchError(ContainSubstring("name: required")))
			},
		)

		It(
			"Should broadcast the create with the writer's dispatch key after commit",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				rec := &Recorder[schematic.Key, schematic.Action]{}
				DeferCleanup(svc.OnAction(rec.Record))
				v := schematic.Schematic{Project: &proj.Key, Name: "broadcast"}
				Expect(
					svc.NewWriter(tx).WithDispatchKey("dk-1").Create(ctx, &v),
				).To(Succeed())
				Expect(rec.Snapshot()).To(BeEmpty())
				Expect(tx.Commit(ctx)).To(Succeed())
				seen := rec.Snapshot()
				Expect(seen).To(HaveLen(1))
				Expect(seen[0].Key).To(Equal(v.Key))
				Expect(seen[0].DispatchKey).To(Equal("dk-1"))
				Expect(seen[0].Actions).To(HaveLen(1))
				Expect(seen[0].Actions[0].Type).To(Equal(schematic.ActionTypeCreate))
			},
		)

		It(
			"Should not broadcast a create whose transaction closes uncommitted",
			func(ctx SpecContext) {
				rec := &Recorder[schematic.Key, schematic.Action]{}
				DeferCleanup(svc.OnAction(rec.Record))
				uncommitted := db.OpenTx()
				v := schematic.Schematic{Project: &proj.Key, Name: "discarded"}
				Expect(
					svc.NewWriter(uncommitted).Create(ctx, &v),
				).To(Succeed())
				Expect(uncommitted.Close()).To(Succeed())
				Expect(rec.Snapshot()).To(BeEmpty())
			},
		)
	})
	Describe("CreateMany", func() {
		It("Should create multiple schematics", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			schematics := []schematic.Schematic{
				{Name: "schematic-1", Project: &proj.Key},
				{Name: "schematic-2", Project: &proj.Key},
			}
			Expect(
				svc.NewWriter(tx).CreateMany(ctx, &schematics),
			).To(Succeed())

			var retrieved []schematic.Schematic
			Expect(svc.NewRetrieve().Where(schematic.MatchKeys(
				schematics[0].Key,
				schematics[1].Key,
			)).Entries(&retrieved).Exec(ctx, tx)).To(Succeed())
			Expect(retrieved).To(HaveLen(2))
		})
	})
	Describe("SetProject", func() {
		It("Should move the schematics into the project", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			dest := project.Project{Name: "destination"}
			Expect(projectSvc.NewWriter(tx).Create(ctx, &dest)).To(Succeed())
			v := schematic.Schematic{Project: &proj.Key, Name: "moved"}
			Expect(svc.NewWriter(tx).Create(ctx, &v)).To(Succeed())
			Expect(svc.NewWriter(tx).SetProject(ctx, dest.Key, v.Key)).To(Succeed())
			var res schematic.Schematic
			Expect(svc.NewRetrieve().
				Where(schematic.MatchKeys(v.Key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res.Project).To(HaveValue(Equal(dest.Key)))
		})

		It("Should reject a project that does not exist", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			v := schematic.Schematic{Project: &proj.Key, Name: "stuck"}
			Expect(svc.NewWriter(tx).Create(ctx, &v)).To(Succeed())
			Expect(svc.NewWriter(tx).SetProject(ctx, uuid.New(), v.Key)).
				To(MatchError(ContainSubstring("does not exist")))
		})
	})

	Describe("Dispatch", func() {
		It(
			"Should reject an action that leaves the schematic invalid",
			func(ctx SpecContext) {
				s := schematic.Schematic{Project: &proj.Key, Name: "test"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &s)
				})).To(Succeed())
				Expect(
					dispatch(ctx, s.Key, "session-1", []schematic.Action{
						schematic.NewSetAction(schematic.SetPayload{
							Path:  []string{"name"},
							Value: "",
						}),
					}),
				).To(MatchError(ContainSubstring("name: required")))
				var res schematic.Schematic
				Expect(
					svc.NewRetrieve().
						Where(schematic.MatchKeys(s.Key)).
						Entry(&res).
						Exec(ctx, db),
				).To(Succeed())
				Expect(res.Name).To(Equal("test"))
			},
		)
		It("Should reject an action with no payload", func(ctx SpecContext) {
			s := schematic.Schematic{Project: &proj.Key, Name: "test"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &s)
			})).To(Succeed())
			Expect(
				dispatch(ctx, s.Key, "session-1", []schematic.Action{
					{Type: schematic.ActionTypeSet},
				}),
			).To(MatchError(union.ErrMissingPayload))
		})

		It(
			"Should reject a create without changing the schematic it names",
			func(ctx SpecContext) {
				target := schematic.Schematic{Project: &proj.Key, Name: "target"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &target)
				})).To(Succeed())
				other := schematic.Schematic{Project: &proj.Key, Name: "other"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &other)
				})).To(Succeed())
				Expect(dispatch(ctx, target.Key, "d1", []schematic.Action{
					schematic.NewCreateAction(schematic.CreatePayload{
						Schematic: schematic.Schematic{
							Key:  other.Key,
							Name: "overwritten",
						},
					}),
				})).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(
						ContainSubstring("only the Core dispatches a schematic create"),
					),
				))
				var res schematic.Schematic
				Expect(svc.NewRetrieve().
					Where(schematic.MatchKeys(other.Key)).
					Entry(&res).
					Exec(ctx, db)).To(Succeed())
				Expect(res.Name).To(Equal("other"))
			},
		)

		It("Should apply a single SetNodePosition action", func(ctx SpecContext) {
			s := schematic.Schematic{
				Project: &proj.Key,
				Name:    "test",
				Nodes: []schematic.Node{
					{Key: "n1", Position: spatial.XY{X: 0, Y: 0}},
				},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &s)
			})).To(Succeed())
			Expect(
				dispatch(ctx, s.Key, "session-1", []schematic.Action{
					schematic.NewSetNodePositionAction(schematic.SetNodePositionPayload{
						Key:      "n1",
						Position: spatial.XY{X: 100, Y: 200},
					}),
				}),
			).To(Succeed())
			var res schematic.Schematic
			Expect(
				svc.NewRetrieve().
					Where(schematic.MatchKeys(s.Key)).
					Entry(&res).
					Exec(ctx, db),
			).To(Succeed())
			Expect(res.Nodes).To(HaveLen(1))
			Expect(res.Nodes[0].Position).To(Equal(spatial.XY{X: 100, Y: 200}))
		})
		It(
			"Should apply a sequence of mixed actions atomically",
			func(ctx SpecContext) {
				s := schematic.Schematic{Project: &proj.Key, Name: "test"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &s)
				})).To(Succeed())
				Expect(
					dispatch(ctx, s.Key, "session-1", []schematic.Action{
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
						schematic.NewAddEdgeAction(schematic.AddEdgePayload{
							Edge: schematic.Edge{
								Key:    "e1",
								Source: schematic.Handle{Node: "n1"},
								Target: schematic.Handle{Node: "n2"},
							},
						}),
					}),
				).To(Succeed())
				var res schematic.Schematic
				Expect(
					svc.NewRetrieve().
						Where(schematic.MatchKeys(s.Key)).
						Entry(&res).
						Exec(ctx, db),
				).To(Succeed())
				Expect(res.Nodes).To(HaveLen(2))
				Expect(res.Edges).To(HaveLen(1))
			},
		)
		It(
			"Should be a no-op when actions reference non-existent keys",
			func(ctx SpecContext) {
				s := schematic.Schematic{Project: &proj.Key, Name: "test"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &s)
				})).To(Succeed())
				Expect(
					dispatch(ctx, s.Key, "session-1", []schematic.Action{
						schematic.NewRemoveNodeAction(
							schematic.RemoveNodePayload{Key: "ghost"},
						),
						schematic.NewRemoveEdgeAction(
							schematic.RemoveEdgePayload{Key: "ghost-edge"},
						),
					}),
				).To(Succeed())
				var res schematic.Schematic
				Expect(
					svc.NewRetrieve().
						Where(schematic.MatchKeys(s.Key)).
						Entry(&res).
						Exec(ctx, db),
				).To(Succeed())
				Expect(res.Nodes).To(BeEmpty())
				Expect(res.Edges).To(BeEmpty())
			},
		)

		It(
			"Should converge a 30-action drag storm to the final position",
			func(ctx SpecContext) {
				s := schematic.Schematic{
					Project: &proj.Key,
					Name:    "drag-storm",
					Nodes:   []schematic.Node{{Key: "pump"}},
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &s)
				})).To(Succeed())
				actions := make([]schematic.Action, 0, 30)
				for i := range 30 {
					actions = append(
						actions,
						schematic.NewSetNodePositionAction(
							schematic.SetNodePositionPayload{
								Key:      "pump",
								Position: spatial.XY{X: float64(i), Y: float64(i * 2)},
							},
						),
					)
				}
				Expect(
					dispatch(ctx, s.Key, "session-1", actions),
				).To(Succeed())
				var res schematic.Schematic
				Expect(
					svc.NewRetrieve().
						Where(schematic.MatchKeys(s.Key)).
						Entry(&res).
						Exec(ctx, db),
				).To(Succeed())
				Expect(res.Nodes[0].Position).To(Equal(spatial.XY{X: 29, Y: 58}))
			},
		)

		It(
			"Should build a graph atomically from an empty schematic in one Dispatch",
			func(ctx SpecContext) {
				s := schematic.Schematic{Project: &proj.Key, Name: "graph"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &s)
				})).To(Succeed())
				Expect(
					dispatch(ctx, s.Key, "session-1", []schematic.Action{
						schematic.NewSetNodeAction(schematic.SetNodePayload{
							Node: schematic.Node{
								Key:      "pump",
								Position: spatial.XY{X: 0, Y: 0},
							},
						}),
						schematic.NewSetNodeAction(schematic.SetNodePayload{
							Node: schematic.Node{
								Key:      "valve",
								Position: spatial.XY{X: 100, Y: 0},
							},
						}),
						schematic.NewAddEdgeAction(
							schematic.AddEdgePayload{Edge: schematic.Edge{
								Key: "e1",
								Source: schematic.Handle{
									Node:  "pump",
									Param: "out",
								},
								Target: schematic.Handle{
									Node:  "valve",
									Param: "in",
								},
							}},
						),
						schematic.NewSetConfigAction(schematic.SetConfigPayload{
							Key:    "pump",
							Config: tankCfg("Main Pump", ""),
						}),
					}),
				).To(Succeed())
				var res schematic.Schematic
				Expect(
					svc.NewRetrieve().
						Where(schematic.MatchKeys(s.Key)).
						Entry(&res).
						Exec(ctx, db),
				).To(Succeed())
				Expect(res.Nodes).To(HaveLen(2))
				Expect(res.Edges).To(HaveLen(1))
				Expect(res.Configs["pump"]).To(Equal(tankCfg("Main Pump", "")))
			},
		)

		It(
			"Should notify subscribers with the dispatched ScopedAction on success",
			func(ctx SpecContext) {
				s := schematic.Schematic{Project: &proj.Key, Name: "observed"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &s)
				})).To(Succeed())
				rec := &Recorder[schematic.Key, schematic.Action]{}
				disconnect := svc.OnAction(rec.Record)
				DeferCleanup(disconnect)
				actions := []schematic.Action{
					schematic.NewSetNodeAction(schematic.SetNodePayload{
						Node: schematic.Node{Key: "n1"},
					}),
					schematic.NewSetNodePositionAction(schematic.SetNodePositionPayload{
						Key:      "n1",
						Position: spatial.XY{X: 5, Y: 5},
					}),
				}
				Expect(
					dispatch(ctx, s.Key, "client-xyz", actions),
				).To(Succeed())
				seen := rec.Snapshot()
				Expect(seen).To(HaveLen(1))
				Expect(seen[0].Key).To(Equal(s.Key))
				Expect(seen[0].DispatchKey).To(Equal("client-xyz"))
				Expect(seen[0].Seq).To(BeNumerically(">", uint64(0)))
				Expect(seen[0].Actions).To(HaveLen(2))
				Expect(seen[0].Actions[0].Type).To(Equal(schematic.ActionTypeSetNode))
				Expect(
					seen[0].Actions[1].Type,
				).To(Equal(schematic.ActionTypeSetNodePosition))
			},
		)

		It(
			"Should stamp strictly increasing Seq values onto successive Dispatch broadcasts",
			func(ctx SpecContext) {
				s := schematic.Schematic{Project: &proj.Key, Name: "seq-test"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &s)
				})).To(Succeed())
				rec := &Recorder[schematic.Key, schematic.Action]{}
				DeferCleanup(svc.OnAction(rec.Record))
				action := []schematic.Action{
					schematic.NewSetNodeAction(schematic.SetNodePayload{
						Node: schematic.Node{Key: "n1"},
					}),
				}
				for range 3 {
					Expect(
						dispatch(ctx, s.Key, "client-xyz", action),
					).To(Succeed())
				}
				seen := rec.Snapshot()
				Expect(seen).To(HaveLen(3))
				Expect(seen[1].Seq).To(BeNumerically(">", seen[0].Seq))
				Expect(seen[2].Seq).To(BeNumerically(">", seen[1].Seq))
			},
		)

		It(
			"Should fail with query.ErrNotFound and not notify subscribers when the target schematic does not exist",
			func(ctx SpecContext) {
				rec := &Recorder[schematic.Key, schematic.Action]{}
				DeferCleanup(svc.OnAction(rec.Record))
				Expect(
					dispatch(ctx, uuid.New(), "client-xyz", []schematic.Action{
						schematic.NewRemoveNodeAction(
							schematic.RemoveNodePayload{Key: "n1"},
						),
					}),
				).To(MatchError(query.ErrNotFound))
				Expect(rec.Snapshot()).To(BeEmpty())
			},
		)
	})

	Describe("Copy", func() {
		It(
			"Should copy a Schematic with a new name under the same project",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				s := schematic.Schematic{Project: &proj.Key, Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
				var cpy schematic.Schematic
				Expect(
					svc.NewWriter(tx).Copy(ctx, s.Key, "test2", &cpy),
				).To(Succeed())
				Expect(cpy.Key).ToNot(Equal(s.Key))
				Expect(cpy.Name).To(Equal("test2"))
				Expect(cpy.Project).To(HaveValue(Equal(proj.Key)))
			},
		)
	})
})
