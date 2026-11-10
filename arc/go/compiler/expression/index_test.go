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
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

var _ = Describe("CompileIndex", func() {
	DescribeTable("should not convert an index that is i32 in WASM",
		func(bCtx SpecContext, indexType types.Type) {
			expectSeriesExpression(
				bCtx,
				"s[i]",
				[]symbol.Symbol{
					seriesSymbol("s", types.F64(), 0),
					scalarSymbol("i", indexType, 1),
				},
				types.F64(),
				OpLocalGet, 0, OpLocalGet, 1, OpCall, uint32(0),
			)
		},
		Entry("i8", types.I8()),
		Entry("i16", types.I16()),
		Entry("i32", types.I32()),
		Entry("u8", types.U8()),
		Entry("u16", types.U16()),
		Entry("u32", types.U32()),
	)

	DescribeTable("should clamp a 64-bit index to the i32 range",
		func(bCtx SpecContext, indexType types.Type, convert Opcode) {
			expectSeriesExpression(
				bCtx,
				"s[i]",
				[]symbol.Symbol{
					seriesSymbol("s", types.F64(), 0),
					scalarSymbol("i", indexType, 1),
				},
				types.F64(),
				OpLocalGet, 0,
				OpLocalGet, 1,
				convert,
				OpF64Const, float64(math.MaxInt32), OpF64Min,
				OpF64Const, float64(math.MinInt32), OpF64Max,
				OpI32TruncF64S,
				OpCall, uint32(0),
			)
		},
		Entry("i64", types.I64(), OpF64ConvertI64S),
		Entry("u64", types.U64(), OpF64ConvertI64U),
	)

	It("should clamp both 64-bit slice bounds", func(bCtx SpecContext) {
		expectSeriesExpression(
			bCtx,
			"s[i:i]",
			[]symbol.Symbol{
				seriesSymbol("s", types.F64(), 0),
				scalarSymbol("i", types.I64(), 1),
			},
			types.Series(types.F64()),
			OpLocalGet, 0,
			OpLocalGet, 1,
			OpF64ConvertI64S,
			OpF64Const, float64(math.MaxInt32), OpF64Min,
			OpF64Const, float64(math.MinInt32), OpF64Max,
			OpI32TruncF64S,
			OpLocalGet, 1,
			OpF64ConvertI64S,
			OpF64Const, float64(math.MaxInt32), OpF64Min,
			OpF64Const, float64(math.MinInt32), OpF64Max,
			OpI32TruncF64S,
			OpCall, uint32(0),
		)
	})
})
