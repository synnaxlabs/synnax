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

// Behavioral coverage for the dotted module syntax added in SY-3964
// (time.now, math.avg, etc.). These tests invoke a dotted module from
// within a user function and assert that the call dispatches correctly,
// returns the right type, and produces sane values.
var _ = Describe("Dotted modules", func() {
	lastI64 := func(fr telem.Frame[uint32], key uint32) int64 {
		ch := fr.Get(key)
		Expect(ch.Series).ToNot(BeEmpty(), "channel %d not written", key)
		s := ch.Series[len(ch.Series)-1]
		vals := telem.UnmarshalSeries[int64](s)
		Expect(vals).ToNot(BeEmpty())
		return vals[len(vals)-1]
	}

	// time.now() returns a wall-clock timestamp (telem.Now() → ns since
	// Unix epoch). The dotted dispatch in SY-3964 could regress in ways
	// that are silent: returning 0, returning a stale cached value,
	// returning a different module's value, or failing to produce any
	// output. This test drives two invocations, asserts the first is a
	// plausible post-2017 wall-clock value, and that the second is
	// monotonically non-decreasing within a sane delta.
	It("time.now() dispatches, returns a wall-clock timestamp, and is monotonic", func(ctx SpecContext) {
		resolver := channelSymbols(map[string]channelDef{
			"trig":    {types.U8(), 100},
			"now_out": {types.I64(), 101},
		})
		h := newRuntimeHarness(ctx, `
			func get_now(t u8) i64 {
			    return time.now()
			}
			trig -> get_now{} -> now_out`, resolver,
			channel.Digest{Key: 100, DataType: telem.Uint8T},
			channel.Digest{Key: 101, DataType: telem.Int64T},
		)
		defer h.Close(ctx)

		h.Ingest(100, telem.NewSeriesV[uint8](1))
		h.Tick(ctx, telem.Millisecond)
		h.channelState.ClearReads()
		out, _ := h.Flush()
		t0 := lastI64(out, 101)

		h.Ingest(100, telem.NewSeriesV[uint8](1))
		h.Tick(ctx, 2*telem.Millisecond)
		h.channelState.ClearReads()
		out, _ = h.Flush()
		t1 := lastI64(out, 101)

		// 1.5e18 ns since Unix epoch is roughly 2017-07-14. A healthy
		// time.now() must return a value well past this floor.
		Expect(t0).To(BeNumerically(">", int64(1.5e18)),
			"time.now() should return a realistic wall-clock nanosecond timestamp")
		Expect(t1).To(BeNumerically(">=", t0),
			"time.now() must be monotonically non-decreasing across calls")
		// The two calls happen microseconds apart in test time. A delta
		// over one second would indicate something is very wrong.
		Expect(t1-t0).To(BeNumerically("<", int64(telem.Second)),
			"successive time.now() calls in a test should be close in wall-clock time")
	})
})
