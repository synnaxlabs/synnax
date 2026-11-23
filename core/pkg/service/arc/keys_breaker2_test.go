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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	arclang "github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/crdt"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// deployMarker marks the context of the compile that hands a Driver its program.
type deployMarker struct{}

var _ = Describe("Compile keys breaker 2", Ordered, ContinueOnFailure, func() {
	var in, out, other channel.Channel
	newChannel := func(ctx SpecContext, name string) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{Name: name, Virtual: true, DataType: telem.Float64T}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	write := func(ctx context.Context, fn func(arc.Writer) error) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error { return fn(svc.NewWriter(tx)) })
	}
	createArc := func(ctx SpecContext, b arc.Base) arc.Arc {
		GinkgoHelper()
		a := newAutomation(b)
		Expect(write(ctx, func(w arc.Writer) error { return w.Create(ctx, &a) })).
			To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(write(ctx, func(w arc.Writer) error {
				return w.Delete(ctx, a.Base().Key)
			})).To(Succeed())
		})
		return a
	}
	textBase := func(name, raw string) arc.Base {
		return arc.Base{
			Name:   name,
			Source: arc.Source{Variant: arc.TextSource{Text: text.Text{Raw: raw}}},
		}
	}
	graphBase := func(name string, source, sink channel.Key) arc.Base {
		return arc.Base{
			Name: name,
			Source: arc.Source{Variant: arc.GraphSource{Graph: graph.Graph{
				Nodes: graph.Nodes{{Key: "source"}, {Key: "sink"}},
				Edges: graph.Edges{{
					Key:    "edge",
					Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
					Target: ir.Handle{Node: "sink", Param: ir.DefaultInputParam},
				}},
				Inputs: map[string]msgpack.EncodedJSON{
					"source": {"type": "on", "channel": float64(source)},
					"sink":   {"type": "write", "channel": float64(sink)},
				},
			}}},
		}
	}
	retrieveArc := func(ctx context.Context, key arc.Key) arc.Arc {
		GinkgoHelper()
		var a arc.Arc
		Expect(svc.NewRetrieve().
			Where(arc.MatchKeys(key)).
			Entry(&a).
			Exec(ctx, db)).To(Succeed())
		return a
	}
	lists := func(ctx context.Context, key arc.Key) func() [2][]channel.Key {
		return func() [2][]channel.Key {
			b := retrieveArc(ctx, key).Base()
			return [2][]channel.Key{b.Reads, b.Writes}
		}
	}
	want := func(r, w channel.Channel) [2][]channel.Key {
		return [2][]channel.Key{{r.Key()}, {w.Key()}}
	}
	var editor uint32 = 9000
	// overwrite replaces the text of the Arc with key by raw.
	overwrite := func(ctx context.Context, key arc.Key, raw string) {
		GinkgoHelper()
		src, ok := retrieveArc(ctx, key).Base().Source.Variant.(arc.TextSource)
		Expect(ok).To(BeTrue())
		editor++
		client := crdt.New(editor)
		client.Load(src.Text.Doc.Inserts, src.Text.Doc.Deletes)
		deletes := client.Delete(0, client.Len())
		inserts := client.Insert(0, raw)
		acts := make([]arc.Action, 0, len(deletes)+len(inserts))
		for _, op := range deletes {
			acts = append(
				acts,
				arc.NewDeleteCharAction(arc.DeleteCharPayload{ID: op.ID}),
			)
		}
		for _, op := range inserts {
			acts = append(acts, arc.NewInsertCharAction(arc.InsertCharPayload{
				ID: op.ID, Origin: op.Origin, Side: op.Side, Char: op.Char,
			}))
		}
		Expect(dispatch(ctx, key, "kb2", acts)).To(Succeed())
	}
	retrieveTask := func(ctx SpecContext, snap gorp.Snapshot, key task.Key) task.Task {
		GinkgoHelper()
		var tsk task.Task
		Expect(taskSvc.NewRetrieve().
			Where(task.MatchKeys(key)).
			Entry(&tsk).
			Exec(ctx, snap)).To(Succeed())
		return tsk
	}
	// deploy binds the Arc with key to a task and takes its program as the embedded
	// Driver's configure does.
	deploy := func(ctx SpecContext, key arc.Key) task.Key {
		GinkgoHelper()
		var tsk *task.Task
		Expect(write(ctx, func(w arc.Writer) (err error) {
			tsk, err = w.UpdateTask(ctx, key, arc.TaskUpdate{Rack: new(testRack.Key)})
			return err
		})).To(Succeed())
		MustSucceed(svc.DeployProgram(ctx, key, testRack.Key))
		return tsk.Key
	}
	// run reports the task with key as running the config it holds, as the Driver does.
	run := func(ctx SpecContext, key task.Key) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return statusSvc.NewWriter(tx).Set(ctx, &task.Status{
				Owner:   new(task.ResourceID(key)),
				Name:    "kb2",
				Variant: status.VariantSuccess,
				Time:    telem.Now(),
				Details: task.StatusDetails{
					Running:    true,
					ConfigHash: retrieveTask(ctx, tx, key).ConfigHash,
				},
			})
		})).To(Succeed())
	}
	// uses returns the holders of each use of ch.
	uses := func(ctx SpecContext, ch channel.Channel) []gorp.Target {
		GinkgoHelper()
		held := MustSucceed(channelSvc.Uses(ctx, db, []channel.Key{ch.Key()}))
		targets := make([]gorp.Target, len(held[0]))
		for i, u := range held[0] {
			targets[i] = u.Holder
		}
		return targets
	}
	arcTarget := func(key arc.Key) gorp.Target {
		return gorp.NewTarget(string(resource.TypeArc), key)
	}

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		in = newChannel(ctx, "kb2_in")
		out = newChannel(ctx, "kb2_out")
		other = newChannel(ctx, "kb2_other")
	})

	It("Should name the task of a deployed graph as a use of the channels that its "+
		"nodes read and write", func(ctx SpecContext) {
		key := createArc(ctx, graphBase("kb2_graph", in.Key(), out.Key())).Base().Key
		tk := deploy(ctx, key)
		run(ctx, tk)
		drifted := gorp.NewTarget(arc.DriftTable, tk)
		Expect(uses(ctx, in)).To(ConsistOf(arcTarget(key), drifted))
		Expect(uses(ctx, out)).To(ConsistOf(arcTarget(key), drifted))
	})

	It("Should keep the task as a use of the channel that its running graph reads "+
		"after an edit moves the node to another channel", func(ctx SpecContext) {
		key := createArc(ctx, graphBase("kb2_graph_edit", in.Key(), out.Key())).
			Base().Key
		tk := deploy(ctx, key)
		run(ctx, tk)
		first := retrieveTask(ctx, db, tk).ConfigHash
		Expect(dispatch(ctx, key, "kb2", []arc.Action{
			arc.NewSetNodeInputsAction(arc.SetNodeInputsPayload{
				Key:    "source",
				Inputs: msgpack.EncodedJSON{"channel": float64(other.Key())},
			}),
		})).To(Succeed())
		// The resync of the edit gives the task a new config. The Driver still runs
		// the program that reads kb2_in.
		Eventually(func() string { return retrieveTask(ctx, db, tk).ConfigHash }).
			ShouldNot(Equal(first))
		Expect(uses(ctx, in)).To(ContainElement(gorp.NewTarget(arc.DriftTable, tk)))
	})

	It("Should hold the channels of the Arc's text after an edit and its resync "+
		"land while a Driver takes the older program", func(ctx SpecContext) {
		key := createArc(ctx, textBase("kb2_race", "kb2_in * 2.0 -> kb2_out")).
			Base().Key
		Eventually(lists(ctx, key)).Should(Equal(want(in, out)))
		racing := MustOpen(arc.OpenService(ctx, arc.ServiceConfig{
			DB:               db,
			Table:            arcTable,
			Channel:          channelSvc,
			Task:             taskSvc,
			MatchTasks:       arcTaskSvc.MatchArcs,
			Status:           statusSvc,
			ImEx:             imex.NewService(),
			TaskSyncDebounce: 5 * telem.Millisecond,
			Compile: func(
				ctx context.Context,
				a arc.AutomationArc,
				root *symbol.Symbol,
			) (arclang.Program, error) {
				var (
					prog arclang.Program
					err  error
				)
				switch src := a.Source.Variant.(type) {
				case arc.TextSource:
					prog, err = arclang.CompileText(ctx, src.Text.Materialize(), root)
				case arc.GraphSource:
					prog, err = arclang.CompileGraph(ctx, src.Graph, root)
				}
				if ctx.Value(deployMarker{}) == nil {
					return prog, err
				}
				// A user edits the Arc after this compile read it, and the resync of
				// the edit stores its lists before the deploy records its own.
				overwrite(ctx, key, "kb2_other * 2.0 -> kb2_out")
				Eventually(lists(ctx, key)).Should(Equal(want(other, out)))
				time.Sleep(300 * time.Millisecond)
				Expect(lists(ctx, key)()).To(Equal(want(other, out)))
				return prog, err
			},
		}))
		deployCtx := context.WithValue(ctx, deployMarker{}, true)
		MustSucceed(racing.DeployProgram(deployCtx, key, testRack.Key))
		// The Arc's text reads kb2_other, and no compile of that text is pending.
		Consistently(lists(ctx, key), 500*time.Millisecond).
			Should(Equal(want(other, out)))
		Expect(uses(ctx, in)).ToNot(ContainElement(arcTarget(key)))
	})
})
