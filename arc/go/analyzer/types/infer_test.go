// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/arc/analyzer/constraints"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/testutil"
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
			Expect(inferredType.Kind).To(Equal(types.KindVariable))
			Expect(inferredType.Constraint).ToNot(BeNil())
			Expect(inferredType.Constraint.Kind).To(Equal(types.KindIntegerConstant))
		})

		It("should infer decimal literals as type variables with float constraint", func() {
			expr := MustSucceed(parser.ParseExpression(`3.14`))
			ctx := acontext.CreateRoot(bCtx, expr, testResolver)
			inferredType := atypes.InferFromExpression(ctx)

			// Float literals infer as type variables with float constraint
			Expect(inferredType.Kind).To(Equal(types.KindVariable))
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
			expr := MustSucceed(parser.ParseExpression(`temp_sensor > 100 and pressure < 50`))
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

		It("should reject type variables", func() {
			tv := types.Variable("T", nil)
			Expect(atypes.Compatible(tv, types.F32())).To(BeFalse())
			Expect(atypes.Compatible(types.F32(), tv)).To(BeFalse())
			Expect(atypes.Compatible(tv, tv)).To(BeFalse())
		})

		It("should reject channel vs series mismatch", func() {
			Expect(atypes.Compatible(types.Chan(types.F32()), types.Series(types.F32()))).To(BeFalse())
			Expect(atypes.Compatible(types.Series(types.I32()), types.Chan(types.I32()))).To(BeFalse())
		})

		It("should handle series compatibility", func() {
			Expect(atypes.Compatible(types.Series(types.F32()), types.Series(types.F32()))).To(BeTrue())
			Expect(atypes.Compatible(types.Series(types.I64()), types.I64())).To(BeTrue())
			Expect(atypes.Compatible(types.I64(), types.Series(types.I64()))).To(BeTrue())
		})

		It("should reject nested channels of different depth", func() {
			// chan<chan<i32>> and chan<i32> are fundamentally different types
			Expect(atypes.Compatible(types.Chan(types.Chan(types.I32())), types.Chan(types.I32()))).To(BeFalse())
			Expect(atypes.Compatible(types.Chan(types.I32()), types.Chan(types.Chan(types.I32())))).To(BeFalse())
		})

		It("should handle nested channels of same depth", func() {
			// chan<chan<i32>> should be compatible with chan<chan<i32>>
			Expect(atypes.Compatible(types.Chan(types.Chan(types.I32())), types.Chan(types.Chan(types.I32())))).To(BeTrue())
		})

		It("should reject invalid types", func() {
			invalid := types.Type{}
			Expect(atypes.Compatible(invalid, types.F32())).To(BeFalse())
			Expect(atypes.Compatible(types.F32(), invalid)).To(BeFalse())
			Expect(atypes.Compatible(invalid, invalid)).To(BeFalse())
		})

		It("should reject nested series vs base type", func() {
			Expect(atypes.Compatible(types.Series(types.Series(types.F32())), types.F32())).To(BeFalse())
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

	Context("Check function", func() {
		var cs *constraints.System

		BeforeEach(func() {
			cs = constraints.New()
		})

		It("should add constraint for type variables", func() {
			tv := types.Variable("T", nil)
			ast := testutil.NewMockAST(1)
			err := atypes.Check(cs, tv, types.I32(), ast, "test")
			Expect(err).ToNot(HaveOccurred())
			Expect(cs.Constraints).ToNot(BeEmpty())
		})

		It("should recursively check channel types", func() {
			ast := testutil.NewMockAST(1)
			Expect(atypes.Check(cs, types.Chan(types.F32()), types.Chan(types.F32()), ast, "test")).To(Succeed())
		})

		It("should error on channel type mismatch", func() {
			ast := testutil.NewMockAST(1)
			err := atypes.Check(cs, types.Chan(types.F32()), types.F32(), ast, "test")
			Expect(err).To(MatchError(ContainSubstring("type mismatch")))
		})

		It("should recursively check series types", func() {
			ast := testutil.NewMockAST(1)
			err := atypes.Check(cs, types.Series(types.I64()), types.Series(types.I64()), ast, "test")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should error on series type mismatch", func() {
			ast := testutil.NewMockAST(1)
			err := atypes.Check(cs, types.Series(types.I64()), types.I64(), ast, "test")
			Expect(err).To(MatchError(ContainSubstring("type mismatch")))
		})

		It("should check concrete types for equality", func() {
			ast := testutil.NewMockAST(1)
			err := atypes.Check(cs, types.F32(), types.F32(), ast, "test")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should error on concrete type mismatch", func() {
			ast := testutil.NewMockAST(1)
			err := atypes.Check(cs, types.F32(), types.F64(), ast, "test")
			Expect(err).To(MatchError(ContainSubstring("type mismatch")))
		})
	})

	Context("InferFromTypeContext", func() {
		It("should return zero type for nil context", func() {
			t, err := atypes.InferFromTypeContext(nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(t).To(Equal(types.Type{}))
		})

		It("should infer types from variable declarations", func() {
			stmt := MustSucceed(parser.ParseStatement("x i32 := 5"))
			varDecl := stmt.VariableDeclaration().LocalVariable()
			typeCtx := varDecl.Type_()
			t, err := atypes.InferFromTypeContext(typeCtx)
			Expect(err).ToNot(HaveOccurred())
			Expect(t.Kind).To(Equal(types.KindI32))
		})

		It("should infer f64 type from declaration", func() {
			stmt := MustSucceed(parser.ParseStatement("y f64 := 3.14"))
			varDecl := stmt.VariableDeclaration().LocalVariable()
			typeCtx := varDecl.Type_()
			t, err := atypes.InferFromTypeContext(typeCtx)
			Expect(err).ToNot(HaveOccurred())
			Expect(t.Kind).To(Equal(types.KindF64))
		})

		It("should infer string type", func() {
			stmt := MustSucceed(parser.ParseStatement("s str := \"hello\""))
			varDecl := stmt.VariableDeclaration().LocalVariable()
			typeCtx := varDecl.Type_()
			t, err := atypes.InferFromTypeContext(typeCtx)
			Expect(err).ToNot(HaveOccurred())
			Expect(t.Kind).To(Equal(types.KindString))
		})

		It("should infer channel types", func() {
			stmt := MustSucceed(parser.ParseStatement("c chan f32 := temp_sensor"))
			varDecl := stmt.VariableDeclaration().LocalVariable()
			typeCtx := varDecl.Type_()
			t, err := atypes.InferFromTypeContext(typeCtx)
			Expect(err).ToNot(HaveOccurred())
			Expect(t.Kind).To(Equal(types.KindChan))
			Expect(t.Unwrap().Kind).To(Equal(types.KindF32))
		})

		It("should infer series types", func() {
			stmt := MustSucceed(parser.ParseStatement("s series i64 := data"))
			varDecl := stmt.VariableDeclaration().LocalVariable()
			typeCtx := varDecl.Type_()
			t, err := atypes.InferFromTypeContext(typeCtx)
			Expect(err).ToNot(HaveOccurred())
			Expect(t.Kind).To(Equal(types.KindSeries))
			Expect(t.Unwrap().Kind).To(Equal(types.KindI64))
		})
	})
})
