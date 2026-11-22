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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Compile keys check 2", Ordered, func() {
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
	createArc := func(ctx SpecContext, name, raw string) arc.Key {
		GinkgoHelper()
		a := newAutomation(arc.Base{
			Name:   name,
			Source: arc.Source{Variant: arc.TextSource{Text: text.Text{Raw: raw}}},
		})
		Expect(write(ctx, func(w arc.Writer) error { return w.Create(ctx, &a) })).
			To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(write(ctx, func(w arc.Writer) error {
				return w.Delete(ctx, a.Base().Key)
			})).To(Succeed())
		})
		return a.Base().Key
	}
	lists := func(ctx context.Context, key arc.Key) func() [2][]channel.Key {
		return func() [2][]channel.Key {
			var a arc.Arc
			Expect(svc.NewRetrieve().
				Where(arc.MatchKeys(key)).
				Entry(&a).
				Exec(ctx, db)).To(Succeed())
			return [2][]channel.Key{a.Base().Reads, a.Base().Writes}
		}
	}
	updateTask := func(ctx SpecContext, key arc.Key) {
		GinkgoHelper()
		Expect(write(ctx, func(w arc.Writer) error {
			_, err := w.UpdateTask(ctx, key, arc.TaskUpdate{Rack: new(testRack.Key)})
			return err
		})).To(Succeed())
	}

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		in = newChannel(ctx, "kc2_in")
		out = newChannel(ctx, "kc2_out")
	})

	It("Should write no drift record when a Driver takes a program with the "+
		"deployed lists", func(ctx SpecContext) {
		key := createArc(ctx, "kc2_equal", "kc2_in * 2.0 -> kc2_out")
		want := [2][]channel.Key{{in.Key()}, {out.Key()}}
		Eventually(lists(ctx, key)).Should(Equal(want))
		updateTask(ctx, key)
		MustSucceed(svc.CompileProgram(ctx, key))
		var writes atomic.Int32
		stop := db.Observe().OnChange(func(_ context.Context, c gorp.Commit) {
			for _, ch := range c.Changes {
				if ch.Record.Table == arc.DriftTable {
					writes.Add(1)
				}
			}
		})
		DeferCleanup(stop)
		time.Sleep(200 * time.Millisecond)
		writes.Store(0)
		MustSucceed(svc.CompileProgram(ctx, key))
		Consistently(writes.Load, 100*time.Millisecond).Should(BeZero())
	})

	It("Should store the Arc's lists in the transaction that binds its task", func(
		ctx SpecContext,
	) {
		key := createArc(ctx, "kc2_bind", "kc2_late * 2.0 -> kc2_out")
		Consistently(lists(ctx, key), 100*time.Millisecond).
			Should(Equal([2][]channel.Key{nil, nil}))
		late := newChannel(ctx, "kc2_late")
		// A channel create schedules no resync, so the lists stay empty until a
		// compile.
		Consistently(lists(ctx, key), 100*time.Millisecond).
			Should(Equal([2][]channel.Key{nil, nil}))
		updateTask(ctx, key)
		Expect(lists(ctx, key)()).
			To(Equal([2][]channel.Key{{late.Key()}, {out.Key()}}))
	})
})
