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
	"github.com/synnaxlabs/arc/analyzer/constraints"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/testutil"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

// inferExprType is a helper that parses an expression and infers its type.
func inferExprType(
	bCtx context.Context,
	resolver symbol.MapResolver,
	expr string,
) types.Type {
	parsed := MustSucceed(parser.ParseExpression(expr))
	ctx := acontext.CreateRoot(bCtx, parsed, resolver)
	return atypes.InferFromExpression(ctx)
}

// parseTypeFromDecl is a helper that parses a type from a variable declaration.
func parseTypeFromDecl(decl string) parser.ITypeContext {
	stmt := MustSucceed(parser.ParseStatement(decl))
	return stmt.VariableDeclaration().LocalVariable().Type_()
}

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
			"data_series": symbol.Symbol{
				Name: "data_series",
				Kind: symbol.KindVariable,
				Type: types.Series(types.I64()),
			},
			"my_func": symbol.Symbol{
				Name: "my_func",
				Kind: symbol.KindVariable,
				Type: types.Function(types.FunctionProperties{
					Inputs:  types.Params{{Name: "x", Type: types.F32()}},
					Outputs: types.Params{{Name: "", Type: types.F64()}},
				}),
			},
		}
	})

	Describe("InferFromExpression", func() {
		DescribeTable("channel unwrapping",
			func(expr string, expectedKind types.Kind) {
				t := inferExprType(bCtx, testResolver, expr)
				Expect(t.Kind).To(Equal(expectedKind))
			},
			Entry("additive with literal", "temp_sensor + 10", types.KindF32),
			Entry("multiplicative with literal", "temp_sensor * 2", types.KindF32),
			Entry("channel with channel", "temp_sensor + temp_sensor", types.KindF32),
			Entry("f64 channel division", "pressure / 2", types.KindF64),
		)

		DescribeTable("literal type inference",
			func(expr string, expectedKind types.Kind, constraintKind types.Kind) {
				t := inferExprType(bCtx, testResolver, expr)
				Expect(t.Kind).To(Equal(expectedKind))
				if constraintKind != types.KindInvalid {
					Expect(t.Constraint).ToNot(BeNil())
					Expect(t.Constraint.Kind).To(Equal(constraintKind))
				}
			},
			Entry("integer literal", "42", types.KindVariable, types.KindIntegerConstant),
			Entry("float literal", "3.14", types.KindVariable, types.KindFloatConstant),
			Entry("string literal", `"hello"`, types.KindString, types.KindInvalid),
			Entry("boolean true", "true", types.KindU8, types.KindInvalid),
			Entry("boolean false", "false", types.KindU8, types.KindInvalid),
		)

		DescribeTable("literal inference from context",
			func(expr string, expectedKind types.Kind) {
				t := inferExprType(bCtx, testResolver, expr)
				Expect(t.Kind).To(Equal(expectedKind))
			},
			Entry("float literal with f32 channel", "temp_sensor * 1.8", types.KindF32),
			Entry("integer literal with f64 channel", "pressure / 2", types.KindF64),
			Entry("integer literal with f32 channel", "temp_sensor + 32", types.KindF32),
		)

		DescribeTable("comparison and logical expressions",
			func(expr string, expectedKind types.Kind) {
				t := inferExprType(bCtx, testResolver, expr)
				Expect(t.Kind).To(Equal(expectedKind))
			},
			Entry("greater than", "temp_sensor > 100", types.KindU8),
			Entry("less than", "pressure < 50", types.KindU8),
			Entry("equality", "temp_sensor == 0", types.KindU8),
			Entry("inequality", "pressure != 0", types.KindU8),
			Entry("logical and", "temp_sensor > 100 and pressure < 50", types.KindU8),
			Entry("logical or", "temp_sensor > 100 or pressure < 50", types.KindU8),
		)

		DescribeTable("complex expressions",
			func(expr string, expectedKind types.Kind) {
				t := inferExprType(bCtx, testResolver, expr)
				Expect(t.Kind).To(Equal(expectedKind))
			},
			Entry("parenthesized", "(temp_sensor * 2) + 1", types.KindF32),
			Entry("temperature conversion", "temp_sensor * 1.8 + 32", types.KindF32),
			Entry("temperature conversion with parens", "(temp_sensor * 1.8) + 32", types.KindF32),
			Entry("power with channel", "temp_sensor ^ 2", types.KindF32),
			Entry("unary negation with channel", "-temp_sensor", types.KindF32),
			Entry("type cast", "f32(42)", types.KindF32),
		)

		Context("postfix expressions", func() {
			It("should handle function call return type", func() {
				t := inferExprType(bCtx, testResolver, "my_func(1.0)")
				Expect(t.Kind).To(Equal(types.KindF64))
			})

			It("should handle series indexing", func() {
				t := inferExprType(bCtx, testResolver, "data_series[0]")
				Expect(t.Kind).To(Equal(types.KindI64))
			})
		})
	})

	Describe("Compatible", func() {
		DescribeTable("type compatibility",
			func(t1, t2 types.Type, expected bool) {
				Expect(atypes.Compatible(t1, t2)).To(Equal(expected))
			},
			// Same types
			Entry("f32 with f32", types.F32(), types.F32(), true),
			Entry("i64 with i64", types.I64(), types.I64(), true),

			// Channel unwrapping
			Entry("chan f32 with f32", types.Chan(types.F32()), types.F32(), true),
			Entry("f32 with chan f32", types.F32(), types.Chan(types.F32()), true),
			Entry("chan f32 with chan f32", types.Chan(types.F32()), types.Chan(types.F32()), true),

			// Series handling
			Entry("series f32 with series f32", types.Series(types.F32()), types.Series(types.F32()), true),
			Entry("series i64 with i64", types.Series(types.I64()), types.I64(), true),
			Entry("i64 with series i64", types.I64(), types.Series(types.I64()), true),

			// Incompatible types
			Entry("f32 with f64", types.F32(), types.F64(), false),
			Entry("i32 with f32", types.I32(), types.F32(), false),

			// Channel vs series mismatch
			Entry("chan f32 with series f32", types.Chan(types.F32()), types.Series(types.F32()), false),
			Entry("series i32 with chan i32", types.Series(types.I32()), types.Chan(types.I32()), false),

			// Nested channels
			Entry("nested chan same depth", types.Chan(types.Chan(types.I32())), types.Chan(types.Chan(types.I32())), true),
			Entry("nested chan different depth 1", types.Chan(types.Chan(types.I32())), types.Chan(types.I32()), false),
			Entry("nested chan different depth 2", types.Chan(types.I32()), types.Chan(types.Chan(types.I32())), false),

			// Nested series
			Entry("nested series vs base", types.Series(types.Series(types.F32())), types.F32(), false),

			// Type variables (should use constraint system)
			Entry("type var with f32", types.Variable("T", nil), types.F32(), false),
			Entry("f32 with type var", types.F32(), types.Variable("T", nil), false),
			Entry("type var with type var", types.Variable("T", nil), types.Variable("T", nil), false),

			// Invalid types
			Entry("invalid with f32", types.Type{}, types.F32(), false),
			Entry("f32 with invalid", types.F32(), types.Type{}, false),
			Entry("invalid with invalid", types.Type{}, types.Type{}, false),
		)
	})

	Describe("LiteralAssignmentCompatible", func() {
		DescribeTable("literal assignment",
			func(varType, litType types.Type, expected bool) {
				Expect(atypes.LiteralAssignmentCompatible(varType, litType)).To(Equal(expected))
			},
			Entry("same type f32", types.F32(), types.F32(), true),
			Entry("same type i32", types.I32(), types.I32(), true),
			Entry("signed int to unsigned", types.U32(), types.I64(), true),
			Entry("int to float", types.F64(), types.I64(), true),
			Entry("float to wider float", types.F32(), types.F64(), true),
			Entry("invalid var type", types.Type{}, types.I32(), false),
			Entry("invalid lit type", types.F32(), types.Type{}, false),
		)
	})

	Describe("Check", func() {
		var cs *constraints.System

		BeforeEach(func() {
			cs = constraints.New()
		})

		DescribeTable("type checking",
			func(t1, t2 types.Type, expectError bool, errSubstring string) {
				ast := testutil.NewMockAST(1)
				err := atypes.Check(cs, t1, t2, ast, "test")
				if expectError {
					Expect(err).To(MatchError(ContainSubstring(errSubstring)))
				} else {
					Expect(err).ToNot(HaveOccurred())
				}
			},
			// Type variables add constraints, no error
			Entry("type var with concrete", types.Variable("T", nil), types.I32(), false, ""),
			Entry("concrete with type var", types.F32(), types.Variable("T", nil), false, ""),

			// Matching types
			Entry("f32 with f32", types.F32(), types.F32(), false, ""),
			Entry("chan f32 with chan f32", types.Chan(types.F32()), types.Chan(types.F32()), false, ""),
			Entry("series i64 with series i64", types.Series(types.I64()), types.Series(types.I64()), false, ""),

			// Mismatched types
			Entry("f32 with f64", types.F32(), types.F64(), true, "type mismatch"),
			Entry("chan f32 with f32", types.Chan(types.F32()), types.F32(), true, "type mismatch"),
			Entry("series i64 with i64", types.Series(types.I64()), types.I64(), true, "type mismatch"),
		)

		It("should add constraint for type variables", func() {
			tv := types.Variable("T", nil)
			ast := testutil.NewMockAST(1)
			Expect(atypes.Check(cs, tv, types.I32(), ast, "test")).To(Succeed())
			Expect(cs.Constraints).ToNot(BeEmpty())
		})
	})

	Describe("InferFromTypeContext", func() {
		It("should return zero type for nil context", func() {
			t, err := atypes.InferFromTypeContext(nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(t).To(Equal(types.Type{}))
		})

		DescribeTable("primitive types",
			func(decl string, expectedKind types.Kind) {
				typeCtx := parseTypeFromDecl(decl)
				t := MustSucceed(atypes.InferFromTypeContext(typeCtx))
				Expect(t.Kind).To(Equal(expectedKind))
			},
			// Integer types
			Entry("i8", "x i8 := 0", types.KindI8),
			Entry("i16", "x i16 := 0", types.KindI16),
			Entry("i32", "x i32 := 0", types.KindI32),
			Entry("i64", "x i64 := 0", types.KindI64),
			Entry("u8", "x u8 := 0", types.KindU8),
			Entry("u16", "x u16 := 0", types.KindU16),
			Entry("u32", "x u32 := 0", types.KindU32),
			Entry("u64", "x u64 := 0", types.KindU64),

			// Float types
			Entry("f32", "x f32 := 0", types.KindF32),
			Entry("f64", "x f64 := 0", types.KindF64),

			// String type
			Entry("str", "x str := \"\"", types.KindString),
		)

		DescribeTable("composite types",
			func(decl string, expectedKind types.Kind, expectedElemKind types.Kind) {
				typeCtx := parseTypeFromDecl(decl)
				t := MustSucceed(atypes.InferFromTypeContext(typeCtx))
				Expect(t.Kind).To(Equal(expectedKind))
				Expect(t.Unwrap().Kind).To(Equal(expectedElemKind))
			},
			Entry("chan f32", "x chan f32 := c", types.KindChan, types.KindF32),
			Entry("chan i64", "x chan i64 := c", types.KindChan, types.KindI64),
			Entry("series f32", "x series f32 := s", types.KindSeries, types.KindF32),
			Entry("series i64", "x series i64 := s", types.KindSeries, types.KindI64),
		)

		Context("types with units", func() {
			It("should parse type with unit suffix", func() {
				typeCtx := parseTypeFromDecl("p f32 psi := 0")
				t := MustSucceed(atypes.InferFromTypeContext(typeCtx))
				Expect(t.Kind).To(Equal(types.KindF32))
				Expect(t.Unit.Name).To(Equal("psi"))
			})

			It("should parse channel type with unit suffix", func() {
				typeCtx := parseTypeFromDecl("p chan f32 psi := c")
				t := MustSucceed(atypes.InferFromTypeContext(typeCtx))
				Expect(t.Kind).To(Equal(types.KindChan))
				Expect(t.Unwrap().Unit.Name).To(Equal("psi"))
			})

			It("should parse series type with unit suffix", func() {
				typeCtx := parseTypeFromDecl("t series f64 s := s")
				t := MustSucceed(atypes.InferFromTypeContext(typeCtx))
				Expect(t.Kind).To(Equal(types.KindSeries))
				Expect(t.Unwrap().Unit.Name).To(Equal("s"))
			})

			It("should error on unknown unit", func() {
				typeCtx := parseTypeFromDecl("x f32 unknownunit := 0")
				_, err := atypes.InferFromTypeContext(typeCtx)
				Expect(err).To(MatchError(ContainSubstring("unknown unit")))
			})
		})
	})
})
