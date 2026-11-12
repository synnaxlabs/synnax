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
	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/expression"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

// These specs exercise AnalyzeCall through calls in func bodies. Named-argument
// matching and the externallySatisfied trigger path are driven by the flow analyzer
// and are covered in the flow package's specs.
var _ = Describe("AnalyzeCall", func() {
	Describe("Argument count", func() {
		DescribeTable("valid argument counts",
			func(ctx SpecContext, code string) { expectSuccess(ctx, code, nil) },
			Entry("exact positional arguments", `
				func add() {x i32, y i32} i32 { return x + y }
				func main() { result := add(10, 20) }
			`),
			Entry("required param with optional omitted", `
				func add() {x i64, y i64 = 0} i64 { return x + y }
				func main() { result := add(10) }
			`),
			Entry("all optional params omitted", `
				func defaults() {a i64 = 1, b i64 = 2} i64 { return a + b }
				func main() { result := defaults() }
			`),
		)

		DescribeTable("invalid argument counts",
			func(ctx SpecContext, code, expectedMsg string) {
				expectFailure(ctx, code, nil, expectedMsg)
			},
			Entry("missing required argument", `
				func add() {x i32, y i32} i32 { return x + y }
				func main() { result := add(5) }
			`, "missing required argument for parameter 'y' of func 'add'"),
			Entry("missing the only required argument", `
				func getValue() {x i32} i32 { return x }
				func main() { result := getValue() }
			`, "missing required argument for parameter 'x' of func 'getValue'"),
			Entry("too many arguments", `
				func add() {x i32, y i32} i32 { return x + y }
				func main() { result := add(5, 10, 15) }
			`, "too many arguments for func 'add': expected at most 2"),
		)
	})

	Describe("Argument types", func() {
		DescribeTable("type mismatches",
			func(ctx SpecContext, code, expectedMsg string) {
				expectFailure(ctx, code, nil, expectedMsg)
			},
			Entry("string literal to i32 parameter", `
				func add() {x i32, y i32} i32 { return x + y }
				func main() { result := add(5, "hello") }
			`, "argument 'y' of 'add'"),
			Entry("i32 variable to f32 parameter", `
				func process() {x f32} f32 { return x * 2.0 }
				func main() {
					x i32 := 5
					result := process(x)
				}
			`, "argument 'x' of 'process'"),
		)

		It(
			"should give the cast as the fix for a numeric mismatch",
			func(ctx SpecContext) {
				ast := MustSucceed(parser.Parse(`
				func process() {x f32} f32 { return x * 2.0 }
				func main() {
					x i32 := 5
					result := process(x)
				}
			`))
				aCtx := context.NewRoot(ctx, ast, NewRoot(nil))
				analyzer.AnalyzeProgram(aCtx)
				Expect(*aCtx.Diagnostics).To(HaveLen(1), aCtx.Diagnostics.String())
				d := (*aCtx.Diagnostics)[0]
				Expect(d.Fixes).To(HaveLen(1))
				Expect(d.Fixes[0].Title).To(Equal("Change to `f32(x)`"))
				Expect(d.Fixes[0].Edits).To(HaveLen(1))
				Expect(d.Fixes[0].Edits[0].NewText).To(Equal("f32(x)"))
				Expect(d.Notes).To(HaveLen(2))
				Expect(d.Notes[0].Message).To(Equal("hint: change `x` to `f32(x)`"))
				Expect(d.Notes[1].Message).To(HavePrefix("signature: "))
			},
		)

		It("should give no fix for a mismatch that a cast does not fix",
			func(ctx SpecContext) {
				ast := MustSucceed(parser.Parse(`
					func add() {x i32, y i32} i32 { return x + y }
					func main() { result := add(5, "hello") }
				`))
				aCtx := context.NewRoot(ctx, ast, NewRoot(nil))
				analyzer.AnalyzeProgram(aCtx)
				Expect(*aCtx.Diagnostics).To(HaveLen(1), aCtx.Diagnostics.String())
				d := (*aCtx.Diagnostics)[0]
				Expect(d.Fixes).To(BeEmpty())
				Expect(d.Notes).To(HaveLen(1))
				Expect(d.Notes[0].Message).To(HavePrefix("signature: "))
			})
	})

	Describe("Channel arguments", func() {
		It(
			"Should dereference a channel argument to a value-typed parameter",
			func(ctx SpecContext) {
				expectSuccess(ctx, `
				func read() {x f32} f32 { return x }
				func main() { result := read(temp) }
			`, []symbol.Symbol{
					{
						Name: "temp",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F32()),
						ID:   10,
					},
				})
			},
		)
	})
})

var _ = Describe("AnalyzeCallSite", func() {
	var root, caller, callee *symbol.Symbol
	BeforeEach(func(bCtx SpecContext) {
		root = NewRoot(nil, symbol.Symbol{
			Name: "ch",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F32()),
			ID:   10,
		})
		caller = MustSucceed(root.Add(bCtx, symbol.Symbol{
			Name: "caller",
			Kind: symbol.KindFunction,
			Type: types.Function(types.FunctionProperties{}),
		}))
		callee = MustSucceed(root.Add(bCtx, symbol.Symbol{
			Name: "callee",
			Kind: symbol.KindFunction,
			Exec: symbol.ExecBoth,
			Type: types.Function(types.FunctionProperties{
				Inputs: types.Params{{Name: "c", Type: types.Chan(types.F32())}},
			}),
		}))
	})
	call := func(
		bCtx SpecContext,
		arg symbol.Argument,
	) context.Context[parser.IExpressionContext] {
		GinkgoHelper()
		expr := MustSucceed(parser.ParseExpression("ch"))
		arg.Expr, arg.AST = expr, expr
		ctx := context.NewRoot(bCtx, expr, root).WithScope(caller)
		expression.AnalyzeCallSite(
			ctx, callee, "callee", callee.Type, []symbol.Argument{arg}, expr,
		)
		return ctx
	}
	mapping := context.ChannelMapping{ChannelID: 10, ChannelName: "ch"}

	It(
		"Should record a call edge binding a positional channel argument",
		func(bCtx SpecContext) {
			ctx := call(bCtx, symbol.Argument{Index: 0})
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			Expect(*ctx.CallEdges).To(HaveLen(1))
			edge := (*ctx.CallEdges)[0]
			Expect(edge.Caller).To(BeIdenticalTo(caller))
			Expect(edge.Callee).To(BeIdenticalTo(callee))
			Expect(edge.ArgChannels).To(HaveKeyWithValue(0, mapping))
		},
	)

	It(
		"Should bind a named channel argument by its parameter position",
		func(bCtx SpecContext) {
			ctx := call(bCtx, symbol.Argument{Name: "c"})
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			Expect((*ctx.CallEdges)[0].ArgChannels).To(HaveKeyWithValue(0, mapping))
		},
	)

	It("Should reject a flow-only callee inside a func body", func(bCtx SpecContext) {
		callee.Exec = symbol.ExecFlow
		ctx := call(bCtx, symbol.Argument{Index: 0})
		Expect(ctx.Diagnostics.String()).To(
			ContainSubstring("cannot be called inside a func block"),
		)
		Expect(*ctx.CallEdges).To(BeEmpty())
	})
})

var _ = Describe("Computed arguments", func() {
	It("Should type a computed argument by its node's output", func(bCtx SpecContext) {
		root := NewRoot(nil)
		expr := MustSucceed(parser.ParseExpression("1.5 * 2.0"))
		float := types.FloatConstraint()
		out := types.Variable("out", &float)
		MustSucceed(root.Add(bCtx, symbol.Symbol{
			Kind: symbol.KindFunction,
			AST:  expr,
			Type: types.Function(types.FunctionProperties{
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: out}},
			}),
		}))
		ctx := context.NewRoot(bCtx, expr, root)
		fnType := types.Function(types.FunctionProperties{
			Inputs: types.Params{{Name: "a", Type: types.F32()}},
		})
		expression.AnalyzeCall(
			ctx,
			"f",
			fnType,
			[]symbol.Argument{{Name: "a", Expr: expr, AST: expr}},
			nil,
			expr,
			"",
		)
		Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		Expect(ctx.Constraints.Unify()).To(Succeed())
		Expect(ctx.Constraints.ApplySubstitutions(out)).To(Equal(types.F32()))
	})
})
