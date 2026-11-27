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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/stl/channels"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Scope paths", func() {
	const source = `
	sequence main {
	    stage first {
	        go_cmd == 1 => next
	    }
	    stage second {
	        go_cmd == 2 => next
	    }
	    1 -> out
	}
	start_cmd => main`

	var (
		h       *runtimeHarness
		reports [][][]string
	)

	BeforeEach(func(ctx SpecContext) {
		h = newRuntimeHarness(ctx, source, channelSymbols(map[string]channelDef{
			"start_cmd": {types.U8(), 100},
			"go_cmd":    {types.U8(), 101},
			"out":       {types.U8(), 102},
		}),
			channels.Digest{Key: 100, DataType: telem.Uint8T},
			channels.Digest{Key: 101, DataType: telem.Uint8T},
			channels.Digest{Key: 102, DataType: telem.Uint8T},
		)
		reports = nil
		h.scheduler.SetScopeHandler(func(_ context.Context, paths [][]string) {
			reports = append(reports, paths)
		})
		DeferCleanup(h.Close)
	})

	tick := func(ctx SpecContext) {
		h.Tick(ctx, telem.Millisecond)
		h.channelState.ClearReads()
	}

	push := func(ctx SpecContext, key uint32, v uint8) {
		h.Ingest(key, telem.NewSeriesV(v))
		tick(ctx)
	}

	It(
		"Should report no paths on the first cycle of an idle program",
		func(ctx SpecContext) {
			tick(ctx)
			Expect(reports).To(Equal([][][]string{nil}))
		},
	)

	It(
		"Should report each start, stage change, and end of a sequence",
		func(ctx SpecContext) {
			tick(ctx)
			push(ctx, 100, 1)
			push(ctx, 101, 1)
			push(ctx, 101, 2)
			Expect(reports).To(Equal([][][]string{
				nil,
				{{"main", "first"}},
				{{"main", "second"}},
				nil,
			}))
			Expect(h.scheduler.ActivePaths()).To(BeEmpty())
		},
	)

	It("Should not report a cycle that changes no path", func(ctx SpecContext) {
		tick(ctx)
		push(ctx, 100, 1)
		for range 5 {
			tick(ctx)
		}
		push(ctx, 101, 0)
		push(ctx, 100, 1)
		Expect(reports).To(HaveLen(2))
		Expect(h.scheduler.ActivePaths()).To(Equal([][]string{{"main", "first"}}))
	})
})
