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
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/crdt"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Compile keys", Ordered, func() {
	var in, out, other channel.Channel
	write := func(ctx context.Context, fn func(arc.Writer) error) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error { return fn(svc.NewWriter(tx)) })
	}
	createArc := func(ctx SpecContext, t arc.Type, b arc.Base) arc.Arc {
		GinkgoHelper()
		a := newArc(t, b)
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
	retrieve := func(ctx SpecContext, key arc.Key) arc.Base {
		GinkgoHelper()
		var a arc.Arc
		Expect(svc.NewRetrieve().
			Where(arc.MatchKeys(key)).
			Entry(&a).
			Exec(ctx, db)).To(Succeed())
		return a.Base()
	}
	// lists returns the stored reads and writes of the Arc with key.
	lists := func(ctx SpecContext, key arc.Key) func() [2][]channel.Key {
		return func() [2][]channel.Key {
			b := retrieve(ctx, key)
			return [2][]channel.Key{b.Reads, b.Writes}
		}
	}
	keys := func(chs ...channel.Channel) []channel.Key {
		out := make([]channel.Key, len(chs))
		for i, ch := range chs {
			out[i] = ch.Key()
		}
		return out
	}
	var editor uint32 = 500
	// overwrite replaces the text of the Arc with key by raw.
	overwrite := func(ctx SpecContext, key arc.Key, raw string) {
		GinkgoHelper()
		doc := textOf(retrieve(ctx, key)).Doc
		editor++
		client := crdt.New(editor)
		client.Load(doc.Inserts, doc.Deletes)
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
		Expect(dispatch(ctx, key, "ck", acts)).To(Succeed())
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
		var tsk task.Task
		Expect(taskSvc.NewRetrieve().
			Where(task.MatchKeys(key)).
			Entry(&tsk).
			Exec(ctx, db)).To(Succeed())
		stat := &task.Status{
			Owner:   new(task.ResourceID(key)),
			Name:    "ck",
			Variant: status.VariantSuccess,
			Time:    telem.Now(),
			Details: task.StatusDetails{Running: true, ConfigHash: tsk.ConfigHash},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return statusSvc.NewWriter(tx).Set(ctx, stat)
		})).To(Succeed())
	}
	configHash := func(ctx SpecContext, key task.Key) func() string {
		return func() string {
			var tsk task.Task
			Expect(taskSvc.NewRetrieve().
				Where(task.MatchKeys(key)).
				Entry(&tsk).
				Exec(ctx, db)).To(Succeed())
			return tsk.ConfigHash
		}
	}
	// deployed returns the deployed lists in the drift record of the task of the Arc
	// with key.
	deployed := func(ctx SpecContext, key arc.Key) func() [2][]channel.Key {
		return func() [2][]channel.Key {
			rec := MustSucceed(svc.RetrieveDrift(ctx, db, key))[key]
			return [2][]channel.Key{rec.Reads, rec.Writes}
		}
	}

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		for _, ch := range []*channel.Channel{&in, &out, &other} {
			ch.Virtual, ch.DataType = true, telem.Float64T
		}
		in.Name, out.Name, other.Name = "ck_in", "ck_out", "ck_other"
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			for _, ch := range []*channel.Channel{&in, &out, &other} {
				if err := channelSvc.NewWriter(tx).Create(ctx, ch); err != nil {
					return err
				}
			}
			return nil
		})).To(Succeed())
	})

	It(
		"Should store the channels that an automation with no task reads and writes",
		func(
			ctx SpecContext,
		) {
			a := createArc(
				ctx,
				arc.TypeAutomation,
				textBase("ck_a", "ck_in * 2.0 -> ck_out"),
			)
			Eventually(lists(ctx, a.Base().Key)).
				Should(Equal([2][]channel.Key{keys(in), keys(out)}))
		},
	)

	It("Should keep the lists when an edit does not compile", func(ctx SpecContext) {
		a := createArc(
			ctx,
			arc.TypeAutomation,
			textBase("ck_a", "ck_in * 2.0 -> ck_out"),
		)
		want := [2][]channel.Key{keys(in), keys(out)}
		Eventually(lists(ctx, a.Base().Key)).Should(Equal(want))
		key := deploy(ctx, a.Base().Key)
		compiled := configHash(ctx, key)()
		overwrite(ctx, a.Base().Key, "ck_in * -> ck_out")
		// The resync of the edit hashes the text that does not compile.
		Eventually(configHash(ctx, key)).ShouldNot(Equal(compiled))
		_, err := svc.CompileProgram(ctx, a.Base().Key)
		_, failed := errors.AsType[*diagnostics.Diagnostics](err)
		Expect(failed).To(BeTrue(), "compile error: %v", err)
		Expect(lists(ctx, a.Base().Key)()).To(Equal(want))
	})

	It("Should write no Arc at a resync whose lists are equal", func(ctx SpecContext) {
		a := createArc(
			ctx,
			arc.TypeAutomation,
			textBase("ck_a", "ck_in * 2.0 -> ck_out"),
		)
		Eventually(lists(ctx, a.Base().Key)).
			Should(Equal([2][]channel.Key{keys(in), keys(out)}))
		key := deploy(ctx, a.Base().Key)
		compiled := configHash(ctx, key)()
		var writes atomic.Int32
		stop := db.Observe().OnChange(func(_ context.Context, c gorp.Commit) {
			for _, ch := range c.Changes {
				if ch.Record.Table == string(resource.TypeArc) &&
					ch.Record.Key == a.Base().Key.String() {
					writes.Add(1)
				}
			}
		})
		DeferCleanup(stop)
		overwrite(ctx, a.Base().Key, "ck_in * 3.0 -> ck_out")
		// A new program with the same channels: the resync writes the task alone, so
		// the edit is the one write of the Arc.
		Eventually(configHash(ctx, key)).ShouldNot(Equal(compiled))
		Expect(writes.Load()).To(BeEquivalentTo(1))
	})

	It("Should store the channels that the functions of a graph read and write", func(
		ctx SpecContext,
	) {
		g := graph.Graph{
			Functions: ir.Functions{{
				Key:     "double",
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.F64()}},
				Body:    ir.Body{Raw: "{\n\treturn ck_in * 2.0\n}"},
			}},
			Nodes: graph.Nodes{{Key: "calc"}, {Key: "sink"}},
			Edges: graph.Edges{{
				Key:    "edge",
				Source: ir.Handle{Node: "calc", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "sink", Param: ir.DefaultInputParam},
			}},
			Inputs: map[string]msgpack.EncodedJSON{
				"calc": {"type": "double"},
				"sink": {"type": "write", "channel": float64(out.Key())},
			},
		}
		a := createArc(ctx, arc.TypeAutomation, arc.Base{
			Name:   "ck_fn",
			Source: arc.Source{Variant: arc.GraphSource{Graph: g}},
		})
		// The node config of sink references ck_out, so only the function's channels
		// are in the lists.
		Eventually(lists(ctx, a.Base().Key)).
			Should(Equal([2][]channel.Key{keys(in), nil}))
		Expect(uses(ctx, out)).To(ContainElement(arcTarget(a.Base().Key)))
	})

	It(
		"Should store the lists of the next compile after an edit",
		func(ctx SpecContext) {
			a := createArc(
				ctx,
				arc.TypeAutomation,
				textBase("ck_a", "ck_in * 2.0 -> ck_out"),
			)
			Eventually(lists(ctx, a.Base().Key)).
				Should(Equal([2][]channel.Key{keys(in), keys(out)}))
			overwrite(ctx, a.Base().Key, "ck_other * 2.0 -> ck_out")
			Eventually(lists(ctx, a.Base().Key)).
				Should(Equal([2][]channel.Key{keys(other), keys(out)}))
		},
	)

	It(
		"Should store the channels that a module's functions read",
		func(ctx SpecContext) {
			m := createArc(ctx, arc.TypeModule, textBase(
				"ck_m",
				"export func f() {t u8} f64 { return ck_in }",
			))
			Eventually(lists(ctx, m.Base().Key)).
				Should(Equal([2][]channel.Key{keys(in), nil}))
		},
	)

	It("Should refuse a create that sends a list", func(ctx SpecContext) {
		b := textBase("ck_a", "ck_in * 2.0 -> ck_out")
		b.Reads = keys(other)
		a := newArc(arc.TypeAutomation, b)
		Expect(write(ctx, func(w arc.Writer) error { return w.Create(ctx, &a) })).
			To(And(
				MatchError(ContainSubstring(
					"reads and writes: set by the Core at each compile",
				)),
				MatchError(validate.ErrValidation),
			))
	})

	It("Should leave the lists out of an export", func(ctx SpecContext) {
		a := createArc(
			ctx,
			arc.TypeAutomation,
			textBase("ck_a", "ck_in * 2.0 -> ck_out"),
		)
		Eventually(lists(ctx, a.Base().Key)).
			Should(Equal([2][]channel.Key{keys(in), keys(out)}))
		env := MustSucceed(svc.Export(ctx, db, a.ResourceID()))
		wire := WireRoundTrip(env)
		body := MustSucceed(wire.Decode[map[string]any]())
		Expect(body).ToNot(HaveKey("reads"))
		Expect(body).ToNot(HaveKey("writes"))
	})

	It("Should refuse an action that sets a list", func(ctx SpecContext) {
		a := createArc(
			ctx,
			arc.TypeAutomation,
			textBase("ck_a", "ck_in * 2.0 -> ck_out"),
		)
		Expect(dispatch(ctx, a.Base().Key, "ck", []arc.Action{
			arc.NewSetAction(arc.SetPayload{
				Path:  []string{"reads"},
				Value: keys(other),
			}),
		})).To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`unknown object member name "reads"`)),
		))
	})

	It("Should name the channel of a graph node as used even when the graph does not "+
		"compile", func(ctx SpecContext) {
		g := graph.Graph{
			Nodes: graph.Nodes{{Key: "source"}, {Key: "sink"}},
			Inputs: map[string]msgpack.EncodedJSON{
				"source": {"type": "on", "channel": float64(in.Key())},
				"sink":   {"type": "write", "channel": float64(other.Key())},
			},
		}
		a := createArc(ctx, arc.TypeAutomation, arc.Base{
			Name:   "ck_g",
			Source: arc.Source{Variant: arc.GraphSource{Graph: g}},
		})
		Expect(uses(ctx, in)).To(ContainElement(arcTarget(a.Base().Key)))
		Expect(uses(ctx, other)).To(ContainElement(arcTarget(a.Base().Key)))
		held := MustSucceed(channelSvc.Uses(ctx, db, keys(in, other)))
		Expect(
			held[0],
		).To(ContainElement(HaveField("Direction", channel.DirectionRead)))
		Expect(
			held[1],
		).To(ContainElement(HaveField("Direction", channel.DirectionWrite)))
	})

	It("Should name an Arc and its task as uses of a channel and keep both after the "+
		"channel is deleted", func(ctx SpecContext) {
		ch := channel.Channel{Name: "ck_gone", Virtual: true, DataType: telem.Float64T}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		a := createArc(
			ctx,
			arc.TypeAutomation,
			textBase("ck_a", "ck_gone * 2.0 -> ck_out"),
		)
		Eventually(lists(ctx, a.Base().Key)).
			Should(Equal([2][]channel.Key{keys(ch), keys(out)}))
		key := deploy(ctx, a.Base().Key)
		run(ctx, key)
		drifted := gorp.NewTarget(arc.DriftTable, key)
		Eventually(func() []gorp.Target { return uses(ctx, ch) }).
			Should(ConsistOf(arcTarget(a.Base().Key), drifted))
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Delete(ctx, ch.Key(), false)
		})).To(Succeed())
		Expect(uses(ctx, ch)).To(ConsistOf(arcTarget(a.Base().Key), drifted))
	})

	It("Should hold the channel that the Driver's program reads after the channel is "+
		"made again under its name and the Core restarts", func(ctx SpecContext) {
		gone := channel.Channel{Name: "ck_again", DataType: telem.Float64T}
		gone.Virtual = true
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &gone)
		})).To(Succeed())
		a := createArc(
			ctx,
			arc.TypeAutomation,
			textBase("ck_a", "ck_again * 2.0 -> ck_out"),
		)
		key := a.Base().Key
		deploy(ctx, key)
		Expect(deployed(ctx, key)()).To(Equal([2][]channel.Key{keys(gone), keys(out)}))
		again := channel.Channel{Name: "ck_again", DataType: telem.Float64T}
		again.Virtual = true
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := channelSvc.NewWriter(tx)
			if err := w.Delete(ctx, gone.Key(), false); err != nil {
				return err
			}
			return w.Create(ctx, &again)
		})).To(Succeed())
		// A Core start configures the task again, which takes the program again.
		MustSucceed(svc.DeployProgram(ctx, key, testRack.Key))
		Expect(deployed(ctx, key)()).To(Equal([2][]channel.Key{keys(again), keys(out)}))
	})

	It("Should keep the deployed lists while edits change the Arc's lists, and take "+
		"the new lists at the next deploy", func(ctx SpecContext) {
		a := createArc(
			ctx,
			arc.TypeAutomation,
			textBase("ck_a", "ck_in * 2.0 -> ck_out"),
		)
		first := [2][]channel.Key{keys(in), keys(out)}
		Eventually(lists(ctx, a.Base().Key)).Should(Equal(first))
		key := deploy(ctx, a.Base().Key)
		run(ctx, key)
		Expect(deployed(ctx, a.Base().Key)()).To(Equal(first))
		overwrite(ctx, a.Base().Key, "ck_other * 2.0 -> ck_out")
		second := [2][]channel.Key{keys(other), keys(out)}
		Eventually(lists(ctx, a.Base().Key)).Should(Equal(second))
		Expect(deployed(ctx, a.Base().Key)()).To(Equal(first))
		drifted := gorp.NewTarget(arc.DriftTable, key)
		Expect(uses(ctx, in)).To(ContainElement(drifted))
		Expect(uses(ctx, other)).ToNot(ContainElement(drifted))
		Expect(deploy(ctx, a.Base().Key)).To(Equal(key))
		Expect(deployed(ctx, a.Base().Key)()).To(Equal(second))
		Expect(uses(ctx, in)).ToNot(ContainElement(drifted))
		Expect(uses(ctx, other)).To(ContainElement(drifted))
	})
})
