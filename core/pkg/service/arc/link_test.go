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
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Task links", func() {
	var a arc.Arc

	newArc := func(ctx SpecContext, tx gorp.Tx) arc.Arc {
		GinkgoHelper()
		created := arc.Arc{Name: "linked", Mode: arc.ModeText, Text: newText("a -> b")}
		Expect(svc.NewWriter(tx).Create(ctx, &created)).To(Succeed())
		return created
	}
	newTask := func(key arc.Key) task.Task {
		return task.Task{
			Rack:   testRack.Key,
			Name:   "linked",
			Type:   arc.TaskType,
			Config: msgpack.EncodedJSON{"arc_key": key.String()},
		}
	}
	createTask := func(ctx SpecContext, tx gorp.Tx, key arc.Key) task.Task {
		GinkgoHelper()
		t := newTask(key)
		Expect(taskSvc.NewWriter(tx).Create(ctx, &t)).To(Succeed())
		return t
	}
	taskKeys := func(ctx context.Context, tx gorp.Tx, key arc.Key) []task.Key {
		GinkgoHelper()
		return MustSucceed(svc.NewWriter(tx).TaskKeys(ctx, key))
	}
	unlink := func(ctx SpecContext, tx gorp.Tx, key arc.Key, t task.Task) {
		GinkgoHelper()
		Expect(otg.NewWriter(tx).DeleteRelationships(ctx, ontology.Relationship{
			From: arc.OntologyID(key),
			Type: ontology.RelationshipTypeParentOf,
			To:   t.OntologyID(),
		})).To(Succeed())
	}
	setArcKey := func(ctx SpecContext, t task.Task, key arc.Key) error {
		_, err := taskSvc.Dispatch(ctx, t.Key, "", []task.Action{
			task.NewSetAction(task.SetPayload{
				Path:  []string{"config", "arc_key"},
				Value: key.String(),
			}),
		})
		return err
	}

	BeforeEach(func(ctx SpecContext) { a = newArc(ctx, tx) })

	Describe("Create", func() {
		It("Should link a task to the Arc its config names", func(ctx SpecContext) {
			t := createTask(ctx, tx, a.Key)
			Expect(taskKeys(ctx, tx, a.Key)).To(ConsistOf([]task.Key{t.Key}))
		})
		It(
			"Should keep one link when the task is written again",
			func(ctx SpecContext) {
				t := createTask(ctx, tx, a.Key)
				Expect(otg.NewWriter(tx).DefineRelationships(
					ctx,
					a.OntologyID(),
					ontology.RelationshipTypeParentOf,
					t.OntologyID(),
				)).To(Succeed())
				Expect(taskSvc.NewWriter(tx).Create(ctx, &t)).To(Succeed())
				Expect(taskKeys(ctx, tx, a.Key)).To(ConsistOf([]task.Key{t.Key}))
			},
		)
		It(
			"Should move the link when the task names another Arc",
			func(ctx SpecContext) {
				t := createTask(ctx, tx, a.Key)
				other := newArc(ctx, tx)
				t.Config = msgpack.EncodedJSON{"arc_key": other.Key.String()}
				Expect(taskSvc.NewWriter(tx).Create(ctx, &t)).To(Succeed())
				Expect(taskKeys(ctx, tx, a.Key)).To(BeEmpty())
				Expect(taskKeys(ctx, tx, other.Key)).To(ConsistOf([]task.Key{t.Key}))
			},
		)
		It("Should link nothing when the config names no Arc", func(ctx SpecContext) {
			t := task.Task{
				Rack:   testRack.Key,
				Name:   "unnamed",
				Type:   arc.TaskType,
				Config: msgpack.EncodedJSON{},
			}
			Expect(taskSvc.NewWriter(tx).Create(ctx, &t)).To(Succeed())
			Expect(taskKeys(ctx, tx, uuid.Nil())).To(BeEmpty())
		})
		It("Should reject a task that names a missing Arc", func(ctx SpecContext) {
			t := newTask(uuid.New())
			Expect(taskSvc.NewWriter(tx).Create(ctx, &t)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("does not exist")),
			))
		})
		It("Should reject a task that names a deleted Arc", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Delete(ctx, a.Key)).To(Succeed())
			t := newTask(a.Key)
			Expect(taskSvc.NewWriter(tx).Create(ctx, &t)).
				To(MatchError(validate.ErrValidation))
		})
	})

	Describe("Dispatch", func() {
		It("Should move the link when a set names another Arc", func(ctx SpecContext) {
			first, second := newArc(ctx, nil), newArc(ctx, nil)
			t := createTask(ctx, nil, first.Key)
			Expect(setArcKey(ctx, t, second.Key)).To(Succeed())
			Expect(taskKeys(ctx, nil, first.Key)).To(BeEmpty())
			Expect(taskKeys(ctx, nil, second.Key)).To(ConsistOf([]task.Key{t.Key}))
		})
		It("Should reject a set that names a missing Arc", func(ctx SpecContext) {
			first := newArc(ctx, nil)
			t := createTask(ctx, nil, first.Key)
			Expect(setArcKey(ctx, t, uuid.New())).To(MatchError(validate.ErrValidation))
			Expect(taskKeys(ctx, nil, first.Key)).To(ConsistOf([]task.Key{t.Key}))
		})
		It("Should rename a task whose Arc is gone", func(ctx SpecContext) {
			first := newArc(ctx, nil)
			t := createTask(ctx, nil, first.Key)
			unlink(ctx, nil, first.Key, t)
			Expect(svc.NewWriter(nil).Delete(ctx, first.Key)).To(Succeed())
			MustSucceed(taskSvc.Dispatch(ctx, t.Key, "", []task.Action{
				task.NewSetAction(task.SetPayload{
					Path:  []string{"name"},
					Value: "renamed",
				}),
			}))
		})
	})

	Describe("Copy", func() {
		It("Should link a copy to the Arc", func(ctx SpecContext) {
			t := createTask(ctx, tx, a.Key)
			c := MustSucceed(taskSvc.NewWriter(tx).Copy(ctx, t.Key, "copy", false))
			Expect(taskKeys(ctx, tx, a.Key)).To(ConsistOf(t.Key, c.Key))
		})
		It(
			"Should sync the task and its copy after an Arc edit",
			func(ctx SpecContext) {
				t := createTask(ctx, tx, a.Key)
				c := MustSucceed(taskSvc.NewWriter(tx).Copy(ctx, t.Key, "copy", false))
				a.Text = newText("c -> d")
				Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
				hash := MustSucceed(arc.Hash(a))
				for _, key := range []task.Key{t.Key, c.Key} {
					var stored task.Task
					Expect(taskSvc.NewRetrieve().
						Where(task.MatchKeys(key)).
						Entry(&stored).
						Exec(ctx, tx)).To(Succeed())
					Expect(stored.Config).To(HaveKeyWithValue("hash", hash))
				}
			},
		)
		It("Should not link a snapshot, so Arc delete keeps it", func(ctx SpecContext) {
			t := createTask(ctx, tx, a.Key)
			c := MustSucceed(taskSvc.NewWriter(tx).Copy(ctx, t.Key, "copy", true))
			Expect(taskKeys(ctx, tx, a.Key)).To(ConsistOf([]task.Key{t.Key}))
			Expect(svc.NewWriter(tx).Delete(ctx, a.Key)).To(Succeed())
			Expect(taskSvc.NewRetrieve().
				Where(task.MatchKeys(c.Key)).
				Exec(ctx, tx)).To(Succeed())
		})
	})

	Describe("NewTaskLink", func() {
		It("Should skip a task of another type", func(ctx SpecContext) {
			t := task.Task{
				Key:    uuid.New(),
				Type:   "other",
				Config: msgpack.EncodedJSON{"arc_key": a.Key.String()},
			}
			Expect(arc.NewTaskLink(otg)(ctx, tx, t)).To(Succeed())
			Expect(taskKeys(ctx, tx, a.Key)).To(BeEmpty())
		})
	})

	Describe("Delete", func() {
		It("Should delete a task that had no ontology call", func(ctx SpecContext) {
			t := createTask(ctx, tx, a.Key)
			Expect(svc.NewWriter(tx).Delete(ctx, a.Key)).To(Succeed())
			Expect(taskSvc.NewRetrieve().
				Where(task.MatchKeys(t.Key)).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("NewTaskLinkMigration", func() {
		run := func(ctx SpecContext) {
			GinkgoHelper()
			Expect(arc.NewTaskLinkMigration(otg, taskSvc).Run(
				ctx, tx, alamos.Instrumentation{},
			)).To(Succeed())
		}
		It("Should link a stored task with no link", func(ctx SpecContext) {
			t := createTask(ctx, tx, a.Key)
			unlink(ctx, tx, a.Key, t)
			Expect(taskKeys(ctx, tx, a.Key)).To(BeEmpty())
			run(ctx)
			Expect(taskKeys(ctx, tx, a.Key)).To(ConsistOf([]task.Key{t.Key}))
		})
		It("Should skip a task whose Arc is gone", func(ctx SpecContext) {
			t := createTask(ctx, tx, a.Key)
			unlink(ctx, tx, a.Key, t)
			Expect(svc.NewWriter(tx).Delete(ctx, a.Key)).To(Succeed())
			run(ctx)
			Expect(taskSvc.NewRetrieve().
				Where(task.MatchKeys(t.Key)).
				Exec(ctx, tx)).To(Succeed())
		})
	})
})
