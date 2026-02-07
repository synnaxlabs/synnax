// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/stl/channel"
	"github.com/synnaxlabs/arc/stl/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var ctx = context.Background()

var _ = Describe("Channel", func() {
	var (
		rt  *testutil.MockHostRuntime
		cs  *state.ChannelState
		ss  *state.StringHandleStore
		mod *channel.Module
	)

	BeforeEach(func() {
		rt = testutil.NewMockHostRuntime()
		cs = state.NewChannelState([]state.ChannelDigest{
			{Key: 1, DataType: telem.Float64T},
			{Key: 2, DataType: telem.Int32T},
			{Key: 3, DataType: telem.StringT},
		})
		ss = state.NewStringHandleStore()
		mod = channel.NewModule(cs, ss)
		Expect(mod.BindTo(ctx, rt)).To(Succeed())
	})

	Describe("i32 types", func() {
		It("Should write and read back u8 values", func() {
			write := testutil.Get[func(context.Context, uint32, uint32)](rt, "channel", "write_u8")
			read := testutil.Get[func(context.Context, uint32) uint32](rt, "channel", "read_u8")
			write(ctx, 2, 42)
			fr := telem.Frame[uint32]{}
			fr, _ = cs.Flush(fr)
			cs.Ingest(fr)
			Expect(read(ctx, 2)).To(Equal(uint32(42)))
		})

		It("Should write and read back i32 values", func() {
			write := testutil.Get[func(context.Context, uint32, uint32)](rt, "channel", "write_i32")
			read := testutil.Get[func(context.Context, uint32) uint32](rt, "channel", "read_i32")
			write(ctx, 2, 100)
			fr := telem.Frame[uint32]{}
			fr, _ = cs.Flush(fr)
			cs.Ingest(fr)
			Expect(read(ctx, 2)).To(Equal(uint32(100)))
		})
	})

	Describe("i64 types", func() {
		It("Should write and read back u64 values", func() {
			write := testutil.Get[func(context.Context, uint32, uint64)](rt, "channel", "write_u64")
			read := testutil.Get[func(context.Context, uint32) uint64](rt, "channel", "read_u64")
			write(ctx, 1, 12345)
			fr := telem.Frame[uint32]{}
			fr, _ = cs.Flush(fr)
			cs.Ingest(fr)
			Expect(read(ctx, 1)).To(Equal(uint64(12345)))
		})

		It("Should write and read back i64 values", func() {
			write := testutil.Get[func(context.Context, uint32, uint64)](rt, "channel", "write_i64")
			read := testutil.Get[func(context.Context, uint32) uint64](rt, "channel", "read_i64")
			write(ctx, 1, 99999)
			fr := telem.Frame[uint32]{}
			fr, _ = cs.Flush(fr)
			cs.Ingest(fr)
			Expect(read(ctx, 1)).To(Equal(uint64(99999)))
		})
	})

	Describe("float types", func() {
		It("Should write and read back f32 values", func() {
			write := testutil.Get[func(context.Context, uint32, float32)](rt, "channel", "write_f32")
			read := testutil.Get[func(context.Context, uint32) float32](rt, "channel", "read_f32")
			write(ctx, 1, 3.14)
			fr := telem.Frame[uint32]{}
			fr, _ = cs.Flush(fr)
			cs.Ingest(fr)
			Expect(read(ctx, 1)).To(BeNumerically("~", 3.14, 0.001))
		})

		It("Should write and read back f64 values", func() {
			write := testutil.Get[func(context.Context, uint32, float64)](rt, "channel", "write_f64")
			read := testutil.Get[func(context.Context, uint32) float64](rt, "channel", "read_f64")
			write(ctx, 1, 2.718281828)
			fr := telem.Frame[uint32]{}
			fr, _ = cs.Flush(fr)
			cs.Ingest(fr)
			Expect(read(ctx, 1)).To(BeNumerically("~", 2.718281828, 0.0001))
		})
	})

	Describe("string type", func() {
		It("Should write and read back string values via handles", func() {
			write := testutil.Get[func(context.Context, uint32, uint32)](rt, "channel", "write_str")
			read := testutil.Get[func(context.Context, uint32) uint32](rt, "channel", "read_str")
			h := ss.Create("hello world")
			write(ctx, 3, h)
			fr := telem.Frame[uint32]{}
			fr, _ = cs.Flush(fr)
			cs.Ingest(fr)
			rh := read(ctx, 3)
			Expect(rh).ToNot(BeZero())
			Expect(MustBeOk(ss.Get(rh))).To(Equal("hello world"))
		})
	})

	Describe("read with no data", func() {
		It("Should return 0 when no data has been ingested", func() {
			read := testutil.Get[func(context.Context, uint32) float64](rt, "channel", "read_f64")
			Expect(read(ctx, 1)).To(Equal(float64(0)))
		})
	})
})
