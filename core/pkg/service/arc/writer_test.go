// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/spatial"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create an Arc with generated key", func(ctx SpecContext) {
			a := arc.Arc{Name: "test-arc"}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			Expect(a.Key).ToNot(Equal(uuid.Nil))
		})

		It("Should create an Arc with explicit key", func(ctx SpecContext) {
			key := uuid.New()
			a := arc.Arc{Key: key, Name: "test-arc-with-key"}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			Expect(a.Key).To(Equal(key))
		})

		It("Should handle multiple arc creations", func(ctx SpecContext) {
			a1 := arc.Arc{Name: "arc-1"}
			a2 := arc.Arc{Name: "arc-2"}
			Expect(svc.NewWriter(tx).Create(ctx, &a1)).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, &a2)).To(Succeed())
			Expect(a1.Key).ToNot(Equal(uuid.Nil))
			Expect(a2.Key).ToNot(Equal(uuid.Nil))
			Expect(a1.Key).ToNot(Equal(a2.Key))
		})
	})

	Describe("CreateMany", func() {
		It("Should create multiple Arcs", func(ctx SpecContext) {
			arcs := []arc.Arc{
				{Name: "arc-many-1"},
				{Name: "arc-many-2"},
			}
			Expect(svc.NewWriter(tx).CreateMany(ctx, &arcs)).To(Succeed())

			var retrieved []arc.Arc
			Expect(svc.NewRetrieve().Where(arc.MatchKeys(
				arcs[0].Key,
				arcs[1].Key,
			)).Entries(&retrieved).Exec(ctx, tx)).To(Succeed())
			Expect(retrieved).To(HaveLen(2))
		})
	})

	Describe("Update", func() {
		It("Should update an existing Arc", func(ctx SpecContext) {
			key := uuid.New()
			a := arc.Arc{
				Key:   key,
				Name:  "existing-arc",
				Graph: graph.Graph{},
				Text:  text.Text{},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			a.Name = "updated-arc"
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			var retrieved arc.Arc
			Expect(svc.NewRetrieve().Where(arc.MatchKeys(key)).Entry(&retrieved).Exec(ctx, tx)).To(Succeed())
			Expect(retrieved.Name).To(Equal("updated-arc"))
		})
	})

	Describe("Delete", func() {
		It("Should delete an Arc", func(ctx SpecContext) {
			a := arc.Arc{
				Name:  "arc-to-delete",
				Graph: graph.Graph{},
				Text:  text.Text{},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, a.Key)).To(Succeed())
			Expect(svc.NewRetrieve().Where(arc.MatchKeys(a.Key)).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
		})

		It("Should delete multiple Arcs", func(ctx SpecContext) {
			a1 := arc.Arc{Name: "arc-to-delete-1"}
			a2 := arc.Arc{Name: "arc-to-delete-2"}
			w := svc.NewWriter(tx)
			Expect(w.Create(ctx, &a1)).To(Succeed())
			Expect(w.Create(ctx, &a2)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, a1.Key, a2.Key)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(arc.MatchKeys(a1.Key, a2.Key)).
				Exec(ctx, tx),
			).To(MatchError(query.ErrNotFound))
		})

		It("Should handle delete of non-existent arc gracefully", func(ctx SpecContext) {
			nonExistentKey := uuid.New()
			Expect(svc.NewWriter(tx).Delete(ctx, nonExistentKey)).To(Succeed())
		})

		It("Should delete child tasks when deleting an arc", func(ctx SpecContext) {
			a := arc.Arc{Name: "arc-with-task"}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())

			t := &task.Task{
				Key:  task.NewKey(testRack.Key, 0),
				Name: "child-task",
				Type: "arc",
			}
			Expect(taskSvc.NewWriter(tx).Create(ctx, t)).To(Succeed())

			Expect(otg.NewWriter(tx).DefineRelationship(
				ctx,
				arc.OntologyID(a.Key),
				ontology.RelationshipTypeParentOf,
				task.OntologyID(t.Key),
			)).To(Succeed())

			Expect(svc.NewWriter(tx).Delete(ctx, a.Key)).To(Succeed())

			Expect(svc.NewRetrieve().Where(arc.MatchKeys(a.Key)).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
			Expect(taskSvc.NewRetrieve().Where(task.MatchKeys(t.Key)).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
		})

		It("Should handle arc deletion when arc has no child tasks", func(ctx SpecContext) {
			a := arc.Arc{Name: "arc-without-tasks"}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, a.Key)).To(Succeed())
			Expect(svc.NewRetrieve().Where(arc.MatchKeys(a.Key)).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
		})

		It("Should delete multiple child tasks when deleting an arc", func(ctx SpecContext) {
			a := arc.Arc{Name: "arc-with-multiple-tasks"}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())

			t1 := &task.Task{
				Key:  task.NewKey(testRack.Key, 0),
				Name: "child-task-1",
				Type: "arc",
			}
			t2 := &task.Task{
				Key:  task.NewKey(testRack.Key, 0),
				Name: "child-task-2",
				Type: "arc",
			}
			Expect(taskSvc.NewWriter(tx).Create(ctx, t1)).To(Succeed())
			Expect(taskSvc.NewWriter(tx).Create(ctx, t2)).To(Succeed())

			otgWriter := otg.NewWriter(tx)
			Expect(otgWriter.DefineRelationship(
				ctx,
				arc.OntologyID(a.Key),
				ontology.RelationshipTypeParentOf,
				task.OntologyID(t1.Key),
			)).To(Succeed())
			Expect(otgWriter.DefineRelationship(
				ctx,
				arc.OntologyID(a.Key),
				ontology.RelationshipTypeParentOf,
				task.OntologyID(t2.Key),
			)).To(Succeed())

			Expect(svc.NewWriter(tx).Delete(ctx, a.Key)).To(Succeed())

			Expect(svc.NewRetrieve().Where(arc.MatchKeys(a.Key)).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
			Expect(taskSvc.NewRetrieve().Where(task.MatchKeys(t1.Key)).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
			Expect(taskSvc.NewRetrieve().Where(task.MatchKeys(t2.Key)).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Dispatch", func() {
		It("Should apply a single SetNodePosition action to the target Arc", func(ctx SpecContext) {
			a := arc.Arc{Name: "dispatch-pos", Graph: graph.Graph{Nodes: graph.Nodes{
				{Key: "n1", Position: spatial.XY{X: 0, Y: 0}},
			}}}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			Expect(svc.NewWriter(tx).Dispatch(ctx, a.Key, "session-1", []arc.Action{
				arc.NewSetNodePositionAction(arc.SetNodePositionPayload{
					Key:      "n1",
					Position: spatial.XY{X: 100, Y: 200},
				}),
			})).To(Succeed())
			var res arc.Arc
			Expect(svc.NewRetrieve().Where(arc.MatchKeys(a.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Graph.Nodes[0].Position).To(Equal(spatial.XY{X: 100, Y: 200}))
		})

		It("Should apply a sequence of mixed actions atomically", func(ctx SpecContext) {
			a := arc.Arc{Name: "dispatch-seq"}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			Expect(svc.NewWriter(tx).Dispatch(ctx, a.Key, "session-1", []arc.Action{
				arc.NewSetNodeAction(arc.SetNodePayload{Node: graph.Node{Key: "n1"}}),
				arc.NewSetNodeAction(arc.SetNodePayload{Node: graph.Node{Key: "n2"}}),
				arc.NewAddEdgeAction(arc.AddEdgePayload{Edge: ir.Edge{
					Source: ir.Handle{Node: "n1", Param: "out"},
					Target: ir.Handle{Node: "n2", Param: "in"},
					Kind:   ir.EdgeKindContinuous,
				}}),
			})).To(Succeed())
			var res arc.Arc
			Expect(svc.NewRetrieve().Where(arc.MatchKeys(a.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Graph.Nodes).To(HaveLen(2))
			Expect(res.Graph.Edges).To(HaveLen(1))
		})

		It("Should notify subscribers with the dispatched scoped action on success", func(ctx SpecContext) {
			a := arc.Arc{Name: "observed"}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			rec := &Recorder[arc.Key, arc.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			actions := []arc.Action{
				arc.NewSetNodeAction(arc.SetNodePayload{Node: graph.Node{Key: "n1"}}),
				arc.NewRenameAction(arc.RenamePayload{Name: "observed-renamed"}),
			}
			Expect(svc.NewWriter(tx).Dispatch(ctx, a.Key, "client-xyz", actions)).To(Succeed())
			seen := rec.Snapshot()
			Expect(seen).To(HaveLen(1))
			Expect(seen[0].Key).To(Equal(a.Key))
			Expect(seen[0].DispatchKey).To(Equal("client-xyz"))
			Expect(seen[0].Seq).To(BeNumerically(">", uint64(0)))
			Expect(seen[0].Actions).To(HaveLen(2))
			Expect(seen[0].Actions[0].Type).To(Equal(arc.ActionTypeSetNode))
			Expect(seen[0].Actions[1].Type).To(Equal(arc.ActionTypeRename))
		})

		It("Should stamp strictly increasing Seq values onto successive broadcasts", func(ctx SpecContext) {
			a := arc.Arc{Name: "seq-test"}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			rec := &Recorder[arc.Key, arc.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			action := []arc.Action{arc.NewSetNodeAction(arc.SetNodePayload{Node: graph.Node{Key: "n1"}})}
			for range 3 {
				Expect(svc.NewWriter(tx).Dispatch(ctx, a.Key, "client-xyz", action)).To(Succeed())
			}
			seen := rec.Snapshot()
			Expect(seen).To(HaveLen(3))
			Expect(seen[1].Seq).To(BeNumerically(">", seen[0].Seq))
			Expect(seen[2].Seq).To(BeNumerically(">", seen[1].Seq))
		})

		It("Should fail with query.ErrNotFound and not notify when the target Arc does not exist", func(ctx SpecContext) {
			rec := &Recorder[arc.Key, arc.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			Expect(svc.NewWriter(tx).Dispatch(ctx, uuid.New(), "client-xyz", []arc.Action{
				arc.NewRenameAction(arc.RenamePayload{Name: "ghost"}),
			})).To(MatchError(query.ErrNotFound))
			Expect(rec.Snapshot()).To(BeEmpty())
		})
	})
})
