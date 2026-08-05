// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package flow_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/flow"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("AnalyzeSingleExpression", func() {
	var root *symbol.Symbol
	BeforeEach(func() {
		testChannels := []symbol.Symbol{
			{
				Name: "temp_sensor",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F32()),
				ID:   10,
			},
			{
				Name: "pressure",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F64()),
				ID:   11,
			},
			{
				Name: "ox_pt_1",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F64()),
				ID:   12,
			},
			{
				Name: "ox_pt_2",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F64()),
				ID:   13,
			},
		}
		root = NewRoot(nil, testChannels...)
	})

	Describe("Pure Literals", func() {
		It("should create KindConstant for integer literal", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`42`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			constSym := MustSucceed(ctx.Scope.Resolve(ctx, "constant_0"))
			Expect(constSym.Kind).To(Equal(symbol.KindConstant))
			Expect(constSym.Type.Kind).To(Equal(types.KindFunction))
			valueCfg := MustBeOk(constSym.Type.Inputs.Get("value"))
			Expect(
				valueCfg.Type.IsNumeric() || valueCfg.Type.Kind == types.KindVariable,
			).To(BeTrue())
			output := MustBeOk(constSym.Type.Outputs.Get(ir.DefaultOutputParam))
			Expect(
				output.Type.IsNumeric() || output.Type.Kind == types.KindVariable,
			).To(BeTrue())
		})

		It("should create KindConstant for float literal", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`3.14`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			constSym := MustSucceed(ctx.Scope.Resolve(ctx, "constant_0"))
			Expect(constSym.Kind).To(Equal(symbol.KindConstant))
			Expect(constSym.Type.Kind).To(Equal(types.KindFunction))
			valueCfg := MustBeOk(constSym.Type.Inputs.Get("value"))
			Expect(
				valueCfg.Type.IsFloat() || valueCfg.Type.Kind == types.KindVariable,
			).To(BeTrue())
		})

		It("should create KindConstant for string literal", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`"hello"`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			constSym := MustSucceed(ctx.Scope.Resolve(ctx, "constant_0"))
			Expect(constSym.Kind).To(Equal(symbol.KindConstant))
			Expect(constSym.Type.Kind).To(Equal(types.KindFunction))
			valueCfg := MustBeOk(constSym.Type.Inputs.Get("value"))
			Expect(valueCfg.Type).To(Equal(types.String()))
		})

		It(
			"should auto-increment constant names for multiple literals",
			func(bCtx SpecContext) {
				expr0 := MustSucceed(parser.ParseExpression(`42`))
				ctx := context.NewRoot(bCtx, expr0, root)
				flow.AnalyzeSingleExpression(ctx)

				expr1 := MustSucceed(parser.ParseExpression(`100`))
				ctx1 := context.Context[parser.IExpressionContext]{
					Context:     bCtx,
					Scope:       ctx.Scope,
					Diagnostics: ctx.Diagnostics,
					Constraints: ctx.Constraints,
					TypeMap:     ctx.TypeMap,
					AST:         expr1,
				}
				flow.AnalyzeSingleExpression(ctx1)

				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				const0 := MustSucceed(ctx.Scope.Resolve(ctx, "constant_0"))
				const1 := MustSucceed(ctx.Scope.Resolve(ctx, "constant_1"))
				Expect(const0.Kind).To(Equal(symbol.KindConstant))
				Expect(const1.Kind).To(Equal(symbol.KindConstant))
			},
		)
	})

	Describe("Complex Expressions", func() {
		It(
			"should create KindFunction for binary expression with channel",
			func(bCtx SpecContext) {
				expr := MustSucceed(parser.ParseExpression(`ox_pt_1 > 100`))
				ctx := context.NewRoot(bCtx, expr, root)
				flow.AnalyzeSingleExpression(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				fnSym := MustSucceed(ctx.Scope.Resolve(ctx, "expression_0"))
				Expect(fnSym.Kind).To(Equal(symbol.KindFunction))
				Expect(fnSym.Type.Kind).To(Equal(types.KindFunction))
				output := MustBeOk(fnSym.Type.Outputs.Get(ir.DefaultOutputParam))
				Expect(output.Type).To(Equal(types.U8()))
			},
		)

		It("should accumulate read channels from expression", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`ox_pt_1 > 100`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			fnSym := MustSucceed(ctx.Scope.Resolve(ctx, "expression_0"))
			Expect(fnSym.Channels.Read).To(HaveLen(1))
			Expect(fnSym.Channels.Read[12]).To(Equal("ox_pt_1"))
		})

		It(
			"should accumulate multiple channels from arithmetic expression",
			func(bCtx SpecContext) {
				expr := MustSucceed(parser.ParseExpression(`ox_pt_1 + ox_pt_2`))
				ctx := context.NewRoot(bCtx, expr, root)
				flow.AnalyzeSingleExpression(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				fnSym := MustSucceed(ctx.Scope.Resolve(ctx, "expression_0"))
				Expect(fnSym.Channels.Read).To(HaveLen(2))
				Expect(fnSym.Channels.Read[12]).To(Equal("ox_pt_1"))
				Expect(fnSym.Channels.Read[13]).To(Equal("ox_pt_2"))
			},
		)

		It(
			"should create KindFunction for logical AND expression",
			func(bCtx SpecContext) {
				expr := MustSucceed(
					parser.ParseExpression(`ox_pt_1 > 100 and pressure > 50`),
				)
				ctx := context.NewRoot(bCtx, expr, root)
				flow.AnalyzeSingleExpression(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				fnSym := MustSucceed(ctx.Scope.Resolve(ctx, "expression_0"))
				Expect(fnSym.Kind).To(Equal(symbol.KindFunction))
				output := MustBeOk(fnSym.Type.Outputs.Get(ir.DefaultOutputParam))
				Expect(output.Type).To(Equal(types.U8()))
			},
		)

		It("should auto-increment expression names", func(bCtx SpecContext) {
			expr0 := MustSucceed(parser.ParseExpression(`ox_pt_1 > 100`))
			ctx := context.NewRoot(bCtx, expr0, root)
			flow.AnalyzeSingleExpression(ctx)

			expr1 := MustSucceed(parser.ParseExpression(`pressure < 50`))
			ctx1 := context.Context[parser.IExpressionContext]{
				Context:     bCtx,
				Scope:       ctx.Scope,
				Diagnostics: ctx.Diagnostics,
				Constraints: ctx.Constraints,
				TypeMap:     ctx.TypeMap,
				AST:         expr1,
			}
			flow.AnalyzeSingleExpression(ctx1)

			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			fn0 := MustSucceed(ctx.Scope.Resolve(ctx, "expression_0"))
			fn1 := MustSucceed(ctx.Scope.Resolve(ctx, "expression_1"))
			Expect(fn0.Kind).To(Equal(symbol.KindFunction))
			Expect(fn1.Kind).To(Equal(symbol.KindFunction))
		})

		It("should handle parenthesized expressions", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`(ox_pt_1 + ox_pt_2) * 2`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			fnSym := MustSucceed(ctx.Scope.Resolve(ctx, "expression_0"))
			Expect(fnSym.Kind).To(Equal(symbol.KindFunction))
		})

		It("should handle type cast expressions", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`f64(temp_sensor)`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			fnSym := MustSucceed(ctx.Scope.Resolve(ctx, "expression_0"))
			Expect(fnSym.Kind).To(Equal(symbol.KindFunction))
			output := MustBeOk(fnSym.Type.Outputs.Get(ir.DefaultOutputParam))
			Expect(output.Type).To(Equal(types.F64()))
		})
	})

	Describe("Raw String Format Literals", func() {
		It(
			"should create KindConstant for raw string without placeholders",
			func(bCtx SpecContext) {
				expr := MustSucceed(parser.ParseExpression(`f"static"`))
				ctx := context.NewRoot(bCtx, expr, root)
				flow.AnalyzeSingleExpression(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				constSym := MustSucceed(ctx.Scope.Resolve(ctx, "constant_0"))
				Expect(constSym.Kind).To(Equal(symbol.KindConstant))
			},
		)

		It(
			"should create fmt_str synthetic function for raw string with placeholder",
			func(bCtx SpecContext) {
				expr := MustSucceed(parser.ParseExpression(`f"v={ox_pt_1}"`))
				ctx := context.NewRoot(bCtx, expr, root)
				flow.AnalyzeSingleExpression(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				fnSym := MustSucceed(ctx.Scope.Resolve(ctx, "fmt_str_0"))
				Expect(fnSym.Kind).To(Equal(symbol.KindFunction))
				Expect(fnSym.Type.Kind).To(Equal(types.KindFunction))
				output := MustBeOk(fnSym.Type.Outputs.Get(ir.DefaultOutputParam))
				Expect(output.Type).To(Equal(types.String()))
			},
		)

		It(
			"should track placeholder channel reads on the synthetic function",
			func(bCtx SpecContext) {
				expr := MustSucceed(parser.ParseExpression(`f"v={ox_pt_1}"`))
				ctx := context.NewRoot(bCtx, expr, root)
				flow.AnalyzeSingleExpression(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				fnSym := MustSucceed(ctx.Scope.Resolve(ctx, "fmt_str_0"))
				Expect(fnSym.Channels.Read).To(HaveLen(1))
				Expect(fnSym.Channels.Read[12]).To(Equal("ox_pt_1"))
			},
		)

		It("should track multiple placeholder channel reads", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`f"a={ox_pt_1} b={ox_pt_2}"`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			fnSym := MustSucceed(ctx.Scope.Resolve(ctx, "fmt_str_0"))
			Expect(fnSym.Channels.Read).To(HaveLen(2))
			Expect(fnSym.Channels.Read[12]).To(Equal("ox_pt_1"))
			Expect(fnSym.Channels.Read[13]).To(Equal("ox_pt_2"))
		})

		It(
			"should accept a numeric literal placeholder with format spec",
			func(bCtx SpecContext) {
				expr := MustSucceed(parser.ParseExpression(`f"x={42%05d}"`))
				ctx := context.NewRoot(bCtx, expr, root)
				flow.AnalyzeSingleExpression(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				fnSym := MustSucceed(ctx.Scope.Resolve(ctx, "fmt_str_0"))
				Expect(fnSym.Kind).To(Equal(symbol.KindFunction))
			},
		)

		It(
			"should auto-increment fmt_str names across multiple raw strings",
			func(bCtx SpecContext) {
				expr0 := MustSucceed(parser.ParseExpression(`f"{ox_pt_1}"`))
				ctx := context.NewRoot(bCtx, expr0, root)
				flow.AnalyzeSingleExpression(ctx)

				expr1 := MustSucceed(parser.ParseExpression(`f"{ox_pt_2}"`))
				ctx1 := context.Context[parser.IExpressionContext]{
					Context:     bCtx,
					Scope:       ctx.Scope,
					Diagnostics: ctx.Diagnostics,
					Constraints: ctx.Constraints,
					TypeMap:     ctx.TypeMap,
					AST:         expr1,
				}
				flow.AnalyzeSingleExpression(ctx1)

				Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
				MustSucceed(ctx.Scope.Resolve(ctx, "fmt_str_0"))
				MustSucceed(ctx.Scope.Resolve(ctx, "fmt_str_1"))
			},
		)

		It(
			"should report unmatched opening brace in raw string body",
			func(bCtx SpecContext) {
				expr := MustSucceed(parser.ParseExpression(`f"{x"`))
				ctx := context.NewRoot(bCtx, expr, root)
				flow.AnalyzeSingleExpression(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring("unmatched"))
			},
		)

		It("should report empty placeholder", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`f"pre {} post"`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(
				(*ctx.Diagnostics)[0].Message,
			).To(ContainSubstring("must contain an expression"))
		})

		It("should report undefined identifier in placeholder", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`f"x={unknown_ch}"`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(
				(*ctx.Diagnostics)[0].Message,
			).To(ContainSubstring("undefined symbol"))
		})

		It(
			"should report invalid format spec for placeholder type",
			func(bCtx SpecContext) {
				expr := MustSucceed(parser.ParseExpression(`f"x={ox_pt_1:s}"`))
				ctx := context.NewRoot(bCtx, expr, root)
				flow.AnalyzeSingleExpression(ctx)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(
					(*ctx.Diagnostics)[0].Message,
				).To(ContainSubstring("invalid format spec"))
			},
		)
	})

	Describe("Error Cases", func() {
		It("should report undefined symbol in expression", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`unknown_channel > 100`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(1))
			Expect(
				(*ctx.Diagnostics)[0].Message,
			).To(Equal("undefined symbol: unknown_channel"))
		})

		It("should report multiple undefined symbols", func(bCtx SpecContext) {
			expr := MustSucceed(parser.ParseExpression(`foo + bar`))
			ctx := context.NewRoot(bCtx, expr, root)
			flow.AnalyzeSingleExpression(ctx)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(*ctx.Diagnostics).To(HaveLen(2))
			Expect((*ctx.Diagnostics)[0].Message).To(Equal("undefined symbol: foo"))
			Expect((*ctx.Diagnostics)[1].Message).To(Equal("undefined symbol: bar"))
		})
	})
})

var _ = Describe("LiftVarReads", func() {
	var (
		root *symbol.Symbol
		fn   *symbol.Symbol
	)
	src := 12
	rwChan := func(elem types.Type) types.Type {
		t := types.Chan(elem)
		t.ChanDirection = types.ChanDirectionRead | types.ChanDirectionWrite
		return t
	}
	addVar := func(bCtx SpecContext, sym symbol.Symbol) {
		MustSucceed(root.Add(bCtx, sym))
	}
	lift := func(bCtx SpecContext, code string) context.Context[parser.IExpressionContext] {
		expr := MustSucceed(parser.ParseExpression(code))
		ctx := context.NewRoot(bCtx, expr, root)
		flow.LiftVarReads(ctx, fn, expr)
		return ctx
	}
	BeforeEach(func(bCtx SpecContext) {
		root = NewRoot(nil, symbol.Symbol{
			Name: "temp_sensor",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F32()),
			ID:   10,
		})
		fn = MustSucceed(root.Add(bCtx, symbol.Symbol{
			Name: "synth",
			Kind: symbol.KindFunction,
			Type: types.Function(types.FunctionProperties{}),
		}))
	})

	It("Should lift a reassigned variable as an input param", func(bCtx SpecContext) {
		addVar(bCtx, symbol.Symbol{
			Name: "x", Kind: symbol.KindVariable,
			Type: types.I32(), Reassigned: true,
		})
		ctx := lift(bCtx, "x + 1")
		Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		Expect(MustBeOk(fn.Type.Inputs.Get("x")).Type).To(Equal(types.I32()))
		child := fn.FindChild("x")
		Expect(child).ToNot(BeNil())
		Expect(child.Kind).To(Equal(symbol.KindInput))
		Expect(child.Internal).To(BeTrue())
	})

	It("Should not lift a variable that was never reassigned", func(bCtx SpecContext) {
		addVar(bCtx, symbol.Symbol{
			Name: "x", Kind: symbol.KindVariable, Type: types.I32(),
		})
		lift(bCtx, "x + 1")
		Expect(fn.Type.Inputs).To(BeEmpty())
	})

	It(
		"Should lift a reactive variable with its unwrapped value type",
		func(bCtx SpecContext) {
			addVar(bCtx, symbol.Symbol{
				Name: "r", Kind: symbol.KindVariable,
				Type: types.ReadChan(types.F32()),
			})
			lift(bCtx, "r * 2.0")
			Expect(MustBeOk(fn.Type.Inputs.Get("r")).Type).To(Equal(types.F32()))
		},
	)

	It(
		"Should lift a reassigned alias with its unwrapped value type",
		func(bCtx SpecContext) {
			addVar(bCtx, symbol.Symbol{
				Name: "c", Kind: symbol.KindVariable,
				Type: rwChan(types.F32()), SourceID: &src, Reassigned: true,
			})
			lift(bCtx, "c + 1.0")
			Expect(MustBeOk(fn.Type.Inputs.Get("c")).Type).To(Equal(types.F32()))
		},
	)

	It("Should skip a name that is already an input param", func(bCtx SpecContext) {
		fn.Type.Inputs = types.Params{{Name: "x", Type: types.I32()}}
		addVar(bCtx, symbol.Symbol{
			Name: "x", Kind: symbol.KindVariable,
			Type: types.I32(), Reassigned: true,
		})
		lift(bCtx, "x + 1")
		Expect(fn.Type.Inputs).To(HaveLen(1))
		Expect(fn.FindChild("x")).To(BeNil())
	})

	It(
		"Should skip an unresolvable identifier without diagnostics",
		func(bCtx SpecContext) {
			ctx := lift(bCtx, "ghost + 1")
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			Expect(fn.Type.Inputs).To(BeEmpty())
		},
	)

	It("Should not lift a channel read", func(bCtx SpecContext) {
		lift(bCtx, "temp_sensor > 100.0")
		Expect(fn.Type.Inputs).To(BeEmpty())
	})

	It("Should lift multiple variables in source order", func(bCtx SpecContext) {
		addVar(bCtx, symbol.Symbol{
			Name: "y", Kind: symbol.KindVariable,
			Type: types.F64(), Reassigned: true,
		})
		addVar(bCtx, symbol.Symbol{
			Name: "x", Kind: symbol.KindVariable,
			Type: types.I32(), Reassigned: true,
		})
		lift(bCtx, "x * y")
		Expect(fn.Type.Inputs).To(HaveLen(2))
		Expect(fn.Type.Inputs[0].Name).To(Equal("x"))
		Expect(fn.Type.Inputs[1].Name).To(Equal("y"))
	})

	It("Should lift a reassigned stateful variable", func(bCtx SpecContext) {
		addVar(bCtx, symbol.Symbol{
			Name: "s", Kind: symbol.KindStatefulVariable,
			Type: types.I64(), Reassigned: true,
		})
		lift(bCtx, "s + 1")
		Expect(MustBeOk(fn.Type.Inputs.Get("s")).Type).To(Equal(types.I64()))
	})

	It(
		"Should not lift an unreassigned read-only variable backed by a real channel",
		func(bCtx SpecContext) {
			addVar(bCtx, symbol.Symbol{
				Name: "r", Kind: symbol.KindVariable,
				Type: types.ReadChan(types.F32()), SourceID: &src,
			})
			lift(bCtx, "r * 2.0")
			Expect(fn.Type.Inputs).To(BeEmpty())
		},
	)
})

var _ = Describe("LiftFmtStrVarReads", func() {
	var (
		root *symbol.Symbol
		fn   *symbol.Symbol
	)
	liftFmt := func(bCtx SpecContext, code string) context.Context[parser.IExpressionContext] {
		expr := MustSucceed(parser.ParseExpression(code))
		ctx := context.NewRoot(bCtx, expr, root)
		flow.LiftFmtStrVarReads(ctx, fn, expr)
		return ctx
	}
	BeforeEach(func(bCtx SpecContext) {
		root = NewRoot(nil)
		fn = MustSucceed(root.Add(bCtx, symbol.Symbol{
			Name: "fmt_fn",
			Kind: symbol.KindFunction,
			Type: types.Function(types.FunctionProperties{}),
		}))
		MustSucceed(root.Add(bCtx, symbol.Symbol{
			Name: "x", Kind: symbol.KindVariable,
			Type: types.I32(), Reassigned: true,
		}))
		MustSucceed(root.Add(bCtx, symbol.Symbol{
			Name: "y", Kind: symbol.KindVariable,
			Type: types.F64(), Reassigned: true,
		}))
	})

	It(
		"Should lift a reassigned variable read in a placeholder",
		func(bCtx SpecContext) {
			ctx := liftFmt(bCtx, `f"value: {x}"`)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			Expect(MustBeOk(fn.Type.Inputs.Get("x")).Type).To(Equal(types.I32()))
		},
	)

	It("Should lift from every placeholder", func(bCtx SpecContext) {
		liftFmt(bCtx, `f"{x} and {y}"`)
		Expect(fn.Type.Inputs).To(HaveLen(2))
	})

	It("Should lift variables inside a placeholder expression", func(bCtx SpecContext) {
		liftFmt(bCtx, `f"{x + 1}"`)
		Expect(MustBeOk(fn.Type.Inputs.Get("x")).Type).To(Equal(types.I32()))
	})

	It("Should lift a placeholder that carries a format spec", func(bCtx SpecContext) {
		liftFmt(bCtx, `f"{y:.2f}"`)
		Expect(MustBeOk(fn.Type.Inputs.Get("y")).Type).To(Equal(types.F64()))
	})

	It("Should lift from a raw-format string", func(bCtx SpecContext) {
		liftFmt(bCtx, `rf"path: {x}"`)
		Expect(MustBeOk(fn.Type.Inputs.Get("x")).Type).To(Equal(types.I32()))
	})

	It("Should be a no-op for a plain string literal", func(bCtx SpecContext) {
		liftFmt(bCtx, `"value: {x}"`)
		Expect(fn.Type.Inputs).To(BeEmpty())
	})

	It("Should be a no-op for a numeric literal", func(bCtx SpecContext) {
		liftFmt(bCtx, "42")
		Expect(fn.Type.Inputs).To(BeEmpty())
	})

	It("Should be a no-op for a malformed format body", func(bCtx SpecContext) {
		ctx := liftFmt(bCtx, `f"{x oops"`)
		Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		Expect(fn.Type.Inputs).To(BeEmpty())
	})

	It("Should skip an empty placeholder without panicking", func(bCtx SpecContext) {
		ctx := liftFmt(bCtx, `f"{}"`)
		Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		Expect(fn.Type.Inputs).To(BeEmpty())
	})

	It("Should be a no-op for a non-literal expression", func(bCtx SpecContext) {
		liftFmt(bCtx, "x + 1")
		Expect(fn.Type.Inputs).To(BeEmpty())
	})

	It(
		"Should skip an unparseable placeholder without diagnostics",
		func(bCtx SpecContext) {
			ctx := liftFmt(bCtx, `f"{x +} {y}"`)
			Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
			_, ok := fn.Type.Inputs.Get("x")
			Expect(ok).To(BeFalse())
			Expect(MustBeOk(fn.Type.Inputs.Get("y")).Type).To(Equal(types.F64()))
		},
	)
})
