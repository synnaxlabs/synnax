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
	"fmt"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	arclang "github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/channel/resolver"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// loopMarker marks the context of the compiles that a write races on every try.
type loopMarker struct{}

var _ = Describe("Compile keys breaker 3", Ordered, ContinueOnFailure, func() {
	var in, out channel.Channel
	newChannel := func(ctx context.Context, name string) channel.Channel {
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
	createArc := func(ctx SpecContext, a arc.Arc) arc.Key {
		GinkgoHelper()
		Expect(write(ctx, func(w arc.Writer) error { return w.Create(ctx, &a) })).
			To(Succeed())
		key := a.Base().Key
		DeferCleanup(func(ctx SpecContext) {
			Expect(write(ctx, func(w arc.Writer) error {
				return w.Delete(ctx, key)
			})).To(Succeed())
		})
		return key
	}
	textBase := func(name, raw string) arc.Base {
		return arc.Base{
			Name:   name,
			Source: arc.Source{Variant: arc.TextSource{Text: text.Text{Raw: raw}}},
		}
	}
	bind := func(ctx SpecContext, key arc.Key, r rack.Key) task.Key {
		GinkgoHelper()
		var tsk *task.Task
		Expect(write(ctx, func(w arc.Writer) (err error) {
			tsk, err = w.UpdateTask(ctx, key, arc.TaskUpdate{Rack: new(r)})
			return err
		})).To(Succeed())
		return tsk.Key
	}
	deployed := func(ctx context.Context, key arc.Key) (rec [2][]channel.Key, ok bool) {
		GinkgoHelper()
		r, ok := MustSucceed(svc.RetrieveDrift(ctx, db, key))[key]
		return [2][]channel.Key{r.Reads, r.Writes}, ok
	}
	lists := func(ctx context.Context, key arc.Key) [2][]channel.Key {
		GinkgoHelper()
		rec, ok := deployed(ctx, key)
		Expect(ok).To(BeTrue())
		return rec
	}
	uses := func(ctx context.Context, ch channel.Channel) []gorp.Target {
		GinkgoHelper()
		held := MustSucceed(channelSvc.Uses(ctx, db, []channel.Key{ch.Key()}))
		targets := make([]gorp.Target, len(held[0]))
		for i, u := range held[0] {
			targets[i] = u.Holder
		}
		return targets
	}
	newRack := func(ctx SpecContext, name string) rack.Rack {
		GinkgoHelper()
		r := rack.Rack{Name: name}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return rackSvc.NewWriter(tx).Create(ctx, &r)
		})).To(Succeed())
		return r
	}

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		in = newChannel(ctx, "kb3_in")
		out = newChannel(ctx, "kb3_out")
	})

	It("Should refuse a module in CompileRackProgram", func(ctx SpecContext) {
		key := createArc(ctx, newModule(textBase("kb3_module", "func f() f64 {\n"+
			"    return 1.0\n}")))
		_, err := svc.CompileRackProgram(ctx, key, testRack.Key)
		Expect(err).To(MatchError(validate.ErrValidation))
		Expect(err).To(MatchError(ContainSubstring(
			"modules cannot be deployed independently",
		)))
	})

	It("Should compile and record nothing for a rack when the automation has no "+
		"task", func(ctx SpecContext) {
		key := createArc(ctx, newAutomation(textBase(
			"kb3_no_task", "kb3_in * 2.0 -> kb3_out",
		)))
		taken := MustSucceed(svc.CompileRackProgram(ctx, key, testRack.Key))
		Expect(taken.Program).ToNot(BeNil())
		_, ok := deployed(ctx, key)
		Expect(ok).To(BeFalse())
	})

	It("Should record only the take of the rack that holds the task after the "+
		"task moves", func(ctx SpecContext) {
		key := createArc(ctx, newAutomation(textBase(
			"kb3_move", "kb3_move_in * 2.0 -> kb3_out",
		)))
		first := newChannel(ctx, "kb3_move_in")
		bind(ctx, key, testRack.Key)
		MustSucceed(svc.CompileRackProgram(ctx, key, testRack.Key))
		Expect(lists(ctx, key)).To(Equal([2][]channel.Key{
			{first.Key()}, {out.Key()},
		}))
		other := newRack(ctx, "kb3 other rack")
		bind(ctx, key, other.Key)
		// The channel is made again under its path, so a compile binds a new key.
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).
				Rename(ctx, first.Key(), "kb3_move_was", false)
		})).To(Succeed())
		again := newChannel(ctx, "kb3_move_in")
		MustSucceed(svc.CompileRackProgram(ctx, key, testRack.Key))
		Expect(lists(ctx, key)).To(Equal([2][]channel.Key{
			{first.Key()}, {out.Key()},
		}))
		MustSucceed(svc.CompileRackProgram(ctx, key, other.Key))
		Expect(lists(ctx, key)).To(Equal([2][]channel.Key{
			{again.Key()}, {out.Key()},
		}))
	})

	It("Should create the drift record of a task that has none at the embedded "+
		"configure and name it as a use", func(ctx SpecContext) {
		key := createArc(ctx, newAutomation(textBase(
			"kb3_no_record", "kb3_in * 2.0 -> kb3_out",
		)))
		cfg := MustSucceed(msgpack.NewEncodedJSON(map[string]any{
			"arc_key": key.String(),
		}))
		tsk := task.Task{
			Rack: testRack.Key, Name: "kb3_no_record", Type: arc.TaskType,
			Config: cfg,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return taskSvc.NewWriter(tx).Create(ctx, &tsk)
		})).To(Succeed())
		_, ok := deployed(ctx, key)
		Expect(ok).To(BeFalse())
		MustSucceed(svc.CompileProgram(ctx, key))
		Expect(lists(ctx, key)).To(Equal([2][]channel.Key{{in.Key()}, {out.Key()}}))
		Expect(
			uses(ctx, in),
		).To(ContainElement(gorp.NewTarget(arc.DriftTable, tsk.Key)))
		Expect(uses(ctx, out)).
			To(ContainElement(gorp.NewTarget(arc.DriftTable, tsk.Key)))
	})

	It("Should end with ErrLookupsChanged and record nothing when a write changes "+
		"the compile's reads on every try", func(ctx SpecContext) {
		key := createArc(ctx, newAutomation(textBase(
			"kb3_loop", "kb3_loop_in * 2.0 -> kb3_out",
		)))
		first := newChannel(ctx, "kb3_loop_in")
		bind(ctx, key, testRack.Key)
		MustSucceed(svc.CompileProgram(ctx, key))
		before, _ := deployed(ctx, key)
		Expect(before).To(Equal([2][]channel.Key{{first.Key()}, {out.Key()}}))
		var compiles atomic.Int32
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
				cctx context.Context,
				a arc.AutomationArc,
				root *symbol.Symbol,
			) (arclang.Program, error) {
				var (
					prog arclang.Program
					err  error
				)
				switch src := a.Source.Variant.(type) {
				case arc.TextSource:
					prog, err = arclang.CompileText(cctx, src.Text.Materialize(), root)
				case arc.GraphSource:
					prog, err = arclang.CompileGraph(cctx, src.Graph, root)
				}
				if cctx.Value(loopMarker{}) == nil || a.Key != key {
					return prog, err
				}
				n := compiles.Add(1)
				// Each compile ends with the read channel made again under its path.
				Expect(db.WithTx(cctx, func(tx gorp.Tx) error {
					var held []channel.Channel
					if err := channelSvc.NewRetrieve().
						Where(channel.MatchNames("kb3_loop_in")).
						Entries(&held).
						Exec(cctx, tx); err != nil {
						return err
					}
					w := channelSvc.NewWriter(tx)
					for _, ch := range held {
						if err := w.Rename(
							cctx, ch.Key(), fmt.Sprintf("kb3_loop_was_%d", n), false,
						); err != nil {
							return err
						}
					}
					ch := channel.Channel{
						Name:     "kb3_loop_in",
						Virtual:  true,
						DataType: telem.Float64T,
					}
					return w.Create(cctx, &ch)
				})).To(Succeed())
				return prog, err
			},
		}))
		start := time.Now()
		_, err := racing.CompileRackProgram(
			context.WithValue(ctx, loopMarker{}, true),
			key,
			testRack.Key,
		)
		elapsed := time.Since(start)
		Expect(err).To(MatchError(resolver.ErrLookupsChanged))
		Expect(compiles.Load()).To(BeEquivalentTo(15))
		Expect(elapsed).To(BeNumerically("<", 5*time.Second))
		after, _ := deployed(ctx, key)
		Expect(after).To(Equal(before))
	})
})
