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
	"strings"

	"github.com/antlr4-go/antlr/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ccontext "github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/expression"
	"github.com/synnaxlabs/arc/compiler/resolve"
	. "github.com/synnaxlabs/arc/compiler/testutil"
	. "github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("CompileCall", func() {
	var ctx ccontext.Context[antlr.ParserRuleContext]
	addFunc := func(
		name string,
		exec symbol.ExecContext,
		inputs, outputs types.Params,
	) *symbol.Symbol {
		GinkgoHelper()
		return MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{
			Name: name,
			Kind: symbol.KindFunction,
			Exec: exec,
			Type: types.Function(types.FunctionProperties{
				Inputs:  inputs,
				Outputs: outputs,
			}),
		}))
	}
	i64Out := types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}}

	// Local call indices follow registration order: pow 0, half 1, same 2, noout 3,
	// bad_default 4. old_pow routes to pow and flow_only never compiles.
	BeforeEach(func(bCtx SpecContext) {
		ctx = NewContext(bCtx)
		for i, name := range []string{"pow", "half", "same", "noout", "bad_default"} {
			ctx.Resolver.RegisterLocal(name, uint32(i))
		}
		pow := addFunc("pow", symbol.ExecWASM, types.Params{
			{Name: "base", Type: types.I64()},
			{Name: "exp", Type: types.I64(), Value: int64(2)},
		}, i64Out)
		addFunc("half", symbol.ExecWASM,
			types.Params{{Name: "x", Type: types.F64()}},
			types.Params{{Name: ir.DefaultOutputParam, Type: types.F64()}},
		)
		addFunc("same", symbol.ExecWASM,
			types.Params{{Name: "x", Type: types.Variable("T", nil)}},
			types.Params{{Name: ir.DefaultOutputParam, Type: types.Variable("T", nil)}},
		)
		addFunc("noout", symbol.ExecWASM, nil, nil)
		addFunc("bad_default", symbol.ExecWASM,
			types.Params{{Name: "x", Type: types.I64(), Value: "oops"}}, i64Out,
		)
		addFunc("flow_only", symbol.ExecFlow, nil, i64Out)
		MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{
			Name:       "old_pow",
			Kind:       symbol.KindFunction,
			Exec:       symbol.ExecWASM,
			Type:       pow.Type,
			Deprecated: pow,
		}))
		MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{
			Name: "v",
			Kind: symbol.KindVariable,
			Type: types.I64(),
		}))
		MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{
			Name: "s",
			Kind: symbol.KindVariable,
			Type: types.Series(types.I64()),
		}))
	})

	// args parses each source as an argument. A "name=value" source is named, any
	// other source is positional at its index.
	args := func(sources ...string) []symbol.Argument {
		GinkgoHelper()
		out := make([]symbol.Argument, len(sources))
		for i, source := range sources {
			name, value, named := strings.Cut(source, "=")
			if !named {
				name, value = "", source
			}
			expr := MustSucceed(parser.ParseExpression(value))
			out[i] = symbol.Argument{Index: i, Name: name, Expr: expr, AST: expr}
		}
		return out
	}

	DescribeTable("should bind arguments to parameters",
		func(name string, sources []string, out types.Type, expected ...any) {
			Expect(expression.CompileCall(ctx, name, "", args(sources...))).
				To(Equal(out))
			Expect(FinalizeContext(ctx)).To(MatchOpcodes(expected...))
		},
		Entry("positional in order", "pow", []string{"3", "4"}, types.I64(),
			OpI64Const, int64(3), OpI64Const, int64(4), OpCall, uint32(0)),
		Entry("named in order", "pow", []string{"base=3", "exp=4"}, types.I64(),
			OpI64Const, int64(3), OpI64Const, int64(4), OpCall, uint32(0)),
		Entry("named out of order", "pow", []string{"exp=4", "base=3"}, types.I64(),
			OpI64Const, int64(3), OpI64Const, int64(4), OpCall, uint32(0)),
		Entry("positional then named", "pow", []string{"3", "exp=4"}, types.I64(),
			OpI64Const, int64(3), OpI64Const, int64(4), OpCall, uint32(0)),
		Entry("default fills a missing optional parameter", "pow", []string{"3"},
			types.I64(),
			OpI64Const, int64(3), OpI64Const, int64(2), OpCall, uint32(0)),
		Entry("argument cast to the parameter type", "half", []string{"i32(3)"},
			types.F64(),
			OpI32Const, int32(3), OpF64ConvertI32S, OpCall, uint32(1)),
		Entry("type variable resolved from the argument", "same", []string{"f32(3)"},
			types.F32(),
			OpF32Const, float32(3), OpCall, uint32(2)),
		Entry("no output returns no type", "noout", []string{}, types.Type{},
			OpCall, uint32(3)),
		Entry("deprecated alias calls its replacement", "old_pow", []string{"3", "4"},
			types.I64(),
			OpI64Const, int64(3), OpI64Const, int64(4), OpCall, uint32(0)),
	)

	DescribeTable("should dispatch the len builtins without resolving a symbol",
		func(head, tail, source string) {
			Expect(expression.CompileCall(ctx, head, tail, args(source))).
				To(Equal(types.I64()))
			Expect(FinalizeContext(ctx)).ToNot(BeEmpty())
		},
		Entry("len", "len", "", `"hello"`),
		Entry("string.len", "string", "len", `"hello"`),
		Entry("series.len", "series", "len", "s"),
	)

	DescribeTable("should reject a call that cannot bind",
		func(head, tail string, sources []string, substr string) {
			Expect(expression.CompileCall(ctx, head, tail, args(sources...))).Error().
				To(MatchError(ContainSubstring(substr)))
		},
		Entry("missing a required parameter", "pow", "", []string{"exp=4"},
			"function pow is missing an argument for parameter base"),
		Entry("more arguments than parameters", "pow", "", []string{"3", "4", "5"},
			"function pow expects at most 2 arguments"),
		Entry("unknown parameter name", "pow", "", []string{"nope=3"},
			"function pow has no parameter nope"),
		Entry("default value of the wrong type", "bad_default", "", []string{},
			"unexpected default value type string for i64"),
		Entry("callee is not a function", "v", "", []string{},
			"v is not a function"),
		Entry("callee is flow-only", "flow_only", "", []string{},
			"cannot be called inside a func block"),
		Entry("callee is undefined", "nope", "", []string{},
			"undefined symbol: nope"),
		Entry("len with two arguments", "len", "", []string{`"a"`, `"b"`},
			"len() requires exactly 1 argument, got 2"),
	)

	It("Should resolve a qualified name through its module", func(bCtx SpecContext) {
		root := symbol.NewRoot(nil, stl.NewSymbols())
		autoImportSTL(bCtx, root)
		ctx := ccontext.NewRoot(
			bCtx, root, map[antlr.ParserRuleContext]types.Type{}, resolve.NewResolver(),
		)
		Expect(expression.CompileCall(ctx, "time", "now", nil)).
			To(Equal(types.TimeStamp()))
		Expect(FinalizeContext(ctx)).To(MatchOpcodes(OpCall, uint32(0)))
	})
})
