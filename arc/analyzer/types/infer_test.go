// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Type Inference", func() {
	var (
		bCtx         context.Context
		testResolver symbol.MapResolver
	)

	BeforeEach(func() {
		bCtx = context.Background()
		testResolver = symbol.MapResolver{
			"temp_sensor": symbol.Symbol{
				Name: "temp_sensor",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F32()),
				ID:   1,
			},
			"pressure": symbol.Symbol{
				Name: "pressure",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F64()),
				ID:   2,
			},
			"float_var": symbol.Symbol{
				Name: "float_var",
				Kind: symbol.KindVariable,
				Type: types.F32(),
			},
		}
	})

	Context("Channel unwrapping in expressions", func() {
		It("should unwrap channel types when used in expressions", func() {
			expr := MustSucceed(parser.ParseExpression(`temp_sensor + 10`))
			ctx := acontext.CreateRoot(bCtx, expr, testResolver)
			inferredType := atypes.InferFromExpression(ctx)

			// Should infer as f32, not chan f32
			Expect(inferredType.Kind).To(Equal(types.KindF32))
		})

		It("should unwrap channel types in multiplicative expressions", func() {
			expr := MustSucceed(parser.ParseExpression(`temp_sensor * 2`))
			ctx := acontext.CreateRoot(bCtx, expr, testResolver)
			inferredType := atypes.InferFromExpression(ctx)

			Expect(inferredType.Kind).To(Equal(types.KindF32))
		})

		It("should handle multiple channel operations", func() {
			expr := MustSucceed(parser.ParseExpression(`temp_sensor + temp_sensor`))
			ctx := acontext.CreateRoot(bCtx, expr, testResolver)
			inferredType := atypes.InferFromExpression(ctx)

			Expect(inferredType.Kind).To(Equal(types.KindF32))
		})
	})

	Context("Literal type inference with type hints", func() {
		It("should infer integer literals as type variables with integer constraint", func() {
			expr := MustSucceed(parser.ParseExpression(`42`))
			ctx := acontext.CreateRoot(bCtx, expr, testResolver)
			inferredType := atypes.InferFromExpression(ctx)

			// Integer literals infer as type variables with integer constraint
			Expect(inferredType.Kind).To(Equal(types.KindTypeVariable))
			Expect(inferredType.Constraint).ToNot(BeNil())
			Expect(inferredType.Constraint.Kind).To(Equal(types.KindIntegerConstant))
		})

		It("should infer decimal literals as type variables with float constraint", func() {
			expr := MustSucceed(parser.ParseExpression(`3.14`))
			ctx := acontext.CreateRoot(bCtx, expr, testResolver)
			inferredType := atypes.InferFromExpression(ctx)

			// Float literals infer as type variables with float constraint
			Expect(inferredType.Kind).To(Equal(types.KindTypeVariable))
			Expect(inferredType.Constraint).ToNot(BeNil())
			Expect(inferredType.Constraint.Kind).To(Equal(types.KindFloatConstant))
		})

		It("should infer literals from binary operation context - f32 channel", func() {
			expr := MustSucceed(parser.ParseExpression(`temp_sensor * 1.8`))
			ctx := acontext.CreateRoot(bCtx, expr, testResolver)
			inferredType := atypes.InferFromExpression(ctx)

			// The literal 1.8 should infer as f32 from temp_sensor's type
			Expect(inferredType.Kind).To(Equal(types.KindF32))
		})

		It("should infer literals from binary operation context - f64 channel", func() {
			expr := MustSucceed(parser.ParseExpression(`pressure / 2`))
			ctx := acontext.CreateRoot(bCtx, expr, testResolver)
			inferredType := atypes.InferFromExpression(ctx)

			// The literal 2 should infer as f64 from pressure's type
			Expect(inferredType.Kind).To(Equal(types.KindF64))
		})

		It("should infer literals in additive operations", func() {
			expr := MustSucceed(parser.ParseExpression(`temp_sensor + 32`))
			ctx := acontext.CreateRoot(bCtx, expr, testResolver)
			inferredType := atypes.InferFromExpression(ctx)

			Expect(inferredType.Kind).To(Equal(types.KindF32))
		})
	})

	Context("Complex expressions", func() {
		It("should handle parenthesized expressions", func() {
			expr := MustSucceed(parser.ParseExpression(`(temp_sensor * 2) + 1`))
			ctx := acontext.CreateRoot(bCtx, expr, testResolver)
			inferredType := atypes.InferFromExpression(ctx)

			Expect(inferredType.Kind).To(Equal(types.KindF32))
		})

		It("should handle comparison expressions", func() {
			expr := MustSucceed(parser.ParseExpression(`temp_sensor > 100`))
			ctx := acontext.CreateRoot(bCtx, expr, testResolver)
			inferredType := atypes.InferFromExpression(ctx)

			// Comparisons return u8 (boolean)
			Expect(inferredType.Kind).To(Equal(types.KindU8))
		})

		It("should handle logical expressions", func() {
			expr := MustSucceed(parser.ParseExpression(`temp_sensor > 100 && pressure < 50`))
			ctx := acontext.CreateRoot(bCtx, expr, testResolver)
			inferredType := atypes.InferFromExpression(ctx)

			Expect(inferredType.Kind).To(Equal(types.KindU8))
		})
	})

	Context("Regression tests", func() {
		It("should correctly infer temp_sensor * 1.8 + 32 (regression)", func() {
			// This was failing with "cannot use chan f32 and f64 in * operation"
			// The literal 1.8 should infer as f32 from temp_sensor's type
			expr := MustSucceed(parser.ParseExpression(`temp_sensor * 1.8 + 32`))
			ctx := acontext.CreateRoot(bCtx, expr, testResolver)
			inferredType := atypes.InferFromExpression(ctx)

			// Should infer as f32 (channel unwrapped, literals inferred as f32)
			Expect(inferredType.Kind).To(Equal(types.KindF32))
		})

		It("should handle channel in complex arithmetic (regression)", func() {
			expr := MustSucceed(parser.ParseExpression(`(temp_sensor * 1.8) + 32`))
			ctx := acontext.CreateRoot(bCtx, expr, testResolver)
			inferredType := atypes.InferFromExpression(ctx)

			Expect(inferredType.Kind).To(Equal(types.KindF32))
		})
	})

	Context("Type compatibility", func() {
		It("should consider compatible types equal", func() {
			Expect(atypes.Compatible(types.F32(), types.F32())).To(BeTrue())
		})

		It("should unwrap channels for compatibility", func() {
			Expect(atypes.Compatible(types.Chan(types.F32()), types.F32())).To(BeTrue())
		})

		It("should unwrap both channels for compatibility", func() {
			Expect(atypes.Compatible(types.Chan(types.F32()), types.Chan(types.F32()))).To(BeTrue())
		})

		It("should reject incompatible types", func() {
			Expect(atypes.Compatible(types.F32(), types.F64())).To(BeFalse())
		})
	})

	Context("Literal assignment compatibility", func() {
		It("should allow same type assignment", func() {
			Expect(atypes.LiteralAssignmentCompatible(types.F32(), types.F32())).To(BeTrue())
		})

		It("should allow signed integer to any integer", func() {
			Expect(atypes.LiteralAssignmentCompatible(types.U32(), types.I64())).To(BeTrue())
		})

		It("should allow numeric to float", func() {
			Expect(atypes.LiteralAssignmentCompatible(types.F64(), types.I64())).To(BeTrue())
			Expect(atypes.LiteralAssignmentCompatible(types.F32(), types.F64())).To(BeTrue())
		})
	})
})
