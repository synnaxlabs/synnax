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
	acontext "github.com/synnaxlabs/arc/analyzer/context"
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
			Entry("chained power operations", "2 ^ 3 ^ 2", types.KindVariable),
			Entry("unary negation with channel", "-temp_sensor", types.KindF32),
			Entry("double unary negation", "--temp_sensor", types.KindF32),
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

			It("should handle function with no return type", func() {
				testResolver["void_func"] = symbol.Symbol{
					Name: "void_func",
					Kind: symbol.KindVariable,
					Type: types.Function(types.FunctionProperties{
						Inputs: types.Params{{Name: "x", Type: types.I32()}},
					}),
				}
				t := inferExprType(bCtx, testResolver, "void_func(5)")
				Expect(t.Kind).To(Equal(types.KindInvalid))
			})
		})

		Context("series in additive expressions", func() {
			It("should infer series type for series + scalar", func() {
				t := inferExprType(bCtx, testResolver, "data_series + 10")
				Expect(t.Kind).To(Equal(types.KindSeries))
				Expect(t.Unwrap().Kind).To(Equal(types.KindI64))
			})

			It("should infer series type for scalar + series", func() {
				t := inferExprType(bCtx, testResolver, "10 + data_series")
				Expect(t.Kind).To(Equal(types.KindSeries))
				Expect(t.Unwrap().Kind).To(Equal(types.KindI64))
			})

			It("should infer series type for series - scalar", func() {
				t := inferExprType(bCtx, testResolver, "data_series - 5")
				Expect(t.Kind).To(Equal(types.KindSeries))
				Expect(t.Unwrap().Kind).To(Equal(types.KindI64))
			})

			It("should handle incompatible series element types", func() {
				t := inferExprType(bCtx, testResolver, "data_series + float_var")
				Expect(t.Kind).To(Equal(types.KindSeries))
				Expect(t.Unwrap().Kind).To(Equal(types.KindI64))
			})
		})

		Context("series in multiplicative expressions", func() {
			It("should infer series type for series * scalar", func() {
				t := inferExprType(bCtx, testResolver, "data_series * 2")
				Expect(t.Kind).To(Equal(types.KindSeries))
				Expect(t.Unwrap().Kind).To(Equal(types.KindI64))
			})

			It("should handle incompatible series element types", func() {
				t := inferExprType(bCtx, testResolver, "data_series * float_var")
				Expect(t.Kind).To(Equal(types.KindSeries))
				Expect(t.Unwrap().Kind).To(Equal(types.KindI64))
			})

			It("should handle incompatible scalar types", func() {
				testResolver["int_var"] = symbol.Symbol{
					Name: "int_var",
					Kind: symbol.KindVariable,
					Type: types.I32(),
				}
				t := inferExprType(bCtx, testResolver, "float_var * int_var")
				Expect(t.Kind).To(Equal(types.KindF32))
			})
		})

		Context("numeric literals with units", func() {
			It("should infer type variable with unit for integer literal", func() {
				t := inferExprType(bCtx, testResolver, "5psi")
				Expect(t.Kind).To(Equal(types.KindVariable))
				Expect(t.Unit.Name).To(Equal("psi"))
			})

			It("should infer type variable with unit for float literal", func() {
				t := inferExprType(bCtx, testResolver, "3.5s")
				Expect(t.Kind).To(Equal(types.KindVariable))
				Expect(t.Unit.Name).To(Equal("s"))
			})
		})

		Context("edge cases", func() {
			It("should handle division of incompatible types", func() {
				testResolver["f32_series"] = symbol.Symbol{
					Name: "f32_series",
					Kind: symbol.KindVariable,
					Type: types.Series(types.F32()),
				}
				t := inferExprType(bCtx, testResolver, "f32_series / data_series")
				Expect(t.Kind).To(Equal(types.KindSeries))
			})

			It("should handle modulo with incompatible series", func() {
				testResolver["f64_series"] = symbol.Symbol{
					Name: "f64_series",
					Kind: symbol.KindVariable,
					Type: types.Series(types.F64()),
				}
				t := inferExprType(bCtx, testResolver, "data_series % f64_series")
				Expect(t.Kind).To(Equal(types.KindSeries))
			})

			It("should return invalid type for unresolved identifier", func() {
				t := inferExprType(bCtx, testResolver, "undefined_var")
				Expect(t.Kind).To(Equal(types.KindInvalid))
			})

			It("should return invalid type for identifier with invalid type", func() {
				testResolver["invalid_var"] = symbol.Symbol{
					Name: "invalid_var",
					Kind: symbol.KindVariable,
					Type: types.Type{},
				}
				t := inferExprType(bCtx, testResolver, "invalid_var")
				Expect(t.Kind).To(Equal(types.KindInvalid))
			})
		})

		Context("series literal type inference", func() {
			It("should infer empty series as series with type variable element", func() {
				t := inferExprType(bCtx, testResolver, "[]")
				Expect(t.Kind).To(Equal(types.KindSeries))
				Expect(t.Elem).ToNot(BeNil())
				Expect(t.Elem.Kind).To(Equal(types.KindVariable))
			})

			It("should infer single integer literal as series with integer constraint", func() {
				t := inferExprType(bCtx, testResolver, "[42]")
				Expect(t.Kind).To(Equal(types.KindSeries))
				Expect(t.Elem).ToNot(BeNil())
				Expect(t.Elem.Kind).To(Equal(types.KindVariable))
				Expect(t.Elem.Constraint).ToNot(BeNil())
				Expect(t.Elem.Constraint.Kind).To(Equal(types.KindIntegerConstant))
			})

			It("should infer single float literal as series with float constraint", func() {
				t := inferExprType(bCtx, testResolver, "[3.14]")
				Expect(t.Kind).To(Equal(types.KindSeries))
				Expect(t.Elem).ToNot(BeNil())
				Expect(t.Elem.Kind).To(Equal(types.KindVariable))
				Expect(t.Elem.Constraint).ToNot(BeNil())
				Expect(t.Elem.Constraint.Kind).To(Equal(types.KindFloatConstant))
			})

			It("should infer multiple integer literals as series with integer constraint", func() {
				t := inferExprType(bCtx, testResolver, "[1, 2, 3]")
				Expect(t.Kind).To(Equal(types.KindSeries))
				Expect(t.Elem).ToNot(BeNil())
				Expect(t.Elem.Kind).To(Equal(types.KindVariable))
				Expect(t.Elem.Constraint).ToNot(BeNil())
				Expect(t.Elem.Constraint.Kind).To(Equal(types.KindIntegerConstant))
			})

			It("should infer series with typed variable as series of that type", func() {
				testResolver["int_var"] = symbol.Symbol{
					Name: "int_var",
					Kind: symbol.KindVariable,
					Type: types.I32(),
				}
				t := inferExprType(bCtx, testResolver, "[int_var, 1, 2]")
				Expect(t.Kind).To(Equal(types.KindSeries))
				Expect(t.Elem).ToNot(BeNil())
				Expect(t.Elem.Kind).To(Equal(types.KindI32))
			})

			It("should prefer concrete type over type variable in series", func() {
				testResolver["i64_var"] = symbol.Symbol{
					Name: "i64_var",
					Kind: symbol.KindVariable,
					Type: types.I64(),
				}
				t := inferExprType(bCtx, testResolver, "[1, i64_var]")
				Expect(t.Kind).To(Equal(types.KindSeries))
				Expect(t.Elem).ToNot(BeNil())
				Expect(t.Elem.Kind).To(Equal(types.KindI64))
			})

			It("should infer series with expression as series", func() {
				t := inferExprType(bCtx, testResolver, "[1 + 2, 3 * 4]")
				Expect(t.Kind).To(Equal(types.KindSeries))
				Expect(t.Elem).ToNot(BeNil())
			})

			It("should infer series with variable expression", func() {
				testResolver["i32_var"] = symbol.Symbol{
					Name: "i32_var",
					Kind: symbol.KindVariable,
					Type: types.I32(),
				}
				t := inferExprType(bCtx, testResolver, "[i32_var + 1, i32_var * 2]")
				Expect(t.Kind).To(Equal(types.KindSeries))
				Expect(t.Elem).ToNot(BeNil())
				Expect(t.Elem.Kind).To(Equal(types.KindI32))
			})

			It("should infer series with channel references as series of unwrapped value type", func() {
				t := inferExprType(bCtx, testResolver, "[temp_sensor, temp_sensor]")
				Expect(t.Kind).To(Equal(types.KindSeries))
				Expect(t.Elem).ToNot(BeNil())
				Expect(t.Elem.Kind).To(Equal(types.KindF32))
			})

			It("should infer series with channel expression as series of unwrapped type", func() {
				t := inferExprType(bCtx, testResolver, "[temp_sensor + 0, temp_sensor * 1]")
				Expect(t.Kind).To(Equal(types.KindSeries))
				Expect(t.Elem).ToNot(BeNil())
				Expect(t.Elem.Kind).To(Equal(types.KindF32))
			})

			It("should infer series with function call", func() {
				testResolver["get_value"] = symbol.Symbol{
					Name: "get_value",
					Kind: symbol.KindVariable,
					Type: types.Function(types.FunctionProperties{
						Outputs: types.Params{{Type: types.I64()}},
					}),
				}
				t := inferExprType(bCtx, testResolver, "[get_value(), 42]")
				Expect(t.Kind).To(Equal(types.KindSeries))
				Expect(t.Elem).ToNot(BeNil())
				Expect(t.Elem.Kind).To(Equal(types.KindI64))
			})

			It("should infer series with multiple typed variables", func() {
				testResolver["a"] = symbol.Symbol{
					Name: "a",
					Kind: symbol.KindVariable,
					Type: types.F64(),
				}
				testResolver["b"] = symbol.Symbol{
					Name: "b",
					Kind: symbol.KindVariable,
					Type: types.F64(),
				}
				t := inferExprType(bCtx, testResolver, "[a, b, a + b]")
				Expect(t.Kind).To(Equal(types.KindSeries))
				Expect(t.Elem).ToNot(BeNil())
				Expect(t.Elem.Kind).To(Equal(types.KindF64))
			})

			It("should infer series with negated literals", func() {
				t := inferExprType(bCtx, testResolver, "[-1, -2, -3]")
				Expect(t.Kind).To(Equal(types.KindSeries))
				Expect(t.Elem).ToNot(BeNil())
			})

			It("should infer nested series expression correctly", func() {
				t := inferExprType(bCtx, testResolver, "[data_series[0], data_series[1]]")
				Expect(t.Kind).To(Equal(types.KindSeries))
				Expect(t.Elem).ToNot(BeNil())
				Expect(t.Elem.Kind).To(Equal(types.KindI64))
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

	Describe("AssignmentCompatible", func() {
		DescribeTable("assignment compatibility",
			func(varType, exprType types.Type, expected bool) {
				Expect(atypes.AssignmentCompatible(varType, exprType)).To(Equal(expected))
			},
			Entry("f32 with f32", types.F32(), types.F32(), true),
			Entry("i64 with i64", types.I64(), types.I64(), true),
			Entry("series f32 with series f32", types.Series(types.F32()), types.Series(types.F32()), true),
			Entry("chan f32 with chan f32", types.Chan(types.F32()), types.Chan(types.F32()), true),
			Entry("type var to concrete", types.Variable("T", nil), types.I32(), true),
			Entry("concrete to type var", types.F32(), types.Variable("T", nil), true),
			Entry("series to scalar", types.I32(), types.Series(types.I32()), false),
			Entry("scalar to series", types.Series(types.I32()), types.I32(), false),
			Entry("channel to scalar", types.I32(), types.Chan(types.I32()), false),
			Entry("scalar to channel", types.Chan(types.I32()), types.I32(), false),
			Entry("series to channel", types.Chan(types.I32()), types.Series(types.I32()), false),
			Entry("channel to series", types.Series(types.I32()), types.Chan(types.I32()), false),
			Entry("invalid with concrete", types.Type{}, types.I32(), false),
			Entry("concrete with invalid", types.F32(), types.Type{}, false),
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
			Entry("series to scalar", types.I32(), types.Series(types.I32()), false),
			Entry("scalar to series", types.Series(types.I32()), types.I32(), false),
			Entry("channel to scalar", types.I32(), types.Chan(types.I32()), false),
			Entry("scalar to channel", types.Chan(types.I32()), types.I32(), false),
		)
	})

	Describe("InferFromTypeContext", func() {
		It("should return zero type for nil context", func() {
			t := MustSucceed(atypes.InferFromTypeContext(nil))
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

		It("should parse channel with series element type", func() {
			typeCtx := parseTypeFromDecl("x chan series f32 := c")
			t := MustSucceed(atypes.InferFromTypeContext(typeCtx))
			Expect(t.Kind).To(Equal(types.KindChan))
			Expect(t.Elem.Kind).To(Equal(types.KindSeries))
			Expect(t.Elem.Unwrap().Kind).To(Equal(types.KindF32))
		})

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
				Expect(atypes.InferFromTypeContext(typeCtx)).Error().To(
					MatchError(ContainSubstring("unknown unit")),
				)
			})

			It("should error on unknown unit in channel type", func() {
				typeCtx := parseTypeFromDecl("x chan f32 unknownunit := c")
				Expect(atypes.InferFromTypeContext(typeCtx)).Error().To(
					MatchError(ContainSubstring("unknown unit")),
				)
			})

			It("should error on unknown unit in series type", func() {
				typeCtx := parseTypeFromDecl("x series f32 unknownunit := s")
				Expect(atypes.InferFromTypeContext(typeCtx)).Error().To(
					MatchError(ContainSubstring("unknown unit")),
				)
			})
		})
	})
})
