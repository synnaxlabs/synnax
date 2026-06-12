// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pb_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"

	"github.com/synnaxlabs/x/telem"
	telempb "github.com/synnaxlabs/x/telem/pb"
)

var _ = Describe("Frame Translator", func() {
	Describe("SeriesToPB + SeriesFromPB", func() {
		It("Should round-trip a Series", func() {
			original := telem.Series{
				DataType:  telem.Float32T,
				Data:      []byte{0, 0, 128, 63, 0, 0, 0, 64},
				Alignment: telem.NewAlignment(1, 5),
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(1000),
					End:   telem.TimeStamp(2000),
				},
			}
			pb := MustSucceed(telempb.SeriesToPB(original))
			result := MustSucceed(telempb.SeriesFromPB(pb))
			Expect(result.DataType).To(Equal(original.DataType))
			Expect(result.Data).To(Equal(original.Data))
			Expect(result.Alignment).To(Equal(original.Alignment))
			Expect(result.TimeRange.Start).To(Equal(original.TimeRange.Start))
			Expect(result.TimeRange.End).To(Equal(original.TimeRange.End))
		})

		It("Should handle a nil protobuf Series", func() {
			result := MustSucceed(telempb.SeriesFromPB(nil))
			Expect(result).To(Equal(telem.Series{}))
		})

		It("Should handle a zero-value Series", func() {
			original := telem.Series{}
			pb := MustSucceed(telempb.SeriesToPB(original))
			result := MustSucceed(telempb.SeriesFromPB(pb))
			Expect(result.DataType).To(Equal(original.DataType))
			Expect(result.Alignment).To(Equal(original.Alignment))
		})
	})

	Describe("ManySeriesToPB + ManySeriesFromPB", func() {
		It("Should round-trip a slice of Series", func() {
			original := []telem.Series{
				{DataType: telem.Float32T, Data: []byte{1, 2, 3, 4}},
				{DataType: telem.Int64T, Data: []byte{5, 6, 7, 8, 9, 10, 11, 12}},
			}
			pb := MustSucceed(telempb.ManySeriesToPB(original))
			Expect(pb).To(HaveLen(2))
			result := MustSucceed(telempb.ManySeriesFromPB(pb))
			Expect(result).To(HaveLen(2))
			Expect(result[0].DataType).To(Equal(telem.Float32T))
			Expect(result[0].Data).To(Equal([]byte{1, 2, 3, 4}))
			Expect(result[1].DataType).To(Equal(telem.Int64T))
			Expect(result[1].Data).To(Equal([]byte{5, 6, 7, 8, 9, 10, 11, 12}))
		})

		It("Should handle an empty slice", func() {
			pb := MustSucceed(telempb.ManySeriesToPB(nil))
			Expect(pb).To(BeEmpty())
			result := MustSucceed(telempb.ManySeriesFromPB(nil))
			Expect(result).To(BeEmpty())
		})
	})

	Describe("FrameToPB + FrameFromPB", func() {
		It("Should round-trip a Frame", func() {
			original := telem.MultiFrame[uint32](
				[]uint32{1, 2},
				[]telem.Series{
					{DataType: telem.Float64T, Data: []byte{1, 2, 3, 4, 5, 6, 7, 8}},
					{DataType: telem.Int32T, Data: []byte{9, 10, 11, 12}},
				},
			)
			pb := MustSucceed(telempb.FrameToPB(original))
			result := MustSucceed(telempb.FrameFromPB[uint32](pb))
			Expect(result.Count()).To(Equal(2))
			keys := result.KeysSlice()
			Expect(keys).To(Equal([]uint32{1, 2}))
			series := result.SeriesSlice()
			Expect(series[0].DataType).To(Equal(telem.Float64T))
			Expect(series[0].Data).To(Equal([]byte{1, 2, 3, 4, 5, 6, 7, 8}))
			Expect(series[1].DataType).To(Equal(telem.Int32T))
			Expect(series[1].Data).To(Equal([]byte{9, 10, 11, 12}))
		})

		It("Should handle a nil protobuf Frame", func() {
			result := MustSucceed(telempb.FrameFromPB[uint32](nil))
			Expect(result.Empty()).To(BeTrue())
		})

		It("Should handle an empty Frame", func() {
			original := telem.MultiFrame[uint32](nil, nil)
			pb := MustSucceed(telempb.FrameToPB(original))
			result := MustSucceed(telempb.FrameFromPB[uint32](pb))
			Expect(result.Empty()).To(BeTrue())
		})
	})
})
