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
	"encoding/json/v2"
	"slices"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/drift"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Compile keys breaker", Ordered, ContinueOnFailure, func() {
	var in, out channel.Channel
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
	deleteLater := func(key arc.Key) {
		DeferCleanup(func(ctx SpecContext) {
			Expect(write(ctx, func(w arc.Writer) error {
				return w.Delete(ctx, key)
			})).To(Succeed())
		})
	}
	createArc := func(ctx SpecContext, b arc.Base) arc.Arc {
		GinkgoHelper()
		a := newAutomation(b)
		Expect(write(ctx, func(w arc.Writer) error { return w.Create(ctx, &a) })).
			To(Succeed())
		deleteLater(a.Base().Key)
		return a
	}
	textBase := func(name, raw string) arc.Base {
		return arc.Base{
			Name:   name,
			Source: arc.Source{Variant: arc.TextSource{Text: text.Text{Raw: raw}}},
		}
	}
	retrieveArc := func(ctx SpecContext, key arc.Key) arc.Arc {
		GinkgoHelper()
		var a arc.Arc
		Expect(svc.NewRetrieve().
			Where(arc.MatchKeys(key)).
			Entry(&a).
			Exec(ctx, db)).To(Succeed())
		return a
	}
	lists := func(ctx SpecContext, key arc.Key) func() [2][]channel.Key {
		return func() [2][]channel.Key {
			b := retrieveArc(ctx, key).Base()
			return [2][]channel.Key{b.Reads, b.Writes}
		}
	}
	deployed := func(ctx SpecContext, key arc.Key) func() [2][]channel.Key {
		return func() [2][]channel.Key {
			rec := MustSucceed(svc.RetrieveDrift(ctx, db, key))[key]
			return [2][]channel.Key{rec.Reads, rec.Writes}
		}
	}
	want := func(r, w channel.Channel) [2][]channel.Key {
		return [2][]channel.Key{{r.Key()}, {w.Key()}}
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
		MustSucceed(svc.CompileProgram(ctx, key))
		return tsk.Key
	}
	// running returns the status of a Driver that runs the config the task with key
	// holds in snap.
	running := func(ctx SpecContext, snap gorp.Snapshot, key task.Key) *task.Status {
		GinkgoHelper()
		return &task.Status{
			Owner:   new(task.ResourceID(key)),
			Name:    "kb",
			Variant: status.VariantSuccess,
			Time:    telem.Now(),
			Details: task.StatusDetails{
				Running:    true,
				ConfigHash: retrieveTask(ctx, snap, key).ConfigHash,
			},
		}
	}
	run := func(ctx SpecContext, key task.Key) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return statusSvc.NewWriter(tx).Set(ctx, running(ctx, tx, key))
		})).To(Succeed())
	}
	// upgrade leaves the Arc with key as the v4 to v5 migration leaves a stored Arc:
	// with no lists, written with no resync.
	upgrade := func(ctx SpecContext, key arc.Key) {
		GinkgoHelper()
		a := retrieveArc(ctx, key)
		b := a.Base()
		b.Reads, b.Writes = nil, nil
		a.SetBase(b)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return gorp.WrapWriter[arc.Key, arc.Arc](tx).Set(ctx, a)
		})).To(Succeed())
		Expect(lists(ctx, key)()).To(Equal([2][]channel.Key{nil, nil}))
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

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		in = newChannel(ctx, "kb_in")
		out = newChannel(ctx, "kb_out")
	})

	It("Should hold the channels of the program the Driver runs after a channel "+
		"rename and a redeploy", func(ctx SpecContext) {
		old := newChannel(ctx, "kb_rn_in")
		key := createArc(ctx, textBase("kb_rn", "kb_rn_in * 2.0 -> kb_out")).Base().Key
		Eventually(lists(ctx, key)).Should(Equal(want(old, out)))
		tk := deploy(ctx, key)
		run(ctx, tk)
		Eventually(deployed(ctx, key)).Should(Equal(want(old, out)))
		first := retrieveTask(ctx, db, tk).ConfigHash
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Rename(ctx, old.Key(), "kb_rn_was", false)
		})).To(Succeed())
		fresh := newChannel(ctx, "kb_rn_in")
		Expect(deploy(ctx, key)).To(Equal(tk))
		// The redeploy compiled a program that reads the fresh channel.
		Expect(retrieveTask(ctx, db, tk).ConfigHash).ToNot(Equal(first))
		run(ctx, tk)
		Eventually(deployed(ctx, key)).Should(Equal(want(fresh, out)))
	})

	It("Should hold the channels of the program the Driver runs after a channel is "+
		"deleted, made again under its name, and redeployed", func(ctx SpecContext) {
		old := newChannel(ctx, "kb_dm_in")
		key := createArc(ctx, textBase("kb_dm", "kb_dm_in * 2.0 -> kb_out")).Base().Key
		Eventually(lists(ctx, key)).Should(Equal(want(old, out)))
		tk := deploy(ctx, key)
		run(ctx, tk)
		Eventually(deployed(ctx, key)).Should(Equal(want(old, out)))
		first := retrieveTask(ctx, db, tk).ConfigHash
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Delete(ctx, old.Key(), false)
		})).To(Succeed())
		fresh := newChannel(ctx, "kb_dm_in")
		Expect(deploy(ctx, key)).To(Equal(tk))
		Expect(retrieveTask(ctx, db, tk).ConfigHash).ToNot(Equal(first))
		run(ctx, tk)
		Eventually(deployed(ctx, key)).Should(Equal(want(fresh, out)))
	})

	It("Should fill the deployed lists of an upgraded Arc at its next deploy", func(
		ctx SpecContext,
	) {
		key := createArc(ctx, textBase("kb_up", "kb_in * 2.0 -> kb_out")).Base().Key
		Eventually(lists(ctx, key)).Should(Equal(want(in, out)))
		upgrade(ctx, key)
		tk := deploy(ctx, key)
		run(ctx, tk)
		Eventually(deployed(ctx, key)).Should(Equal(want(in, out)))
	})

	It("Should fill the lists of an upgraded Arc when the Core starts", func(
		ctx SpecContext,
	) {
		key := createArc(ctx, textBase("kb_start", "kb_in * 2.0 -> kb_out")).Base().Key
		Eventually(lists(ctx, key)).Should(Equal(want(in, out)))
		upgrade(ctx, key)
		MustOpen(arc.OpenService(ctx, arc.ServiceConfig{
			DB:               db,
			Table:            arcTable,
			Channel:          channelSvc,
			Task:             taskSvc,
			MatchTasks:       arcTaskSvc.MatchArcs,
			Status:           statusSvc,
			ImEx:             imex.NewService(),
			TaskSyncDebounce: 5 * telem.Millisecond,
		}))
		Eventually(lists(ctx, key)).Should(Equal(want(in, out)))
	})

	It("Should read the current lists as deployed in a snapshot where the Driver "+
		"runs the current config", func(ctx SpecContext) {
		key := createArc(ctx, textBase("kb_read", "kb_in * 2.0 -> kb_out")).Base().Key
		Eventually(lists(ctx, key)).Should(Equal(want(in, out)))
		tk := deploy(ctx, key)
		var rec drift.Record
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := statusSvc.NewWriter(tx).
				Set(ctx, running(ctx, tx, tk)); err != nil {
				return err
			}
			recs, err := svc.RetrieveDrift(ctx, tx, key)
			rec = recs[key]
			return err
		})).To(Succeed())
		Expect(rec.Deployed).To(Equal(rec.Stored))
		Expect([2][]channel.Key{rec.Reads, rec.Writes}).To(Equal(want(in, out)))
	})

	It("Should import a graph Arc whose node holds a channel key this Core never "+
		"issued", func(ctx SpecContext) {
		a := createArc(ctx, graphBase("kb_g", in.Key(), out.Key()))
		env := MustSucceed(svc.Export(ctx, db, a.ResourceID()))
		var body map[string]any
		Expect(json.Unmarshal(MustSucceed(json.Marshal(env)), &body)).To(Succeed())
		body["name"] = "kb_g_imported"
		field := func(m map[string]any, name string) map[string]any {
			GinkgoHelper()
			v, ok := m[name].(map[string]any)
			Expect(ok).To(BeTrue(), name)
			return v
		}
		source := field(
			field(field(field(body, "source"), "graph"), "inputs"),
			"source",
		)
		never := in.Key() + 100000
		source["channel"] = float64(never)
		var imported imex.Envelope
		Expect(json.Unmarshal(MustSucceed(json.Marshal(body)), &imported)).
			To(Succeed())
		var id resource.ID
		Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
			id, err = imexSvc.Import(
				ctx,
				tx,
				imported,
				imex.ImportOptions{Parent: testParent},
			)
			return err
		})).To(Succeed())
		deleteLater(MustSucceed(uuid.Parse(id.Key)))
	})

	It("Should keep the lists when a graph node's channel input is not a key", func(
		ctx SpecContext,
	) {
		// D-295: a graph's lists hold the channels of its functions only.
		b := graphBase("kb_gs", in.Key(), out.Key())
		src, ok := b.Source.Variant.(arc.GraphSource)
		Expect(ok).To(BeTrue())
		src.Graph.Functions = ir.Functions{{
			Key:     "kb_read",
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.F64()}},
			Body:    ir.Body{Raw: "{\n\treturn kb_in\n}"},
		}}
		b.Source.Variant = src
		key := createArc(ctx, b).Base().Key
		Eventually(lists(ctx, key)).Should(Equal([2][]channel.Key{{in.Key()}, nil}))
		Expect(dispatch(ctx, key, "kb", []arc.Action{
			arc.NewSetNodeInputsAction(arc.SetNodeInputsPayload{
				Key:    "source",
				Inputs: msgpack.EncodedJSON{"channel": "kb_in"},
			}),
		})).To(Succeed())
		Consistently(lists(ctx, key), 200*time.Millisecond).
			Should(Equal([2][]channel.Key{{in.Key()}, nil}))
	})

	It("Should hold the channels that a module's sequence writes", func(
		ctx SpecContext,
	) {
		seq := newChannel(ctx, "kb_mseq")
		m := newModule(textBase("kb_mod_seq", `
export func f() {t u8} f64 { return kb_in }
export sequence s { stage a { 1.0 -> kb_mseq } }
`))
		Expect(write(ctx, func(w arc.Writer) error { return w.Create(ctx, &m) })).
			To(Succeed())
		deleteLater(m.Base().Key)
		Eventually(lists(ctx, m.Base().Key)).Should(Equal(want(in, seq)))
	})

	It("Should hold the channels of an automation's sequence and of the module "+
		"functions it calls", func(ctx SpecContext) {
		trig := channel.Channel{Name: "kb_trig", Virtual: true, DataType: telem.Uint8T}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &trig)
		})).To(Succeed())
		fn := newChannel(ctx, "kb_fn")
		seqOut := newChannel(ctx, "kb_seq_out")
		m := newModule(
			textBase("kb_mod_f", "export func f() {t u8} f64 { return kb_in }"),
		)
		Expect(write(ctx, func(w arc.Writer) error { return w.Create(ctx, &m) })).
			To(Succeed())
		deleteLater(m.Base().Key)
		key := createArc(ctx, textBase("kb_calls", `
import kb_mod_f

kb_trig -> kb_mod_f.f() -> kb_fn
kb_trig == 1 => s
sequence s { stage a { 2.0 -> kb_seq_out } }
`)).Base().Key
		sorted := func(chs ...channel.Channel) []channel.Key {
			keys := make([]channel.Key, len(chs))
			for i, ch := range chs {
				keys[i] = ch.Key()
			}
			slices.Sort(keys)
			return keys
		}
		Eventually(lists(ctx, key)).Should(Equal([2][]channel.Key{
			sorted(in, trig),
			sorted(fn, seqOut),
		}))
	})

	It("Should drop the task from the uses of a channel when the task is deleted",
		func(ctx SpecContext) {
			key := createArc(
				ctx,
				textBase("kb_del", "kb_in * 2.0 -> kb_out"),
			).Base().
				Key
			Eventually(lists(ctx, key)).Should(Equal(want(in, out)))
			tk := deploy(ctx, key)
			run(ctx, tk)
			Eventually(deployed(ctx, key)).Should(Equal(want(in, out)))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return taskSvc.NewWriter(tx).Delete(ctx, tk, false)
			})).To(Succeed())
			held := MustSucceed(channelSvc.Uses(ctx, db, []channel.Key{in.Key()}))
			Expect(held[0]).ToNot(ContainElement(
				HaveField("Holder", gorp.NewTarget(arc.DriftTable, tk)),
			))
		})

	// D-295 build note 5: the Core refuses the lists at a create, so the client's
	// create drops them, as this spec does.
	It("Should create an Arc again from its retrieved record after a delete, as the "+
		"Console's restore does", func(ctx SpecContext) {
		a := newAutomation(textBase("kb_restore", "kb_in * 2.0 -> kb_out"))
		Expect(write(ctx, func(w arc.Writer) error { return w.Create(ctx, &a) })).
			To(Succeed())
		key := a.Base().Key
		Eventually(lists(ctx, key)).Should(Equal(want(in, out)))
		corpse := retrieveArc(ctx, key)
		b := corpse.Base()
		b.Reads, b.Writes = nil, nil
		corpse.SetBase(b)
		Expect(write(ctx, func(w arc.Writer) error { return w.Delete(ctx, key) })).
			To(Succeed())
		Expect(write(ctx, func(w arc.Writer) error { return w.Create(ctx, &corpse) })).
			To(Succeed())
		deleteLater(key)
	})
})
