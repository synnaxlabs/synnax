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
			OpI64Const,
			int64(10),
			OpI32WrapI64,
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
			OpF64Const,
			float64(1.5),
			OpF32DemoteF64,
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
			OpI64Const,
			int64(1),
			OpI32WrapI64,
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
			OpI64Const,
			int64(20),
			OpI32WrapI64,
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
			OpI64Const,
			int64(3),
			OpI32WrapI64,
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
			OpI64Const,
			int64(20),
			OpI32WrapI64,
			OpI32Const,
			int32(4),
			OpI32DivS,
		),

		Entry(
			"unsigned u32 division",
			"u32(20) / u32(4)",
			types.U32(),
			OpI64Const,
			int64(20),
			OpI32WrapI64,
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
			OpI64Const,
			int64(17),
			OpI32WrapI64,
			OpI32Const,
			int32(5),
			OpI32RemS,
		),

		Entry(
			"unsigned u32 modulo",
			"u32(17) % u32(5)",
			types.U32(),
			OpI64Const,
			int64(17),
			OpI32WrapI64,
			OpI32Const,
			int32(5),
			OpI32RemU,
		),

		// Operator Precedence
		Entry(
			"multiplication over addition",
			"i32(2) + i32(3) * i32(4)",
			types.I32(),
			OpI64Const,
			int64(2),
			OpI32WrapI64,
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
			OpI64Const,
			int64(2),
			OpI32WrapI64,
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
			OpI64Const,
			int64(10),
			OpI32WrapI64,
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
			OpI64Const,
			int64(10),
			OpI32WrapI64,
			OpI32Const,
			int32(20),
			OpI32Ne,
		),

		// Comparison Operations - Relational
		Entry(
			"signed i32 less than",
			"i32(5) < i32(10)",
			types.U8(),
			OpI64Const,
			int64(5),
			OpI32WrapI64,
			OpI32Const,
			int32(10),
			OpI32LtS,
		),

		Entry(
			"unsigned u32 less than",
			"u32(5) < u32(10)",
			types.U8(),
			OpI64Const,
			int64(5),
			OpI32WrapI64,
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
			OpI64Const,
			int64(3),
			OpI32WrapI64,
			OpI32Const,
			int32(3),
			OpI32LeS,
		),

		Entry(
			"greater than or equal",
			"i32(10) >= i32(5)",
			types.U8(),
			OpI64Const,
			int64(10),
			OpI32WrapI64,
			OpI32Const,
			int32(5),
			OpI32GeS,
		),

		// Complex Expressions
		Entry(
			"nested arithmetic",
			"(i32(10) + i32(20)) * (i32(30) - i32(10))",
			types.I32(),
			OpI64Const,
			int64(10),
			OpI32WrapI64,
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
			OpI64Const,
			int64(2),
			OpI32WrapI64,
			OpI32Const,
			int32(3),
			OpI32Add,
			OpI32Const,
			int32(4),
			OpI32GtS,
		),
	)

	Describe("Literal Coercion", func() {
		It("Should coerce a literal type", func() {
			ctx := NewContext(bCtx)
			MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{
				Name: "x",
				Kind: symbol.KindVariable,
				Type: types.F32(),
			}))
			compiled, t := compileWithCtx(ctx, "x + 1")
			Expect(t).To(Equal(types.F32()))
			Expect(compiled).To(Equal(WASM(
				OpLocalGet, 0,
				OpF32Const,
				float32(1),
				OpF32Add,
			)))
		})
	})
})
