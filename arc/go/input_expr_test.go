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
	"github.com/synnaxlabs/arc/stl/channels"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("expression brace inputs end-to-end", func() {
	lastI64 := func(fr telem.Frame[uint32], key uint32) int64 {
		ch := fr.Get(key)
		Expect(ch.Series).ToNot(BeEmpty(), "channel %d not written", key)
		vals := telem.UnmarshalSeries[int64](ch.Series[len(ch.Series)-1])
		Expect(vals).ToNot(BeEmpty())
		return vals[len(vals)-1]
	}

	It("compiles a non-literal brace input and feeds the computed value", func(ctx SpecContext) {
		resolver := channelSymbols(map[string]channelDef{
			"trig": {types.U8(), 100},
			"out":  {types.I64(), 101},
		})
		h := newRuntimeHarness(ctx,
			`func emit{val i64} () i64 { return val }
			trig -> emit{val = (40 + 2)} -> out`, resolver,
			channels.Digest{Key: 100, DataType: telem.Uint8T},
			channels.Digest{Key: 101, DataType: telem.Int64T},
		)
		defer h.Close(ctx)
		h.Ingest(100, telem.NewSeriesV[uint8](1))
		for range 5 {
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()
		}
		out, _ := h.Flush()
		Expect(lastI64(out, 101)).To(Equal(int64(42)))
	})

	It("allows the brace expression to call a function with arguments", func(ctx SpecContext) {
		resolver := channelSymbols(map[string]channelDef{
			"trig": {types.U8(), 100},
			"out":  {types.I64(), 101},
		})
		h := newRuntimeHarness(ctx,
			`func double(x i64) i64 { return x * 2 }
			func emit{val i64} () i64 { return val }
			trig -> emit{val = double(21)} -> out`, resolver,
			channels.Digest{Key: 100, DataType: telem.Uint8T},
			channels.Digest{Key: 101, DataType: telem.Int64T},
		)
		defer h.Close(ctx)
		h.Ingest(100, telem.NewSeriesV[uint8](1))
		for range 5 {
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()
		}
		out, _ := h.Flush()
		Expect(lastI64(out, 101)).To(Equal(int64(42)))
	})

	It("samples a channel read in a brace expression at the trigger's fire time", func(ctx SpecContext) {
		resolver := channelSymbols(map[string]channelDef{
			"trig": {types.U8(), 100},
			"v":    {types.I64(), 102},
			"out":  {types.I64(), 101},
		})
		h := newRuntimeHarness(ctx,
			`func emit{val i64} () i64 { return val }
			trig -> emit{val = v + 1} -> out`, resolver,
			channels.Digest{Key: 100, DataType: telem.Uint8T},
			channels.Digest{Key: 102, DataType: telem.Int64T},
			channels.Digest{Key: 101, DataType: telem.Int64T},
		)
		defer h.Close(ctx)
		h.Ingest(102, telem.NewSeriesV[int64](40))
		h.Ingest(100, telem.NewSeriesV[uint8](1))
		for range 5 {
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()
		}
		out, _ := h.Flush()
		Expect(lastI64(out, 101)).To(Equal(int64(41)))
	})

	It("fires the host exactly once per trigger and never on a channel-only update", func(ctx SpecContext) {
		countSamples := func(fr telem.Frame[uint32], key uint32) int {
			total := 0
			for _, s := range fr.Get(key).Series {
				total += int(s.Len())
			}
			return total
		}
		resolver := channelSymbols(map[string]channelDef{
			"trig": {types.U8(), 100},
			"v":    {types.I64(), 102},
			"out":  {types.I64(), 101},
		})
		h := newRuntimeHarness(ctx,
			`func emit{val i64} () i64 { return val }
			trig -> emit{val = v + 1} -> out`, resolver,
			channels.Digest{Key: 100, DataType: telem.Uint8T},
			channels.Digest{Key: 102, DataType: telem.Int64T},
			channels.Digest{Key: 101, DataType: telem.Int64T},
		)
		defer h.Close(ctx)
		h.Ingest(102, telem.NewSeriesV[int64](40))
		h.Tick(ctx, telem.Millisecond)
		h.channelState.ClearReads()
		h.Ingest(102, telem.NewSeriesV[int64](50))
		h.Tick(ctx, telem.Millisecond)
		h.channelState.ClearReads()
		h.Ingest(100, telem.NewSeriesV[uint8](1))
		h.Tick(ctx, telem.Millisecond)
		h.channelState.ClearReads()
		h.Ingest(100, telem.NewSeriesV[uint8](1))
		h.Tick(ctx, telem.Millisecond)
		h.channelState.ClearReads()
		out, _ := h.Flush()
		Expect(countSamples(out, 101)).To(Equal(2))
		Expect(lastI64(out, 101)).To(Equal(int64(51)))
	})

	It("handles a positional brace expression that concatenates a channel value", func(ctx SpecContext) {
		lastStr := func(fr telem.Frame[uint32], key uint32) string {
			ch := fr.Get(key)
			Expect(ch.Series).ToNot(BeEmpty())
			vals := telem.UnmarshalSeries[string](ch.Series[len(ch.Series)-1])
			return vals[len(vals)-1]
		}
		resolver := channelSymbols(map[string]channelDef{
			"trig": {types.U8(), 100},
			"v":    {types.I64(), 102},
			"name": {types.String(), 101},
		})
		h := newRuntimeHarness(ctx,
			`func emit{label str} () str { return label }
			trig -> emit{"MY_RANGE_" + str(v)} -> name`, resolver,
			channels.Digest{Key: 100, DataType: telem.Uint8T},
			channels.Digest{Key: 102, DataType: telem.Int64T},
			channels.Digest{Key: 101, DataType: telem.StringT},
		)
		defer h.Close(ctx)
		h.Ingest(102, telem.NewSeriesV[int64](42))
		h.Ingest(100, telem.NewSeriesV[uint8](1))
		for range 5 {
			h.Tick(ctx, telem.Millisecond)
			h.channelState.ClearReads()
		}
		out, _ := h.Flush()
		Expect(lastStr(out, 101)).To(Equal("MY_RANGE_42"))
	})
})
