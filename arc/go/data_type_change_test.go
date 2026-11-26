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

var _ = Describe("A channel that changes its data type", func() {
	run := func(ctx SpecContext, source string, x telem.Series) telem.Frame[uint32] {
		GinkgoHelper()
		h := newRuntimeHarness(ctx, source, channelSymbols(map[string]channelDef{
			"x":    {types.F64(), 100},
			"trig": {types.U8(), 101},
			"out":  {types.F64(), 102},
		}),
			channels.Digest{Key: 100, DataType: telem.Float64T},
			channels.Digest{Key: 101, DataType: telem.Uint8T},
			channels.Digest{Key: 102, DataType: telem.Float64T},
		)
		defer h.Close(ctx)
		h.Ingest(100, x)
		h.Ingest(101, telem.NewSeriesV[uint8](1))
		h.Tick(ctx, telem.Millisecond)
		out, _ := h.Flush()
		return out
	}

	It("Should compute with a sample of the new type read through a flow", func(
		ctx SpecContext,
	) {
		out := run(ctx, `
		func f() {v f64} f64 {
			return v * 2.0
		}
		x -> f() -> out`, telem.NewSeriesV[int32](300))
		Expect(out.Get(102).Series).To(HaveLen(1))
		Expect(out.Get(102).Series[0].Unmarshal[float64]()).To(Equal([]float64{600}))
	})

	It("Should compute with a sample of the new type read in a function body", func(
		ctx SpecContext,
	) {
		out := run(ctx, `
		func f() f64 {
			return x * 2.0
		}
		trig -> f() -> out`, telem.NewSeriesV[int32](300))
		Expect(out.Get(102).Series).To(HaveLen(1))
		Expect(out.Get(102).Series[0].Unmarshal[float64]()).To(Equal([]float64{600}))
	})
})

var _ = Describe("A string channel that changes to a number type", func() {
	It("Should not invent text from the bytes of a number series", func(
		ctx SpecContext,
	) {
		h := newRuntimeHarness(ctx, `
		func f() {
			out = label
		}
		trig -> f()`, channelSymbols(map[string]channelDef{
			"label": {types.String(), 100},
			"trig":  {types.U8(), 101},
			"out":   {types.String(), 102},
		}),
			channels.Digest{Key: 100, DataType: telem.StringT},
			channels.Digest{Key: 101, DataType: telem.Uint8T},
			channels.Digest{Key: 102, DataType: telem.StringT},
		)
		defer h.Close(ctx)
		// Two uint32 samples: 2, then the bytes "hi" and two zero bytes.
		h.Ingest(100, telem.NewSeriesV[uint32](2, 0x6968))
		h.Ingest(101, telem.NewSeriesV[uint8](1))
		h.Tick(ctx, telem.Millisecond)
		out, _ := h.Flush()
		series := out.Get(102).Series
		written := make([]string, 0, len(series))
		for _, s := range series {
			written = append(written, s.Unmarshal[string]()...)
		}
		Expect(written).To(BeEmpty())
	})
})
