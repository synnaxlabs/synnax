// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Pb", func() {
	Describe("TranslateTimeRangeForward/Backward", func() {
		It("should translate TimeRange forward and backward", func() {
			tr := telem.TimeRange{Start: 123, End: 456}
			pb := telem.TranslateTimeRangeForward(tr)
			Expect(pb).NotTo(BeNil())
			Expect(pb.Start).To(Equal(int64(123)))
			Expect(pb.End).To(Equal(int64(456)))

			back := telem.TranslateTimeRangeBackward(pb)
			Expect(back).To(Equal(tr))
		})

		It("should handle nil PBTimeRange in backward translation", func() {
			var pb *telem.PBTimeRange = nil
			back := telem.TranslateTimeRangeBackward(pb)
			Expect(back.Start).To(BeZero())
			Expect(back.End).To(BeZero())
		})
	})

	Describe("TranslateSeriesForward/Backward", func() {
		It("should translate Series forward and backward", func() {
			series := telem.Series{
				DataType:  telem.Int64T,
				TimeRange: telem.TimeRange{Start: 1, End: 2},
				Data:      []byte{1, 2, 3, 4},
				Alignment: telem.Alignment(42),
			}
			pb := telem.TranslateSeriesForward(series)
			Expect(pb).NotTo(BeNil())
			Expect(pb.DataType).To(Equal(string(telem.Int64T)))
			Expect(pb.TimeRange).NotTo(BeNil())
			Expect(pb.TimeRange.Start).To(Equal(int64(1)))
			Expect(pb.TimeRange.End).To(Equal(int64(2)))
			Expect(pb.Data).To(Equal([]byte{1, 2, 3, 4}))
			Expect(pb.Alignment).To(Equal(uint64(42)))

			back := telem.TranslateSeriesBackward(pb)
			Expect(back.DataType).To(Equal(telem.Int64T))
			Expect(back.TimeRange).To(Equal(series.TimeRange))
			Expect(back.Data).To(Equal(series.Data))
			Expect(back.Alignment).To(Equal(series.Alignment))
		})

		It("should handle nil TimeRange in PBSeries", func() {
			pb := &telem.PBSeries{
				DataType:  string(telem.Int64T),
				TimeRange: nil,
				Data:      []byte{1, 2, 3},
				Alignment: 7,
			}
			back := telem.TranslateSeriesBackward(pb)
			Expect(back.TimeRange.Start).To(BeZero())
			Expect(back.TimeRange.End).To(BeZero())
		})
	})

	Describe("TranslateManySeriesForward/Backward", func() {
		It("should translate many Series forward and backward", func() {
			series := []telem.Series{
				{
					DataType:  telem.Int32T,
					TimeRange: telem.TimeRange{Start: 10, End: 20},
					Data:      []byte{10, 20},
					Alignment: telem.Alignment(1),
				},
				{
					DataType:  telem.Float64T,
					TimeRange: telem.TimeRange{Start: 30, End: 40},
					Data:      []byte{30, 40},
					Alignment: telem.Alignment(2),
				},
			}
			pbSeries := telem.TranslateManySeriesForward(series)
			Expect(pbSeries).To(HaveLen(2))
			Expect(pbSeries[0].DataType).To(Equal(string(telem.Int32T)))
			Expect(pbSeries[1].DataType).To(Equal(string(telem.Float64T)))

			back := telem.TranslateManySeriesBackward(pbSeries)
			Expect(back).To(HaveLen(2))
			Expect(back[0].DataType).To(Equal(telem.Int32T))
			Expect(back[1].DataType).To(Equal(telem.Float64T))
			Expect(back[0].TimeRange).To(Equal(series[0].TimeRange))
			Expect(back[1].TimeRange).To(Equal(series[1].TimeRange))
			Expect(back[0].Data).To(Equal(series[0].Data))
			Expect(back[1].Data).To(Equal(series[1].Data))
			Expect(back[0].Alignment).To(Equal(series[0].Alignment))
			Expect(back[1].Alignment).To(Equal(series[1].Alignment))
		})

		It("should handle empty slices", func() {
			var series []telem.Series
			pbSeries := telem.TranslateManySeriesForward(series)
			Expect(pbSeries).To(BeEmpty())

			var pbSlice []*telem.PBSeries
			back := telem.TranslateManySeriesBackward(pbSlice)
			Expect(back).To(BeEmpty())
		})
	})
})
