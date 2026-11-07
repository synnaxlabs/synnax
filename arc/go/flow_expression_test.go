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

var _ = Describe("Literal expressions in flows", func() {
	DescribeTable("Should write a literal expression at the sink's type",
		func(ctx SpecContext, source string, dt types.Type, expected float64) {
			dataType := dt.ToTelem()
			h := newRuntimeHarness(ctx, source, channelSymbols(map[string]channelDef{
				"x":   {types.U8(), 100},
				"out": {dt, 200},
			}),
				channels.Digest{Key: 100, DataType: telem.Uint8T},
				channels.Digest{Key: 200, DataType: dataType},
			)
			defer h.Close(ctx)
			for range 3 {
				h.Ingest(100, telem.NewSeriesV[uint8](1))
				h.Tick(ctx, telem.Millisecond)
				h.channelState.ClearReads()
			}
			out, _ := h.Flush()
			series := out.Get(200).Series
			Expect(series).ToNot(BeEmpty())
			want := telem.NewSeriesFromAny(expected, dataType).Data
			last := series[len(series)-1].Data
			Expect(last[len(last)-len(want):]).To(Equal(want))
		},
		Entry("integers to i32", "x -> 1 + 2 -> out", types.I32(), 3.0),
		Entry("integers to u8", "x -> 1 + 2 -> out", types.U8(), 3.0),
		Entry("floats to f32", "x -> 1.5 + 2.5 -> out", types.F32(), 4.0),
		Entry(
			"integers to i32 in a stage",
			"sequence main {\n stage a {\n 1 + 2 -> out\n }\n}\nx => main",
			types.I32(),
			3.0,
		),
	)
})
