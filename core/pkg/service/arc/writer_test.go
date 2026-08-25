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
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/query"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create an Arc with generated key", func(ctx SpecContext) {
			a := arc.Arc{Name: "test-arc", Mode: arc.ModeText}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			Expect(a.Key).ToNot(Equal(uuid.Nil))
		})

		It(
			"Should return a validation error when the name is empty",
			func(ctx SpecContext) {
				a := arc.Arc{Mode: arc.ModeText}
				Expect(svc.NewWriter(tx).Create(ctx, &a)).
					To(MatchError(ContainSubstring("name: required")))
			},
		)

		It("Should create an Arc with explicit key", func(ctx SpecContext) {
			key := uuid.New()
			a := arc.Arc{Key: key, Name: "test-arc-with-key", Mode: arc.ModeText}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			Expect(a.Key).To(Equal(key))
		})

		It("Should handle multiple Arc creations", func(ctx SpecContext) {
			a1 := arc.Arc{Name: "arc-1", Mode: arc.ModeText}
			a2 := arc.Arc{Name: "arc-2", Mode: arc.ModeText}
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
				{Name: "arc-many-1", Mode: arc.ModeText},
				{Name: "arc-many-2", Mode: arc.ModeText},
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
				Mode:  arc.ModeText,
				Graph: graph.Graph{},
				Text:  text.Text{},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			a.Name = "updated-arc"
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			var retrieved arc.Arc
			Expect(
				svc.NewRetrieve().
					Where(arc.MatchKeys(key)).
					Entry(&retrieved).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(retrieved.Name).To(Equal("updated-arc"))
		})
	})

	Describe("Delete", func() {
		It("Should delete an Arc", func(ctx SpecContext) {
			a := arc.Arc{
				Name:  "arc-to-delete",
				Mode:  arc.ModeText,
				Graph: graph.Graph{},
				Text:  text.Text{},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, a.Key)).To(Succeed())
			Expect(svc.NewRetrieve().Where(arc.MatchKeys(a.Key)).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
		})

		It("Should delete multiple Arcs", func(ctx SpecContext) {
			a1 := arc.Arc{Name: "arc-to-delete-1", Mode: arc.ModeText}
			a2 := arc.Arc{Name: "arc-to-delete-2", Mode: arc.ModeText}
			w := svc.NewWriter(tx)
			Expect(w.Create(ctx, &a1)).To(Succeed())
			Expect(w.Create(ctx, &a2)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, a1.Key, a2.Key)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(arc.MatchKeys(a1.Key, a2.Key)).
				Exec(ctx, tx),
			).To(MatchError(query.ErrNotFound))
		})

		It(
			"Should handle delete of non-existent Arc gracefully",
			func(ctx SpecContext) {
				nonExistentKey := uuid.New()
				Expect(svc.NewWriter(tx).Delete(ctx, nonExistentKey)).To(Succeed())
			},
		)

		It("Should delete child tasks when deleting an Arc", func(ctx SpecContext) {
			a := arc.Arc{Name: "arc-with-task", Mode: arc.ModeText}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())

			t := &task.Task{
				Rack: testRack.Key,
				Name: "child-task",
				Type: arc.TaskType,
			}
			Expect(taskSvc.NewWriter(tx).Create(ctx, t)).To(Succeed())

			Expect(otg.NewWriter(tx).DefineRelationships(
				ctx,
				a.OntologyID(),
				ontology.RelationshipTypeParentOf,
				t.OntologyID(),
			)).To(Succeed())

			Expect(svc.NewWriter(tx).Delete(ctx, a.Key)).To(Succeed())

			Expect(svc.NewRetrieve().Where(arc.MatchKeys(a.Key)).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
			Expect(taskSvc.NewRetrieve().Where(task.MatchKeys(t.Key)).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
		})

		It(
			"Should handle Arc deletion when Arc has no child tasks",
			func(ctx SpecContext) {
				a := arc.Arc{Name: "arc-without-tasks", Mode: arc.ModeText}
				Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
				Expect(svc.NewWriter(tx).Delete(ctx, a.Key)).To(Succeed())
				Expect(svc.NewRetrieve().Where(arc.MatchKeys(a.Key)).Exec(ctx, tx)).
					To(MatchError(query.ErrNotFound))
			},
		)

		It(
			"Should delete multiple child tasks when deleting an Arc",
			func(ctx SpecContext) {
				a := arc.Arc{Name: "arc-with-multiple-tasks", Mode: arc.ModeText}
				Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())

				t1 := &task.Task{
					Rack: testRack.Key,
					Name: "child-task-1",
					Type: arc.TaskType,
				}
				t2 := &task.Task{
					Rack: testRack.Key,
					Name: "child-task-2",
					Type: arc.TaskType,
				}
				Expect(taskSvc.NewWriter(tx).Create(ctx, t1)).To(Succeed())
				Expect(taskSvc.NewWriter(tx).Create(ctx, t2)).To(Succeed())

				otgWriter := otg.NewWriter(tx)
				Expect(otgWriter.DefineRelationships(
					ctx,
					a.OntologyID(),
					ontology.RelationshipTypeParentOf,
					t1.OntologyID(),
				)).To(Succeed())
				Expect(otgWriter.DefineRelationships(
					ctx,
					a.OntologyID(),
					ontology.RelationshipTypeParentOf,
					t2.OntologyID(),
				)).To(Succeed())

				Expect(svc.NewWriter(tx).Delete(ctx, a.Key)).To(Succeed())

				Expect(svc.NewRetrieve().Where(arc.MatchKeys(a.Key)).Exec(ctx, tx)).
					To(MatchError(query.ErrNotFound))
				Expect(
					taskSvc.NewRetrieve().Where(task.MatchKeys(t1.Key)).Exec(ctx, tx),
				).
					To(MatchError(query.ErrNotFound))
				Expect(
					taskSvc.NewRetrieve().Where(task.MatchKeys(t2.Key)).Exec(ctx, tx),
				).
					To(MatchError(query.ErrNotFound))
			},
		)
	})
})
