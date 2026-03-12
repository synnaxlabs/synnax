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
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/stl/channel"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ProgramState", func() {
	var s *channel.ProgramState

	BeforeEach(func() {
		s = channel.NewProgramState([]channel.Digest{
			{Key: 1, DataType: telem.Float32T, Index: 2},
			{Key: 3, DataType: telem.Int32T},
			{Key: 5, DataType: telem.Float64T, Index: 6},
		})
	})

	Describe("NewProgramState", func() {
		It("Should initialize index mappings from digests", func() {
			cs := channel.NewProgramState([]channel.Digest{
				{Key: 10, Index: 11},
				{Key: 20, Index: 21},
			})
			cs.WriteChannel(10, telem.NewSeriesV[float32](1.0), telem.NewSeriesSecondsTSV(100))
			fr, changed := cs.Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(fr.Get(11).Series).To(HaveLen(1))
		})

		It("Should handle nil digests", func() {
			cs := channel.NewProgramState(nil)
			Expect(cs).ToNot(BeNil())
			_, ok := cs.ReadValue(1)
			Expect(ok).To(BeFalse())
		})

		It("Should handle empty digests", func() {
			cs := channel.NewProgramState([]channel.Digest{})
			Expect(cs).ToNot(BeNil())
			_, ok := cs.ReadValue(1)
			Expect(ok).To(BeFalse())
		})

		It("Should ignore zero-value index in digests", func() {
			cs := channel.NewProgramState([]channel.Digest{{Key: 10, Index: 0}})
			cs.WriteValue(10, telem.NewSeriesV[int32](42))
			fr, changed := cs.Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(fr.Get(10).Series).To(HaveLen(1))
			Expect(fr.Get(0).Series).To(BeEmpty())
		})
	})

	Describe("Ingest", func() {
		It("Should buffer ingested frame data for later reads", func() {
			s.Ingest(telem.UnaryFrame[uint32](1, telem.NewSeriesV[float32](1.5, 2.5)))
			ser := MustBeOk(s.ReadValue(1))
			Expect(ser).To(telem.MatchSeries(telem.NewSeriesV[float32](1.5, 2.5)))
		})

		It("Should accumulate multiple ingestions into MultiSeries", func() {
			s.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](1)))
			s.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](2)))
			data, _, ok := s.ReadSeries(3)
			Expect(ok).To(BeTrue())
			Expect(data.Series).To(HaveLen(2))
		})

		It("Should handle multiple channels in a single frame", func() {
			fr := telem.Frame[uint32]{}
			fr = fr.Append(1, telem.NewSeriesV[float32](10.0))
			fr = fr.Append(3, telem.NewSeriesV[int32](42))
			s.Ingest(fr)
			Expect(MustBeOk(s.ReadValue(1))).To(
				telem.MatchSeries(telem.NewSeriesV[float32](10.0)),
			)
			Expect(MustBeOk(s.ReadValue(3))).To(
				telem.MatchSeries(telem.NewSeriesV[int32](42)),
			)
		})

		It("Should handle ingestion of channels not in digests", func() {
			s.Ingest(telem.UnaryFrame[uint32](999, telem.NewSeriesV[float64](1.0)))
			ser := MustBeOk(s.ReadValue(999))
			Expect(ser).To(telem.MatchSeries(telem.NewSeriesV[float64](1.0)))
		})

		It("Should handle an empty frame without panicking", func() {
			Expect(func() { s.Ingest(telem.Frame[uint32]{}) }).ToNot(Panic())
		})

		It("Should handle series with boundary float values", func() {
			s.Ingest(telem.UnaryFrame[uint32](
				5, telem.NewSeriesV[float64](math.MaxFloat64, math.SmallestNonzeroFloat64, math.Inf(1), math.Inf(-1)),
			))
			ser := MustBeOk(s.ReadValue(5))
			Expect(ser.Len()).To(Equal(int64(4)))
		})

		It("Should handle series with NaN values", func() {
			s.Ingest(telem.UnaryFrame[uint32](5, telem.NewSeriesV[float64](math.NaN())))
			ser := MustBeOk(s.ReadValue(5))
			Expect(ser.Len()).To(Equal(int64(1)))
			Expect(math.IsNaN(telem.ValueAt[float64](ser, 0))).To(BeTrue())
		})

		It("Should handle series with max/min integer values", func() {
			s.Ingest(telem.UnaryFrame[uint32](
				3, telem.NewSeriesV[int32](math.MaxInt32, math.MinInt32, 0),
			))
			ser := MustBeOk(s.ReadValue(3))
			Expect(telem.ValueAt[int32](ser, 0)).To(Equal(int32(math.MaxInt32)))
			Expect(telem.ValueAt[int32](ser, 1)).To(Equal(int32(math.MinInt32)))
			Expect(telem.ValueAt[int32](ser, 2)).To(Equal(int32(0)))
		})

		It("Should handle empty series", func() {
			s.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32]()))
			ser := MustBeOk(s.ReadValue(3))
			Expect(ser.Len()).To(Equal(int64(0)))
		})
	})

	Describe("ReadValue", func() {
		It("Should return the last series from the read buffer", func() {
			s.Ingest(telem.UnaryFrame[uint32](1, telem.NewSeriesV[float32](1.0)))
			s.Ingest(telem.UnaryFrame[uint32](1, telem.NewSeriesV[float32](2.0)))
			ser := MustBeOk(s.ReadValue(1))
			Expect(ser).To(telem.MatchSeries(telem.NewSeriesV[float32](2.0)))
		})

		It("Should return false for an unknown channel", func() {
			_, ok := s.ReadValue(999)
			Expect(ok).To(BeFalse())
		})

		It("Should return false for a channel with no data", func() {
			s = channel.NewProgramState([]channel.Digest{{Key: 10}})
			_, ok := s.ReadValue(10)
			Expect(ok).To(BeFalse())
		})

		It("Should return false for channel key zero", func() {
			_, ok := s.ReadValue(0)
			Expect(ok).To(BeFalse())
		})

		It("Should return the latest series after many ingestions", func() {
			for i := range 100 {
				s.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](int32(i))))
			}
			ser := MustBeOk(s.ReadValue(3))
			Expect(telem.ValueAt[int32](ser, 0)).To(Equal(int32(99)))
		})
	})

	Describe("WriteValue", func() {
		It("Should buffer the write for later flushing", func() {
			s.WriteValue(3, telem.NewSeriesV[int32](100))
			fr, changed := s.Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(fr.Get(3).Series[0]).To(
				telem.MatchSeries(telem.NewSeriesV[int32](100)),
			)
		})

		It("Should auto-write index channel for indexed channels", func() {
			s.WriteValue(1, telem.NewSeriesV[float32](5.0))
			fr, changed := s.Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(fr.Get(1).Series).To(HaveLen(1))
			Expect(fr.Get(2).Series).To(HaveLen(1))
			Expect(fr.Get(2).Series[0].DataType).To(Equal(telem.TimeStampT))
		})

		It("Should not write to an index for non-indexed channels", func() {
			s.WriteValue(3, telem.NewSeriesV[int32](50))
			fr, _ := s.Flush(telem.Frame[uint32]{})
			Expect(fr.Get(3).Series).To(HaveLen(1))
			Expect(fr.Get(0).Series).To(BeEmpty())
		})

		It("Should accumulate multiple writes in same cycle", func() {
			s.WriteValue(3, telem.NewSeriesV[int32](1))
			s.WriteValue(3, telem.NewSeriesV[int32](2))
			fr, _ := s.Flush(telem.Frame[uint32]{})
			Expect(fr.Get(3).Series[0]).To(
				telem.MatchSeries(telem.NewSeriesV[int32](1, 2)),
			)
		})

		It("Should handle writes to channels not in digests", func() {
			s.WriteValue(888, telem.NewSeriesV[float64](3.14))
			fr, changed := s.Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(fr.Get(888).Series[0]).To(
				telem.MatchSeries(telem.NewSeriesV[float64](3.14)),
			)
		})
	})

	Describe("Flush", func() {
		It("Should return false when no writes are buffered", func() {
			_, changed := s.Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeFalse())
		})

		It("Should extract buffered writes and clear the write buffer", func() {
			s.WriteValue(1, telem.NewSeriesV[float32](9.9))
			fr, changed := s.Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(fr.Get(1).Series).To(HaveLen(1))
			_, changed = s.Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeFalse())
		})

		It("Should deep copy data so mutations don't affect flushed frames", func() {
			original := telem.NewSeriesV[float32](1.0, 2.0, 3.0)
			s.WriteValue(1, original)
			fr, _ := s.Flush(telem.Frame[uint32]{})
			original.Data[0] = 0xFF
			Expect(fr.Get(1).Series[0].Data[0]).ToNot(Equal(byte(0xFF)))
		})

		It("Should append to an existing frame", func() {
			existing := telem.UnaryFrame[uint32](100, telem.NewSeriesV[int32](1))
			s.WriteValue(3, telem.NewSeriesV[int32](2))
			fr, changed := s.Flush(existing)
			Expect(changed).To(BeTrue())
			Expect(fr.Get(100).Series).To(HaveLen(1))
			Expect(fr.Get(3).Series).To(HaveLen(1))
		})

		It("Should flush writes for multiple channels", func() {
			s.WriteValue(1, telem.NewSeriesV[float32](1.0))
			s.WriteValue(3, telem.NewSeriesV[int32](2))
			fr, changed := s.Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(fr.Get(1).Series).To(HaveLen(1))
			Expect(fr.Get(3).Series).To(HaveLen(1))
		})
	})

	Describe("ReadSeries", func() {
		It("Should return data without time for non-indexed channels", func() {
			s.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](1, 2)))
			data, time, ok := s.ReadSeries(3)
			Expect(ok).To(BeTrue())
			Expect(data.Series).To(HaveLen(1))
			Expect(time.Series).To(BeEmpty())
		})

		It("Should return data with time for indexed channels", func() {
			fr := telem.Frame[uint32]{}
			fr = fr.Append(1, telem.NewSeriesV[float32](1.0, 2.0))
			fr = fr.Append(2, telem.NewSeriesSecondsTSV(10, 20))
			s.Ingest(fr)
			data, time, ok := s.ReadSeries(1)
			Expect(ok).To(BeTrue())
			Expect(data.Series).To(HaveLen(1))
			Expect(time.Series).To(HaveLen(1))
		})

		It("Should return false when index data is missing", func() {
			s.Ingest(telem.UnaryFrame[uint32](1, telem.NewSeriesV[float32](1.0)))
			_, _, ok := s.ReadSeries(1)
			Expect(ok).To(BeFalse())
		})

		It("Should return false for unknown channel", func() {
			_, _, ok := s.ReadSeries(999)
			Expect(ok).To(BeFalse())
		})

		It("Should return ok when data series has zero-length content", func() {
			s.Ingest(telem.UnaryFrame[uint32](3, telem.Series{}))
			data, _, ok := s.ReadSeries(3)
			Expect(ok).To(BeTrue())
			Expect(data.Series).To(HaveLen(1))
		})

		It("Should return accumulated data across multiple ingestions", func() {
			fr1 := telem.Frame[uint32]{}
			fr1 = fr1.Append(1, telem.NewSeriesV[float32](1.0))
			fr1 = fr1.Append(2, telem.NewSeriesSecondsTSV(10))
			s.Ingest(fr1)
			fr2 := telem.Frame[uint32]{}
			fr2 = fr2.Append(1, telem.NewSeriesV[float32](2.0))
			fr2 = fr2.Append(2, telem.NewSeriesSecondsTSV(20))
			s.Ingest(fr2)
			data, time, ok := s.ReadSeries(1)
			Expect(ok).To(BeTrue())
			Expect(data.Series).To(HaveLen(2))
			Expect(time.Series).To(HaveLen(2))
		})
	})

	Describe("WriteChannel", func() {
		It("Should buffer both data and time series", func() {
			data := telem.NewSeriesV[float32](1.0, 2.0)
			time := telem.NewSeriesSecondsTSV(100, 200)
			s.WriteChannel(1, data, time)
			fr, changed := s.Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(fr.Get(1).Series[0]).To(telem.MatchSeries(data))
			Expect(fr.Get(2).Series[0]).To(telem.MatchSeries(time))
		})

		It("Should write time to index channel", func() {
			data := telem.NewSeriesV[float64](9.9)
			time := telem.NewSeriesSecondsTSV(500)
			s.WriteChannel(5, data, time)
			fr, _ := s.Flush(telem.Frame[uint32]{})
			Expect(fr.Get(6).Series).To(HaveLen(1))
		})

		It("Should not write time to index for non-indexed channels", func() {
			data := telem.NewSeriesV[int32](42)
			time := telem.NewSeriesSecondsTSV(100)
			s.WriteChannel(3, data, time)
			fr, _ := s.Flush(telem.Frame[uint32]{})
			Expect(fr.Get(3).Series).To(HaveLen(1))
			Expect(fr.Get(0).Series).To(BeEmpty())
		})
	})

	Describe("ClearReads", func() {
		It("Should preserve the latest series for each channel", func() {
			s.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](1)))
			s.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](2)))
			s.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](3)))
			s.ClearReads()
			data, _, ok := s.ReadSeries(3)
			Expect(ok).To(BeTrue())
			Expect(data.Series).To(HaveLen(1))
			Expect(data.Series[0]).To(telem.MatchSeries(telem.NewSeriesV[int32](3)))
		})

		It("Should be a no-op for channels with a single series", func() {
			s.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](1)))
			s.ClearReads()
			data, _, ok := s.ReadSeries(3)
			Expect(ok).To(BeTrue())
			Expect(data.Series).To(HaveLen(1))
		})

		It("Should be a no-op for empty state", func() {
			Expect(func() { s.ClearReads() }).ToNot(Panic())
		})

		It("Should handle multiple channels independently", func() {
			s.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](1)))
			s.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](2)))
			fr := telem.Frame[uint32]{}
			fr = fr.Append(1, telem.NewSeriesV[float32](10.0))
			fr = fr.Append(2, telem.NewSeriesSecondsTSV(100))
			s.Ingest(fr)
			s.ClearReads()
			d3, _, ok3 := s.ReadSeries(3)
			Expect(ok3).To(BeTrue())
			Expect(d3.Series).To(HaveLen(1))
			Expect(d3.Series[0]).To(telem.MatchSeries(telem.NewSeriesV[int32](2)))
			d1, _, ok1 := s.ReadSeries(1)
			Expect(ok1).To(BeTrue())
			Expect(d1.Series).To(HaveLen(1))
		})

		It("Should allow new ingestions after clear", func() {
			s.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](1)))
			s.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](2)))
			s.ClearReads()
			s.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](3)))
			data, _, ok := s.ReadSeries(3)
			Expect(ok).To(BeTrue())
			Expect(data.Series).To(HaveLen(2))
		})

		It("Should use fresh allocation when capacity exceeds threshold", func() {
			for range 100 {
				s.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](1)))
			}
			s.ClearReads()
			data, _, ok := s.ReadSeries(3)
			Expect(ok).To(BeTrue())
			Expect(data.Series).To(HaveLen(1))
		})
	})

	Describe("WriteChannelFixed", func() {
		It("Should write all fixed numeric types and auto-index timestamps", func() {
			s.WriteChannelI32(1, 42)
			fr, changed := s.Flush(telem.Frame[uint32]{})
			Expect(changed).To(BeTrue())
			Expect(fr.Get(1).Series).To(HaveLen(1))
			Expect(telem.ValueAt[int32](fr.Get(1).Series[0], 0)).To(Equal(int32(42)))
			Expect(fr.Get(2).Series).To(HaveLen(1))
			Expect(fr.Get(2).Series[0].DataType).To(Equal(telem.TimeStampT))
		})

		It("Should not write index for channels without one", func() {
			s.WriteChannelI32(3, 10)
			fr, _ := s.Flush(telem.Frame[uint32]{})
			Expect(fr.Get(3).Series).To(HaveLen(1))
			Expect(fr.Get(0).Series).To(BeEmpty())
		})
	})

	Describe("Write accumulation metadata", func() {
		It("Should merge time ranges across multiple writes", func() {
			ser1 := telem.NewSeriesV[int32](1)
			ser1.TimeRange = telem.TimeRange{Start: 100, End: 200}
			ser2 := telem.NewSeriesV[int32](2)
			ser2.TimeRange = telem.TimeRange{Start: 50, End: 300}
			s.WriteValue(3, ser1)
			s.WriteValue(3, ser2)
			fr, _ := s.Flush(telem.Frame[uint32]{})
			Expect(fr.Get(3).Series[0].TimeRange.Start).To(Equal(telem.TimeStamp(50)))
			Expect(fr.Get(3).Series[0].TimeRange.End).To(Equal(telem.TimeStamp(300)))
		})
	})

	Describe("Ingest then Flush roundtrip", func() {
		It("Should not cross-contaminate reads and writes", func() {
			s.Ingest(telem.UnaryFrame[uint32](3, telem.NewSeriesV[int32](10)))
			s.WriteValue(3, telem.NewSeriesV[int32](20))
			readSer := MustBeOk(s.ReadValue(3))
			Expect(telem.ValueAt[int32](readSer, 0)).To(Equal(int32(10)))
			fr, _ := s.Flush(telem.Frame[uint32]{})
			Expect(telem.ValueAt[int32](fr.Get(3).Series[0], 0)).To(Equal(int32(20)))
		})
	})
})
