// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package graph_test

import (
	"context"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	graph "github.com/synnaxlabs/synnax/pkg/service/calculation/graph"
	calctestutil "github.com/synnaxlabs/synnax/pkg/service/calculation/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// holdGraph returns a function that holds g on the batch that writes the channel named
// gate, until the returned release runs, and a channel that closes once g holds. A held
// graph adds every later change to its next batch.
func holdGraph(g *graph.Graph, gate string) (held <-chan struct{}, release func()) {
	h, r := make(chan struct{}), make(chan struct{})
	var once sync.Once
	g.Observe().OnChange(func(_ context.Context, b graph.Changes) {
		if !batchWrites(b, gate) {
			return
		}
		close(h)
		<-r
	})
	return h, func() { once.Do(func() { close(r) }) }
}

// waitForBatch returns a channel that closes once g notifies a batch that writes the
// channel named name.
func waitForBatch(g *graph.Graph, name string) <-chan struct{} {
	done := make(chan struct{})
	var once sync.Once
	g.Observe().OnChange(func(_ context.Context, b graph.Changes) {
		if batchWrites(b, name) {
			once.Do(func() { close(done) })
		}
	})
	return done
}

var _ = Describe("Runtime status breaker a4", func() {
	report := func(name string) *calculation.Status {
		return &calculation.Status{
			Name:    name,
			Variant: status.VariantError,
			Message: "Failed to run calculation",
			Time:    telem.Now(),
		}
	}

	It("Should clear a status that the old expression's runtime reported after an edit",
		func(ctx SpecContext) {
			g := openGraph(ctx)
			createDep(ctx, "rt_a4_race_dep")
			processed := waitForBatch(g, "rt_a4_race")
			held, release := holdGraph(g, "rt_a4_race_gate")
			DeferCleanup(release)
			createDep(ctx, "rt_a4_race_gate")
			Eventually(held, 2*time.Second).Should(BeClosed())

			calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name: "rt_a4_race", DataType: telem.Int64T,
				Expression: "return rt_a4_race_dep + 1",
			}))
			// The runtime of the first expression still runs: the calculation service
			// swaps it only once the graph notifies the edit below.
			setExpression(ctx, calc.Key(), "return rt_a4_race_dep + 2")
			Expect(g.SetRuntimeStatus(ctx, calc.Key(), report("rt_a4_race"))).
				To(Succeed())
			release()

			Eventually(processed, 2*time.Second).Should(BeClosed())
			_, ok := fetchStatus(ctx, calc.Key())
			Expect(ok).To(BeFalse(),
				"the status of return rt_a4_race_dep + 1 outlived the edit")
		},
	)

	It(
		"Should keep a status reported for an expression that an edit and its revert keep",
		func(ctx SpecContext) {
			g := openGraph(ctx)
			createDep(ctx, "rt_a4_aba_dep")
			processed := waitForBatch(g, "rt_a4_aba")
			held, release := holdGraph(g, "rt_a4_aba_gate")
			DeferCleanup(release)
			createDep(ctx, "rt_a4_aba_gate")
			Eventually(held, 2*time.Second).Should(BeClosed())

			calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name: "rt_a4_aba", DataType: telem.Int64T,
				Expression: "return rt_a4_aba_dep + 1",
			}))
			Expect(g.SetRuntimeStatus(ctx, calc.Key(), report("rt_a4_aba"))).
				To(Succeed())
			setExpression(ctx, calc.Key(), "return rt_a4_aba_dep + 2")
			setExpression(ctx, calc.Key(), "return rt_a4_aba_dep + 1")
			release()

			Eventually(processed, 2*time.Second).Should(BeClosed())
			_, ok := fetchStatus(ctx, calc.Key())
			Expect(ok).To(BeTrue())
		},
	)

	It("Should clear a status after two edits in one batch", func(ctx SpecContext) {
		g := openGraph(ctx)
		createDep(ctx, "rt_a4_two_dep")
		processed := waitForBatch(g, "rt_a4_two")
		held, release := holdGraph(g, "rt_a4_two_gate")
		DeferCleanup(release)
		createDep(ctx, "rt_a4_two_gate")
		Eventually(held, 2*time.Second).Should(BeClosed())

		calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
			Name: "rt_a4_two", DataType: telem.Int64T,
			Expression: "return rt_a4_two_dep + 1",
		}))
		Expect(g.SetRuntimeStatus(ctx, calc.Key(), report("rt_a4_two"))).To(Succeed())
		setExpression(ctx, calc.Key(), "return rt_a4_two_dep + 2")
		setExpression(ctx, calc.Key(), "return rt_a4_two_dep + 3")
		release()

		Eventually(processed, 2*time.Second).Should(BeClosed())
		_, ok := fetchStatus(ctx, calc.Key())
		Expect(ok).To(BeFalse())
	})

	It("Should clear a status of a tracked node after an edit and its revert",
		func(ctx SpecContext) {
			g := openGraph(ctx)
			createDep(ctx, "rt_a4_taba_dep")
			tracked := waitForBatch(g, "rt_a4_taba")
			calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name: "rt_a4_taba", DataType: telem.Int64T,
				Expression: "return rt_a4_taba_dep + 1",
			}))
			Eventually(tracked, 2*time.Second).Should(BeClosed())
			Expect(g.SetRuntimeStatus(ctx, calc.Key(), report("rt_a4_taba"))).
				To(Succeed())
			setExpression(ctx, calc.Key(), "return rt_a4_taba_dep + 2")
			eventuallyExpectNoStatus(ctx, calc.Key())
			Expect(g.SetRuntimeStatus(ctx, calc.Key(), report("rt_a4_taba"))).
				To(Succeed())
			setExpression(ctx, calc.Key(), "return rt_a4_taba_dep + 1")
			eventuallyExpectNoStatus(ctx, calc.Key())
		},
	)

	It("Should replace a runtime status with the compile error of an expression that "+
		"never compiled", func(ctx SpecContext) {
		g := openGraph(ctx)
		createDep(ctx, "rt_a4_never_dep")
		processed := waitForBatch(g, "rt_a4_never")
		held, release := holdGraph(g, "rt_a4_never_gate")
		DeferCleanup(release)
		createDep(ctx, "rt_a4_never_gate")
		Eventually(held, 2*time.Second).Should(BeClosed())

		calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
			Name: "rt_a4_never", DataType: telem.Int64T,
			Expression: "return rt_a4_never_dep + 1",
		}))
		deleteDep(ctx, "rt_a4_never_dep")
		Expect(g.SetRuntimeStatus(ctx, calc.Key(), report("rt_a4_never"))).
			To(Succeed())
		release()

		Eventually(processed, 2*time.Second).Should(BeClosed())
		Expect(expectStatus(ctx, calc.Key()).Message).
			To(Equal("Failed to compile calculation"))
	})
})
