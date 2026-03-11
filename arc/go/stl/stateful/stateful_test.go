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
	"github.com/synnaxlabs/arc/stl/series"
	"github.com/synnaxlabs/arc/stl/stateful"
	"github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/stl/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Vars", func() {
	var (
		ctx     context.Context
		rt      *testutil.Runtime
		seriesS *series.ProgramState
		strS    *strings.ProgramState
		mod     *stateful.Module
	)

	call := func(fn string, args ...uint64) []uint64 {
		return rt.Call(ctx, "state", fn, args...)
	}
	callU32 := func(fn string, args ...uint64) uint32 {
		return testutil.AsU32(call(fn, args...)[0])
	}
	callU64 := func(fn string, args ...uint64) uint64 {
		return call(fn, args...)[0]
	}
	callF32 := func(fn string, args ...uint64) float32 {
		return testutil.AsF32(call(fn, args...)[0])
	}
	callF64 := func(fn string, args ...uint64) float64 {
		return testutil.AsF64(call(fn, args...)[0])
	}

	BeforeEach(func() {
		ctx = context.Background()
		rt = testutil.NewRuntime(ctx)
		seriesS = series.NewProgramState()
		strS = strings.NewProgramState()
		var err error
		mod, err = stateful.NewModule(ctx, seriesS, strS, rt.Underlying())
		Expect(err).ToNot(HaveOccurred())
		rt.Passthrough(ctx, "state")
	})

	AfterEach(func() {
		Expect(rt.Close(ctx)).To(Succeed())
	})

	Describe("i32 scalar types", func() {
		It("Should load initial value and persist stores", func() {
			mod.SetNodeKey("node1")
			Expect(callU32("load_i32", testutil.U32(0), testutil.U32(42))).To(Equal(uint32(42)))
			Expect(callU32("load_i32", testutil.U32(0), testutil.U32(99))).To(Equal(uint32(42)))
			rt.CallVoid(ctx, "state", "store_i32", testutil.U32(0), testutil.U32(100))
			Expect(callU32("load_i32", testutil.U32(0), testutil.U32(99))).To(Equal(uint32(100)))
		})

		It("Should keep different varIDs independent", func() {
			mod.SetNodeKey("node1")
			rt.CallVoid(ctx, "state", "store_u8", testutil.U32(0), testutil.U32(10))
			rt.CallVoid(ctx, "state", "store_u8", testutil.U32(1), testutil.U32(20))
			Expect(callU32("load_u8", testutil.U32(0), testutil.U32(0))).To(Equal(uint32(10)))
			Expect(callU32("load_u8", testutil.U32(1), testutil.U32(0))).To(Equal(uint32(20)))
		})

		It("Should return latest value after overwrite", func() {
			mod.SetNodeKey("node1")
			rt.CallVoid(ctx, "state", "store_i32", testutil.U32(0), testutil.U32(10))
			rt.CallVoid(ctx, "state", "store_i32", testutil.U32(0), testutil.U32(20))
			rt.CallVoid(ctx, "state", "store_i32", testutil.U32(0), testutil.U32(30))
			Expect(callU32("load_i32", testutil.U32(0), testutil.U32(0))).To(Equal(uint32(30)))
		})

		It("Should use stored value instead of init on subsequent loads", func() {
			mod.SetNodeKey("node1")
			Expect(callU32("load_i32", testutil.U32(0), testutil.U32(42))).To(Equal(uint32(42)))
			rt.CallVoid(ctx, "state", "store_i32", testutil.U32(0), testutil.U32(100))
			Expect(callU32("load_i32", testutil.U32(0), testutil.U32(99))).To(Equal(uint32(100)))
		})
	})

	Describe("i64 scalar types", func() {
		It("Should load initial value and persist stores", func() {
			mod.SetNodeKey("node1")
			Expect(callU64("load_u64", testutil.U32(0), testutil.U64(1000))).To(Equal(uint64(1000)))
			rt.CallVoid(ctx, "state", "store_u64", testutil.U32(0), testutil.U64(2000))
			Expect(callU64("load_u64", testutil.U32(0), testutil.U64(0))).To(Equal(uint64(2000)))
		})

		It("Should create inner map on store to a fresh node key", func() {
			mod.SetNodeKey("fresh")
			rt.CallVoid(ctx, "state", "store_i64", testutil.U32(0), testutil.I64(999))
			Expect(callU64("load_i64", testutil.U32(0), testutil.I64(0))).To(Equal(uint64(999)))
		})
	})

	Describe("float types", func() {
		It("Should load and store f32", func() {
			mod.SetNodeKey("node1")
			Expect(callF32("load_f32", testutil.U32(0), testutil.F32(3.14))).To(Equal(float32(3.14)))
			rt.CallVoid(ctx, "state", "store_f32", testutil.U32(0), testutil.F32(2.718))
			Expect(callF32("load_f32", testutil.U32(0), testutil.F32(0))).To(Equal(float32(2.718)))
		})

		It("Should create inner map on f32 store to a fresh node key", func() {
			mod.SetNodeKey("fresh")
			rt.CallVoid(ctx, "state", "store_f32", testutil.U32(0), testutil.F32(1.5))
			Expect(callF32("load_f32", testutil.U32(0), testutil.F32(0))).To(Equal(float32(1.5)))
		})

		It("Should load and store f64", func() {
			mod.SetNodeKey("node1")
			Expect(callF64("load_f64", testutil.U32(0), testutil.F64(3.14159))).To(Equal(3.14159))
			rt.CallVoid(ctx, "state", "store_f64", testutil.U32(0), testutil.F64(2.71828))
			Expect(callF64("load_f64", testutil.U32(0), testutil.F64(0))).To(Equal(2.71828))
		})

		It("Should create inner map on f64 store to a fresh node key", func() {
			mod.SetNodeKey("fresh")
			rt.CallVoid(ctx, "state", "store_f64", testutil.U32(0), testutil.F64(9.99))
			Expect(callF64("load_f64", testutil.U32(0), testutil.F64(0))).To(Equal(9.99))
		})
	})

	Describe("node key isolation", func() {
		It("Should isolate state between different node keys", func() {
			mod.SetNodeKey("node1")
			rt.CallVoid(ctx, "state", "store_i32", testutil.U32(0), testutil.U32(100))
			mod.SetNodeKey("node2")
			rt.CallVoid(ctx, "state", "store_i32", testutil.U32(0), testutil.U32(200))
			mod.SetNodeKey("node1")
			Expect(callU32("load_i32", testutil.U32(0), testutil.U32(0))).To(Equal(uint32(100)))
			mod.SetNodeKey("node2")
			Expect(callU32("load_i32", testutil.U32(0), testutil.U32(0))).To(Equal(uint32(200)))
		})
	})

	Describe("string state", func() {
		It("Should load and store strings via handles", func() {
			mod.SetNodeKey("node1")
			initH := strS.Create("initial")
			rh := callU32("load_str", testutil.U32(0), testutil.U32(initH))
			Expect(rh).To(Equal(initH))
			Expect(MustBeOk(strS.Get(rh))).To(Equal("initial"))
			newH := strS.Create("updated")
			rt.CallVoid(ctx, "state", "store_str", testutil.U32(0), testutil.U32(newH))
			rh2 := callU32("load_str", testutil.U32(0), testutil.U32(0))
			Expect(MustBeOk(strS.Get(rh2))).To(Equal("updated"))
		})

		It("Should create a new handle on reload with stored value", func() {
			mod.SetNodeKey("node1")
			h := strS.Create("persisted")
			rt.CallVoid(ctx, "state", "store_str", testutil.U32(0), testutil.U32(h))
			rh := callU32("load_str", testutil.U32(0), testutil.U32(0))
			Expect(MustBeOk(strS.Get(rh))).To(Equal("persisted"))
		})

		It("Should silently ignore store with invalid handle", func() {
			mod.SetNodeKey("node1")
			rt.CallVoid(ctx, "state", "store_str", testutil.U32(0), testutil.U32(9999))
			fallbackH := strS.Create("fallback")
			Expect(MustBeOk(strS.Get(callU32("load_str", testutil.U32(0), testutil.U32(fallbackH))))).To(Equal("fallback"))
		})
	})

	Describe("series state", func() {
		It("Should load and store series via handles", func() {
			mod.SetNodeKey("node1")
			initH := seriesS.Store(telem.NewSeriesV(1.0, 2.0, 3.0))
			rh := callU32("load_series_f64", testutil.U32(0), testutil.U32(initH))
			Expect(rh).To(Equal(initH))
			Expect(MustBeOk(seriesS.Get(rh)).Len()).To(Equal(int64(3)))
			rt.CallVoid(ctx, "state", "store_series_f64", testutil.U32(0), testutil.U32(seriesS.Store(telem.NewSeriesV(10.0, 20.0))))
			ser2 := MustBeOk(seriesS.Get(callU32("load_series_f64", testutil.U32(0), testutil.U32(0))))
			Expect(ser2.Len()).To(Equal(int64(2)))
			Expect(telem.ValueAt[float64](ser2, 0)).To(Equal(10.0))
		})

		It("Should deep copy on initial load so mutations don't corrupt state", func() {
			mod.SetNodeKey("node1")
			initSer := telem.NewSeriesV[int32](1, 2, 3)
			initH := seriesS.Store(initSer)
			callU32("load_series_i32", testutil.U32(0), testutil.U32(initH))
			initSer.Data[0] = 0xFF
			stored := MustBeOk(seriesS.Get(callU32("load_series_i32", testutil.U32(0), testutil.U32(0))))
			Expect(telem.ValueAt[int32](stored, 0)).To(Equal(int32(1)))
		})

		It("Should deep copy on store so mutations don't corrupt state", func() {
			mod.SetNodeKey("node1")
			ser := telem.NewSeriesV[int32](10, 20, 30)
			rt.CallVoid(ctx, "state", "store_series_i32", testutil.U32(0), testutil.U32(seriesS.Store(ser)))
			ser.Data[0] = 0xFF
			stored := MustBeOk(seriesS.Get(callU32("load_series_i32", testutil.U32(0), testutil.U32(0))))
			Expect(telem.ValueAt[int32](stored, 0)).To(Equal(int32(10)))
		})

		It("Should isolate series state between different node keys", func() {
			mod.SetNodeKey("node1")
			rt.CallVoid(ctx, "state", "store_series_f64", testutil.U32(0), testutil.U32(seriesS.Store(telem.NewSeriesV(1.0, 2.0))))
			mod.SetNodeKey("node2")
			rt.CallVoid(ctx, "state", "store_series_f64", testutil.U32(0), testutil.U32(seriesS.Store(telem.NewSeriesV(100.0, 200.0, 300.0))))
			mod.SetNodeKey("node1")
			Expect(MustBeOk(seriesS.Get(callU32("load_series_f64", testutil.U32(0), testutil.U32(0)))).Len()).To(Equal(int64(2)))
			mod.SetNodeKey("node2")
			Expect(MustBeOk(seriesS.Get(callU32("load_series_f64", testutil.U32(0), testutil.U32(0)))).Len()).To(Equal(int64(3)))
		})
	})
})
