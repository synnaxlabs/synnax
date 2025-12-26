// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package units_test

import (
	stdctx "context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/units"
	"github.com/synnaxlabs/arc/diagnostics"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

func testCtx() context.Context[parser.IProgramContext] {
	return context.Context[parser.IProgramContext]{
		Context:     stdctx.Background(),
		Diagnostics: &diagnostics.Diagnostics{},
	}
}

// makeType is a helper to create a types.Type with an optional unit.
func makeType(unitName string) types.Type {
	if unitName == "" {
		return types.Type{Kind: types.KindF64}
	}
	return types.Type{Kind: types.KindF64, Unit: units.MustResolve(unitName)}
}

var _ = Describe("Analysis", func() {
	Describe("ValidateBinaryOp", func() {
		// Multiplication and division are always dimensionally valid
		Context("Multiplication and Division", func() {
			DescribeTable("should allow any dimension combinations",
				func(op, leftUnit, rightUnit string) {
					ctx := testCtx()
					left, right := makeType(leftUnit), makeType(rightUnit)
					Expect(units.ValidateBinaryOp(ctx, op, left, right)).To(BeTrue())
					Expect(ctx.Diagnostics.Errors()).To(BeEmpty())
				},
				Entry("multiply same dimensions", "*", "m", "m"),
				Entry("multiply different dimensions", "*", "m", "s"),
				Entry("multiply dimensionless values", "*", "", ""),
				Entry("multiply unit by dimensionless", "*", "m", ""),
				Entry("divide same dimensions", "/", "m", "m"),
				Entry("divide different dimensions", "/", "m", "s"),
				Entry("divide dimensionless values", "/", "", ""),
				Entry("divide unit by dimensionless", "/", "m", ""),
			)
		})

		// Addition, subtraction, modulo, and comparisons require matching dimensions
		Context("Dimension-Matching Operations", func() {
			DescribeTable("should allow matching or dimensionless operands",
				func(op, leftUnit, rightUnit string) {
					ctx := testCtx()
					left, right := makeType(leftUnit), makeType(rightUnit)
					Expect(units.ValidateBinaryOp(ctx, op, left, right)).To(BeTrue())
					Expect(ctx.Diagnostics.Errors()).To(BeEmpty())
				},
				// Addition
				Entry("add same dimensions (pressure)", "+", "psi", "Pa"),
				Entry("add dimensionless to dimensioned", "+", "", "psi"),
				Entry("add dimensionless values", "+", "", ""),
				// Subtraction
				Entry("subtract same dimensions", "-", "psi", "Pa"),
				Entry("subtract dimensionless values", "-", "", ""),
				// Modulo
				Entry("modulo same dimensions", "%", "m", "m"),
				Entry("modulo dimensionless values", "%", "", ""),
				// Comparisons
				Entry("greater than same dimensions", ">", "psi", "Pa"),
				Entry("less than dimensionless", "<", "", ""),
				Entry("equals dimensionless", "==", "", ""),
				Entry("greater than or equal with unit to dimensionless", ">=", "psi", ""),
				Entry("not equals same dimensions", "!=", "m", "m"),
				Entry("less than or equal same dimensions", "<=", "s", "ms"),
			)

			DescribeTable("should reject incompatible dimensions",
				func(op, leftUnit, rightUnit string) {
					ctx := testCtx()
					left, right := makeType(leftUnit), makeType(rightUnit)
					Expect(units.ValidateBinaryOp(ctx, op, left, right)).To(BeFalse())
					Expect(ctx.Diagnostics.Errors()).To(HaveLen(1))
				},
				Entry("add pressure and time", "+", "psi", "s"),
				Entry("subtract length and time", "-", "m", "s"),
				Entry("modulo length and time", "%", "m", "s"),
				Entry("compare pressure and time", "<", "psi", "s"),
				Entry("compare length and frequency", ">", "m", "Hz"),
				Entry("equals pressure and voltage", "==", "psi", "V"),
			)
		})
	})

	Describe("ValidatePowerOp", func() {
		DescribeTable("valid power operations",
			func(baseUnit, expUnit string, isLiteral bool) {
				base, exp := makeType(baseUnit), makeType(expUnit)
				Expect(units.ValidatePowerOp(base, exp, isLiteral)).To(Succeed())
			},
			Entry("dimensionless base and exponent", "", "", false),
			Entry("dimensioned base with literal exponent", "m", "", true),
			Entry("dimensionless base with dimensionless exponent", "", "", true),
		)

		DescribeTable("invalid power operations",
			func(baseUnit, expUnit string, isLiteral bool, expectedErr error, msgSubstring string) {
				base, exp := makeType(baseUnit), makeType(expUnit)
				err := units.ValidatePowerOp(base, exp, isLiteral)
				Expect(err).To(MatchError(expectedErr))
				Expect(err.Error()).To(ContainSubstring(msgSubstring))
			},
			Entry("dimensioned exponent",
				"m", "s", false,
				units.DimensionsError, "dimensionless"),
			Entry("non-literal exponent with dimensioned base",
				"m", "", false,
				units.DimensionsError, "literal integer exponent"),
		)
	})

	Describe("ScaleFactor", func() {
		DescribeTable("valid conversions",
			func(fromUnit, toUnit string, expected float64, approx bool) {
				from := MustBeOk(units.Resolve(fromUnit))
				to := MustBeOk(units.Resolve(toUnit))
				factor := MustSucceed(units.ScaleFactor(from, to))
				if approx {
					Expect(factor).To(BeNumerically("~", expected, 0.01))
				} else {
					Expect(factor).To(Equal(expected))
				}
			},
			Entry("km to m", "km", "meter", 1000.0, false),
			Entry("ms to s", "ms", "s", 1e-3, false),
			Entry("psi to Pa", "psi", "Pa", 6894.76, true),
			Entry("same unit returns 1.0", "psi", "psi", 1.0, false),
		)

		It("should return 1.0 for both nil", func() {
			factor := MustSucceed(units.ScaleFactor(nil, nil))
			Expect(factor).To(Equal(1.0))
		})

		DescribeTable("invalid conversions",
			func(from, to *types.Unit, expectedErr error, msgSubstring string) {
				_, err := units.ScaleFactor(from, to)
				Expect(err).To(MatchError(expectedErr))
				Expect(err.Error()).To(ContainSubstring(msgSubstring))
			},
			Entry("incompatible dimensions",
				MustBeOk(units.Resolve("psi")), MustBeOk(units.Resolve("s")),
				units.IncompatibleDimensionsError, "cannot convert"),
			Entry("nil to dimensioned",
				nil, MustBeOk(units.Resolve("psi")),
				units.DimensionsError, "cannot convert between dimensioned and dimensionless"),
			Entry("dimensioned to nil",
				MustBeOk(units.Resolve("psi")), nil,
				units.DimensionsError, "cannot convert between dimensioned and dimensionless"),
		)
	})
})
