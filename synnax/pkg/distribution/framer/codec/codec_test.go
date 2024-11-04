// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package codec_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"testing"
)

var _ = Describe("Encoder", func() {
	DescribeTable("Tests", func(
		channels channel.Keys,
		dataTypes []telem.DataType,
		fr framer.Frame,
	) {
		cdc := codec.NewCodec(dataTypes, channels)
		encoded := MustSucceed(cdc.Encode(fr, 0))
		decoded := MustSucceed(cdc.Decode(encoded))
		Expect(decoded.Series).To(HaveLen(len(fr.Series)))
		for i, k := range decoded.Keys {
			dcs := decoded.Series[i]
			os_ := fr.Get(k)
			Expect(os_).ToNot(BeEmpty())
			os := os_[0]
			Expect(dcs.TimeRange).To(Equal(os.TimeRange))
			Expect(dcs.Alignment).To(Equal(os.Alignment))
			Expect(dcs.String()).To(Equal(os.String()))
		}
	},
		Entry("Empty Frame", channel.Keys{}, []telem.DataType{}, framer.Frame{}),
		Entry("All Channels Present, In Order",
			channel.Keys{1, 2, 3},
			[]telem.DataType{telem.Int64T, telem.Float32T, telem.Float64T},
			framer.Frame{
				Keys: channel.Keys{1, 2, 3},
				Series: []telem.Series{
					telem.NewSeriesV[int64](1, 2, 3),
					telem.NewSeriesV[float32](4, 5, 6),
					telem.NewSeriesV[float64](7, 8, 9),
				},
			},
		),
		Entry("All Channels Present, Out of Order",
			channel.Keys{3, 1, 2},
			[]telem.DataType{telem.Float64T, telem.Int64T, telem.Float32T},
			framer.Frame{
				Keys: channel.Keys{2, 3, 1},
				Series: []telem.Series{
					telem.NewSeriesV[float32](3, 2, 1),
					telem.NewSeriesV[float64](1, 2, 3),
					telem.NewSeriesV[int64](5, 6, 7),
				},
			},
		),
		Entry("Some Channels Present, In Order",
			channel.Keys{1, 2, 3},
			[]telem.DataType{telem.Uint8T, telem.Float32T, telem.Float64T},
			framer.Frame{
				Keys: channel.Keys{1, 3},
				Series: []telem.Series{
					telem.NewSeriesV[uint8](1, 2, 3),
					telem.NewSeriesV[float64](7, 8, 9),
				},
			},
		),
		Entry("Some Channels Present, Out of Order",
			channel.Keys{1, 2, 3},
			[]telem.DataType{telem.Uint8T, telem.Float32T, telem.Float64T},
			framer.Frame{
				Keys: channel.Keys{3, 1},
				Series: []telem.Series{
					telem.NewSeriesV[float64](7, 8, 9),
					telem.NewSeriesV[uint8](1, 2, 3),
				},
			},
		),
		Entry("All Same Time Range",
			channel.Keys{1, 2},
			[]telem.DataType{telem.Uint8T, telem.Float32T},
			framer.Frame{
				Keys: channel.Keys{1, 2},
				Series: []telem.Series{
					{
						DataType:  telem.Uint8T,
						Data:      []byte{1},
						TimeRange: telem.TimeStamp(0).SpanRange(5),
					},
					{
						DataType:  telem.Float32T,
						Data:      []byte{1, 2, 3, 4},
						TimeRange: telem.TimeStamp(0).SpanRange(5),
					},
				},
			},
		),
		Entry("Different Time Ranges",
			channel.Keys{1, 2},
			[]telem.DataType{telem.Uint8T, telem.Float32T},
			framer.Frame{
				Keys: channel.Keys{1, 2},
				Series: []telem.Series{
					{
						DataType:  telem.Uint8T,
						Data:      []byte{1},
						TimeRange: telem.TimeStamp(0).SpanRange(5),
					},
					{
						DataType:  telem.Float32T,
						Data:      []byte{1, 2, 3, 4},
						TimeRange: telem.TimeStamp(5).SpanRange(5),
					},
				},
			},
		),
		Entry("Partial Present, Different Lengths",
			channel.Keys{1, 2, 3},
			[]telem.DataType{telem.Uint8T, telem.Float32T, telem.Float64T},
			framer.Frame{
				Keys: channel.Keys{1, 3},
				Series: []telem.Series{
					telem.NewSeriesV[uint8](1),
					telem.NewSeriesV[float64](1, 2, 3, 4),
				},
			},
		),
		Entry("Same Alignments",
			channel.Keys{1, 2},
			[]telem.DataType{telem.Uint8T, telem.Float32T},
			framer.Frame{
				Keys: channel.Keys{1, 2},
				Series: []telem.Series{
					{
						DataType:  telem.Uint8T,
						Data:      []byte{1},
						Alignment: 5,
					},
					{
						DataType:  telem.Float32T,
						Data:      []byte{1, 2, 3, 4},
						Alignment: 5,
					},
				},
			},
		),
		Entry("Different Alignments",
			channel.Keys{1, 2},
			[]telem.DataType{telem.Uint8T, telem.Float32T},
			framer.Frame{
				Keys: channel.Keys{1, 2},
				Series: []telem.Series{
					{
						DataType:  telem.Uint8T,
						Data:      []byte{1},
						Alignment: 5,
					},
					{
						DataType:  telem.Float32T,
						Data:      []byte{1, 2, 3, 4},
						Alignment: 10,
					},
				},
			},
		),
	)
})

func BenchmarkCodec(b *testing.B) {
	dataTypes := []telem.DataType{"int32"}
	keys := channel.Keys{1}
	fr := api.Frame{
		Keys:   keys,
		Series: []telem.Series{telem.NewSeriesV[int32](1, 2, 3)},
	}
	cd := codec.NewCodec(dataTypes, keys)
	for range b.N {
		cd.Encode(fr, 0)
	}
}

//
//func BenchmarkCodecJSON(b *testing.B) {
//	keys := channel.Keys{1}
//	fr := api.Frame{
//		Keys:   keys,
//		Series: []telem.Series{telem.NewSeriesV[int32](1, 2, 3)},
//	}
//	cd := &binary.JSONCodec{}
//	for range b.N {
//		cd.Encode(ctx, fr)
//	}
//}
