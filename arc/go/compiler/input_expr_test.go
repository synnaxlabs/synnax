// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package compiler_test

import (
	"context"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/compiler"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/tetratelabs/wazero"
)

var _ = Describe("Input Expression Synthetics", func() {
	var r wazero.Runtime
	BeforeEach(func(ctx SpecContext) { r = wazero.NewRuntime(ctx) })
	AfterEach(func(ctx SpecContext) { Expect(r.Close(ctx)).To(Succeed()) })

	flowResolver := []symbol.Symbol{
		{Name: "trig", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 100},
		{Name: "out", Kind: symbol.KindChannel, Type: types.WriteChan(types.I64()), ID: 101},
	}

	// compileAndFindSynth compiles source and returns the lone input_expr$ synthetic's
	// export name alongside the output.
	compileAndFindSynth := func(ctx context.Context, source string) (compiler.Output, string) {
		prog := MustSucceed(text.Parse(text.Text{Raw: source}))
		inter, diag := text.Analyze(ctx, prog, NewRoot(nil, flowResolver...))
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		var synthKey string
		for _, f := range inter.Functions {
			if strings.HasPrefix(f.Key, compiler.InputExprSyntheticPrefix) {
				Expect(synthKey).To(BeEmpty(), "expected exactly one input_expr synthetic")
				synthKey = f.Key
				Expect(f.Inputs).To(BeEmpty(), "synthetic must be nullary")
				Expect(f.Outputs).To(HaveLen(1))
			}
		}
		return MustSucceed(compiler.Compile(ctx, inter)), synthKey
	}

	It("Should compile an arithmetic brace input into a callable nullary synthetic", func(ctx SpecContext) {
		out, synthKey := compileAndFindSynth(ctx, `
		func emit{val i64} () i64 { return val }
		trig -> emit{val = 40 + 2} -> out`)
		Expect(synthKey).ToNot(BeEmpty())

		mod := MustSucceed(r.Instantiate(ctx, out.WASM))
		fn := mod.ExportedFunction(synthKey)
		Expect(fn).ToNot(BeNil())
		Expect(fn.Call(ctx)).To(ConsistOf(uint64(42)))
	})

	It("Should compile a function-call brace input into a callable synthetic", func(ctx SpecContext) {
		out, synthKey := compileAndFindSynth(ctx, `
		func double(x i64) i64 { return x * 2 }
		func emit{val i64} () i64 { return val }
		trig -> emit{val = double(21)} -> out`)
		Expect(synthKey).ToNot(BeEmpty())

		mod := MustSucceed(r.Instantiate(ctx, out.WASM))
		Expect(mod.ExportedFunction(synthKey).Call(ctx)).To(ConsistOf(uint64(42)))
	})

	It("Should emit one synthetic per non-literal brace input", func(ctx SpecContext) {
		prog := MustSucceed(text.Parse(text.Text{Raw: `
		func emit{a i64, b i64} () i64 { return a + b }
		trig -> emit{a = 1 + 1, b = 2 + 2} -> out`}))
		inter, diag := text.Analyze(ctx, prog, NewRoot(nil, flowResolver...))
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		count := 0
		for _, f := range inter.Functions {
			if strings.HasPrefix(f.Key, compiler.InputExprSyntheticPrefix) {
				count++
			}
		}
		Expect(count).To(Equal(2))
		MustSucceed(compiler.Compile(ctx, inter))
	})

	It("Should not emit a synthetic for a literal brace input", func(ctx SpecContext) {
		prog := MustSucceed(text.Parse(text.Text{Raw: `
		func emit{val i64} () i64 { return val }
		trig -> emit{val = 42} -> out`}))
		inter, diag := text.Analyze(ctx, prog, NewRoot(nil, flowResolver...))
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		for _, f := range inter.Functions {
			Expect(f.Key).ToNot(HavePrefix(compiler.InputExprSyntheticPrefix))
		}
	})

	It("Should return an error when an input_expr synthetic lacks an expression AST", func(ctx SpecContext) {
		inter := ir.IR{Functions: ir.Functions{{
			Key:     compiler.InputExprSyntheticPrefix + "broken",
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
		}}}
		Expect(compiler.Compile(ctx, inter)).Error().To(MatchError(ContainSubstring("input expr synthetic")))
	})
})
