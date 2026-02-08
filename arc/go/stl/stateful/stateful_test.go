// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package stateful_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/stl/stateful"
	"github.com/synnaxlabs/arc/stl/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var ctx = context.Background()

var _ = Describe("Vars", func() {
	var (
		rt      *testutil.MockHostRuntime
		seriesS *state.SeriesHandleStore
		strS    *state.StringHandleStore
		mod     *stateful.Module
	)

	BeforeEach(func() {
		rt = testutil.NewMockHostRuntime()
		seriesS = state.NewSeriesHandleStore()
		strS = state.NewStringHandleStore()
		mod = stateful.NewModule(seriesS, strS)
		Expect(mod.BindTo(ctx, rt)).To(Succeed())
	})

	Describe("i32 scalar types", func() {
		It("Should load initial value and persist stores", func() {
			nodeCtx := stl.WithNodeKey(ctx, "node1")
			load := testutil.Get[func(context.Context, uint32, uint32) uint32](rt, "state", "load_i32")
			store := testutil.Get[func(context.Context, uint32, uint32)](rt, "state", "store_i32")

			Expect(load(nodeCtx, 0, 42)).To(Equal(uint32(42)))
			Expect(load(nodeCtx, 0, 99)).To(Equal(uint32(42)))
			store(nodeCtx, 0, 100)
			Expect(load(nodeCtx, 0, 99)).To(Equal(uint32(100)))
		})

		It("Should keep different varIDs independent", func() {
			nodeCtx := stl.WithNodeKey(ctx, "node1")
			load := testutil.Get[func(context.Context, uint32, uint32) uint32](rt, "state", "load_u8")
			store := testutil.Get[func(context.Context, uint32, uint32)](rt, "state", "store_u8")

			store(nodeCtx, 0, 10)
			store(nodeCtx, 1, 20)
			Expect(load(nodeCtx, 0, 0)).To(Equal(uint32(10)))
			Expect(load(nodeCtx, 1, 0)).To(Equal(uint32(20)))
		})
	})

	Describe("i64 scalar types", func() {
		It("Should load initial value and persist stores", func() {
			nodeCtx := stl.WithNodeKey(ctx, "node1")
			load := testutil.Get[func(context.Context, uint32, uint64) uint64](rt, "state", "load_u64")
			store := testutil.Get[func(context.Context, uint32, uint64)](rt, "state", "store_u64")

			Expect(load(nodeCtx, 0, 1000)).To(Equal(uint64(1000)))
			store(nodeCtx, 0, 2000)
			Expect(load(nodeCtx, 0, 0)).To(Equal(uint64(2000)))
		})
	})

	Describe("float types", func() {
		It("Should load and store f32", func() {
			nodeCtx := stl.WithNodeKey(ctx, "node1")
			load := testutil.Get[func(context.Context, uint32, float32) float32](rt, "state", "load_f32")
			store := testutil.Get[func(context.Context, uint32, float32)](rt, "state", "store_f32")

			Expect(load(nodeCtx, 0, float32(3.14))).To(Equal(float32(3.14)))
			store(nodeCtx, 0, float32(2.718))
			Expect(load(nodeCtx, 0, 0)).To(Equal(float32(2.718)))
		})

		It("Should load and store f64", func() {
			nodeCtx := stl.WithNodeKey(ctx, "node1")
			load := testutil.Get[func(context.Context, uint32, float64) float64](rt, "state", "load_f64")
			store := testutil.Get[func(context.Context, uint32, float64)](rt, "state", "store_f64")

			Expect(load(nodeCtx, 0, 3.14159)).To(Equal(3.14159))
			store(nodeCtx, 0, 2.71828)
			Expect(load(nodeCtx, 0, 0)).To(Equal(2.71828))
		})
	})

	Describe("node key isolation", func() {
		It("Should isolate state between different node keys", func() {
			ctx1 := stl.WithNodeKey(ctx, "node1")
			ctx2 := stl.WithNodeKey(ctx, "node2")
			load := testutil.Get[func(context.Context, uint32, uint32) uint32](rt, "state", "load_i32")
			store := testutil.Get[func(context.Context, uint32, uint32)](rt, "state", "store_i32")

			store(ctx1, 0, 100)
			store(ctx2, 0, 200)
			Expect(load(ctx1, 0, 0)).To(Equal(uint32(100)))
			Expect(load(ctx2, 0, 0)).To(Equal(uint32(200)))
		})
	})

	Describe("string state", func() {
		It("Should load and store strings via handles", func() {
			nodeCtx := stl.WithNodeKey(ctx, "node1")
			load := testutil.Get[func(context.Context, uint32, uint32) uint32](rt, "state", "load_str")
			store := testutil.Get[func(context.Context, uint32, uint32)](rt, "state", "store_str")

			initH := strS.Create("initial")
			rh := load(nodeCtx, 0, initH)
			Expect(rh).To(Equal(initH))
			Expect(MustBeOk(strS.Get(rh))).To(Equal("initial"))

			newH := strS.Create("updated")
			store(nodeCtx, 0, newH)
			rh2 := load(nodeCtx, 0, 0)
			Expect(MustBeOk(strS.Get(rh2))).To(Equal("updated"))
		})
	})

	Describe("series state", func() {
		It("Should load and store series via handles", func() {
			nodeCtx := stl.WithNodeKey(ctx, "node1")
			load := testutil.Get[func(context.Context, uint32, uint32) uint32](rt, "state", "load_series_f64")
			store := testutil.Get[func(context.Context, uint32, uint32)](rt, "state", "store_series_f64")

			initSer := telem.NewSeriesV(1.0, 2.0, 3.0)
			initH := seriesS.Store(initSer)

			rh := load(nodeCtx, 0, initH)
			Expect(rh).To(Equal(initH))
			ser := MustBeOk(seriesS.Get(rh))
			Expect(ser.Len()).To(Equal(int64(3)))

			newSer := telem.NewSeriesV(10.0, 20.0)
			newH := seriesS.Store(newSer)
			store(nodeCtx, 0, newH)

			rh2 := load(nodeCtx, 0, 0)
			ser2 := MustBeOk(seriesS.Get(rh2))
			Expect(ser2.Len()).To(Equal(int64(2)))
			Expect(telem.ValueAt[float64](ser2, 0)).To(Equal(10.0))
		})
	})
})
