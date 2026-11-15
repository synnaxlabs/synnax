// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package runtime_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

const budgetSource = `
func spin() {
	for {}
}

func count() i64 {
	n := 0
	for n < 100 {
		n = n + 1
	}
	return n
}

func recurse() i64 {
	return recurse()
}
`

// instantiate compiles source and instantiates it in a new wazero runtime that closes
// with the calling spec or benchmark.
func instantiate(ctx context.Context, source string) (api.Module, func()) {
	program, err := arc.CompileText(ctx, arc.Text{Raw: source}, arc.NewRoot(nil))
	if err != nil {
		panic(err)
	}
	rt := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigCompiler())
	guest, err := rt.Instantiate(ctx, program.WASM)
	if err != nil {
		panic(err)
	}
	return guest, func() {
		if err := rt.Close(ctx); err != nil {
			panic(err)
		}
	}
}

var _ = Describe("Budget", func() {
	var guest api.Module
	BeforeEach(func(ctx SpecContext) {
		var closeGuest func()
		guest, closeGuest = instantiate(ctx, budgetSource)
		DeferCleanup(closeGuest)
	})

	It("Should trap a loop that uses the whole budget", func(ctx SpecContext) {
		budget := MustSucceed(runtime.NewBudget(guest, 1000))
		budget.Reset()
		Expect(guest.ExportedFunction("spin").Call(ctx)).Error().
			To(MatchError(ContainSubstring("unreachable")))
		Expect(budget.Err()).To(MatchError(runtime.ErrLoopBudget))
	})

	It("Should trap every later call once the budget is used up", func(
		ctx SpecContext,
	) {
		budget := MustSucceed(runtime.NewBudget(guest, 1000))
		budget.Reset()
		for range 2 {
			Expect(guest.ExportedFunction("spin").Call(ctx)).Error().
				To(MatchError(ContainSubstring("unreachable")))
		}
		Expect(guest.ExportedFunction("count").Call(ctx)).Error().
			To(MatchError(ContainSubstring("unreachable")))
		Expect(budget.Err()).To(MatchError(runtime.ErrLoopBudget))
	})

	It("Should trap a recursive function with no loop", func(ctx SpecContext) {
		budget := MustSucceed(runtime.NewBudget(guest, 1000))
		budget.Reset()
		Expect(guest.ExportedFunction("recurse").Call(ctx)).Error().
			To(MatchError(ContainSubstring("unreachable")))
		Expect(budget.Err()).To(MatchError(runtime.ErrLoopBudget))
	})

	It("Should run a loop that stays under the budget", func(ctx SpecContext) {
		budget := MustSucceed(runtime.NewBudget(guest, 1000))
		budget.Reset()
		Expect(guest.ExportedFunction("count").Call(ctx)).To(Equal([]uint64{100}))
		Expect(budget.Err()).To(Succeed())
	})

	It("Should give the guest its whole budget again on Reset", func(ctx SpecContext) {
		budget := MustSucceed(runtime.NewBudget(guest, 150))
		for range 3 {
			budget.Reset()
			Expect(guest.ExportedFunction("count").Call(ctx)).
				To(Equal([]uint64{100}))
			Expect(budget.Err()).To(Succeed())
		}
	})

	It("Should call the guest without allocating", func(ctx SpecContext) {
		budget := MustSucceed(runtime.NewBudget(guest, 1000))
		count := guest.ExportedFunction("count")
		stack := make([]uint64, 1)
		Expect(testing.AllocsPerRun(100, func() {
			budget.Reset()
			if err := count.CallWithStack(ctx, stack); err != nil {
				panic(err)
			}
		})).To(BeZero())
	})

	It("Should return an error for a guest without the budget global", func(
		ctx SpecContext,
	) {
		rt := wazero.NewRuntime(ctx)
		DeferCleanup(rt.Close)
		bare := MustSucceed(rt.Instantiate(ctx, wasm.NewModule().Generate()))
		Expect(runtime.NewBudget(bare, 1000)).Error().
			To(MatchError(ContainSubstring(wasm.LoopBudgetExport)))
	})

	It("Should never run out for a program with no guest", func() {
		var budget runtime.Budget
		budget.Reset()
		Expect(budget.Err()).To(Succeed())
	})
})

func BenchmarkLoop(b *testing.B) {
	guest, closeGuest := instantiate(b.Context(), `
func count() i64 {
	n := 0
	for n < 1000000 {
		n = n + 1
	}
	return n
}
`)
	b.Cleanup(closeGuest)
	budget, err := runtime.NewBudget(guest, runtime.DefaultLoopBudget)
	if err != nil {
		b.Fatal(err)
	}
	count := guest.ExportedFunction("count")
	stack := make([]uint64, 1)
	for b.Loop() {
		budget.Reset()
		if err := count.CallWithStack(b.Context(), stack); err != nil {
			b.Fatal(err)
		}
	}
}
