// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package state_test

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/stl/series/state"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("State", func() {
	var s *state.State

	BeforeEach(func() {
		s = state.New()
	})

	Describe("Store", func() {
		It("Should return a non-zero handle", func() {
			h := s.Store(telem.NewSeriesV[float32](1.0, 2.0))
			Expect(h).ToNot(BeZero())
		})

		It("Should return unique handles for successive stores", func() {
			h1 := s.Store(telem.NewSeriesV[float32](1.0))
			h2 := s.Store(telem.NewSeriesV[float32](2.0))
			h3 := s.Store(telem.NewSeriesV[int32](3))
			Expect(h1).ToNot(Equal(h2))
			Expect(h2).ToNot(Equal(h3))
		})

		It("Should return monotonically increasing handles", func() {
			h1 := s.Store(telem.NewSeriesV[int32](1))
			h2 := s.Store(telem.NewSeriesV[int32](2))
			h3 := s.Store(telem.NewSeriesV[int32](3))
			Expect(h2).To(Equal(h1 + 1))
			Expect(h3).To(Equal(h2 + 1))
		})

		It("Should store empty series", func() {
			h := s.Store(telem.NewSeriesV[float32]())
			ser := MustBeOk(s.Get(h))
			Expect(ser.Len()).To(Equal(int64(0)))
		})

		It("Should store series with boundary float values", func() {
			ser := telem.NewSeriesV[float64](
				math.MaxFloat64, math.SmallestNonzeroFloat64, math.Inf(1), math.NaN(),
			)
			h := s.Store(ser)
			retrieved := MustBeOk(s.Get(h))
			Expect(retrieved.Len()).To(Equal(int64(4)))
			Expect(telem.ValueAt[float64](retrieved, 0)).To(Equal(math.MaxFloat64))
			Expect(math.IsNaN(telem.ValueAt[float64](retrieved, 3))).To(BeTrue())
		})

		It("Should store series with max/min integer values", func() {
			ser := telem.NewSeriesV[int32](math.MaxInt32, math.MinInt32)
			h := s.Store(ser)
			retrieved := MustBeOk(s.Get(h))
			Expect(telem.ValueAt[int32](retrieved, 0)).To(Equal(int32(math.MaxInt32)))
			Expect(telem.ValueAt[int32](retrieved, 1)).To(Equal(int32(math.MinInt32)))
		})

		It("Should handle many successive stores", func() {
			handles := make([]uint32, 1000)
			for i := range handles {
				handles[i] = s.Store(telem.NewSeriesV[int32](int32(i)))
			}
			for i, h := range handles {
				ser := MustBeOk(s.Get(h))
				Expect(telem.ValueAt[int32](ser, 0)).To(Equal(int32(i)))
			}
		})

		It("Should store different data types", func() {
			h1 := s.Store(telem.NewSeriesV[uint8](255))
			h2 := s.Store(telem.NewSeriesV[int64](math.MaxInt64))
			h3 := s.Store(telem.NewSeriesV[float32](3.14))
			Expect(MustBeOk(s.Get(h1)).DataType).To(Equal(telem.Uint8T))
			Expect(MustBeOk(s.Get(h2)).DataType).To(Equal(telem.Int64T))
			Expect(MustBeOk(s.Get(h3)).DataType).To(Equal(telem.Float32T))
		})
	})

	Describe("Get", func() {
		It("Should retrieve a previously stored series", func() {
			original := telem.NewSeriesV[float64](3.14, 2.71)
			h := s.Store(original)
			retrieved := MustBeOk(s.Get(h))
			Expect(retrieved).To(telem.MatchSeries(original))
		})

		It("Should return false for an unknown handle", func() {
			_, ok := s.Get(999)
			Expect(ok).To(BeFalse())
		})

		It("Should return false for handle zero", func() {
			_, ok := s.Get(0)
			Expect(ok).To(BeFalse())
		})

		It("Should retrieve different series by their handles", func() {
			s1 := telem.NewSeriesV[int32](10, 20)
			s2 := telem.NewSeriesV[float32](1.5)
			h1 := s.Store(s1)
			h2 := s.Store(s2)
			Expect(MustBeOk(s.Get(h1))).To(telem.MatchSeries(s1))
			Expect(MustBeOk(s.Get(h2))).To(telem.MatchSeries(s2))
		})

		It("Should return false for a handle that was cleared", func() {
			h := s.Store(telem.NewSeriesV[float32](1.0))
			s.Clear()
			_, ok := s.Get(h)
			Expect(ok).To(BeFalse())
		})

		It("Should return false for max uint32 handle", func() {
			_, ok := s.Get(math.MaxUint32)
			Expect(ok).To(BeFalse())
		})
	})

	Describe("Clear", func() {
		It("Should remove all stored series", func() {
			h := s.Store(telem.NewSeriesV[float32](1.0))
			s.Clear()
			_, ok := s.Get(h)
			Expect(ok).To(BeFalse())
		})

		It("Should reset the counter so new handles start from 1", func() {
			s.Store(telem.NewSeriesV[float32](1.0))
			s.Store(telem.NewSeriesV[float32](2.0))
			s.Clear()
			h := s.Store(telem.NewSeriesV[float32](3.0))
			Expect(h).To(Equal(uint32(1)))
		})

		It("Should allow storing new series after clear", func() {
			s.Store(telem.NewSeriesV[float32](1.0))
			s.Clear()
			h := s.Store(telem.NewSeriesV[int64](42))
			retrieved := MustBeOk(s.Get(h))
			Expect(retrieved).To(telem.MatchSeries(telem.NewSeriesV[int64](42)))
		})

		It("Should be safe to call on an already empty state", func() {
			Expect(func() { s.Clear() }).ToNot(Panic())
		})

		It("Should be safe to call multiple times consecutively", func() {
			s.Store(telem.NewSeriesV[int32](1))
			s.Clear()
			s.Clear()
			s.Clear()
			h := s.Store(telem.NewSeriesV[int32](2))
			Expect(h).To(Equal(uint32(1)))
		})

		It("Should clear all handles from a large store", func() {
			for range 500 {
				s.Store(telem.NewSeriesV[int32](1))
			}
			s.Clear()
			_, ok := s.Get(1)
			Expect(ok).To(BeFalse())
			_, ok = s.Get(500)
			Expect(ok).To(BeFalse())
		})
	})

	Describe("New", func() {
		It("Should create a state where first handle is 1", func() {
			h := s.Store(telem.NewSeriesV[uint8](255))
			Expect(h).To(Equal(uint32(1)))
		})
	})
})
