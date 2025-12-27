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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer/units"
	"github.com/synnaxlabs/arc/types"
)

// makeTyped creates a types.Type with the given kind and optional unit.
func makeTyped(kind types.Kind, unitName string) types.Type {
	if unitName == "" {
		return types.Type{Kind: kind}
	}
	return types.Type{Kind: kind, Unit: units.MustResolve(unitName)}
}

var _ = Describe("Magnitude Safety", func() {
	Describe("Additive Scale Safety (via ValidateBinaryOp)", func() {
		DescribeTable("should warn for large magnitude differences",
			func(leftUnit, rightUnit string, kind types.Kind, expectedMagnitude string) {
				ctx := testCtx()
				left, right := makeTyped(kind, leftUnit), makeTyped(kind, rightUnit)
				Expect(units.ValidateBinaryOp(ctx, "+", left, right)).To(BeTrue())
				warnings := ctx.Diagnostics.Warnings()
				Expect(warnings).To(HaveLen(1))
				Expect(warnings[0].Message).To(ContainSubstring(expectedMagnitude))
				Expect(warnings[0].Message).To(ContainSubstring("orders of magnitude"))
			},
			Entry("K and pK (12 orders)", "K", "pK", types.KindF64, "12"),
			Entry("f32 m and nm (9 orders, lower threshold)", "m", "nm", types.KindF32, "orders of magnitude"),
		)

		// Special case: K and fK may report 14 or 15 due to floating point
		It("should warn when adding K and fK (14-15 orders of magnitude)", func() {
			ctx := testCtx()
			left := makeTyped(types.KindF64, "K")
			right := makeTyped(types.KindF64, "fK")
			Expect(units.ValidateBinaryOp(ctx, "+", left, right)).To(BeTrue())
			warnings := ctx.Diagnostics.Warnings()
			Expect(warnings).To(HaveLen(1))
			Expect(warnings[0].Message).To(SatisfyAny(
				ContainSubstring("14"),
				ContainSubstring("15"),
			))
		})

		DescribeTable("should not warn for small magnitude differences",
			func(leftUnit, rightUnit string) {
				ctx := testCtx()
				left, right := makeType(leftUnit), makeType(rightUnit)
				Expect(units.ValidateBinaryOp(ctx, "+", left, right)).To(BeTrue())
				Expect(ctx.Diagnostics.Warnings()).To(BeEmpty())
			},
			Entry("m and km (3 orders)", "m", "km"),
			Entry("ms and s (3 orders)", "ms", "s"),
		)

		DescribeTable("should not warn when units are nil",
			func(leftUnit, rightUnit string) {
				ctx := testCtx()
				left, right := makeType(leftUnit), makeType(rightUnit)
				Expect(units.ValidateBinaryOp(ctx, "+", left, right)).To(BeTrue())
				Expect(ctx.Diagnostics.Warnings()).To(BeEmpty())
			},
			Entry("left nil", "", "m"),
			Entry("right nil", "m", ""),
			Entry("both nil", "", ""),
		)

		It("should error (not warn) when dimensions don't match", func() {
			ctx := testCtx()
			Expect(units.ValidateBinaryOp(ctx, "+", makeType("m"), makeType("s"))).To(BeFalse())
			Expect(ctx.Diagnostics.Errors()).To(HaveLen(1))
			Expect(ctx.Diagnostics.Warnings()).To(BeEmpty())
		})

		It("should not check magnitude for non-additive operations", func() {
			ctx := testCtx()
			Expect(units.ValidateBinaryOp(ctx, "*", makeType("K"), makeType("pK"))).To(BeTrue())
			Expect(ctx.Diagnostics.Warnings()).To(BeEmpty())
		})
	})

	Describe("CheckAssignmentScaleSafety", func() {
		DescribeTable("should warn for truncation to zero",
			func(srcUnit string, dstKind types.Kind, dstUnit string) {
				ctx := testCtx()
				src := makeTyped(types.KindF64, srcUnit)
				dst := makeTyped(dstKind, dstUnit)
				units.CheckAssignmentScaleSafety(ctx, src, dst, nil)
				warnings := ctx.Diagnostics.Warnings()
				Expect(warnings).To(HaveLen(1))
				Expect(warnings[0].Message).To(ContainSubstring("truncate to zero"))
			},
			Entry("pK to i32 K", "pK", types.KindI32, "K"),
			Entry("nK to i64 K", "nK", types.KindI64, "K"),
		)

		DescribeTable("should warn for overflow",
			func(srcUnit string, dstKind types.Kind, dstUnit string) {
				ctx := testCtx()
				src := makeTyped(types.KindF64, srcUnit)
				dst := makeTyped(dstKind, dstUnit)
				units.CheckAssignmentScaleSafety(ctx, src, dst, nil)
				warnings := ctx.Diagnostics.Warnings()
				Expect(warnings).To(HaveLen(1))
				Expect(warnings[0].Message).To(ContainSubstring("overflow"))
			},
			Entry("K to i8 pK", "K", types.KindI8, "pK"),
			Entry("K to i16 pK", "K", types.KindI16, "pK"),
		)

		DescribeTable("should not warn for safe assignments",
			func(srcKind types.Kind, srcUnit string, dstKind types.Kind, dstUnit string) {
				ctx := testCtx()
				src := makeTyped(srcKind, srcUnit)
				dst := makeTyped(dstKind, dstUnit)
				units.CheckAssignmentScaleSafety(ctx, src, dst, nil)
				Expect(ctx.Diagnostics.Warnings()).To(BeEmpty())
			},
			Entry("km to i32 m (scale up)", types.KindF64, "km", types.KindI32, "m"),
			Entry("K to f64 pK (float target)", types.KindF64, "K", types.KindF64, "pK"),
			Entry("same unit", types.KindI32, "K", types.KindI32, "K"),
			Entry("src nil", types.KindI32, "", types.KindI32, "K"),
			Entry("dst nil", types.KindI32, "K", types.KindI32, ""),
			Entry("mismatched dimensions", types.KindI32, "m", types.KindI32, "s"),
		)

		It("should use literal value for overflow check", func() {
			src := makeTyped(types.KindF64, "K")
			dst := makeTyped(types.KindI32, "mK")

			// 1000K -> 1e6 mK fits in i32
			ctx1 := testCtx()
			smallVal := 1000.0
			units.CheckAssignmentScaleSafety(ctx1, src, dst, &smallVal)
			Expect(ctx1.Diagnostics.Warnings()).To(BeEmpty())

			// 1e7 K -> 1e10 mK overflows i32
			ctx2 := testCtx()
			largeVal := 1e7
			units.CheckAssignmentScaleSafety(ctx2, src, dst, &largeVal)
			Expect(ctx2.Diagnostics.Warnings()).To(HaveLen(1))
			Expect(ctx2.Diagnostics.Warnings()[0].Message).To(ContainSubstring("overflows"))
		})
	})
})
