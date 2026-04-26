// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/stl/channel"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

// Behavioral coverage for Arc for-loops inside function bodies (SY-3826).
// These tests run real programs through the full text → IR → WASM pipeline
// and assert on the computed output, catching miscompiles that wouldn't show
// up in compiler unit tests.
var _ = Describe("For loops", func() {
	lastI64 := func(fr telem.Frame[uint32], key uint32) int64 {
		ch := fr.Get(key)
		Expect(ch.Series).ToNot(BeEmpty(), "channel %d not written", key)
		s := ch.Series[len(ch.Series)-1]
		vals := telem.UnmarshalSeries[int64](s)
		Expect(vals).ToNot(BeEmpty())
		return vals[len(vals)-1]
	}

	// SY-3826 fixed a bug in compileAndCast (arc/go/compiler/statement/loop.go)
	// where TypeMap lookup silently failed for variable references in range
	// arguments, causing range(var_i32, var_i64) to crash at WASM runtime.
	// A single variable bound plus a literal-cast bound is enough to exercise
	// the fixed path: the variable lands on the TypeMap-bypass branch that
	// now resolves via expression.Compile's return type.
	It("Handles range() with a variable bound and a wider literal-cast bound", func(ctx SpecContext) {
		resolver := channelSymbols(map[string]channelDef{
			"start":   {types.I32(), 100},
			"sum_out": {types.I64(), 101},
		})
		h := newRuntimeHarness(ctx, `
			func sum_to(n i32) i64 {
			    total i64 := 0
			    for i := range(n, i64(100)) {
			        total = total + i
			    }
			    return total
			}

			start -> sum_to{} -> sum_out`, resolver,
			channel.Digest{Key: 100, DataType: telem.Int32T},
			channel.Digest{Key: 101, DataType: telem.Int64T},
		)
		defer h.Close(ctx)

		h.Ingest(100, telem.NewSeriesV[int32](5))
		h.Tick(ctx, telem.Millisecond)
		h.channelState.ClearReads()

		out, _ := h.Flush()
		// range(5, 100) yields 5, 6, ..., 99 (half-open). Sum = (5+99)*95/2 = 4940.
		Expect(lastI64(out, 101)).To(Equal(int64(4940)),
			"range(n_i32, i64(100)) with n=5 should sum to 4940; "+
				"any regression in compileAndCast's variable-ref path would either "+
				"miscompile the range bounds or WASM-trap at runtime")
	})

	// Series iteration pulls length from a host function that returns i64,
	// then compares against an i32 loop counter. The compiler emits
	// OpI32WrapI64 (loop.go:286) to bridge the types. A regression that
	// drops the wrap would fail WASM validation; a regression in the exit
	// comparison would execute the body on an empty series (out-of-bounds
	// read → WASM trap → sum_out never written). The paired empty/non-empty
	// entries pin both the zero-iteration boundary and the general path.
	DescribeTable("Iterates over local series literals",
		func(ctx SpecContext, source string, expected int64) {
			resolver := channelSymbols(map[string]channelDef{
				"trig":    {types.U8(), 100},
				"sum_out": {types.I64(), 101},
			})
			h := newRuntimeHarness(ctx, source, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Int64T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()

			out, _ := h.Flush()
			Expect(lastI64(out, 101)).To(Equal(expected))
		},
		Entry("empty series yields zero iterations", `
			func compute(t u8) i64 {
			    data series i64 := []
			    total i64 := 0
			    for x := data {
			        total = total + x
			    }
			    return total
			}
			trig -> compute{} -> sum_out`, int64(0)),
		Entry("three-element series sums correctly", `
			func compute(t u8) i64 {
			    data series i64 := [10, 20, 30]
			    total i64 := 0
			    for x := data {
			        total = total + x
			    }
			    return total
			}
			trig -> compute{} -> sum_out`, int64(60)),
	)

	// loop.go:172-191 branches on step sign: positive step uses >= for
	// exit, negative step uses <=. Getting the comparison direction wrong
	// causes either an infinite loop (never exits when it should) or
	// skipping the body entirely (exits when it shouldn't). The three
	// entries cover:
	//   - a normal descending range that iterates multiple times,
	//   - a descending range that is empty from the start (start is
	//     already on the "wrong side" of end for the step direction),
	//   - a single iteration where the step overshoots end by a lot
	//     (tests that the exit check uses i vs end, not |i-start| vs step).
	DescribeTable("Handles descending range with negative step",
		func(ctx SpecContext, source string, expected int64) {
			resolver := channelSymbols(map[string]channelDef{
				"trig":    {types.U8(), 100},
				"sum_out": {types.I64(), 101},
			})
			h := newRuntimeHarness(ctx, source, resolver,
				channel.Digest{Key: 100, DataType: telem.Uint8T},
				channel.Digest{Key: 101, DataType: telem.Int64T},
			)
			defer h.Close(ctx)

			h.Ingest(100, telem.NewSeriesV[uint8](1))
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()

			out, _ := h.Flush()
			Expect(lastI64(out, 101)).To(Equal(expected))
		},
		Entry("range(10, 0, -2) yields 10,8,6,4,2", `
			func compute(t u8) i64 {
			    total i64 := 0
			    for i := range(10, 0, -2) {
			        total = total + i
			    }
			    return total
			}
			trig -> compute{} -> sum_out`, int64(30)),
		Entry("range(0, 10, -1) is empty (start < end with negative step)", `
			func compute(t u8) i64 {
			    total i64 := 0
			    for i := range(0, 10, -1) {
			        total = total + i
			    }
			    return total
			}
			trig -> compute{} -> sum_out`, int64(0)),
		Entry("range(5, 0, -100) runs exactly once even though step overshoots end", `
			func compute(t u8) i64 {
			    total i64 := 0
			    for i := range(5, 0, -100) {
			        total = total + i
			    }
			    return total
			}
			trig -> compute{} -> sum_out`, int64(5)),
	)
})
