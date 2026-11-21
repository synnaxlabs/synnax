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
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/codes"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/expression"
	"github.com/synnaxlabs/arc/analyzer/testutil"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
)

var (
	unitMS  = &types.Unit{Dimensions: types.DimTime, Scale: 1e6, Name: "ms"}
	unitS   = &types.Unit{Dimensions: types.DimTime, Scale: 1e9, Name: "s"}
	unitMin = &types.Unit{Dimensions: types.DimTime, Scale: 60e9, Name: "min"}
	unitPa  = &types.Unit{Dimensions: types.DimPressure, Scale: 1, Name: "Pa"}
	unitPsi = &types.Unit{Dimensions: types.DimPressure, Scale: 6894.76, Name: "psi"}
	unitNS  = &types.Unit{Dimensions: types.DimTime, Scale: 1, Name: "ns"}
	unitHz  = &types.Unit{Dimensions: types.DimFrequency, Scale: 1, Name: "hz"}
)

var _ = Describe("CheckUnits", func() {
	i64 := func(unit *types.Unit) types.Type {
		return types.WithUnit(types.I64(), unit)
	}
	check := func(
		bCtx SpecContext,
		source string,
		w expression.Write,
	) (*diagnostics.Diagnostics, bool, bool) {
		GinkgoHelper()
		expr := MustSucceed(parser.ParseExpression(source))
		ctx := acontext.NewRoot(bCtx, expr, nil)
		w.AST, w.Expr, w.Subject = expr, expr, "'d'"
		converts, ok := expression.CheckUnits(ctx, w)
		return ctx.Diagnostics, converts, ok
	}

	DescribeTable(
		"values that the target takes",
		func(bCtx SpecContext, source string, to types.Type, converts bool, warnings int) {
			diag, got, ok := check(bCtx, source, expression.Write{
				Source: i64(unitS),
				Target: to,
			})
			Expect(ok).To(BeTrue())
			Expect(got).To(Equal(converts))
			Expect(diag.Errors()).To(BeEmpty())
			found := testutil.Warnings(diag)
			Expect(found).To(HaveLen(warnings), diag.String())
			for _, w := range found {
				Expect(w.Code).To(Equal(codes.UnitPrecisionLoss.Code))
			}
		},
		Entry("the same unit", "x", i64(unitS), false, 0),
		Entry("a smaller unit", "x", i64(unitMS), true, 0),
		Entry("a larger unit", "x", i64(unitMin), true, 1),
		Entry("a float in a larger unit",
			"x", types.WithUnit(types.F64(), unitMin), true, 0),
		Entry("a literal with no remainder", "120s", i64(unitMin), false, 0),
		Entry("a literal with a remainder", "90s", i64(unitMin), false, 1),
		Entry("a target that is not a number", "x", types.String(), false, 0),
	)

	DescribeTable("rates that a time span takes",
		func(bCtx SpecContext, source string, to types.Type, converts bool) {
			diag, got, ok := check(bCtx, source, expression.Write{
				Source: i64(unitHz),
				Target: to,
			})
			Expect(ok).To(BeTrue())
			Expect(got).To(Equal(converts))
			Expect(*diag).To(BeEmpty())
		},
		Entry("a variable", "x", i64(unitNS), true),
		Entry("a variable with a period that is not whole", "x", i64(unitMin), true),
		Entry("a float", "x", types.WithUnit(types.F64(), unitS), true),
		Entry("a literal", "10hz", i64(unitNS), false),
	)

	DescribeTable("values that the target rejects",
		func(bCtx SpecContext, from, to types.Type, message string) {
			diag, converts, ok := check(bCtx, "x", expression.Write{
				Source: from,
				Target: to,
			})
			Expect(ok).To(BeFalse())
			Expect(converts).To(BeFalse())
			Expect(diag.Errors()).To(HaveLen(1), diag.String())
			Expect(diag.Errors()[0].Message).To(Equal(message))
		},
		Entry("a target with no unit", i64(unitS), types.I64(),
			"missing unit: 'd' has no unit and cannot take a value in s"),
		Entry("another dimension", i64(unitS), i64(unitPa),
			"incompatible dimensions: cannot write s to 'd' (Pa)"),
		Entry("a series in another unit",
			types.Series(i64(unitS)), types.Series(i64(unitMS)),
			"type mismatch: 'd' is a series in ms and cannot take a series in s"),
		Entry("a time span for a rate", i64(unitS), i64(unitHz),
			"incompatible dimensions: cannot write s to 'd' (hz)"),
		Entry("a rate for a pressure", i64(unitHz), i64(unitPa),
			"incompatible dimensions: cannot write hz to 'd' (Pa)"),
		Entry("a series of rates for a series of time spans",
			types.Series(i64(unitHz)), types.Series(i64(unitNS)),
			"type mismatch: 'd' is a series in ns and cannot take a series in hz"),
	)

	It("Should leave the truncation warning to the lowering", func(bCtx SpecContext) {
		diag, converts, ok := check(bCtx, "x", expression.Write{
			Source:  i64(unitS),
			Target:  i64(unitMin),
			Lowered: true,
		})
		Expect(ok).To(BeTrue())
		Expect(converts).To(BeTrue())
		Expect(*diag).To(BeEmpty())
	})
})

var _ = Describe("Unit fixes", func() {
	// expectFixes expects the one diagnostic of code in source to have fixes whose
	// edits have the new text of expected.
	expectFixes := func(
		bCtx SpecContext,
		source string,
		code diagnostics.ErrorCode,
		expected [][]string,
	) {
		GinkgoHelper()
		ast := MustSucceed(parser.Parse(source))
		ctx := acontext.NewRoot(bCtx, ast, buildExpressionRoot(nil))
		analyzer.AnalyzeProgram(ctx)
		found := lo.Filter(
			*ctx.Diagnostics,
			func(d diagnostics.Diagnostic, _ int) bool { return d.Code == code },
		)
		Expect(found).To(HaveLen(1), ctx.Diagnostics.String())
		Expect(found[0].Fixes).To(HaveLen(len(expected)))
		for i, f := range found[0].Fixes {
			Expect(lo.Map(f.Edits, func(e protocol.TextEdit, _ int) string {
				return e.NewText
			})).To(Equal(expected[i]))
		}
	}
	fix := func(edits ...string) []string { return edits }

	DescribeTable("a write that truncates an integer",
		func(bCtx SpecContext, source string, expected ...[]string) {
			expectFixes(bCtx, source, codes.UnitPrecisionLoss.Code, expected)
		},
		Entry("a declaration", "func f() {\nd i64 min := 90s\n}",
			fix("d f64 min := 90s"), fix("d i64 s := 90s")),
		Entry("a stateful declaration", "func f() {\nd i64 min $= 90s\n}",
			fix("d f64 min $= 90s"), fix("d i64 s $= 90s")),
		Entry("a declaration far under one of the larger unit",
			"func f() {\nd i64 min := 1ns\n}",
			fix("d f64 min := 1ns"), fix("d i64 ns := 1ns")),
		Entry("a 32-bit declaration far under one of the larger unit",
			"func f() {\nx i32 K := 6pK\n}",
			fix("x f64 K := 6pK"), fix("x i32 pK := 6pK")),
		Entry("an assignment of a literal far under one of the larger unit",
			"func f() {\nd i64 min := 0min\nd = 1ns\n}",
			fix("d f64 min := 0min"), fix("d i64 ns := 0ns")),
		Entry("an assignment of a literal",
			"func f() {\nd i64 min := 0min\nd = 90s\n}",
			fix("d f64 min := 0min"), fix("d i64 s := 0s")),
		Entry("an assignment of a variable",
			"func f(x i64 s) {\nd i64 min := 0min\nd = x\n}",
			fix("d f64 min := 0min", "f64 s(x)"), fix("d i64 s := 0s")),
		Entry("a variable with no declared type",
			"func f(x i64 s) {\nd := 5min\nd = x\n}",
			fix("d f64 min := 5min", "f64 s(x)"), fix("d := 300s")),
		Entry("a variable declared from a number with no unit",
			"func f(x i64 s) {\nd i64 min := 5\nd = x\n}",
			fix("d f64 min := 5", "f64 s(x)"), fix("d i64 s := 300")),
		Entry("a variable that is not whole in the other unit",
			"func f(x i64 Pa) {\nd i64 psi := 5psi\nd = x\n}",
			fix("d f64 psi := 5psi", "f64 Pa(x)")),
		Entry("a declaration from a variable", "func f(x i64 s) {\nd i64 min := x\n}"),
	)

	DescribeTable("a cast that removes a unit",
		func(bCtx SpecContext, source string, expected ...[]string) {
			expectFixes(bCtx, source, codes.UnitDropped.Code, expected)
		},
		Entry("a literal", "func f() {\nd i64 ms := i64(5s)\n}",
			fix("5s"), fix("5ms")),
		Entry("a float literal into an integer",
			"func f() {\nd i64 ms := i64(1.5s)\n}", fix("i64 ms(1.5s)")),
		Entry("a float literal into a float",
			"func f() {\nd f64 ms := f64(1.5s)\n}", fix("1.5s"), fix("1.5ms")),
		Entry("a variable", "func f(x i64 s) {\nd i64 ms := 0ms\nd = i64(x)\n}",
			fix("i64 ms(x)")),
		Entry("a rate for a time span",
			"func f(x i64 hz) {\nd i64 ms := 0ms\nd = i64(x)\n}", fix("i64 ms(x)")),
	)

	DescribeTable("a value with a unit for a name with no unit",
		func(bCtx SpecContext, source string, expected ...[]string) {
			expectFixes(bCtx, source, codes.UnitMissing.Code, expected)
		},
		Entry("a compound assignment", "func f() {\ntotal i64 := 0\ntotal += 5ms\n}",
			fix("total i64 ms := 0"), fix("i64(5ms)")),
		Entry("a result", "func f() {d i64 ns} i64 {\nreturn d\n}",
			fix("i64 ns"), fix("i64(d)")),
		Entry("a variable declared from a variable",
			"func f(x i64 s, y i64) {\nd i64 := y\nd = x\n}", fix("i64(x)")),
	)
})

var _ = Describe("CheckOperandUnits", func() {
	in := func(t types.Type, unit *types.Unit) types.Type {
		return types.WithUnit(t, unit)
	}
	check := func(
		bCtx SpecContext,
		left, right types.Type,
	) (*diagnostics.Diagnostics, bool) {
		GinkgoHelper()
		expr := MustSucceed(parser.ParseExpression("x"))
		ctx := acontext.NewRoot(bCtx, expr, nil)
		ok := expression.CheckOperandUnits(ctx, left, right)
		return ctx.Diagnostics, ok
	}

	DescribeTable("operands that convert",
		func(bCtx SpecContext, left, right types.Type, warnings ...string) {
			diag, ok := check(bCtx, left, right)
			Expect(ok).To(BeTrue())
			Expect(diag.Errors()).To(BeEmpty())
			found := testutil.Warnings(diag)
			Expect(found).To(HaveLen(len(warnings)), diag.String())
			for i, w := range found {
				Expect(w.Code).To(Equal(codes.UnitPrecisionLoss.Code))
				Expect(w.Message).To(ContainSubstring(warnings[i]))
			}
		},
		Entry("the same unit", in(types.I64(), unitS), in(types.I64(), unitS)),
		Entry("a whole factor", in(types.I64(), unitS), in(types.I64(), unitMS)),
		Entry("a factor that is not whole",
			in(types.I64(), unitPsi), in(types.I64(), unitPa), "from psi to Pa"),
		Entry("the larger unit on the right",
			in(types.I64(), unitPa), in(types.I64(), unitPsi), "from psi to Pa"),
		Entry("a float by a factor that is not whole",
			in(types.F64(), unitPsi), in(types.F64(), unitPa)),
		Entry("another dimension", in(types.I64(), unitS), in(types.I64(), unitPa)),
		Entry("no unit on one side", in(types.I64(), unitS), types.I64()),
		Entry("an integer series by a whole factor",
			types.Series(in(types.I64(), unitS)), in(types.I64(), unitMS)),
		Entry("an integer series in the smaller unit",
			types.Series(in(types.I64(), unitPa)), in(types.I64(), unitPsi),
			"from psi to Pa"),
		Entry("a float series by a factor that is not whole",
			types.Series(in(types.F64(), unitPsi)), in(types.F64(), unitPa)),
	)

	DescribeTable("integer series that do not convert",
		func(bCtx SpecContext, left, right types.Type, message string) {
			diag, ok := check(bCtx, left, right)
			Expect(ok).To(BeFalse())
			Expect(diag.Errors()).To(HaveLen(1), diag.String())
			Expect(diag.Errors()[0].Message).To(Equal("invalid operand: " + message))
		},
		Entry("a series on the left",
			types.Series(in(types.I64(), unitPsi)), in(types.I64(), unitPa),
			"a series of i64 in psi does not convert to Pa, use a float series or the "+
				"same unit"),
		Entry("a series on the right",
			in(types.I64(), unitPa), types.Series(in(types.I64(), unitPsi)),
			"a series of i64 in psi does not convert to Pa, use a float series or the "+
				"same unit"),
		Entry("a 32-bit series by a whole factor that does not fit",
			types.Series(in(types.I32(), unitMin)), in(types.I32(), unitNS),
			"a series of i32 in min does not convert to ns, use a float series or the "+
				"same unit"),
	)
})

var _ = Describe("TakeUnit", func() {
	seconds := types.WithUnit(types.I64(), unitS)

	DescribeTable("the node of an expression",
		func(
			bCtx SpecContext,
			kind symbol.Kind,
			t types.Type,
			unit, input, output *types.Unit,
		) {
			expr := MustSucceed(parser.ParseExpression("x * 2"))
			root := NewRoot(nil)
			node := MustSucceed(root.Add(bCtx, symbol.Symbol{
				Kind: kind,
				AST:  expr,
				Type: types.Function(types.FunctionProperties{
					Inputs:  types.Params{{Name: ir.DefaultInputParam, Type: t}},
					Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: t}},
				}),
			}))
			expression.TakeUnit(acontext.NewRoot(bCtx, expr, root), expr, unit)
			Expect(node.Type.Inputs[0].Type.Unit).To(BeIdenticalTo(input))
			Expect(node.Type.Outputs[0].Type.Unit).To(BeIdenticalTo(output))
		},
		Entry("an output takes the unit",
			symbol.KindFunction, seconds, unitMS, unitS, unitMS),
		Entry("a constant takes the unit on its input",
			symbol.KindConstant, seconds, unitMS, unitMS, unitMS),
		Entry(
			"a node with no unit keeps none",
			symbol.KindFunction,
			types.I64(),
			unitMS,
			(*types.Unit)(nil),
			(*types.Unit)(nil),
		),
		Entry("a node in another dimension keeps its unit",
			symbol.KindFunction, seconds, unitPa, unitS, unitS),
		Entry("a rate takes the unit of a time span",
			symbol.KindFunction, types.WithUnit(types.I64(), unitHz),
			unitNS, unitHz, unitNS),
		Entry("a time span keeps its unit for a rate",
			symbol.KindFunction, seconds, unitHz, unitS, unitS),
	)

	It("Should do nothing for an expression with no node", func(bCtx SpecContext) {
		expr := MustSucceed(parser.ParseExpression("x * 2"))
		ctx := acontext.NewRoot(bCtx, expr, nil)
		Expect(func() { expression.TakeUnit(ctx, expr, unitMS) }).ToNot(Panic())
	})
})
