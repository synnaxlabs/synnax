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
	"github.com/synnaxlabs/arc/types"
)

var _ = Describe("Type Cast Compilation", func() {
	DescribeTable("should compile type casts correctly",
		expectExpression,

		// Integer to Integer
		Entry(
			"i64 to i32",
			"i32(42)",
			types.I32(),
			OpI64Const,
			int64(42),
			OpI32WrapI64,
		),

		Entry(
			"i32 expression to i64",
			"i64(i32(42))",
			types.I64(),
			OpI64Const,
			int64(42),
			OpI32WrapI64,
			OpI64ExtendI32S,
		),

		// Integer to Float
		Entry(
			"i64 to f32",
			"f32(42)",
			types.F32(),
			OpI64Const,
			int64(42),
			OpF32ConvertI64S,
		),

		Entry(
			"i64 to f64",
			"f64(42)",
			types.F64(),
			OpI64Const,
			int64(42),
			OpF64ConvertI64S,
		),

		// Float to Integer
		Entry(
			"f64 to i32",
			"i32(3.14)",
			types.I32(),
			OpF64Const,
			float64(3.14),
			OpI32TruncF64S,
		),

		Entry(
			"f64 to i64",
			"i64(3.14)",
			types.I64(),
			OpF64Const,
			float64(3.14),
			OpI64TruncF64S,
		),

		// Float to Float
		Entry(
			"f64 to f32",
			"f32(3.14)",
			types.F32(),
			OpF64Const,
			float64(3.14),
			OpF32DemoteF64,
		),

		Entry(
			"f32 expression to f64",
			"f64(f32(3.14))",
			types.F64(),
			OpF64Const,
			float64(3.14),
			OpF32DemoteF64,
			OpF64PromoteF32,
		),

		// Unsigned Types
		Entry(
			"i64 to u32",
			"u32(42)",
			types.U32(),
			OpI64Const,
			int64(42),
			OpI32WrapI64,
		),

		Entry(
			"u32 to f32",
			"f32(u32(42))",
			types.F32(),
			OpI64Const,
			int64(42),
			OpI32WrapI64,
			OpF32ConvertI32U,
		),
	)
})
