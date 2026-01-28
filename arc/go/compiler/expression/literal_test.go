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
	. "github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

var _ = Describe("Literal Compilation", func() {
	DescribeTable("should compile literals correctly",
		expectExpression,

		// Integer Literals
		Entry(
			"integer literals as i64",
			"42",
			types.I64(),
			OpI64Const,
			int64(42),
		),

		Entry(
			"zero",
			"0",
			types.I64(),
			OpI64Const,
			int64(0),
		),

		Entry(
			"large integers",
			"1000000",
			types.I64(),
			OpI64Const,
			int64(1000000),
		),

		// Float Literals
		Entry(
			"float literals as f64",
			"3.14",
			types.F64(),
			OpF64Const,
			float64(3.14),
		),

		Entry(
			"simple decimals",
			"2.5",
			types.F64(),
			OpF64Const,
			float64(2.5),
		),

		Entry(
			"float with leading dot",
			".5",
			types.F64(),
			OpF64Const,
			float64(0.5),
		),

		Entry(
			"float with trailing dot",
			"1.",
			types.F64(),
			OpF64Const,
			float64(1.0),
		),

		// Parenthesized Expressions
		Entry(
			"parenthesized integer",
			"(42)",
			types.I64(),
			OpI64Const,
			int64(42),
		),

		Entry(
			"nested parentheses",
			"((42))",
			types.I64(),
			OpI64Const,
			int64(42),
		),

		Entry(
			"parenthesized float",
			"(3.14)",
			types.F64(),
			OpF64Const,
			float64(3.14),
		),

		// Boolean Literals (parsed as identifiers in the grammar)
		Entry(
			"boolean true",
			"true",
			types.U8(),
			OpI32Const,
			int32(1),
		),

		Entry(
			"boolean false",
			"false",
			types.U8(),
			OpI32Const,
			int32(0),
		),

		Entry(
			"parenthesized boolean",
			"(true)",
			types.U8(),
			OpI32Const,
			int32(1),
		),
	)

	Describe("Series Literal Compilation", func() {
		Describe("Empty Series", func() {
			DescribeTable("should compile empty series with type hint",
				expectSeriesLiteralWithHint,

				Entry("empty i8 series", "[]", nil, types.Series(types.I8()),
					OpI32Const, int32(0), OpCall, seriesCreateEmptyIdx(types.I8()),
				),
				Entry("empty i16 series", "[]", nil, types.Series(types.I16()),
					OpI32Const, int32(0), OpCall, seriesCreateEmptyIdx(types.I16()),
				),
				Entry("empty i32 series", "[]", nil, types.Series(types.I32()),
					OpI32Const, int32(0), OpCall, seriesCreateEmptyIdx(types.I32()),
				),
				Entry("empty i64 series", "[]", nil, types.Series(types.I64()),
					OpI32Const, int32(0), OpCall, seriesCreateEmptyIdx(types.I64()),
				),
				Entry("empty u8 series", "[]", nil, types.Series(types.U8()),
					OpI32Const, int32(0), OpCall, seriesCreateEmptyIdx(types.U8()),
				),
				Entry("empty u16 series", "[]", nil, types.Series(types.U16()),
					OpI32Const, int32(0), OpCall, seriesCreateEmptyIdx(types.U16()),
				),
				Entry("empty u32 series", "[]", nil, types.Series(types.U32()),
					OpI32Const, int32(0), OpCall, seriesCreateEmptyIdx(types.U32()),
				),
				Entry("empty u64 series", "[]", nil, types.Series(types.U64()),
					OpI32Const, int32(0), OpCall, seriesCreateEmptyIdx(types.U64()),
				),
				Entry("empty f32 series", "[]", nil, types.Series(types.F32()),
					OpI32Const, int32(0), OpCall, seriesCreateEmptyIdx(types.F32()),
				),
				Entry("empty f64 series", "[]", nil, types.Series(types.F64()),
					OpI32Const, int32(0), OpCall, seriesCreateEmptyIdx(types.F64()),
				),
			)
		})

		Describe("Single Element Series", func() {
			DescribeTable("should compile single element series",
				expectSeriesLiteralWithHint,

				Entry("single i8", "[i8(42)]", nil, types.Series(types.I8()),
					OpI32Const, int32(1), OpCall, seriesCreateEmptyIdx(types.I8()),
					OpI32Const, int32(0), OpI32Const, int32(42), OpCall, seriesSetElementIdx(types.I8()),
				),
				Entry("single i16", "[i16(42)]", nil, types.Series(types.I16()),
					OpI32Const, int32(1), OpCall, seriesCreateEmptyIdx(types.I16()),
					OpI32Const, int32(0), OpI32Const, int32(42), OpCall, seriesSetElementIdx(types.I16()),
				),
				Entry("single i32", "[i32(42)]", nil, types.Series(types.I32()),
					OpI32Const, int32(1), OpCall, seriesCreateEmptyIdx(types.I32()),
					OpI32Const, int32(0), OpI32Const, int32(42), OpCall, seriesSetElementIdx(types.I32()),
				),
				Entry("single i64", "[42]", nil, types.Series(types.I64()),
					OpI32Const, int32(1), OpCall, seriesCreateEmptyIdx(types.I64()),
					OpI32Const, int32(0), OpI64Const, int64(42), OpCall, seriesSetElementIdx(types.I64()),
				),
				Entry("single u8", "[u8(42)]", nil, types.Series(types.U8()),
					OpI32Const, int32(1), OpCall, seriesCreateEmptyIdx(types.U8()),
					OpI32Const, int32(0), OpI32Const, int32(42), OpCall, seriesSetElementIdx(types.U8()),
				),
				Entry("single u16", "[u16(42)]", nil, types.Series(types.U16()),
					OpI32Const, int32(1), OpCall, seriesCreateEmptyIdx(types.U16()),
					OpI32Const, int32(0), OpI32Const, int32(42), OpCall, seriesSetElementIdx(types.U16()),
				),
				Entry("single u32", "[u32(42)]", nil, types.Series(types.U32()),
					OpI32Const, int32(1), OpCall, seriesCreateEmptyIdx(types.U32()),
					OpI32Const, int32(0), OpI32Const, int32(42), OpCall, seriesSetElementIdx(types.U32()),
				),
				Entry("single u64", "[u64(42)]", nil, types.Series(types.U64()),
					OpI32Const, int32(1), OpCall, seriesCreateEmptyIdx(types.U64()),
					OpI32Const, int32(0), OpI64Const, int64(42), OpCall, seriesSetElementIdx(types.U64()),
				),
				Entry("single f32", "[f32(3.14)]", nil, types.Series(types.F32()),
					OpI32Const, int32(1), OpCall, seriesCreateEmptyIdx(types.F32()),
					OpI32Const, int32(0), OpF32Const, float32(3.14), OpCall, seriesSetElementIdx(types.F32()),
				),
				Entry("single f64", "[3.14]", nil, types.Series(types.F64()),
					OpI32Const, int32(1), OpCall, seriesCreateEmptyIdx(types.F64()),
					OpI32Const, int32(0), OpF64Const, float64(3.14), OpCall, seriesSetElementIdx(types.F64()),
				),
			)
		})

		Describe("Multiple Literals", func() {
			DescribeTable("should compile multi-element series",
				expectSeriesLiteralWithHint,

				Entry("three i32 literals", "[i32(1), i32(2), i32(3)]", nil, types.Series(types.I32()),
					OpI32Const, int32(3), OpCall, seriesCreateEmptyIdx(types.I32()),
					OpI32Const, int32(0), OpI32Const, int32(1), OpCall, seriesSetElementIdx(types.I32()),
					OpI32Const, int32(1), OpI32Const, int32(2), OpCall, seriesSetElementIdx(types.I32()),
					OpI32Const, int32(2), OpI32Const, int32(3), OpCall, seriesSetElementIdx(types.I32()),
				),
				Entry("three i64 literals", "[1, 2, 3]", nil, types.Series(types.I64()),
					OpI32Const, int32(3), OpCall, seriesCreateEmptyIdx(types.I64()),
					OpI32Const, int32(0), OpI64Const, int64(1), OpCall, seriesSetElementIdx(types.I64()),
					OpI32Const, int32(1), OpI64Const, int64(2), OpCall, seriesSetElementIdx(types.I64()),
					OpI32Const, int32(2), OpI64Const, int64(3), OpCall, seriesSetElementIdx(types.I64()),
				),
				Entry("three f64 literals", "[1.1, 2.2, 3.3]", nil, types.Series(types.F64()),
					OpI32Const, int32(3), OpCall, seriesCreateEmptyIdx(types.F64()),
					OpI32Const, int32(0), OpF64Const, float64(1.1), OpCall, seriesSetElementIdx(types.F64()),
					OpI32Const, int32(1), OpF64Const, float64(2.2), OpCall, seriesSetElementIdx(types.F64()),
					OpI32Const, int32(2), OpF64Const, float64(3.3), OpCall, seriesSetElementIdx(types.F64()),
				),
				Entry("five u8 literals", "[u8(10), u8(20), u8(30), u8(40), u8(50)]", nil, types.Series(types.U8()),
					OpI32Const, int32(5), OpCall, seriesCreateEmptyIdx(types.U8()),
					OpI32Const, int32(0), OpI32Const, int32(10), OpCall, seriesSetElementIdx(types.U8()),
					OpI32Const, int32(1), OpI32Const, int32(20), OpCall, seriesSetElementIdx(types.U8()),
					OpI32Const, int32(2), OpI32Const, int32(30), OpCall, seriesSetElementIdx(types.U8()),
					OpI32Const, int32(3), OpI32Const, int32(40), OpCall, seriesSetElementIdx(types.U8()),
					OpI32Const, int32(4), OpI32Const, int32(50), OpCall, seriesSetElementIdx(types.U8()),
				),
			)
		})

		Describe("With Variables", func() {
			DescribeTable("should compile series with variable elements",
				expectSeriesLiteralWithHint,

				Entry("single variable", "[x]",
					symbol.MapResolver{
						"x": scalarSymbol("x", types.I32(), 0),
					},
					types.Series(types.I32()),
					OpI32Const, int32(1), OpCall, seriesCreateEmptyIdx(types.I32()),
					OpI32Const, int32(0), OpLocalGet, 0, OpCall, seriesSetElementIdx(types.I32()),
				),
				Entry("two i32 variables", "[x, y]",
					symbol.MapResolver{
						"x": scalarSymbol("x", types.I32(), 0),
						"y": scalarSymbol("y", types.I32(), 1),
					},
					types.Series(types.I32()),
					OpI32Const, int32(2), OpCall, seriesCreateEmptyIdx(types.I32()),
					OpI32Const, int32(0), OpLocalGet, 0, OpCall, seriesSetElementIdx(types.I32()),
					OpI32Const, int32(1), OpLocalGet, 1, OpCall, seriesSetElementIdx(types.I32()),
				),
				Entry("three f64 variables", "[a, b, c]",
					symbol.MapResolver{
						"a": scalarSymbol("a", types.F64(), 0),
						"b": scalarSymbol("b", types.F64(), 1),
						"c": scalarSymbol("c", types.F64(), 2),
					},
					types.Series(types.F64()),
					OpI32Const, int32(3), OpCall, seriesCreateEmptyIdx(types.F64()),
					OpI32Const, int32(0), OpLocalGet, 0, OpCall, seriesSetElementIdx(types.F64()),
					OpI32Const, int32(1), OpLocalGet, 1, OpCall, seriesSetElementIdx(types.F64()),
					OpI32Const, int32(2), OpLocalGet, 2, OpCall, seriesSetElementIdx(types.F64()),
				),
				Entry("u8 variable", "[x]",
					symbol.MapResolver{
						"x": scalarSymbol("x", types.U8(), 0),
					},
					types.Series(types.U8()),
					OpI32Const, int32(1), OpCall, seriesCreateEmptyIdx(types.U8()),
					OpI32Const, int32(0), OpLocalGet, 0, OpCall, seriesSetElementIdx(types.U8()),
				),
				Entry("i64 variable", "[x]",
					symbol.MapResolver{
						"x": scalarSymbol("x", types.I64(), 0),
					},
					types.Series(types.I64()),
					OpI32Const, int32(1), OpCall, seriesCreateEmptyIdx(types.I64()),
					OpI32Const, int32(0), OpLocalGet, 0, OpCall, seriesSetElementIdx(types.I64()),
				),
			)
		})

		Describe("With Binary Expressions", func() {
			DescribeTable("should compile series with binary expression elements",
				expectSeriesLiteralWithHint,

				Entry("x + 1", "[x + i32(1)]",
					symbol.MapResolver{
						"x": scalarSymbol("x", types.I32(), 0),
					},
					types.Series(types.I32()),
					OpI32Const, int32(1), OpCall, seriesCreateEmptyIdx(types.I32()),
					OpI32Const, int32(0),
					OpLocalGet, 0, OpI32Const, int32(1), OpI32Add,
					OpCall, seriesSetElementIdx(types.I32()),
				),
				Entry("x * 2", "[x * i32(2)]",
					symbol.MapResolver{
						"x": scalarSymbol("x", types.I32(), 0),
					},
					types.Series(types.I32()),
					OpI32Const, int32(1), OpCall, seriesCreateEmptyIdx(types.I32()),
					OpI32Const, int32(0),
					OpLocalGet, 0, OpI32Const, int32(2), OpI32Mul,
					OpCall, seriesSetElementIdx(types.I32()),
				),
				Entry("a + b, a - b", "[a + b, a - b]",
					symbol.MapResolver{
						"a": scalarSymbol("a", types.I32(), 0),
						"b": scalarSymbol("b", types.I32(), 1),
					},
					types.Series(types.I32()),
					OpI32Const, int32(2), OpCall, seriesCreateEmptyIdx(types.I32()),
					OpI32Const, int32(0),
					OpLocalGet, 0, OpLocalGet, 1, OpI32Add,
					OpCall, seriesSetElementIdx(types.I32()),
					OpI32Const, int32(1),
					OpLocalGet, 0, OpLocalGet, 1, OpI32Sub,
					OpCall, seriesSetElementIdx(types.I32()),
				),
				Entry("f64 multiplication", "[x * 2.0]",
					symbol.MapResolver{
						"x": scalarSymbol("x", types.F64(), 0),
					},
					types.Series(types.F64()),
					OpI32Const, int32(1), OpCall, seriesCreateEmptyIdx(types.F64()),
					OpI32Const, int32(0),
					OpLocalGet, 0, OpF64Const, float64(2.0), OpF64Mul,
					OpCall, seriesSetElementIdx(types.F64()),
				),
				Entry("f64 division", "[x / 2.0]",
					symbol.MapResolver{
						"x": scalarSymbol("x", types.F64(), 0),
					},
					types.Series(types.F64()),
					OpI32Const, int32(1), OpCall, seriesCreateEmptyIdx(types.F64()),
					OpI32Const, int32(0),
					OpLocalGet, 0, OpF64Const, float64(2.0), OpF64Div,
					OpCall, seriesSetElementIdx(types.F64()),
				),
			)
		})

		Describe("With Function Calls", func() {
			DescribeTable("should compile series with function call elements",
				expectSeriesWithFunctions,

				Entry("single function call", "[getI32()]",
					map[string]uint32{"getI32": 5},
					[]symbol.Symbol{{
						Name: "getI32",
						Kind: symbol.KindFunction,
						Type: types.Function(types.FunctionProperties{
							Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I32()}},
						}),
					}},
					types.Series(types.I32()),
					OpI32Const, int32(1), OpCall, seriesCreateEmptyIdx(types.I32()),
					OpI32Const, int32(0), OpCall, uint32(5), OpCall, seriesSetElementIdx(types.I32()),
				),
				Entry("two function calls", "[getI32(), getI32()]",
					map[string]uint32{"getI32": 5},
					[]symbol.Symbol{{
						Name: "getI32",
						Kind: symbol.KindFunction,
						Type: types.Function(types.FunctionProperties{
							Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I32()}},
						}),
					}},
					types.Series(types.I32()),
					OpI32Const, int32(2), OpCall, seriesCreateEmptyIdx(types.I32()),
					OpI32Const, int32(0), OpCall, uint32(5), OpCall, seriesSetElementIdx(types.I32()),
					OpI32Const, int32(1), OpCall, uint32(5), OpCall, seriesSetElementIdx(types.I32()),
				),
				Entry("function call returning f64", "[getF64()]",
					map[string]uint32{"getF64": 7},
					[]symbol.Symbol{{
						Name: "getF64",
						Kind: symbol.KindFunction,
						Type: types.Function(types.FunctionProperties{
							Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.F64()}},
						}),
					}},
					types.Series(types.F64()),
					OpI32Const, int32(1), OpCall, seriesCreateEmptyIdx(types.F64()),
					OpI32Const, int32(0), OpCall, uint32(7), OpCall, seriesSetElementIdx(types.F64()),
				),
				Entry("function call with argument", "[add(1, 2)]",
					map[string]uint32{"add": 3},
					[]symbol.Symbol{{
						Name: "add",
						Kind: symbol.KindFunction,
						Type: types.Function(types.FunctionProperties{
							Inputs:  types.Params{{Name: "a", Type: types.I32()}, {Name: "b", Type: types.I32()}},
							Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I32()}},
						}),
					}},
					types.Series(types.I32()),
					OpI32Const, int32(1), OpCall, seriesCreateEmptyIdx(types.I32()),
					OpI32Const, int32(0),
					OpI32Const, int32(1), OpI32Const, int32(2), OpCall, uint32(3),
					OpCall, seriesSetElementIdx(types.I32()),
				),
				Entry("literal then function call", "[i32(42), getI32()]",
					map[string]uint32{"getI32": 5},
					[]symbol.Symbol{{
						Name: "getI32",
						Kind: symbol.KindFunction,
						Type: types.Function(types.FunctionProperties{
							Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I32()}},
						}),
					}},
					types.Series(types.I32()),
					OpI32Const, int32(2), OpCall, seriesCreateEmptyIdx(types.I32()),
					OpI32Const, int32(0), OpI32Const, int32(42), OpCall, seriesSetElementIdx(types.I32()),
					OpI32Const, int32(1), OpCall, uint32(5), OpCall, seriesSetElementIdx(types.I32()),
				),
				Entry("function call then literal", "[getI32(), i32(42)]",
					map[string]uint32{"getI32": 5},
					[]symbol.Symbol{{
						Name: "getI32",
						Kind: symbol.KindFunction,
						Type: types.Function(types.FunctionProperties{
							Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I32()}},
						}),
					}},
					types.Series(types.I32()),
					OpI32Const, int32(2), OpCall, seriesCreateEmptyIdx(types.I32()),
					OpI32Const, int32(0), OpCall, uint32(5), OpCall, seriesSetElementIdx(types.I32()),
					OpI32Const, int32(1), OpI32Const, int32(42), OpCall, seriesSetElementIdx(types.I32()),
				),
				Entry("function call in expression", "[getI32() + i32(1)]",
					map[string]uint32{"getI32": 5},
					[]symbol.Symbol{{
						Name: "getI32",
						Kind: symbol.KindFunction,
						Type: types.Function(types.FunctionProperties{
							Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I32()}},
						}),
					}},
					types.Series(types.I32()),
					OpI32Const, int32(1), OpCall, seriesCreateEmptyIdx(types.I32()),
					OpI32Const, int32(0),
					OpCall, uint32(5), OpI32Const, int32(1), OpI32Add,
					OpCall, seriesSetElementIdx(types.I32()),
				),
			)
		})

		Describe("With Unary Expressions", func() {
			DescribeTable("should compile series with unary expression elements",
				expectSeriesLiteralWithHint,

				Entry("negated i32 variable", "[-x]",
					symbol.MapResolver{
						"x": scalarSymbol("x", types.I32(), 0),
					},
					types.Series(types.I32()),
					OpI32Const, int32(1), OpCall, seriesCreateEmptyIdx(types.I32()),
					OpI32Const, int32(0),
					OpLocalGet, 0, OpI32Const, int32(-1), OpI32Mul,
					OpCall, seriesSetElementIdx(types.I32()),
				),
				Entry("negated f64 variable", "[-x]",
					symbol.MapResolver{
						"x": scalarSymbol("x", types.F64(), 0),
					},
					types.Series(types.F64()),
					OpI32Const, int32(1), OpCall, seriesCreateEmptyIdx(types.F64()),
					OpI32Const, int32(0),
					OpLocalGet, 0, OpF64Neg,
					OpCall, seriesSetElementIdx(types.F64()),
				),
				Entry("multiple negated f64", "[-x, -y]",
					symbol.MapResolver{
						"x": scalarSymbol("x", types.F64(), 0),
						"y": scalarSymbol("y", types.F64(), 1),
					},
					types.Series(types.F64()),
					OpI32Const, int32(2), OpCall, seriesCreateEmptyIdx(types.F64()),
					OpI32Const, int32(0),
					OpLocalGet, 0, OpF64Neg,
					OpCall, seriesSetElementIdx(types.F64()),
					OpI32Const, int32(1),
					OpLocalGet, 1, OpF64Neg,
					OpCall, seriesSetElementIdx(types.F64()),
				),
				Entry("negated i64 variable", "[-x]",
					symbol.MapResolver{
						"x": scalarSymbol("x", types.I64(), 0),
					},
					types.Series(types.I64()),
					OpI32Const, int32(1), OpCall, seriesCreateEmptyIdx(types.I64()),
					OpI32Const, int32(0),
					OpLocalGet, 0, OpI64Const, int64(-1), OpI64Mul,
					OpCall, seriesSetElementIdx(types.I64()),
				),
			)
		})

		Describe("Mixed Elements", func() {
			DescribeTable("should compile series with mixed element types",
				expectSeriesLiteralWithHint,

				Entry("literal and variable", "[i32(1), x]",
					symbol.MapResolver{
						"x": scalarSymbol("x", types.I32(), 0),
					},
					types.Series(types.I32()),
					OpI32Const, int32(2), OpCall, seriesCreateEmptyIdx(types.I32()),
					OpI32Const, int32(0), OpI32Const, int32(1), OpCall, seriesSetElementIdx(types.I32()),
					OpI32Const, int32(1), OpLocalGet, 0, OpCall, seriesSetElementIdx(types.I32()),
				),
				Entry("variable and literal", "[x, i32(2)]",
					symbol.MapResolver{
						"x": scalarSymbol("x", types.I32(), 0),
					},
					types.Series(types.I32()),
					OpI32Const, int32(2), OpCall, seriesCreateEmptyIdx(types.I32()),
					OpI32Const, int32(0), OpLocalGet, 0, OpCall, seriesSetElementIdx(types.I32()),
					OpI32Const, int32(1), OpI32Const, int32(2), OpCall, seriesSetElementIdx(types.I32()),
				),
				Entry("literal, variable, expression", "[i32(1), x, x + i32(1)]",
					symbol.MapResolver{
						"x": scalarSymbol("x", types.I32(), 0),
					},
					types.Series(types.I32()),
					OpI32Const, int32(3), OpCall, seriesCreateEmptyIdx(types.I32()),
					OpI32Const, int32(0), OpI32Const, int32(1), OpCall, seriesSetElementIdx(types.I32()),
					OpI32Const, int32(1), OpLocalGet, 0, OpCall, seriesSetElementIdx(types.I32()),
					OpI32Const, int32(2),
					OpLocalGet, 0, OpI32Const, int32(1), OpI32Add,
					OpCall, seriesSetElementIdx(types.I32()),
				),
				Entry("f64 literal and variable", "[1.5, x]",
					symbol.MapResolver{
						"x": scalarSymbol("x", types.F64(), 0),
					},
					types.Series(types.F64()),
					OpI32Const, int32(2), OpCall, seriesCreateEmptyIdx(types.F64()),
					OpI32Const, int32(0), OpF64Const, float64(1.5), OpCall, seriesSetElementIdx(types.F64()),
					OpI32Const, int32(1), OpLocalGet, 0, OpCall, seriesSetElementIdx(types.F64()),
				),
				Entry("variable and expression", "[x, y * i32(2)]",
					symbol.MapResolver{
						"x": scalarSymbol("x", types.I32(), 0),
						"y": scalarSymbol("y", types.I32(), 1),
					},
					types.Series(types.I32()),
					OpI32Const, int32(2), OpCall, seriesCreateEmptyIdx(types.I32()),
					OpI32Const, int32(0), OpLocalGet, 0, OpCall, seriesSetElementIdx(types.I32()),
					OpI32Const, int32(1),
					OpLocalGet, 1, OpI32Const, int32(2), OpI32Mul,
					OpCall, seriesSetElementIdx(types.I32()),
				),
			)
		})

		Describe("Type Coercion", func() {
			DescribeTable("should compile series with type coercion",
				expectSeriesLiteralWithHint,

				Entry("int literals in f64 series", "[1, 2, 3]", nil, types.Series(types.F64()),
					OpI32Const, int32(3), OpCall, seriesCreateEmptyIdx(types.F64()),
					OpI32Const, int32(0), OpF64Const, float64(1), OpCall, seriesSetElementIdx(types.F64()),
					OpI32Const, int32(1), OpF64Const, float64(2), OpCall, seriesSetElementIdx(types.F64()),
					OpI32Const, int32(2), OpF64Const, float64(3), OpCall, seriesSetElementIdx(types.F64()),
				),
				Entry("int literals in f32 series", "[1, 2]", nil, types.Series(types.F32()),
					OpI32Const, int32(2), OpCall, seriesCreateEmptyIdx(types.F32()),
					OpI32Const, int32(0), OpF32Const, float32(1), OpCall, seriesSetElementIdx(types.F32()),
					OpI32Const, int32(1), OpF32Const, float32(2), OpCall, seriesSetElementIdx(types.F32()),
				),
				Entry("mixed int and float literals in f64", "[1, 2.5]", nil, types.Series(types.F64()),
					OpI32Const, int32(2), OpCall, seriesCreateEmptyIdx(types.F64()),
					OpI32Const, int32(0), OpF64Const, float64(1), OpCall, seriesSetElementIdx(types.F64()),
					OpI32Const, int32(1), OpF64Const, float64(2.5), OpCall, seriesSetElementIdx(types.F64()),
				),
			)
		})
	})
})
