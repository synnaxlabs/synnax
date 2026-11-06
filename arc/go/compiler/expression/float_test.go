// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression_test

import (
	"encoding/binary"
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/compiler/expression"
	. "github.com/synnaxlabs/arc/compiler/testutil"
	. "github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Float checks and constants", func() {
	var (
		inf32 = float32(math.Inf(1))
		inf64 = math.Inf(1)
	)
	channels := []symbol.Symbol{
		{Name: "a", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 1},
		{Name: "b", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 2},
	}

	DescribeTable("Should emit a check without a host call",
		func(ctx SpecContext, source string, expected ...any) {
			bytecode, exprType := compileWithAnalyzer(ctx, source, channels)
			Expect(exprType).To(Equal(types.Bool()))
			Expect(bytecode).To(MatchOpcodes(expected...))
		},
		Entry("is_nan f32", "math.is_nan(a)",
			OpI32Const, int32(1), OpCall, uint32(0),
			OpF32Abs, OpF32Const, inf32, OpF32Le, OpI32Eqz),
		Entry("is_nan f64", "math.is_nan(b)",
			OpI32Const, int32(2), OpCall, uint32(0),
			OpF64Abs, OpF64Const, inf64, OpF64Le, OpI32Eqz),
		Entry("is_inf f32", "math.is_inf(a)",
			OpI32Const, int32(1), OpCall, uint32(0),
			OpF32Abs, OpF32Const, inf32, OpF32Eq),
		Entry("is_inf f64", "math.is_inf(b)",
			OpI32Const, int32(2), OpCall, uint32(0),
			OpF64Abs, OpF64Const, inf64, OpF64Eq),
		Entry("is_finite f32", "math.is_finite(a)",
			OpI32Const, int32(1), OpCall, uint32(0),
			OpF32Abs, OpF32Const, inf32, OpF32Lt),
		Entry("is_finite f64", "math.is_finite(b)",
			OpI32Const, int32(2), OpCall, uint32(0),
			OpF64Abs, OpF64Const, inf64, OpF64Lt),
	)

	DescribeTable("Should emit a constant as the type of its other operand",
		func(ctx SpecContext, source string, expected ...any) {
			bytecode, _ := compileWithAnalyzer(ctx, source, channels)
			Expect(bytecode).To(MatchOpcodes(expected...))
		},
		Entry("f32", "a < math.inf",
			OpI32Const, int32(1), OpCall, uint32(0), OpF32Const, inf32, OpF32Lt),
		Entry("f64", "b < math.inf",
			OpI32Const, int32(2), OpCall, uint32(0), OpF64Const, inf64, OpF64Lt),
		Entry("negated f32", "a > -math.inf",
			OpI32Const, int32(1), OpCall, uint32(0),
			OpF32Const, inf32, OpF32Neg, OpF32Gt),
		Entry("first operand f32", "math.inf > a",
			OpF32Const, inf32, OpI32Const, int32(1), OpCall, uint32(0), OpF32Gt),
	)

	It("Should keep a first operand under an outer float hint", func(ctx SpecContext) {
		expr := "f64(math.inf + a)"
		bytecode, exprType := compileWithAnalyzer(ctx, expr, channels)
		Expect(exprType).To(Equal(types.F64()))
		Expect(bytecode).To(MatchOpcodes(
			OpF32Const, inf32,
			OpI32Const, int32(1), OpCall, uint32(0),
			OpF32Add, OpF64PromoteF32,
		))
	})

	It("Should fail on a constant without a float value", func(ctx SpecContext) {
		cCtx := NewContext(ctx)
		MustSucceed(cCtx.Scope.Add(cCtx, symbol.Symbol{
			Name: "broken",
			Kind: symbol.KindConstant,
		}))
		expr := MustSucceed(parser.ParseExpression("broken"))
		Expect(expression.Compile(cCtx.Child(expr))).Error().
			To(MatchError(ContainSubstring("has no float value")))
	})

	It("Should default a constant to f64", func(ctx SpecContext) {
		bytecode, exprType := compileWithAnalyzer(ctx, "math.inf", nil)
		Expect(exprType).To(Equal(types.F64()))
		Expect(bytecode).To(MatchOpcodes(OpF64Const, inf64))
	})

	It("Should write a real f32 NaN", func(ctx SpecContext) {
		bytecode, _ := compileWithAnalyzer(ctx, "a == math.nan", channels)
		i := len(bytecode) - 6
		Expect(Opcode(bytecode[i])).To(Equal(OpF32Const))
		bits := binary.LittleEndian.Uint32(bytecode[i+1:])
		Expect(math.IsNaN(float64(math.Float32frombits(bits)))).To(BeTrue())
	})
})
