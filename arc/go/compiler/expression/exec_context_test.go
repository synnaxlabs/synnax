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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/expression"
	. "github.com/synnaxlabs/arc/compiler/testutil"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ExecContext", func() {
	It("Should reject a flow-only function in a func block", func(bCtx SpecContext) {
		ctx := NewContext(bCtx)
		funcType := types.Function(types.FunctionProperties{
			Inputs:  types.Params{{Name: "a", Type: types.I64()}},
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
		})
		MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{
			Name: "avg",
			Kind: symbol.KindFunction,
			Type: funcType,
			Exec: symbol.ExecFlow,
		}))
		expr := MustSucceed(parser.ParseExpression("avg(10)"))
		_, err := expression.Compile(context.Child(ctx, expr))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("cannot be called inside a func block"))
	})

	It("Should allow a WASM function in a func block", func(bCtx SpecContext) {
		ctx := NewContext(bCtx)
		ctx.Resolver.RegisterLocal("pow", 3)
		funcType := types.Function(types.FunctionProperties{
			Inputs:  types.Params{{Name: "base", Type: types.I64()}, {Name: "exp", Type: types.I64()}},
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
		})
		MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{
			Name: "pow",
			Kind: symbol.KindFunction,
			Type: funcType,
			Exec: symbol.ExecWASM,
		}))
		_, exprType := compileWithCtx(ctx, "pow(2, 3)")
		Expect(exprType).To(Equal(types.I64()))
	})

	It("Should allow an ExecBoth function in a func block", func(bCtx SpecContext) {
		ctx := NewContext(bCtx)
		ctx.Resolver.RegisterLocal("dual", 4)
		funcType := types.Function(types.FunctionProperties{
			Inputs:  types.Params{{Name: "x", Type: types.I64()}},
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
		})
		MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{
			Name: "dual",
			Kind: symbol.KindFunction,
			Type: funcType,
			Exec: symbol.ExecBoth,
		}))
		_, exprType := compileWithCtx(ctx, "dual(5)")
		Expect(exprType).To(Equal(types.I64()))
	})
})

var _ = Describe("SignedNumericConstraint Promotion", func() {
	It("Should promote u8 to i16 for math.neg", func(bCtx SpecContext) {
		resolver := symbol.CompoundResolver{
			symbol.MapResolver{
				"x": {Name: "x", Kind: symbol.KindVariable, Type: types.U8(), ID: 0},
			},
			stl.SymbolResolver,
		}
		_, exprType := compileWithAnalyzer(bCtx, "math.neg(x)", resolver)
		Expect(exprType).To(Equal(types.I16()))
	})
	It("Should promote u16 to i32 for math.neg", func(bCtx SpecContext) {
		resolver := symbol.CompoundResolver{
			symbol.MapResolver{
				"x": {Name: "x", Kind: symbol.KindVariable, Type: types.U16(), ID: 0},
			},
			stl.SymbolResolver,
		}
		_, exprType := compileWithAnalyzer(bCtx, "math.neg(x)", resolver)
		Expect(exprType).To(Equal(types.I32()))
	})
	It("Should promote u32 to i64 for math.neg", func(bCtx SpecContext) {
		resolver := symbol.CompoundResolver{
			symbol.MapResolver{
				"x": {Name: "x", Kind: symbol.KindVariable, Type: types.U32(), ID: 0},
			},
			stl.SymbolResolver,
		}
		_, exprType := compileWithAnalyzer(bCtx, "math.neg(x)", resolver)
		Expect(exprType).To(Equal(types.I64()))
	})
	It("Should promote u64 to f64 for math.neg", func(bCtx SpecContext) {
		resolver := symbol.CompoundResolver{
			symbol.MapResolver{
				"x": {Name: "x", Kind: symbol.KindVariable, Type: types.U64(), ID: 0},
			},
			stl.SymbolResolver,
		}
		_, exprType := compileWithAnalyzer(bCtx, "math.neg(x)", resolver)
		Expect(exprType).To(Equal(types.F64()))
	})
	It("Should not promote signed types for math.neg", func(bCtx SpecContext) {
		resolver := symbol.CompoundResolver{
			symbol.MapResolver{
				"x": {Name: "x", Kind: symbol.KindVariable, Type: types.I32(), ID: 0},
			},
			stl.SymbolResolver,
		}
		_, exprType := compileWithAnalyzer(bCtx, "math.neg(x)", resolver)
		Expect(exprType).To(Equal(types.I32()))
	})
})
