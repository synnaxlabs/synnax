// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
	. "github.com/synnaxlabs/arc/compiler/testutil"
	. "github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Binary Operations", func() {
	DescribeTable("should compile binary expressions correctly",
		expectExpression,

		// Arithmetic Operations - Addition
		Entry(
			"i32 addition",
			"i32(10) + i32(20)",
			types.I32(),
			OpI32Const,
			int32(10),
			OpI32Const,
			int32(20),
			OpI32Add,
		),

		Entry(
			"i64 addition",
			"100 + 200",
			types.I64(),
			OpI64Const,
			int64(100),
			OpI64Const,
			int64(200),
			OpI64Add,
		),

		Entry(
			"f32 addition",
			"f32(1.5) + f32(2.5)",
			types.F32(),
			OpF32Const,
			float32(1.5),
			OpF32Const,
			float32(2.5),
			OpF32Add,
		),

		Entry(
			"f64 addition",
			"1.5 + 2.5",
			types.F64(),
			OpF64Const,
			float64(1.5),
			OpF64Const,
			float64(2.5),
			OpF64Add,
		),

		Entry(
			"multiple additions (left-associative)",
			"i32(1) + i32(2) + i32(3)",
			types.I32(),
			OpI32Const,
			int32(1),
			OpI32Const,
			int32(2),
			OpI32Add,
			OpI32Const,
			int32(3),
			OpI32Add,
		),

		// Arithmetic Operations - Subtraction
		Entry(
			"i32 subtraction",
			"i32(20) - i32(10)",
			types.I32(),
			OpI32Const,
			int32(20),
			OpI32Const,
			int32(10),
			OpI32Sub,
		),

		Entry(
			"f64 subtraction",
			"5.0 - 2.0",
			types.F64(),
			OpF64Const,
			float64(5.0),
			OpF64Const,
			float64(2.0),
			OpF64Sub,
		),

		// Arithmetic Operations - Multiplication
		Entry(
			"i32 multiplication",
			"i32(3) * i32(4)",
			types.I32(),
			OpI32Const,
			int32(3),
			OpI32Const,
			int32(4),
			OpI32Mul,
		),

		Entry(
			"f64 multiplication",
			"2.5 * 4.0",
			types.F64(),
			OpF64Const,
			float64(2.5),
			OpF64Const,
			float64(4.0),
			OpF64Mul,
		),

		// Arithmetic Operations - Division
		Entry(
			"signed i32 division",
			"i32(20) / i32(4)",
			types.I32(),
			OpI32Const,
			int32(20),
			OpI32Const,
			int32(4),
			OpI32DivS,
		),

		Entry(
			"unsigned u32 division",
			"u32(20) / u32(4)",
			types.U32(),
			OpI32Const,
			int32(20),
			OpI32Const,
			int32(4),
			OpI32DivU,
		),

		Entry(
			"f64 division",
			"10.0 / 2.0",
			types.F64(),
			OpF64Const,
			float64(10.0),
			OpF64Const,
			float64(2.0),
			OpF64Div,
		),

		// Arithmetic Operations - Modulo
		Entry(
			"signed i32 modulo",
			"i32(17) % i32(5)",
			types.I32(),
			OpI32Const,
			int32(17),
			OpI32Const,
			int32(5),
			OpI32RemS,
		),

		Entry(
			"unsigned u32 modulo",
			"u32(17) % u32(5)",
			types.U32(),
			OpI32Const,
			int32(17),
			OpI32Const,
			int32(5),
			OpI32RemU,
		),

		// Operator Precedence
		Entry(
			"multiplication over addition",
			"i32(2) + i32(3) * i32(4)",
			types.I32(),
			OpI32Const,
			int32(2),
			OpI32Const,
			int32(3),
			OpI32Const,
			int32(4),
			OpI32Mul,
			OpI32Add,
		),

		Entry(
			"parentheses precedence",
			"(i32(2) + i32(3)) * i32(4)",
			types.I32(),
			OpI32Const,
			int32(2),
			OpI32Const,
			int32(3),
			OpI32Add,
			OpI32Const,
			int32(4),
			OpI32Mul,
		),

		// Comparison Operations - Equality
		Entry(
			"i32 equality",
			"i32(10) == i32(10)",
			types.U8(),
			OpI32Const,
			int32(10),
			OpI32Const,
			int32(10),
			OpI32Eq,
		),

		Entry(
			"f64 equality",
			"3.14 == 3.14",
			types.U8(),
			OpF64Const,
			float64(3.14),
			OpF64Const,
			float64(3.14),
			OpF64Eq,
		),

		Entry(
			"i32 inequality",
			"i32(10) != i32(20)",
			types.U8(),
			OpI32Const,
			int32(10),
			OpI32Const,
			int32(20),
			OpI32Ne,
		),

		// Comparison Operations - Relational
		Entry(
			"signed i32 less than",
			"i32(5) < i32(10)",
			types.U8(),
			OpI32Const,
			int32(5),
			OpI32Const,
			int32(10),
			OpI32LtS,
		),

		Entry(
			"unsigned u32 less than",
			"u32(5) < u32(10)",
			types.U8(),
			OpI32Const,
			int32(5),
			OpI32Const,
			int32(10),
			OpI32LtU,
		),

		Entry(
			"f64 greater than",
			"5.0 > 2.0",
			types.U8(),
			OpF64Const,
			float64(5.0),
			OpF64Const,
			float64(2.0),
			OpF64Gt,
		),

		Entry(
			"less than or equal",
			"i32(3) <= i32(3)",
			types.U8(),
			OpI32Const,
			int32(3),
			OpI32Const,
			int32(3),
			OpI32LeS,
		),

		Entry(
			"greater than or equal",
			"i32(10) >= i32(5)",
			types.U8(),
			OpI32Const,
			int32(10),
			OpI32Const,
			int32(5),
			OpI32GeS,
		),

		// Comprehensive Type Coverage - u8 (optimized: literals cast directly)
		Entry("u8 addition", "u8(10) + u8(20)", types.U8(), OpI32Const, int32(10), OpI32Const, int32(20), OpI32Add),
		Entry("u8 subtraction", "u8(20) - u8(10)", types.U8(), OpI32Const, int32(20), OpI32Const, int32(10), OpI32Sub),
		Entry("u8 multiplication", "u8(3) * u8(4)", types.U8(), OpI32Const, int32(3), OpI32Const, int32(4), OpI32Mul),
		Entry("u8 division", "u8(20) / u8(4)", types.U8(), OpI32Const, int32(20), OpI32Const, int32(4), OpI32DivU),
		Entry("u8 modulo", "u8(17) % u8(5)", types.U8(), OpI32Const, int32(17), OpI32Const, int32(5), OpI32RemU),
		Entry("u8 equality", "u8(10) == u8(10)", types.U8(), OpI32Const, int32(10), OpI32Const, int32(10), OpI32Eq),
		Entry("u8 inequality", "u8(10) != u8(20)", types.U8(), OpI32Const, int32(10), OpI32Const, int32(20), OpI32Ne),
		Entry("u8 less than", "u8(5) < u8(10)", types.U8(), OpI32Const, int32(5), OpI32Const, int32(10), OpI32LtU),
		Entry("u8 greater than", "u8(10) > u8(5)", types.U8(), OpI32Const, int32(10), OpI32Const, int32(5), OpI32GtU),
		Entry("u8 less than or equal", "u8(5) <= u8(5)", types.U8(), OpI32Const, int32(5), OpI32Const, int32(5), OpI32LeU),
		Entry("u8 greater than or equal", "u8(10) >= u8(5)", types.U8(), OpI32Const, int32(10), OpI32Const, int32(5), OpI32GeU),

		// Comprehensive Type Coverage - u16 (optimized: literals cast directly)
		Entry("u16 addition", "u16(10) + u16(20)", types.U16(), OpI32Const, int32(10), OpI32Const, int32(20), OpI32Add),
		Entry("u16 subtraction", "u16(20) - u16(10)", types.U16(), OpI32Const, int32(20), OpI32Const, int32(10), OpI32Sub),
		Entry("u16 multiplication", "u16(3) * u16(4)", types.U16(), OpI32Const, int32(3), OpI32Const, int32(4), OpI32Mul),
		Entry("u16 division", "u16(20) / u16(4)", types.U16(), OpI32Const, int32(20), OpI32Const, int32(4), OpI32DivU),
		Entry("u16 modulo", "u16(17) % u16(5)", types.U16(), OpI32Const, int32(17), OpI32Const, int32(5), OpI32RemU),
		Entry("u16 equality", "u16(10) == u16(10)", types.U8(), OpI32Const, int32(10), OpI32Const, int32(10), OpI32Eq),
		Entry("u16 inequality", "u16(10) != u16(20)", types.U8(), OpI32Const, int32(10), OpI32Const, int32(20), OpI32Ne),
		Entry("u16 less than", "u16(5) < u16(10)", types.U8(), OpI32Const, int32(5), OpI32Const, int32(10), OpI32LtU),
		Entry("u16 greater than", "u16(10) > u16(5)", types.U8(), OpI32Const, int32(10), OpI32Const, int32(5), OpI32GtU),
		Entry("u16 less than or equal", "u16(5) <= u16(5)", types.U8(), OpI32Const, int32(5), OpI32Const, int32(5), OpI32LeU),
		Entry("u16 greater than or equal", "u16(10) >= u16(5)", types.U8(), OpI32Const, int32(10), OpI32Const, int32(5), OpI32GeU),

		// Comprehensive Type Coverage - u32 (addition, subtraction, multiplication not previously tested)
		Entry("u32 addition", "u32(10) + u32(20)", types.U32(), OpI32Const, int32(10), OpI32Const, int32(20), OpI32Add),
		Entry("u32 subtraction", "u32(20) - u32(10)", types.U32(), OpI32Const, int32(20), OpI32Const, int32(10), OpI32Sub),
		Entry("u32 multiplication", "u32(3) * u32(4)", types.U32(), OpI32Const, int32(3), OpI32Const, int32(4), OpI32Mul),
		Entry("u32 equality", "u32(10) == u32(10)", types.U8(), OpI32Const, int32(10), OpI32Const, int32(10), OpI32Eq),
		Entry("u32 inequality", "u32(10) != u32(20)", types.U8(), OpI32Const, int32(10), OpI32Const, int32(20), OpI32Ne),
		Entry("u32 greater than", "u32(10) > u32(5)", types.U8(), OpI32Const, int32(10), OpI32Const, int32(5), OpI32GtU),
		Entry("u32 less than or equal", "u32(5) <= u32(5)", types.U8(), OpI32Const, int32(5), OpI32Const, int32(5), OpI32LeU),
		Entry("u32 greater than or equal", "u32(10) >= u32(5)", types.U8(), OpI32Const, int32(10), OpI32Const, int32(5), OpI32GeU),

		// Comprehensive Type Coverage - u64
		Entry("u64 addition", "u64(10) + u64(20)", types.U64(), OpI64Const, int64(10), OpI64Const, int64(20), OpI64Add),
		Entry("u64 subtraction", "u64(20) - u64(10)", types.U64(), OpI64Const, int64(20), OpI64Const, int64(10), OpI64Sub),
		Entry("u64 multiplication", "u64(3) * u64(4)", types.U64(), OpI64Const, int64(3), OpI64Const, int64(4), OpI64Mul),
		Entry("u64 division", "u64(20) / u64(4)", types.U64(), OpI64Const, int64(20), OpI64Const, int64(4), OpI64DivU),
		Entry("u64 modulo", "u64(17) % u64(5)", types.U64(), OpI64Const, int64(17), OpI64Const, int64(5), OpI64RemU),
		Entry("u64 equality", "u64(10) == u64(10)", types.U8(), OpI64Const, int64(10), OpI64Const, int64(10), OpI64Eq),
		Entry("u64 inequality", "u64(10) != u64(20)", types.U8(), OpI64Const, int64(10), OpI64Const, int64(20), OpI64Ne),
		Entry("u64 less than", "u64(5) < u64(10)", types.U8(), OpI64Const, int64(5), OpI64Const, int64(10), OpI64LtU),
		Entry("u64 greater than", "u64(10) > u64(5)", types.U8(), OpI64Const, int64(10), OpI64Const, int64(5), OpI64GtU),
		Entry("u64 less than or equal", "u64(5) <= u64(5)", types.U8(), OpI64Const, int64(5), OpI64Const, int64(5), OpI64LeU),
		Entry("u64 greater than or equal", "u64(10) >= u64(5)", types.U8(), OpI64Const, int64(10), OpI64Const, int64(5), OpI64GeU),

		// Comprehensive Type Coverage - i8
		Entry("i8 addition", "i8(10) + i8(20)", types.I8(), OpI32Const, int32(10), OpI32Const, int32(20), OpI32Add),
		Entry("i8 subtraction", "i8(20) - i8(10)", types.I8(), OpI32Const, int32(20), OpI32Const, int32(10), OpI32Sub),
		Entry("i8 multiplication", "i8(3) * i8(4)", types.I8(), OpI32Const, int32(3), OpI32Const, int32(4), OpI32Mul),
		Entry("i8 division", "i8(20) / i8(4)", types.I8(), OpI32Const, int32(20), OpI32Const, int32(4), OpI32DivS),
		Entry("i8 modulo", "i8(17) % i8(5)", types.I8(), OpI32Const, int32(17), OpI32Const, int32(5), OpI32RemS),
		Entry("i8 equality", "i8(10) == i8(10)", types.U8(), OpI32Const, int32(10), OpI32Const, int32(10), OpI32Eq),
		Entry("i8 inequality", "i8(10) != i8(20)", types.U8(), OpI32Const, int32(10), OpI32Const, int32(20), OpI32Ne),
		Entry("i8 less than", "i8(5) < i8(10)", types.U8(), OpI32Const, int32(5), OpI32Const, int32(10), OpI32LtS),
		Entry("i8 greater than", "i8(10) > i8(5)", types.U8(), OpI32Const, int32(10), OpI32Const, int32(5), OpI32GtS),
		Entry("i8 less than or equal", "i8(5) <= i8(5)", types.U8(), OpI32Const, int32(5), OpI32Const, int32(5), OpI32LeS),
		Entry("i8 greater than or equal", "i8(10) >= i8(5)", types.U8(), OpI32Const, int32(10), OpI32Const, int32(5), OpI32GeS),

		// Comprehensive Type Coverage - i16
		Entry("i16 addition", "i16(10) + i16(20)", types.I16(), OpI32Const, int32(10), OpI32Const, int32(20), OpI32Add),
		Entry("i16 subtraction", "i16(20) - i16(10)", types.I16(), OpI32Const, int32(20), OpI32Const, int32(10), OpI32Sub),
		Entry("i16 multiplication", "i16(3) * i16(4)", types.I16(), OpI32Const, int32(3), OpI32Const, int32(4), OpI32Mul),
		Entry("i16 division", "i16(20) / i16(4)", types.I16(), OpI32Const, int32(20), OpI32Const, int32(4), OpI32DivS),
		Entry("i16 modulo", "i16(17) % i16(5)", types.I16(), OpI32Const, int32(17), OpI32Const, int32(5), OpI32RemS),
		Entry("i16 equality", "i16(10) == i16(10)", types.U8(), OpI32Const, int32(10), OpI32Const, int32(10), OpI32Eq),
		Entry("i16 inequality", "i16(10) != i16(20)", types.U8(), OpI32Const, int32(10), OpI32Const, int32(20), OpI32Ne),
		Entry("i16 less than", "i16(5) < i16(10)", types.U8(), OpI32Const, int32(5), OpI32Const, int32(10), OpI32LtS),
		Entry("i16 greater than", "i16(10) > i16(5)", types.U8(), OpI32Const, int32(10), OpI32Const, int32(5), OpI32GtS),
		Entry("i16 less than or equal", "i16(5) <= i16(5)", types.U8(), OpI32Const, int32(5), OpI32Const, int32(5), OpI32LeS),
		Entry("i16 greater than or equal", "i16(10) >= i16(5)", types.U8(), OpI32Const, int32(10), OpI32Const, int32(5), OpI32GeS),

		// Comprehensive Type Coverage - i32 (fill in missing operators)
		Entry("i32 modulo", "i32(17) % i32(5)", types.I32(), OpI32Const, int32(17), OpI32Const, int32(5), OpI32RemS),
		Entry("i32 greater than", "i32(10) > i32(5)", types.U8(), OpI32Const, int32(10), OpI32Const, int32(5), OpI32GtS),

		// Comprehensive Type Coverage - i64
		Entry("i64 subtraction", "100 - 50", types.I64(), OpI64Const, int64(100), OpI64Const, int64(50), OpI64Sub),
		Entry("i64 multiplication", "10 * 20", types.I64(), OpI64Const, int64(10), OpI64Const, int64(20), OpI64Mul),
		Entry("i64 division", "100 / 5", types.I64(), OpI64Const, int64(100), OpI64Const, int64(5), OpI64DivS),
		Entry("i64 modulo", "17 % 5", types.I64(), OpI64Const, int64(17), OpI64Const, int64(5), OpI64RemS),
		Entry("i64 equality", "100 == 100", types.U8(), OpI64Const, int64(100), OpI64Const, int64(100), OpI64Eq),
		Entry("i64 inequality", "100 != 200", types.U8(), OpI64Const, int64(100), OpI64Const, int64(200), OpI64Ne),
		Entry("i64 less than", "50 < 100", types.U8(), OpI64Const, int64(50), OpI64Const, int64(100), OpI64LtS),
		Entry("i64 greater than", "100 > 50", types.U8(), OpI64Const, int64(100), OpI64Const, int64(50), OpI64GtS),
		Entry("i64 less than or equal", "50 <= 50", types.U8(), OpI64Const, int64(50), OpI64Const, int64(50), OpI64LeS),
		Entry("i64 greater than or equal", "100 >= 50", types.U8(), OpI64Const, int64(100), OpI64Const, int64(50), OpI64GeS),

		// Comprehensive Type Coverage - f32 (fill in missing operators)
		Entry("f32 subtraction", "f32(5.0) - f32(2.0)", types.F32(), OpF32Const, float32(5.0), OpF32Const, float32(2.0), OpF32Sub),
		Entry("f32 multiplication", "f32(2.5) * f32(4.0)", types.F32(), OpF32Const, float32(2.5), OpF32Const, float32(4.0), OpF32Mul),
		Entry("f32 division", "f32(10.0) / f32(2.0)", types.F32(), OpF32Const, float32(10.0), OpF32Const, float32(2.0), OpF32Div),
		Entry("f32 equality", "f32(3.14) == f32(3.14)", types.U8(), OpF32Const, float32(3.14), OpF32Const, float32(3.14), OpF32Eq),
		Entry("f32 inequality", "f32(3.14) != f32(2.71)", types.U8(), OpF32Const, float32(3.14), OpF32Const, float32(2.71), OpF32Ne),
		Entry("f32 less than", "f32(2.0) < f32(5.0)", types.U8(), OpF32Const, float32(2.0), OpF32Const, float32(5.0), OpF32Lt),
		Entry("f32 greater than", "f32(5.0) > f32(2.0)", types.U8(), OpF32Const, float32(5.0), OpF32Const, float32(2.0), OpF32Gt),
		Entry("f32 less than or equal", "f32(3.0) <= f32(3.0)", types.U8(), OpF32Const, float32(3.0), OpF32Const, float32(3.0), OpF32Le),
		Entry("f32 greater than or equal", "f32(5.0) >= f32(3.0)", types.U8(), OpF32Const, float32(5.0), OpF32Const, float32(3.0), OpF32Ge),

		// Comprehensive Type Coverage - f64 (fill in missing operators)
		Entry("f64 inequality", "3.14 != 2.71", types.U8(), OpF64Const, float64(3.14), OpF64Const, float64(2.71), OpF64Ne),
		Entry("f64 less than", "2.0 < 5.0", types.U8(), OpF64Const, float64(2.0), OpF64Const, float64(5.0), OpF64Lt),
		Entry("f64 less than or equal", "3.0 <= 3.0", types.U8(), OpF64Const, float64(3.0), OpF64Const, float64(3.0), OpF64Le),
		Entry("f64 multiplication", "2.5 * 3.0", types.F64(), OpF64Const, float64(2.5), OpF64Const, float64(3.0), OpF64Mul),
		Entry("f64 addition", "1.5 + 3.5", types.F64(), OpF64Const, float64(1.5), OpF64Const, float64(3.5), OpF64Add),
		Entry("f64 greater than or equal", "5.0 >= 3.0", types.U8(), OpF64Const, float64(5.0), OpF64Const, float64(3.0), OpF64Ge),

		// Complex Expressions
		Entry(
			"nested arithmetic",
			"(i32(10) + i32(20)) * (i32(30) - i32(10))",
			types.I32(),
			OpI32Const,
			int32(10),
			OpI32Const,
			int32(20),
			OpI32Add,
			OpI32Const,
			int32(30),
			OpI32Const,
			int32(10),
			OpI32Sub,
			OpI32Mul,
		),

		Entry(
			"comparison with arithmetic",
			"(i32(2) + i32(3)) > i32(4)",
			types.U8(),
			OpI32Const,
			int32(2),
			OpI32Const,
			int32(3),
			OpI32Add,
			OpI32Const,
			int32(4),
			OpI32GtS,
		),

		// Power Operations
		Entry(
			"i32 power - simple 2^3",
			"i32(2) ^ i32(3)",
			types.I32(),
			OpI32Const,
			int32(2),
			OpI32Const,
			int32(3),
			OpCall,
			uint32(267), // MathIntPowI32
		),

		Entry(
			"i64 power - 2^10",
			"2 ^ 10",
			types.I64(),
			OpI64Const,
			int64(2),
			OpI64Const,
			int64(10),
			OpCall,
			uint32(268), // MathIntPowI64
		),

		Entry(
			"u32 power - 3^4",
			"u32(3) ^ u32(4)",
			types.U32(),
			OpI32Const,
			int32(3),
			OpI32Const,
			int32(4),
			OpCall,
			uint32(263), // MathIntPowU32
		),

		Entry(
			"u64 power - 5^3",
			"u64(5) ^ u64(3)",
			types.U64(),
			OpI64Const,
			int64(5),
			OpI64Const,
			int64(3),
			OpCall,
			uint32(264), // MathIntPowU64
		),

		Entry(
			"f32 power - 2.0^3.0",
			"f32(2.0) ^ f32(3.0)",
			types.F32(),
			OpF32Const,
			float32(2.0),
			OpF32Const,
			float32(3.0),
			OpCall,
			uint32(259), // MathPowF32
		),

		Entry(
			"f64 power - 2.5^2.0",
			"2.5 ^ 2.0",
			types.F64(),
			OpF64Const,
			float64(2.5),
			OpF64Const,
			float64(2.0),
			OpCall,
			uint32(260), // MathPowF64
		),

		Entry(
			"power right-associative - 2^3^2 = 2^(3^2) = 2^9",
			"i32(2) ^ i32(3) ^ i32(2)",
			types.I32(),
			OpI32Const,
			int32(2),
			OpI32Const,
			int32(3),
			OpI32Const,
			int32(2),
			OpCall,
			uint32(267), // MathIntPowI32 for 3^2
			OpCall,
			uint32(267), // MathIntPowI32 for 2^9
		),

		Entry(
			"power with higher precedence than addition - 2 + 3^2",
			"i32(2) + i32(3) ^ i32(2)",
			types.I32(),
			OpI32Const,
			int32(2),
			OpI32Const,
			int32(3),
			OpI32Const,
			int32(2),
			OpCall,
			uint32(267), // MathIntPowI32 for 3^2
			OpI32Add,    // then add
		),

		Entry(
			"power with parentheses - (2 + 3)^2",
			"(i32(2) + i32(3)) ^ i32(2)",
			types.I32(),
			OpI32Const,
			int32(2),
			OpI32Const,
			int32(3),
			OpI32Add,
			OpI32Const,
			int32(2),
			OpCall,
			uint32(267), // MathIntPowI32
		),
	)

	Describe("Literal Coercion", func() {
		It("Should coerce a literal type", func() {
			ctx := NewContext(bCtx)
			scope := MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{
				Name: "x",
				Kind: symbol.KindVariable,
				Type: types.F32(),
			}))
			Expect(scope).ToNot(BeNil())
			compiled, t := compileWithCtx(ctx, "x + 1")
			Expect(t).To(Equal(types.F32()))
			Expect(compiled).To(MatchOpcodes(
				OpLocalGet, 0,
				OpF32Const,
				float32(1),
				OpF32Add,
			))
		})

		It("Should coerce a literal type", func() {
			bytecode, exprType := compileWithAnalyzer("2 + x", symbol.MapResolver{
				"x": symbol.Symbol{
					Name: "x",
					Kind: symbol.KindVariable,
					Type: types.F64(),
				},
			})
			Expect(exprType).To(Equal(types.F64()))
			Expect(bytecode).To(MatchOpcodes(
				OpF64Const,
				float64(2),
				OpLocalGet, 0,
				OpF64Add,
			))
		})
	})

	Describe("Channel Literal Operations", func() {
		DescribeTable("should correctly infer literal types from channel operations",
			func(expr string, resolver symbol.MapResolver, expectedType types.Type, expectedOpcodes ...any) {
				bytecode, exprType := compileWithAnalyzer(expr, resolver)
				Expect(exprType).To(Equal(expectedType))
				Expect(bytecode).To(MatchOpcodes(expectedOpcodes...))
			},
			Entry(
				"f64 literal * f64 channel",
				"2 * sensor",
				symbol.MapResolver{
					"sensor": symbol.Symbol{
						Name: "sensor",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   0,
					},
				},
				types.F64(),
				OpF64Const,
				float64(2),
				OpI32Const,
				int32(0),
				OpCall,
				uint32(27),
				OpF64Mul,
			),

			Entry(
				"f64 channel * f64 literal",
				"sensor * 2",
				symbol.MapResolver{
					"sensor": symbol.Symbol{
						Name: "sensor",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   0,
					},
				},
				types.F64(),
				OpI32Const,
				int32(0),
				OpCall,
				uint32(27),
				OpF64Const,
				float64(2),
				OpF64Mul,
			),

			Entry(
				"i32 channel + i32 literal",
				"temp + 10",
				symbol.MapResolver{
					"temp": symbol.Symbol{
						Name: "temp",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   0,
					},
				},
				types.I32(),
				OpI32Const,
				int32(0),
				OpCall,
				uint32(18),
				OpI32Const,
				int32(10),
				OpI32Add,
			),

			Entry(
				"i32 literal + i32 channel",
				"10 + temp",
				symbol.MapResolver{
					"temp": symbol.Symbol{
						Name: "temp",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   0,
					},
				},
				types.I32(),
				OpI32Const,
				int32(10),
				OpI32Const,
				int32(0),
				OpCall,
				uint32(18),
				OpI32Add,
			),

			Entry(
				"f64 channel - f64 literal",
				"pressure - 5",
				symbol.MapResolver{
					"pressure": symbol.Symbol{
						Name: "pressure",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   0,
					},
				},
				types.F64(),
				OpI32Const,
				int32(0),
				OpCall,
				uint32(27),
				OpF64Const,
				float64(5),
				OpF64Sub,
			),

			Entry(
				"f64 literal / f64 channel",
				"100 / rate",
				symbol.MapResolver{
					"rate": symbol.Symbol{
						Name: "rate",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   0,
					},
				},
				types.F64(),
				OpF64Const,
				float64(100),
				OpI32Const,
				int32(0),
				OpCall,
				uint32(27),
				OpF64Div,
			),

			Entry(
				"i64 channel % i64 literal",
				"count % 3",
				symbol.MapResolver{
					"count": symbol.Symbol{
						Name: "count",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I64()),
						ID:   0,
					},
				},
				types.I64(),
				OpI32Const,
				int32(0),
				OpCall,
				uint32(21),
				OpI64Const,
				int64(3),
				OpI64RemS,
			),

			Entry(
				"f32 literal * f32 channel",
				"0.5 * velocity",
				symbol.MapResolver{
					"velocity": symbol.Symbol{
						Name: "velocity",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F32()),
						ID:   0,
					},
				},
				types.F32(),
				OpF32Const,
				float32(0.5),
				OpI32Const,
				int32(0),
				OpCall,
				uint32(24),
				OpF32Mul,
			),

			Entry(
				"u32 channel + u32 literal",
				"flags + 1",
				symbol.MapResolver{
					"flags": symbol.Symbol{
						Name: "flags",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.U32()),
						ID:   0,
					},
				},
				types.U32(),
				OpI32Const,
				int32(0),
				OpCall,
				uint32(6),
				OpI32Const,
				int32(1),
				OpI32Add,
			),

			Entry(
				"f64 channel > f64 literal",
				"pressure > 100",
				symbol.MapResolver{
					"pressure": symbol.Symbol{
						Name: "pressure",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   0,
					},
				},
				types.U8(),
				OpI32Const,
				int32(0),
				OpCall,
				uint32(27),
				OpF64Const,
				float64(100),
				OpF64Gt,
			),

			Entry(
				"i32 literal < i32 channel",
				"0 < temp",
				symbol.MapResolver{
					"temp": symbol.Symbol{
						Name: "temp",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   0,
					},
				},
				types.U8(),
				OpI32Const,
				int32(0),
				OpI32Const,
				int32(0),
				OpCall,
				uint32(18),
				OpI32LtS,
			),

			Entry(
				"f64 channel >= f64 literal",
				"altitude >= 1000",
				symbol.MapResolver{
					"altitude": symbol.Symbol{
						Name: "altitude",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   0,
					},
				},
				types.U8(),
				OpI32Const,
				int32(0),
				OpCall,
				uint32(27),
				OpF64Const,
				float64(1000),
				OpF64Ge,
			),

			Entry(
				"f64 channel == f64 literal",
				"value == 3.14",
				symbol.MapResolver{
					"value": symbol.Symbol{
						Name: "value",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   0,
					},
				},
				types.U8(),
				OpI32Const,
				int32(0),
				OpCall,
				uint32(27),
				OpF64Const,
				float64(3.14),
				OpF64Eq,
			),

			Entry(
				"i32 literal != i32 channel",
				"42 != answer",
				symbol.MapResolver{
					"answer": symbol.Symbol{
						Name: "answer",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.I32()),
						ID:   0,
					},
				},
				types.U8(),
				OpI32Const,
				int32(42),
				OpI32Const,
				int32(0),
				OpCall,
				uint32(18),
				OpI32Ne,
			),

			Entry(
				"complex: (literal * channel) + (literal * channel)",
				"2 * a + 3 * b",
				symbol.MapResolver{
					"a": symbol.Symbol{
						Name: "a",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   0,
					},
					"b": symbol.Symbol{
						Name: "b",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   1,
					},
				},
				types.F64(),
				OpF64Const,
				float64(2),
				OpI32Const,
				int32(0),
				OpCall,
				uint32(27),
				OpF64Mul,
				OpF64Const,
				float64(3),
				OpI32Const,
				int32(1),
				OpCall,
				uint32(27),
				OpF64Mul,
				OpF64Add,
			),

			Entry(
				"complex: (channel + literal) > (channel - literal)",
				"a + 5 > b - 3",
				symbol.MapResolver{
					"a": symbol.Symbol{
						Name: "a",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   0,
					},
					"b": symbol.Symbol{
						Name: "b",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   1,
					},
				},
				types.U8(),
				OpI32Const,
				int32(0),
				OpCall,
				uint32(27),
				OpF64Const,
				float64(5),
				OpF64Add,
				OpI32Const,
				int32(1),
				OpCall,
				uint32(27),
				OpF64Const,
				float64(3),
				OpF64Sub,
				OpF64Gt,
			),

			Entry(
				"u64 channel + u64 literal",
				"counter + 1000",
				symbol.MapResolver{
					"counter": symbol.Symbol{
						Name: "counter",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.U64()),
						ID:   0,
					},
				},
				types.U64(),
				OpI32Const,
				int32(0),
				OpCall,
				uint32(9),
				OpI64Const,
				int64(1000),
				OpI64Add,
			),

			Entry(
				"u8 channel - u8 literal",
				"byte - 5",
				symbol.MapResolver{
					"byte": symbol.Symbol{
						Name: "byte",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.U8()),
						ID:   0,
					},
				},
				types.U8(),
				OpI32Const,
				int32(0),
				OpCall,
				uint32(0),
				OpI32Const,
				int32(5),
				OpI32Sub,
			),

			Entry(
				"u16 channel * u16 literal",
				"word * 3",
				symbol.MapResolver{
					"word": symbol.Symbol{
						Name: "word",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.U16()),
						ID:   0,
					},
				},
				types.U16(),
				OpI32Const,
				int32(0),
				OpCall,
				uint32(3),
				OpI32Const,
				int32(3),
				OpI32Mul,
			),

			Entry(
				"complex: channel - literal + channel",
				"x - 10 + y",
				symbol.MapResolver{
					"x": symbol.Symbol{
						Name: "x",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   0,
					},
					"y": symbol.Symbol{
						Name: "y",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   1,
					},
				},
				types.F64(),
				OpI32Const,
				int32(0),
				OpCall,
				uint32(27),
				OpF64Const,
				float64(10),
				OpF64Sub,
				OpI32Const,
				int32(1),
				OpCall,
				uint32(27),
				OpF64Add,
			),

			Entry(
				"complex: literal / channel * literal",
				"100 / rate * 2",
				symbol.MapResolver{
					"rate": symbol.Symbol{
						Name: "rate",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.F64()),
						ID:   0,
					},
				},
				types.F64(),
				OpF64Const,
				float64(100),
				OpI32Const,
				int32(0),
				OpCall,
				uint32(27),
				OpF64Div,
				OpF64Const,
				float64(2),
				OpF64Mul,
			),
		)
	})
})
