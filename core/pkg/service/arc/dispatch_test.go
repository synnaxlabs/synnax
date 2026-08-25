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
	"sync"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/x/crdt"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Dispatch", func() {
	It(
		"Should apply a single SetNodePosition action to the target Arc",
		func(ctx SpecContext) {
			a := arc.Arc{
				Name: "dispatch-pos",
				Mode: arc.ModeGraph,
				Graph: graph.Graph{
					Nodes: graph.Nodes{
						{Key: "n1", Position: spatial.XY{X: 0, Y: 0}},
					},
				},
			}
			Expect(writer.Create(ctx, &a)).To(Succeed())
			Expect(svc.Dispatch(ctx, a.Key, "session-1", []arc.Action{
				arc.NewSetNodePositionAction(arc.SetNodePositionPayload{
					Key:      "n1",
					Position: spatial.XY{X: 100, Y: 200},
				}),
			})).To(Succeed())
			var res arc.Arc
			Expect(
				svc.NewRetrieve().
					Where(arc.MatchKeys(a.Key)).
					Entry(&res).
					Exec(ctx, nil),
			).To(Succeed())
			Expect(
				res.Graph.Nodes[0].Position,
			).To(Equal(spatial.XY{X: 100, Y: 200}))
		},
	)

	It(
		"Should apply a sequence of mixed actions atomically",
		func(ctx SpecContext) {
			a := arc.Arc{Name: "dispatch-seq", Mode: arc.ModeGraph}
			Expect(writer.Create(ctx, &a)).To(Succeed())
			Expect(svc.Dispatch(ctx, a.Key, "session-1", []arc.Action{
				arc.NewSetNodeAction(
					arc.SetNodePayload{Node: graph.Node{Key: "n1"}},
				),
				arc.NewSetNodeAction(
					arc.SetNodePayload{Node: graph.Node{Key: "n2"}},
				),
				arc.NewAddEdgeAction(
					arc.AddEdgePayload{Edge: graph.Edge{Edge: ir.Edge{
						Source: ir.Handle{Node: "n1", Param: "out"},
						Target: ir.Handle{Node: "n2", Param: "in"},
						Kind:   ir.EdgeKindContinuous,
					}}},
				),
			})).To(Succeed())
			var res arc.Arc
			Expect(
				svc.NewRetrieve().
					Where(arc.MatchKeys(a.Key)).
					Entry(&res).
					Exec(ctx, nil),
			).To(Succeed())
			Expect(res.Graph.Nodes).To(HaveLen(2))
			Expect(res.Graph.Edges).To(HaveLen(1))
		},
	)

	It(
		"Should notify subscribers with the dispatched scoped action on success",
		func(ctx SpecContext) {
			a := arc.Arc{Name: "observed", Mode: arc.ModeGraph}
			Expect(writer.Create(ctx, &a)).To(Succeed())
			rec := &Recorder[arc.Key, arc.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			actions := []arc.Action{
				arc.NewSetNodeAction(
					arc.SetNodePayload{Node: graph.Node{Key: "n1"}},
				),
				arc.NewRenameAction(arc.RenamePayload{Name: "observed-renamed"}),
			}
			Expect(
				svc.Dispatch(ctx, a.Key, "client-xyz", actions),
			).To(Succeed())
			seen := rec.Snapshot()
			Expect(seen).To(HaveLen(1))
			Expect(seen[0].Key).To(Equal(a.Key))
			Expect(seen[0].DispatchKey).To(Equal("client-xyz"))
			Expect(seen[0].Seq).To(BeNumerically(">", uint64(0)))
			Expect(seen[0].Actions).To(HaveLen(2))
			Expect(seen[0].Actions[0].Type).To(Equal(arc.ActionTypeSetNode))
			Expect(seen[0].Actions[1].Type).To(Equal(arc.ActionTypeRename))
		},
	)

	It(
		"Should stamp strictly increasing Seq values onto successive broadcasts",
		func(ctx SpecContext) {
			a := arc.Arc{Name: "seq-test", Mode: arc.ModeGraph}
			Expect(writer.Create(ctx, &a)).To(Succeed())
			rec := &Recorder[arc.Key, arc.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			action := []arc.Action{
				arc.NewSetNodeAction(
					arc.SetNodePayload{Node: graph.Node{Key: "n1"}},
				),
			}
			for range 3 {
				Expect(
					svc.Dispatch(ctx, a.Key, "client-xyz", action),
				).To(Succeed())
			}
			seen := rec.Snapshot()
			Expect(seen).To(HaveLen(3))
			Expect(seen[1].Seq).To(BeNumerically(">", seen[0].Seq))
			Expect(seen[2].Seq).To(BeNumerically(">", seen[1].Seq))
		},
	)

	It(
		"Should fail with query.ErrNotFound and not notify when the target Arc does not exist",
		func(ctx SpecContext) {
			rec := &Recorder[arc.Key, arc.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			Expect(
				svc.Dispatch(ctx, uuid.New(), "client-xyz", []arc.Action{
					arc.NewRenameAction(arc.RenamePayload{Name: "ghost"}),
				}),
			).To(MatchError(query.ErrNotFound))
			Expect(rec.Snapshot()).To(BeEmpty())
		},
	)
})

var _ = Describe("Collaborative text", func() {
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
	fetch := func(ctx SpecContext, key arc.Key) arc.Arc {
		GinkgoHelper()
		var got arc.Arc
		Expect(
			svc.NewRetrieve().Where(arc.MatchKeys(key)).Entry(&got).Exec(ctx, nil),
		).
			To(Succeed())
		return got
	}

	It(
		"Should materialize dispatched insertions into the Arc's text",
		func(ctx SpecContext) {
			a := &arc.Arc{Name: "collab-empty", Mode: arc.ModeText}
			Expect(writer.Create(ctx, a)).To(Succeed())

			client := crdt.New(2)
			Expect(
				svc.Dispatch(
					ctx,
					a.Key,
					"dk",
					toInsertActions(client.Insert(0, "hello")),
				),
			).
				To(Succeed())

			Expect(fetch(ctx, a.Key).Text.Materialize().Raw).To(Equal("hello"))
		},
	)

	It(
		"Should seed the document from raw on create and materialize a bootstrapped edit",
		func(ctx SpecContext) {
			a := &arc.Arc{Name: "collab-seeded", Mode: arc.ModeText}
			a.Text.Raw = "base"
			Expect(writer.Create(ctx, a)).To(Succeed())

			seeded := fetch(ctx, a.Key)
			client := crdt.New(2)
			client.Load(seeded.Text.Doc.Inserts, seeded.Text.Doc.Deletes)
			Expect(client.String()).To(Equal("base"))

			Expect(
				svc.Dispatch(
					ctx,
					a.Key,
					"dk",
					toInsertActions(client.Insert(0, "X")),
				),
			).
				To(Succeed())

			Expect(fetch(ctx, a.Key).Text.Materialize().Raw).To(Equal("Xbase"))
		},
	)

	It(
		"Should reclaim tombstoned characters only after the text falls quiet",
		func(ctx SpecContext) {
			base := telem.Now()
			arcClock = func() telem.TimeStamp { return base }
			DeferCleanup(func() { arcClock = telem.Now })

			rec := &Recorder[arc.Key, arc.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			forgottenCount := func() int {
				n := 0
				for _, s := range rec.Snapshot() {
					for _, op := range s.Actions {
						if op.Type == arc.ActionTypeForgetChars {
							n += len(op.ForgetChars.IDs)
						}
					}
				}
				return n
			}

			a := &arc.Arc{Name: "collab-sweep", Mode: arc.ModeText}
			a.Text.Raw = "hello world"
			Expect(writer.Create(ctx, a)).To(Succeed())
			client := crdt.New(2)
			seeded := fetch(ctx, a.Key)
			client.Load(seeded.Text.Doc.Inserts, seeded.Text.Doc.Deletes)

			Expect(
				svc.Dispatch(
					ctx,
					a.Key,
					"dk",
					toDeleteActions(client.Delete(6, 5)),
				),
			).
				To(Succeed())
			Expect(fetch(ctx, a.Key).Text.Doc.Deletes).To(HaveLen(5))

			base = base.Add(2 * telem.Second)
			Expect(
				svc.Dispatch(
					ctx,
					a.Key,
					"dk",
					toInsertActions(client.Insert(0, "X")),
				),
			).
				To(Succeed())
			Expect(fetch(ctx, a.Key).Text.Doc.Deletes).To(HaveLen(5))
			Expect(forgottenCount()).To(Equal(0))

			base = base.Add(10 * telem.Second)
			Expect(
				svc.Dispatch(
					ctx,
					a.Key,
					"dk",
					toInsertActions(client.Insert(0, "Y")),
				),
			).
				To(Succeed())

			Expect(forgottenCount()).To(Equal(5))
			swept := fetch(ctx, a.Key)
			Expect(swept.Text.Doc.Deletes).To(BeEmpty())
			Expect(swept.Text.Doc.Inserts).To(HaveLen(8))
			Expect(swept.Text.Materialize().Raw).To(Equal("YXhello "))
		},
	)

	It(
		"Should preserve every edit when dispatches race",
		func(ctx SpecContext) {
			a := &arc.Arc{Name: "collab-race", Mode: arc.ModeText}
			a.Text.Raw = "ranges.crea"
			Expect(writer.Create(ctx, a)).To(Succeed())
			client := crdt.New(2)
			seeded := fetch(ctx, a.Key)
			client.Load(seeded.Text.Doc.Inserts, seeded.Text.Doc.Deletes)

			// Author one keystroke per batch up front, then send every batch
			// concurrently: the op-log materializes independent of arrival order, so
			// the only way an op goes missing is a dispatch overwriting another.
			chars := []rune(`te{"Main"}`)
			batches := make([][]arc.Action, len(chars))
			for i, ch := range chars {
				batches[i] = toInsertActions(client.Insert(11+i, string(ch)))
			}
			var wg sync.WaitGroup
			for _, batch := range batches {
				wg.Go(func() {
					defer GinkgoRecover()
					Expect(svc.Dispatch(ctx, a.Key, "dk", batch)).To(Succeed())
				})
			}
			wg.Wait()

			Expect(fetch(ctx, a.Key).Text.Materialize().Raw).
				To(Equal(`ranges.create{"Main"}`))
		},
	)
})
