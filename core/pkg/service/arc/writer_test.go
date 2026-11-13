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
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create an Arc with generated key", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			a := newAutomation(
				arc.Base{
					Name:   "test-arc",
					Source: arc.Source{Variant: arc.TextSource{}},
				},
			)
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			Expect(a.Base().Key).ToNot(Equal(uuid.Nil()))
		})

		It(
			"Should return a validation error when the name is empty",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				a := newAutomation(
					arc.Base{Source: arc.Source{Variant: arc.TextSource{}}},
				)
				Expect(svc.NewWriter(tx).Create(ctx, &a)).
					To(MatchError(ContainSubstring("name: required")))
			},
		)

		It("Should reject a decoded Arc whose source is null", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			var a arc.Arc
			Expect(json.Unmarshal(
				[]byte(`{"type":"automation","name":"null-source","source":null}`),
				&a,
			)).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("source: required")),
			))
		})

		It("Should give a decoded Arc that omits its source a text source", func(
			ctx SpecContext,
		) {
			tx := DeferClose(db.OpenTx())
			var a arc.Arc
			Expect(json.Unmarshal(
				[]byte(`{"type":"automation","name":"no-source"}`),
				&a,
			)).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			Expect(a.Base().Source.Variant).To(BeAssignableToTypeOf(arc.TextSource{}))
		})

		It("Should create an Arc with explicit key", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			key := uuid.New()
			a := newAutomation(
				arc.Base{
					Key:    key,
					Name:   "test-arc-with-key",
					Source: arc.Source{Variant: arc.TextSource{}},
				},
			)
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			Expect(a.Base().Key).To(Equal(key))
		})

		It("Should put the Arc in the root directory", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			a := newAutomation(
				arc.Base{
					Name:   "parented-arc",
					Source: arc.Source{Variant: arc.TextSource{}},
				},
			)
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			root := rootDirectory(ctx, tx)
			Expect(root.Name).To(Equal("Arcs"))
			Expect(a.Base().Directory).To(HaveValue(Equal(root.Key)))
		})

		It("Should reject an Arc that is neither an automation nor a module", func(
			ctx SpecContext,
		) {
			tx := DeferClose(db.OpenTx())
			Expect(svc.NewWriter(tx).Create(ctx, &arc.Arc{})).
				To(MatchError(validate.ErrValidation))
		})

		It("Should refuse an Arc whose key exists", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			a := newModule(
				arc.Base{Name: "typed", Source: arc.Source{Variant: arc.TextSource{}}},
			)
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			key := a.Base().Key
			again := newAutomation(arc.Base{
				Key:    key,
				Name:   "retyped",
				Source: arc.Source{Variant: arc.TextSource{}},
			})
			Expect(svc.NewWriter(tx).Create(ctx, &again)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("arc %s already exists", key)),
			))
			var stored arc.Arc
			Expect(svc.NewRetrieve().
				Where(arc.MatchKeys(key)).
				Entry(&stored).
				Exec(ctx, tx)).To(Succeed())
			Expect(stored.Type()).To(Equal(arc.TypeModule))
			Expect(stored.Base().Name).To(Equal("typed"))
		})

		It("Should handle multiple Arc creations", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			a1 := newAutomation(
				arc.Base{Name: "arc-1", Source: arc.Source{Variant: arc.TextSource{}}},
			)
			a2 := newAutomation(
				arc.Base{Name: "arc-2", Source: arc.Source{Variant: arc.TextSource{}}},
			)
			Expect(svc.NewWriter(tx).Create(ctx, &a1)).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, &a2)).To(Succeed())
			Expect(a1.Base().Key).ToNot(Equal(uuid.Nil()))
			Expect(a2.Base().Key).ToNot(Equal(uuid.Nil()))
			Expect(a1.Base().Key).ToNot(Equal(a2.Base().Key))
		})

		It(
			"Should broadcast the create with the writer's dispatch key after commit",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				rec := &Recorder[arc.Key, arc.Action]{}
				DeferCleanup(svc.OnAction(rec.Record))
				v := newAutomation(
					arc.Base{
						Name:   "broadcast",
						Source: arc.Source{Variant: arc.TextSource{}},
					},
				)
				Expect(
					svc.NewWriter(tx).WithDispatchKey("dk-1").Create(ctx, &v),
				).To(Succeed())
				Expect(rec.Snapshot()).To(BeEmpty())
				Expect(tx.Commit(ctx)).To(Succeed())
				seen := rec.Snapshot()
				Expect(seen).To(HaveLen(1))
				Expect(seen[0].Key).To(Equal(v.Base().Key))
				Expect(seen[0].DispatchKey).To(Equal("dk-1"))
				Expect(seen[0].Actions).To(HaveLen(1))
				Expect(seen[0].Actions[0].Type).To(Equal(arc.ActionTypeCreate))
			},
		)

		It(
			"Should not broadcast a create whose transaction closes uncommitted",
			func(ctx SpecContext) {
				rec := &Recorder[arc.Key, arc.Action]{}
				DeferCleanup(svc.OnAction(rec.Record))
				uncommitted := db.OpenTx()
				v := newAutomation(
					arc.Base{
						Name:   "discarded",
						Source: arc.Source{Variant: arc.TextSource{}},
					},
				)
				Expect(svc.NewWriter(uncommitted).Create(ctx, &v)).To(Succeed())
				Expect(uncommitted.Close()).To(Succeed())
				Expect(rec.Snapshot()).To(BeEmpty())
			},
		)
	})

	Describe("CreateMany", func() {
		var tx gorp.Tx
		BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
		It("Should create multiple Arcs", func(ctx SpecContext) {
			arcs := []arc.Arc{
				newAutomation(
					arc.Base{
						Name:   "arc-many-1",
						Source: arc.Source{Variant: arc.TextSource{}},
					},
				),
				newAutomation(
					arc.Base{
						Name:   "arc-many-2",
						Source: arc.Source{Variant: arc.TextSource{}},
					},
				),
			}
			Expect(svc.NewWriter(tx).CreateMany(ctx, &arcs)).To(Succeed())

			var retrieved []arc.Arc
			Expect(svc.NewRetrieve().Where(arc.MatchKeys(
				arcs[0].Base().Key,
				arcs[1].Base().Key,
			)).Entries(&retrieved).Exec(ctx, tx)).To(Succeed())
			Expect(retrieved).To(HaveLen(2))
		})
	})

	Describe("Delete", func() {
		var tx gorp.Tx
		BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
		It("Should delete an Arc", func(ctx SpecContext) {
			a := newAutomation(arc.Base{
				Name:   "arc-to-delete",
				Source: arc.Source{Variant: arc.TextSource{Text: text.Text{}}},
			})
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, a.Base().Key)).To(Succeed())
			Expect(svc.NewRetrieve().Where(arc.MatchKeys(a.Base().Key)).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
		})

		It("Should delete multiple Arcs", func(ctx SpecContext) {
			a1 := newAutomation(
				arc.Base{
					Name:   "arc-to-delete-1",
					Source: arc.Source{Variant: arc.TextSource{}},
				},
			)
			a2 := newAutomation(
				arc.Base{
					Name:   "arc-to-delete-2",
					Source: arc.Source{Variant: arc.TextSource{}},
				},
			)
			w := svc.NewWriter(tx)
			Expect(w.Create(ctx, &a1)).To(Succeed())
			Expect(w.Create(ctx, &a2)).To(Succeed())
			Expect(
				svc.NewWriter(tx).Delete(ctx, a1.Base().Key, a2.Base().Key),
			).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(arc.MatchKeys(a1.Base().Key, a2.Base().Key)).
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
			a := newAutomation(
				arc.Base{
					Name:   "arc-with-task",
					Source: arc.Source{Variant: arc.TextSource{}},
				},
			)
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			b := newAutomation(arc.Base{
				Name:   "other-arc",
				Source: arc.Source{Variant: arc.TextSource{}},
			})
			Expect(svc.NewWriter(tx).Create(ctx, &b)).To(Succeed())

			t := &task.Task{
				Rack:   testRack.Key,
				Name:   "child-task",
				Type:   arc.TaskType,
				Config: msgpack.EncodedJSON{"arc_key": a.Base().Key.String()},
			}
			other := &task.Task{
				Rack:   testRack.Key,
				Name:   "other-arc-task",
				Type:   arc.TaskType,
				Config: msgpack.EncodedJSON{"arc_key": b.Base().Key.String()},
			}
			Expect(taskSvc.NewWriter(tx).Create(ctx, t)).To(Succeed())
			Expect(taskSvc.NewWriter(tx).Create(ctx, other)).To(Succeed())

			Expect(svc.NewWriter(tx).Delete(ctx, a.Base().Key)).To(Succeed())

			Expect(svc.NewRetrieve().Where(arc.MatchKeys(a.Base().Key)).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
			Expect(taskSvc.NewRetrieve().Where(task.MatchKeys(t.Key)).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
			Expect(taskSvc.NewRetrieve().
				Where(task.MatchKeys(other.Key)).
				Exists(ctx, tx)).To(BeTrue())
		})

		It(
			"Should delete the config records of its child tasks",
			func(ctx SpecContext) {
				a := newAutomation(arc.Base{
					Name:   "arc-with-config",
					Source: arc.Source{Variant: arc.TextSource{}},
				})
				Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
				t := &task.Task{
					Rack:   testRack.Key,
					Name:   "child-task",
					Type:   arc.TaskType,
					Config: msgpack.EncodedJSON{"arc_key": a.Base().Key.String()},
				}
				Expect(taskSvc.NewWriter(tx).Create(ctx, t)).To(Succeed())
				MustSucceed(arcTaskSvc.Config.Read(ctx, tx, t.Key))
				Expect(svc.NewWriter(tx).Delete(ctx, a.Base().Key)).To(Succeed())
				Expect(arcTaskSvc.Config.Read(ctx, tx, t.Key)).Error().
					To(MatchError(query.ErrNotFound))
			},
		)

		It(
			"Should handle Arc deletion when Arc has no child tasks",
			func(ctx SpecContext) {
				a := newAutomation(
					arc.Base{
						Name:   "arc-without-tasks",
						Source: arc.Source{Variant: arc.TextSource{}},
					},
				)
				Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
				Expect(svc.NewWriter(tx).Delete(ctx, a.Base().Key)).To(Succeed())
				Expect(
					svc.NewRetrieve().Where(arc.MatchKeys(a.Base().Key)).Exec(ctx, tx),
				).
					To(MatchError(query.ErrNotFound))
			},
		)

		It(
			"Should delete multiple child tasks when deleting an Arc",
			func(ctx SpecContext) {
				a := newAutomation(
					arc.Base{
						Name:   "arc-with-multiple-tasks",
						Source: arc.Source{Variant: arc.TextSource{}},
					},
				)
				Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())

				cfg := msgpack.EncodedJSON{"arc_key": a.Base().Key.String()}
				t1 := &task.Task{
					Rack:   testRack.Key,
					Name:   "child-task-1",
					Type:   arc.TaskType,
					Config: cfg,
				}
				t2 := &task.Task{
					Rack:   testRack.Key,
					Name:   "child-task-2",
					Type:   arc.TaskType,
					Config: cfg,
				}
				Expect(taskSvc.NewWriter(tx).Create(ctx, t1)).To(Succeed())
				Expect(taskSvc.NewWriter(tx).Create(ctx, t2)).To(Succeed())

				Expect(svc.NewWriter(tx).Delete(ctx, a.Base().Key)).To(Succeed())

				Expect(
					svc.NewRetrieve().Where(arc.MatchKeys(a.Base().Key)).Exec(ctx, tx),
				).
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
