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
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/crdt"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("SetRack", func() {
	var a arc.Arc

	createArc := func(ctx SpecContext, raw string) {
		a = arc.Arc{Name: "deployable", Mode: arc.ModeText, Text: newText(raw)}
		Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
	}

	retrieveTask := func(ctx SpecContext, key task.Key) task.Task {
		var tsk task.Task
		Expect(taskSvc.NewRetrieve().
			Where(task.MatchKeys(key)).
			Entry(&tsk).
			Exec(ctx, tx)).To(Succeed())
		return tsk
	}

	It(
		"Should create a task on the rack stamped with the arc's hash",
		func(ctx SpecContext) {
			createArc(ctx, "a -> b")
			tsk := MustSucceed(svc.NewWriter(tx).SetRack(ctx, a.Key, testRack.Key))
			Expect(tsk).ToNot(BeNil())
			Expect(tsk.Rack).To(Equal(testRack.Key))
			Expect(tsk.Type).To(Equal(arc.TaskType))
			Expect(tsk.Name).To(Equal(a.Name))
			Expect(tsk.Config).To(HaveKeyWithValue("arc_key", a.Key.String()))
			Expect(tsk.Config).To(HaveKey("hash"))
			stored := retrieveTask(ctx, tsk.Key)
			Expect(stored.ConfigHash).ToNot(BeEmpty())
		},
	)

	It("Should reuse the task when the rack is set again", func(ctx SpecContext) {
		createArc(ctx, "a -> b")
		w := svc.NewWriter(tx)
		first := MustSucceed(w.SetRack(ctx, a.Key, testRack.Key))
		second := MustSucceed(w.SetRack(ctx, a.Key, testRack.Key))
		Expect(second.Key).To(Equal(first.Key))
		stored := retrieveTask(ctx, first.Key)
		Expect(stored.ConfigHash).To(Equal(retrieveTask(ctx, second.Key).ConfigHash))
	})

	It("Should keep the task's identity on a rack move", func(ctx SpecContext) {
		createArc(ctx, "a -> b")
		other := &rack.Rack{Name: "Other Rack"}
		Expect(rackSvc.NewWriter(tx).Create(ctx, other)).To(Succeed())
		w := svc.NewWriter(tx)
		first := MustSucceed(w.SetRack(ctx, a.Key, testRack.Key))
		moved := MustSucceed(w.SetRack(ctx, a.Key, other.Key))
		Expect(moved.Key).To(Equal(first.Key))
		Expect(retrieveTask(ctx, first.Key).Rack).To(Equal(other.Key))
	})

	It(
		"Should restamp the hash when the rack is set after a program change",
		func(ctx SpecContext) {
			createArc(ctx, "a -> b")
			w := svc.NewWriter(tx)
			first := MustSucceed(w.SetRack(ctx, a.Key, testRack.Key))
			firstHash := retrieveTask(ctx, first.Key).ConfigHash
			edited := arc.Arc{
				Key:  a.Key,
				Name: a.Name,
				Mode: arc.ModeText,
				Text: newText("a -> c"),
			}
			Expect(w.Create(ctx, &edited)).To(Succeed())
			second := MustSucceed(w.SetRack(ctx, a.Key, testRack.Key))
			Expect(retrieveTask(ctx, second.Key).ConfigHash).ToNot(Equal(firstHash))
		},
	)

	It("Should clear the rack by deleting the task", func(ctx SpecContext) {
		createArc(ctx, "a -> b")
		w := svc.NewWriter(tx)
		tsk := MustSucceed(w.SetRack(ctx, a.Key, testRack.Key))
		Expect(MustSucceed(w.SetRack(ctx, a.Key, 0))).To(BeNil())
		Expect(taskSvc.NewRetrieve().Where(task.MatchKeys(tsk.Key)).Exec(ctx, tx)).
			To(MatchError(query.ErrNotFound))
	})

	It(
		"Should be a no-op to clear the rack of an arc with no task",
		func(ctx SpecContext) {
			createArc(ctx, "a -> b")
			Expect(MustSucceed(svc.NewWriter(tx).SetRack(ctx, a.Key, 0))).To(BeNil())
		},
	)

	It("Should reject clearing the rack of a running arc", func(ctx SpecContext) {
		createArc(ctx, "a -> b")
		w := svc.NewWriter(tx)
		tsk := MustSucceed(w.SetRack(ctx, a.Key, testRack.Key))
		stored := retrieveTask(ctx, tsk.Key)
		stat := task.Status{
			Key:     stored.OntologyID().String(),
			Name:    stored.Name,
			Variant: status.VariantSuccess,
			Message: "running",
			Time:    telem.Now(),
			Details: task.NewStatusDetails(stored, true),
		}
		Expect(statusSvc.NewWriter(tx).Set(ctx, &stat)).To(Succeed())
		Expect(w.SetRack(ctx, a.Key, 0)).Error().
			To(MatchError(validate.ErrValidation))
		Expect(taskSvc.NewRetrieve().Where(task.MatchKeys(tsk.Key)).Exec(ctx, tx)).
			To(Succeed())
	})

	It("Should return not found for a nonexistent arc", func(ctx SpecContext) {
		Expect(svc.NewWriter(tx).SetRack(ctx, uuid.New(), testRack.Key)).Error().
			To(MatchError(query.ErrNotFound))
	})
})

var _ = Describe("Task sync", func() {
	var a arc.Arc

	createArc := func(ctx SpecContext, mode arc.Mode) {
		a = arc.Arc{Name: "syncable", Mode: mode}
		Expect(writer.Create(ctx, &a)).To(Succeed())
	}

	retrieveTask := func(ctx SpecContext, key task.Key) task.Task {
		GinkgoHelper()
		var tsk task.Task
		Expect(taskSvc.NewRetrieve().
			Where(task.MatchKeys(key)).
			Entry(&tsk).
			Exec(ctx, tx)).To(Succeed())
		return tsk
	}

	// taskConfigHash returns a poll function for Eventually/Consistently: the task
	// sync runs on a debouncer, so hash assertions after a dispatch must wait.
	taskConfigHash := func(ctx SpecContext, key task.Key) func() string {
		return func() string {
			var tsk task.Task
			if err := taskSvc.NewRetrieve().
				Where(task.MatchKeys(key)).
				Entry(&tsk).
				Exec(ctx, nil); err != nil {
				return ""
			}
			return tsk.ConfigHash
		}
	}

	toInsertActions := func(inserts []crdt.Insert) []arc.Action {
		out := make([]arc.Action, len(inserts))
		for i, op := range inserts {
			out[i] = arc.NewInsertCharAction(arc.InsertCharPayload{
				ID:     op.ID,
				Origin: op.Origin,
				Side:   op.Side,
				Char:   op.Char,
			})
		}
		return out
	}

	toDeleteActions := func(deletes []crdt.Delete) []arc.Action {
		out := make([]arc.Action, len(deletes))
		for i, op := range deletes {
			out[i] = arc.NewDeleteCharAction(arc.DeleteCharPayload{ID: op.ID})
		}
		return out
	}

	setNode := func(x float64) []arc.Action {
		return []arc.Action{arc.NewSetNodeAction(arc.SetNodePayload{
			Node: graph.Node{Key: "n1", Position: spatial.XY{X: x}},
		})}
	}

	It(
		"Should rewrite the task config when an edit changes the content",
		func(ctx SpecContext) {
			createArc(ctx, arc.ModeText)
			tsk := MustSucceed(writer.SetRack(ctx, a.Key, testRack.Key))
			before := retrieveTask(ctx, tsk.Key)
			client := crdt.New(2)
			Expect(
				svc.Dispatch(ctx, a.Key, "dk", toInsertActions(client.Insert(0, "x"))),
			).
				To(Succeed())
			Eventually(taskConfigHash(ctx, tsk.Key)).
				ShouldNot(Equal(before.ConfigHash))
			after := retrieveTask(ctx, tsk.Key)
			Expect(after.Config["hash"]).ToNot(Equal(before.Config["hash"]))
		},
	)

	It("Should not create a task for an arc with no rack bound", func(ctx SpecContext) {
		unbound := arc.Arc{Name: "unbound", Mode: arc.ModeText}
		Expect(writer.Create(ctx, &unbound)).To(Succeed())
		client := crdt.New(2)
		Expect(
			svc.Dispatch(
				ctx,
				unbound.Key,
				"dk",
				toInsertActions(client.Insert(0, "x")),
			),
		).To(Succeed())
		tasksByName := func(g Gomega) {
			var tasks []task.Task
			g.Expect(taskSvc.NewRetrieve().
				Where(task.MatchNames(unbound.Name)).
				Entries(&tasks).
				Exec(ctx, nil)).To(Succeed())
			g.Expect(tasks).To(BeEmpty())
		}
		Consistently(
			tasksByName,
			time.Millisecond*100,
			time.Millisecond*10,
		).Should(Succeed())
	})

	It("Should leave the task config on a layout-only edit", func(ctx SpecContext) {
		createArc(ctx, arc.ModeGraph)
		Expect(svc.Dispatch(ctx, a.Key, "dk", setNode(0))).To(Succeed())
		tsk := MustSucceed(writer.SetRack(ctx, a.Key, testRack.Key))
		before := retrieveTask(ctx, tsk.Key)
		Expect(svc.Dispatch(ctx, a.Key, "dk", setNode(9))).To(Succeed())
		Consistently(
			taskConfigHash(ctx, tsk.Key),
			time.Millisecond*100,
			time.Millisecond*10,
		).
			Should(Equal(before.ConfigHash))
	})

	It(
		"Should rewrite the task config when an overwrite changes the content",
		func(ctx SpecContext) {
			createArc(ctx, arc.ModeText)
			w := svc.NewWriter(tx)
			tsk := MustSucceed(w.SetRack(ctx, a.Key, testRack.Key))
			before := retrieveTask(ctx, tsk.Key)
			edited := arc.Arc{
				Key:  a.Key,
				Name: a.Name,
				Mode: arc.ModeText,
				Text: newText("a -> c"),
			}
			Expect(w.Create(ctx, &edited)).To(Succeed())
			Expect(retrieveTask(ctx, tsk.Key).ConfigHash).
				ToNot(Equal(before.ConfigHash))
		},
	)

	It(
		"Should hash the stored doc when an overwrite carries a mismatched raw",
		func(ctx SpecContext) {
			deployed := arc.Arc{
				Name: "lying-raw",
				Mode: arc.ModeText,
				Text: newText("a -> b"),
			}
			w := svc.NewWriter(tx)
			Expect(w.Create(ctx, &deployed)).To(Succeed())
			tsk := MustSucceed(w.SetRack(ctx, deployed.Key, testRack.Key))
			before := retrieveTask(ctx, tsk.Key)
			lying := newText("a -> b")
			lying.Raw = "a -> c"
			edited := arc.Arc{
				Key:  deployed.Key,
				Name: deployed.Name,
				Mode: arc.ModeText,
				Text: lying,
			}
			Expect(w.Create(ctx, &edited)).To(Succeed())
			Expect(retrieveTask(ctx, tsk.Key).Config["hash"]).
				To(Equal(before.Config["hash"]))
		},
	)

	It(
		"Should restore the deployed config when an edit is undone",
		func(ctx SpecContext) {
			createArc(ctx, arc.ModeText)
			tsk := MustSucceed(writer.SetRack(ctx, a.Key, testRack.Key))
			before := retrieveTask(ctx, tsk.Key)
			client := crdt.New(2)
			Expect(
				svc.Dispatch(ctx, a.Key, "dk", toInsertActions(client.Insert(0, "x"))),
			).
				To(Succeed())
			Expect(
				svc.Dispatch(ctx, a.Key, "dk", toDeleteActions(client.Delete(0, 1))),
			).
				To(Succeed())
			Consistently(
				taskConfigHash(ctx, tsk.Key),
				time.Millisecond*100,
				time.Millisecond*10,
			).
				Should(Equal(before.ConfigHash))
		},
	)
})
