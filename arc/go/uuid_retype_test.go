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

var _ = Describe("A string channel that changes to uuid", func() {
	It("Should write only string series to a string channel", func(ctx SpecContext) {
		syms := channelSymbols(map[string]channelDef{
			"label": {types.String(), 100},
			"out":   {types.String(), 102},
		})
		h := newRuntimeHarness(ctx, `label -> out`, syms,
			channels.Digest{Key: 100, DataType: telem.StringT},
			channels.Digest{Key: 102, DataType: telem.StringT},
		)
		defer h.Close(ctx)
		h.Ingest(100, telem.Series{DataType: telem.UUIDT, Data: make([]byte, 16)})
		h.Tick(ctx, telem.Millisecond)
		out, _ := h.Flush()
		for _, s := range out.Get(102).Series {
			Expect(s.DataType).To(Equal(telem.StringT))
		}
	})
})
