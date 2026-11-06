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
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/arc/compiler/testutil"
	. "github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

var _ = Describe("Float checks and constants", func() {
	var (
		inf32 = float32(math.Inf(1))
		inf64 = math.Inf(1)
	)
	channels := []symbol.Symbol{
		{Name: "a", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 1},
		{Name: "b", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 2},
	}

	DescribeTable("Should emit a check without a host call",
		func(ctx SpecContext, source string, expected ...any) {
			bytecode, exprType := compileWithAnalyzer(ctx, source, channels)
			Expect(exprType).To(Equal(types.Bool()))
			Expect(bytecode).To(MatchOpcodes(expected...))
		},
		Entry("is_nan f32", "math.is_nan(a)",
			OpI32Const, int32(1), OpCall, uint32(0),
			OpF32Abs, OpF32Const, inf32, OpF32Le, OpI32Eqz),
		Entry("is_nan f64", "math.is_nan(b)",
			OpI32Const, int32(2), OpCall, uint32(0),
			OpF64Abs, OpF64Const, inf64, OpF64Le, OpI32Eqz),
		Entry("is_infinite f32", "math.is_infinite(a)",
			OpI32Const, int32(1), OpCall, uint32(0),
			OpF32Abs, OpF32Const, inf32, OpF32Eq),
		Entry("is_infinite f64", "math.is_infinite(b)",
			OpI32Const, int32(2), OpCall, uint32(0),
			OpF64Abs, OpF64Const, inf64, OpF64Eq),
		Entry("is_finite f32", "math.is_finite(a)",
			OpI32Const, int32(1), OpCall, uint32(0),
			OpF32Abs, OpF32Const, inf32, OpF32Lt),
		Entry("is_finite f64", "math.is_finite(b)",
			OpI32Const, int32(2), OpCall, uint32(0),
			OpF64Abs, OpF64Const, inf64, OpF64Lt),
	)
})
