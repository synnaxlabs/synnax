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

var _ = Describe("Logical Operations", func() {
	DescribeTable("should compile logical expressions correctly",
		expectExpression,

		// Logical AND Operations
		Entry(
			"simple AND of comparisons",
			"(i32(5) > i32(3)) && (i32(10) < i32(20))",
			types.U8(),
			// First comparison: 5 > 3
			OpI64Const, int64(5), OpI32WrapI64,
			OpI32Const, int32(3),
			OpI32GtS,
			// Normalize first operand
			OpI32Const, int32(0), OpI32Ne,
			// Check if zero for short-circuit
			OpI32Eqz,
			OpIf, byte(I32),
			// Was zero: result is 0
			OpI32Const, int32(0),
			OpElse,
			// Was non-zero: evaluate second comparison
			OpI64Const, int64(10), OpI32WrapI64,
			OpI32Const, int32(20),
			OpI32LtS,
			// Normalize result
			OpI32Const, int32(0), OpI32Ne,
			OpEnd,
		),

		Entry(
			"AND with false left operand (short-circuits)",
			"(i32(2) < i32(1)) && (i32(10) / i32(0))",
			types.U8(),
			// First comparison: 2 < 1 (false)
			OpI64Const, int64(2), OpI32WrapI64,
			OpI32Const, int32(1),
			OpI32LtS,
			// Normalize first operand
			OpI32Const, int32(0), OpI32Ne,
			// Check if zero for short-circuit
			OpI32Eqz,
			OpIf, byte(I32),
			// Was zero: result is 0 (short-circuit - division never happens)
			OpI32Const, int32(0),
			OpElse,
			// This branch won't be taken due to short-circuit
			OpI64Const, int64(10), OpI32WrapI64,
			OpI32Const, int32(0),
			OpI32DivS,
			// Normalize result
			OpI32Const, int32(0), OpI32Ne,
			OpEnd,
		),

		Entry(
			"chained AND operations",
			"(i32(1) == i32(1)) && (i32(2) == i32(2)) && (i32(3) == i32(3))",
			types.U8(),
			// First: 1 == 1
			OpI64Const, int64(1), OpI32WrapI64,
			OpI32Const, int32(1),
			OpI32Eq,
			// Normalize
			OpI32Const, int32(0), OpI32Ne,
			// First && check
			OpI32Eqz,
			OpIf, byte(I32),
			OpI32Const, int32(0),
			OpElse,
			// Second: 2 == 2
			OpI64Const, int64(2), OpI32WrapI64,
			OpI32Const, int32(2),
			OpI32Eq,
			// Normalize
			OpI32Const, int32(0), OpI32Ne,
			OpEnd,
			// Second && check
			OpI32Eqz,
			OpIf, byte(I32),
			OpI32Const, int32(0),
			OpElse,
			// Third: 3 == 3
			OpI64Const, int64(3), OpI32WrapI64,
			OpI32Const, int32(3),
			OpI32Eq,
			// Normalize
			OpI32Const, int32(0), OpI32Ne,
			OpEnd,
		),

		// Logical OR Operations
		Entry(
			"simple OR of comparisons",
			"(i32(5) < i32(3)) || (i32(10) < i32(20))",
			types.U8(),
			// First comparison: 5 < 3 (false)
			OpI64Const, int64(5), OpI32WrapI64,
			OpI32Const, int32(3),
			OpI32LtS,
			// Normalize first operand
			OpI32Const, int32(0), OpI32Ne,
			// Check if true for short-circuit
			OpIf, byte(I32),
			// Was true: result is 1
			OpI32Const, int32(1),
			OpElse,
			// Was false: evaluate second comparison
			OpI64Const, int64(10), OpI32WrapI64,
			OpI32Const, int32(20),
			OpI32LtS,
			// Normalize result
			OpI32Const, int32(0), OpI32Ne,
			OpEnd,
		),

		Entry(
			"OR with true left operand (short-circuits)",
			"(i32(5) > i32(3)) || (i32(10) / i32(0))",
			types.U8(),
			// First comparison: 5 > 3 (true)
			OpI64Const, int64(5), OpI32WrapI64,
			OpI32Const, int32(3),
			OpI32GtS,
			// Normalize first operand
			OpI32Const, int32(0), OpI32Ne,
			// Check if true for short-circuit
			OpIf, byte(I32),
			// Was true: result is 1 (short-circuit - division never happens)
			OpI32Const, int32(1),
			OpElse,
			// This branch won't be taken due to short-circuit
			OpI64Const, int64(10), OpI32WrapI64,
			OpI32Const, int32(0),
			OpI32DivS,
			// Normalize result
			OpI32Const, int32(0), OpI32Ne,
			OpEnd,
		),

		Entry(
			"chained OR operations",
			"(i32(1) == i32(0)) || (i32(2) == i32(0)) || (i32(3) == i32(3))",
			types.U8(),
			// First: 1 == 0 (false)
			OpI64Const, int64(1), OpI32WrapI64,
			OpI32Const, int32(0),
			OpI32Eq,
			// Normalize
			OpI32Const, int32(0), OpI32Ne,
			// First || check
			OpIf, byte(I32),
			OpI32Const, int32(1),
			OpElse,
			// Second: 2 == 0 (false)
			OpI64Const, int64(2), OpI32WrapI64,
			OpI32Const, int32(0),
			OpI32Eq,
			// Normalize
			OpI32Const, int32(0), OpI32Ne,
			OpEnd,
			// Second || check
			OpIf, byte(I32),
			OpI32Const, int32(1),
			OpElse,
			// Third: 3 == 3 (true)
			OpI64Const, int64(3), OpI32WrapI64,
			OpI32Const, int32(3),
			OpI32Eq,
			// Normalize
			OpI32Const, int32(0), OpI32Ne,
			OpEnd,
		),

		// Mixed Logical Operations
		Entry(
			"mixed AND and OR",
			"(i32(1) == i32(1)) && ((i32(2) < i32(1)) || (i32(3) > i32(2)))",
			types.U8(),
			// First: 1 == 1
			OpI64Const, int64(1), OpI32WrapI64,
			OpI32Const, int32(1),
			OpI32Eq,
			// Normalize
			OpI32Const, int32(0), OpI32Ne,
			// AND check
			OpI32Eqz,
			OpIf, byte(I32),
			OpI32Const, int32(0),
			OpElse,
			// OR expression: (2 < 1) || (3 > 2)
			// First: 2 < 1
			OpI64Const, int64(2), OpI32WrapI64,
			OpI32Const, int32(1),
			OpI32LtS,
			// Normalize
			OpI32Const, int32(0), OpI32Ne,
			// OR check
			OpIf, byte(I32),
			OpI32Const, int32(1),
			OpElse,
			// Second: 3 > 2
			OpI64Const, int64(3), OpI32WrapI64,
			OpI32Const, int32(2),
			OpI32GtS,
			// Normalize
			OpI32Const, int32(0), OpI32Ne,
			OpEnd,
			// Normalize final result
			OpI32Const, int32(0), OpI32Ne,
			OpEnd,
		),

		// Normalization Tests
		Entry(
			"AND normalizes non-boolean values",
			"i32(42) && i32(100)",
			types.U8(),
			// First: 42 (truthy)
			OpI64Const, int64(42), OpI32WrapI64,
			// Normalize to 1
			OpI32Const, int32(0), OpI32Ne,
			// AND check
			OpI32Eqz,
			OpIf, byte(I32),
			OpI32Const, int32(0),
			OpElse,
			// Second: 100 (truthy)
			OpI64Const, int64(100), OpI32WrapI64,
			// Normalize to 1
			OpI32Const, int32(0), OpI32Ne,
			OpEnd,
		),

		Entry(
			"OR normalizes non-boolean values",
			"i32(0) || i32(42)",
			types.U8(),
			// First: 0 (falsy)
			OpI64Const, int64(0), OpI32WrapI64,
			// Normalize to 0
			OpI32Const, int32(0), OpI32Ne,
			// OR check
			OpIf, byte(I32),
			OpI32Const, int32(1),
			OpElse,
			// Second: 42 (truthy)
			OpI64Const, int64(42), OpI32WrapI64,
			// Normalize to 1
			OpI32Const, int32(0), OpI32Ne,
			OpEnd,
		),
	)
})
