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
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

var _ = Describe("CompileIndex", func() {
	DescribeTable("should not convert an index that fits in an i32",
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
	)

	DescribeTable("should compile a literal index to an i32 constant",
		func(bCtx SpecContext, expr string, index int32) {
			expectSeriesExpression(
				bCtx,
				expr,
				[]symbol.Symbol{seriesSymbol("s", types.F64(), 0)},
				types.F64(),
				OpLocalGet, 0, OpI32Const, index, OpCall, uint32(0),
			)
		},
		Entry("zero", "s[0]", int32(0)),
		Entry("positive", "s[2]", int32(2)),
	)

	It(
		"should compile a negative literal index with no conversion",
		func(bCtx SpecContext) {
			expectSeriesExpression(
				bCtx,
				"s[-1]",
				[]symbol.Symbol{seriesSymbol("s", types.F64(), 0)},
				types.F64(),
				OpLocalGet, 0,
				OpI32Const, int32(1), OpI32Const, int32(-1), OpI32Mul,
				OpCall, uint32(0),
			)
		},
	)

	// The harness registers the i64 index function first and the u64 one second. A
	// module function takes the index after the last import.
	DescribeTable("should narrow a 64-bit index with the module function",
		expectSeriesExpression,
		Entry("i64 index", "s[i]",
			[]symbol.Symbol{
				seriesSymbol("s", types.F64(), 0),
				scalarSymbol("i", types.I64(), 1),
			},
			types.F64(),
			OpLocalGet, 0, OpLocalGet, 1, OpCall, uint32(1), OpCall, uint32(0),
		),
		Entry("u64 index", "s[i]",
			[]symbol.Symbol{
				seriesSymbol("s", types.F64(), 0),
				scalarSymbol("i", types.U64(), 1),
			},
			types.F64(),
			OpLocalGet, 0, OpLocalGet, 1, OpCall, uint32(2), OpCall, uint32(0),
		),
		Entry("u32 index", "s[i]",
			[]symbol.Symbol{
				seriesSymbol("s", types.F64(), 0),
				scalarSymbol("i", types.U32(), 1),
			},
			types.F64(),
			OpLocalGet, 0,
			OpLocalGet, 1, OpI64ExtendI32U, OpCall, uint32(1),
			OpCall, uint32(0),
		),
		Entry("i64 slice bounds", "s[i:i]",
			[]symbol.Symbol{
				seriesSymbol("s", types.F64(), 0),
				scalarSymbol("i", types.I64(), 1),
			},
			types.Series(types.F64()),
			OpLocalGet, 0,
			OpLocalGet, 1, OpCall, uint32(1),
			OpLocalGet, 1, OpCall, uint32(1),
			OpCall, uint32(0),
		),
	)
})
