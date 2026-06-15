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
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create an Arc with generated key", func(ctx SpecContext) {
			a := MustSucceed(svc.NewWriter(tx).Create(ctx, arc.New{Name: "test-arc"}))
			Expect(a.Key).ToNot(Equal(uuid.Nil))
		})

		It("Should create an Arc with explicit key", func(ctx SpecContext) {
			key := uuid.New()
			a := MustSucceed(svc.NewWriter(tx).Create(ctx, arc.New{Key: key, Name: "test-arc-with-key"}))
			Expect(a.Key).To(Equal(key))
		})

		It("Should handle multiple arc creations", func(ctx SpecContext) {
			a1 := MustSucceed(svc.NewWriter(tx).Create(ctx, arc.New{Name: "arc-1"}))
			a2 := MustSucceed(svc.NewWriter(tx).Create(ctx, arc.New{Name: "arc-2"}))
			Expect(a1.Key).ToNot(Equal(uuid.Nil))
			Expect(a2.Key).ToNot(Equal(uuid.Nil))
			Expect(a1.Key).ToNot(Equal(a2.Key))
		})
	})

	Describe("CreateMany", func() {
		It("Should create multiple Arcs", func(ctx SpecContext) {
			arcs := MustSucceed(svc.NewWriter(tx).CreateMany(ctx, []arc.New{
				{Name: "arc-many-1"},
				{Name: "arc-many-2"},
			}))
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
			MustSucceed(svc.NewWriter(tx).Create(ctx, arc.New{
				Key:   key,
				Name:  "existing-arc",
				Graph: graph.Graph{},
				Text:  text.Text{},
			}))
			MustSucceed(svc.NewWriter(tx).Create(ctx, arc.New{
				Key:   key,
				Name:  "updated-arc",
				Graph: graph.Graph{},
				Text:  text.Text{},
			}))
			var retrieved arc.Arc
			Expect(svc.NewRetrieve().Where(arc.MatchKeys(key)).Entry(&retrieved).Exec(ctx, tx)).To(Succeed())
			Expect(retrieved.Name).To(Equal("updated-arc"))
		})
	})

	Describe("Delete", func() {
		It("Should delete an Arc", func(ctx SpecContext) {
			a := MustSucceed(svc.NewWriter(tx).Create(ctx, arc.New{
				Name:  "arc-to-delete",
				Graph: graph.Graph{},
				Text:  text.Text{},
			}))
			Expect(svc.NewWriter(tx).Delete(ctx, a.Key)).To(Succeed())
			Expect(svc.NewRetrieve().Where(arc.MatchKeys(a.Key)).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
		})

		It("Should delete multiple Arcs", func(ctx SpecContext) {
			w := svc.NewWriter(tx)
			a1 := MustSucceed(w.Create(ctx, arc.New{Name: "arc-to-delete-1"}))
			a2 := MustSucceed(w.Create(ctx, arc.New{Name: "arc-to-delete-2"}))
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
			a := MustSucceed(svc.NewWriter(tx).Create(ctx, arc.New{Name: "arc-with-task"}))

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
			a := MustSucceed(svc.NewWriter(tx).Create(ctx, arc.New{Name: "arc-without-tasks"}))
			Expect(svc.NewWriter(tx).Delete(ctx, a.Key)).To(Succeed())
			Expect(svc.NewRetrieve().Where(arc.MatchKeys(a.Key)).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
		})

		It("Should delete multiple child tasks when deleting an arc", func(ctx SpecContext) {
			a := MustSucceed(svc.NewWriter(tx).Create(ctx, arc.New{Name: "arc-with-multiple-tasks"}))

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

	Describe("Task Association", func() {
		It("Should create and associate a task on the provided rack", func(ctx SpecContext) {
			a := MustSucceed(svc.NewWriter(tx).Create(ctx, arc.New{
				Name: "deployed-arc",
				Rack: &testRack.Key,
			}))
			Expect(a.Task).ToNot(BeNil())
			Expect(a.Task.Rack()).To(Equal(testRack.Key))

			var t task.Task
			Expect(taskSvc.NewRetrieve().
				Where(task.MatchKeys(*a.Task)).
				Entry(&t).
				Exec(ctx, tx)).To(Succeed())
			Expect(t.Type).To(Equal(arc.TaskType))
			Expect(t.Config["arc_key"]).To(Equal(a.Key.String()))
		})

		It("Should not create a task when no rack is provided", func(ctx SpecContext) {
			a := MustSucceed(svc.NewWriter(tx).Create(ctx, arc.New{Name: "undeployed-arc"}))
			Expect(a.Task).To(BeNil())
		})

		It("Should reuse the task when redeploying to the same rack", func(ctx SpecContext) {
			w := svc.NewWriter(tx)
			a := MustSucceed(w.Create(ctx, arc.New{Name: "redeploy-same", Rack: &testRack.Key}))
			first := *a.Task

			redeployed := MustSucceed(w.Create(ctx, arc.New{
				Key:  a.Key,
				Name: a.Name,
				Rack: &testRack.Key,
			}))
			Expect(*redeployed.Task).To(Equal(first))
		})

		It("Should migrate the task when redeploying to a different rack", func(ctx SpecContext) {
			otherRack := &rack.Rack{Name: "Other Rack"}
			Expect(rackSvc.NewWriter(db).Create(ctx, otherRack)).To(Succeed())

			w := svc.NewWriter(tx)
			a := MustSucceed(w.Create(ctx, arc.New{Name: "migrate", Rack: &testRack.Key}))
			original := *a.Task

			migrated := MustSucceed(w.Create(ctx, arc.New{
				Key:  a.Key,
				Name: a.Name,
				Rack: &otherRack.Key,
			}))
			Expect(*migrated.Task).ToNot(Equal(original))
			Expect(migrated.Task.Rack()).To(Equal(otherRack.Key))
			Expect(taskSvc.NewRetrieve().Where(task.MatchKeys(original)).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Retrieve Resolution", func() {
		It("Should resolve the task key with IncludeTask", func(ctx SpecContext) {
			a := MustSucceed(svc.NewWriter(tx).Create(ctx, arc.New{
				Name: "resolve-task",
				Rack: &testRack.Key,
			}))

			var got arc.Arc
			Expect(svc.NewRetrieve().
				Where(arc.MatchKeys(a.Key)).
				IncludeTask(true).
				Entry(&got).
				Exec(ctx, tx)).To(Succeed())
			Expect(got.Task).ToNot(BeNil())
			Expect(*got.Task).To(Equal(*a.Task))
		})

		It("Should leave the task nil for an undeployed arc", func(ctx SpecContext) {
			a := MustSucceed(svc.NewWriter(tx).Create(ctx, arc.New{Name: "resolve-none"}))

			var got arc.Arc
			Expect(svc.NewRetrieve().
				Where(arc.MatchKeys(a.Key)).
				IncludeTask(true).
				Entry(&got).
				Exec(ctx, tx)).To(Succeed())
			Expect(got.Task).To(BeNil())
		})

		It("Should resolve the task status with IncludeStatus", func(ctx SpecContext) {
			a := MustSucceed(svc.NewWriter(tx).Create(ctx, arc.New{
				Name: "resolve-status",
				Rack: &testRack.Key,
			}))

			var got arc.Arc
			Expect(svc.NewRetrieve().
				Where(arc.MatchKeys(a.Key)).
				IncludeStatus(true).
				Entry(&got).
				Exec(ctx, tx)).To(Succeed())
			Expect(got.Task).ToNot(BeNil())
			Expect(got.Status).ToNot(BeNil())
		})
	})
})
