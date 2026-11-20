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

var _ = Describe("Empty series literals", func() {
	DescribeTable("Should load and read the length of an empty series",
		func(ctx SpecContext, element string) {
			h := newRuntimeHarness(ctx, `
			func check() {v i64} i64 {
				y series `+element+` := []
				n := len(y)
				y = []
				return n + len(y)
			}
			x -> check() -> out`, channelSymbols(map[string]channelDef{
				"x":   {types.I64(), 100},
				"out": {types.I64(), 200},
			}),
				channels.Digest{Key: 100, DataType: telem.Int64T},
				channels.Digest{Key: 200, DataType: telem.Int64T},
			)
			defer h.Close(ctx)
			h.Ingest(100, telem.NewSeriesV[int64](1))
			h.Tick(ctx, telem.Millisecond)
			out, _ := h.Flush()
			Expect(out.Get(200).Series[0].Unmarshal[int64]()[0]).To(Equal(int64(0)))
		},
		Entry("str", "str"),
		Entry("bool", "bool"),
		Entry("f32", "f32"),
		Entry("f64", "f64"),
		Entry("i8", "i8"),
		Entry("i16", "i16"),
		Entry("i32", "i32"),
		Entry("i64", "i64"),
		Entry("u8", "u8"),
		Entry("u16", "u16"),
		Entry("u32", "u32"),
		Entry("u64", "u64"),
		Entry("timestamp", "i64 ns"),
	)
})
