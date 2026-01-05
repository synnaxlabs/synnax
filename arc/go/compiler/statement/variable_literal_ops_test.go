// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package statement_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/arc/compiler/testutil"
	. "github.com/synnaxlabs/arc/compiler/wasm"
)

var _ = Describe("Variable + Literal Operations", func() {
	DescribeTable("should compile variable + literal with correct bytecode for all numeric types",
		func(source string, instructions ...any) {
			Expect(compileBlock(source)).To(MatchOpcodes(instructions...))
		},

		// u8 operations
		Entry("u8 variable + u8 literal",
			`
			x u8 := 10
			y u8 := x + 5
			`,
			OpI32Const, int32(10),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(5),
			OpI32Add,
			OpLocalSet, 1,
		),

		Entry("u8 variable - u8 literal",
			`
			x u8 := 20
			y u8 := x - 5
			`,
			OpI32Const, int32(20),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(5),
			OpI32Sub,
			OpLocalSet, 1,
		),

		Entry("u8 variable * u8 literal",
			`
			x u8 := 3
			y u8 := x * 4
			`,
			OpI32Const, int32(3),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(4),
			OpI32Mul,
			OpLocalSet, 1,
		),

		Entry("u8 variable / u8 literal",
			`
			x u8 := 20
			y u8 := x / 4
			`,
			OpI32Const, int32(20),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(4),
			OpI32DivU,
			OpLocalSet, 1,
		),

		Entry("u8 variable % u8 literal",
			`
			x u8 := 17
			y u8 := x % 5
			`,
			OpI32Const, int32(17),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(5),
			OpI32RemU,
			OpLocalSet, 1,
		),

		// u16 operations
		Entry("u16 variable + u16 literal",
			`
			x u16 := 100
			y u16 := x + 50
			`,
			OpI32Const, int32(100),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(50),
			OpI32Add,
			OpLocalSet, 1,
		),

		Entry("u16 variable / u16 literal",
			`
			x u16 := 200
			y u16 := x / 10
			`,
			OpI32Const, int32(200),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(10),
			OpI32DivU,
			OpLocalSet, 1,
		),

		Entry("u16 variable % u16 literal",
			`
			x u16 := 77
			y u16 := x % 8
			`,
			OpI32Const, int32(77),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(8),
			OpI32RemU,
			OpLocalSet, 1,
		),

		// u32 operations
		Entry("u32 variable + u32 literal",
			`
			x u32 := 1000
			y u32 := x + 500
			`,
			OpI32Const, int32(1000),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(500),
			OpI32Add,
			OpLocalSet, 1,
		),

		Entry("u32 variable / u32 literal",
			`
			x u32 := 1000
			y u32 := x / 10
			`,
			OpI32Const, int32(1000),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(10),
			OpI32DivU,
			OpLocalSet, 1,
		),

		Entry("u32 variable % u32 literal",
			`
			x u32 := 177
			y u32 := x % 15
			`,
			OpI32Const, int32(177),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(15),
			OpI32RemU,
			OpLocalSet, 1,
		),

		// u64 operations
		Entry("u64 variable + u64 literal",
			`
			x u64 := 10000
			y u64 := x + 5000
			`,
			OpI64Const, int64(10000),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI64Const, int64(5000),
			OpI64Add,
			OpLocalSet, 1,
		),

		Entry("u64 variable / u64 literal",
			`
			x u64 := 10000
			y u64 := x / 100
			`,
			OpI64Const, int64(10000),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI64Const, int64(100),
			OpI64DivU,
			OpLocalSet, 1,
		),

		Entry("u64 variable % u64 literal",
			`
			x u64 := 1777
			y u64 := x % 150
			`,
			OpI64Const, int64(1777),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI64Const, int64(150),
			OpI64RemU,
			OpLocalSet, 1,
		),

		// i8 operations
		Entry("i8 variable + i8 literal",
			`
			x i8 := 10
			y i8 := x + 5
			`,
			OpI32Const, int32(10),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(5),
			OpI32Add,
			OpLocalSet, 1,
		),

		Entry("i8 variable - i8 literal",
			`
			x i8 := 20
			y i8 := x - 5
			`,
			OpI32Const, int32(20),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(5),
			OpI32Sub,
			OpLocalSet, 1,
		),

		Entry("i8 variable / i8 literal",
			`
			x i8 := 20
			y i8 := x / 4
			`,
			OpI32Const, int32(20),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(4),
			OpI32DivS,
			OpLocalSet, 1,
		),

		Entry("i8 variable % i8 literal",
			`
			x i8 := 17
			y i8 := x % 5
			`,
			OpI32Const, int32(17),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(5),
			OpI32RemS,
			OpLocalSet, 1,
		),

		// i16 operations
		Entry("i16 variable + i16 literal",
			`
			x i16 := 100
			y i16 := x + 50
			`,
			OpI32Const, int32(100),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(50),
			OpI32Add,
			OpLocalSet, 1,
		),

		Entry("i16 variable / i16 literal",
			`
			x i16 := 200
			y i16 := x / 10
			`,
			OpI32Const, int32(200),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(10),
			OpI32DivS,
			OpLocalSet, 1,
		),

		Entry("i16 variable % i16 literal",
			`
			x i16 := 77
			y i16 := x % 8
			`,
			OpI32Const, int32(77),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(8),
			OpI32RemS,
			OpLocalSet, 1,
		),

		// i32 operations
		Entry("i32 variable + i32 literal",
			`
			x i32 := 1000
			y i32 := x + 500
			`,
			OpI32Const, int32(1000),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(500),
			OpI32Add,
			OpLocalSet, 1,
		),

		Entry("i32 variable / i32 literal",
			`
			x i32 := 1000
			y i32 := x / 10
			`,
			OpI32Const, int32(1000),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(10),
			OpI32DivS,
			OpLocalSet, 1,
		),

		Entry("i32 variable % i32 literal",
			`
			x i32 := 177
			y i32 := x % 15
			`,
			OpI32Const, int32(177),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(15),
			OpI32RemS,
			OpLocalSet, 1,
		),

		// i64 operations
		Entry("i64 variable + i64 literal",
			`
			x i64 := 10000
			y i64 := x + 5000
			`,
			OpI64Const, int64(10000),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI64Const, int64(5000),
			OpI64Add,
			OpLocalSet, 1,
		),

		Entry("i64 variable - i64 literal",
			`
			x i64 := 20000
			y i64 := x - 5000
			`,
			OpI64Const, int64(20000),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI64Const, int64(5000),
			OpI64Sub,
			OpLocalSet, 1,
		),

		Entry("i64 variable / i64 literal",
			`
			x i64 := 10000
			y i64 := x / 100
			`,
			OpI64Const, int64(10000),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI64Const, int64(100),
			OpI64DivS,
			OpLocalSet, 1,
		),

		Entry("i64 variable % i64 literal",
			`
			x i64 := 1777
			y i64 := x % 150
			`,
			OpI64Const, int64(1777),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI64Const, int64(150),
			OpI64RemS,
			OpLocalSet, 1,
		),

		// f32 operations
		Entry("f32 variable + f32 literal",
			`
			x f32 := 10.5
			y f32 := x + 5.5
			`,
			OpF32Const, float32(10.5),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpF32Const, float32(5.5),
			OpF32Add,
			OpLocalSet, 1,
		),

		Entry("f32 variable - f32 literal",
			`
			x f32 := 20.5
			y f32 := x - 5.5
			`,
			OpF32Const, float32(20.5),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpF32Const, float32(5.5),
			OpF32Sub,
			OpLocalSet, 1,
		),

		Entry("f32 variable * f32 literal",
			`
			x f32 := 3.5
			y f32 := x * 2.0
			`,
			OpF32Const, float32(3.5),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpF32Const, float32(2.0),
			OpF32Mul,
			OpLocalSet, 1,
		),

		Entry("f32 variable / f32 literal",
			`
			x f32 := 10.0
			y f32 := x / 2.0
			`,
			OpF32Const, float32(10.0),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpF32Const, float32(2.0),
			OpF32Div,
			OpLocalSet, 1,
		),

		// f64 operations
		Entry("f64 variable + f64 literal",
			`
			x f64 := 100.5
			y f64 := x + 50.5
			`,
			OpF64Const, float64(100.5),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpF64Const, float64(50.5),
			OpF64Add,
			OpLocalSet, 1,
		),

		Entry("f64 variable - f64 literal",
			`
			x f64 := 200.5
			y f64 := x - 50.5
			`,
			OpF64Const, float64(200.5),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpF64Const, float64(50.5),
			OpF64Sub,
			OpLocalSet, 1,
		),

		Entry("f64 variable * f64 literal",
			`
			x f64 := 35.5
			y f64 := x * 2.0
			`,
			OpF64Const, float64(35.5),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpF64Const, float64(2.0),
			OpF64Mul,
			OpLocalSet, 1,
		),

		Entry("f64 variable / f64 literal",
			`
			x f64 := 100.0
			y f64 := x / 2.0
			`,
			OpF64Const, float64(100.0),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpF64Const, float64(2.0),
			OpF64Div,
			OpLocalSet, 1,
		),

		// Comparison operations - unsigned types
		Entry("u8 variable < u8 literal",
			`
			x u8 := 5
			result := x < 10
			`,
			OpI32Const, int32(5),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(10),
			OpI32LtU,
			OpLocalSet, 1,
		),

		Entry("u16 variable > u16 literal",
			`
			x u16 := 50
			result := x > 30
			`,
			OpI32Const, int32(50),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(30),
			OpI32GtU,
			OpLocalSet, 1,
		),

		Entry("u32 variable <= u32 literal",
			`
			x u32 := 100
			result := x <= 100
			`,
			OpI32Const, int32(100),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(100),
			OpI32LeU,
			OpLocalSet, 1,
		),

		Entry("u64 variable >= u64 literal",
			`
			x u64 := 1000
			result := x >= 500
			`,
			OpI64Const, int64(1000),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI64Const, int64(500),
			OpI64GeU,
			OpLocalSet, 1,
		),

		// Comparison operations - signed types
		Entry("i8 variable < i8 literal",
			`
			x i8 := 5
			result := x < 10
			`,
			OpI32Const, int32(5),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(10),
			OpI32LtS,
			OpLocalSet, 1,
		),

		Entry("i16 variable > i16 literal",
			`
			x i16 := 50
			result := x > 30
			`,
			OpI32Const, int32(50),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(30),
			OpI32GtS,
			OpLocalSet, 1,
		),

		Entry("i32 variable <= i32 literal",
			`
			x i32 := 100
			result := x <= 100
			`,
			OpI32Const, int32(100),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(100),
			OpI32LeS,
			OpLocalSet, 1,
		),

		Entry("i64 variable >= i64 literal",
			`
			x i64 := 1000
			result := x >= 500
			`,
			OpI64Const, int64(1000),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI64Const, int64(500),
			OpI64GeS,
			OpLocalSet, 1,
		),

		// Comparison operations - float types
		Entry("f32 variable < f32 literal",
			`
			x f32 := 5.5
			result := x < 10.5
			`,
			OpF32Const, float32(5.5),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpF32Const, float32(10.5),
			OpF32Lt,
			OpLocalSet, 1,
		),

		Entry("f64 variable > f64 literal",
			`
			x f64 := 50.5
			result := x > 30.5
			`,
			OpF64Const, float64(50.5),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpF64Const, float64(30.5),
			OpF64Gt,
			OpLocalSet, 1,
		),

		// Equality operations (work for all types)
		Entry("u8 variable == u8 literal",
			`
			x u8 := 10
			result := x == 10
			`,
			OpI32Const, int32(10),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(10),
			OpI32Eq,
			OpLocalSet, 1,
		),

		Entry("i32 variable != i32 literal",
			`
			x i32 := 100
			result := x != 50
			`,
			OpI32Const, int32(100),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpI32Const, int32(50),
			OpI32Ne,
			OpLocalSet, 1,
		),

		Entry("f64 variable == f64 literal",
			`
			x f64 := 3.14
			result := x == 3.14
			`,
			OpF64Const, float64(3.14),
			OpLocalSet, 0,
			OpLocalGet, 0,
			OpF64Const, float64(3.14),
			OpF64Eq,
			OpLocalSet, 1,
		),

		// Mixed operations - literal on left side
		Entry("u8 literal + u8 variable",
			`
			x u8 := 10
			y u8 := 5 + x
			`,
			OpI32Const, int32(10),
			OpLocalSet, 0,
			OpI32Const, int32(5),
			OpLocalGet, 0,
			OpI32Add,
			OpLocalSet, 1,
		),

		Entry("i32 literal - i32 variable",
			`
			x i32 := 10
			y i32 := 100 - x
			`,
			OpI32Const, int32(10),
			OpLocalSet, 0,
			OpI32Const, int32(100),
			OpLocalGet, 0,
			OpI32Sub,
			OpLocalSet, 1,
		),

		Entry("f32 literal * f32 variable",
			`
			x f32 := 2.5
			y f32 := 3.0 * x
			`,
			OpF32Const, float32(2.5),
			OpLocalSet, 0,
			OpF32Const, float32(3.0),
			OpLocalGet, 0,
			OpF32Mul,
			OpLocalSet, 1,
		),
	)
})
