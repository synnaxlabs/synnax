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

var _ = Describe("A string channel that changes to a number type (break)", func() {
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
		var written []string
		for _, s := range out.Get(102).Series {
			written = append(written, s.Unmarshal[string]()...)
		}
		Expect(written).To(BeEmpty())
	})
})
