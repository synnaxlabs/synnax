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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create a Schematic", func(ctx SpecContext) {
			s := schematic.Schematic{Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			Expect(s.Key).ToNot(Equal(uuid.Nil))
		})
	})
	Describe("Dispatch", func() {
		It("Should apply a single SetNodePosition action", func(ctx SpecContext) {
			s := schematic.Schematic{
				Name: "test",
				Nodes: []schematic.Node{
					{Key: "n1", Position: spatial.XY{X: 0, Y: 0}},
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "session-1", []schematic.Action{
				schematic.NewSetNodePositionAction(schematic.SetNodePositionPayload{
					Key:      "n1",
					Position: spatial.XY{X: 100, Y: 200},
				}),
			})).To(Succeed())
			var res schematic.Schematic
			Expect(svc.NewRetrieve().Where(schematic.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Nodes).To(HaveLen(1))
			Expect(res.Nodes[0].Position).To(Equal(spatial.XY{X: 100, Y: 200}))
		})
		It("Should apply a sequence of mixed actions atomically", func(ctx SpecContext) {
			s := schematic.Schematic{Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "session-1", []schematic.Action{
				schematic.NewSetNodeAction(schematic.SetNodePayload{
					Node: schematic.Node{Key: "n1", Position: spatial.XY{X: 1, Y: 2}},
				}),
				schematic.NewSetNodeAction(schematic.SetNodePayload{
					Node: schematic.Node{Key: "n2", Position: spatial.XY{X: 3, Y: 4}},
				}),
				schematic.NewAddEdgeAction(schematic.AddEdgePayload{
					Edge: schematic.Edge{
						Key:    "e1",
						Source: schematic.Handle{Node: "n1"},
						Target: schematic.Handle{Node: "n2"},
					},
				}),
			})).To(Succeed())
			var res schematic.Schematic
			Expect(svc.NewRetrieve().Where(schematic.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Nodes).To(HaveLen(2))
			Expect(res.Edges).To(HaveLen(1))
		})
		It("Should reject non-Rename Dispatch on a snapshot schematic", func(ctx SpecContext) {
			s := schematic.Schematic{Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			var snap schematic.Schematic
			Expect(svc.NewWriter(tx).Copy(ctx, s.Key, "snap", true, &snap)).To(Succeed())
			Expect(svc.NewWriter(tx).Dispatch(ctx, snap.Key, "session-1", []schematic.Action{
				schematic.NewRemoveNodeAction(schematic.RemoveNodePayload{Key: "n1"}),
			})).To(MatchError(validate.ErrValidation))
		})
		It("Should allow Rename Dispatch on a snapshot schematic", func(ctx SpecContext) {
			s := schematic.Schematic{Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			var snap schematic.Schematic
			Expect(svc.NewWriter(tx).Copy(ctx, s.Key, "snap", true, &snap)).To(Succeed())
			Expect(svc.NewWriter(tx).Dispatch(ctx, snap.Key, "session-1", []schematic.Action{
				schematic.NewRenameAction(schematic.RenamePayload{Name: "renamed-snap"}),
			})).To(Succeed())
			var res schematic.Schematic
			Expect(svc.NewRetrieve().Where(schematic.MatchKeys(snap.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("renamed-snap"))
		})
		It("Should reject Dispatch on a snapshot when any action is not a Rename", func(ctx SpecContext) {
			s := schematic.Schematic{Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			var snap schematic.Schematic
			Expect(svc.NewWriter(tx).Copy(ctx, s.Key, "snap", true, &snap)).To(Succeed())
			Expect(svc.NewWriter(tx).Dispatch(ctx, snap.Key, "session-1", []schematic.Action{
				schematic.NewRenameAction(schematic.RenamePayload{Name: "renamed-snap"}),
				schematic.NewRemoveNodeAction(schematic.RemoveNodePayload{Key: "n1"}),
			})).To(MatchError(validate.ErrValidation))
			var res schematic.Schematic
			Expect(svc.NewRetrieve().Where(schematic.MatchKeys(snap.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("snap"))
		})
		It("Should be a no-op when actions reference non-existent keys", func(ctx SpecContext) {
			s := schematic.Schematic{Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "session-1", []schematic.Action{
				schematic.NewRemoveNodeAction(schematic.RemoveNodePayload{Key: "ghost"}),
				schematic.NewRemoveEdgeAction(schematic.RemoveEdgePayload{Key: "ghost-edge"}),
			})).To(Succeed())
			var res schematic.Schematic
			Expect(svc.NewRetrieve().Where(schematic.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Nodes).To(BeEmpty())
			Expect(res.Edges).To(BeEmpty())
		})

		It("Should converge a 30-action drag storm to the final position", func(ctx SpecContext) {
			s := schematic.Schematic{
				Name:  "drag-storm",
				Nodes: []schematic.Node{{Key: "pump"}},
			}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			actions := make([]schematic.Action, 0, 30)
			for i := range 30 {
				actions = append(actions, schematic.NewSetNodePositionAction(schematic.SetNodePositionPayload{
					Key:      "pump",
					Position: spatial.XY{X: float64(i), Y: float64(i * 2)},
				}))
			}
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "session-1", actions)).To(Succeed())
			var res schematic.Schematic
			Expect(svc.NewRetrieve().Where(schematic.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Nodes[0].Position).To(Equal(spatial.XY{X: 29, Y: 58}))
		})

		It("Should build a graph atomically from an empty schematic in one Dispatch", func(ctx SpecContext) {
			s := schematic.Schematic{Name: "graph"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "session-1", []schematic.Action{
				schematic.NewSetNodeAction(schematic.SetNodePayload{
					Node: schematic.Node{Key: "pump", Position: spatial.XY{X: 0, Y: 0}},
				}),
				schematic.NewSetNodeAction(schematic.SetNodePayload{
					Node: schematic.Node{Key: "valve", Position: spatial.XY{X: 100, Y: 0}},
				}),
				schematic.NewAddEdgeAction(schematic.AddEdgePayload{Edge: schematic.Edge{
					Key:    "e1",
					Source: schematic.Handle{Node: "pump", Param: "out"},
					Target: schematic.Handle{Node: "valve", Param: "in"},
				}}),
				schematic.NewSetConfigAction(schematic.SetConfigPayload{
					Key:    "pump",
					Config: msgpack.EncodedJSON{"label": "Main Pump"},
				}),
			})).To(Succeed())
			var res schematic.Schematic
			Expect(svc.NewRetrieve().Where(schematic.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Nodes).To(HaveLen(2))
			Expect(res.Edges).To(HaveLen(1))
			Expect(res.Configs["pump"]).To(HaveKeyWithValue("label", "Main Pump"))
		})

		It("Should notify subscribers with the dispatched ScopedAction on success", func(ctx SpecContext) {
			s := schematic.Schematic{Name: "observed"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
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
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "client-xyz", actions)).To(Succeed())
			seen := rec.Snapshot()
			Expect(seen).To(HaveLen(1))
			Expect(seen[0].Key).To(Equal(s.Key))
			Expect(seen[0].DispatchKey).To(Equal("client-xyz"))
			Expect(seen[0].Seq).To(BeNumerically(">", uint64(0)))
			Expect(seen[0].Actions).To(HaveLen(2))
			Expect(seen[0].Actions[0].Type).To(Equal(schematic.ActionTypeSetNode))
			Expect(seen[0].Actions[1].Type).To(Equal(schematic.ActionTypeSetNodePosition))
		})

		It("Should stamp strictly increasing Seq values onto successive Dispatch broadcasts", func(ctx SpecContext) {
			s := schematic.Schematic{Name: "seq-test"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			rec := &Recorder[schematic.Key, schematic.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			action := []schematic.Action{schematic.NewSetNodeAction(schematic.SetNodePayload{
				Node: schematic.Node{Key: "n1"},
			})}
			for range 3 {
				Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "client-xyz", action)).To(Succeed())
			}
			seen := rec.Snapshot()
			Expect(seen).To(HaveLen(3))
			Expect(seen[1].Seq).To(BeNumerically(">", seen[0].Seq))
			Expect(seen[2].Seq).To(BeNumerically(">", seen[1].Seq))
		})

		It("Should not notify subscribers when Dispatch is rejected on a snapshot", func(ctx SpecContext) {
			s := schematic.Schematic{Name: "snap-test"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			var snap schematic.Schematic
			Expect(svc.NewWriter(tx).Copy(ctx, s.Key, "snap", true, &snap)).To(Succeed())
			rec := &Recorder[schematic.Key, schematic.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			Expect(svc.NewWriter(tx).Dispatch(ctx, snap.Key, "client-xyz", []schematic.Action{
				schematic.NewRemoveNodeAction(schematic.RemoveNodePayload{Key: "n1"}),
			})).To(MatchError(validate.ErrValidation))
			Expect(rec.Snapshot()).To(BeEmpty())
		})

		It("Should fail with query.ErrNotFound and not notify subscribers when the target schematic does not exist", func(ctx SpecContext) {
			rec := &Recorder[schematic.Key, schematic.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			Expect(svc.NewWriter(tx).Dispatch(ctx, uuid.New(), "client-xyz", []schematic.Action{
				schematic.NewRemoveNodeAction(schematic.RemoveNodePayload{Key: "n1"}),
			})).To(MatchError(query.ErrNotFound))
			Expect(rec.Snapshot()).To(BeEmpty())
		})
	})

	Describe("Copy", func() {
		It("Should copy a Schematic with a new name under the same workspace", func(ctx SpecContext) {
			s := schematic.Schematic{Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			var cpy schematic.Schematic
			Expect(svc.NewWriter(tx).Copy(ctx, s.Key, "test2", false, &cpy)).To(Succeed())
			Expect(cpy.Key).ToNot(Equal(s.Key))
			Expect(cpy.Name).To(Equal("test2"))
			var res []ontology.Resource
			Expect(otg.NewRetrieve().WhereIDs(ws.OntologyID()).TraverseTo(ontology.ChildrenTraverser).Entries(&res).Exec(ctx, tx)).To(Succeed())
			keys := lo.Map(res, func(r ontology.Resource, _ int) string { return r.ID.Key })
			Expect(keys).To(ContainElement(cpy.Key.String()))
		})
		It("Should copy a Schematic into a snapshot that cannot be modified", func(ctx SpecContext) {
			s := schematic.Schematic{Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			var cpy schematic.Schematic
			Expect(svc.NewWriter(tx).Copy(ctx, s.Key, "test2", true, &cpy)).To(Succeed())
			Expect(svc.NewWriter(tx).Dispatch(ctx, cpy.Key, "session-1", []schematic.Action{
				schematic.NewSetNodeAction(schematic.SetNodePayload{
					Node: schematic.Node{Key: "n1"},
				}),
			})).To(MatchError(validate.ErrValidation))
		})
	})
})
