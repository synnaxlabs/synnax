// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channels_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/stl/channels"
	"github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/stl/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Body reads across a type change (break)", func() {
	var (
		rt *testutil.Runtime
		cs *channels.ProgramState
		ss *strings.ProgramState
	)

	BeforeEach(func(ctx SpecContext) {
		rt = testutil.NewRuntime(ctx)
		cs = channels.NewProgramState([]channels.Digest{
			{Key: 3, Name: "label", DataType: telem.StringT},
			{Key: 4, Name: "armed", DataType: telem.BooleanT},
		})
		ss = strings.NewProgramState()
		MustSucceed(channels.NewHost(ctx, rt.Underlying(), cs, ss))
		rt.Passthrough(ctx, "channels")
	})

	AfterEach(func(ctx SpecContext) {
		Expect(rt.Close(ctx)).To(Succeed())
	})

	It("Should fail a string read of a channel whose last series holds numbers", func(
		ctx SpecContext,
	) {
		// Two uint32 samples: 2, then the bytes "hi" and two zero bytes.
		cs.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[uint32](2, 0x6968)))
		var read string
		failed := func() (failed bool) {
			defer func() { failed = recover() != nil }()
			result := rt.Call(ctx, "channels", "read_str", testutil.U32(3))
			read, _ = ss.Get(testutil.AsU32(result[0]))
			return false
		}()
		Expect(failed).To(BeTrue(), "read_str returned %q", read)
	})

	It("Should name the boolean type when a boolean read meets text", func(
		ctx SpecContext,
	) {
		cs.Ingest(telem.UnaryFrame[uint32](4, telem.NewSeriesV("on")))
		Expect(func() {
			rt.Call(ctx, "channels", "read_bool", testutil.U32(4))
		}).To(PanicWith(ContainSubstring(
			"channel armed (4): cannot convert a string series to boolean",
		)))
	})
})
