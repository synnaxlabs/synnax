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
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/query"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create an Arc with generated key", func() {
			a := arc.Arc{Name: "test-arc"}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			Expect(a.Key).ToNot(Equal(uuid.Nil))
		})

		It("Should create an Arc with explicit key", func() {
			key := uuid.New()
			a := arc.Arc{Key: key, Name: "test-arc-with-key"}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			Expect(a.Key).To(Equal(key))
		})

		It("Should handle multiple arc creations", func() {
			a1 := arc.Arc{Name: "arc-1"}
			a2 := arc.Arc{Name: "arc-2"}
			Expect(svc.NewWriter(tx).Create(ctx, &a1)).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, &a2)).To(Succeed())
			Expect(a1.Key).ToNot(Equal(uuid.Nil))
			Expect(a2.Key).ToNot(Equal(uuid.Nil))
			Expect(a1.Key).ToNot(Equal(a2.Key))
		})
	})

	Describe("Update", func() {
		It("Should update an existing Arc", func() {
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
			Expect(svc.NewRetrieve().WhereKeys(key).Entry(&retrieved).Exec(ctx, tx)).To(Succeed())
			Expect(retrieved.Name).To(Equal("updated-arc"))
		})
	})

	Describe("Delete", func() {
		It("Should delete an Arc", func() {
			a := arc.Arc{
				Name:  "arc-to-delete",
				Graph: graph.Graph{},
				Text:  text.Text{},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, a.Key)).To(Succeed())
			Expect(svc.NewRetrieve().WhereKeys(a.Key).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
		})

		It("Should delete multiple Arcs", func() {
			a1 := arc.Arc{Name: "arc-to-delete-1"}
			a2 := arc.Arc{Name: "arc-to-delete-2"}
			w := svc.NewWriter(tx)
			Expect(w.Create(ctx, &a1)).To(Succeed())
			Expect(w.Create(ctx, &a2)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, a1.Key, a2.Key)).To(Succeed())
			Expect(svc.NewRetrieve().
				WhereKeys(a1.Key, a2.Key).
				Exec(ctx, tx),
			).To(MatchError(query.ErrNotFound))
		})

		It("Should handle delete of non-existent arc gracefully", func() {
			nonExistentKey := uuid.New()
			Expect(svc.NewWriter(tx).Delete(ctx, nonExistentKey)).To(Succeed())
		})

		It("Should delete child tasks when deleting an arc", func() {
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

			Expect(svc.NewRetrieve().WhereKeys(a.Key).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
			Expect(taskSvc.NewRetrieve().WhereKeys(t.Key).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
		})

		It("Should handle arc deletion when arc has no child tasks", func() {
			a := arc.Arc{Name: "arc-without-tasks"}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, a.Key)).To(Succeed())
			Expect(svc.NewRetrieve().WhereKeys(a.Key).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
		})

		It("Should delete multiple child tasks when deleting an arc", func() {
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

			Expect(svc.NewRetrieve().WhereKeys(a.Key).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
			Expect(taskSvc.NewRetrieve().WhereKeys(t1.Key).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
			Expect(taskSvc.NewRetrieve().WhereKeys(t2.Key).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
		})
	})
})
