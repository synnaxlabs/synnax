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
	"bytes"
	"encoding/json"
	"math/rand"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Codec", func() {
	DescribeTable("Encode + Decode", func(
		channels channel.Keys,
		dataTypes []telem.DataType,
		fr framer.Frame,
	) {
		cdc := codec.NewStatic(dataTypes, channels)
		encoded := MustSucceed(cdc.Encode(fr))
		decoded := MustSucceed(cdc.Decode(encoded))
		Expect(fr.Frame).To(telem.MatchFrame(decoded.Frame))
	},
		Entry("Empty Frame", channel.Keys{}, []telem.DataType{}, framer.Frame{}),
		Entry("All Channels Present, In Order",
			channel.Keys{1, 2, 3},
			[]telem.DataType{telem.Int64T, telem.Float32T, telem.Float64T},
			core.MultiFrame(
				channel.Keys{1, 2, 3},
				[]telem.Series{
					telem.NewSeriesV[int64](1, 2, 3),
					telem.NewSeriesV[float32](4, 5, 6),
					telem.NewSeriesV[float64](7, 8, 9),
				},
			),
		),
		Entry("All Channels Present, Out of Order",
			channel.Keys{3, 1, 2},
			[]telem.DataType{telem.Float64T, telem.Int64T, telem.Float32T},
			core.MultiFrame(
				channel.Keys{2, 3, 1},
				[]telem.Series{
					telem.NewSeriesV[float32](3, 2, 1),
					telem.NewSeriesV[float64](1, 2, 3),
					telem.NewSeriesV[int64](5, 6, 7),
				},
			),
		),
		Entry("Some Channels Present, In Order",
			channel.Keys{1, 2, 3},
			[]telem.DataType{telem.Uint8T, telem.Float32T, telem.Float64T},
			core.MultiFrame(
				channel.Keys{1, 3},
				[]telem.Series{
					telem.NewSeriesV[uint8](1, 2, 3),
					telem.NewSeriesV[float64](7, 8, 9),
				},
			),
		),
		Entry("Some Channels Present, Out of Order",
			channel.Keys{1, 2, 3},
			[]telem.DataType{telem.Uint8T, telem.Float32T, telem.Float64T},
			core.MultiFrame(
				channel.Keys{3, 1},
				[]telem.Series{
					telem.NewSeriesV[float64](7, 8, 9),
					telem.NewSeriesV[uint8](1, 2, 3),
				},
			),
		),
		Entry("All Same Time Range",
			channel.Keys{1, 2},
			[]telem.DataType{telem.Uint8T, telem.Float32T},
			core.MultiFrame(
				channel.Keys{1, 2},
				[]telem.Series{
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
			),
		),
		Entry("Different Time Ranges",
			channel.Keys{1, 2},
			[]telem.DataType{telem.Uint8T, telem.Float32T},
			core.MultiFrame(
				channel.Keys{1, 2},
				[]telem.Series{
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
			),
		),
		Entry("Partial Present, Different Lengths",
			channel.Keys{1, 2, 3},
			[]telem.DataType{telem.Uint8T, telem.Float32T, telem.Float64T},
			core.MultiFrame(
				channel.Keys{1, 3},
				[]telem.Series{
					telem.NewSeriesV[uint8](1),
					telem.NewSeriesV[float64](1, 2, 3, 4),
				},
			),
		),
		Entry("Same Alignments",
			channel.Keys{1, 2},
			[]telem.DataType{telem.Uint8T, telem.Float32T},
			core.MultiFrame(
				channel.Keys{1, 2},
				[]telem.Series{
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
			),
		),
		Entry("Different Alignments",
			channel.Keys{1, 2},
			[]telem.DataType{telem.Uint8T, telem.Float32T},
			core.MultiFrame(
				channel.Keys{1, 2},
				[]telem.Series{
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
			),
		),
		Entry("Variable Data Types",
			channel.Keys{1, 2, 3},
			[]telem.DataType{telem.Uint8T, telem.StringT, telem.JSONT},
			core.MultiFrame(
				channel.Keys{1, 2, 3},
				[]telem.Series{
					telem.NewSeriesV[uint8](1, 2, 3),
					telem.NewStringsV("cat", "dog"),
					telem.NewStaticJSONV(
						map[string]interface{}{"key": "value"},
						map[string]interface{}{"key": "value2"},
					),
				},
			),
		),
	)

	Describe("Complex Frames", func() {
		It("Should correctly serialize and deserialize a complex frame", func() {
			keys := channel.Keys{1, 2, 3, 4}
			dataTypes := []telem.DataType{"int32", "float32", "string", "uint8"}
			s1 := telem.NewSeriesV[int32](1, 2, 3)
			s1.TimeRange = telem.NewSecondsRange(1, 12)
			s1.Alignment = 7
			float32Data := make([]float32, 5000)
			for i := range float32Data {
				float32Data[i] = 1.234 + float32(i)*rand.Float32()
			}
			s2 := telem.NewSeries[float32](float32Data)
			s2.TimeRange = telem.NewSecondsRange(3, 5)
			s2.Alignment = 10
			s3 := telem.NewStringsV("cat", "dog", "rabbit", "frog")
			s3.TimeRange = telem.NewSecondsRange(1, 5)
			s3.Alignment = 5
			s4 := telem.AllocSeries(telem.Uint8T, 5000)
			s4.Alignment = telem.LeadingAlignment(5000, 5)
			s4.TimeRange = telem.NewSecondsRange(9999999, 999999999)
			originalFrame := core.MultiFrame(
				keys,
				[]telem.Series{
					s1,
					s2,
					s3,
					s4,
				},
			)

			cdc := codec.NewStatic(dataTypes, keys)
			encoded := MustSucceed(cdc.Encode(originalFrame))
			decoded := MustSucceed(cdc.Decode(encoded))
			Expect(originalFrame.Frame).To(telem.MatchFrame(decoded.Frame))
		})
	})

	Describe("Lazy Codec", func() {
		It("Should allow the caller to update the list of channels", func() {
			builder := mock.NewBuilder()
			dist := builder.New(ctx)
			w := dist.Channel.NewWriter(nil)
			idx := channel.Channel{
				DataType: telem.TimeStampT,
				Name:     "time",
				IsIndex:  true,
			}
			Expect(w.Create(ctx, &idx)).To(Succeed())
			dataCh := channel.Channel{
				Name:       "data",
				DataType:   telem.Float32T,
				LocalIndex: idx.Key().LocalKey(),
			}
			Expect(w.Create(ctx, &dataCh)).To(Succeed())
			lazyCodec := codec.NewDynamic(dist.Channel)
			Expect(lazyCodec.Update(ctx, []channel.Key{dataCh.Key(), idx.Key()})).To(Succeed())
			fr := core.MultiFrame(
				channel.Keys{dataCh.Key(), idx.Key()},
				[]telem.Series{
					telem.NewSeriesV[float32](1, 2, 3, 4),
					telem.NewSecondsTSV(1, 2, 3, 4),
				},
			)
			encoded := MustSucceed(lazyCodec.Encode(fr))
			decoded := MustSucceed(lazyCodec.Decode(encoded))
			Expect(fr.Frame).To(telem.MatchFrame[channel.Key](decoded.Frame))
		})

		//It("Should panic if the codec is not initialized", func() {
		//	lazyCodec := codec.NewDynamic(nil)
		//	Expect(func() {
		//		fr := framer.Frame{}
		//		lazyCodec.Encode(fr)
		//	}).To(Panic())
		//})
	})
})

func BenchmarkEncode(b *testing.B) {
	dataTypes := []telem.DataType{"int32"}
	keys := channel.Keys{1}
	fr := core.MultiFrame(
		keys,
		[]telem.Series{telem.NewSeriesV[int32](1, 2, 3)},
	)
	cd := codec.NewStatic(dataTypes, keys)
	w := bytes.NewBuffer(nil)
	if err := cd.EncodeStream(w, fr); err != nil {
		b.Fatalf("failed to encode stream: %v", err)
	}
	for range b.N {
		if err := cd.EncodeStream(w, fr); err != nil {
			b.Fatalf("failed to encode stream: %v", err)
		}
		w.Reset()
	}
}

func BenchmarkJSONEncode(b *testing.B) {
	b.Skip()
	keys := channel.Keys{1}
	fr := core.MultiFrame(
		keys,
		[]telem.Series{telem.NewSeriesV[int32](1, 2, 3)},
	)
	for range b.N {
		if _, err := json.Marshal(fr); err != nil {
			b.Fatalf("failed to encode stream: %v", err)
		}
	}
}

func BenchmarkDecode(b *testing.B) {
	var (
		dataTypes = []telem.DataType{"int32"}
		keys      = channel.Keys{1}
		fr        = core.MultiFrame(
			keys,
			[]telem.Series{telem.NewSeriesV[int32](1, 2, 3)},
		)
		cd      = codec.NewStatic(dataTypes, keys)
		encoded = MustSucceed(cd.Encode(fr))
		r       = bytes.NewReader(encoded)
	)
	for range b.N {
		if _, err := r.Seek(0, 0); err != nil {
			b.Fatalf("failed to seek: %v", err)
		}
		if fr, err := cd.DecodeStream(r); err != nil || fr.Empty() {
			b.Fatalf("failed to decode stream: %v", err)
		}
	}
}

func BenchmarkJSONDecode(b *testing.B) {
	b.Skip()
	keys := channel.Keys{1}
	encoded, err := json.Marshal(core.MultiFrame(
		keys,
		[]telem.Series{telem.NewSeriesV[int32](1, 2, 3)},
	))
	if err != nil {
		b.Fatalf("failed to encode stream: %v", err)
	}
	var v framer.Frame
	for range b.N {
		if err := json.Unmarshal(encoded, &v); err != nil {
			b.Fatalf("failed to decode stream: %v", err)
		}
	}
}
