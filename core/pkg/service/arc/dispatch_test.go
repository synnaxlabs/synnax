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
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/x/crdt"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Dispatch", func() {
	It(
		"Should reject a create without changing the Arc it names",
		func(ctx SpecContext) {
			target := newAutomation(
				arc.Base{
					Name:   "target",
					Source: arc.Source{Variant: arc.GraphSource{}},
				},
			)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &target)
			})).To(Succeed())
			other := newAutomation(
				arc.Base{Name: "other", Source: arc.Source{Variant: arc.GraphSource{}}},
			)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &other)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).
						Delete(ctx, target.Base().Key, other.Base().Key)
				})).To(Succeed())
			})
			Expect(dispatch(ctx, target.Base().Key, "d1", []arc.Action{
				arc.NewCreateAction(arc.CreatePayload{
					Arc: newAutomation(arc.Base{
						Key:    other.Base().Key,
						Name:   "overwritten",
						Source: arc.Source{Variant: arc.GraphSource{}},
					}),
				}),
			})).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("only the Core dispatches an Arc create")),
			))
			var res arc.Arc
			Expect(svc.NewRetrieve().
				Where(arc.MatchKeys(other.Base().Key)).
				Entry(&res).
				Exec(ctx, db)).To(Succeed())
			Expect(res.Base().Name).To(Equal("other"))
		},
	)

	It(
		"Should apply a single SetNodePosition action to the target Arc",
		func(ctx SpecContext) {
			a := newAutomation(arc.Base{
				Name: "dispatch-pos",
				Source: arc.Source{Variant: arc.GraphSource{Graph: graph.Graph{
					Nodes: graph.Nodes{
						{Key: "n1", Position: spatial.XY{X: 0, Y: 0}},
					},
				}}},
			})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &a)
			})).To(Succeed())
			Expect(dispatch(ctx, a.Base().Key, "session-1", []arc.Action{
				arc.NewSetNodePositionAction(arc.SetNodePositionPayload{
					Key:      "n1",
					Position: spatial.XY{X: 100, Y: 200},
				}),
			})).To(Succeed())
			var res arc.Arc
			Expect(
				svc.NewRetrieve().
					Where(arc.MatchKeys(a.Base().Key)).
					Entry(&res).
					Exec(ctx, db),
			).To(Succeed())
			Expect(
				graphOf(res.Base()).Nodes[0].Position,
			).To(Equal(spatial.XY{X: 100, Y: 200}))
		},
	)

	It(
		"Should apply a sequence of mixed actions atomically",
		func(ctx SpecContext) {
			a := newAutomation(
				arc.Base{
					Name:   "dispatch-seq",
					Source: arc.Source{Variant: arc.GraphSource{}},
				},
			)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &a)
			})).To(Succeed())
			Expect(dispatch(ctx, a.Base().Key, "session-1", []arc.Action{
				arc.NewSetNodeAction(
					arc.SetNodePayload{Node: graph.Node{Key: "n1"}},
				),
				arc.NewSetNodeAction(
					arc.SetNodePayload{Node: graph.Node{Key: "n2"}},
				),
				arc.NewAddEdgeAction(
					arc.AddEdgePayload{
						Edge: graph.Edge{
							Source: ir.Handle{Node: "n1", Param: "out"},
							Target: ir.Handle{Node: "n2", Param: "in"},
							Kind:   ir.EdgeKindContinuous,
						},
					},
				),
			})).To(Succeed())
			var res arc.Arc
			Expect(
				svc.NewRetrieve().
					Where(arc.MatchKeys(a.Base().Key)).
					Entry(&res).
					Exec(ctx, db),
			).To(Succeed())
			Expect(graphOf(res.Base()).Nodes).To(HaveLen(2))
			Expect(graphOf(res.Base()).Edges).To(HaveLen(1))
		},
	)

	It(
		"Should notify subscribers with the dispatched scoped action on success",
		func(ctx SpecContext) {
			a := newAutomation(
				arc.Base{
					Name:   "observed",
					Source: arc.Source{Variant: arc.GraphSource{}},
				},
			)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &a)
			})).To(Succeed())
			rec := &Recorder[arc.Key, arc.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			actions := []arc.Action{
				arc.NewSetNodeAction(
					arc.SetNodePayload{Node: graph.Node{Key: "n1"}},
				),
				arc.NewSetAction(
					arc.SetPayload{Path: []string{"name"}, Value: "observed-renamed"},
				),
			}
			Expect(
				dispatch(ctx, a.Base().Key, "client-xyz", actions),
			).To(Succeed())
			seen := rec.Snapshot()
			Expect(seen).To(HaveLen(1))
			Expect(seen[0].Key).To(Equal(a.Base().Key))
			Expect(seen[0].DispatchKey).To(Equal("client-xyz"))
			Expect(seen[0].Seq).To(BeNumerically(">", uint64(0)))
			Expect(seen[0].Actions).To(HaveLen(2))
			Expect(seen[0].Actions[0].Type).To(Equal(arc.ActionTypeSetNode))
			Expect(seen[0].Actions[1].Type).To(Equal(arc.ActionTypeSet))
		},
	)

	It(
		"Should stamp strictly increasing Seq values onto successive broadcasts",
		func(ctx SpecContext) {
			a := newAutomation(
				arc.Base{
					Name:   "seq-test",
					Source: arc.Source{Variant: arc.GraphSource{}},
				},
			)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &a)
			})).To(Succeed())
			rec := &Recorder[arc.Key, arc.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			action := []arc.Action{
				arc.NewSetNodeAction(
					arc.SetNodePayload{Node: graph.Node{Key: "n1"}},
				),
			}
			for range 3 {
				Expect(
					dispatch(ctx, a.Base().Key, "client-xyz", action),
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
				dispatch(ctx, uuid.New(), "client-xyz", []arc.Action{
					arc.NewSetAction(
						arc.SetPayload{Path: []string{"name"}, Value: "ghost"},
					),
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
			svc.NewRetrieve().Where(arc.MatchKeys(key)).Entry(&got).Exec(ctx, db),
		).
			To(Succeed())
		return got
	}

	It(
		"Should materialize dispatched insertions into the Arc's text",
		func(ctx SpecContext) {
			a := newAutomation(
				arc.Base{
					Name:   "collab-empty",
					Source: arc.Source{Variant: arc.TextSource{}},
				},
			)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &a)
			})).To(Succeed())

			client := crdt.New(2)
			Expect(
				dispatch(
					ctx,
					a.Base().Key,
					"dk",
					toInsertActions(client.Insert(0, "hello")),
				),
			).
				To(Succeed())

			Expect(
				textOf(fetch(ctx, a.Base().Key).Base()).Materialize().Raw,
			).To(Equal("hello"))
		},
	)

	It(
		"Should seed the document from raw on create and materialize a bootstrapped edit",
		func(ctx SpecContext) {
			a := newAutomation(arc.Base{
				Name: "collab-seeded",
				Source: arc.Source{
					Variant: arc.TextSource{Text: text.Text{Raw: "base"}},
				},
			})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &a)
			})).To(Succeed())

			seeded := fetch(ctx, a.Base().Key)
			client := crdt.New(2)
			client.Load(
				textOf(seeded.Base()).Doc.Inserts,
				textOf(seeded.Base()).Doc.Deletes,
			)
			Expect(client.String()).To(Equal("base"))

			Expect(
				dispatch(
					ctx,
					a.Base().Key,
					"dk",
					toInsertActions(client.Insert(0, "X")),
				),
			).
				To(Succeed())

			Expect(
				textOf(fetch(ctx, a.Base().Key).Base()).Materialize().Raw,
			).To(Equal("Xbase"))
		},
	)

	It(
		"Should reclaim tombstoned characters only after the text falls quiet",
		func(ctx SpecContext) {
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

			a := newAutomation(arc.Base{
				Name: "collab-sweep",
				Source: arc.Source{
					Variant: arc.TextSource{Text: text.Text{Raw: "hello world"}},
				},
			})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &a)
			})).To(Succeed())
			client := crdt.New(2)
			seeded := fetch(ctx, a.Base().Key)
			client.Load(
				textOf(seeded.Base()).Doc.Inserts,
				textOf(seeded.Base()).Doc.Deletes,
			)

			Expect(
				dispatch(
					ctx,
					a.Base().Key,
					"dk",
					toDeleteActions(client.Delete(6, 5)),
				),
			).
				To(Succeed())
			Expect(textOf(fetch(ctx, a.Base().Key).Base()).Doc.Deletes).To(HaveLen(5))

			clock.Advance(2 * time.Second)
			Expect(
				dispatch(
					ctx,
					a.Base().Key,
					"dk",
					toInsertActions(client.Insert(0, "X")),
				),
			).
				To(Succeed())
			Expect(textOf(fetch(ctx, a.Base().Key).Base()).Doc.Deletes).To(HaveLen(5))
			Expect(forgottenCount()).To(Equal(0))

			clock.Advance(10 * time.Second)
			Expect(
				dispatch(
					ctx,
					a.Base().Key,
					"dk",
					toInsertActions(client.Insert(0, "Y")),
				),
			).
				To(Succeed())

			Expect(forgottenCount()).To(Equal(5))
			swept := fetch(ctx, a.Base().Key)
			Expect(textOf(swept.Base()).Doc.Deletes).To(BeEmpty())
			Expect(textOf(swept.Base()).Doc.Inserts).To(HaveLen(8))
			Expect(textOf(swept.Base()).Materialize().Raw).To(Equal("YXhello "))
		},
	)

	It(
		"Should keep the quiet window of an Arc whose delete rolls back",
		func(ctx SpecContext) {
			a := newAutomation(arc.Base{
				Name: "collab-sweep-rollback",
				Source: arc.Source{
					Variant: arc.TextSource{Text: text.Text{Raw: "hello world"}},
				},
			})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &a)
			})).To(Succeed())
			client := crdt.New(2)
			seeded := fetch(ctx, a.Base().Key)
			client.Load(
				textOf(seeded.Base()).Doc.Inserts,
				textOf(seeded.Base()).Doc.Deletes,
			)
			Expect(dispatch(
				ctx,
				a.Base().Key,
				"dk",
				toDeleteActions(client.Delete(6, 5)),
			)).To(Succeed())
			clock.Advance(2 * time.Second)
			rollback := errors.New("rolled back")
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := svc.NewWriter(tx).Delete(ctx, a.Base().Key); err != nil {
					return err
				}
				return rollback
			})).To(MatchError(rollback))
			Expect(dispatch(
				ctx,
				a.Base().Key,
				"dk",
				toInsertActions(client.Insert(0, "X")),
			)).To(Succeed())
			Expect(textOf(fetch(ctx, a.Base().Key).Base()).Doc.Deletes).To(HaveLen(5))
		},
	)

	It(
		"Should not move the edit time of a text edit that rolls back",
		func(ctx SpecContext) {
			a := newAutomation(arc.Base{
				Name: "collab-sweep-denied",
				Source: arc.Source{
					Variant: arc.TextSource{Text: text.Text{Raw: "hello world"}},
				},
			})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &a)
			})).To(Succeed())
			load := func() *crdt.Text {
				GinkgoHelper()
				c := crdt.New(2)
				doc := textOf(fetch(ctx, a.Base().Key).Base()).Doc
				c.Load(doc.Inserts, doc.Deletes)
				return c
			}
			Expect(dispatch(
				ctx,
				a.Base().Key,
				"dk",
				toDeleteActions(load().Delete(6, 5)),
			)).To(Succeed())
			clock.Advance(10 * time.Second)
			rollback := errors.New("rolled back")
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := svc.NewWriter(tx).Apply(
					ctx,
					a.Base().Key,
					toInsertActions(load().Insert(0, "Z"))...,
				); err != nil {
					return err
				}
				return rollback
			})).To(MatchError(rollback))
			Expect(dispatch(
				ctx,
				a.Base().Key,
				"dk",
				toInsertActions(load().Insert(0, "X")),
			)).To(Succeed())
			Expect(textOf(fetch(ctx, a.Base().Key).Base()).Doc.Deletes).To(BeEmpty())
		},
	)

	It(
		"Should attach the sweep after the edit in the same commit",
		func(ctx SpecContext) {
			a := newAutomation(arc.Base{
				Name: "collab-sweep-order",
				Source: arc.Source{
					Variant: arc.TextSource{Text: text.Text{Raw: "hello world"}},
				},
			})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &a)
			})).To(Succeed())
			client := crdt.New(2)
			seeded := fetch(ctx, a.Base().Key)
			client.Load(
				textOf(seeded.Base()).Doc.Inserts,
				textOf(seeded.Base()).Doc.Deletes,
			)
			Expect(dispatch(
				ctx,
				a.Base().Key,
				"dk",
				toDeleteActions(client.Delete(6, 5)),
			)).To(Succeed())
			clock.Advance(12 * time.Second)
			rec := &Recorder[arc.Key, arc.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			edit := toInsertActions(client.Insert(0, "Y"))
			Expect(dispatch(ctx, a.Base().Key, "dk", edit)).To(Succeed())
			seen := rec.Snapshot()
			Expect(seen).To(HaveLen(2))
			Expect(seen[0].DispatchKey).To(Equal("dk"))
			Expect(seen[0].Actions).To(Equal(edit))
			Expect(seen[1].DispatchKey).To(BeEmpty())
			Expect(seen[1].Actions).To(HaveExactElements(
				HaveField("Type", arc.ActionTypeForgetChars),
			))
			Expect(seen[1].Seq).To(Equal(seen[0].Seq + 1))
		},
	)

	It(
		"Should preserve every edit when dispatches race",
		func(ctx SpecContext) {
			a := newAutomation(arc.Base{
				Name: "collab-race",
				Source: arc.Source{
					Variant: arc.TextSource{Text: text.Text{Raw: "ranges.crea"}},
				},
			})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &a)
			})).To(Succeed())
			client := crdt.New(2)
			seeded := fetch(ctx, a.Base().Key)
			client.Load(
				textOf(seeded.Base()).Doc.Inserts,
				textOf(seeded.Base()).Doc.Deletes,
			)

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
					Expect(dispatch(ctx, a.Base().Key, "dk", batch)).To(Succeed())
				})
			}
			wg.Wait()

			Expect(textOf(fetch(ctx, a.Base().Key).Base()).Materialize().Raw).
				To(Equal(`ranges.create{"Main"}`))
		},
	)
})
