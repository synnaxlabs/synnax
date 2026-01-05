// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package units_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer/units"
	"github.com/synnaxlabs/arc/types"
)

var _ = Describe("Magnitude Safety", func() {
	Describe("Additive Scale Safety (via ValidateBinaryOp)", func() {
		Context("Float precision loss", func() {
			It("Should warn when adding K and pK (12 orders of magnitude)", func() {
				ctx := testCtx()
				kelvin := types.Type{
					Kind: types.KindF64,
					Unit: units.MustResolve("K"),
				}
				picoKelvin := types.Type{
					Kind: types.KindF64,
					Unit: units.MustResolve("pK"),
				}
				Expect(units.ValidateBinaryOp(ctx, "+", kelvin, picoKelvin)).To(BeTrue())
				warnings := ctx.Diagnostics.Warnings()
				Expect(warnings).To(HaveLen(1))
				Expect(warnings[0].Message).To(ContainSubstring("12"))
				Expect(warnings[0].Message).To(ContainSubstring("orders of magnitude"))
			})

			It("Should warn when adding K and fK (15 orders of magnitude)", func() {
				ctx := testCtx()
				kelvin := types.Type{
					Kind: types.KindF64,
					Unit: units.MustResolve("K"),
				}
				femtoKelvin := types.Type{
					Kind: types.KindF64,
					Unit: units.MustResolve("fK"),
				}
				Expect(units.ValidateBinaryOp(ctx, "+", kelvin, femtoKelvin)).To(BeTrue())
				warnings := ctx.Diagnostics.Warnings()
				Expect(warnings).To(HaveLen(1))
				// log10(1e15) may give 14 or 15 due to floating point precision
				Expect(warnings[0].Message).To(SatisfyAny(
					ContainSubstring("14"),
					ContainSubstring("15"),
				))
				Expect(warnings[0].Message).To(ContainSubstring("orders of magnitude"))
			})

			It("Should not warn when adding m and km (3 orders of magnitude)", func() {
				ctx := testCtx()
				meter := types.Type{
					Kind: types.KindF64,
					Unit: units.MustResolve("m"),
				}
				kilometer := types.Type{
					Kind: types.KindF64,
					Unit: units.MustResolve("km"),
				}
				Expect(units.ValidateBinaryOp(ctx, "+", meter, kilometer)).To(BeTrue())
				Expect(ctx.Diagnostics.Warnings()).To(BeEmpty())
			})

			It("Should not warn when adding ms and s (3 orders of magnitude)", func() {
				ctx := testCtx()
				milli := types.Type{
					Kind: types.KindF64,
					Unit: units.MustResolve("ms"),
				}
				sec := types.Type{
					Kind: types.KindF64,
					Unit: units.MustResolve("s"),
				}
				Expect(units.ValidateBinaryOp(ctx, "+", milli, sec)).To(BeTrue())
				Expect(ctx.Diagnostics.Warnings()).To(BeEmpty())
			})

			It("Should warn at lower threshold for f32", func() {
				ctx := testCtx()
				meter := types.Type{
					Kind: types.KindF32,
					Unit: units.MustResolve("m"),
				}
				nanometer := types.Type{
					Kind: types.KindF32,
					Unit: units.MustResolve("nm"),
				}
				// m to nm is 9 orders of magnitude, above f32 threshold of 1e5
				Expect(units.ValidateBinaryOp(ctx, "+", meter, nanometer)).To(BeTrue())
				warnings := ctx.Diagnostics.Warnings()
				Expect(warnings).To(HaveLen(1))
				Expect(warnings[0].Message).To(ContainSubstring("orders of magnitude"))
				Expect(warnings[0].Message).To(ContainSubstring("lose precision"))
			})

			It("Should not warn when units are nil", func() {
				ctx := testCtx()
				noUnit := types.Type{Kind: types.KindF64}
				withUnit := types.Type{
					Kind: types.KindF64,
					Unit: units.MustResolve("m"),
				}
				Expect(units.ValidateBinaryOp(ctx, "+", noUnit, withUnit)).To(BeTrue())
				Expect(ctx.Diagnostics.Warnings()).To(BeEmpty())

				ctx2 := testCtx()
				Expect(units.ValidateBinaryOp(ctx2, "+", withUnit, noUnit)).To(BeTrue())
				Expect(ctx2.Diagnostics.Warnings()).To(BeEmpty())

				ctx3 := testCtx()
				Expect(units.ValidateBinaryOp(ctx3, "+", noUnit, noUnit)).To(BeTrue())
				Expect(ctx3.Diagnostics.Warnings()).To(BeEmpty())
			})

			It("Should error (not warn) when dimensions don't match", func() {
				ctx := testCtx()
				meter := types.Type{
					Kind: types.KindF64,
					Unit: units.MustResolve("m"),
				}
				second := types.Type{
					Kind: types.KindF64,
					Unit: units.MustResolve("s"),
				}
				// Dimensions don't match - returns error, not warning
				Expect(units.ValidateBinaryOp(ctx, "+", meter, second)).To(BeFalse())
				Expect(ctx.Diagnostics.Errors()).To(HaveLen(1))
				Expect(ctx.Diagnostics.Warnings()).To(BeEmpty())
			})

			It("Should not check magnitude for non-additive operations", func() {
				ctx := testCtx()
				kelvin := types.Type{
					Kind: types.KindF64,
					Unit: units.MustResolve("K"),
				}
				picoKelvin := types.Type{
					Kind: types.KindF64,
					Unit: units.MustResolve("pK"),
				}
				// Multiplication doesn't trigger magnitude warnings
				Expect(units.ValidateBinaryOp(ctx, "*", kelvin, picoKelvin)).To(BeTrue())
				Expect(ctx.Diagnostics.Warnings()).To(BeEmpty())
			})
		})
	})

	Describe("CheckAssignmentScaleSafety", func() {
		Context("Integer truncation", func() {
			It("Should warn when assigning pK to i32 K", func() {
				ctx := testCtx()
				picoKelvin := types.Type{
					Kind: types.KindF64,
					Unit: units.MustResolve("pK"),
				}
				kelvinI32 := types.Type{
					Kind: types.KindI32,
					Unit: units.MustResolve("K"),
				}
				units.CheckAssignmentScaleSafety(ctx, picoKelvin, kelvinI32, nil)
				warnings := ctx.Diagnostics.Warnings()
				Expect(warnings).To(HaveLen(1))
				Expect(warnings[0].Message).To(ContainSubstring("truncate to zero"))
			})

			It("Should warn when assigning nK to i64 K", func() {
				ctx := testCtx()
				nanoKelvin := types.Type{
					Kind: types.KindF64,
					Unit: units.MustResolve("nK"),
				}
				kelvinI64 := types.Type{
					Kind: types.KindI64,
					Unit: units.MustResolve("K"),
				}
				units.CheckAssignmentScaleSafety(ctx, nanoKelvin, kelvinI64, nil)
				warnings := ctx.Diagnostics.Warnings()
				Expect(warnings).To(HaveLen(1))
				Expect(warnings[0].Message).To(ContainSubstring("truncate to zero"))
			})

			It("Should not warn when assigning km to i32 m", func() {
				ctx := testCtx()
				kilometer := types.Type{
					Kind: types.KindF64,
					Unit: units.MustResolve("km"),
				}
				meterI32 := types.Type{
					Kind: types.KindI32,
					Unit: units.MustResolve("m"),
				}
				units.CheckAssignmentScaleSafety(ctx, kilometer, meterI32, nil)
				// km to m is scale up, not truncation
				Expect(ctx.Diagnostics.Warnings()).To(BeEmpty())
			})
		})

		Context("Integer overflow", func() {
			It("Should warn when assigning K to i8 pK", func() {
				ctx := testCtx()
				kelvin := types.Type{
					Kind: types.KindF64,
					Unit: units.MustResolve("K"),
				}
				picoKelvinI8 := types.Type{
					Kind: types.KindI8,
					Unit: units.MustResolve("pK"),
				}
				units.CheckAssignmentScaleSafety(ctx, kelvin, picoKelvinI8, nil)
				warnings := ctx.Diagnostics.Warnings()
				Expect(warnings).To(HaveLen(1))
				Expect(warnings[0].Message).To(ContainSubstring("overflow"))
			})

			It("Should warn when assigning K to i16 pK", func() {
				ctx := testCtx()
				kelvin := types.Type{
					Kind: types.KindF64,
					Unit: units.MustResolve("K"),
				}
				picoKelvinI16 := types.Type{
					Kind: types.KindI16,
					Unit: units.MustResolve("pK"),
				}
				units.CheckAssignmentScaleSafety(ctx, kelvin, picoKelvinI16, nil)
				warnings := ctx.Diagnostics.Warnings()
				Expect(warnings).To(HaveLen(1))
				Expect(warnings[0].Message).To(ContainSubstring("overflow"))
			})

			It("Should not warn for assignment to float types", func() {
				ctx := testCtx()
				kelvin := types.Type{
					Kind: types.KindF64,
					Unit: units.MustResolve("K"),
				}
				picoKelvinF64 := types.Type{
					Kind: types.KindF64,
					Unit: units.MustResolve("pK"),
				}
				units.CheckAssignmentScaleSafety(ctx, kelvin, picoKelvinF64, nil)
				Expect(ctx.Diagnostics.Warnings()).To(BeEmpty())
			})

			It("Should use literal value when available for overflow check", func() {
				kelvin := types.Type{
					Kind: types.KindF64,
					Unit: units.MustResolve("K"),
				}
				milliKelvinI32 := types.Type{
					Kind: types.KindI32,
					Unit: units.MustResolve("mK"),
				}

				// 1000K -> 1e6 mK, fits in i32
				ctx1 := testCtx()
				smallVal := 1000.0
				units.CheckAssignmentScaleSafety(ctx1, kelvin, milliKelvinI32, &smallVal)
				Expect(ctx1.Diagnostics.Warnings()).To(BeEmpty())

				// 1e7 K -> 1e10 mK, overflows i32 (max ~2.1e9)
				ctx2 := testCtx()
				largeVal := 1e7
				units.CheckAssignmentScaleSafety(ctx2, kelvin, milliKelvinI32, &largeVal)
				warnings := ctx2.Diagnostics.Warnings()
				Expect(warnings).To(HaveLen(1))
				Expect(warnings[0].Message).To(ContainSubstring("overflows"))
			})
		})

		Context("Safe assignments", func() {
			It("Should not warn when units are nil", func() {
				ctx1 := testCtx()
				noUnit := types.Type{Kind: types.KindI32}
				withUnit := types.Type{
					Kind: types.KindI32,
					Unit: units.MustResolve("K"),
				}
				units.CheckAssignmentScaleSafety(ctx1, noUnit, withUnit, nil)
				Expect(ctx1.Diagnostics.Warnings()).To(BeEmpty())

				ctx2 := testCtx()
				units.CheckAssignmentScaleSafety(ctx2, withUnit, noUnit, nil)
				Expect(ctx2.Diagnostics.Warnings()).To(BeEmpty())
			})

			It("Should not warn when dimensions don't match", func() {
				ctx := testCtx()
				meter := types.Type{
					Kind: types.KindI32,
					Unit: units.MustResolve("m"),
				}
				second := types.Type{
					Kind: types.KindI32,
					Unit: units.MustResolve("s"),
				}
				units.CheckAssignmentScaleSafety(ctx, meter, second, nil)
				Expect(ctx.Diagnostics.Warnings()).To(BeEmpty())
			})

			It("Should not warn for same unit assignment", func() {
				ctx := testCtx()
				kelvin := types.Type{
					Kind: types.KindI32,
					Unit: units.MustResolve("K"),
				}
				units.CheckAssignmentScaleSafety(ctx, kelvin, kelvin, nil)
				Expect(ctx.Diagnostics.Warnings()).To(BeEmpty())
			})
		})
	})
})
