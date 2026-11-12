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
)

var (
	unitMS  = &types.Unit{Dimensions: types.DimTime, Scale: 1e6, Name: "ms"}
	unitS   = &types.Unit{Dimensions: types.DimTime, Scale: 1e9, Name: "s"}
	unitMin = &types.Unit{Dimensions: types.DimTime, Scale: 60e9, Name: "min"}
	unitPa  = &types.Unit{Dimensions: types.DimPressure, Scale: 1, Name: "Pa"}
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
				Expect(w.Code).To(Equal(codes.UnitTruncation.Code))
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
			"'d' has no unit and cannot take a value in s"),
		Entry("another dimension", i64(unitS), i64(unitPa),
			"incompatible dimensions: cannot write s to 'd' (Pa)"),
		Entry("a series in another unit",
			types.Series(i64(unitS)), types.Series(i64(unitMS)),
			"'d' is a series in ms and cannot take a series in s"),
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
	)

	It("Should do nothing for an expression with no node", func(bCtx SpecContext) {
		expr := MustSucceed(parser.ParseExpression("x * 2"))
		ctx := acontext.NewRoot(bCtx, expr, nil)
		Expect(func() { expression.TakeUnit(ctx, expr, unitMS) }).ToNot(Panic())
	})
})
