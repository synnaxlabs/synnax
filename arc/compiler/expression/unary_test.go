// Copyright 2025 Synnax Labs, Inc.
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
)

var _ = Describe("Unary Operations", func() {
	DescribeTable("should compile unary expressions correctly",
		expectExpression,

		// Unary Minus - Integer negation
		Entry(
			"i32 negation",
			"-i32(10)",
			ir.I32{},
			OpI64Const,
			int64(10),
			OpI32WrapI64,
			OpI32Const,
			int32(-1),
			OpI32Mul,
		),

		Entry(
			"i64 negation",
			"-100",
			ir.I64{},
			OpI64Const,
			int64(100),
			OpI64Const,
			int64(-1),
			OpI64Mul,
		),

		Entry(
			"u32 negation",
			"-u32(5)",
			ir.U32{},
			OpI64Const,
			int64(5),
			OpI32WrapI64,
			OpI32Const,
			int32(-1),
			OpI32Mul,
		),

		// Unary Minus - Float negation
		Entry(
			"f32 negation",
			"-f32(3.14)",
			ir.F32{},
			OpF64Const,
			float64(3.14),
			OpF32DemoteF64,
			OpF32Neg,
		),

		Entry(
			"f64 negation",
			"-2.5",
			ir.F64{},
			OpF64Const,
			float64(2.5),
			OpF64Neg,
		),

		// Unary Minus - Complex negation
		Entry(
			"double negation",
			"--i32(42)",
			ir.I32{},
			OpI64Const,
			int64(42),
			OpI32WrapI64,
			OpI32Const,
			int32(-1),
			OpI32Mul,
			OpI32Const,
			int32(-1),
			OpI32Mul,
		),

		Entry(
			"negation in expression",
			"i32(10) + -i32(5)",
			ir.I32{},
			OpI64Const,
			int64(10),
			OpI32WrapI64,
			OpI64Const,
			int64(5),
			OpI32WrapI64,
			OpI32Const,
			int32(-1),
			OpI32Mul,
			OpI32Add,
		),

		Entry(
			"negated parenthesized expression",
			"-(i32(10) + i32(5))",
			ir.I32{},
			OpI64Const,
			int64(10),
			OpI32WrapI64,
			OpI64Const,
			int64(5),
			OpI32WrapI64,
			OpI32Add,
			OpI32Const,
			int32(-1),
			OpI32Mul,
		),

		// Logical NOT - Boolean negation
		Entry(
			"NOT of comparison",
			"!(i32(5) > i32(3))",
			ir.U8{},
			OpI64Const,
			int64(5),
			OpI32WrapI64,
			OpI64Const,
			int64(3),
			OpI32WrapI64,
			OpI32GtS,
			OpI32Eqz,
		),

		Entry(
			"NOT of equality",
			"!(i32(10) == i32(10))",
			ir.U8{},
			OpI64Const,
			int64(10),
			OpI32WrapI64,
			OpI64Const,
			int64(10),
			OpI32WrapI64,
			OpI32Eq,
			OpI32Eqz,
		),

		Entry(
			"double NOT",
			"!!(i32(5) < i32(10))",
			ir.U8{},
			OpI64Const,
			int64(5),
			OpI32WrapI64,
			OpI64Const,
			int64(10),
			OpI32WrapI64,
			OpI32LtS,
			OpI32Eqz,
			OpI32Eqz,
		),

		// NOT with other operations
		Entry(
			"NOT with arithmetic comparison",
			"!((i32(2) + i32(3)) > i32(4))",
			ir.U8{},
			OpI64Const,
			int64(2),
			OpI32WrapI64,
			OpI64Const,
			int64(3),
			OpI32WrapI64,
			OpI32Add,
			OpI64Const,
			int64(4),
			OpI32WrapI64,
			OpI32GtS,
			OpI32Eqz,
		),

		// Mixed Unary Operations
		Entry(
			"negation and NOT in same expression",
			"!(-i32(5) < i32(0))",
			ir.U8{},
			OpI64Const,
			int64(5),
			OpI32WrapI64,
			OpI32Const,
			int32(-1),
			OpI32Mul,
			OpI64Const,
			int64(0),
			OpI32WrapI64,
			OpI32LtS,
			OpI32Eqz,
		),

		Entry(
			"operator precedence with unary",
			"-i32(2) * i32(3)",
			ir.I32{},
			OpI64Const,
			int64(2),
			OpI32WrapI64,
			OpI32Const,
			int32(-1),
			OpI32Mul,
			OpI64Const,
			int64(3),
			OpI32WrapI64,
			OpI32Mul,
		),
	)
})
