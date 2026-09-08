// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package series_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/synnaxlabs/arc/stl/series"
	"github.com/synnaxlabs/arc/stl/testutil"
	"github.com/synnaxlabs/x/telem"
)

// benchSizes are the input series lengths each logical op is measured at.
var benchSizes = []int{1, 1024}

// boolSeries returns a bool series of the given length with alternating values.
func boolSeries(size int) telem.Series {
	vals := make([]bool, size)
	for i := range vals {
		vals[i] = i%2 == 0
	}
	return telem.NewSeries(vals)
}

// benchLogical measures one logical host func through the WASM ABI. Each iteration
// clears the handle store and re-stores the inputs, matching the per-cycle
// store/clear pattern of the runtime.
func benchLogical(
	b *testing.B,
	fn string,
	args func(ss *series.ProgramState, lhs, rhs telem.Series) []uint64,
) {
	for _, size := range benchSizes {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			ctx := context.Background()
			rt := testutil.NewRuntime(ctx)
			defer func() {
				if err := rt.Close(ctx); err != nil {
					b.Errorf("close runtime: %v", err)
				}
			}()
			ss := series.NewProgramState()
			if _, err := series.NewHost(ctx, rt.Underlying(), ss); err != nil {
				b.Fatalf("instantiate host: %v", err)
			}
			rt.Passthrough(ctx, "series")
			lhs, rhs := boolSeries(size), boolSeries(size)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				ss.Clear()
				rt.Call(ctx, "series", fn, args(ss, lhs, rhs)...)
			}
		})
	}
}

// binaryArgs stores both inputs and passes their handles.
func binaryArgs(ss *series.ProgramState, lhs, rhs telem.Series) []uint64 {
	return []uint64{testutil.U32(ss.Store(lhs)), testutil.U32(ss.Store(rhs))}
}

// scalarArgs stores the series input and passes a true scalar.
func scalarArgs(ss *series.ProgramState, lhs, _ telem.Series) []uint64 {
	return []uint64{testutil.U32(ss.Store(lhs)), testutil.U32(1)}
}

// unaryArgs stores the single series input.
func unaryArgs(ss *series.ProgramState, lhs, _ telem.Series) []uint64 {
	return []uint64{testutil.U32(ss.Store(lhs))}
}

func BenchmarkAnd(b *testing.B)       { benchLogical(b, "and", binaryArgs) }
func BenchmarkOr(b *testing.B)        { benchLogical(b, "or", binaryArgs) }
func BenchmarkAndScalar(b *testing.B) { benchLogical(b, "and_scalar", scalarArgs) }
func BenchmarkOrScalar(b *testing.B)  { benchLogical(b, "or_scalar", scalarArgs) }
func BenchmarkNot(b *testing.B)       { benchLogical(b, "not", unaryArgs) }
