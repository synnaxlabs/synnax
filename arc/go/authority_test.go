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

// Behavioral coverage for set_authority and the control.ProgramState
// flush pipeline. These tests pin the syntax, the dispatch, and the shape
// of the buffered AuthorityChange records — everything a caller of the
// runtime needs to depend on when translating Arc authority statements
// into real hardware control decisions.
var _ = Describe("Authority", func() {
	// set_authority{value=V, channel=C} buffers a single AuthorityChange
	// on the program's control.ProgramState when the node activates. The
	// change carries V as the authority and a pointer to C's key as the
	// target. A downstream runtime task would drain the buffer after each
	// cycle and apply the changes before committing channel writes.
	//
	// This test pins three properties of the pipeline: (1) the set_authority
	// syntax compiles and dispatches at runtime, (2) the buffered change
	// has the configured value, and (3) the buffered change targets the
	// right channel key. A regression in any of these would silently break
	// authority semantics downstream.
	It("set_authority{value,channel} buffers a change with the configured value and target", func(ctx SpecContext) {
		resolver := channelSymbols(map[string]channelDef{
			"trigger_cmd": {types.U8(), 100},
			"valve_cmd":   {types.U8(), 101},
		})
		h := newRuntimeHarness(ctx, `
			trigger_cmd -> set_authority{value=200, channel=valve_cmd}`, resolver,
			channel.Digest{Key: 100, DataType: telem.Uint8T},
			channel.Digest{Key: 101, DataType: telem.Uint8T},
		)
		defer h.Close(ctx)

		h.Ingest(100, telem.NewSeriesV[uint8](1))
		h.Tick(ctx, telem.Millisecond)
		h.channelState.ClearReads()

		changes := h.FlushAuthority()
		Expect(changes).To(HaveLen(1),
			"set_authority should buffer exactly one authority change per activation")
		Expect(changes[0].Authority).To(Equal(uint8(200)),
			"buffered change should carry the value from the config block")
		Expect(changes[0].Channel).ToNot(BeNil(),
			"channel=valve_cmd was configured, so Channel must not be nil (nil means global)")
		Expect(*changes[0].Channel).To(Equal(uint32(101)),
			"buffered Channel should be valve_cmd's resolved key (101)")
	})
})
